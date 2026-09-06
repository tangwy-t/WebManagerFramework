# WebManagerFramework

> 企业级 RBAC 权限管理系统 · Go + Vue 3 全栈方案

A production-grade web administration framework built with Go (Gin) on the backend and Vue 3 + TypeScript on the frontend. It provides a complete RBAC (Role-Based Access Control) system with a plugin-based frontend architecture, suitable as a foundation for enterprise management consoles.

中文 | [English](#english)

---

## 特性

### 后端（`server/`）

- **RBAC 权限体系**：用户 / 角色 / 菜单 / 部门 / 数据权限（DataScope）五位一体，支持菜单、按钮、接口三级鉴权
- **数据权限自动注入**：基于 GORM 插件的 DataScope，随实体定义注册，按角色配置自动过滤数据行（全部 / 本部门 / 本部门及以下 / 仅本人）
- **多端安全认证**：JWT + Redis 会话，bcrypt 密码哈希，登录限流、验证码、账号锁定
- **可观测性**：OpenTelemetry 分布式追踪、Zap 结构化日志、pprof、慢查询统计、服务端/ SQL 监控指标
- **基础设施完备**：Snowflake 雪花 ID、Cron 定时任务、WebSocket 实时推送、结构化版本迁移（v001 ~ v021）
- **优雅停机**：分阶段 drain → cleanup 生命周期管理，编排 HTTP/DB/Redis/调度器/连接的有序关闭

### 前端（`web/`）

- **插件化架构**：`src/modules/<name>/` 一个目录即一个业务插件，增删文件夹即启用 / 停用，开箱即用
- **现代化技术栈**：Vue 3.5 + TypeScript + Vite 7 + Pinia + Element Plus + Tailwind CSS 4
- **丰富的企业级能力**：动态路由与权限守卫、主题 / 布局 / 菜单样式切换、国际化、多 Tab 工作台、锁屏、水印
- **内置业务模块**：仪表盘、用户 / 角色 / 菜单 / 部门 / 字典 / 配置 / 文件 / 定时任务 / 日志 / 通知 / 系统监控

### 工程化

- **一键部署**：Docker Compose 编排前后端，健康检查、依赖启动顺序、数据卷持久化
- **数据库版本迁移**：启动时自动执行版本化迁移，随二进制交付
- **Swagger 文档**：API 自动生成文档，`/api/v1` 前缀统一

---

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.26 |
| Web 框架 | Gin 1.12 |
| ORM | GORM（MySQL / SQLite） |
| 缓存 / 会话 | Redis（go-redis v9） |
| 认证 | JWT（golang-jwt/v5）+ bcrypt |
| 配置 | Viper（环境变量覆盖） |
| 日志 | Zap + lumberjack（日志轮转） |
| 可观测 | OpenTelemetry + pprof + gopsutil |
| 前端框架 | Vue 3.5 + TypeScript |
| 构建工具 | Vite 7 |
| UI 组件库 | Element Plus 2.11 + Tailwind CSS 4 |
| 状态管理 | Pinia + pinia-plugin-persistedstate |
| 图表 | ECharts 6 |

---

## 目录结构

```
WebManagerFramework/
├── server/                    # Go 后端
│   ├── cmd/server/            # 入口（main.go，生命周期编排）
│   ├── configs/               # 配置文件（本地 / docker 环境）
│   ├── docs/                  # Swagger 生成物（docs.go / swagger.json / yaml）
│   └── internal/
│       ├── handler/           # HTTP 处理层
│       ├── middleware/        # 中间件（鉴权 / 限流 / 日志 / 数据范围）
│       ├── service/           # 业务逻辑层
│       ├── repository/        # 数据访问层（GORM）
│       ├── model/             # entity / dto
│       ├── router/            # 路由注册
│       ├── scheduler/         # 定时任务调度
│       ├── task/              # 任务注册表
│       └── pkg/               # 基础设施（jwt / redis / datascope / migration …）
├── web/                       # Vue 3 前端
│   ├── src/
│   │   ├── framework/         # 插件运行时 · 路由注册 · 生命周期
│   │   ├── components/        # 组件库（art-*）
│   │   ├── modules/           # 业务插件（一目录 = 一模块）
│   │   ├── router/            # 动态路由 · 权限守卫
│   │   └── utils/             # http / storage / socket 等
│   └── public/
├── data/uploads/              # 上传文件持久化目录（运行时）
├── docker-compose.yml         # 前后端容器编排
└── .env.example               # 环境变量模板
```

---

## 快速开始

### 环境要求

- **Go** ≥ 1.26
- **Node.js** ≥ 20.19 + **pnpm** ≥ 8.8
- **MySQL** 5.7+ / 8.x、**Redis** 5+（需现存实例）

### 本地开发

```bash
# ── 后端 ──
cd server
cp configs/config.yaml configs/config.local.yaml   # （可选）覆盖本地配置
go mod download
make run          # 启动于 http://localhost:9999

# 常用命令
make build        # 编译（注入版本 / 构建时间 / 提交 Hash）
make test         # 运行测试
make lint         # 静态检查（golangci-lint，缺失时降级 go vet）
make swagger      # 重新生成 API 文档（swag v1.16.6）

# ── 前端 ──
cd web
pnpm install
pnpm dev          # http://localhost:3006
pnpm build        # 类型检查 + 产物构建
pnpm test         # Vitest 单元测试
```

### Docker Compose 一键部署

> MySQL / Redis 复用宿主机现存实例（经 `host.docker.internal` 访问），compose 不启动数据库服务。

```bash
# 1. 准备环境变量（可选，覆盖默认值）
cp .env.example .env

# 2. 构建并启动
docker compose up -d --build

# 3. 查看状态与日志
docker compose ps
docker compose logs -f server
docker compose logs -f web

# 4. 停止
docker compose down
```

| 端口 | 服务 | 说明 |
|------|------|------|
| `8088` | 后端 API | 可由 `SERVER_PORT` 覆盖 |
| `80` | 前端 | nginx 托管 dist + `/api` 反代到后端，可由 `WEB_PORT` 覆盖 |

启动后访问：

- 前端管理后台：http://localhost
- Swagger API 文档：http://localhost:8088/api/v1/health （或 Swagger UI）

---

## 配置

配置采用 **配置文件 + 环境变量覆盖** 双轨制（Viper，键名点号 → 下划线映射，环境变量优先级更高）。

关键配置项（见 `.env.example`）：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `DATABASE_HOST` | `host.docker.internal` | MySQL 地址 |
| `DATABASE_PORT` | `3306` | MySQL 端口 |
| `DATABASE_USER` | `root` | MySQL 用户 |
| `DATABASE_PASSWORD` | `password` | MySQL 密码 |
| `DATABASE_DBNAME` | `web_manager_framework` | 数据库名 |
| `REDIS_ADDR` | `host.docker.internal:6379` | Redis 地址 |
| `SERVER_PORT` | `8088` | 后端映射端口 |
| `WEB_PORT` | `80` | 前端映射端口 |

> ⚠️ 生产环境请通过环境变量注入真实密钥，切勿在配置文件或仓库中硬编码敏感信息。

---

## 权限体系

系统采用经典的 RBAC 模型，并扩展了数据权限（DataScope）：

```
用户 (User)
  └─ 多角色 (Role)
        ├─ 菜单权限 (Menu / Permission)     —— 决定"能看什么页面、按钮"
        └─ 数据权限 (DataScope)             —— 决定"能看哪些数据行"
              ├─ 全部数据
              ├─ 本部门数据
              ├─ 本部门及以下数据
              └─ 仅本人数据
```

前端动态路由由后端下发的菜单 + 权限标识生成，按钮级权限通过指令（`v-auth` / `v-perm` / `v-roles`）控制。

---

## 内置业务模块

| 模块 | 目录 | 说明 |
|------|------|------|
| 仪表盘 | `web/src/modules/dashboard` | 数据总览 |
| 用户管理 | `system-user` | 用户 CRUD、部门树、角色分配 |
| 角色管理 | `system-role` | 角色、菜单/数据权限分配 |
| 菜单管理 | `system-menu` | 菜单 / 路由 / 权限点 |
| 部门管理 | `system-dept` | 组织架构树 |
| 字典管理 | `system-dict` | 数据字典类型 / 数据 |
| 配置管理 | `system-config` | 系统参数（可热更） |
| 文件管理 | `system-file` | 上传 / 下载 / 分片 / 缩略图 |
| 定时任务 | `system-job` | Cron 任务、执行日志 |
| 日志管理 | `system-log` | 登录日志 / 操作日志 |
| 通知公告 | `system-notice` | 公告发布、已读聚合 |
| 系统监控 | `system-monitor` | 服务端 / SQL / Redis 缓存 / pprof |
| 个人中心 | `system-user-center` | 账号设置、登录活动 |

---

## API 文档

后端基于 [swag](https://github.com/swaggo/swag) 自动生成 Swagger 文档（`server/docs/`），运行时在 Swagger UI 中访问。接口统一前缀 `/api/v1`，采用 RESTful 风格，Bearer JWT 鉴权。

---

## 开发规范

- 后端采用 **分层架构**：`handler → service → repository → model`，依赖注入经 `wireup` 装配
- 前端业务以 **插件目录** 组织，`index.ts` 声明模块元信息与路由
- 提交遵循 Conventional Commits 规范
- 代码格式化：Go `gofmt` / `go vet` / `golangci-lint`；前端 ESLint + Prettier

---

## License

本项目仅供学习与参考。UI 视觉风格借鉴自 [art-design-pro](https://github.com/Daymychen/art-design-pro)，特此致谢。

---

## English

**WebManagerFramework** is an enterprise-grade RBAC (Role-Based Access Control) administration system powered by Go (Gin) on the backend and Vue 3 + TypeScript on the frontend.

**Highlights**

- Full RBAC with row-level DataScope auto-injection (GORM plugin)
- JWT + Redis session, bcrypt hashing, login rate-limiting, captcha
- Plugin-based frontend: one directory = one business module
- Versioned DB migrations, cron scheduling, WebSocket, OpenTelemetry tracing
- One-click deployment via Docker Compose

**Quick Start**

```bash
# Backend (requires Go >= 1.26)
cd server && go mod download && make run    # http://localhost:9999

# Frontend (requires Node >= 20.19, pnpm >= 8.8)
cd web && pnpm install && pnpm dev          # http://localhost:3006

# Or deploy the full stack with Docker Compose
cp .env.example .env && docker compose up -d --build
```

See the Chinese sections above for detailed configuration, architecture, and module reference.