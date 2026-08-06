# 服务器配置说明

本文记录当前仓库提供的部署配置模板。模板用于测试/生产环境落地，真实域名、密码和密钥必须在服务器上替换，不能提交到代码仓库。

## 文件

- `backend/etc/app.yaml.example`：后端配置模板，包含 HTTP、PostgreSQL、后台 token、自动任务和七牛 Kodo 对象存储配置。
- `backend/etc/app.production.yaml.example`：生产配置模板，默认 `RuntimeMode: production`，使用真实微信链路并关闭短信 dev provider。
- `.env.example`：环境变量示例，供 CI、构建脚本或服务器环境文件参考。
- `deploy/wplink.env.example`：服务器 `/etc/wplink/wplink.env` 示例。
- `deploy/nginx/wplink.conf`：Nginx 反向代理示例；后台静态文件由 Go 服务在 `/admin/` 下提供。
- `deploy/systemd/wplink-api.service`：后端 API 进程托管示例。
- `deploy/scripts/build-release.sh`：发布构建脚本，会先嵌入后台构建产物，再输出 Go 二进制和部署模板。
- `deploy/scripts/deploy-server.sh`：本地执行的自动化部署脚本，通过 SSH/SCP 上传发布包，并在服务器上完成安装、migration、systemd 重启和健康检查。
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

Go 服务由 `NewGoZeroServer` 创建，`handler.RegisterHandlers` 注册 `backend/app/api/app.api` 生成的全部业务路由，并额外提供 `/healthz`、`/readyz`。`adminweb.EmbeddedHandler("/admin/")` 仅通过 NotFound 处理承接 `/admin` 静态资源与 Vue history 回退；其他未知路径（包括未声明 API）返回 404，已声明路径使用错误 HTTP method 返回 405。

后台 API 客户端默认使用同源 `/api/...`，一体化部署时不需要设置 `VITE_API_BASE_URL`。本地分离开发时可以设置 `VITE_API_BASE_URL=http://127.0.0.1:4000`。

生产发布推荐直接执行：

```bash
bash deploy/scripts/build-release.sh
```

该脚本会自动完成后台嵌入构建和后端二进制构建，避免上线后 `/admin/` 仍显示占位页面。

`build-release.sh` 默认构建 Linux x86-64 后端二进制，即 `WPLINK_RELEASE_GOOS=linux`、`WPLINK_RELEASE_GOARCH=amd64`。如果服务器是 ARM64 Linux，可执行：

```bash
WPLINK_RELEASE_GOARCH=arm64 bash deploy/scripts/build-release.sh
```

如需要从本地电脑自动发布到服务器，可执行：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh
```

脚本会先调用 `deploy/scripts/build-release.sh` 生成发布包，再通过 SSH/SCP 上传到服务器，安装 `/opt/wplink/wplink-api`，写入 systemd 服务，执行未记录的 migration，重启 `wplink-api`，并检查 `/healthz` 和 `/readyz`。如果服务器上还没有 `/etc/wplink/app.yaml` 或 `/etc/wplink/wplink.env`，脚本会先按模板创建这两个文件并停止；填写生产数据库、JWT、微信、短信和七牛配置后再次执行即可。

上述普通一键命令只适用于干净新库，或已存在 `audit_lease_until` 且新旧版本都兼容协调协议的后续发布。若已有 `resources` 表但没有该字段，脚本会在 migration 前默认拒绝继续；这是首次引入固定锁和审核租约协议的维护窗口，不能按普通滚动发布处理。必须先停止并核对所有主机上的旧 API/Scheduler，再按[首次引入协调协议](../deployment.md#首次引入协调协议)使用 `--confirm-no-legacy-schedulers`。该参数只表达运维已完成全主机确认，脚本只能停止当前目标机的 `wplink-api`，不会伪装成已检查其他服务器。

如果服务器使用 SSH 私钥登录，可通过项目专属变量指定私钥：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER WPLINK_SSH_KEY=~/.ssh/wplink_prod_ed25519 bash deploy/scripts/deploy-server.sh
```

也可以先在 `~/.ssh/config` 配置 Host 别名，再把 `WPLINK_DEPLOY_TARGET` 设置为该别名。

常用参数：

```bash
# 跳过数据库 migration
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --skip-migrations

# 服务器已经手动执行过当前 migration 时，只补写 schema_migrations 记录
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --mark-migrations-applied

# 同时安装 Nginx 模板并 reload Nginx，证书和域名前必须先确认
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --install-nginx
```

## 生产必填配置

正式运营建议设置 `RuntimeMode: production`。该模式会在服务启动前校验以下关键配置，缺失时直接失败：

- `Postgres.DSN`、`Postgres.MaxOpenConns`、`Postgres.MaxIdleConns`、`Postgres.ConnMaxLifetime`、`Postgres.ConnMaxIdleTime`
- `AdminAuth.TokenSecret`
- `UserAuth.TokenSecret`
- `Wechat.AppID`、`Wechat.AppSecret`
- `SMS.Provider`，以及对应供应商所需字段；`http` 模式需要 `SMS.SendURL`、`SMS.VerifyURL`、`SMS.AccessKeySecret`
- `Tasks.ResourceLifecycleInterval`
- `Storage.Provider`、`Storage.Endpoint`、`Storage.Bucket`、`Storage.AccessKeyID`、`Storage.AccessKeySecret`、`Storage.PublicBaseURL`

## 微信与短信

微信登录通过 `jscode2session` 获取 OpenID，手机号授权也直接调用微信接口。本地、测试和生产环境均须使用测试或正式小程序实际生成的 code，并配置与该小程序匹配的 `Wechat.AppID`、`Wechat.AppSecret`。文字审核会直接调用微信内容安全接口；启用图片审核时，测试和生产环境都须配置微信可访问的 HTTPS 回调地址 `/api/v1/wechat/content-audit/media-callback`。

短信验证码支持两种落地方式：

- 本地开发：`SMS.Provider: dev`，使用 `SMS.DevCode` 校验，生产模式会拒绝该 provider。
- 正式运营：推荐先接入 `SMS.Provider: http`，由现有验证码服务提供发送和校验接口。后端会向 `SMS.SendURL` 发送 `{ "phone": "..." }`，向 `SMS.VerifyURL` 发送 `{ "phone": "...", "code": "..." }`，并在配置 `SMS.AccessKeySecret` 时附带 `Authorization: Bearer <secret>`。接口返回 2xx 且 JSON 中 `ok: true` 或 `valid: true` 即视为成功。

短信发送带有本进程限频保护：`SMS.SendMinInterval` 默认 60 秒，`SMS.DailySendLimit` 默认每天 10 次，同一手机号超过限制会返回 `RATE_LIMITED`。多实例部署时仍建议在短信服务、API 网关或 Redis 限流层增加统一限制，避免跨实例绕过。

如直接接入阿里云、腾讯云等厂商 SDK，可保留 `Provider`、`AccessKeyID`、`AccessKeySecret`、`SignName`、`TemplateCode` 配置，并在 `ConfiguredSMSVerifier` 中实现对应 provider 分支。

## 权限边界

后台 `/api/v1/admin/*` 接口在配置 admin token 服务时会校验 `Authorization: Bearer <token>`，只有 `platform_operator` 和 `super_admin` 可访问。小程序侧供需信息发布、草稿、我的发布列表、刷新、成交反馈、下架、再发类似、权益查看和置顶券核销等商家操作，在生产服务启用用户 token 后，会校验当前用户与目标商家的 active 管理绑定关系；未绑定商家会返回 `FORBIDDEN`。

用户私有数据接口在生产启用用户 token 后以 token 身份为准，不信任前端传入的 `userId`。当前覆盖供需信息发布、草稿和我的发布、用户消息列表和消息已读；商家角色消息 `merchant:<merchantId>` 还会校验当前用户是否能管理该商家，点击后可按商家角色标记已读。

供需信息发布和草稿保存接口在生产启用用户 token 或后台 token 后，会把 `resources.created_by` 绑定为后端解析出的用户或后台操作员；前端不能提交或覆盖供需信息创建人身份。

供需信息提交审核 `POST /api/v1/resources/{resourceId}/submit` 在生产启用用户 token 后，会按供需信息真实所属商家校验管理权限，不接受请求体中的 `merchantId` 作为权限依据。

供需信息刷新、成交反馈、下架和再发类似等带 `resourceId` 的商家操作，在生产启用用户 token 后同样按供需信息真实所属商家校验权限，不接受请求体或 query 中的 `merchantId` 作为权限依据。

置顶券核销 `POST /api/v1/top-vouchers/{voucherId}/redeem` 在生产启用用户 token 后，会按置顶券真实所属商家校验管理权限，不接受请求体中的 `merchantId` 作为权限依据；兑换 SQL 仍会校验供需信息与置顶券属于同一商家且供需信息已发布。

供需信息联系行为 `POST /api/v1/resources/{resourceId}/contact-events` 中，`phone` 和 `wechat` 属于完整联系方式解锁动作，生产环境必须携带用户 token，成功解锁后才计入电话点击或微信复制指标；后端解析出的 token 用户为归因身份，不接受前端 body 中的 `userId`。`merchant_home`、`merchant_profile` 和 `share` 可继续作为非联系方式解锁事件记录。

供需信息搜索日志 `GET /api/v1/resource-search` 允许匿名记录关键词和筛选条件；请求携带用户 token 时以后端解析出的 token 用户为准，不接受 query 中的 `userId` 作为搜索归因身份。

供需信息指标 `GET /api/v1/resources/{resourceId}/metrics` 和商家指标汇总 `GET /api/v1/merchants/{merchantId}/metrics/summary` 属于商家经营数据；生产启用用户 token 后，会校验当前用户是否能管理对应商家，或使用具备后台访问角色的 admin token 访问。

上传凭证接口 `POST /api/v1/uploads/token` 在只配置上传服务、未配置用户或后台 token 服务时保留本地开发兼容；生产接入用户 token 或后台 token 服务后，必须携带合法的用户 token 或具备后台访问角色的 admin token 才会签发对象存储上传凭证。

## PostgreSQL 连接池

`Postgres` 配置支持连接池参数，生产模板当前值为：

- `MaxOpenConns: 30`：应用进程最多同时打开 30 个数据库连接。
- `MaxIdleConns: 10`：保留最多 10 个空闲连接，减少频繁建连。
- `ConnMaxLifetime: 30m`：连接最长使用 30 分钟后回收，降低长期连接被网络设备或数据库端断开的风险。
- `ConnMaxIdleTime: 5m`：空闲 5 分钟后回收，控制低峰期连接占用。

四类自动任务可能同时持有四条专用 `*sql.Conn`，每条连接要在同一 PostgreSQL 后端会话内完成 Advisory Lock 加锁、业务执行和解锁。因此每个 API 实例除 HTTP、事务和健康检查之外，还必须预留四条任务连接；再按 `后端实例数 * MaxOpenConns` 评估 PostgreSQL `max_connections`，避免上线后连接数耗尽。

任务 Coordinator 的 DSN 应直连 PostgreSQL。若必须经过连接池代理，只能使用保证整个客户端连接固定到同一后端会话的 session-pooling；transaction-pooling 会在事务之间切换后端连接，不适用于 session-level Advisory Lock，禁止用于该 Coordinator 连接。完整预算与失锁排障见[多实例自动任务部署与运维](../deployment.md#数据库连接预算)。

## 自动任务

衣货通没有独立 Worker；每个兼容版本的 API 实例都应保持 `Tasks.Enabled: true`，随进程启动同一套 Scheduler。未配置 `Enabled` 时默认启用。`false` 仅用于维护窗口或事故应急停用全部自动任务，不能长期只开一台“主任务实例”，否则会失去自动接管能力。

多实例通过两层机制保护：第一层是 PostgreSQL session-level Advisory Lock，固定锁号为 `1001=resource_lifecycle`、`1002=content_audit_retry`、`1003=payment_reconciliation`、`1004=merchant_map_event_cleanup`；同类任务只有抢锁成功的实例执行，持锁实例退出或会话断开后，其他实例下一周期自动接管。第二层是业务幂等、条件更新以及内容审核的 `audit_processing_by + audit_lease_until` 租约，防止连接异常或旧实例迟到结果破坏业务状态。Advisory Lock 不能替代第二层保护。

生产模板的四个任务超时分别为：

| 任务 | 周期 | 超时 | 固定锁号 |
|---|---:|---:|---:|
| 资源生命周期 | `1h` | `5m` | `1001` |
| 内容审核重试 | `1m` | `10m` | `1002` |
| 微信支付补偿 | `1m` | `5m` | `1003` |
| 商家地图行为清理 | `24h` | `10m` | `1004` |

生产模式要求周期与四个 timeout 均大于零。首次从没有固定锁/审核租约的旧版本升级，必须先停止所有旧 Scheduler 再执行 `000035_resource_audit_retry_lease`；只有协议兼容版本之间才能让所有实例保持 `Tasks.Enabled: true` 滚动发布。架构细节见[多实例自动任务架构](../architecture.md)，发布步骤见[多实例自动任务部署与运维](../deployment.md)。

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
- `ADMIN_TOKEN_SECRET`、`USER_TOKEN_SECRET`、`QINIU_SECRET_KEY`、数据库密码只放服务器环境或密钥管理系统。
- 正式小程序必须使用 HTTPS API 域名，并在微信公众平台配置 request 合法域名。
