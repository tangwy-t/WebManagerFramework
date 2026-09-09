package middleware

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// PermissionGuard coalesces concurrent cache-miss loads for the same user into a
// single backend call, preventing thundering-herd DB queries.
// Create one instance at startup with its dependencies injected; routes then
// only pass the required permission code (no per-route dependency plumbing).
type PermissionGuard struct {
	sfGroup   singleflight.Group
	authSvc   AuthServiceInterface
	permStore SessionStoreInterface
	cfgProv   ConfigGetterInterface
	logger    logger.LoggerInterface
}

// NewPermissionGuard creates a PermissionGuard with its collaborators.
func NewPermissionGuard(authSvc AuthServiceInterface, permStore SessionStoreInterface, cfgProv ConfigGetterInterface, logger logger.LoggerInterface) *PermissionGuard {
	return &PermissionGuard{
		authSvc:   authSvc,
		permStore: permStore,
		cfgProv:   cfgProv,
		logger:    logger,
	}
}

// Permission returns a middleware that enforces RBAC permission checks.
// It loads the user's permission set from the session store (falling back to
// AuthService) and verifies that either the "admin" super-permission or the
// required permission string is present.
func (g *PermissionGuard) Permission(requiredPerm string) gin.HandlerFunc {
	authSvc, permStore, cfgProv, logger := g.authSvc, g.permStore, g.cfgProv, g.logger
	return func(c *gin.Context) {
		claimsRaw, exists := c.Get(CtxClaims)
		if !exists {
			logger.Error("permission check: claims not found in context")
			app.Error(c, apperror.Unauthorized("未登录或 token 已过期"))
			c.Abort()
			return
		}
		claims, ok := claimsRaw.(*jwt.Claims)
		if !ok {
			logger.Error("permission check: claims has unexpected type", zap.Any("claims", claimsRaw))
			app.Error(c, apperror.Internal("服务器内部错误"))
			c.Abort()
			return
		}
		uid := claims.UserID
		logger.Debug("permission check", zap.Uint64("userId", uid), zap.String("requiredPerm", requiredPerm))

		// Check session store cache
		perms, err := permStore.LoadPerms(c.Request.Context(), uid)
		if err != nil {
			logger.Warn("permission cache read failed, falling back to service", zap.Error(err))
		}
		if perms == nil {
			// singleflight:合并同一用户的并发缓存 miss 为一次 DB 查询
			v, err, _ := g.sfGroup.Do(strconv.FormatUint(uid, 10), func() (any, error) {
				// 回源必须携带请求 ctx:经 ScopeResolverHandler(router 组级中间件)
				// 注入 ScopeContext,scope 插件据此过滤 sys_menu,权限点与运行时同源。
				// 传 Background 会让 scope 静默失效(旧行为)。
				p, loadErr := authSvc.GetUserPermissions(c.Request.Context(), uid)
				if loadErr != nil {
					return nil, loadErr
				}
				// 写入缓存
				ttl := time.Duration(cfgProv.GetInt(context.Background(), "sys.jwt.accessExpire", 7200)) * time.Second
				if storeErr := permStore.StorePerms(context.Background(), uid, p, ttl); storeErr != nil {
					logger.Warn("failed to cache permissions", zap.Error(storeErr))
				}
				return p, nil
			})
			if err != nil {
				logger.Error("failed to load permissions", zap.Error(err))
				app.Error(c, apperror.Internal("服务器内部错误"))
				c.Abort()
				return
			}
			perms = v.([]string)
		}

		// admin role bypass
		if slices.Contains(perms, "admin") {
			c.Next()
			return
		}

		if !slices.Contains(perms, requiredPerm) {
			app.Error(c, apperror.Forbidden("无操作权限"))
			c.Abort()
			return
		}
		c.Next()
	}
}
