# 服务器配置说明

本文记录当前仓库提供的部署配置模板。模板用于测试/生产环境落地，真实域名、密码和密钥必须在服务器上替换，不能提交到代码仓库。

## 文件

- `backend/etc/app.yaml.example`：后端配置模板，包含 HTTP、PostgreSQL、后台 token、自动任务和七牛 Kodo 对象存储配置。
- `backend/etc/app.production.yaml.example`：生产配置模板，默认 `RuntimeMode: production`，关闭微信开发 code 和短信 dev provider。
- `.env.example`：环境变量示例，供 CI、构建脚本或服务器环境文件参考。
- `deploy/wplink.env.example`：服务器 `/etc/wplink/wplink.env` 示例。
- `deploy/nginx/wplink.conf`：Nginx 反向代理示例；后台静态文件由 Go 服务在 `/admin/` 下提供。
- `deploy/systemd/wplink-api.service`：后端 API 进程托管示例。
- `deploy/scripts/build-release.sh`：发布构建脚本，会先嵌入后台构建产物，再输出 Go 二进制和部署模板。
- `docs/product/production-release-checklist.md`：生产发布检查清单。
- `docs/product/wxapp-manual-acceptance.md`：微信小程序真机/开发者工具手工验收清单。

## 一体化后台部署

管理后台采用 Go embed 一体化部署。构建时先把 Vue 后台按 `/admin/` 子路径打包，再复制到 Go 的嵌入目录：

```bash
node backend/scripts/prepare_admin_embed.mjs
```

脚本会执行：

1. `VITE_ADMIN_BASE=/admin/ npm run build`
2. 将 `admin-web/dist` 复制到 `backend/app/internal/adminweb/dist`

Go 服务已提供 `adminweb.EmbeddedHandler("/admin/")` 和业务 API router，Vue history 路由刷新会回退到 `index.html`，缺失的静态资源仍返回 404。当前 `backend/app/api/app.api` 中的 MVP API 已接入；后续新增但未接线的 `/api/` 路由会返回 `API_NOT_CONNECTED`。

后台 API 客户端默认使用同源 `/api/...`，一体化部署时不需要设置 `VITE_API_BASE_URL`。本地分离开发时可以设置 `VITE_API_BASE_URL=http://127.0.0.1:4000`。

生产发布推荐直接执行：

```bash
bash deploy/scripts/build-release.sh
```

该脚本会自动完成后台嵌入构建和后端二进制构建，避免上线后 `/admin/` 仍显示占位页面。

## 生产必填配置

正式运营建议设置 `RuntimeMode: production`。该模式会在服务启动前校验以下关键配置，缺失或开启开发登录 fallback 时直接失败：

- `Postgres.DSN`、`Postgres.MaxOpenConns`、`Postgres.MaxIdleConns`、`Postgres.ConnMaxLifetime`、`Postgres.ConnMaxIdleTime`
- `AdminAuth.TokenSecret`
- `Wechat.AppID`、`Wechat.AppSecret`
- `SMS.Provider`，以及对应供应商所需字段；`http` 模式需要 `SMS.SendURL`、`SMS.VerifyURL`、`SMS.AccessKeySecret`
- `Tasks.ResourceLifecycleInterval`
- `Storage.Provider`、`Storage.Endpoint`、`Storage.Bucket`、`Storage.AccessKeyID`、`Storage.AccessKeySecret`、`Storage.PublicBaseURL`

## 微信与短信

微信登录已通过 `jscode2session` 获取 openid；本地开发可设置 `Wechat.AllowDevCode: true` 使用 `local-dev-*` code，生产模式禁止开启该选项。

短信验证码支持两种落地方式：

- 本地开发：`SMS.Provider: dev`，使用 `SMS.DevCode` 校验，生产模式会拒绝该 provider。
- 正式运营：推荐先接入 `SMS.Provider: http`，由现有验证码服务提供发送和校验接口。后端会向 `SMS.SendURL` 发送 `{ "phone": "..." }`，向 `SMS.VerifyURL` 发送 `{ "phone": "...", "code": "..." }`，并在配置 `SMS.AccessKeySecret` 时附带 `Authorization: Bearer <secret>`。接口返回 2xx 且 JSON 中 `ok: true` 或 `valid: true` 即视为成功。

短信发送带有本进程限频保护：`SMS.SendMinInterval` 默认 60 秒，`SMS.DailySendLimit` 默认每天 10 次，同一手机号超过限制会返回 `RATE_LIMITED`。多实例部署时仍建议在短信服务、API 网关或 Redis 限流层增加统一限制，避免跨实例绕过。

如直接接入阿里云、腾讯云等厂商 SDK，可保留 `Provider`、`AccessKeyID`、`AccessKeySecret`、`SignName`、`TemplateCode` 配置，并在 `ConfiguredSMSVerifier` 中实现对应 provider 分支。

## 权限边界

后台 `/api/v1/admin/*` 接口在配置 admin token 服务时会校验 `Authorization: Bearer <token>`，只有 `platform_operator` 和 `super_admin` 可访问。小程序侧资源发布、草稿、我的发布列表、刷新、成交反馈、下架、再发类似、权益查看和置顶券核销等商家操作，在生产服务启用用户 token 后，会校验当前用户与目标商家的 active 管理绑定关系；未绑定商家会返回 `FORBIDDEN`。

用户私有数据接口在生产启用用户 token 后以 token 身份为准，不信任前端传入的 `userId`。当前覆盖采购需求提交、“我的采购需求”、认证提交、用户消息列表和消息已读；商家角色消息 `merchant:<merchantId>` 还会校验当前用户是否能管理该商家，点击后可按商家角色标记已读。

资源发布和草稿保存接口在生产启用用户 token 或后台 token 后，会把 `resources.created_by` 绑定为后端解析出的用户或后台操作员；前端不能提交或覆盖资源创建人身份。

资源提交审核 `POST /api/v1/resources/{resourceId}/submit` 在生产启用用户 token 后，会按资源真实所属商家校验管理权限，不接受请求体中的 `merchantId` 作为权限依据。

资源刷新、成交反馈、下架和再发类似等带 `resourceId` 的商家操作，在生产启用用户 token 后同样按资源真实所属商家校验权限，不接受请求体或 query 中的 `merchantId` 作为权限依据。

置顶券核销 `POST /api/v1/top-vouchers/{voucherId}/redeem` 在生产启用用户 token 后，会按置顶券真实所属商家校验管理权限，不接受请求体中的 `merchantId` 作为权限依据；兑换 SQL 仍会校验资源与置顶券属于同一商家且资源已发布。

资源联系行为 `POST /api/v1/resources/{resourceId}/contact-events` 中，`phone` 和 `wechat` 属于完整联系方式解锁动作，生产环境必须携带用户 token，成功解锁后才计入电话点击或微信复制指标；后端解析出的 token 用户为归因身份，不接受前端 body 中的 `userId`。`merchant_home`、`merchant_profile` 和 `share` 可继续作为非联系方式解锁事件记录。

资源搜索日志 `GET /api/v1/resource-search` 允许匿名记录关键词和筛选条件；请求携带用户 token 时以后端解析出的 token 用户为准，不接受 query 中的 `userId` 作为搜索归因身份。

资源指标 `GET /api/v1/resources/{resourceId}/metrics` 和商家指标汇总 `GET /api/v1/merchants/{merchantId}/metrics/summary` 属于商家经营数据；生产启用用户 token 后，会校验当前用户是否能管理对应商家，或使用具备后台访问角色的 admin token 访问。

上传凭证接口 `POST /api/v1/uploads/token` 在只配置上传服务、未配置用户或后台 token 服务时保留本地开发兼容；生产接入用户 token 或后台 token 服务后，必须携带合法的用户 token 或具备后台访问角色的 admin token 才会签发对象存储上传凭证。

## PostgreSQL 连接池

`Postgres` 配置支持连接池参数，模板默认值适合单实例 MVP 起步：

- `MaxOpenConns: 30`：应用进程最多同时打开 30 个数据库连接。
- `MaxIdleConns: 10`：保留最多 10 个空闲连接，减少频繁建连。
- `ConnMaxLifetime: 30m`：连接最长使用 30 分钟后回收，降低长期连接被网络设备或数据库端断开的风险。
- `ConnMaxIdleTime: 5m`：空闲 5 分钟后回收，控制低峰期连接占用。

若数据库实例规格较小或后端多实例部署，需要按 `后端实例数 * MaxOpenConns` 评估 PostgreSQL `max_connections`，避免上线后连接数耗尽。

## 自动任务

`Tasks.ResourceLifecycleInterval` 控制资源生命周期任务执行间隔，模板默认 `1h`。服务启动后会先执行一次，再按间隔持续扫描：

- 已到期资源会自动标记为 `expired`，并向商家发送过期提醒。
- 即将过期资源会向商家发送提醒消息。

生产模式要求该配置大于 0。多实例部署时每个实例都会执行该任务，正式运营建议只让一个后端实例启用自动任务，或后续迁移到独立 worker/分布式锁，避免重复提醒。

## 七牛 Kodo 状态

当前代码已实现 `POST /api/v1/uploads/token` 上传凭证签发，前端可用返回的上传域名、对象 key 和凭证直传七牛 Kodo。业务表中的图片字段仍保存 URL：

- `merchants.images`
- `resources.images`
- `banner_topics.cover_url`

前端上传完成后，再把 `QINIU_PUBLIC_BASE_URL + key` 写入业务接口。

七牛配置项含义：

- `Provider`：固定为 `qiniu-kodo`。
- `Endpoint`：七牛上传域名，例如华南区域 `https://upload-z2.qiniup.com`。
- `Bucket`：七牛空间名称。
- `Region`：七牛区域编号，例如 `z0`、`z1`、`z2`、`na0`、`as0`。
- `AccessKeyID` / `AccessKeySecret`：七牛 AccessKey 和 SecretKey。
- `PublicBaseURL`：绑定空间的 CDN 域名，业务接口保存公开访问 URL 时使用。

## 服务器建议

- API 服务只监听 `127.0.0.1:4000`，公网通过 Nginx 代理；后台通过同一个 Go 服务的 `/admin/` 访问。
- 监控和发布检查使用 `/healthz` 判断进程存活，使用 `/readyz` 判断 PostgreSQL 就绪；`/readyz` 返回 503 时不应切入流量。
- 生产只开放 `80/443/22`，不要开放 PostgreSQL 和 API 内部端口。
- PostgreSQL 使用独立业务用户，不使用超级用户连接应用。
- `JWT_SECRET`、`QINIU_SECRET_KEY`、数据库密码只放服务器环境或密钥管理系统。
- 正式小程序必须使用 HTTPS API 域名，并在微信公众平台配置 request 合法域名。
