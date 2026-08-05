# 生产可靠性技术债第一批治理实施计划

> **供智能体执行者使用：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，按任务逐项实施；所有步骤使用复选框跟踪。

**目标：** 在不新增 Worker、Redis、MQ 或监控基础设施的前提下，强制执行真实 PostgreSQL 多实例/限流验证，并为核心第三方 HTTP 调用补齐统一、安全、可检索的结构化观测日志。

**架构：** 使用 CI 内已迁移的隔离 PostgreSQL 数据库执行 Coordinator 与 SQL 短信限流集成测试；在 `backend/common/externalcall` 提供无业务副作用的 Observer/Call 组件，各第三方客户端显式提交一次调用终态。观测层只输出白名单枚举和数值字段，继续沿用现有超时、错误映射、幂等和重试逻辑。

**技术栈：** Go 1.23、go-zero `logx`、PostgreSQL 17、GitHub Actions、Node.js 22、原生 `testing` / `node:test`。

## 全局约束

- 自动任务继续运行在每个 API 实例内，不单独部署 Worker。
- 不新增 Redis、MQ、Prometheus 服务、公开指标端点、自动熔断或自动重试。
- 不改变 `.api` 契约、业务状态机、用户中文提示、支付幂等和内容审核重试协议。
- PostgreSQL 集成测试只能连接明确可丢弃的测试库，禁止使用生产 DSN。
- 统一日志不得包含手机号、OpenID、SessionKey、access token、签名、密钥、Authorization header、请求体、响应体或原始错误全文。
- 所有实现遵循 `backend-friendly-errors-logging`：错误对用户友好、诊断日志含必要上下文但不泄露敏感信息，复杂分支添加中文注释。
- 所有代码改动按 TDD 顺序执行；每个任务结束时运行本任务测试并独立提交。
- 不处理 goctl 路由迁移、大文件拆分或历史兼容代码删除。

---

## 文件职责映射

### 新增文件

- `backend/common/externalcall/observer.go`：provider/operation/outcome 常量、`Event`、`Observer`、单次终态 `Call` 和网络错误分类。
- `backend/common/externalcall/observer_test.go`：单次上报、负耗时归零、取消/超时/网络错误分类测试。
- `backend/common/externalcall/log_observer.go`：基于 go-zero `logx` 的生产结构化日志 Observer。
- `backend/common/externalcall/log_observer_test.go`：日志字段白名单、状态码省略和敏感值不外泄测试。
- `backend/app/internal/logic/auth/sms_send_limiter_integration_test.go`：真实 PostgreSQL 短信限流并发与回滚测试。

### 修改文件

- `.github/workflows/ci.yml`：创建并迁移 `wplink_integration`，设置测试 DSN，强制运行 PostgreSQL 集成门禁。
- `Makefile`：增加显式失败的 `check-postgres` 目标。
- `backend/app/internal/logic/auth/sms_verifier.go` / `sms_verifier_test.go`：短信发送、校验外呼观测。
- `backend/app/internal/logic/location/tencent_map_geocoder.go` / `tencent_map_geocoder_test.go`：腾讯地图逆地理编码外呼观测。
- `backend/app/internal/logic/auth/wechat_session.go` / `wechat_session_test.go`：微信登录、手机号和 access token 外呼观测。
- `backend/app/internal/logic/contentaudit/wechat_auditor.go` / `wechat_auditor_test.go`：微信文本审核、媒体审核和 access token 外呼观测。
- `backend/app/internal/logic/payment/wechat_pay_gateway.go` / `wechat_pay_gateway_test.go`：微信支付下单、查单和关单外呼观测。
- `backend/app/internal/svc/service_context.go` / `service_context_test.go`：创建并注入一个共享生产 Observer。
- `docs/product/technical-architecture.md`：纠正短信限流技术债，记录统一第三方调用事件。
- `docs/product/local-dev-runbook.md`：记录可丢弃 PostgreSQL 的强制验证命令。
- `docs/product/production-release-checklist.md`：增加发布后日志安全抽检和 24 小时基线观察。
- `docs/deployment.md`：增加 `external_call` 查询和告警口径。
- `docs/superpowers/specs/2026-08-05-production-reliability-debt-governance-design.md`：移除不存在的媒体下载外呼并校正支付错误分类测试要求。

---

### Task 1：建立第三方调用统一观测组件

**Files:**
- Create: `backend/common/externalcall/observer.go`
- Create: `backend/common/externalcall/observer_test.go`
- Create: `backend/common/externalcall/log_observer.go`
- Create: `backend/common/externalcall/log_observer_test.go`

**Interfaces:**
- Produces: `externalcall.Observer.Observe(context.Context, externalcall.Event)`。
- Produces: `externalcall.Start(Observer, provider, operation string) *Call`。
- Produces: `(*Call).Finish(context.Context, Outcome, statusCode int)`，同一 Call 只上报一次。
- Produces: `externalcall.ClassifyTransport(context.Context, error) Outcome`。
- Produces: provider 常量 `ProviderWechat`、`ProviderSMS`、`ProviderTencentMap`。
- Produces: operation 常量 `OperationCodeToSession`、`OperationPhoneNumber`、`OperationAccessToken`、`OperationContentTextCheck`、`OperationContentMediaSubmit`、`OperationPayCreate`、`OperationPayQuery`、`OperationPayClose`、`OperationSMSCodeSend`、`OperationSMSCodeVerify`、`OperationReverseGeocode`。

- [ ] **Step 1：先写 Call 与错误分类失败测试**

创建 `observer_test.go`，明确单次终态、负耗时和错误优先级：

```go
package externalcall

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

type recordingObserver struct {
	events []Event
}

func (o *recordingObserver) Observe(_ context.Context, event Event) {
	o.events = append(o.events, event)
}

func TestCallFinishReportsOnceAndClampsNegativeDuration(t *testing.T) {
	base := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	times := []time.Time{base, base.Add(-time.Second)}
	index := 0
	now := func() time.Time {
		value := times[index]
		index++
		return value
	}
	recorder := &recordingObserver{}
	call := startWithClock(recorder, ProviderWechat, OperationCodeToSession, now)

	call.Finish(context.Background(), OutcomeSuccess, 200)
	call.Finish(context.Background(), OutcomeHTTPError, 500)

	if len(recorder.events) != 1 {
		t.Fatalf("events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.Duration != 0 || event.Outcome != OutcomeSuccess || event.StatusCode != 200 {
		t.Fatalf("event = %+v, want clamped success event", event)
	}
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

var _ net.Error = timeoutNetError{}

func TestClassifyTransport(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want Outcome
	}{
		{name: "canceled", ctx: canceledContext(), err: context.Canceled, want: OutcomeCanceled},
		{name: "deadline", ctx: context.Background(), err: context.DeadlineExceeded, want: OutcomeTimeout},
		{name: "net timeout", ctx: context.Background(), err: timeoutNetError{}, want: OutcomeTimeout},
		{name: "network", ctx: context.Background(), err: errors.New("connection reset"), want: OutcomeTransportError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyTransport(tt.ctx, tt.err); got != tt.want {
				t.Fatalf("ClassifyTransport() = %q, want %q", got, tt.want)
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
```

- [ ] **Step 2：运行公共组件测试确认失败**

Run: `cd backend && go test ./common/externalcall -run 'Test(CallFinish|ClassifyTransport)' -count=1`

Expected: FAIL，提示 `Event`、`Start`、Outcome 或 provider/operation 常量未定义。

- [ ] **Step 3：实现事件模型、Call 和网络错误分类**

在 `observer.go` 实现以下最小结构；常量值必须与设计文档完全一致：

```go
package externalcall

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

const EventName = "external_call"

const (
	ProviderWechat     = "wechat"
	ProviderSMS        = "sms"
	ProviderTencentMap = "tencent_map"
)

const (
	OperationCodeToSession       = "code_to_session"
	OperationPhoneNumber         = "phone_number"
	OperationAccessToken         = "access_token"
	OperationContentTextCheck    = "content_text_check"
	OperationContentMediaSubmit  = "content_media_submit"
	OperationPayCreate           = "pay_create"
	OperationPayQuery            = "pay_query"
	OperationPayClose            = "pay_close"
	OperationSMSCodeSend         = "code_send"
	OperationSMSCodeVerify       = "code_verify"
	OperationReverseGeocode      = "reverse_geocode"
)

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

type Call struct {
	observer  Observer
	provider  string
	operation string
	startedAt time.Time
	now       func() time.Time
	once      sync.Once
}

func Start(observer Observer, provider string, operation string) *Call {
	return startWithClock(observer, provider, operation, time.Now)
}

func startWithClock(observer Observer, provider string, operation string, now func() time.Time) *Call {
	if observer == nil {
		observer = NewLogObserver()
	}
	if now == nil {
		now = time.Now
	}
	return &Call{observer: observer, provider: provider, operation: operation, startedAt: now(), now: now}
}

func (c *Call) Finish(ctx context.Context, outcome Outcome, statusCode int) {
	if c == nil {
		return
	}
	c.once.Do(func() {
		duration := c.now().Sub(c.startedAt)
		if duration < 0 {
			duration = 0
		}
		if statusCode < 0 {
			statusCode = 0
		}
		c.observer.Observe(ctx, Event{Provider: c.provider, Operation: c.operation, Outcome: outcome, Duration: duration, StatusCode: statusCode})
	})
}

func ClassifyTransport(ctx context.Context, err error) Outcome {
	if errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)) {
		return OutcomeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
		return OutcomeTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return OutcomeTimeout
	}
	return OutcomeTransportError
}
```

- [ ] **Step 4：先写生产日志白名单失败测试**

创建 `log_observer_test.go`，用 go-zero 自带 Collector 捕获 JSON：

```go
func TestLogObserverWritesWhitelistedFieldsOnly(t *testing.T) {
	collector := logtest.NewCollector(t)
	observer := NewLogObserver()
	observer.Observe(context.Background(), Event{
		Provider: ProviderSMS, Operation: OperationSMSCodeSend,
		Outcome: OutcomeHTTPError, Duration: 1500 * time.Millisecond, StatusCode: 502,
	})

	var payload map[string]any
	if err := json.Unmarshal(collector.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v, log = %s", err, collector.Bytes())
	}
	for key, want := range map[string]any{
		"event": EventName, "provider": ProviderSMS, "operation": OperationSMSCodeSend,
		"outcome": string(OutcomeHTTPError), "duration_ms": float64(1500), "status_code": float64(502),
	} {
		if payload[key] != want {
			t.Fatalf("payload[%q] = %#v, want %#v", key, payload[key], want)
		}
	}
	serialized := string(collector.Bytes())
	for _, forbiddenKey := range []string{"phone", "token", "authorization", "request_body", "response_body", "error_detail"} {
		if _, exists := payload[forbiddenKey]; exists {
			t.Fatalf("log contains forbidden key %q: %s", forbiddenKey, serialized)
		}
	}
	for _, forbidden := range []string{"18800000001", "sms-secret", "Bearer test-token"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("log contains forbidden value %q: %s", forbidden, serialized)
		}
	}
}

func TestLogObserverOmitsEmptyStatusCode(t *testing.T) {
	collector := logtest.NewCollector(t)
	NewLogObserver().Observe(context.Background(), Event{
		Provider: ProviderWechat, Operation: OperationCodeToSession,
		Outcome: OutcomeTimeout, Duration: time.Second,
	})
	if strings.Contains(string(collector.Bytes()), "status_code") {
		t.Fatalf("log should omit status_code: %s", collector.Bytes())
	}
}
```

- [ ] **Step 5：实现生产日志 Observer**

在 `log_observer.go` 中只输出白名单字段；`success`、`canceled`、`provider_rejected` 使用 info，其余故障结果使用 error：

```go
package externalcall

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogObserver struct{}

func NewLogObserver() Observer {
	return LogObserver{}
}

func (LogObserver) Observe(ctx context.Context, event Event) {
	fields := []logx.LogField{
		logx.Field("event", EventName),
		logx.Field("provider", event.Provider),
		logx.Field("operation", event.Operation),
		logx.Field("outcome", string(event.Outcome)),
		logx.Field("duration_ms", event.Duration.Milliseconds()),
	}
	if event.StatusCode > 0 {
		fields = append(fields, logx.Field("status_code", event.StatusCode))
	}
	logger := logx.WithContext(ctx)
	switch event.Outcome {
	case OutcomeSuccess, OutcomeCanceled, OutcomeProviderRejected:
		logger.Infow("第三方调用完成", fields...)
	default:
		logger.Errorw("第三方调用失败", fields...)
	}
}
```

- [ ] **Step 6：运行公共组件测试并提交**

Run: `cd backend && go test ./common/externalcall -count=1`

Expected: PASS。

```bash
git add backend/common/externalcall
git commit -m "feat: 增加第三方调用统一观测组件"
```

---

### Task 2：强制执行 PostgreSQL Coordinator 与短信限流集成验证

**Files:**
- Create: `backend/app/internal/logic/auth/sms_send_limiter_integration_test.go`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: 现有 `SQLSMSSendLimiter.Reserve/Rollback` 和 `ConfiguredSMSVerifier.SendSMSCode`。
- Produces: `make check-postgres`，要求 `WPLINK_TEST_POSTGRES_DSN` 指向已完成 migrations 的可丢弃 PostgreSQL。
- Produces: CI 数据库 `wplink_integration`，仅存在于单次 GitHub Actions backend job。

- [ ] **Step 1：先写 SQL 短信限流真实 PostgreSQL 测试**

创建 `sms_send_limiter_integration_test.go`。使用以下 helper 读取 `WPLINK_TEST_POSTGRES_DSN`、连接数据库并仅清理本测试手机号：

```go
func openSMSLimiterIntegrationDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("WPLINK_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("未设置 WPLINK_TEST_POSTGRES_DSN")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 测试库失败: %v", err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("连接 PostgreSQL 测试库失败: %v", err)
	}
	phone := fmt.Sprintf("it-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, `DELETE FROM sms_send_limits WHERE phone=$1`, phone); err != nil {
			t.Errorf("清理短信限流测试数据失败: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("关闭 PostgreSQL 测试连接失败: %v", err)
		}
	})
	return db, phone
}
```

测试文件单独导入 `_ "github.com/lib/pq"` 注册 PostgreSQL driver。

增加以下三个明确测试：

```go
func TestSQLSMSSendLimiterIntegrationConcurrentReserve(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	limiter := NewSQLSMSSendLimiter(db)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	const workers = 8
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := limiter.Reserve(context.Background(), phone, now, time.Minute, 10)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrSMSSendTooFrequent):
			limited++
		default:
			t.Fatalf("Reserve() error = %v", err)
		}
	}
	if successes != 1 || limited != workers-1 {
		t.Fatalf("successes/limited = %d/%d, want 1/%d", successes, limited, workers-1)
	}
}

func TestSQLSMSSendLimiterIntegrationDailyLimitAndTokenGuard(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	limiter := NewSQLSMSSendLimiter(db)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	oldToken, err := limiter.Reserve(context.Background(), phone, now, time.Minute, 2)
	if err != nil { t.Fatalf("first Reserve() error = %v", err) }
	newToken, err := limiter.Reserve(context.Background(), phone, now.Add(2*time.Minute), time.Minute, 2)
	if err != nil { t.Fatalf("second Reserve() error = %v", err) }
	if err := limiter.Rollback(context.Background(), phone, now, oldToken); err != nil {
		t.Fatalf("old Rollback() error = %v", err)
	}
	var count int
	var storedToken string
	if err := db.QueryRow(`SELECT send_count, reservation_token FROM sms_send_limits WHERE phone=$1`, phone).Scan(&count, &storedToken); err != nil {
		t.Fatalf("query limit row: %v", err)
	}
	if count != 2 || storedToken != newToken {
		t.Fatalf("count/token = %d/%q, want 2/%q", count, storedToken, newToken)
	}
	if _, err := limiter.Reserve(context.Background(), phone, now.Add(4*time.Minute), time.Minute, 2); !errors.Is(err, ErrSMSDailyLimit) {
		t.Fatalf("third Reserve() error = %v, want daily limit", err)
	}
}

func TestSQLSMSSendLimiterIntegrationProviderFailureRollsBack(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	cfg := config.SMSConfig{Provider: "http", SendURL: "https://sms.test/send", SendMinInterval: time.Minute, DailySendLimit: 1}
	failing := NewConfiguredSMSVerifierWithLimiter(cfg, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}, NewSQLSMSSendLimiter(db))
	failing.now = func() time.Time { return now }
	if err := failing.SendSMSCode(context.Background(), phone); err == nil {
		t.Fatal("first SendSMSCode() error = nil, want provider failure")
	}

	success := NewConfiguredSMSVerifierWithLimiter(cfg, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}, NewSQLSMSSendLimiter(db))
	success.now = func() time.Time { return now }
	if err := success.SendSMSCode(context.Background(), phone); err != nil {
		t.Fatalf("second SendSMSCode() error = %v, want rollback to release reservation", err)
	}
}
```

再增加 `TestSQLSMSSendLimiterIntegrationConcurrentRollbackNeverNegative`：先取得一个 token，同时调用两次 `Rollback`，最后查询并断言 `send_count=0`。

- [ ] **Step 2：先验证缺少测试 DSN 时门禁明确失败**

Run: `env -u WPLINK_TEST_POSTGRES_DSN make check-postgres`

Expected: FAIL，明确提示 `WPLINK_TEST_POSTGRES_DSN` 必须指向可丢弃测试库；该行为验证直接执行真实入口，不读取或匹配 Makefile/CI 源码文字。

- [ ] **Step 3：实现 Makefile 与 CI 强制入口**

在 `Makefile` 增加：

```make
.PHONY: check check-backend check-admin check-wxapp check-postgres

check-postgres:
	@test -n "$(WPLINK_TEST_POSTGRES_DSN)" || (echo "WPLINK_TEST_POSTGRES_DSN 必须指向可丢弃测试库"; exit 1)
	cd backend && go test ./app/internal/task -run '^TestCoordinatorIntegration' -count=1 -v
	cd backend && go test ./app/internal/logic/auth -run '^TestSQLSMSSendLimiterIntegration' -count=1 -v
```

在 `.github/workflows/ci.yml` backend job 的 env 增加：

```yaml
WPLINK_TEST_POSTGRES_DSN: postgres://postgres:postgres@127.0.0.1:5432/wplink_integration?sslmode=disable
```

在动态 migration 验证后、Go 全量测试前增加：

```yaml
- name: 准备 PostgreSQL 集成测试库
  working-directory: backend
  run: |
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'CREATE DATABASE wplink_integration'
    for file in migrations/*.up.sql; do
      psql "$WPLINK_TEST_POSTGRES_DSN" -v ON_ERROR_STOP=1 -f "$file"
    done
- name: 运行 PostgreSQL 并发集成测试
  run: make check-postgres
```

- [ ] **Step 4：验证门禁和真实数据库测试**

Run: `env -u WPLINK_TEST_POSTGRES_DSN make check-postgres`

Expected: FAIL，并输出缺少可丢弃测试库 DSN 的明确提示。

Run（仅可丢弃测试库）: `WPLINK_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:5432/wplink_integration?sslmode=disable' make check-postgres`

Expected: Coordinator 三个集成测试和 SQL 短信限流四个集成测试均 PASS，无 SKIP。

- [ ] **Step 5：提交 PostgreSQL 验证闭环**

```bash
git add .github/workflows/ci.yml Makefile backend/app/internal/logic/auth/sms_send_limiter_integration_test.go
git commit -m "test: 强制执行 PostgreSQL 并发验证"
```

---

### Task 3：接入短信与腾讯地图外呼观测

**Files:**
- Modify: `backend/app/internal/logic/auth/sms_verifier.go`
- Modify: `backend/app/internal/logic/auth/sms_verifier_test.go`
- Modify: `backend/app/internal/logic/location/tencent_map_geocoder.go`
- Modify: `backend/app/internal/logic/location/tencent_map_geocoder_test.go`

**Interfaces:**
- Consumes: `externalcall.Observer`、`Start`、`ClassifyTransport`、短信/地图 operation 常量。
- Produces: `NewConfiguredSMSVerifierWithLimiter(cfg config.SMSConfig, client *http.Client, limiter SMSSendLimiter, observers ...externalcall.Observer) *ConfiguredSMSVerifier`。
- Produces: `NewTencentMapGeocoder(cfg config.TencentMapConfig, client *http.Client, observers ...externalcall.Observer) *TencentMapGeocoder`。

- [ ] **Step 1：先写短信观测失败测试**

在 `sms_verifier_test.go` 增加本包捕获 Observer，并用表驱动测试明确五种终态：

```go
type recordingExternalCallObserver struct { events []externalcall.Event }
func (o *recordingExternalCallObserver) Observe(_ context.Context, event externalcall.Event) { o.events = append(o.events, event) }

func testSMSResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestConfiguredSMSVerifierObservesHTTPOutcomes(t *testing.T) {
	tests := []struct {
		name string
		operation func(*ConfiguredSMSVerifier) error
		response *http.Response
		transportErr error
		want externalcall.Outcome
		wantStatus int
	}{
		{name: "send success", operation: func(v *ConfiguredSMSVerifier) error { return v.SendSMSCode(context.Background(), "18800000001") }, response: testSMSResponse(200, `{"ok":true}`), want: externalcall.OutcomeSuccess, wantStatus: 200},
		{name: "verify provider rejected", operation: func(v *ConfiguredSMSVerifier) error { return v.VerifySMSCode(context.Background(), "18800000001", "000000") }, response: testSMSResponse(200, `{"valid":false}`), want: externalcall.OutcomeProviderRejected, wantStatus: 200},
		{name: "http error", operation: func(v *ConfiguredSMSVerifier) error { return v.SendSMSCode(context.Background(), "18800000001") }, response: testSMSResponse(502, `{}`), want: externalcall.OutcomeHTTPError, wantStatus: 502},
		{name: "decode error", operation: func(v *ConfiguredSMSVerifier) error { return v.VerifySMSCode(context.Background(), "18800000001", "123456") }, response: testSMSResponse(200, `{`), want: externalcall.OutcomeDecodeError, wantStatus: 200},
		{name: "timeout", operation: func(v *ConfiguredSMSVerifier) error { return v.SendSMSCode(context.Background(), "18800000001") }, transportErr: context.DeadlineExceeded, want: externalcall.OutcomeTimeout},
	}
	// 每个子测试使用新的 MemorySMSSendLimiter、HTTP client 和 Observer；断言恰好一个事件、provider=sms、operation 与 send/verify 匹配、outcome/status 与表一致。
}
```

测试代码中必须把注释所述断言写成实际 `if`/`t.Fatalf`，不能只保留注释。

- [ ] **Step 2：运行短信测试确认编译失败**

Run: `cd backend && go test ./app/internal/logic/auth -run TestConfiguredSMSVerifierObservesHTTPOutcomes -count=1`

Expected: FAIL，构造函数尚不接受 Observer，或没有观测事件。

- [ ] **Step 3：实现短信外呼观测**

为 `ConfiguredSMSVerifier` 增加 `observer externalcall.Observer`，所有构造函数保持原参数可用，并把最底层构造函数改为 variadic Observer。把 `postSMS` 改为接收 operation：发送传 `OperationSMSCodeSend`，校验传 `OperationSMSCodeVerify`。

`postSMS` 在 `http.NewRequestWithContext` 成功后调用 `externalcall.Start`，并在以下每条终态分支调用一次 `Finish`：

```go
resp, err := v.client.Do(req)
if err != nil {
	call.Finish(ctx, externalcall.ClassifyTransport(ctx, err), 0)
	return errx.New(errx.CodeInternalError, publicError)
}
defer resp.Body.Close()
statusCode := resp.StatusCode
respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
if err != nil {
	call.Finish(ctx, externalcall.OutcomeTransportError, statusCode)
	return errx.New(errx.CodeInternalError, publicError)
}
if statusCode < 200 || statusCode >= 300 {
	call.Finish(ctx, externalcall.OutcomeHTTPError, statusCode)
	return errx.New(errx.CodeInternalError, publicError)
}
```

JSON 解析失败使用 `decode_error`；`data.Error != ""`、`valid=false`、`ok=false` 使用 `provider_rejected`；有效成功或空 2xx body 使用 `success`。本地配置错误、参数校验、dev provider 和 SQL 限流拦截不创建 Call。

- [ ] **Step 4：先写腾讯地图观测失败测试**

在 `tencent_map_geocoder_test.go` 增加五个用例：

- `TestTencentMapGeocoderObservesSuccess`：HTTP 200、`status=0`，断言 `tencent_map/reverse_geocode/success/200`。
- `TestTencentMapGeocoderObservesProviderRejected`：HTTP 200、`status=121`，断言 `provider_rejected/200`，原有 `CodeRateLimited` 不变。
- `TestTencentMapGeocoderObservesHTTPError`：HTTP 429，断言 `http_error/429`，原有 `CodeRateLimited` 不变。
- `TestTencentMapGeocoderObservesDecodeError`：HTTP 200 非法 JSON，断言 `decode_error/200`。
- `TestTencentMapGeocoderObservesTimeout`：Transport 返回 `context.DeadlineExceeded`，断言 `timeout/0`。

每个用例注入独立捕获 Observer，断言恰好一个事件。

- [ ] **Step 5：实现腾讯地图外呼观测**

为 `TencentMapGeocoder` 增加 Observer，并把构造函数改为：

```go
func NewTencentMapGeocoder(cfg config.TencentMapConfig, client *http.Client, observers ...externalcall.Observer) *TencentMapGeocoder
```

只在请求创建成功后 Start；Transport/读取 body 失败分别按 `ClassifyTransport`/`transport_error`，非 2xx 为 `http_error`，JSON 失败为 `decode_error`，`decoded.Status != 0` 为 `provider_rejected`，有效地址为 `success`。`status=0` 但地址为空属于可解析但必需字段缺失，记录 `decode_error`，用户提示保持“未解析到详细地址”。

- [ ] **Step 6：运行短信与地图测试并提交**

Run: `cd backend && go test ./app/internal/logic/auth ./app/internal/logic/location -count=1`

Expected: PASS。

```bash
git add backend/app/internal/logic/auth/sms_verifier.go backend/app/internal/logic/auth/sms_verifier_test.go backend/app/internal/logic/location/tencent_map_geocoder.go backend/app/internal/logic/location/tencent_map_geocoder_test.go
git commit -m "feat: 观测短信和地图第三方调用"
```

---

### Task 4：接入微信登录与手机号外呼观测

**Files:**
- Modify: `backend/app/internal/logic/auth/wechat_session.go`
- Modify: `backend/app/internal/logic/auth/wechat_session_test.go`

**Interfaces:**
- Consumes: `externalcall.Observer` 与微信 auth operation 常量。
- Produces: `NewWechatSessionClient(cfg config.WechatConfig, baseURL string, client *http.Client, observers ...externalcall.Observer) *HTTPWechatSessionClient`。
- Produces: `code_to_session`、`phone_number`、`access_token` 三类事件。

- [ ] **Step 1：先扩展现有成功链路测试**

修改现有 Code2Session 成功测试，注入捕获 Observer，断言一个事件：

```go
if got := observer.events; len(got) != 1 || got[0].Provider != externalcall.ProviderWechat || got[0].Operation != externalcall.OperationCodeToSession || got[0].Outcome != externalcall.OutcomeSuccess || got[0].StatusCode != http.StatusOK {
	t.Fatalf("events = %+v, want one successful code_to_session event", got)
}
```

修改现有 GetPhoneNumber 成功测试，断言按顺序产生两个事件：`access_token/success`、`phone_number/success`。access token 命中客户端缓存时不得产生第二条 access token 事件。

- [ ] **Step 2：增加失败分类测试并确认失败**

新增：

- `TestWechatSessionClientObservesCode2SessionTimeout`：Transport 返回 `context.DeadlineExceeded`，断言 `timeout`。
- `TestWechatSessionClientObservesCode2SessionProviderRejected`：HTTP 200 返回 `{"errcode":40029,"errmsg":"invalid code"}`，断言 `provider_rejected`，公开错误仍为未授权。
- `TestWechatSessionClientObservesCode2SessionDecodeError`：HTTP 200 返回非法 JSON，断言 `decode_error`。
- `TestWechatSessionClientObservesPhoneNumberProviderRejected`：access token 成功、手机号接口返回微信错误码，断言第二个事件为 `phone_number/provider_rejected`。
- `TestWechatSessionClientDoesNotObserveLocalValidation`：空 code 和 `local-dev-` code 均不产生事件。

Run: `cd backend && go test ./app/internal/logic/auth -run 'TestWechatSessionClient|TestHTTPWechatSessionClient' -count=1`

Expected: FAIL，尚未注入或提交 Observer 事件。

- [ ] **Step 3：实现微信登录客户端观测**

为 `HTTPWechatSessionClient` 增加 Observer，并把构造函数改为 variadic Observer。分别在三个实际请求创建成功后 Start：

```go
call := externalcall.Start(c.observer, externalcall.ProviderWechat, externalcall.OperationCodeToSession)
```

三个方法使用相同优先级：

1. `client.Do` 失败：`ClassifyTransport`。
2. body 读取失败：`transport_error`。
3. 非 2xx：`http_error` 优先，不改变当前业务返回路径。
4. JSON 失败或必需字段缺失：`decode_error`。
5. 微信 `errcode != 0`：HTTP 为 2xx 时 `provider_rejected`。
6. 有效响应：`success`。

本地 code 校验、历史开发凭证拒绝、配置缺失和 URL/Request 构造失败不记录外部调用。GetPhoneNumber 的 token 请求与手机号请求分别记录；缓存 token 不产生外呼事件。

- [ ] **Step 4：运行认证包测试并提交**

Run: `cd backend && go test ./app/internal/logic/auth -count=1`

Expected: PASS，原有中文公开错误断言保持不变。

```bash
git add backend/app/internal/logic/auth/wechat_session.go backend/app/internal/logic/auth/wechat_session_test.go
git commit -m "feat: 观测微信认证第三方调用"
```

---

### Task 5：接入微信内容安全外呼观测

**Files:**
- Modify: `backend/app/internal/logic/contentaudit/wechat_auditor.go`
- Modify: `backend/app/internal/logic/contentaudit/wechat_auditor_test.go`

**Interfaces:**
- Consumes: `externalcall.Observer`、`access_token`、`content_text_check`、`content_media_submit`。
- Produces: `NewWechatAuditor(wechat config.WechatConfig, cfg config.ContentAuditConfig, mediaBaseURL string, client *http.Client, observers ...externalcall.Observer) *WechatAuditor`。
- Produces: `doJSON(req, out) (statusCode int, outcome externalcall.Outcome, err error)`。

- [ ] **Step 1：先写内容审核事件测试**

扩展现有文本/图片审核成功测试，注入 Observer 并明确事件顺序：

- 首次完整审核：`access_token/success`、`content_text_check/success`，每张图片各一个 `content_media_submit/success`。
- 第二次审核命中 token 缓存：不再产生 `access_token` 事件。
- 文本接口 HTTP 200 且 `errcode != 0`：`content_text_check/provider_rejected`。
- 图片接口 HTTP 200 但缺少 `trace_id`：`content_media_submit/decode_error`。
- 任一接口非法 JSON：对应 operation 的 `decode_error`。
- Transport 超时：对应 operation 的 `timeout`。
- 风险审核决定 `suggest=risky`：外部调用为 `success`，不得记录 `provider_rejected`。

每个用例断言事件数量、顺序、provider、operation、outcome 和 status code；现有审核 Decision、Labels 和 TraceID 断言继续保留。

- [ ] **Step 2：运行内容审核测试确认失败**

Run: `cd backend && go test ./app/internal/logic/contentaudit -run 'TestWechatAuditor' -count=1`

Expected: FAIL，构造函数/事件尚未实现。

- [ ] **Step 3：重构 doJSON 返回稳定分类**

把 `doJSON` 改为只负责 HTTP/JSON 层并返回分类，不直接上报：

```go
func (a *WechatAuditor) doJSON(req *http.Request, out any) (int, externalcall.Outcome, error) {
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, externalcall.ClassifyTransport(req.Context(), err), err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return resp.StatusCode, externalcall.OutcomeTransportError, fmt.Errorf("读取微信内容安全响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp.StatusCode, externalcall.OutcomeHTTPError, fmt.Errorf("微信内容安全 HTTP 状态异常: status=%d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, externalcall.OutcomeDecodeError, fmt.Errorf("解析微信内容安全响应失败: %w", err)
	}
	return resp.StatusCode, externalcall.OutcomeSuccess, nil
}
```

- [ ] **Step 4：在三个调用点提交唯一终态**

为 `WechatAuditor` 增加 Observer 和 variadic 构造参数。`getAccessToken`、`checkText`、`submitMedia` 各自在调用 `doJSON` 前 Start；`doJSON` 失败时使用其返回 outcome；微信 `ErrCode != 0` 使用 `provider_rejected`；access token/trace ID 等必需字段缺失使用 `decode_error`；其余使用 `success`。

文本 `suggest=risky`、`review` 或 `pass` 都表示接口成功，只能记录 `success`。`normalizeMediaURL` 不发 HTTP 请求，不增加媒体下载 operation。

- [ ] **Step 5：运行内容审核测试并提交**

Run: `cd backend && go test ./app/internal/logic/contentaudit ./app/internal/logic/resource -count=1`

Expected: PASS，资源审核状态机和旧回调代际测试保持通过。

```bash
git add backend/app/internal/logic/contentaudit/wechat_auditor.go backend/app/internal/logic/contentaudit/wechat_auditor_test.go
git commit -m "feat: 观测微信内容安全调用"
```

---

### Task 6：接入微信支付观测并统一依赖注入

**Files:**
- Modify: `backend/app/internal/logic/payment/wechat_pay_gateway.go`
- Modify: `backend/app/internal/logic/payment/wechat_pay_gateway_test.go`
- Modify: `backend/app/internal/svc/service_context.go`
- Modify: `backend/app/internal/svc/service_context_test.go`

**Interfaces:**
- Consumes: `externalcall.Observer`、`pay_create`、`pay_query`、`pay_close`。
- Produces: `NewHTTPWechatPayGateway(cfg config.WechatPayConfig, observers ...externalcall.Observer) (*HTTPWechatPayGateway, error)`。
- Produces: `ServiceContext.ExternalCallObserver externalcall.Observer`，同一实例内所有第三方客户端共享。

- [ ] **Step 1：先写支付观测失败测试**

修改 `TestHTTPWechatPayGatewayQueriesAndClosesOrder`，为直接构造的 Gateway 注入 Observer，断言事件按顺序为：

```go
[]externalcall.Event{
	{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayQuery, Outcome: externalcall.OutcomeSuccess, StatusCode: 200},
	{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayClose, Outcome: externalcall.OutcomeSuccess, StatusCode: 204},
}
```

比较时忽略非确定 Duration，但断言 Duration 非负。

新增：

- `TestHTTPWechatPayGatewayObservesCreatePrepaySuccess`：复用测试 RSA 私钥和 `signedWechatHTTPResponse` 返回 `{"prepay_id":"wx-prepay-1"}`，断言 `pay_create/success/200`。
- `TestHTTPWechatPayGatewayObservesQueryTimeout`：Transport 返回 `context.DeadlineExceeded`，断言 `pay_query/timeout/0`。
- `TestHTTPWechatPayGatewayObservesCloseHTTPError`：返回 HTTP 500，断言 `pay_close/http_error/500`。
- 修改 `TestHTTPWechatPayGatewayRejectsUnsignedQueryResponse`：断言 `pay_query/decode_error/200`。
- `TestHTTPWechatPayGatewayDoesNotObserveNotifyDecode`：调用入站 `DecodeNotify` 失败，断言没有 `external_call`，因为回调处理没有外呼。

- [ ] **Step 2：运行支付测试确认失败**

Run: `cd backend && go test ./app/internal/logic/payment -run TestHTTPWechatPayGateway -count=1`

Expected: FAIL，Gateway 尚无 Observer 或未产生事件。

- [ ] **Step 3：实现支付三个外呼终态**

为 Gateway 增加 Observer，构造函数使用 variadic Observer。`CreatePrepay`、`QueryOrder`、`CloseOrder` 在请求完成本地签名且即将 `httpClient.Do` 前 Start：

- Do 失败：`ClassifyTransport`。
- body 读取失败：`transport_error`。
- 非 2xx：`http_error`；仍返回现有 `WechatPayAPIError` 或现有友好错误。
- 响应验签失败、JSON 失败、必需字段缺失：`decode_error`。
- 有效响应：`success`。

微信支付 API 使用非 2xx 表达供应商错误，本批不把这些错误强行改成 `provider_rejected`。`DecodeNotify`、本地私钥签名和参数校验均不属于外部调用，不产生事件。

- [ ] **Step 4：先写 ServiceContext 共享 Observer 失败测试**

在 `service_context_test.go` 的 `TestNewServiceContextBuildsServerDependencies` 增加：

```go
if ctx.ExternalCallObserver == nil {
	t.Fatal("ExternalCallObserver = nil, want shared production observer")
}
```

Run: `cd backend && go test ./app/internal/svc -run TestNewServiceContextBuildsServerDependencies -count=1`

Expected: FAIL，字段尚未定义。

- [ ] **Step 5：在 ServiceContext 创建并注入共享 Observer**

在 `ServiceContext` 增加字段，并在 `NewServiceContext` 开头创建：

```go
externalCallObserver := externalcall.NewLogObserver()
```

将其传入：

```go
paymentlogic.NewHTTPWechatPayGateway(c.WechatPay, externalCallObserver)
contentaudit.NewWechatAuditor(c.Wechat, c.ContentAudit, c.Storage.PublicBaseURL, nil, externalCallObserver)
locationlogic.NewTencentMapGeocoder(c.TencentMap, nil, externalCallObserver)
authlogic.NewWechatSessionClient(c.Wechat, "", nil, externalCallObserver)
authlogic.NewConfiguredSMSVerifierWithLimiter(c.SMS, nil, authlogic.NewSQLSMSSendLimiter(db), externalCallObserver)
```

并把 `ExternalCallObserver: externalCallObserver` 写入返回结构。不得改动现有数据库 Model、Token、支付或审核依赖的初始化顺序。

- [ ] **Step 6：运行支付、依赖注入和后端全量测试并提交**

Run: `cd backend && go test ./app/internal/logic/payment ./app/internal/svc -count=1`

Run: `cd backend && go test ./...`

Expected: PASS。

```bash
git add backend/app/internal/logic/payment/wechat_pay_gateway.go backend/app/internal/logic/payment/wechat_pay_gateway_test.go backend/app/internal/svc/service_context.go backend/app/internal/svc/service_context_test.go
git commit -m "feat: 观测微信支付并统一外呼依赖"
```

---

### Task 7：更新技术债、开发、部署与发布文档

**Files:**
- Modify: `docs/product/technical-architecture.md:665`
- Modify: `docs/product/technical-architecture.md:765`
- Modify: `docs/product/technical-architecture.md:793`
- Modify: `docs/product/local-dev-runbook.md:68`
- Modify: `docs/product/local-dev-runbook.md:189`
- Modify: `docs/product/production-release-checklist.md:83`
- Modify: `docs/deployment.md:30`
- Modify: `docs/deployment.md:149`
- Modify: `docs/superpowers/specs/2026-08-05-production-reliability-debt-governance-design.md`

**Interfaces:**
- Consumes: `make check-postgres`、`event=external_call` 和所有稳定枚举。
- Produces: 后续维护者可直接执行的本地验证、生产日志查询、发布抽检和剩余技术债清单。

- [ ] **Step 1：更新当前架构和真实技术债**

在 `technical-architecture.md`：

- 外部依赖/可观测性章节增加 `external_call` 字段、Outcome 和不记录敏感值的约束。
- 将 16.3 从“进程内限流”改为“跨实例限流已完成，验证门禁需长期保持”，写明 `SQLSMSSendLimiter + sms_send_limits`。
- 将 16.4 改为“已有统一调用日志但尚无熔断与指标平台”，明确是否引入熔断由真实基线决定。
- 保留路由双轨、单体大文件、历史兼容代码等后续债务，不宣称已经解决。

- [ ] **Step 2：更新本地、部署与发布手册**

`local-dev-runbook.md` 增加命令：

```bash
WPLINK_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:5432/wplink_integration?sslmode=disable' make check-postgres
```

紧邻命令明确：目标库必须可丢弃并已执行全部 up migrations；禁止传入生产 DSN；普通 `make check` 不替代该验证。

`deployment.md` 增加实际查询：

```bash
rg -n '"event":"external_call"' /opt/wplink/logs
rg -n '"event":"external_call".*"outcome":"(timeout|transport_error|http_error|decode_error)"' /opt/wplink/logs
rg -n '"event":"external_call".*"provider":"wechat".*"operation":"pay_' /opt/wplink/logs
```

记录 24 小时基线及建议告警：5 分钟至少 20 次且故障结果超过 5%；同 operation 5 分钟至少 3 次 timeout；支付 decode error 高优先级。说明 `provider_rejected` 不自动等于供应商宕机，`canceled` 不计入故障率。

`production-release-checklist.md` 增加发布后抽检：字段完整、没有敏感值、观察 24 小时后再设置阈值。

- [ ] **Step 3：同步设计文档事实修正并验证真实命令**

确认设计文档的 Operation 清单不存在 `content_media_download`，并明确支付非 2xx 覆盖 `http_error`，不是强制 `provider_rejected`。

Run: `cd backend && node --test scripts/validate_migrations.test.mjs`

Run: `env -u WPLINK_TEST_POSTGRES_DSN make check-postgres`

Expected: migration 验证 PASS；PostgreSQL 门禁因缺少测试 DSN 按设计明确失败。文档内容由任务审查直接核对，不增加读取文档源码并匹配固定文字的脆弱测试。

- [ ] **Step 4：提交文档闭环**

```bash
git add docs/product/technical-architecture.md docs/product/local-dev-runbook.md docs/product/production-release-checklist.md docs/deployment.md docs/superpowers/specs/2026-08-05-production-reliability-debt-governance-design.md
git commit -m "docs: 完善生产可靠性维护闭环"
```

---

### Task 8：执行最终验证与范围审计

**Files:**
- Verify only; do not create implementation files unless a failing test proves an in-scope defect.

**Interfaces:**
- Consumes: Tasks 1–7 的所有产物。
- Produces: 可复核的完整测试记录、干净工作区和剩余环境说明。

- [ ] **Step 1：格式化和静态差异检查**

Run:

```bash
cd backend && gofmt -w \
  common/externalcall/observer.go common/externalcall/observer_test.go \
  common/externalcall/log_observer.go common/externalcall/log_observer_test.go \
  app/internal/logic/auth/sms_send_limiter_integration_test.go \
  app/internal/logic/auth/sms_verifier.go app/internal/logic/auth/sms_verifier_test.go \
  app/internal/logic/auth/wechat_session.go app/internal/logic/auth/wechat_session_test.go \
  app/internal/logic/location/tencent_map_geocoder.go app/internal/logic/location/tencent_map_geocoder_test.go \
  app/internal/logic/contentaudit/wechat_auditor.go app/internal/logic/contentaudit/wechat_auditor_test.go \
  app/internal/logic/payment/wechat_pay_gateway.go app/internal/logic/payment/wechat_pay_gateway_test.go \
  app/internal/svc/service_context.go app/internal/svc/service_context_test.go
```

Run: `git diff --check`

Expected: 无格式和空白错误。格式化命令只能改变本计划涉及的 Go 文件；执行前先用 `git diff --name-only` 核对范围，不能覆盖其他并行工作。

- [ ] **Step 2：执行真实 PostgreSQL 门禁**

在已创建且完整迁移的可丢弃 `wplink_integration` 上运行：

```bash
WPLINK_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:5432/wplink_integration?sslmode=disable' make check-postgres
```

Expected: Coordinator 三项和 SQL 短信限流四项均 PASS，没有 SKIP。若本机没有可丢弃 PostgreSQL，必须使用 CI PostgreSQL service 或一次性本地 PostgreSQL；不得用生产库替代验证。

- [ ] **Step 3：执行项目完整门禁**

Run: `make check`

Expected:

- migration/API 契约测试全部 PASS。
- `go test ./...` 和 `go vet ./...` PASS。
- 管理端测试与 Vite 构建 PASS。
- 小程序页面/流程校验、Node 测试和微信小程序构建 PASS。
- 允许记录现有 Sass legacy API / `@import` 弃用提示，但不得出现新增失败。

- [ ] **Step 4：审计外呼覆盖和敏感字段**

Run:

```bash
rg -n 'client\.Do|httpClient\.Do' backend/app/internal/logic/auth backend/app/internal/logic/contentaudit backend/app/internal/logic/location backend/app/internal/logic/payment
rg -n 'external_call|Operation(CodeToSession|PhoneNumber|AccessToken|ContentTextCheck|ContentMediaSubmit|PayCreate|PayQuery|PayClose|SMSCodeSend|SMSCodeVerify|ReverseGeocode)' backend/common backend/app/internal
```

Expected: 每个当前核心 `Do` 调用都能对应一个 operation；没有 `content_media_download`；公共日志字段中没有请求体、响应体、Token、手机号或错误全文。

- [ ] **Step 5：检查提交与工作区**

Run: `git status --short`

Run: `git log --oneline --decorate -8`

Expected: 工作区干净；提交按公共组件、PostgreSQL 验证、短信/地图、微信认证、内容审核、支付/依赖注入、文档闭环分开。
