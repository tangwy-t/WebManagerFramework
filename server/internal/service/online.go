package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/response"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/session"
	"github.com/tangwy-t/webmanager-server/internal/pkg/util"

	"go.uber.org/zap"
)

// maxOnlineScanUsers 是单次在线列表扫描 user_access:* 的用户数上限(防御性)。
const maxOnlineScanUsers = 5000

// SessionOnlineInterface 定义在消费方:在线用户管理需要的会话查询/吊销能力。
type SessionOnlineInterface interface {
	ListOnlineUserIDs(ctx context.Context, maxCount int64) ([]uint64, error)
	ListUserTokens(ctx context.Context, userID uint64) ([]string, error)
	BulkLoadSessionMeta(ctx context.Context, tokens []string) (map[string]*session.SessionMeta, error)
	IsAccessValid(ctx context.Context, token string) (bool, error)
	RevokeOne(ctx context.Context, userID uint64, token string) (bool, error)
	DeleteRefresh(ctx context.Context, userID uint64) error
}

// OnlineUserLookupInterface 定义在消费方:按 ID 批量查用户(受 DataScope 过滤)。
type OnlineUserLookupInterface interface {
	FindByIDs(ctx context.Context, ids []uint64) ([]entity.SysUser, error)
}

// OnlineConfigGetter 定义在消费方:读取 JWT 密钥以解析旧版 token 时间。
type OnlineConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
}

// OnlineWsKickPublisher 定义在消费方:下发 WS 踢人事件(复用现有 Hub.PublishKick,
// 跨实例广播,按 userId 断开其 WebSocket 连接)。nil 时跳过 WS 断开(测试/降级)。
type OnlineWsKickPublisher interface {
	PublishKick(ctx context.Context, userID uint64, reason string, kickToken string) error
}

// OnlineUserService 提供在线会话列表与强制下线。
type OnlineUserService struct {
	sessions SessionOnlineInterface
	users    OnlineUserLookupInterface
	cfg      OnlineConfigGetter
	wsKick   OnlineWsKickPublisher
	logger   logger.LoggerInterface
}

func NewOnlineUserService(sessions SessionOnlineInterface, users OnlineUserLookupInterface, cfg OnlineConfigGetter, wsKick OnlineWsKickPublisher, logger logger.LoggerInterface) *OnlineUserService {
	return &OnlineUserService{sessions: sessions, users: users, cfg: cfg, wsKick: wsKick, logger: logger}
}

// List 聚合在线会话为逻辑会话行并内存分页。
func (s *OnlineUserService) List(ctx context.Context, req *request.OnlineUserQuery) (*app.PageResponse, error) {
	uids, err := s.sessions.ListOnlineUserIDs(ctx, maxOnlineScanUsers)
	if err != nil {
		s.logger.Error("list online: scan indexes", zap.Error(err))
		return nil, apperror.Internal("查询在线状态失败", err)
	}

	users, err := s.users.FindByIDs(ctx, uids)
	if err != nil {
		s.logger.Error("list online: load users", zap.Error(err))
		return nil, apperror.Internal("查询在线状态失败", err)
	}
	userByID := make(map[uint64]*entity.SysUser, len(users))
	for i := range users {
		userByID[users[i].ID] = &users[i]
	}
	secret := s.cfg.GetString(ctx, jwt.SecretConfigKey, jwt.DefaultSecretFallback)

	sessions := make([]response.OnlineSession, 0)
	for _, uid := range uids {
		u := userByID[uid]
		if u == nil {
			continue // 不在可见范围 或 用户已删除
		}
		if u.Status != nil && *u.Status == entity.UserStatusDisabled {
			continue // 禁用用户残留会话不展示
		}
		groups, err := s.collectUserGroups(ctx, uid, secret)
		if err != nil {
			s.logger.Warn("list online: collect groups", zap.Uint64("userId", uid), zap.Error(err))
			continue
		}
		for _, g := range groups {
			sessions = append(sessions, response.OnlineSession{
				UserID:     uid,
				Username:   u.Username,
				RealName:   derefStr(u.RealName),
				DeptName:   deptNameOf(u.Dept),
				IP:         g.ip,
				Browser:    g.browser,
				OS:         g.os,
				LoginAt:    util.JSONTime(time.Unix(g.loginAt, 0)),
				ExpireAt:   util.JSONTime(time.Unix(g.expireAt, 0)),
				TokenCount: len(g.tokens),
				Sid:        g.sid,
			})
		}
	}

	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		k := strings.ToLower(kw)
		out := sessions[:0]
		for _, r := range sessions {
			if strings.Contains(strings.ToLower(r.Username), k) ||
				strings.Contains(strings.ToLower(r.RealName), k) ||
				strings.Contains(strings.ToLower(r.IP), k) {
				out = append(out, r)
			}
		}
		sessions = out
	}
	sort.Slice(sessions, func(i, j int) bool {
		return time.Time(sessions[i].LoginAt).After(time.Time(sessions[j].LoginAt))
	})

	total := len(sessions)
	start := (req.GetPage() - 1) * req.GetPageSize()
	if start > total {
		start = total
	}
	end := start + req.GetPageSize()
	if end > total {
		end = total
	}
	return app.NewPageResponse(sessions[start:end], int64(total), req.GetPage(), req.GetPageSize()), nil
}

// sessionGroup 是一组被聚合的 token(同一逻辑会话)。
type sessionGroup struct {
	tokens   []string
	ip       string
	browser  string
	os       string
	loginAt  int64
	expireAt int64
	sid      string
}

// collectUserGroups 把某用户的全部在线 token 聚合成逻辑会话组。
func (s *OnlineUserService) collectUserGroups(ctx context.Context, uid uint64, secret string) ([]sessionGroup, error) {
	tokens, err := s.sessions.ListUserTokens(ctx, uid)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}
	metas, err := s.sessions.BulkLoadSessionMeta(ctx, tokens)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]*sessionGroup)
	order := make([]string, 0, len(tokens))
	now := time.Now().Unix()
	for _, tok := range tokens {
		meta := metas[tok]
		var loginAt, expireAt int64
		ip, browser, os := "", "Unknown", "Unknown"
		if meta != nil {
			if meta.ExpireAt > 0 && meta.ExpireAt <= now {
				continue // 残留索引:已过期
			}
			ip, browser, os = meta.IP, meta.Browser, meta.OS
			loginAt, expireAt = meta.LoginAt, meta.ExpireAt
		} else {
			claims, e := jwt.ParseAccessToken(tok, secret)
			if e != nil {
				continue // 无法解析,跳过
			}
			if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
				continue // 已过期
			}
			if claims.IssuedAt != nil {
				loginAt = claims.IssuedAt.Time.Unix()
			}
			if claims.ExpiresAt != nil {
				expireAt = claims.ExpiresAt.Time.Unix()
			}
		}
		key := onlineGroupKey(uid, meta, tok)
		g, ok := byKey[key]
		if !ok {
			g = &sessionGroup{ip: ip, browser: browser, os: os, loginAt: loginAt, expireAt: expireAt}
			byKey[key] = g
			order = append(order, key)
		}
		g.tokens = append(g.tokens, tok)
		if loginAt < g.loginAt || g.loginAt == 0 {
			g.loginAt = loginAt
		}
		if expireAt > g.expireAt {
			g.expireAt = expireAt
		}
		if meta != nil {
			g.sid = session.SID(tok) // 组代表 = 有元数据的最新 token
		}
	}
	out := make([]sessionGroup, 0, len(order))
	for _, k := range order {
		g := byKey[k]
		if g.sid == "" && len(g.tokens) > 0 {
			g.sid = session.SID(g.tokens[len(g.tokens)-1]) // 纯旧版组:取末 token 为代表
		}
		out = append(out, *g)
	}
	return out, nil
}

// onlineGroupKey 计算逻辑会话的聚合键:有元数据按 (uid, ip, ua),无元数据各自成组。
func onlineGroupKey(uid uint64, meta *session.SessionMeta, token string) string {
	if meta == nil {
		return fmt.Sprintf("%d|legacy:%s", uid, token)
	}
	return fmt.Sprintf("%d|%s|%s", uid, meta.IP, meta.UserAgent)
}

// Kick 强制下线一个逻辑会话:吊销组内全部有效 access token + 删用户 refresh。
func (s *OnlineUserService) Kick(ctx context.Context, req *request.KickSessionReq, currentToken string) error {
	uid := uint64(req.UserID)
	if !validSID(req.Sid) {
		return apperror.BadRequest("参数错误")
	}
	if currentToken != "" && session.SID(currentToken) == req.Sid {
		return apperror.BadRequest("不能强制下线当前登录会话")
	}
	// DataScope 校验(防越权踢下线)
	visible, err := s.users.FindByIDs(ctx, []uint64{uid})
	if err != nil {
		return apperror.Internal("会话操作失败", err)
	}
	if len(visible) == 0 {
		return apperror.Forbidden("无权操作该用户会话")
	}

	tokens, err := s.sessions.ListUserTokens(ctx, uid)
	if err != nil {
		return apperror.Internal("会话操作失败", err)
	}
	var repr string
	for _, tok := range tokens {
		if session.SID(tok) == req.Sid {
			repr = tok
			break
		}
	}
	if repr == "" {
		return apperror.NotFound("会话不存在或已下线")
	}
	metas, err := s.sessions.BulkLoadSessionMeta(ctx, tokens)
	if err != nil {
		return apperror.Internal("会话操作失败", err)
	}
	repKey := onlineGroupKey(uid, metas[repr], repr)
	var target []string
	for _, tok := range tokens {
		if onlineGroupKey(uid, metas[tok], tok) == repKey {
			target = append(target, tok)
		}
	}

	// 白名单复核 + 逐个吊销(组内只删仍在白名单的,已失效跳过)
	anyRevoked := false
	for _, tok := range target {
		valid, err := s.sessions.IsAccessValid(ctx, tok)
		if err != nil {
			return apperror.Internal("会话操作失败", err)
		}
		if !valid {
			continue
		}
		if _, err := s.sessions.RevokeOne(ctx, uid, tok); err != nil {
			return apperror.Internal("会话操作失败", err)
		}
		anyRevoked = true
	}
	if !anyRevoked {
		return apperror.NotFound("会话不存在或已下线")
	}
	// 防前端自动 refresh 复活:注销该用户 refresh token
	if err := s.sessions.DeleteRefresh(ctx, uid); err != nil {
		s.logger.Warn("kick: delete refresh", zap.Uint64("userId", uid), zap.Error(err))
		return apperror.Internal("会话操作失败", err)
	}
	// 断开该用户 WebSocket 连接(复用 SSO 顶号的 PublishKick,跨实例)。
	// access/refresh 已吊销,WS 断不开仅是降级(前端收不到 kicked 提示),不阻断下线结果。
	if s.wsKick != nil {
		if err := s.wsKick.PublishKick(ctx, uid, "您已被管理员强制下线", ""); err != nil {
			s.logger.Warn("kick: publish ws event", zap.Uint64("userId", uid), zap.Error(err))
		}
	}
	return nil
}

func validSID(sid string) bool {
	if len(sid) != 64 {
		return false
	}
	_, err := hex.DecodeString(sid)
	return err == nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func deptNameOf(d *entity.SysDept) string {
	if d == nil {
		return ""
	}
	return d.Name
}
