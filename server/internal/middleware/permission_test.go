package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
	"github.com/tangwy-t/webmanager-server/internal/pkg/jwt"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
)

// stubAuthSvc 记录回源 ctx 是否携带 ScopeContext。
type stubAuthSvc struct {
	gotScopeCtx bool
	perms       []string
}

func (s *stubAuthSvc) GetUserPermissions(ctx context.Context, userID uint64) ([]string, error) {
	_, s.gotScopeCtx = datascope.ScopeContextFromCtx(ctx)
	return s.perms, nil
}

type stubPermStore struct{}

func (stubPermStore) LoadPerms(context.Context, uint64) ([]string, error) { return nil, nil }
func (stubPermStore) StorePerms(context.Context, uint64, []string, time.Duration) error {
	return nil
}

type stubCfgGateway struct{}

func (stubCfgGateway) GetString(context.Context, string, string) string { return "" }
func (stubCfgGateway) GetInt(context.Context, string, int) int          { return 7200 }
func (stubCfgGateway) GetBool(context.Context, string, bool) bool       { return false }

// TestPermissionGuardFallbackCarriesScopeContext 缓存 miss 回源时,必须把
// 请求 ctx(含 ScopeContext)传给 GetUserPermissions —— 此前传 Background
// 导致 scope 插件不注入,回源权限点与运行时语义脱节。
func TestPermissionGuardFallbackCarriesScopeContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSvc := &stubAuthSvc{perms: []string{"system:user:list"}}
	guard := NewPermissionGuard(authSvc, stubPermStore{}, stubCfgGateway{}, logger.NewNop())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(CtxClaims, &jwt.Claims{UserID: 1})
	sc := &datascope.ScopeContext{UserID: 1, Dimensions: map[string]*datascope.ResolvedDimension{
		datascope.DimRole: {Level: datascope.ScopeCustom, AllowedIDs: []uint64{1, 2}},
	}}
	ctx := datascope.WithScopeContext(context.Background(), sc)
	c.Request = httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	guard.Permission("system:user:list")(c)

	if !authSvc.gotScopeCtx {
		t.Fatal("fallback 回源 ctx 未携带 ScopeContext:PermissionGuard 必须传请求 ctx 而非 Background")
	}
	if c.IsAborted() {
		t.Fatal("权限应通过,实际被 Abort")
	}
}
