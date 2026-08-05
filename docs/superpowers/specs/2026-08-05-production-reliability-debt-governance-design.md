# 生产可靠性技术债第一批治理设计

## 1. 背景

项目已经完成 API 进程内自动任务的多实例协调：四类任务通过 PostgreSQL Advisory Lock 互斥，内容审核重试通过业务租约支持异常接管，部署仍不需要独立 Worker。继续盘点当前代码、CI 和架构文档后，发现生产可靠性仍有三个证据缺口：

1. `.github/workflows/ci.yml` 已提供 PostgreSQL 服务并执行真实 migration up/down 验证，但 `backend/app/internal/task/coordinator_integration_test.go` 读取的是 `WPLINK_TEST_POSTGRES_DSN`。CI 只设置了 `DATABASE_URL`，因此 Coordinator 的真实 PostgreSQL 测试会被跳过。
2. 短信发送限流已经由 `SQLSMSSendLimiter` 和 `sms_send_limits` 实现跨实例共享，不再是进程内限流；但是当前只有内存限流器和短信客户端的单元测试，没有覆盖 PostgreSQL 原子预约、并发竞争和 reservation token 回滚保护。
3. 微信、短信、微信支付和腾讯地图客户端各自有超时与中文错误映射，但日志字段、结果分类和耗时口径不一致，运维人员无法稳定统计第三方成功率、超时率和接口退化趋势。

`docs/product/technical-architecture.md` 仍把短信限流列为进程内技术债，与当前实现不一致，也会误导后续扩容评估。本批治理以补齐生产证据为目标，不同时进行路由迁移、大文件拆分或历史代码清理。

## 2. 目标与成功标准

### 2.1 目标

- 让 CI 必须执行 Coordinator 和 SQL 短信限流的真实 PostgreSQL 集成测试，不能再因环境变量缺失静默跳过。
- 为所有当前直接发起 HTTP 请求的核心第三方客户端建立统一、安全、可检索的调用日志。
- 修正架构文档中的过期技术债，并为发布后观察和告警提供明确口径。
- 保持现有部署形态：自动任务继续随 API 实例运行，不部署独立 Worker，不新增 Redis、MQ 或监控基础设施。

### 2.2 成功标准

- CI 使用一个已执行全部 up migrations 的隔离 PostgreSQL 数据库运行集成测试。
- CI 中 Coordinator 的同锁互斥、断连接管和不同锁并行测试全部实际执行。
- SQL 短信限流的并发预约、每日上限、供应商失败回滚和旧 token 防误回滚测试全部实际执行。
- 微信登录、微信内容审核、微信支付、短信和腾讯地图的每次实际 HTTP 调用都输出统一的 `external_call` 结构化事件。
- 统一事件至少包含 `provider`、`operation`、`outcome`、`duration_ms`，有 HTTP 响应时附带 `status_code`。
- 统一事件不包含手机号、Token、签名、密钥、请求体、响应体或原始错误全文。
- 原有 API 契约、业务状态机、用户中文提示、重试策略和支付幂等逻辑保持不变。
- `make check` 通过，CI 后端和前端任务全部通过。

## 3. 非目标

本批不实施以下内容：

- 独立 Worker、Redis、MQ 或新的常驻服务。
- 自动熔断、自动重试或第三方降级业务策略。
- 新的公开 Prometheus 端点、日志采集平台或告警平台部署。
- `.api`/goctl 路由契约迁移。
- `api.go`、`domain_routes.go`、`SourcingMapView.vue` 等大文件拆分。
- 历史兼容路由、旧字段和未注册页面的删除。
- 与本批验证和日志口径无关的数据库结构变更。

## 4. 方案选择

### 4.1 采用方案：证据闭环优先

先补齐真实数据库验证，再增加不改变业务决策的统一观测，最后更新文档和运维入口。该方案可以直接关闭已知生产验证盲区，改动边界清晰，出现问题时只需回滚二进制和 CI 配置。

### 4.2 未采用方案：可观测平台优先

直接引入 Prometheus 指标、告警规则和熔断器可以提供更完整的平台能力，但需要新增部署、抓取、阈值调参和故障演练。当前还没有真实调用基线，过早写死熔断和告警阈值容易放大故障。

### 4.3 未采用方案：结构重构优先

先迁移 goctl 路由、拆分大文件和清理兼容代码能够提升长期维护性，但不能优先解决 Coordinator 测试被跳过和第三方调用不可统计的问题，且变更面更大。

## 5. 总体架构

第一批治理分为三个可独立验收的单元：

```text
生产可靠性技术债第一批
├── PostgreSQL 并发证据
│   ├── CI 隔离测试库与完整 up migrations
│   ├── Coordinator 多实例集成测试
│   └── SQL 短信限流集成测试
├── 第三方调用观测
│   ├── common/externalcall 统一事件模型
│   ├── 微信 / 短信 / 支付 / 地图客户端接入
│   └── 安全字段与结果分类测试
└── 文档与运维闭环
    ├── 当前技术债纠偏
    ├── 本地与 CI 验证命令
    └── 日志查询和告警建议
```

三个单元不引入新的业务依赖。数据库测试只运行在明确提供的可丢弃测试库；第三方观测只旁路记录，不参与业务成功或失败判断；文档以实际代码和测试为准。

## 6. PostgreSQL 集成验证设计

### 6.1 CI 数据库准备

CI 继续使用现有 `postgres:17-alpine` service。后端任务按以下顺序执行：

1. 使用 `DATABASE_URL` 运行现有 `go run ./scripts/verify_migrations.go`，验证完整 migration up/down 和演示数据导入。
2. 在 CI service 中创建固定的隔离业务测试库 `wplink_integration`，并按文件名顺序执行全部 `backend/migrations/*.up.sql`。
3. 将该隔离库 DSN 设置为 `WPLINK_TEST_POSTGRES_DSN`。
4. 显式运行 PostgreSQL 集成测试目标，随后执行完整 `go test ./...` 和 `go vet ./...`。

集成测试目标必须在 `WPLINK_TEST_POSTGRES_DSN` 为空时直接失败。普通 `go test ./...` 仍允许开发者在未配置 PostgreSQL 时跳过这些用例；CI 通过单独的强制入口保证不会静默退化。

仓库根 `Makefile` 增加独立目标 `check-postgres`。该目标检查 `WPLINK_TEST_POSTGRES_DSN` 非空后执行 Coordinator 与 SQL 短信限流集成测试，供 CI 和本地可丢弃数据库复用；默认 `make check` 不隐式连接外部数据库。CI 创建并迁移 `wplink_integration` 后调用 `make check-postgres`，因此变量拼写错误、数据库未迁移或目标测试失败都会直接让 job 失败。

### 6.2 Coordinator 验证范围

保留并强制执行以下现有测试：

- 两个数据库连接竞争同一固定锁号时，只有一个 Runner 获得执行权。
- 持锁 PostgreSQL backend 被终止后，另一实例能够重新获取锁并接管。
- 不同固定锁号可以并行获取，不会把四类任务串行化。

测试账号不具备 `pg_terminate_backend` 权限时，本地普通测试可以说明原因后跳过断连接管用例；CI 使用 PostgreSQL service 的测试管理员账号，该用例不得跳过。

### 6.3 SQL 短信限流验证范围

新增真实 PostgreSQL 集成测试，直接复用生产 `sms_send_limits` 表和 `SQLSMSSendLimiter`：

- 多个 goroutine 使用不同 `*sql.DB`/连接同时预约同一手机号和日期，最小发送间隔内只能有一个成功。
- 时间跨过最小间隔后可以继续预约，达到 `DailySendLimit` 后稳定返回 `ErrSMSDailyLimit`。
- 当前 reservation token 回滚后只减少本次预约，不影响其他手机号和日期。
- 旧 reservation token 在后续预约成功后执行回滚，不得清除或减少新预约。
- 两个并发回滚请求不会把 `send_count` 减为负数。
- 使用真实 `SQLSMSSendLimiter` 和返回失败的测试 HTTP Transport 调用 `ConfiguredSMSVerifier.SendSMSCode`，确认供应商请求失败后执行 reservation 回滚，下一次发送仍可预约。

测试使用专属、可识别的测试手机号键，并在清理阶段只删除该键对应的数据，不清空表、不影响其他测试。测试不得复制建表 SQL；表结构必须来自完整 migrations，以便发现代码与 schema 不匹配。

## 7. 第三方调用统一观测设计

### 7.1 组件边界

在 `backend/common/externalcall` 增加最小公共组件；`observer.go` 负责事件模型与接口，`log_observer.go` 负责生产结构化日志。组件职责仅包含：

- 定义稳定的 provider、operation 和 outcome 常量。
- 记录调用开始时间并计算耗时。
- 通过可替换的 `Observer` 接口提交结构化事件。
- 提供生产日志 Observer 和测试捕获 Observer 所需的公共接口。

组件不负责发 HTTP 请求、不解析供应商响应、不重试、不熔断，也不生成用户提示。每个客户端仍然掌握自己的协议和错误映射，并在明确的终态提交一次事件。

建议接口如下：

```go
type Outcome string

const (
	OutcomeSuccess          Outcome = "success"
	OutcomeCanceled         Outcome = "canceled"
	OutcomeTimeout          Outcome = "timeout"
	OutcomeTransportError   Outcome = "transport_error"
	OutcomeHTTPError        Outcome = "http_error"
	OutcomeProviderRejected Outcome = "provider_rejected"
	OutcomeDecodeError      Outcome = "decode_error"
)

type Event struct {
	Provider   string
	Operation  string
	Outcome    Outcome
	Duration   time.Duration
	StatusCode int
}

type Observer interface {
	Observe(ctx context.Context, event Event)
}
```

各客户端构造函数通过向后兼容的可选参数接收 `Observer`；未显式传入时使用生产日志 Observer。`ServiceContext` 创建一个共享生产 Observer 并注入各客户端，测试注入捕获 Observer，直接断言事件，不依赖全局日志输出。

### 7.2 统一日志字段

生产 Observer 输出一条 `event=external_call` 的结构化日志：

| 字段 | 含义 | 是否必填 |
|---|---|---|
| `event` | 固定为 `external_call` | 是 |
| `provider` | `wechat`、`sms` 或 `tencent_map` | 是 |
| `operation` | 稳定操作代码 | 是 |
| `outcome` | 统一结果分类 | 是 |
| `duration_ms` | 从调用开始到响应完成或失败的毫秒数 | 是 |
| `status_code` | HTTP 状态码；未收到响应时省略 | 否 |

日志事件不得包含：手机号、OpenID、SessionKey、access token、支付签名、API key、Authorization header、请求体、响应体和原始错误全文。调用链的业务 ID 继续由现有业务日志记录，公共组件不复制可能包含敏感信息的上下文。

### 7.3 Operation 清单

第一批覆盖所有当前核心外部 HTTP 调用：

| Provider | Operation | 所属客户端 |
|---|---|---|
| `wechat` | `code_to_session` | 微信登录换取 session |
| `wechat` | `phone_number` | 微信手机号授权 |
| `wechat` | `access_token` | 微信 access token |
| `wechat` | `content_text_check` | 文本内容安全检测 |
| `wechat` | `content_media_submit` | 图片内容安全任务提交 |
| `wechat` | `pay_create` | 微信支付预下单 |
| `wechat` | `pay_query` | 微信支付订单查询 |
| `wechat` | `pay_close` | 微信支付关单 |
| `sms` | `code_send` | 短信验证码发送 |
| `sms` | `code_verify` | 短信验证码校验 |
| `tencent_map` | `reverse_geocode` | 腾讯地图逆地理编码 |

七牛上传 Token 在当前实现中由服务端本地签名生成，没有发起第三方 HTTP 请求，因此不产生 `external_call` 事件。

该清单以 `backend/common/externalcall` 的稳定常量为准，未定义也未实现 `content_media_download`；不得为不存在的调用编造 operation。

### 7.4 Outcome 判定

- `success`：HTTP 和供应商协议均成功。内容审核返回“风险内容”属于有效业务判定，外部调用本身仍记为成功。
- `canceled`：`context.Canceled`，通常来自服务退出、上层取消或客户端断开，不计入供应商故障率。
- `timeout`：`context.DeadlineExceeded` 或明确的 HTTP client timeout。
- `transport_error`：DNS、连接建立、TLS、连接重置等未取得有效 HTTP 响应的错误。
- `http_error`：收到非预期的 HTTP 状态码；微信支付的非 2xx 响应明确归入此类，不强制归为 `provider_rejected`。
- `provider_rejected`：HTTP 请求完成且响应可解析，但供应商协议返回错误码或拒绝结果。用户输入失效和供应商配置错误应由 operation 维度进一步判断，不直接等同于供应商宕机。
- `decode_error`：HTTP 响应已取得，但 JSON、签名或必需字段无法解析或验证。

每次实际 HTTP 调用只提交一个终态事件。创建请求前的本地参数校验、配置缺失和短信本地限流拦截不属于外部调用，不记录 `external_call`。

## 8. 错误处理与安全约束

- Observer 不返回错误；观测失败不能改变业务调用结果。
- 客户端现有 Context 和 `http.Client.Timeout` 继续作为超时边界，公共组件不增加第二套超时。
- 公共组件不自动重试。支付下单、短信发送和审核任务均可能产生外部副作用，重试必须继续由各自业务幂等协议控制。
- 公共组件不自动熔断。第一批先获得真实基线，再决定是否需要按 provider/operation 引入熔断。
- 新增日志只能记录枚举和数值字段。现有详细错误日志保持原状，本批不借机重写所有日志；如果接入时发现新增字段可能泄露凭据，必须在该客户端内先脱敏再记录。
- `context.Canceled` 与 `timeout` 分开统计，避免正常优雅停机制造故障告警。

## 9. 测试设计

### 9.1 公共组件测试

- 所有 Outcome 均输出稳定字符串。
- 耗时小于零时归零，避免测试时钟或异常时钟造成非法指标。
- `status_code=0` 时生产日志不输出状态码。
- 日志事件只包含白名单字段。

### 9.2 客户端测试

每个客户端至少覆盖：

- 成功响应产生 `success`。
- Context/HTTP 超时产生 `timeout`。

在 HTTP 2xx 响应体内使用供应商错误码的客户端应覆盖 `provider_rejected`；微信支付等通过非 2xx 表达供应商错误的客户端应覆盖 `http_error`。包含响应解析或验签的客户端还应覆盖 `decode_error`；Transport 返回网络错误时应覆盖 `transport_error`。测试通过注入 Observer 检查 provider、operation、outcome、status code 和事件次数。

### 9.3 安全回归测试

构造包含测试手机号、Token、API key、签名和响应体的失败场景，捕获公共 Observer 输出，断言这些值均不出现在统一事件中。业务日志的既有输出不作为本测试的采集对象。

### 9.4 全量验证

- `make check`：静态 migration/API 契约、全部 Go 测试、`go vet`、管理端测试与构建、小程序测试与构建。
- `make check-postgres`：检查 `WPLINK_TEST_POSTGRES_DSN`，并在已执行完整 up migrations 的可丢弃数据库上运行 Coordinator 与 SQL 短信限流集成测试。
- GitHub Actions：后端和前端 job 均成功，后端日志能看到目标集成测试实际执行而不是跳过。

## 10. 运维与告警建议

统一日志上线后先观察 24 小时基线；当前不新增或部署日志指标/告警平台，只有在后续接入相应平台后才配置阈值。文档提供以下初始建议，阈值不写入业务代码：

- 同一 `provider + operation` 在 5 分钟内至少 20 次调用且 `timeout + transport_error + http_error + decode_error` 占比超过 5%：告警。
- 同一 operation 在 5 分钟内出现至少 3 次 `timeout`：告警。
- 支付 `decode_error` 或连续 `http_error`：高优先级检查证书、签名、时间同步和微信支付配置。
- 内容审核 `provider_rejected` 与审核业务风险判定分开统计；业务判定为风险内容不能计入供应商故障率。
- `canceled` 在发布窗口短时上升属于预期；非发布窗口持续上升时检查上游取消、实例退出和请求超时配置。

建议查询字段固定使用 `event=external_call`，按 `provider`、`operation`、`outcome` 聚合。部署文档同时保留现有任务协调日志查询，两类事件分别反映任务调度与第三方依赖健康，不混为一个告警。

## 11. 发布与回滚

本批不新增生产数据库 migration，也不改变 API 契约、业务状态和配置必填项，可以按当前多实例协议兼容版本滚动发布：

1. 合并前执行 PostgreSQL 强制集成测试与 `make check`。
2. 按实例滚动发布，等待 `/readyz` 成功后再替换下一实例。
3. 发布后确认 `external_call` 事件字段完整且不包含敏感值。
4. 观察 24 小时调用基线；若后续接入日志平台，再按基线配置告警。
5. 若新增观测影响服务，可直接恢复上一版二进制；没有数据结构需要回滚。

CI 数据库只用于自动测试。任何本地 PostgreSQL 集成命令都必须指向可丢弃测试库，不得把生产 DSN 传给测试目标。

## 12. 文档同步范围

- `docs/product/technical-architecture.md`：将短信跨实例限流标记为已完成；保留其他真实技术债；增加第三方统一观测现状。
- `docs/product/local-dev-runbook.md`：增加 PostgreSQL 强制集成测试的可丢弃数据库要求和命令。
- `docs/product/production-release-checklist.md`：增加 `external_call` 日志抽检和 24 小时基线观察。
- `docs/deployment.md`：增加第三方调用事件字段、查询方式和告警建议。

## 13. 后续治理批次

第一批完成后，剩余技术债拆成独立设计与实施周期：

1. goctl 路由契约收敛：按完整领域迁移手工 Router，减少 `.api`、types、Router 和前端调用不一致。
2. 超大文件治理：只在修改相关领域时提取纯逻辑、组件和领域路由，不做一次性全仓拆分。
3. 历史兼容代码清理：通过引用搜索、契约测试、迁移验证和历史数据检查逐项删除。
4. 第三方韧性增强：根据第一批真实数据决定是否引入 Prometheus、熔断、受控重试或供应商降级。

每个批次必须独立设计、独立验证，不能把生产可靠性修复与大范围结构重构放入同一提交链。
