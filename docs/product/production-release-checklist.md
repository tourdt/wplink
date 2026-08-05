# 衣货通生产发布清单

## 1. 构建产物

推荐在本地仓库根目录直接执行自动化部署脚本：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh
```

脚本会先构建发布产物，再通过 SSH/SCP 上传到服务器，完成二进制安装、systemd 服务安装、数据库 migration、服务重启和健康检查。

上面的普通一键命令只适用于干净新库，或数据库已经存在 `audit_lease_until`、运行版本都兼容固定锁与审核租约协议的后续发布。若脚本检测到已有 `resources` 表但尚无 `audit_lease_until`，会在 migration 前默认拒绝继续。此时属于首次协议升级例外：必须先按[多实例自动任务部署与运维](../deployment.md#首次引入协调协议)停止所有主机上的旧 API/Scheduler，确认没有旧任务仍在执行，再显式使用 `--confirm-no-legacy-schedulers`。该参数只表达运维确认；脚本只能停止当前目标主机的 `wplink-api`，不能替你检查其他服务器。

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
- `SMS.Provider: "http"`
- `SMS.DevCode: ""`

生产密钥只放在服务器环境文件或密钥管理系统，不提交到代码仓库。上线前至少轮换 `ADMIN_TOKEN_SECRET`、`USER_TOKEN_SECRET`、数据库密码、微信 AppSecret、短信密钥和七牛密钥。

## 3. 数据库

推荐使用 `deploy/scripts/deploy-server.sh` 自动发布。脚本会从 `backend/migrations/*.up.sql` 生成迁移清单，随发布包上传，并在服务器数据库中维护 `schema_migrations` 表，重复发布时只执行未记录的 migration。

首次从旧 Scheduler 升级时，不得按普通滚动发布直接执行 migration。先完成[首次引入协调协议](../deployment.md#首次引入协调协议)的全主机停旧任务检查，再执行：

```bash
WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --confirm-no-legacy-schedulers
```

脚本会在 migration batch 生成和执行前，直接通过 `to_regclass` 与 `information_schema.columns` 判断数据库代际；干净新库和已经具备租约字段的兼容库不需要此确认参数。

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

## 8. 发布后第三方调用抽检

发布后按[第三方调用日志与基线](../deployment.md#第三方调用日志与基线)先完成依赖/路径预检，再执行 JSON 查询抽检 `external_call`。每条事件必须有 `event`、`provider`、`operation`、`outcome`、`duration_ms`，收到 HTTP 响应时应有 `status_code`。`provider` 仅允许 `wechat`、`sms`、`tencent_map`；`operation` 与 `outcome` 必须使用架构文档第 10.1 节的稳定枚举，不能用供应商原始文案代替。

抽样确认统一事件不含手机号、OpenID、Token、签名、密钥、Authorization header、请求体、响应体或原始错误全文；同时确认业务 API 仍只返回安全中文错误，不会透出上游错误内容。

部署文档中的 shell 查询只做候选抽检，不会自动限定 24 小时或 5 分钟窗口，也不会计算占比。先在实际日志平台按日志时间字段过滤和聚合，连续观察至少 24 小时，记录每个 `provider + operation` 的调用量与 `outcome` 分布，形成真实基线后再设置阈值。建议起点为：5 分钟内至少 20 次调用且 `timeout`、`transport_error`、`http_error`、`decode_error` 合计超过 5%；同一 operation 5 分钟至少 3 次 `timeout`；支付 `decode_error` 或连续 `http_error` 作为高优先级排查项。以上为待基线确认的建议，当前仓库并未部署指标平台、告警或熔断。
