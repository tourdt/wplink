# 衣货通生产发布清单

## 1. 构建产物

推荐在本地仓库根目录直接执行自动化部署脚本：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh
```

脚本会先构建发布产物，再通过 SSH/SCP 上传到服务器，完成二进制安装、systemd 服务安装、数据库 migration、服务重启和健康检查。

首次部署时，如果服务器还没有 `/etc/wplink/app.yaml` 或 `/etc/wplink/wplink.env`，脚本会先创建模板文件并停止。填写生产数据库、JWT、微信、短信和七牛配置后，再次执行同一命令。

如只需要本地构建发布包，不自动上传服务器，可执行：

```bash
bash deploy/scripts/build-release.sh
```

脚本会先执行 `node backend/scripts/prepare_admin_embed.mjs`，把管理后台按 `/admin/` 子路径构建并复制进 Go embed 目录，再输出：

- `dist/release/wplink-api`
- `dist/release/app.yaml.example`
- `dist/release/wplink.env.example`
- `dist/release/wplink-api.service`
- `dist/release/wplink.nginx.conf`

上线前必须确认 `/admin/` 返回真实 Vue 后台页面，不是“后台构建产物尚未嵌入”的占位页。

## 2. 服务器配置

服务器建议路径：

- 程序：`/opt/wplink/wplink-api`
- 配置：`/etc/wplink/app.yaml`
- 环境变量：`/etc/wplink/wplink.env`
- systemd：`/etc/systemd/system/wplink-api.service`
- Nginx：`/etc/nginx/conf.d/wplink.conf`

`/etc/wplink/app.yaml` 应基于 `backend/etc/app.production.yaml.example`，并保持：

- `RuntimeMode: production`
- `Wechat.AllowDevCode: false`
- `SMS.Provider: "http"`
- `SMS.DevCode: ""`

生产密钥只放在服务器环境文件或密钥管理系统，不提交到代码仓库。上线前至少轮换 `ADMIN_TOKEN_SECRET`、`USER_TOKEN_SECRET`、数据库密码、微信 AppSecret、短信密钥和七牛密钥。

## 3. 数据库

推荐使用 `deploy/scripts/deploy-server.sh` 自动发布。脚本会从 `backend/migrations/*.up.sql` 生成迁移清单，随发布包上传，并在服务器数据库中维护 `schema_migrations` 表，重复发布时只执行未记录的 migration。

如需人工初始化干净生产库，可按文件名顺序执行全部 `.up.sql`：

```bash
for file in backend/migrations/*.up.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
done
```

若生产库已经手动执行过这些 migration，但还没有 `schema_migrations` 记录，可在确认数据库结构一致后执行：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --mark-migrations-applied
```

演示数据 `backend/scripts/seed_demo_data.sql` 只用于评审或演示环境，生产正式库按运营需要决定是否导入。

## 4. 首发功能口径

首发正式运营不保留独立“提交需求”、“我的需求”和后台需求处理流程；找货、找厂、找服务等需求方向统一通过供需信息发布、供需信息审核和供需信息列表承接。搜索无结果验收空态、热门词、换条件能力，以及跳转发布页时是否仍走统一供需信息接口。

## 5. 启动检查

部署后执行：

```bash
systemctl daemon-reload
systemctl enable --now wplink-api
systemctl status wplink-api
curl -f http://127.0.0.1:4000/healthz
curl -f http://127.0.0.1:4000/readyz
curl -I https://YOUR_DOMAIN/admin/
```

`/readyz` 必须返回 `ok` 后再切入流量。

## 6. 小程序真机验收

构建微信小程序：

```bash
cd wxapp
VITE_API_BASE_URL=https://YOUR_DOMAIN npm run build:mp-weixin
```

用微信开发者工具导入 `wxapp/dist/mp-weixin`，按 `docs/product/wxapp-manual-acceptance.md` 完成手工验收。

微信公众平台必须配置：

- request 合法域名：API HTTPS 域名
- uploadFile 合法域名：七牛上传域名
- downloadFile 合法域名：七牛 CDN 域名

## 7. 回滚

保留上一版：

- `/opt/wplink/releases/<previous>/wplink-api`
- `/etc/wplink/app.yaml`
- `/etc/wplink/wplink.env`

若发布后 `/healthz` 或 `/readyz` 失败，先恢复上一版二进制并重启 systemd；数据库迁移回滚必须先评估数据兼容性，不直接在生产库执行 down。
