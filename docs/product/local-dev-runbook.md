# 服链通 MVP 本地运行手册

本文用于本地评审和演示 MVP 闭环。当前仓库已具备 API 契约、后端领域逻辑、PostgreSQL migration、管理后台、uni-app 小程序工程，以及可启动的 Go HTTP 服务入口。

## 环境要求

- Go 1.23+
- Node.js 24+ 或兼容当前依赖的 LTS 版本
- PostgreSQL 14+，需支持 `pgcrypto`
- 可选：微信开发者工具，用于导入 `wxapp/dist/build/mp-weixin`

## 配置模板

- 后端配置模板：`backend/etc/app.yaml.example`
- 环境变量模板：`.env.example`
- Nginx 示例：`deploy/nginx/wplink.conf`
- systemd 示例：`deploy/systemd/wplink-api.service`
- 详细说明：`docs/product/deployment-config.md`

当前已预留七牛 Kodo 配置字段，但还没有实现上传接口或七牛 SDK 客户端。图片字段暂时保存 URL。

## 数据库初始化

创建数据库后按顺序执行：

```bash
psql "$DATABASE_URL" -f backend/migrations/000001_admin_auth.up.sql
psql "$DATABASE_URL" -f backend/migrations/000002_core_domain.up.sql
psql "$DATABASE_URL" -f backend/migrations/000003_seed_zhili.up.sql
psql "$DATABASE_URL" -f backend/scripts/seed_demo_data.sql
```

演示数据包含：

- 织里城市站和七类资源类型配置
- 认证工厂、认证库存商、服务商、采购商
- 七类已发布资源，以及待审核、已驳回、即将过期、已过期资源
- 采购需求、open 状态撮合单、候选资源和参与商家
- 消息、资源指标、联系事件、操作日志、置顶券和权益

本地若要清库重跑，建议直接重建数据库后重新执行上述脚本。当前 migration down 文件可用于开发验证，但演示环境优先使用干净数据库。

## 后端验证

当前后端已有 HTTP 服务入口，已挂载 `/healthz`、`/admin/` 一体化后台静态路由，以及城市站/资源类型公开 API。其余业务 API handler 仍待接入，未接入的 `/api/` 路由会返回 `API_NOT_CONNECTED`，避免被误认为静态资源 404。

先运行领域测试和 API 契约校验：

```bash
cd backend
goctl api validate --api app/api/app.api
GOCACHE="$PWD/.cache/go-build" go test ./...
rm -rf .cache
```

启动本地服务：

```bash
cd backend
go run ./app -f etc/app.yaml
```

如果只验证入口和后台静态路由，也可以使用模板配置：

```bash
go run ./app -f etc/app.yaml.example
```

当后续生成并接线业务 API handler 后，应使用同一 `DATABASE_URL` 或配置文件中的 PostgreSQL DSN 启动服务，再让管理后台和小程序指向该服务。

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
- `/match-cases` 人工撮合
- `/banner-topics` Banner 专题
- `/resource-type-configs` 资源配置
- `/operation-logs` 操作日志

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

## 演示账号和标识

演示数据使用固定手机号和 UUID，便于调试：

- 运营：`19900000001`
- 认证工厂管理员：`19900000002`
- 认证库存商管理员：`19900000003`
- 服务商管理员：`19900000004`
- 采购商买家：`19900000005`

小程序“我的”页可填写商家 ID：

- 认证工厂：`20000000-0000-0000-0000-000000000001`
- 认证库存商：`20000000-0000-0000-0000-000000000002`
- 服务商：`20000000-0000-0000-0000-000000000003`

## 已知限制

- 本地当前没有可用 PostgreSQL 时，无法验证 migration up/down 和演示 SQL 实际导入。
- 当前后端 HTTP 服务入口已可启动，城市站和资源类型 API 已接线；真实完整联调还需要继续接入资源、商家、审核、登录等 `/api/` 路由。
- 小程序构建会出现 Sass `@import` 和 legacy JS API 的上游弃用警告，不影响当前构建产物。
