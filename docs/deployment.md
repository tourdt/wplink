# 多实例自动任务部署与运维

衣货通不单独部署 Worker。资源生命周期、内容审核重试、微信支付补偿和商家地图行为清理都随每个 API 实例启动；当所有运行版本都支持相同固定锁号和审核租约协议后，多实例之间由 PostgreSQL Advisory Lock 自动协调，不需要指定“唯一任务实例”。首次从不支持该协议的旧版本升级属于例外，必须按本文“首次引入协调协议”停掉旧 Scheduler 后切换。架构原理见[多实例自动任务架构](architecture.md)，可用生产参数见[配置模板](../backend/etc/app.production.yaml.example)。

## 生产配置

所有 API 实例保持一致并启用自动任务：

```yaml
Tasks:
  Enabled: true
  ResourceLifecycleTimeout: 5m
  ContentAuditRetryTimeout: 10m
  MerchantMapEventCleanupTimeout: 10m
  PaymentReconcileTimeout: 5m
```

上面是本方案新增且必须显式审计的配置。实际生产配置还应保留模板中的周期、批次和保留期字段。生产启动校验要求四个超时均大于零。

`Tasks.Enabled: false` 仅用于维护窗口或事故期间应急暂停全部自动任务。它不是主从实例部署策略：不要长期只在某个“主”实例上启用，否则该实例退出后不会自动接管。正常部署时每个 API 实例都设置为 `true`，未抢到锁的实例只记录正常跳过。

## 数据库连接预算

Coordinator 对每个正在执行的任务独占一条 `*sql.Conn`，并在同一会话上加锁、运行和解锁。四类任务锁不同，可同时运行，最坏占用四条专用数据库连接；锁竞争失败的实例只会短暂申请连接并立即归还。

Coordinator 使用的 DSN 必须直连 PostgreSQL。若架构要求经过 PgBouncer 等连接池代理，只能使用保证客户端连接始终绑定同一后端会话的 session-pooling；transaction-pooling 可能在事务边界切换后端连接，不适用于 session-level Advisory Lock，会破坏“同一会话加锁与解锁”的前提。

生产模板当前为 `Postgres.MaxOpenConns: 30`。调整连接池时应按单个 API 实例预留至少四条任务连接，并在此之外为 HTTP 业务请求、事务和健康检查保留足够余量；同时核对所有 API 实例连接池上限总和没有超过 PostgreSQL 的 `max_connections` 预算。不要把任务超时当成硬性的连接池容量控制：Coordinator 只通过 Context 发出协作式取消，实际连接占用时间取决于 Runner、数据库驱动和第三方依赖是否及时响应 Context；忽略 Context 的调用会一直占用到自身返回或数据库会话断开。

## 结构化日志排障

生产模板将 JSON 日志写到工作目录下的 `logs`，标准部署目录即 `/opt/wplink/logs`。Coordinator 的真实字段为 `event`、`task`、`lock_key`、`instance_id`、`duration_ms`；可先查看所有任务协调事件：

```bash
rg -n '"event":"(task_skipped_lock_held|task_coordination_[^"]+)"' /opt/wplink/logs
```

早期设计使用了较短的运行状态名。最终实现为避免 Scheduler 与 Coordinator 重复记录错误，统一由 Coordinator 输出以下实际事件；排障时不要直接按早期名称过滤：

| 运维语义 / 早期设计名 | 最终代码中的 `event` | 判断方式与处理 |
|---|---|---|
| `task_started` | 无单独事件 | 当前实现不为每轮额外发开始事件。Scheduler 启用日志只能证明调度已启动，不能证明某轮已获得锁；以随后出现的完成、失败或超时终态确认实际执行。若监控必须统计开始次数，需要先修改代码，不能从现有日志可靠推断。 |
| `task_skipped_lock_held` | `task_skipped_lock_held` | 正常竞争结果；同一 `task`、`lock_key` 已由其他实例持有，不告警。 |
| `task_succeeded` | `task_coordination_completed` | 本轮持锁 Runner 成功；用 `duration_ms` 观察耗时。各 Scheduler 另有中文领域计数日志。 |
| `task_failed` | `task_coordination_runner_failed` | Runner 返回错误。申请连接、加锁失败分别是 `task_coordination_connection_error`、`task_coordination_lock_error`。 |
| `task_timeout`（设计稿曾写 `task_timed_out`） | `task_coordination_runner_context_done` | `error` 为 `context deadline exceeded` 时是配置超时；若为 `context canceled`，通常是实例退出或上层取消。 |
| `task_unlock_failed` | `task_coordination_unlock_failed` | 显式解锁失败。重点检查 `unlocked`、`connection_discarded`、`connection_discard_error`；锁状态未知的连接应被丢弃，不能回池复用。 |

按任务和实例查看终态示例：

```bash
rg -n '"task":"content_audit_retry"' /opt/wplink/logs
rg -n '"event":"task_coordination_runner_context_done"' /opt/wplink/logs
rg -n '"event":"task_coordination_unlock_failed"' /opt/wplink/logs
```

内容审核还应关注 `content_audit_retry_manual_review`、`content_audit_retry_lease_lost` 和 `content_audit_retry_failed`。`lease_lost` 表示旧实例结果被安全丢弃，偶发时不等于数据错误；持续大量出现时检查任务耗时是否接近两分钟租约、数据库延迟和审核供应商延迟。

Coordinator 会直接记录返回的 `error`，当前没有通用日志脱敏器。排障和新增错误时都不得把数据库 DSN、Token、微信密钥、支付私钥或完整第三方载荷拼入错误文本；发现此类内容应按凭据泄露流程处理，不能依赖 Coordinator 自动清洗。

## 第三方调用日志与基线

核心第三方 HTTP 调用会单独输出 JSON 事件 `event=external_call`，与任务协调日志无关。稳定字段是 `event`、`provider`、`operation`、`outcome`、`duration_ms`，收到 HTTP 响应时附加 `status_code`；事件不记录手机号、Token、签名、密钥、Authorization header、请求体、响应体或原始错误全文。

先预检依赖与日志目录；这些命令失败时表示查询没有实际执行，不能按“没有事件”处理：

```bash
command -v find >/dev/null 2>&1 || { echo '缺少 find' >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo '缺少 jq，请先安装' >&2; exit 1; }
test -d /opt/wplink/logs || { echo '日志目录不存在: /opt/wplink/logs' >&2; exit 1; }
```

先用 `rg` 快速定位事件。该文本查询只用于发现候选日志；多字段匹配依赖当前字段顺序，不可用于统计或告警：

```bash
command -v rg >/dev/null 2>&1 || { echo '缺少 rg，请先安装' >&2; exit 1; }
rg -n '"event":"external_call"' /opt/wplink/logs
```

字段在 JSON 中的序列不作为查询条件。需要按多个字段筛选时，使用逐行 raw JSON 解析；以下命令排除压缩日志，忽略混合/历史日志中的非 JSON 行，仅输出符合条件的 JSON 对象，兼容字段重排并在这种混合输入下保持成功退出：

```bash
find /opt/wplink/logs -type f ! -name '*.gz' -exec jq -Rrc '
  fromjson?
  | select(type == "object")
  | select(.event == "external_call"
      and (.outcome == "timeout"
        or .outcome == "transport_error"
        or .outcome == "http_error"
        or .outcome == "decode_error"))
' {} +

find /opt/wplink/logs -type f ! -name '*.gz' -exec jq -Rrc '
  fromjson?
  | select(type == "object")
  | select(.event == "external_call"
      and .provider == "wechat"
      and (.operation == "pay_create"
        or .operation == "pay_query"
        or .operation == "pay_close"))
' {} +
```

为快速临时筛查，也可使用下列文本过滤；它们只用于发现候选日志，且依赖字段顺序，统计和告警前仍应以 JSON 字段解析结果为准：

```bash
command -v rg >/dev/null 2>&1 || { echo '缺少 rg，请先安装' >&2; exit 1; }
rg -n '"event":"external_call".*"outcome":"(timeout|transport_error|http_error|decode_error)"' /opt/wplink/logs
rg -n '"event":"external_call".*"provider":"wechat".*"operation":"pay_' /opt/wplink/logs
```

上述 shell 命令只做候选抽检：不会自动限定 24 小时或 5 分钟窗口，也不会计算失败占比。上线后应在实际日志平台按日志时间字段过滤并聚合，先记录连续 24 小时的 `provider + operation + outcome` 基线；当前仓库没有部署指标平台、自动告警、熔断或公共自动重试。基线确认后可在已有或后续接入的日志平台采用以下建议：同一 `provider + operation` 5 分钟至少 20 次调用且 `timeout`、`transport_error`、`http_error`、`decode_error` 合计占比超过 5%；同一 operation 5 分钟至少 3 次 `timeout`；支付 `decode_error` 或连续 `http_error` 按高优先级排查证书、验签、时间同步和支付配置。

`provider_rejected` 表示 HTTP 已完成且供应商协议给出拒绝或错误码，不自动等于供应商宕机；应结合 operation 与业务语义判断。`canceled` 通常来自服务退出、上层取消或客户端断开，不计入供应商故障率。微信支付的非 2xx 响应记录为 `http_error`，不是强制记为 `provider_rejected`。

## 查询 Advisory Lock

使用应用数据库的只读运维连接执行：

```sql
SELECT
  l.pid,
  a.application_name,
  a.client_addr,
  a.state,
  l.granted,
  l.classid::bigint AS lock_key_high,
  l.objid::bigint AS lock_key,
  l.objsubid,
  a.query_start
FROM pg_locks AS l
LEFT JOIN pg_stat_activity AS a ON a.pid = l.pid
WHERE l.locktype = 'advisory'
  AND l.database = (SELECT oid FROM pg_database WHERE datname = current_database())
  AND l.classid::bigint = 0
  AND l.objid::bigint IN (1001, 1002, 1003, 1004)
ORDER BY l.objid, l.pid;
```

当前四个正数 `int64` 锁号都小于 `2^32`，因此 `pg_try_advisory_lock(bigint)` 在 `pg_locks` 中显示为 `classid = 0`、`objid = 1001..1004`、`objsubid = 1`。因为实现使用 try-lock，不会排队等待，查询通常只看到当前已经 `granted` 的持有者；竞争失败应从 `task_skipped_lock_held` 日志观察。

锁号含义：`1001=resource_lifecycle`、`1002=content_audit_retry`、`1003=payment_reconciliation`、`1004=merchant_map_event_cleanup`。

Advisory Lock 不是业务表记录，不能通过删除行解除，也不能在另一个数据库会话调用 `pg_advisory_unlock` 释放持有者的会话锁。异常持锁时先结合 PID、`pg_stat_activity` 和应用日志定位实例，优先让该实例正常退出；连接关闭后 PostgreSQL 自动释放锁。

## 查询过期内容审核租约

以下查询只读，不会改变业务状态：

```sql
SELECT
  id,
  status,
  audit_retry_at,
  audit_lease_until,
  audit_processing_by,
  audit_retry_count,
  audit_last_error,
  updated_at
FROM resources
WHERE deleted_at IS NULL
  AND status = 'audit_retry'
  AND audit_lease_until IS NOT NULL
  AND audit_lease_until <= NOW()
ORDER BY audit_lease_until, updated_at;
```

过期租约仍保持 `status = 'audit_retry'` 是预期状态。下一轮 `content_audit_retry` 会按 `audit_retry_at <= NOW()`、租约为空或已过期的条件，用 `FOR UPDATE SKIP LOCKED` 安全重领。

不要直接把记录手工改成 `pending`。`pending` 表示已创建异步图片审核任务并等待回调；单改状态既会让该资源退出自动重试领取条件，又可能没有对应的 `resource_content_audit_tasks`，造成永久卡住，还会绕过 `audit_processing_by + audit_lease_until` 对旧结果的防覆盖保护。排障时先恢复 Scheduler 和数据库连接；确需人工修复数据，应先停用全部自动任务，并通过经过评审的业务修复脚本保持状态、租约和图片审核任务一致。

## 发布与滚动升级

### 首次引入协调协议

从不支持固定 Advisory Lock、`Tasks.Enabled` 或内容审核租约的旧版本首次升级时，旧 Scheduler 不参与新协议，不能与新 Scheduler 重叠运行。必须安排维护窗口并按旧版本能力选择以下路径：

1. 若旧版本已经支持 `Tasks.Enabled`，先在所有旧实例设置 `Tasks.Enabled: false` 并逐一重启，确认旧 Scheduler 已停止；普通 API 和回调可继续服务。
2. 若旧版本没有 `Tasks.Enabled`，先从负载均衡摘除并停止所有旧 API 进程，确认没有旧 Scheduler 或仍在执行的旧任务。此路径会产生维护窗口，不能为追求无停机而让旧任务与新租约逻辑混跑。
3. 在临时 PostgreSQL 验证 migration `up/down/up`；通过后，在旧 Scheduler 全部停止的前提下执行首次升级发布。自动脚本会直接通过 `to_regclass('public.resources')` 和 `information_schema.columns` 判断协议代际，不依赖 `schema_migrations` 已存在。
4. 在确认所有主机都已消除旧 Scheduler 后，对每个需要发布的目标机显式执行：

   ```bash
   WPLINK_DEPLOY_TARGET=root@YOUR_SERVER bash deploy/scripts/deploy-server.sh --confirm-no-legacy-schedulers
   ```

   该参数只表达运维已经完成全主机确认。脚本无法检查其他服务器，只会在生成和执行 migration batch 前停止当前目标主机上仍 active 的 `wplink-api`；没有参数时对 legacy 数据库默认拒绝继续。干净新库（不存在 `resources`）和已存在 `audit_lease_until` 的兼容库不会触发该闸门。若使用 `--mark-migrations-applied` 但字段实际缺失，脚本同样拒绝继续。
5. 部署支持固定锁号与租约的新版本。先用 `Tasks.Enabled: false` 启动并完成健康检查，再把所有新实例统一切换为 `true` 并滚动重启。
6. 检查 `1001`–`1004` 锁、任务终态日志、支付失败计数和过期审核租约数量，确认只有新协议实例在执行任务。

如果配置系统无法在一次发布中先禁用再启用新实例，可保持旧 API 全停，先完成迁移，再直接以 `Tasks.Enabled: true` 启动新版本；关键约束仍是新 Scheduler 启动前不存在任何旧 Scheduler。

### 协议兼容版本的后续滚动发布

只有新旧双方都支持相同固定锁号 `1001`–`1004`、相同内容审核租约 guard 和 `Tasks.Enabled` 时，才可在所有实例保持 `Tasks.Enabled: true` 的情况下滚动发布：

1. 逐个替换 API 实例，等待健康检查通过后再处理下一实例。
2. 旧实例退出时取消 Scheduler 并等待本轮 Runner 实际返回；如果依赖正确响应 Context，配置截止时间会触发取消。其数据库会话关闭后 Advisory Lock 自动释放。
3. 新实例启动后立即尝试一轮任务；若兼容旧实例仍持锁则记录 `task_skipped_lock_held`，下一周期自动接管。
4. 发布后检查四个锁号、各任务终态日志、支付失败计数和过期审核租约数量。

这种协议兼容版本之间的短暂混跑是安全的：任务锁负责同类任务互斥，状态条件和审核租约负责连接断开或进程中断边界。若必须应急暂停，将所有实例统一改为 `Tasks.Enabled: false` 并滚动重启；故障解除后再统一恢复为 `true`。

## 快速核对清单

- 首次引入协调协议时旧 Scheduler 已全部停止；后续协议兼容版本滚动时每个 API 实例都启用自动任务，没有单独 Worker 或常驻“主任务实例”。
- 连接池在四条最坏任务连接之外仍有业务请求余量。
- 同一周期一个实例出现成功/失败/超时终态，其他竞争实例可出现 `task_skipped_lock_held`。
- `pg_locks` 只出现预期的 `1001`–`1004`，连接随实例退出而释放。
- 过期审核租约可由下一周期重领，没有被手工改成 `pending`。
