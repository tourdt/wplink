# 衣货通生产发布清单

## 1. 构建产物

在仓库根目录执行：

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

生产密钥只放在服务器环境文件或密钥管理系统，不提交到代码仓库。上线前至少轮换 `JWT_SECRET`、数据库密码、微信 AppSecret、短信密钥和七牛密钥。

## 3. 数据库

在干净生产库上按顺序执行：

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
```

演示数据 `backend/scripts/seed_demo_data.sql` 只用于评审或演示环境，生产正式库按运营需要决定是否导入。

## 4. 首发功能口径

首发正式运营暂不开放“提交需求”、“我的需求”和后台“采购需求”处理流程；搜索无结果只验收空态、热门词和换条件能力。相关表、接口和页面可作为后续版本预留，但不作为本次上线验收项。

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

用微信开发者工具导入 `wxapp/dist/build/mp-weixin`，按 `docs/product/wxapp-manual-acceptance.md` 完成手工验收。

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
