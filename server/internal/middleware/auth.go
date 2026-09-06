package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"go.uber.org/zap"
)

// Gin-context keys. Values are set via c.Set and read via c.Get by HTTP
// middleware/handlers. Request-context (context.Context) variants live in
// internal/pkg/contextkeys — services and GORM audit callbacks read from
// there, which keeps the business layer free of transport-layer imports.
const (
	CtxUserID = "userID"
	CtxScopes = "scopes"
)

// Auth returns a middleware that validates a Bearer JWT token from the
// Authorization header and checks the session store whitelist before allowing
// the request to proceed. On success it stores the user ID in the gin context.
func Auth(cfgProv ConfigGetterInterface, tokenStore TokenStoreInterface, logger logger.LoggerInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("auth failed: missing token")
			app.Error(c, apperror.Unauthorized("未登录或 token 已过期"))
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		secret := cfgProv.GetString(c.Request.Context(), jwt.SecretConfigKey, jwt.DefaultSecretFallback)
		claims, err := jwt.ParseAccessToken(tokenStr, secret)
		if err != nil {
			logger.Warn("auth failed: invalid token", zap.Error(err))
			app.Error(c, apperror.Unauthorized("未登录或 token 已过期"))
			c.Abort()
			return
		}

		// Session store whitelist check
		valid, err := tokenStore.IsAccessValid(c.Request.Context(), tokenStr)
		if err != nil {
			logger.Error("auth failed: session store unavailable", zap.Error(err))
			app.Error(c, apperror.Internal("服务暂时不可用"))
			c.Abort()
			return
		}
		if !valid {
			logger.Warn("auth failed: token not in whitelist")
			app.Error(c, apperror.Unauthorized("token 已失效"))
			c.Abort()
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Request = c.Request.WithContext(contextkeys.WithUserID(c.Request.Context(), claims.UserID))
		c.Set(CtxScopes, claims.Scopes)
		// Write scopes info into request context (used by service/auth.go).
		for _, sc := range claims.Scopes {
			if sc.Dimension == "dept" {
				reqCtx := contextkeys.WithDataScope(c.Request.Context(), sc.Level)
				c.Request = c.Request.WithContext(contextkeys.WithDeptID(reqCtx, sc.SelfID))
				break
			}
		}
		c.Next()
	}
}
