# 多实例自动任务部署与运维

衣货通不单独部署 Worker。资源生命周期、内容审核重试、微信支付补偿和商家地图行为清理都随每个 API 实例启动；多实例之间由 PostgreSQL Advisory Lock 自动协调，不需要指定“唯一任务实例”。架构原理见[多实例自动任务架构](architecture.md)，可用生产参数见[配置模板](../backend/etc/app.production.yaml.example)。

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

生产模板当前为 `Postgres.MaxOpenConns: 30`。调整连接池时应按单个 API 实例预留至少四条任务连接，并在此之外为 HTTP 业务请求、事务和健康检查保留足够余量；同时核对所有 API 实例连接池上限总和没有超过 PostgreSQL 的 `max_connections` 预算。不要把“任务超时”当成连接池容量控制手段，超时只用于限制异常任务的最长占用时间。

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

## 滚动发布

滚动发布前确认数据库迁移 `000035_resource_audit_retry_lease` 已执行，并让新旧实例都使用 `Tasks.Enabled: true`。无需挑选唯一任务实例，也无需在发布期间切换“主任务节点”。

建议顺序：

1. 先执行数据库迁移，再启动依赖租约字段的新版本。
2. 逐个替换 API 实例，等待健康检查通过后再处理下一实例。
3. 旧实例退出时取消 Scheduler 并等待本轮返回或超时；其数据库会话关闭后 Advisory Lock 自动释放。
4. 新实例启动后立即尝试一轮任务；若旧实例仍持锁则记录 `task_skipped_lock_held`，下一周期自动接管。
5. 发布后检查四个锁号、各任务终态日志、支付失败计数和过期审核租约数量。

短暂同时运行新旧实例是安全的：任务锁负责同类任务互斥，状态条件和审核租约负责连接断开或进程中断边界。若必须应急暂停，将所有实例统一改为 `Tasks.Enabled: false` 并滚动重启；故障解除后再统一恢复为 `true`。

## 快速核对清单

- 每个 API 实例都启用自动任务，没有单独 Worker 或常驻“主任务实例”。
- 连接池在四条最坏任务连接之外仍有业务请求余量。
- 同一周期一个实例出现成功/失败/超时终态，其他竞争实例可出现 `task_skipped_lock_held`。
- `pg_locks` 只出现预期的 `1001`–`1004`，连接随实例退出而释放。
- 过期审核租约可由下一周期重领，没有被手工改成 `pending`。
