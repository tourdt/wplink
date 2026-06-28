# 服务器配置说明

本文记录当前仓库提供的部署配置模板。模板用于测试/生产环境落地，真实域名、密码和密钥必须在服务器上替换，不能提交到代码仓库。

## 文件

- `backend/etc/app.yaml.example`：后端配置模板，包含 HTTP、PostgreSQL、后台 token 和七牛 Kodo 对象存储配置。
- `.env.example`：环境变量示例，供 CI、构建脚本或服务器环境文件参考。
- `deploy/nginx/wplink.conf`：Nginx 反向代理示例；后台静态文件由 Go 服务在 `/admin/` 下提供。
- `deploy/systemd/wplink-api.service`：后端 API 进程托管示例。

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

## 生产必填配置

正式运营建议设置 `RuntimeMode: production`。该模式会在服务启动前校验以下关键配置，缺失或开启开发登录 fallback 时直接失败：

- `Postgres.DSN`
- `AdminAuth.TokenSecret`
- `Wechat.AppID`、`Wechat.AppSecret`
- `SMS.Provider`、`SMS.AccessKeyID`、`SMS.AccessKeySecret`、`SMS.SignName`、`SMS.TemplateCode`
- `Storage.Provider`、`Storage.Endpoint`、`Storage.Bucket`、`Storage.AccessKeyID`、`Storage.AccessKeySecret`、`Storage.PublicBaseURL`

## 微信与短信

微信登录已通过 `jscode2session` 获取 openid；本地开发可设置 `Wechat.AllowDevCode: true` 使用 `local-dev-*` code，生产模式禁止开启该选项。

短信验证码已抽象为 verifier。当前代码提供开发验证码和生产配置校验边界，不内置具体短信厂商 SDK；上线前需要按所选供应商补齐发送/校验实现，或接入已有验证码服务。

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
- 生产只开放 `80/443/22`，不要开放 PostgreSQL 和 API 内部端口。
- PostgreSQL 使用独立业务用户，不使用超级用户连接应用。
- `JWT_SECRET`、`QINIU_SECRET_KEY`、数据库密码只放服务器环境或密钥管理系统。
- 正式小程序必须使用 HTTPS API 域名，并在微信公众平台配置 request 合法域名。
