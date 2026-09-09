package middleware

import (
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ScopeResolverHandler 返回 Gin 中间件处理函数，从 JWT claims 解析 ScopeContext 并注入 ctx。
func ScopeResolverHandler(sr *datascope.ScopeResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := extractClaims(c)
		if claims == nil {
			c.Next()
			return
		}

		sc := &datascope.ScopeContext{
			UserID:     claims.UserID,
			Dimensions: make(map[string]*datascope.ResolvedDimension),
		}

		for _, scopeClaim := range claims.Scopes {
			resolver, ok := sr.Resolvers[scopeClaim.Dimension]
			if !ok {
				continue
			}
			dim, err := resolver.Resolve(c.Request.Context(), scopeClaim.Level, scopeClaim.SelfID, claims.UserID)
			if err != nil {
				sr.Logger.Warn("scope resolver: resolve failed, falling back to empty result",
					zap.String("dimension", scopeClaim.Dimension),
					zap.Int8("level", scopeClaim.Level),
					zap.Uint64("selfID", scopeClaim.SelfID),
					zap.Uint64("userID", claims.UserID),
					zap.Error(err))
				// 解析失败降级为空结果
				sc.Dimensions[scopeClaim.Dimension] = &datascope.ResolvedDimension{
					Level:  scopeClaim.Level,
					SelfID: scopeClaim.SelfID,
				}
				continue
			}
			sc.Dimensions[scopeClaim.Dimension] = dim
		}

		c.Request = c.Request.WithContext(datascope.WithScopeContext(c.Request.Context(), sc))
		c.Next()
	}
}

// extractClaims 从 gin.Context 提取 Auth 中间件注入的完整 JWT claims。
// 单一来源:不再由 CtxScopes/CtxUserID 两个散值重建影子 Claims 对象。
func extractClaims(c *gin.Context) *jwt.Claims {
	raw, exists := c.Get(CtxClaims)
	if !exists {
		return nil
	}
	claims, ok := raw.(*jwt.Claims)
	if !ok {
		return nil
	}
	return claims
}
