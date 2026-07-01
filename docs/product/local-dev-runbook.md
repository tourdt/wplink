# 衣货通 MVP 本地运行手册

本文用于本地评审和演示 MVP 闭环。当前仓库已具备 API 契约、后端领域逻辑、PostgreSQL migration、管理后台、uni-app 小程序工程，以及可启动的 Go HTTP 服务入口。

## 环境要求

- Go 1.23+
- Node.js 24+ 或兼容当前依赖的 LTS 版本
- PostgreSQL 14+
- 可选：微信开发者工具，用于导入 `wxapp/dist/build/mp-weixin`

## 配置模板

- 后端配置模板：`backend/etc/app.yaml.example`
- 环境变量模板：`.env.example`
- Nginx 示例：`deploy/nginx/wplink.conf`
- systemd 示例：`deploy/systemd/wplink-api.service`
- 详细说明：`docs/product/deployment-config.md`

当前已实现七牛 Kodo 上传凭证签发，小程序可通过 `/api/v1/uploads/token` 获取凭证后直传对象存储。图片字段仍保存最终 CDN URL。生产服务启用用户 token 后，资源发布和“我的发布”管理会校验用户是否绑定对应商家。

正式运营时必须使用 `RuntimeMode: production` 并提供真实 `JWT_SECRET`、PostgreSQL DSN、PostgreSQL 连接池参数、微信小程序 AppID/Secret、短信验证码服务配置、资源生命周期任务间隔和七牛密钥。生产模式会在启动前校验关键配置，缺失时拒绝启动；短信 `dev` provider 仅允许本地开发。

## 数据库初始化

创建数据库后按顺序执行：

```bash
psql "$DATABASE_URL" -f backend/migrations/000001_admin_auth.up.sql
psql "$DATABASE_URL" -f backend/migrations/000002_core_domain.up.sql
psql "$DATABASE_URL" -f backend/migrations/000003_seed_zhili.up.sql
psql "$DATABASE_URL" -f backend/migrations/000004_user_interactions.up.sql
psql "$DATABASE_URL" -f backend/migrations/000005_merchant_logo.up.sql
psql "$DATABASE_URL" -f backend/migrations/000006_merchant_type_change_logs.up.sql
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

- 织里城市站和七类资源类型配置
- 认证工厂、认证库存商、服务商、采购商
- 七类已发布资源，以及待审核、已驳回、即将过期、已过期资源
- 采购需求
- 消息、资源指标、联系事件、操作日志、置顶券和权益

说明：种子数据和后端模型可保留撮合相关预留数据结构，但人工撮合功能首期暂不上线，演示和验收不进入撮合后台。

本地若要清库重跑，建议直接重建数据库后重新执行上述脚本。当前 migration down 文件可用于开发验证，但演示环境优先使用干净数据库。

## 后端验证

当前后端已有 HTTP 服务入口，已挂载 `/healthz`、`/readyz`、`/admin/` 一体化后台静态路由，并接入 `backend/app/api/app.api` 中的账号、城市站、商家、资源、需求、发现、认证、权益、消息、指标和后台管理 API。账号链路首发使用 `/api/v1/auth/wechat-login` 和 `/api/v1/me`；`/api/v1/auth/sms-code`、`/api/v1/me/phone` 为手机号绑定后续版本预留接口。未配置 API handler 的兜底路由仍会返回 `API_NOT_CONNECTED`，用于暴露后续新增接口尚未接线的问题。

先运行领域测试和 API 契约校验：

```bash
cd backend
goctl api validate --api app/api/app.api
node scripts/validate_migrations.mjs
go run ./scripts/verify_migrations.go -config etc/app.yaml
GOCACHE="$PWD/.cache/go-build" go test ./...
rm -rf .cache
```

启动本地服务：

```bash
cd backend
go run ./app -f etc/app.yaml
```

健康检查：

- `/healthz`：只验证 HTTP 进程存活，返回 `ok`。
- `/readyz`：验证服务已连接 PostgreSQL；数据库不可用时返回 `503 not ready`。

服务启动后会按 `Tasks.ResourceLifecycleInterval` 自动执行资源生命周期任务，用于过期资源状态流转和即将过期/已过期消息提醒。本地演示可使用模板默认 `1h`；多实例生产部署时建议只保留一个实例启用该任务。

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
- `/resources/pending` 资源审核
- `/merchants` 商家管理
- `/demands` 采购需求
- `/verifications` 认证审核
- `/entitlements` 权益发放
- `/banner-topics` Banner 专题
- `/resource-type-configs` 资源配置
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
wxapp/dist/build/mp-weixin
```

如需连接本地 API，在构建或开发命令前设置：

```bash
VITE_API_BASE_URL=http://127.0.0.1:4000 npm run build:mp-weixin
```

发布资源和商家认证页面已接入图片上传。正式小程序需同时在微信公众平台配置 request 合法域名和 uploadFile 合法域名，分别指向 API 域名和七牛上传域名。

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
- 当前后端 HTTP 服务入口已可启动，业务 API 已接入账号、城市站、商家、资源、需求、发现、认证、权益、消息、指标和后台管理路由。
- 短信验证码本地可用 `SMS.Provider: dev` 和固定 `DevCode` 验证；相关后端接口已预留。首发小程序不开放手机号绑定入口，正式运营验收不要求短信验证码服务可用。
- 小程序构建会出现 Sass `@import` 和 legacy JS API 的上游弃用警告，不影响当前构建产物。
