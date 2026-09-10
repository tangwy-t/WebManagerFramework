package router

import (
	_ "embed"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
	"github.com/tangwy-t/webmanager-server/internal/pkg/limiter"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tangwy-t/webmanager-server/docs"
	"github.com/tangwy-t/webmanager-server/internal/handler"
	"github.com/tangwy-t/webmanager-server/internal/middleware"
	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/config"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/session"
	"github.com/tangwy-t/webmanager-server/internal/pkg/version"
	wsPkg "github.com/tangwy-t/webmanager-server/internal/pkg/ws"
	"github.com/tangwy-t/webmanager-server/internal/service"
	"go.uber.org/zap"
)

//go:embed scalar.html
var scalarHTML string

// HTTP 动词约定(新增路由请遵守):
//   GET    读取(列表/详情/导出)
//   POST   创建 + 一切"命令"——状态迁移(enable/disable/pause/resume/
//          publish/revoke/unlock)、触发(run)、校验型动作(改密/重置)。
//          命令端点通常无请求体,不承诺幂等,这正是它与 PUT 的分界。
//   PUT    整体更新资源内容(UpdateUserInfo/UpdateRole 等表单回写)
//   DELETE 删除
//
// 此前状态命令一半 PUT 一半 POST(users/enable 用 PUT、notices/publish
// 用 POST)。统一为 POST:PUT 的 HTTP 契约是"以请求体替换该 URI 的资源",
// 无请求体的命令不满足该契约;业界(GitHub API、Google AIP-136 自定义
// 方法)对状态迁移也统一用 POST。

// ── Sub-structs grouped by route module ─────────────────────────────

// InfraDeps holds infrastructure dependencies shared across routes.
type InfraDeps struct {
	AuthSvc       *service.AuthService
	SessionStore  *session.SessionStore
	ConfigProv    *service.ConfigService
	Cfg           *config.Config
	OpLogSvc      *service.OperationLogService
	Logger        *logger.Logger
	Hub           *wsPkg.Hub
	ScopeResolver *datascope.ScopeResolver
	RateLimiter   *limiter.RateLimiter
	PermGuard     *middleware.PermissionGuard
}

// AuthDeps holds authentication handler dependencies.
type AuthDeps struct {
	AuthHdl *handler.AuthHandler
}

// SystemDeps holds user/role/menu/dept handler dependencies.
type SystemDeps struct {
	UserHdl *handler.UserHandler
	RoleHdl *handler.RoleHandler
	MenuHdl *handler.MenuHandler
	DeptHdl *handler.DeptHandler
}

// DictDeps holds dictionary type and data handler dependencies.
type DictDeps struct {
	DictTypeHdl *handler.DictTypeHandler
	DictDataHdl *handler.DictDataHandler
	DictCodeHdl *handler.DictCodeHandler
}

// MonitorDeps holds operation log, login log, notice, server monitor, cache, SQL monitor, and pprof handler dependencies.
type MonitorDeps struct {
	OpLogHdl         *handler.OperationLogHandler
	LoginLogHdl      *handler.LoginLogHandler
	NoticeHdl        *handler.NoticeHandler
	ServerMonitorHdl *handler.ServerMonitorHandler
	CacheHdl         *handler.CacheHandler
	SqlMonitorHdl    *handler.SQLMonitorHandler
	PprofHdl         *handler.PprofHandler
}

// JobDeps holds job handler dependencies.
type JobDeps struct {
	JobHdl *handler.JobHandler
}

// ConfigDeps holds config handler dependencies.
type ConfigDeps struct {
	ConfigHdl *handler.ConfigHandler
}

// FileDeps holds file management handler dependencies.
type FileDeps struct {
	FileHdl *handler.FileHandler
}

// Dependencies holds all injected dependencies for router setup, grouped by module.
type Dependencies struct {
	Infra   InfraDeps
	Auth    AuthDeps
	System  SystemDeps
	Dict    DictDeps
	Monitor MonitorDeps
	Job     JobDeps
	Config  ConfigDeps
	File    FileDeps
}

// Setup registers all middleware and routes on a new gin.Engine.
// All dependencies are injected via the Dependencies struct — no global state.
func Setup(deps Dependencies) *gin.Engine {
	r := gin.New()

	// Gin 默认信任所有代理：ClientIP 会被 X-Forwarded-For 伪造，进而绕过
	// 按 IP 的登录限流与验证码频控。仅信任配置中声明的代理；未配置时
	// 不信任任何代理。
	if len(deps.Infra.Cfg.Server.TrustedProxies) == 0 {
		_ = r.SetTrustedProxies(nil)
	} else if err := r.SetTrustedProxies(deps.Infra.Cfg.Server.TrustedProxies); err != nil {
		deps.Infra.Logger.Error("failed to set trusted proxies", zap.Error(err))
	}

	// 未分类错误（非 AppError）统一落日志，否则线上 500 无法归因。
	app.SetLogger(deps.Infra.Logger)

	// ── Global middleware ──────────────────────────────────────────
	r.Use(middleware.CORS(nil, deps.Infra.Logger))
	r.Use(deps.Infra.RateLimiter.GlobalRateLimit())
	r.Use(middleware.Recovery(deps.Infra.Logger))
	r.Use(middleware.TraceInit()) // 新增：traceId 注入
	r.Use(middleware.RequestLogger(deps.Infra.Logger))

	api := r.Group(deps.Infra.Cfg.Server.APIPrefix)

	// ── Public routes (no auth required) ───────────────────────────
	{
		// Health check
		// 以下端点有意不收录进 swagger(swagger 只描述业务 API):
		//   /health               探活
		//   /ws                   WebSocket 升级
		//   /monitor/debug/pprof/*  gin 原生 pprof 包装
		//   /swagger/*any、/scalar  API 文档自身(仅非 release 模式)
		// 均无业务语义。
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		// API 文档仅开发/测试模式暴露;release 生产环境暴露完整接口
		// 面与参数结构是免费的侦察材料。
		if deps.Infra.Cfg.Server.Mode != "release" {
			// Swagger spec 元数据:docs.go 里的静态默认值由 swag init 生成,
			// 这里在运行时用真实配置/构建信息覆盖,无需人工保持注解与配置同步:
			//   Host     置空 → Swagger UI/Scalar 回退到当前访问域名(含端口),
			//            本地直连、反代、任意端口均自动适配;
			//   BasePath 跟随 server.apiPrefix 配置;
			//   Version  跟随 -ldflags 注入的构建版本(见 Makefile LDFLAGS)。
			basePath := deps.Infra.Cfg.Server.APIPrefix
			if basePath == "" {
				basePath = "/"
			}
			docs.SwaggerInfo.Host = ""
			docs.SwaggerInfo.BasePath = basePath
			docs.SwaggerInfo.Version = version.Version

			// Swagger JSON spec (generated by swag init)
			api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
			// Scalar UI — modern API documentation viewer
			api.GET("/scalar", func(c *gin.Context) {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusOK, scalarHTML)
			})
		}

		api.POST("/login", deps.Infra.RateLimiter.LoginRateLimit(), deps.Auth.AuthHdl.Login)
		api.POST("/refresh", deps.Infra.RateLimiter.RefreshRateLimit(), deps.Auth.AuthHdl.RefreshToken)
		api.POST("/captcha/generate", deps.Auth.AuthHdl.CaptchaGenerate)
		// 头像图片输出:<img> 无法携带 Authorization 头,故公开(仅头像类低敏文件)。
		api.GET("/user/avatar/:id", deps.Auth.AuthHdl.GetAvatar)
	}

	// ── Authenticated routes ───────────────────────────────────────
	auth := api.Group("")
	auth.Use(middleware.Auth(deps.Infra.ConfigProv, deps.Infra.SessionStore, deps.Infra.Logger))
	auth.Use(middleware.OperationLogMiddleware(deps.Infra.OpLogSvc, deps.Infra.Logger))
	auth.Use(middleware.ScopeResolverHandler(deps.Infra.ScopeResolver))
	{
		auth.POST("/logout", middleware.SetModuleName("认证"), deps.Auth.AuthHdl.Logout)

		// 个人中心:自助资料/头像/密码/总览(与管理端 /users「用户管理」区分)。
		// 操作日志模块名由此分组注入 —— 顶层 auth 直挂路由不经过任何
		// SetModuleName,曾导致这些操作的系统模块记录为空。
		userCenter := auth.Group("/user")
		userCenter.Use(middleware.SetModuleName("个人中心"))
		{
			userCenter.GET("/info", deps.Auth.AuthHdl.GetUserInfo)
			userCenter.GET("/overview", deps.Auth.AuthHdl.GetUserOverview)
			userCenter.PUT("/info", deps.Auth.AuthHdl.UpdateProfile)
			userCenter.POST("/info/avatar", deps.Auth.AuthHdl.UploadAvatar)
			userCenter.POST("/password", deps.Auth.AuthHdl.ChangePassword)
			userCenter.POST("/password/verify", deps.Infra.RateLimiter.LoginRateLimit(), deps.Auth.AuthHdl.VerifyPassword)
		}

		// ── Permission-protected routes ────────────────────────────
		// Each route carries its own required permission code; permission
		// middleware is applied per-route rather than at the group level
		// so that list/create/update/delete are independently gated.
		perm := deps.Infra.PermGuard.Permission

		// 用户管理
		users := auth.Group("/users")
		users.Use(middleware.SetModuleName("用户管理"))
		{
			users.GET("", perm("system:user:list"), deps.System.UserHdl.List)
			users.GET("/:id", perm("system:user:query"), deps.System.UserHdl.GetByID)
			users.POST("", perm("system:user:add"), deps.System.UserHdl.Create)
			users.PUT("/:id", perm("system:user:edit"), deps.System.UserHdl.UpdateUserInfo)
			users.PUT("/:id/roles", perm("system:user:assign"), deps.System.UserHdl.AssignRoles)
			users.DELETE("/:id", perm("system:user:delete"), deps.System.UserHdl.Delete)
			users.POST("/:id/enable", perm("system:user:enable"), deps.System.UserHdl.Enable)
			users.POST("/:id/disable", perm("system:user:disable"), deps.System.UserHdl.Disable)
			users.POST("/:id/password/reset", perm("system:user:reset"), deps.System.UserHdl.ResetPassword)
			users.POST("/:id/unlock", perm("system:user:unlock"), deps.System.UserHdl.Unlock)
		}

		// 角色管理
		roles := auth.Group("/roles")
		roles.Use(middleware.SetModuleName("角色管理"))
		{
			roles.GET("", perm("system:role:list"), deps.System.RoleHdl.List)
			// 角色下拉·白名单(仅登录,响应已按 RoleOptionResp 瘦身不含授权明细):
			// 用户/公告表单均需引用角色列表做选择器。
			roles.GET("/all", deps.System.RoleHdl.GetAll)
			roles.GET("/:id", perm("system:role:query"), deps.System.RoleHdl.GetByID)
			roles.POST("", perm("system:role:add"), deps.System.RoleHdl.Create)
			roles.PUT("/:id", perm("system:role:edit"), deps.System.RoleHdl.Update)
			roles.PUT("/sort", perm("system:role:edit"), deps.System.RoleHdl.UpdateSort)
			roles.PUT("/:id/status", perm("system:role:edit"), deps.System.RoleHdl.UpdateStatus)
			roles.DELETE("/:id", perm("system:role:delete"), deps.System.RoleHdl.Delete)
			// 分配用户(角色维度):复用 UserHdl,join 表 sys_user_role 归用户域维护
			roles.GET("/:id/users", perm("system:role:edit"), deps.System.UserHdl.ListByRole)
			roles.POST("/:id/users", perm("system:role:edit"), deps.System.UserHdl.AddRoleUsers)
			roles.DELETE("/:id/users", perm("system:role:edit"), deps.System.UserHdl.RemoveRoleUsers)
		}

		// 菜单管理
		menus := auth.Group("/menus")
		menus.Use(middleware.SetModuleName("菜单管理"))
		{
			// 全树只读·白名单(仅登录):前端路由守卫在 backend 模式下为每一位
			// 登录用户调用,不能挂 system:menu:list,否则普通用户路由构建 403→500。
			// 无需服务端二次裁剪:sys_menu 已声明 role 维度数据范围,datascope
			// 插件对查询自动注入过滤 —— admin 角色(ScopeAll)得全树,普通用户
			// 仅返回其角色已分配的菜单(等价若依 selectMenuTreeByUserId)。
			// 角色授权表单同样消费此树(对齐若依 treeselect,登录即可)。
			menus.GET("", deps.System.MenuHdl.FindTree)
			menus.GET("/:id", perm("system:menu:query"), deps.System.MenuHdl.GetByID)
			menus.POST("", perm("system:menu:add"), deps.System.MenuHdl.Create)
			menus.PUT("/sort", perm("system:menu:edit"), deps.System.MenuHdl.UpdateSort)
			menus.PUT("/:id", perm("system:menu:edit"), deps.System.MenuHdl.Update)
			menus.DELETE("/:id", perm("system:menu:delete"), deps.System.MenuHdl.Delete)
		}

		// 部门管理
		depts := auth.Group("/depts")
		depts.Use(middleware.SetModuleName("部门管理"))
		{
			// 部门树只读·白名单(仅登录):用户/角色/公告三个表单都要引用部门树。
			// 树内容无需手工裁剪:sys_dept 声明 dept/self 维度数据范围,datascope
			// 自动按当前用户部门范围过滤(只暴露其可见的组织子树)。写操作仍挂权限码。
			depts.GET("", deps.System.DeptHdl.FindTree)
			depts.GET("/:id", perm("system:dept:query"), deps.System.DeptHdl.GetByID)
			depts.POST("", perm("system:dept:add"), deps.System.DeptHdl.Create)
			depts.PUT("/sort", perm("system:dept:edit"), deps.System.DeptHdl.UpdateSort)
			depts.PUT("/:id", perm("system:dept:edit"), deps.System.DeptHdl.Update)
			depts.DELETE("/:id", perm("system:dept:delete"), deps.System.DeptHdl.Delete)
		}

		// 操作日志
		opLogs := auth.Group("/logs/operation")
		opLogs.Use(middleware.SetModuleName("操作日志"))
		{
			opLogs.GET("", perm("system:log:operation:list"), deps.Monitor.OpLogHdl.FindPage)
			opLogs.DELETE("", perm("system:log:operation:delete"), deps.Monitor.OpLogHdl.DeleteBefore)
		}

		// 登录日志
		loginLogs := auth.Group("/logs/login")
		loginLogs.Use(middleware.SetModuleName("登录日志"))
		{
			loginLogs.GET("", perm("system:log:login:list"), deps.Monitor.LoginLogHdl.FindPage)
		}

		// 字典管理
		dictTypes := auth.Group("/dict/types")
		dictTypes.Use(middleware.SetModuleName("字典管理"))
		{
			dictTypes.GET("", perm("system:dict:type:list"), deps.Dict.DictTypeHdl.List)
			dictTypes.GET("/:id", perm("system:dict:type:query"), deps.Dict.DictTypeHdl.GetByID)
			dictTypes.POST("", perm("system:dict:type:add"), deps.Dict.DictTypeHdl.Create)
			dictTypes.PUT("/:id", perm("system:dict:type:edit"), deps.Dict.DictTypeHdl.Update)
			dictTypes.DELETE("/:id", perm("system:dict:type:delete"), deps.Dict.DictTypeHdl.Delete)
			dictTypes.GET("/:id/data", perm("system:dict:data:list"), deps.Dict.DictDataHdl.ListByType)
			dictTypes.GET("/:id/data/:dataId", perm("system:dict:data:query"), deps.Dict.DictDataHdl.GetByID)
			dictTypes.POST("/:id/data", perm("system:dict:data:add"), deps.Dict.DictDataHdl.Create)
			dictTypes.PUT("/:id/data/:dataId", perm("system:dict:data:edit"), deps.Dict.DictDataHdl.Update)
			dictTypes.DELETE("/:id/data/:dataId", perm("system:dict:data:delete"), deps.Dict.DictDataHdl.Delete)
		}

		// 字典消费查询（仅需登录，无需权限码）
		dictCodes := auth.Group("/dict/codes")
		dictCodes.Use(middleware.SetModuleName("字典查询"))
		{
			dictCodes.GET("/:code", deps.Dict.DictCodeHdl.GetByCode)
			dictCodes.GET("", deps.Dict.DictCodeHdl.GetByCodes)
		}

		// 通知公告
		notices := auth.Group("/notices")
		notices.Use(middleware.SetModuleName("通知公告"))
		{
			notices.GET("", perm("system:notice:list"), deps.Monitor.NoticeHdl.List)
			notices.GET("/target-users", perm("system:notice:list"), deps.Monitor.NoticeHdl.TargetUsers)
			// 用户公告收件箱·白名单(仅登录,服务端按接收范围过滤,对齐若依 listTop):
			// 与 /notices/:id/read 白名单配套,普通用户也能拉取可见公告与未读数。
			notices.GET("/my", deps.Monitor.NoticeHdl.MyNotices)
			// 批量已读·白名单(与 /my、/:id/read 配套):幂等可重复调用。
			notices.POST("/read-all", deps.Monitor.NoticeHdl.MarkAllRead)
			notices.GET("/:id", perm("system:notice:query"), deps.Monitor.NoticeHdl.GetByID)
			notices.POST("", perm("system:notice:add"), deps.Monitor.NoticeHdl.Create)
			notices.PUT("/:id", perm("system:notice:edit"), deps.Monitor.NoticeHdl.Update)
			notices.DELETE("/:id", perm("system:notice:delete"), deps.Monitor.NoticeHdl.Delete)
			notices.POST("/:id/publish", perm("system:notice:publish"), deps.Monitor.NoticeHdl.Publish)
			notices.POST("/:id/revoke", perm("system:notice:publish"), deps.Monitor.NoticeHdl.Revoke)
			notices.POST("/:id/read", deps.Monitor.NoticeHdl.MarkRead)
			notices.GET("/:id/read-users", perm("system:notice:list"), deps.Monitor.NoticeHdl.ReadUsers)
		}

		// 服务监控
		monitor := auth.Group("/monitor")
		{
			// 服务器监控
			server := monitor.Group("/server")
			server.Use(middleware.SetModuleName("服务器监控"))
			{
				server.GET("/stats", perm("system:server:list"), deps.Monitor.ServerMonitorHdl.ServeHTTP)
				server.GET("/history", perm("system:server:list"), deps.Monitor.ServerMonitorHdl.GetHistory)
			}

			// 缓存管理
			cache := monitor.Group("/cache")
			cache.Use(middleware.SetModuleName("缓存管理"))
			{
				cache.GET("/keys", perm("system:cache:list"), deps.Monitor.CacheHdl.ListKeys)
				cache.GET("/keys/value", perm("system:cache:query"), deps.Monitor.CacheHdl.GetKeyValue)
				cache.DELETE("/keys", perm("system:cache:delete"), deps.Monitor.CacheHdl.DeleteKeys)
				cache.GET("/stats", perm("system:cache:list"), deps.Monitor.CacheHdl.GetStats)
			}

			// SQL监控
			sql := monitor.Group("/sql")
			sql.Use(middleware.SetModuleName("SQL监控"))
			{
				sql.GET("/stats", perm("system:sql:list"), deps.Monitor.SqlMonitorHdl.GetStats)
				sql.GET("/history", perm("system:sql:list"), deps.Monitor.SqlMonitorHdl.GetHistory)
			}

			// pprof 动态启停 + UI 渲染数据接口
			pprofGroup := monitor.Group("/pprof")
			pprofGroup.Use(middleware.SetModuleName("pprof性能分析"))
			{
				// 状态概览始终可访问(仅报告开关/倒计时/profile 清单)
				pprofGroup.GET("/status", perm("system:pprof:list"), deps.Monitor.PprofHdl.HandleStatus)
				pprofGroup.POST("/enable", perm("system:pprof:enable"), deps.Monitor.PprofHdl.HandleEnable)
				pprofGroup.POST("/disable", perm("system:pprof:disable"), deps.Monitor.PprofHdl.HandleDisable)
				// 火焰图数据与原始端点同样受 enabled 开关保护
				pprofGroup.GET("/profile/:name", middleware.PprofGuard(deps.Infra.ConfigProv), perm("system:pprof:list"), deps.Monitor.PprofHdl.HandleProfileFlame)
			}
			pprofDebug := monitor.Group("/debug/pprof")
			pprofDebug.Use(middleware.PprofGuard(deps.Infra.ConfigProv))
			pprofDebug.GET("/*any", perm("system:pprof:list"), handler.AdaptPprof())
		}

		// 参数配置
		configs := auth.Group("/configs")
		configs.Use(middleware.SetModuleName("参数配置"))
		{
			configs.GET("", perm("system:config:list"), deps.Config.ConfigHdl.List)
			configs.GET("/:id", perm("system:config:query"), deps.Config.ConfigHdl.GetByID)
			configs.POST("", perm("system:config:add"), deps.Config.ConfigHdl.Create)
			configs.PUT("/:id", perm("system:config:edit"), deps.Config.ConfigHdl.Update)
			configs.DELETE("/:id", perm("system:config:delete"), deps.Config.ConfigHdl.Delete)
			configs.GET("/key/:key", perm("system:config:query"), deps.Config.ConfigHdl.GetByKey)
		}

		// 定时任务管理
		jobs := auth.Group("/jobs")
		jobs.Use(middleware.SetModuleName("定时任务管理"))
		{
			// 固定路径必须在 :id 之前注册！
			jobs.GET("/targets", perm("system:job:list"), deps.Job.JobHdl.GetTargets)
			jobs.GET("/health", perm("system:job:list"), deps.Job.JobHdl.GetHealth)
			// 日志
			jobs.GET("/logs", perm("system:job:log:list"), deps.Job.JobHdl.FindLogs)
			jobs.DELETE("/logs", perm("system:job:log:delete"), deps.Job.JobHdl.DeleteLogs)
			// CRUD
			jobs.GET("", perm("system:job:list"), deps.Job.JobHdl.List)
			jobs.GET("/:id", perm("system:job:query"), deps.Job.JobHdl.GetByID)
			jobs.POST("", perm("system:job:add"), deps.Job.JobHdl.Create)
			jobs.PUT("/:id", perm("system:job:edit"), deps.Job.JobHdl.Update)
			jobs.DELETE("/:id", perm("system:job:delete"), deps.Job.JobHdl.Delete)
			// Status
			jobs.POST("/:id/pause", perm("system:job:pause"), deps.Job.JobHdl.Pause)
			jobs.POST("/:id/resume", perm("system:job:execute"), deps.Job.JobHdl.Resume)
			// RunOnce
			jobs.POST("/:id/run", perm("system:job:once"), deps.Job.JobHdl.RunOnce)
		}

		// 文件管理
		files := auth.Group("/files")
		files.Use(middleware.SetModuleName("文件管理"))
		{
			files.GET("", perm("system:file:list"), deps.File.FileHdl.List)
			files.GET("/stats", perm("system:file:list"), deps.File.FileHdl.Stats)
			files.POST("", perm("system:file:upload"), deps.File.FileHdl.Upload)
			files.PUT("/:id", perm("system:file:edit"), deps.File.FileHdl.Rename)
			files.DELETE("", perm("system:file:delete"), deps.File.FileHdl.DeleteMany)
			files.GET("/:id/thumbnail", perm("system:file:list"), deps.File.FileHdl.Thumbnail)
			files.GET("/:id/download", perm("system:file:download"), deps.File.FileHdl.Download)
			files.GET("/:id/preview", perm("system:file:download"), deps.File.FileHdl.Preview)
		}
	}

	// WebSocket
	api.GET("/ws", wsPkg.HandleUpgrade(deps.Infra.Hub, deps.Infra.Logger))

	return r
}
