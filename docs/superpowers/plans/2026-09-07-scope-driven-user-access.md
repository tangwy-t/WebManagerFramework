# Scope 驱动的用户访问解析(resolveUserAccess 收敛)实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把"用户权限点(perms)与数据范围(scopes)的生成"收敛到一个 `resolveUserAccess` 入口:菜单/权限过滤全部由 datascope 插件从 ScopeContext 自动注入,删除散落在 login/refresh/userinfo/permission-fallback 各处的手写 `sys_user_role`/`sys_role_menu` join。

**Architecture:** 登录/刷新时先用 `GetUserDataScope` + `GetUserRoleScope` 算出策略级别(scope 的输入,不可消除),由 `buildUserScopes` 生成 ScopeClaim,再复用与中间件相同的 `ScopeResolver` 解析出 `ScopeContext` 注入 ctx;`FindMenuPerms` 是裸查询,由 GORM scope 插件自动追加 `sys_menu.id IN (角色授权菜单 ID)`。admin 即 role 维度 ScopeAll → `AllowedIDs=nil` → 插件天然不过滤,服务层仅补 `"admin"` 通配标记。PermissionGuard 缓存回源改用请求 ctx(已携带 ScopeContext),与运行时同一过滤来源。

**Tech Stack:** Go 1.22+、GORM(scope 插件)、gin、`github.com/glebarez/sqlite`(测试)、JWT(golang-jwt/v5)。

## Global Constraints

- 不新增任何外部依赖;沿用仓库现有分层与"接口定义在消费方"约定。
- **本次不加 `menu.status` 过滤**:`FindMenuPerms` 与旧 `GetUserPermissions` 一样不按 status 过滤,停用菜单的 perms 行为保持不变。
- 注释沿用仓库现有中文注释风格;策略级降级语义与现状一致(权限加载失败 → 空 perms,不阻断发 token)。
- 测试命令:根目录 `make server-test`(等价 `cd server && go test ./... -v -count=1`)。
- 行为差异(有意为之,评审确认点):
  1. 非 admin 的 perms 集合从"角色直授菜单 ID → perms"变为"角色菜单 ID + **祖先链补齐** → perms"(与菜单树共用 `FindRoleMenuIDs`)。祖先目录若带 perms 会并入集合。
  2. `GetUserInfo` / PermissionGuard fallback 与登录侧 perms 语义完全一致(此前三条路径各自为政)。
  3. 登录时"加载角色编码"的死查询(`_ = roleCodes`)被删除。

---

### Task 1: 仓库层新增 `FindMenuPerms`(取权限点的裸查询)

**Files:**
- Modify: `server/internal/repository/auth.go`(新增方法,旧方法暂时保留)
- Modify: `server/internal/service/auth.go:58-72`(`AuthRepositoryInterface` 增加 `FindMenuPerms`)
- Modify: `server/internal/service/auth_test.go`(stubAuthRepo 增加桩)
- Test: `server/internal/repository/auth_scope_test.go`(新建)

**Interfaces:**
- Consumes: `datascope.ScopeContextFromCtx`、`datascope.NewScopePlugin`、`entity.ScopeEntities`
- Produces: `(*AuthRepo).FindMenuPerms(ctx context.Context) ([]string, error)` — 后续 Task 2/3/4 消费

- [ ] **Step 1: 写失败的仓库测试**(验证 scope 插件自动注入 `sys_menu.id IN`,而非手写 join)

创建 `server/internal/repository/auth_scope_test.go`:

```go
package repository

import (
	"context"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/database"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
	"gorm.io/gorm"
)

// TestFindMenuPerms_ScopeInjected 验证权限点查询不手写 role join:
// ctx 携带 ScopeContext 时,scope 插件自动注入 sys_menu.id IN (role 维度允许集合)。
func TestFindMenuPerms_ScopeInjected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	database.NewCallbacks(nil).Register(db)
	plugin := datascope.NewScopePlugin()
	plugin.RegisterEntities(entity.ScopeEntities...)
	plugin.RegisterPlugin(db)
	if err := db.AutoMigrate(&entity.SysMenu{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	seed := []entity.SysMenu{
		{BaseEntity: entity.BaseEntity{ID: 1}, Name: "菜单1", Perms: strPtr("sys:a:list")},
		{BaseEntity: entity.BaseEntity{ID: 2}, Name: "菜单2", Perms: strPtr("sys:b:list")},
		{BaseEntity: entity.BaseEntity{ID: 3}, Name: "菜单3", Perms: strPtr("sys:c:list")},
		{BaseEntity: entity.BaseEntity{ID: 4}, Name: "菜单4", Perms: nil},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed menu %d: %v", seed[i].ID, err)
		}
	}

	// role 维度只允许 {1, 3}:id=2 的 perms 应被 plugin 过滤,id=4 perms 为空被条件过滤。
	sc := &datascope.ScopeContext{UserID: 7, Dimensions: map[string]*datascope.ResolvedDimension{
		datascope.DimRole: {Level: datascope.ScopeCustom, AllowedIDs: []uint64{1, 3}},
	}}
	ctx := datascope.WithScopeContext(context.Background(), sc)

	perms, err := NewAuthRepository(db).FindMenuPerms(ctx)
	if err != nil {
		t.Fatalf("FindMenuPerms: %v", err)
	}

	want := []string{"sys:a:list", "sys:c:list"}
	sort.Strings(perms)
	sort.Strings(want)
	if len(perms) != len(want) {
		t.Fatalf("perms = %v, want %v", perms, want)
	}
	for i := range want {
		if perms[i] != want[i] {
			t.Fatalf("perms = %v, want %v", perms, want)
		}
	}
}
```

(注:`strPtr` 辅助函数已存在于 repository 包测试中,`notice_test.go` 同包已使用,无需新建。)

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/repository/ -run TestFindMenuPerms_ScopeInjected -count=1`
Expected: FAIL(编译错误:`FindMenuPerms undefined`)

- [ ] **Step 3: 实现 `FindMenuPerms`**

在 `server/internal/repository/auth.go` 的 `FindByUsername` 之后插入:

```go
// FindMenuPerms 返回当前 ctx 数据范围下所有非空菜单权限标识。
// 本方法不手写任何角色 join:ctx 携带 ScopeContext 时,scope 插件自动
// 注入 sys_menu.id IN (角色授权菜单 ID) 过滤,与菜单树查询共用同一来源。
func (r *AuthRepo) FindMenuPerms(ctx context.Context) ([]string, error) {
	var perms []string
	err := r.db.WithContext(ctx).Model(&entity.SysMenu{}).
		Where("perms IS NOT NULL AND perms != ''").
		Pluck("perms", &perms).Error
	return perms, err
}
```

- [ ] **Step 4: 更新消费方接口与测试桩**

`server/internal/service/auth.go` 的 `AuthRepositoryInterface`(第 58-72 行)中,在 `GetRoleCodes` 之后新增:

```go
	FindMenuPerms(ctx context.Context) ([]string, error)
```

`server/internal/service/auth_test.go` 的 `stubAuthRepo`(第 20-23 行)增加字段与桩:

```go
type stubAuthRepo struct {
	findByIDUser *entity.SysUser
	findByIDErr  error
	findMenuPermsFn func(context.Context) ([]string, error)
}
```

并在 `GetRoleCodes` 桩之后新增方法(注意 Step 3 是字段改动后的完整方法):

```go
func (m *stubAuthRepo) FindMenuPerms(ctx context.Context) ([]string, error) {
	if m.findMenuPermsFn == nil {
		return nil, nil
	}
	return m.findMenuPermsFn(ctx)
}
```

- [ ] **Step 5: 运行测试确认通过,并保证全包仍可编译**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/repository/ -run TestFindMenuPerms_ScopeInjected -count=1 && go build ./...`
Expected: PASS + 编译成功

- [ ] **Step 6: 提交**

```bash
cd /home/workspace/WebManagerFramework
git add server/internal/repository/auth.go server/internal/service/auth.go server/internal/service/auth_test.go server/internal/repository/auth_scope_test.go
git commit -m "feat(repo): add scope-injected FindMenuPerms for menu permissions"
```

---

### Task 2: AuthService 注入 ScopeResolver,实现 `resolveUserAccess` 与 scope 化的 `GetUserPermissions`

**Files:**
- Modify: `server/internal/service/auth.go`(结构体、构造函数、新方法、wrapper 改写)
- Modify: `server/internal/wireup/wireup.go`(ScopeResolver 构造前移 + 传入 NewAuthService)
- Modify: `server/internal/service/auth_test.go`(newVerifyService 传 nil;新增测试)
- Test: `server/internal/service/auth_test.go`(新增 4 个测试)

**Interfaces:**
- Consumes: Task 1 的 `FindMenuPerms`;`datascope.ScopeResolver` / `DimensionResolver` / `ScopeContext`;`jwt.ScopeClaim`;`buildUserScopes`
- Produces: `(*AuthService).resolveUserAccess(ctx, userID) ([]string, []jwt.ScopeClaim)`;`(*AuthService).scopeCtxFor(ctx, userID, scopes) context.Context`;`roleScopeAllFromCtx(ctx) bool` — Task 3 消费

- [ ] **Step 1: 写失败的测试**(resolveUserAccess 两条路径 + wrapper 两条路径)

在 `server/internal/service/auth_test.go` 末尾追加(import 需增加 `slices`、`datascope`):

```go
// ── scope 驱动的访问解析测试 ────────────────────────────────────

// stubRoleMenuRepo 满足 datascope.RoleMenuRepoInterface。
type stubRoleMenuRepo struct {
	ids   []uint64
	err   error
	calls int
}

func (m *stubRoleMenuRepo) FindRoleMenuIDs(context.Context, uint64) ([]uint64, error) {
	m.calls++
	return m.ids, m.err
}

// newAccessService 构造仅访问解析相关的 AuthService:注入 role/self 两个
// 维度的 resolver(dept 维度缺省跳过,scopeCtxFor 对缺失 resolver 静默跳过)。
func newAccessService(repo AuthRepositoryInterface, roleMenus *stubRoleMenuRepo) *AuthService {
	scopeResolver := datascope.NewScopeResolver([]datascope.DimensionResolver{
		datascope.NewRoleDimensionResolver(roleMenus),
		datascope.NewSelfDimensionResolver(),
	}, logger.NewNop())
	return NewAuthService(nil, repo, logger.NewNop(), nil, nil, nil, "", scopeResolver)
}

func TestResolveUserAccessCustomRoleScope(t *testing.T) {
	roleMenus := &stubRoleMenuRepo{ids: []uint64{11, 12, 15}}
	repo := &stubAuthRepo{
		dataScope: datascope.ScopeCustom,
		deptID:    9,
		roleScope: datascope.ScopeCustom,
		findMenuPermsFn: func(ctx context.Context) ([]string, error) {
			sc, ok := datascope.ScopeContextFromCtx(ctx)
			if !ok || sc == nil {
				t.Fatalf("FindMenuPerms 未收到携带 ScopeContext 的 ctx")
			}
			dim, ok := sc.Dimensions[datascope.DimRole]
			if !ok || dim == nil {
				t.Fatalf("role 维度缺失")
			}
			if len(dim.AllowedIDs) != 3 || dim.AllowedIDs[0] != 11 {
				t.Fatalf("AllowedIDs = %v, want role-menu 集合 [11 12 15]", dim.AllowedIDs)
			}
			return []string{"system:user:list"}, nil
		},
	}
	svc := newAccessService(repo, roleMenus)

	perms, scopes := svc.resolveUserAccess(context.Background(), 7)

	if len(perms) != 1 || perms[0] != "system:user:list" {
		t.Fatalf("perms = %v, want [system:user:list]", perms)
	}
	if slices.Contains(perms, "admin") {
		t.Fatal("ScopeCustom 用户不应获得 admin 标记")
	}
	if len(scopes) != 2 || scopes[1].Dimension != datascope.DimRole || scopes[1].Level != datascope.ScopeCustom {
		t.Fatalf("scopes = %+v, want [dept custom, role custom]", scopes)
	}
	if roleMenus.calls != 1 {
		t.Fatalf("role resolver 调用次数 = %d, want 1", roleMenus.calls)
	}
}

func TestResolveUserAccessAdminAppendsMarker(t *testing.T) {
	repo := &stubAuthRepo{
		dataScope: datascope.ScopeAll,
		deptID:    9,
		roleScope: datascope.ScopeAll,
		findMenuPermsFn: func(context.Context) ([]string, error) {
			return []string{"system:user:list"}, nil
		},
	}
	svc := newAccessService(repo, &stubRoleMenuRepo{})

	perms, _ := svc.resolveUserAccess(context.Background(), 1)

	if len(perms) != 2 || perms[1] != "admin" {
		t.Fatalf("perms = %v, want [system:user:list admin](ScopeAll → plugin 不加过滤 → 全量 + admin 标记)", perms)
	}
}

func TestGetUserPermissionsUsesScopeFromCtx(t *testing.T) {
	repo := &stubAuthRepo{findMenuPermsFn: func(ctx context.Context) ([]string, error) {
		if _, ok := datascope.ScopeContextFromCtx(ctx); !ok {
			t.Fatal("ctx 缺少 ScopeContext:GetUserPermissions 必须由调用方提供已注入 scope 的 ctx")
		}
		return []string{"system:user:list"}, nil
	}}
	svc := newAccessService(repo, &stubRoleMenuRepo{})

	sc := &datascope.ScopeContext{UserID: 1, Dimensions: map[string]*datascope.ResolvedDimension{
		datascope.DimRole: {Level: datascope.ScopeAll},
	}}
	ctx := datascope.WithScopeContext(context.Background(), sc)

	perms, err := svc.GetUserPermissions(ctx, 1)
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	if len(perms) != 2 || perms[1] != "admin" {
		t.Fatalf("perms = %v, want 含 admin 标记", perms)
	}
}

func TestGetUserPermissionsNoScopeCtxNoMarker(t *testing.T) {
	repo := &stubAuthRepo{findMenuPermsFn: func(context.Context) ([]string, error) {
		return []string{"system:user:list"}, nil
	}}
	svc := newAccessService(repo, &stubRoleMenuRepo{})

	perms, err := svc.GetUserPermissions(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	if len(perms) != 1 || slices.Contains(perms, "admin") {
		t.Fatalf("perms = %v, want 无 admin 标记(无 scope ctx 时防御性不加)", perms)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/service/ -run 'TestResolveUserAccess|TestGetUserPermissions' -count=1`
Expected: FAIL(`stubAuthRepo` 尚无 `dataScope/deptID/roleScope` 字段,`resolveUserAccess` 未定义)

- [ ] **Step 3: stub 增加可配置字段**

`server/internal/service/auth_test.go` 的 `stubAuthRepo` 结构体(第 20-23 行)改为:

```go
type stubAuthRepo struct {
	findByIDUser *entity.SysUser
	findByIDErr  error
	findMenuPermsFn func(context.Context) ([]string, error)
	dataScope    int8
	deptID       uint64
	roleScope    int8
}
```

并同步改造两个现有桩方法:

```go
func (m *stubAuthRepo) GetUserDataScope(context.Context, uint64) (int8, uint64, error) {
	return m.dataScope, m.deptID, nil
}
func (m *stubAuthRepo) GetUserRoleScope(context.Context, uint64) int8 { return m.roleScope }
```

- [ ] **Step 4: AuthService 增加 scopeResolver 依赖**

`server/internal/service/auth.go`:

结构体(第 74-82 行)末尾加字段:

```go
type AuthService struct {
	cfgProv      ConfigGetterInterface
	repo         AuthRepositoryInterface
	logger       logger.LoggerInterface
	sessionStore SessionStoreInterface
	loginLogSvc  LoginLogServiceInterface
	captchaSvc   CaptchaInterface
	apiPrefix    string // server.apiPrefix,用于拼接头像访问路径
	scopeResolver *datascope.ScopeResolver
}
```

构造函数(第 85-103 行)增加最后一个参数并赋值:

```go
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
```

- [ ] **Step 5: 实现 scopeCtxFor / resolveUserAccess / roleScopeAllFromCtx,改写 wrapper**

在 `server/internal/service/auth.go` 的 `buildUserScopes`(第 146-159 行)之后插入:

```go
// scopeCtxFor 按 scopes 声明构造 ScopeContext 注入 ctx,复用与
// middleware.ScopeResolverHandler 完全相同的 resolver,权限口径与运行时一致。
// resolver 解析失败降级为 nil 维度(该维度放行),scopeResolver 未注入(测试)时原样返回。
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
			dim = &datascope.ResolvedDimension{Level: claim.Level, SelfID: claim.SelfID}
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
```

同时改写 `GetUserPermissions`(第 384-386 行):

```go
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
```

(注:`context`、`jwt`、`datascope`、`zap` 均已 import,无需新增。)

- [ ] **Step 6: 装配调整 —— wireup 将 ScopeResolver 构造前移并传入 authSvc**

`server/internal/wireup/wireup.go`:

在 `authSvc := ...`(第 106 行)之前插入:

```go
	// ── Scope Resolver ──(前移到 AuthService 之前:登录/刷新时的访问解析
	// resolveUserAccess 依赖它构造 ScopeContext;依赖只需 authRepo/deptRepo/log)
	deptResolver := datascope.NewDeptDimensionResolver(authRepo, deptRepo)
	roleResolver := datascope.NewRoleDimensionResolver(authRepo)
	selfResolver := datascope.NewSelfDimensionResolver()
	scopeResolver := datascope.NewScopeResolver([]datascope.DimensionResolver{deptResolver, selfResolver, roleResolver}, log)
```

第 106 行改为:

```go
	authSvc := service.NewAuthService(configSvc, authRepo, log, sessionStore, loginLogSvc, captchaPkg, cfg.Server.APIPrefix, scopeResolver)
```

删除原第 179-183 行的旧构造块("── Scope Resolver ──"),加注释说明已前移。

`server/internal/service/auth_test.go` 的 `newVerifyService`(第 61-63 行)最后一参数补 `nil`:

```go
	return NewAuthService(nil, repo, logger.NewNop(), nil, nil, nil, "", nil)
```

- [ ] **Step 7: 运行测试确认通过**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/service/ -run 'TestResolveUserAccess|TestGetUserPermissions' -count=1 && go build ./...`
Expected: PASS + 编译成功

- [ ] **Step 8: 提交**

```bash
cd /home/workspace/WebManagerFramework
git add server/internal/service/auth.go server/internal/service/auth_test.go server/internal/wireup/wireup.go
git commit -m "feat(auth): add scope-driven resolveUserAccess to AuthService"
```

---

### Task 3: Login / Refresh / GetUserInfo 切到统一入口,删除散落 join

**Files:**
- Modify: `server/internal/service/auth.go`(三处调用点替换;删除旧代码路径)
- Modify: `server/internal/service/auth_test.go`(删除已不存在的桩方法)
- Modify: `server/internal/repository/auth.go`(删除 `GetUserPermissions` 与 `allMenuPerms`)

**Interfaces:**
- Consumes: Task 2 的 `resolveUserAccess` / `GetUserPermissions`
- Produces: 无(纯收敛)

- [ ] **Step 1: Login 替换**(第 229-255 行)

删掉"Load permissions / Load data scope / Load role codes / Generate JWT tokens"四个块(第 229-255 行),替换为:

```go
	// Load permissions + data scope claims in one consolidated pass
	// (role codes 死查询一并移除:此前加载后仅 `_ = roleCodes` 丢弃)。
	perms, scopes := s.resolveUserAccess(ctx, user.ID)
	accessToken, refreshToken, err := s.issueTokens(ctx, user.ID, perms, scopes)
	if err != nil {
		return nil, apperror.Internal("生成 token 失败")
	}
```

注意:替换前 `err` 已声明,`perms, scopes :=` 引入新变量,`accessToken, refreshToken, err :=` 中 err 与已有变量共存,`:=` 语法合法(accessToken 为新变量)。

- [ ] **Step 2: RefreshToken 替换**(第 596-612 行)

删除"6. Load latest permissions / 7. Load data scope / 8. Generate new scopes"三块,替换为:

```go
	// 6. Load latest permissions + scopes(与 Login 共用 resolveUserAccess 收敛入口)
	perms, scopes := s.resolveUserAccess(ctx, claims.UserID)
	// 7. Generate new tokens
	accessToken, refreshToken, err := s.issueTokens(ctx, claims.UserID, perms, scopes)
	if err != nil {
		return nil, apperror.Internal("生成 token 失败")
	}
```

原第 618-620 行的"9. Store session"块保持原样(变量名不变)。

- [ ] **Step 3: GetUserInfo 替换**(第 366 行)

```go
	perms, err := s.repo.GetUserPermissions(ctx, userID)
```

改为:

```go
	perms, err := s.GetUserPermissions(ctx, userID)
```

(该请求 ctx 已经过 router.go:190 的 ScopeResolverHandler,携带 ScopeContext。)

- [ ] **Step 4: 删除仓库层散落实现**

`server/internal/repository/auth.go` 删除:
- `GetUserPermissions`(第 48-83 行);
- `allMenuPerms`(第 85-92 行)。

`server/internal/service/auth.go` 的 `AuthRepositoryInterface`(第 58-72 行)删除:

```go
	GetUserPermissions(ctx context.Context, userID uint64) ([]string, error)
```

`server/internal/service/auth_test.go` 删除:

```go
func (m *stubAuthRepo) GetUserPermissions(context.Context, uint64) ([]string, error) {
	return nil, nil
}
```

- [ ] **Step 5: 编译 + 全量测试**

Run: `cd /home/workspace/WebManagerFramework && make server-test`
Expected: PASS(含既有 auth/handler/repository 测试)

- [ ] **Step 6: 提交**

```bash
cd /home/workspace/WebManagerFramework
git add server/internal/service/auth.go server/internal/service/auth_test.go server/internal/repository/auth.go
git commit -m "refactor(auth): route login/refresh/userinfo perms through resolveUserAccess"
```

---

### Task 4: PermissionGuard 缓存回源走请求 ctx(scope 注入)

**Files:**
- Modify: `server/internal/middleware/permission.go:71`
- Test: `server/internal/middleware/permission_test.go`(新建)

**Interfaces:**
- Consumes: `datascope.ScopeContextFromCtx`,`(*PermissionGuard).Permission`
- Produces: 无

- [ ] **Step 1: 写失败的中间件测试**(证明回源 ctx 必须携带 ScopeContext)

创建 `server/internal/middleware/permission_test.go`:

```go
package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
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
	c.Set(CtxUserID, uint64(1))
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/middleware/ -run TestPermissionGuardFallbackCarriesScopeContext -count=1`
Expected: FAIL(`gotScopeCtx=false:当前实现传 context.Background()`)

- [ ] **Step 3: 改用请求 ctx**

`server/internal/middleware/permission.go` 第 70-81 行块改为:

```go
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
```

(Redis 的 StorePerms 仍用 Background,与 scope 无关,保持不变。)

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/workspace/WebManagerFramework/server && go test ./internal/middleware/ -count=1`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /home/workspace/WebManagerFramework
git add server/internal/middleware/permission.go server/internal/middleware/permission_test.go
git commit -m "fix(middleware): scope-scoped permission fallback via request ctx"
```

---

### Task 5: 全量验证与行为对照检查

**Files:** 无新增;如检查中发现注释/文档过时则就地修正

- [ ] **Step 1: 全量测试 + 静态检查**

Run: `cd /home/workspace/WebManagerFramework && make server-test && make server-vet`
Expected: 全部 PASS、vet 无输出

- [ ] **Step 2: 遗留引用扫描**

Run: `cd /home/workspace/WebManagerFramework && grep -rn "GetUserPermissions\|allMenuPerms" server/ docs/ README.md 2>/dev/null`
Expected: 仅剩三个合法点——service 的 wrapper 定义与两处调用(GetUserInfo、middleware)、`middleware/interface.go` 的窄接口声明、`handler/auth.go:40` 注释(无需改)。

- [ ] **Step 3: 手工冒烟清单(对照"现状 vs 改后"表)**

1. 非 admin 用户登录:`perms` 只含其角色授权菜单(经 `FindRoleMenuIDs`,含祖先补齐);`scopes` 含 dept(或 self)+ role 两条。
2. admin 登录:perms 含全量菜单权限点 + `"admin"` 标记。
3. 清空 Redis 的 perms 缓存后调用任一 `perm(...)` 保护接口:回源命中 scope 化路径,非 admin 只能通过其授权接口。
4. `GET /user/info` 返回的 permissions 与登录时一致(同一 scope 来源)。
5. 刷新 token 后 token 内 scopes/perms 与登录一致(此前可能发散)。

- [ ] **Step 4: 若有修正则提交,否则结束并汇报对照结果**

```bash
cd /home/workspace/WebManagerFramework && git status --short
```

---

## Self-Review(计划自检)

**1. Spec coverage:** 用户要求三点——①`resolveUserAccess` 收敛(Task 2 实现 + Task 3 接线);②删散落 join(Task 3 删 `GetUserPermissions`/`allMenuPerms`/登录死查询);③fallback 走 scope(Task 4);④不加 status 过滤(Global Constraints 明确,`FindMenuPerms` 不带 status 条件)。全部覆盖。

**2. Placeholder scan:** 无 TBD/TODO;所有步骤均有完整代码与确切命令。

**3. Type consistency:** `FindMenuPerms(ctx context.Context) ([]string, error)` 在 Task 1 定义、Task 2/3 消费一致;`resolveUserAccess(ctx, userID) ([]string, []jwt.ScopeClaim)` 定义与 Login/Refresh 调用一致;`NewAuthService` 第 8 参在两处调用点(Task 2 Step 6)同步更新;stub 字段 `dataScope/deptID/roleScope/findMenuPermsFn` 在 Task 1/2 步骤间一致;middleware 测试引用 `CtxUserID` 常量与 permission.go 现有用法一致;`logger.NewNop()`、`strPtr` 均为仓库既有测试示例中已验证存在的符号。

**确认过的代码事实:**
- `buildUserScopes` 位置:service/auth.go:146-159;签名 `(userID uint64, dataScope int8, deptID uint64, roleScope int8) []jwt.ScopeClaim`
- `datascope.DimRole = "role"`,`ScopeAll=1`,`ScopeCustom=2`,`ScopeSelf=5`
- scope plugin 注册于 cmd/server/main.go:109-110;运行时所有 `sys_menu` 查询自动注入
- `ScopeResolverHandler` 挂在 auth 组(router.go:190),先于路由级 `perm()` 执行
- `AuthRepositoryInterface` 消费链:service/auth.go:58、auth_test stub、wireup 的 `authRepo := repository.NewAuthRepository(db)`