package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/response"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/crypto"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"

	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/util"

	gojwt "github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LoginLogServiceInterface 由 service/interface.go 迁移至此:接口定义在消费方,
// 不再维护包级中央接口文件。
type LoginLogServiceInterface interface {
	RecordLogin(ctx context.Context, userID uint64, username, ip, userAgent string, code int, msg string) error
	RecordLogout(ctx context.Context, userID uint64, username, ip string) error
}

// ConfigGetterInterface 由 service/interface.go 迁移至此:接口定义在消费方,
// 不再维护包级中央接口文件。
type ConfigGetterInterface interface {
	GetString(ctx context.Context, key, defaultVal string) string
	GetInt(ctx context.Context, key string, defaultVal int) int
	GetBool(ctx context.Context, key string, defaultVal bool) bool
}

// CaptchaInterface 由 service/interface.go 迁移至此:接口定义在消费方,
// 不再维护包级中央接口文件。
type CaptchaInterface interface {
	Verify(ctx context.Context, key, code string) error
	IncrementFailCount(ctx context.Context, userID uint64) (int, error)
	IsCaptchaRequired(ctx context.Context, userID uint64) (bool, error)
	IsLocked(ctx context.Context, userID uint64) (bool, int, error)
	SetLock(ctx context.Context, userID uint64) error
	Unlock(ctx context.Context, userID uint64) error
}

// AuthRepositoryInterface 由 service/interface.go 迁移至此:接口定义在消费方,
// 不再维护包级中央接口文件。
type AuthRepositoryInterface interface {
	FindByUsername(ctx context.Context, username string) (*entity.SysUser, error)
	FindByID(ctx context.Context, id uint64) (*entity.SysUser, error)
	GetRoleCodes(ctx context.Context, userID uint64) ([]string, error)
	FindMenuPerms(ctx context.Context) ([]string, error)
	GetUserDataScope(ctx context.Context, userID uint64) (int8, uint64, error)
	GetUserRoleScope(ctx context.Context, userID uint64) int8
	UpdatePassword(ctx context.Context, userID uint64, newPassword string, newSalt *string) error
	UpdateLoginInfo(ctx context.Context, userID uint64, ip string) error
	UpdateProfile(ctx context.Context, userID uint64, realName, email, phone *string) error
	UpdateAvatar(ctx context.Context, userID uint64, avatar string) error
	CountUserLogins(ctx context.Context, userID uint64) (int64, error)
	FindUserLoginLogsSince(ctx context.Context, userID uint64, since time.Time) ([]entity.SysLoginLog, error)
	GetDeptName(ctx context.Context, deptID uint64) (string, error)
}

type AuthService struct {
	cfgProv       ConfigGetterInterface
	repo          AuthRepositoryInterface
	logger        logger.LoggerInterface
	sessionStore  SessionStoreInterface
	loginLogSvc   LoginLogServiceInterface
	captchaSvc    CaptchaInterface
	apiPrefix     string // server.apiPrefix,用于拼接头像访问路径
	scopeResolver *datascope.ScopeResolver
}

// NewAuthService constructs an AuthService with the given dependencies.
func NewAuthService(
	cfgProv ConfigGetterInterface,
	repo AuthRepositoryInterface,
	logger logger.LoggerInterface,
	sessionStore SessionStoreInterface,
	loginLogSvc LoginLogServiceInterface,
	captchaSvc CaptchaInterface,
	apiPrefix string,
	scopeResolver *datascope.ScopeResolver,
) *AuthService {
	return &AuthService{
		cfgProv:       cfgProv,
		repo:          repo,
		logger:        logger,
		sessionStore:  sessionStore,
		loginLogSvc:   loginLogSvc,
		captchaSvc:    captchaSvc,
		apiPrefix:     apiPrefix,
		scopeResolver: scopeResolver,
	}
}

// issueTokens generates a signed access token and refresh token for the given user.
func (s *AuthService) issueTokens(ctx context.Context, userID uint64, perms []string, scopes []jwt.ScopeClaim) (accessToken, refreshToken string, err error) {
	secret := s.cfgProv.GetString(ctx, jwt.SecretConfigKey, jwt.DefaultSecretFallback)
	accessExpire := s.cfgProv.GetInt(ctx, "sys.jwt.accessExpire", 7200)
	refreshExpire := s.cfgProv.GetInt(ctx, "sys.jwt.refreshExpire", 604800)
	accessToken, err = jwt.GenerateAccessToken(userID, perms, scopes, secret, accessExpire)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = jwt.GenerateRefreshToken(userID, secret, refreshExpire)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// storeSession writes the access token, refresh token, and permission cache
// to the session hashStore. A StoreAccess failure is returned to the caller:
// IsAccessValid treats a whitelist miss as invalid, so handing out a token
// whose whitelist write failed means "login succeeds, then immediate 401".
// Refresh/perms writes stay best-effort (degraded, not broken).
func (s *AuthService) storeSession(ctx context.Context, userID uint64, accessToken, refreshToken string, perms []string) error {
	accessTTL := time.Duration(s.cfgProv.GetInt(ctx, "sys.jwt.accessExpire", 7200)) * time.Second
	refreshTTL := time.Duration(s.cfgProv.GetInt(ctx, "sys.jwt.refreshExpire", 604800)) * time.Second
	if err := s.sessionStore.StoreAccess(ctx, accessToken, userID, accessTTL); err != nil {
		s.logger.Error("failed to store access token whitelist", zap.Error(err))
		return err
	}
	if err := s.sessionStore.StoreRefresh(ctx, userID, refreshToken, refreshTTL); err != nil {
		s.logger.Warn("failed to store refresh token", zap.Error(err))
	}
	if err := s.sessionStore.StorePerms(ctx, userID, perms, accessTTL); err != nil {
		s.logger.Warn("failed to store permissions", zap.Error(err))
	}
	return nil
}

// buildUserScopes 构造 JWT 数据范围声明。Login 与 RefreshToken 共用:
// 此前两处独立构造且已发散(refresh 无条件附 dept+self 双维度,login 按
// 数据范围条件构造),同一用户经两条路径签发的 token 可见数据集不同。
// 语义以 Login 为准:仅包含与用户实际数据范围匹配的维度。
func buildUserScopes(userID uint64, dataScope int8, deptID uint64, roleScope int8) []jwt.ScopeClaim {
	var scopes []jwt.ScopeClaim
	if dataScope == datascope.ScopeSelf {
		scopes = []jwt.ScopeClaim{
			{Dimension: "self", Level: datascope.ScopeSelf, SelfID: userID},
		}
	} else {
		scopes = []jwt.ScopeClaim{
			{Dimension: "dept", Level: dataScope, SelfID: deptID},
		}
	}
	scopes = append(scopes, jwt.ScopeClaim{Dimension: "role", Level: roleScope, SelfID: 0})
	return scopes
}

// scopeCtxFor 按 scopes 声明构造 ScopeContext 注入 ctx,复用与
// middleware.ScopeResolverHandler 完全相同的 resolver,权限口径与运行时一致。
// resolver 解析失败降级为该维度空集(fail-closed,不放大为全量),
// scopeResolver 未注入(测试)时原样返回。
func (s *AuthService) scopeCtxFor(ctx context.Context, userID uint64, scopes []jwt.ScopeClaim) context.Context {
	if s.scopeResolver == nil {
		return ctx
	}
	sc := &datascope.ScopeContext{UserID: userID, Dimensions: make(map[string]*datascope.ResolvedDimension, len(scopes))}
	for _, claim := range scopes {
		resolver, ok := s.scopeResolver.Resolvers[claim.Dimension]
		if !ok {
			continue
		}
		dim, err := resolver.Resolve(ctx, claim.Level, claim.SelfID, userID)
		if err != nil {
			s.logger.Warn("scope resolve failed during access resolution",
				zap.String("dimension", claim.Dimension),
				zap.Int8("level", claim.Level),
				zap.Error(err))
			// 降级与 middleware.ScopeResolverHandler 完全一致:写入该维度
			// 并置 **空集**。此前两处的降级都只填 Level/SelfID,AllowedIDs
			// 保持 nil —— 而 plugin.scopeCallback 把 nil 读作"授予全部",
			// 于是"解析失败"被放大为"可见全部",与注释声称的防护相反。
			// 空集才是真正的 fail-closed:该维度无可见项,但仍允许其他
			// 维度授权(插件按维度 OR 拼接)。
			sc.Dimensions[claim.Dimension] = &datascope.ResolvedDimension{
				Level:      claim.Level,
				SelfID:     claim.SelfID,
				AllowedIDs: []uint64{},
			}
			continue
		}
		sc.Dimensions[claim.Dimension] = dim
	}
	return datascope.WithScopeContext(ctx, sc)
}

// resolveUserAccess 一次解析登录/刷新所需的全部权限数据:权限点集合与
// 数据范围声明。权限点由 scope 插件自动注入过滤(scopeCtxFor 构造的
// ScopeContext),与运行时菜单查询共用同一过滤来源;admin(role scope=ScopeAll)
// 附加 "admin" 通配标记。加载失败降级为空权限,不阻断发 token(与旧行为一致)。
func (s *AuthService) resolveUserAccess(ctx context.Context, userID uint64) ([]string, []jwt.ScopeClaim) {
	dataScope, deptID, err := s.repo.GetUserDataScope(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to load user data scope", zap.Uint64("userId", userID), zap.Error(err))
		dataScope = datascope.ScopeSelf
		deptID = 0
	}

	roleScope := s.repo.GetUserRoleScope(ctx, userID)
	scopes := buildUserScopes(userID, dataScope, deptID, roleScope)

	perms, err := s.repo.FindMenuPerms(s.scopeCtxFor(ctx, userID, scopes))
	if err != nil {
		s.logger.Warn("failed to load user permissions", zap.Uint64("userId", userID), zap.Error(err))
		perms = []string{}
	}
	if roleScope == datascope.ScopeAll {
		perms = append(perms, "admin")
	}
	return perms, scopes
}

// roleScopeAllFromCtx 判断 ctx 中 role 维度是否为 ScopeAll(admin)。
// 无 ScopeContext / 无 role 维度时返回 false(防御性不加 admin 标记)。
func roleScopeAllFromCtx(ctx context.Context) bool {
	sc, ok := datascope.ScopeContextFromCtx(ctx)
	if !ok || sc == nil || sc.Dimensions == nil {
		return false
	}
	dim, ok := sc.Dimensions[datascope.DimRole]
	return ok && dim != nil && dim.Level == datascope.ScopeAll
}

func (s *AuthService) Login(ctx context.Context, req *request.LoginReq, ip, userAgent string) (*response.LoginResp, error) {
	user, err := s.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		go s.recordLoginAsync(0, req.Username, ip, userAgent, apperror.CodeUnauthorized, "用户名或密码错误")
		return nil, apperror.Unauthorized("用户名或密码错误")
	}
	// Captcha check
	if s.captchaSvc != nil {
		required, err := s.captchaSvc.IsCaptchaRequired(ctx, user.ID)
		if err != nil {
			s.logger.Warn("failed to check captcha required", zap.Uint64("userId", user.ID), zap.Error(err))
			return nil, apperror.Internal("验证码服务异常")
		}
		if required {
			if req.CaptchaKey == "" || req.CaptchaCode == "" {
				return nil, apperror.CaptchaRequired("验证码已启用，请先获取验证码")
			}
			if err := s.captchaSvc.Verify(ctx, req.CaptchaKey, req.CaptchaCode); err != nil {
				// 验证码错误与密码错误共用同一失败计数与锁定评估:否则攻击者可
				// 无限次枚举验证码而不触发账号锁定。
				s.recordAuthFailure(ctx, user.ID)
				code := apperror.CodeCaptchaIncorrect
				var appErr *apperror.AppError
				if errors.As(err, &appErr) {
					code = appErr.Code
				}
				go s.recordLoginAsync(user.ID, user.Username, ip, userAgent, code, "验证码错误")
				return nil, err
			}
		}
	}

	if user.Status != nil && *user.Status == entity.UserStatusDisabled {
		go s.recordLoginAsync(user.ID, user.Username, ip, userAgent, apperror.CodeUnauthorized, "账号已被禁用")
		return nil, apperror.Unauthorized("账号已被禁用，请联系管理员")
	}

	// Lock check (fail-open: Redis unavailable allows login to proceed)
	if s.captchaSvc != nil {
		locked, remaining, err := s.captchaSvc.IsLocked(ctx, user.ID)
		if err != nil {
			s.logger.Warn("failed to check lock status, failing open", zap.Uint64("userId", user.ID), zap.Error(err))
		} else if locked {
			go s.recordLoginAsync(user.ID, user.Username, ip, userAgent, apperror.CodeAccountLocked, "账号已锁定")
			return nil, apperror.AccountLocked(remaining)
		}
	}

	salt := ""
	if user.PasswordSalt != nil {
		salt = *user.PasswordSalt
	}
	if !crypto.VerifyPassword(req.Password, user.Password, salt) {
		// Increment fail count on wrong password
		if s.captchaSvc != nil {
			s.recordAuthFailure(ctx, user.ID)
		}
		go s.recordLoginAsync(user.ID, user.Username, ip, userAgent, apperror.CodeUnauthorized, "用户名或密码错误")
		return nil, apperror.Unauthorized("用户名或密码错误")
	}

	// Unlock on successful login
	if s.captchaSvc != nil {
		if unlockErr := s.captchaSvc.Unlock(ctx, user.ID); unlockErr != nil {
			s.logger.Warn("failed to unlock", zap.Uint64("userId", user.ID), zap.Error(unlockErr))
		}
	}

	// Load permissions + data scope claims in one consolidated pass
	// (role codes 死查询一并移除:此前加载后仅 `_ = roleCodes` 丢弃)。
	perms, scopes := s.resolveUserAccess(ctx, user.ID)
	accessToken, refreshToken, err := s.issueTokens(ctx, user.ID, perms, scopes)
	if err != nil {
		return nil, apperror.Internal("生成 token 失败")
	}

	// Store session data in Redis
	if err := s.storeSession(ctx, user.ID, accessToken, refreshToken, perms); err != nil {
		return nil, apperror.Internal("会话存储失败", err)
	}

	// Update last-login metadata (best-effort)
	if err := s.repo.UpdateLoginInfo(ctx, user.ID, ip); err != nil {
		s.logger.Warn("failed to update login info", zap.Uint64("userId", user.ID), zap.Error(err))
	}

	// Record successful login asynchronously
	go s.recordLoginAsync(user.ID, user.Username, ip, userAgent, apperror.CodeOK, "")

	s.logger.Info("user login",
		zap.Uint64("userId", user.ID),
		zap.String("username", user.Username),
		zap.String("ip", ip))

	return &response.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfgProv.GetInt(ctx, "sys.jwt.accessExpire", 7200),
	}, nil
}

// recordAuthFailure increments the shared fail counter and locks the account
// once the configured threshold is reached. Wrong-password and wrong-captcha
// paths both go through it so pure captcha enumeration locks the account too.
// Caller must guarantee s.captchaSvc != nil.
func (s *AuthService) recordAuthFailure(ctx context.Context, userID uint64) {
	count, err := s.captchaSvc.IncrementFailCount(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to increment fail count", zap.Uint64("userId", userID), zap.Error(err))
		return
	}
	lockThreshold := s.cfgProv.GetInt(ctx, "sys.auth.lockThreshold", 10)
	if count >= lockThreshold {
		if lockErr := s.captchaSvc.SetLock(ctx, userID); lockErr != nil {
			s.logger.Warn("failed to set lock", zap.Uint64("userId", userID), zap.Error(lockErr))
		}
	}
}

// withAsyncLogContext 统一异步日志记录的公共样板:panic 兜底、loginLogSvc
// nil 检查、带超时的独立 ctx,然后用该 ctx 调用 fn(name 仅用于 panic 日志归属)。
// recover 必须作为跑此方法的 goroutine 的栈顶 defer —— 调用方以 go 关键字
// 起新 goroutine 时,即由这里的 defer 完成兜底,避免后台日志 panic 拖垮进程。
func (s *AuthService) withAsyncLogContext(name string, fn func(ctx context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("async log panic recovered", zap.String("task", name), zap.Any("panic", r))
		}
	}()
	if s.loginLogSvc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfgProv.GetInt(context.Background(), "sys.auth.asyncLogTimeout", 5))*time.Second)
	defer cancel()
	fn(ctx)
}

func (s *AuthService) recordLoginAsync(userID uint64, username, ip, userAgent string, code int, msg string) {
	s.withAsyncLogContext("recordLoginAsync", func(ctx context.Context) {
		if err := s.loginLogSvc.RecordLogin(ctx, userID, username, ip, userAgent, code, msg); err != nil {
			s.logger.Warn("failed to record login log", zap.Error(err))
		}
	})
}

func (s *AuthService) Logout(ctx context.Context, token, ip string) error {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return apperror.Unauthorized("未登录")
	}
	if err := s.sessionStore.RevokeAll(ctx, userID, token); err != nil {
		return err
	}
	// Record logout asynchronously
	go s.withAsyncLogContext("recordLogoutAsync", func(ctx context.Context) {
		user, err := s.repo.FindByID(ctx, userID)
		if err != nil {
			s.logger.Warn("failed to resolve user for logout log", zap.Uint64("userId", userID), zap.Error(err))
			return
		}
		if err := s.loginLogSvc.RecordLogout(ctx, userID, user.Username, ip); err != nil {
			s.logger.Warn("failed to record logout log", zap.Error(err))
		}
	})
	return nil
}

func (s *AuthService) GetUserInfo(ctx context.Context) (*response.UserInfoResp, error) {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return nil, apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, translateNotFound(err, "用户不存在")
	}
	roleCodes, err := s.repo.GetRoleCodes(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to load role codes", zap.Uint64("userId", userID), zap.Error(err))
		roleCodes = []string{}
	}
	perms, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to load user permissions", zap.Uint64("userId", userID), zap.Error(err))
		perms = []string{}
	}

	resp := &response.UserInfoResp{}
	util.CopyEntity(resp, user, s.logger)
	resp.Permissions = perms
	resp.Roles = roleCodes

	// 授权上下文收敛:范围口径唯一来源是 ScopeContext(ScopeResolverHandler
	// 注入,与查询过滤同源)。旧实现在此无条件用 contextkeys.DataScope/DeptID
	// 覆写实体值,ScopeSelf 用户(无 dept 维度)会被覆写为 0/0 丢掉真实部门。
	// 新语义:dept 维度取解析 Level/SelfID;self 维度只定 DataScope=5,
	// DeptID 保留实体值;其余情形保持实体值。
	if sc, has := datascope.ScopeContextFromCtx(ctx); has && sc != nil {
		if d, ok := sc.Dimensions["dept"]; ok && d != nil {
			resp.DataScope = d.Level
			resp.DeptID = d.SelfID
		} else if d, ok := sc.Dimensions["self"]; ok && d != nil {
			resp.DataScope = d.Level
		}
	}
	return resp, nil
}

// GetUserPermissions 从请求 ctx 携带的 ScopeContext 派生权限点:scope 插件
// 自动注入 sys_menu.id 过滤,role 维度为 ScopeAll(admin)时附加 "admin" 标记。
// 消费方是 middleware.PermissionGuard 缓存回源 —— ctx 必须是经过
// ScopeResolverHandler 的请求 ctx(而非裸 Background),否则 scope 不会注入。
func (s *AuthService) GetUserPermissions(ctx context.Context, userID uint64) ([]string, error) {
	perms, err := s.repo.FindMenuPerms(ctx)
	if err != nil {
		return nil, err
	}
	if roleScopeAllFromCtx(ctx) {
		perms = append(perms, "admin")
	}
	return perms, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, req *request.ChangePasswordReq, accessToken string) error {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return translateNotFound(err, "用户不存在")
	}
	salt := ""
	if user.PasswordSalt != nil {
		salt = *user.PasswordSalt
	}
	if !crypto.VerifyPassword(req.OldPassword, user.Password, salt) {
		return apperror.BadRequest("旧密码不正确")
	}
	hashed, newSalt, err := crypto.HashPassword(req.NewPassword, s.cfgProv.GetInt(ctx, "sys.auth.bcryptCost", 10))
	if err != nil {
		return apperror.Internal("密码加密失败")
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hashed), &newSalt); err != nil {
		return err
	}
	// Revoke all existing sessions so that old tokens are invalidated after
	// password change. The current access token is passed in so it dies
	// immediately instead of staying valid (up to its 2h expiry) alongside the
	// possibly stolen one it was reported through.
	if err := s.sessionStore.RevokeAll(ctx, userID, accessToken); err != nil {
		// Error:改密(常因疑似泄露)后旧 token 残留至 TTL——安全敏感窗口。
		// DB 密码已更新不能回滚,故记录 Error 并继续,窗口由运维告警跟进。
		s.logger.Error("failed to revoke sessions after password change", zap.Uint64("userId", userID), zap.Error(err))
	}
	return nil
}

// VerifyPassword checks the provided password against the current user's
// login password. Used by the lock screen unlock flow; intentionally does
// NOT touch tokens or sessions (unlock is a pure front-door check, real
// session security remains with logout / password change).
func (s *AuthService) VerifyPassword(ctx context.Context, req *request.VerifyPasswordReq) (*response.VerifyPasswordResp, error) {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return nil, apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, translateNotFound(err, "用户不存在")
	}
	salt := ""
	if user.PasswordSalt != nil {
		salt = *user.PasswordSalt
	}
	if !crypto.VerifyPassword(req.Password, user.Password, salt) {
		return nil, apperror.BadRequest("密码不正确")
	}
	return &response.VerifyPasswordResp{Valid: true}, nil
}

// ── 个人中心:资料自维护 / 头像上传 ─────────────────────────────────

// 头像上传约束:大小上限 2MB,扩展名白名单。与文件管理共用 sys.file.upload.path
// 上传根目录,头像落在其下 avatars/ 子目录(每用户一个文件,重传覆盖)。
const (
	defaultAvatarMaxSize = 2 << 20 // 2MB
	defaultAvatarDir     = "avatars"
)

// avatarAllowedExts 头像允许的图片扩展名(下载/展示均无需特殊解码)。
var avatarAllowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

// UpdateProfile updates the current user's own basic profile (realName, email,
// phone). Other columns (roles/dept/status/username) are never touched here —
// those flows remain in user management.
func (s *AuthService) UpdateProfile(ctx context.Context, req *request.UpdateProfileReq) error {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return apperror.Unauthorized("未登录")
	}
	if _, err := s.repo.FindByID(ctx, userID); err != nil {
		return translateNotFound(err, "用户不存在")
	}
	if err := s.repo.UpdateProfile(ctx, userID, req.RealName, req.Email, req.Phone); err != nil {
		s.logger.Warn("failed to update profile", zap.Uint64("userId", userID), zap.Error(err))
		return err
	}
	s.logger.Info("profile updated", zap.Uint64("userId", userID))
	return nil
}

// UploadAvatar validates and stores an avatar image for the current user, then
// persists its access path. Returns e.g. "/api/v1/user/avatar/8?v=1757…".
// Files live under <upload.path>/avatars/<userID><ext> (one per user, overwrite
// on re-upload); stale files with other extensions are removed so the
// directory never accumulates orphan images.
func (s *AuthService) UploadAvatar(ctx context.Context, header *multipart.FileHeader) (string, error) {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return "", apperror.Unauthorized("未登录")
	}
	if _, err := s.repo.FindByID(ctx, userID); err != nil {
		return "", translateNotFound(err, "用户不存在")
	}

	if header.Size > defaultAvatarMaxSize {
		return "", apperror.BadRequest("头像大小不能超过 2MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !avatarAllowedExts[ext] {
		return "", apperror.BadRequest("头像仅支持 JPG/PNG/WebP/GIF 格式")
	}

	basePath := s.cfgProv.GetString(ctx, "sys.file.upload.path", defaultFileUploadPath)
	dir := filepath.Join(basePath, defaultAvatarDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.logger.Error("create avatar dir failed", zap.String("path", dir), zap.Error(err))
		return "", apperror.Internal("创建头像目录失败", err)
	}

	fileName := fmt.Sprintf("%d%s", userID, ext)
	fullPath := filepath.Join(dir, fileName)

	// 清理该用户旧扩展名残留(上次为 png、这次为 webp 时),避免孤儿文件。
	if stale, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%d.*", userID))); len(stale) > 0 {
		for _, old := range stale {
			if old != fullPath {
				_ = os.Remove(old)
			}
		}
	}

	if _, err := saveMultipart(header, fullPath, defaultAvatarMaxSize); err != nil {
		return "", err
	}

	// ?v=<unix> 仅作缓存破坏:同一用户 URL 前缀稳定,重传后 ?v 变化触发回源。
	// URL 含 API 前缀,同源/反代部署下 <img> 可直接引用。
	avatarURL := fmt.Sprintf("%s/user/avatar/%d?v=%d", s.apiPrefix, userID, time.Now().Unix())
	if err := s.repo.UpdateAvatar(ctx, userID, avatarURL); err != nil {
		s.logger.Warn("failed to persist avatar", zap.Uint64("userId", userID), zap.Error(err))
		return "", err
	}
	s.logger.Info("avatar uploaded", zap.Uint64("userId", userID))
	return avatarURL, nil
}

// AvatarFilePath resolves the on-disk path of the given user's avatar image.
// The user must exist; a missing image maps to a 404 so clients can fall back
// to their default asset.
func (s *AuthService) AvatarFilePath(ctx context.Context, userID uint64) (string, error) {
	if _, err := s.repo.FindByID(ctx, userID); err != nil {
		return "", translateNotFound(err, "用户不存在")
	}
	basePath := s.cfgProv.GetString(ctx, "sys.file.upload.path", defaultFileUploadPath)
	dir := filepath.Join(basePath, defaultAvatarDir)
	matches, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%d.*", userID)))
	if len(matches) == 0 {
		return "", apperror.NotFound("头像不存在")
	}
	return matches[0], nil
}

// RefreshToken validates a refresh token, issues new tokens, and rotates the old refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, req *request.RefreshTokenReq) (*response.RefreshTokenResp, error) {
	secret := s.cfgProv.GetString(ctx, jwt.SecretConfigKey, jwt.DefaultSecretFallback)

	// 1. Parse the JWT refresh token
	claims, err := jwt.ParseRefreshToken(req.RefreshToken, secret)
	if err != nil {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return nil, apperror.TokenExpired("refresh token 已过期")
		}
		return nil, apperror.Unauthorized("无效的 refresh token")
	}

	// 2. Validate Subject == "refresh"
	sub, err := claims.GetSubject()
	if err != nil || sub != "refresh" {
		return nil, apperror.Unauthorized("无效的 token 类型")
	}

	// 3. Redis double-verify: stored token must match request token
	storedToken, err := s.sessionStore.GetRefresh(ctx, claims.UserID)
	if err != nil {
		s.logger.Warn("failed to get refresh token from redis", zap.Uint64("userId", claims.UserID), zap.Error(err))
	}
	if err != nil || storedToken != req.RefreshToken {
		return nil, apperror.Unauthorized("refresh token 已失效")
	}

	// 4. Rotation: delete old refresh token to prevent replay
	if err := s.sessionStore.DeleteRefresh(ctx, claims.UserID); err != nil {
		s.logger.Warn("failed to delete old refresh token", zap.Uint64("userId", claims.UserID), zap.Error(err))
	}

	// 5. Verify user status
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Unauthorized("用户不存在")
		}
		return nil, err
	}
	if user.Status != nil && *user.Status == entity.UserStatusDisabled {
		return nil, apperror.Unauthorized("账号已被禁用，请联系管理员")
	}

	// 6. Load latest permissions + scopes(与 Login 共用 resolveUserAccess 收敛入口)
	perms, scopes := s.resolveUserAccess(ctx, claims.UserID)
	// 7. Generate new tokens
	accessToken, refreshToken, err := s.issueTokens(ctx, claims.UserID, perms, scopes)
	if err != nil {
		return nil, apperror.Internal("生成 token 失败")
	}

	// 9. Store session
	if err := s.storeSession(ctx, claims.UserID, accessToken, refreshToken, perms); err != nil {
		return nil, apperror.Internal("会话存储失败", err)
	}

	return &response.RefreshTokenResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfgProv.GetInt(ctx, "sys.jwt.accessExpire", 7200),
	}, nil
}

// GetUserOverview 聚合个人中心页所需的全部展示数据:身份(含部门/角色显示名)、
// 登录统计、近 30 天逐日活动序列(零填充)与最近登录尝试。活动数据尽力而为:
// 日志查询失败时降级为零值,身份区块必须始终可渲染。
func (s *AuthService) GetUserOverview(ctx context.Context) (*response.UserOverviewResp, error) {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return nil, apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, translateNotFound(err, "用户不存在")
	}

	resp := &response.UserOverviewResp{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: util.JSONTime(user.CreatedAt),
		Roles:     make([]response.RoleBriefResp, 0, len(user.Roles)),
	}
	if user.RealName != nil {
		resp.RealName = *user.RealName
	}
	if user.Email != nil {
		resp.Email = *user.Email
	}
	if user.Phone != nil {
		resp.Phone = *user.Phone
	}
	if user.Avatar != nil {
		resp.Avatar = *user.Avatar
	}
	if user.LastLoginIP != nil {
		resp.LastLoginIP = *user.LastLoginIP
	}
	if user.LastLoginTime != nil {
		t := util.JSONTime(*user.LastLoginTime)
		resp.LastLoginTime = &t
	}
	if user.DeptID != nil && *user.DeptID > 0 {
		name, err := s.repo.GetDeptName(ctx, *user.DeptID)
		if err != nil {
			s.logger.Warn("failed to load dept name for overview", zap.Uint64("userId", userID), zap.Error(err))
		}
		resp.DeptName = name
	}
	for _, r := range user.Roles {
		resp.Roles = append(resp.Roles, response.RoleBriefResp{Code: r.Code, Name: r.Name})
	}

	// 30 天窗口:含今天在内的 30 个自然日(服务器本地时区,与 login_time 一致)。
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -29)

	if total, err := s.repo.CountUserLogins(ctx, userID); err != nil {
		s.logger.Warn("failed to count user logins", zap.Uint64("userId", userID), zap.Error(err))
	} else {
		resp.Stats.TotalLogins = total
	}

	daily := make([]response.DailyActivityResp, 30)
	index := make(map[string]*response.DailyActivityResp, 30)
	for i := range daily {
		daily[i].Date = since.AddDate(0, 0, i).Format("2006-01-02")
		index[daily[i].Date] = &daily[i]
	}
	resp.DailyActivity = daily

	logs, err := s.repo.FindUserLoginLogsSince(ctx, userID, since)
	if err != nil {
		s.logger.Warn("failed to load login activity", zap.Uint64("userId", userID), zap.Error(err))
	} else {
		const recentLimit = 10 // 与前端文案「最近 10 条登录记录」保持一致
		recent := make([]response.RecentLoginResp, 0, recentLimit)
		for i := range logs {
			entry := &logs[i]
			day := index[entry.LoginTime.Format("2006-01-02")]
			if entry.Code == 0 {
				resp.Stats.Logins30d++
				if day != nil {
					day.Success++
				}
			} else {
				resp.Stats.Failed30d++
				if day != nil {
					day.Failed++
				}
			}
			if len(recent) < recentLimit {
				item := response.RecentLoginResp{
					Time: util.JSONTime(entry.LoginTime),
					Code: entry.Code,
				}
				if entry.IP != nil {
					item.IP = *entry.IP
				}
				if entry.Browser != nil {
					item.Browser = *entry.Browser
				}
				if entry.OS != nil {
					item.OS = *entry.OS
				}
				if entry.Msg != nil {
					item.Msg = *entry.Msg
				}
				recent = append(recent, item)
			}
		}
		resp.RecentLogins = recent
	}
	return resp, nil
}
