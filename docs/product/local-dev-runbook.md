# 衣货通 MVP 本地运行手册

本文用于本地评审和演示 MVP 闭环。当前仓库已具备 API 契约、后端领域逻辑、PostgreSQL migration、管理后台、uni-app 小程序工程，以及可启动的 Go HTTP 服务入口。

## 环境要求

- Go 1.23+
- Node.js 24+ 或兼容当前依赖的 LTS 版本
- PostgreSQL 14+
- goctl 1.7.5（仅由项目生成脚本调用；其他版本会被拒绝）
- 可选：微信开发者工具，用于导入 `wxapp/dist/mp-weixin`

## 配置模板

- 后端配置模板：`backend/etc/app.yaml.example`
- 环境变量模板：`.env.example`
- Nginx 示例：`deploy/nginx/wplink.conf`
- systemd 示例：`deploy/systemd/wplink-api.service`
- 详细说明：`docs/product/deployment-config.md`

当前已实现七牛 Kodo 上传凭证签发，小程序可通过 `/api/v1/uploads/token` 获取凭证后直传对象存储。图片字段仍保存最终 CDN URL。生产服务启用用户 token 后，供需信息发布和“我的发布”管理会校验用户是否绑定对应商家。

正式运营时必须使用 `RuntimeMode: production` 并提供真实 `ADMIN_TOKEN_SECRET`、`USER_TOKEN_SECRET`、PostgreSQL DSN、PostgreSQL 连接池参数、微信小程序 AppID/Secret、短信验证码服务配置、供需信息生命周期任务间隔和七牛密钥。生产模式会在启动前校验关键配置，缺失时拒绝启动；短信 `dev` provider 仅允许本地开发。

## 数据库初始化

创建数据库后按顺序执行：

```bash
psql "$DATABASE_URL" -f backend/migrations/000001_admin_auth.up.sql
psql "$DATABASE_URL" -f backend/migrations/000002_core_domain.up.sql
psql "$DATABASE_URL" -f backend/migrations/000003_seed_zhili.up.sql
psql "$DATABASE_URL" -f backend/migrations/000004_user_interactions.up.sql
psql "$DATABASE_URL" -f backend/migrations/000005_merchant_logo.up.sql
psql "$DATABASE_URL" -f backend/migrations/000006_merchant_type_change_logs.up.sql
psql "$DATABASE_URL" -f backend/migrations/000007_verification_payments.up.sql
psql "$DATABASE_URL" -f backend/migrations/000008_hot_search_keywords.up.sql
psql "$DATABASE_URL" -f backend/migrations/000009_verification_expiration.up.sql
psql "$DATABASE_URL" -f backend/migrations/000010_sourcing_map.up.sql
psql "$DATABASE_URL" -f backend/scripts/seed_demo_data.sql
```

项目通过 `DATABASE_URL` 或 `backend/etc/app.yaml` 中的 `Postgres.DSN` 连接 PostgreSQL。推荐先用 Go 验证器创建临时数据库完整验证 migration up/down 和演示数据导入，避免在业务库上直接执行 down：

```bash
cd backend
go run ./scripts/verify_migrations.go -config etc/app.yaml
```

若当前执行环境暂时缺少 PostgreSQL 连接，只能先运行静态校验，确认 migration 文件成对、down 覆盖 up 创建的表、种子插入表已由前序 migration 创建：

```bash
node backend/scripts/validate_migrations.mjs
```

演示数据包含：

- 织里城市站和七类供需类型配置
- 认证工厂、认证库存商、服务商、采购商
- 七类已发布供需信息，以及待审核、已驳回、即将过期、已过期供需信息
- 已发布的织里利济路拿货地图示范场景、档口和配套点位
- 需求方向供需信息示例
- 消息、供需信息指标、联系事件、操作日志、置顶券和权益

说明：演示数据统一使用 `resources` 表表达供应和需求方向的供需信息，不再包含独立需求线索或后台对接数据。

本地若要清库重跑，建议直接重建数据库后重新执行上述脚本。当前 migration down 文件可用于开发验证，但演示环境优先使用干净数据库。

## 后端验证

当前后端已有 HTTP 服务入口，`server.NewGoZeroServer` 通过 `handler.RegisterHandlers` 注册 `backend/app/api/app.api` 聚合的全部业务路由。生成路由之外只额外提供 `/healthz`、`/readyz`；`/admin` 与 `/admin/` 子路径由嵌入式管理后台静态 NotFound 回退承接。未声明的 API 返回 404，已声明路径使用错误 HTTP method 返回 405。账号链路首发使用 `/api/v1/auth/wechat-login` 和 `/api/v1/me`；`/api/v1/auth/sms-code`、`/api/v1/me/phone` 为手机号绑定后续版本预留接口。

先从仓库根目录运行固定生成与全量检查：

```bash
make check-api-generated
make check
```

`make check-api-generated` 会调用项目生成脚本，校验 goctl 版本必须为 1.7.5，并显式使用仓库内 `backend/app/goctl/` 模板。不要直接调用开发机任意版本的全局 goctl，也不要依赖用户级模板或 `GOCTL_HOME`；项目只接受上述 Makefile 入口和固定生成结果。

如需单独验证数据库迁移，再执行：

```bash
cd backend
node scripts/validate_migrations.mjs
go run ./scripts/verify_migrations.go -config etc/app.yaml
```

### 新增或修改 API

1. 修改对应领域 `.api`；新增领域文件时同步加入 `backend/app/api/app.api` import。
2. 回到仓库根目录运行 `make generate-api`，生成 routes/types 和缺失的 Handler 骨架。
3. 实现 Handler、Logic 以及所需 Model 或外部依赖装配；生成的 `WPLINK_API_HANDLER_STUB` 只能短暂存在，提交前必须实现并清除。
4. 补齐用户身份、商家管理权限或后台模块权限测试，以及成功、参数错误、未授权、越权和依赖失败等行为测试。
5. 运行 `make check-api-generated`，再运行 `cd backend && node --test scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs` 和相关 Go 测试；提交前执行根目录 `make check`。

### PostgreSQL 强制集成验证

迁移、任务协调或短信限流改动还必须执行独立门禁。以下命令先回到仓库根目录，避免上一节的 `cd backend` 影响根 `Makefile` 的查找。测试库必须可丢弃，并且已经按顺序执行全部 up migrations；示例中的 `wplink_integration` 仅用于本地测试，**禁止**传入生产 DSN：

```bash
cd "$(git rev-parse --show-toplevel)"
WPLINK_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:5432/wplink_integration?sslmode=disable' make check-postgres
```

该命令会运行 Coordinator 与 `SQLSMSSendLimiter` 的 PostgreSQL 集成测试。普通 `make check` 不会连接这个测试库，因此不能替代 `make check-postgres`；未设置 `WPLINK_TEST_POSTGRES_DSN` 时该门禁应明确失败，而不是跳过。

启动本地服务：

```bash
cd backend
go run ./app -f etc/app.yaml
```

服务会先加载配置并创建 `ServiceContext`。生产模式在监听端口前执行 `svc.ValidateAPIServiceContext`，依赖缺失会直接拒绝启动，不会通过减少路由降级；校验通过后才创建 `NewGoZeroServer` 并注册完整生成路由。

健康检查：

- `/healthz`：只验证 HTTP 进程存活，返回 `ok`。
- `/readyz`：验证服务已连接 PostgreSQL；数据库不可用时返回 `503 not ready`。

服务启动后会按 `Tasks.ResourceLifecycleInterval` 自动执行供需信息生命周期任务，用于过期供需信息状态流转和即将过期/已过期消息提醒。本地演示可使用模板默认 `1h`；多实例生产部署时应让兼容协议的所有实例启用任务，并由 PostgreSQL Advisory Lock 协调同类任务互斥。

如果只验证入口和后台静态路由，也可以使用模板配置：

```bash
go run ./app -f etc/app.yaml.example
```

业务 API 依赖 PostgreSQL DSN、演示种子数据和后台 token 密钥。管理后台和小程序本地联调时，应让前端 `VITE_API_BASE_URL` 或小程序请求域名指向同一个 Go 服务。

## 管理后台

```bash
cd admin-web
npm install
npm run build
npm run dev
```

如后端服务不在同域，设置 `VITE_API_BASE_URL`：

```bash
VITE_API_BASE_URL=http://127.0.0.1:4000 npm run dev
```

如需打包进 Go 服务端一体化部署，执行：

```bash
node backend/scripts/prepare_admin_embed.mjs
```

该脚本会按 `/admin/` 子路径构建后台，并把构建产物复制到 `backend/app/internal/adminweb/dist`，供 Go embed 使用。

后台核心页面：

- `/dashboard` 数据概览
- `/resources/pending` 供需信息审核
- `/merchants` 商家管理
- `/resources?direction=demand` 需求方向供需信息筛选
- `/entitlements` 权益发放
- `/banner-topics` Banner 专题
- `/resource-type-configs` 供需类型配置
- `/operation-logs` 操作日志
- `/search-logs` 搜索日志

## 小程序

首次安装：

```bash
cd wxapp
npm install --cache "$PWD/.npm-cache"
rm -rf .npm-cache
```

校验和构建：

```bash
npm run validate:pages
npm run validate:flows
npm run build:mp-weixin
```

构建成功后，用微信开发者工具导入：

```text
wxapp/dist/mp-weixin
```

如需连接本地 API，在构建或开发命令前设置：

```bash
VITE_API_BASE_URL=http://127.0.0.1:4000 npm run build:mp-weixin
```

发布供需信息页面已接入图片上传。正式小程序需同时在微信公众平台配置 request 合法域名和 uploadFile 合法域名，分别指向 API 域名和七牛上传域名。

## 演示账号和标识

演示数据使用固定手机号和 TSID 数字字符串，便于调试：

- 运营：`19900000001`
- 认证工厂管理员：`19900000002`
- 认证库存商管理员：`19900000003`
- 服务商管理员：`19900000004`
- 采购商买家：`19900000005`

演示商家标识：

- 认证工厂：`8020000000000000001`
- 认证库存商：`8020000000000000002`
- 服务商：`8020000000000000003`

## 已知限制

- migration 静态校验不能替代真实 PostgreSQL up/down；数据库可连接时应运行 `go run ./scripts/verify_migrations.go -config etc/app.yaml`，由临时数据库完成 up/down 验证。
- `make check` 不能替代 `make check-postgres`；后者仅可连接已执行全部 up migrations 的可丢弃测试库，禁止使用生产 DSN。
- 当前后端 HTTP 服务入口已可启动，业务 API 已接入账号、城市站、商家、供需信息、需求、发现、权益、消息、指标和后台管理路由。
- API 契约与运行时路由必须保持生成一致；发现 404 时先检查领域 `.api` 是否被 `app.api` import 及 `make check-api-generated`，发现 405 时检查客户端 HTTP method。
- 短信验证码本地可用 `SMS.Provider: dev` 和固定 `DevCode` 验证；相关后端接口已预留。首发小程序不开放手机号绑定入口，正式运营验收不要求短信验证码服务可用。
- 小程序构建会出现 Sass `@import` 和 legacy JS API 的上游弃用警告，不影响当前构建产物。
