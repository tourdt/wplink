# MVP Core Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Verify and complete the MVP core loop from resource publishing through review, discovery, contact actions, and merchant metrics.

**Architecture:** Keep the existing Go HTTP router and logic/model boundaries. Add coverage to the current in-memory route test first, then make only the minimal backend fixes required by that failing test. Update the MVP TODO plan only after verification passes or a concrete environment blocker is documented.

**Tech Stack:** Go, go-zero-style internal logic packages, `net/http/httptest`, existing `common/response` envelope, PostgreSQL-backed model interfaces.

---

### Task 1: Complete Core Loop Route Test

**Files:**
- Modify: `backend/app/internal/server/resource_api_test.go`

- [ ] **Step 1: Write the failing test coverage**

Extend `TestResourceAPIRouterRunsPublishReviewSearchContactFlow` so the existing flow also calls detail-view, records a WeChat contact action, checks resource metrics, and confirms my resources includes detail, phone, and WeChat counts.

Add this block after the resource detail assertion:

```go
	detailViewRec := httptest.NewRecorder()
	detailViewReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/detail-view", nil)
	router.ServeHTTP(detailViewRec, detailViewReq)
	detailViewData := decodeEnvelopeData(t, detailViewRec, http.StatusOK)
	if detailViewData["message"] != "浏览行为已记录" {
		t.Fatalf("detail view data = %#v, want message", detailViewData)
	}
	if store.metricDelta.DetailViewCount != 1 {
		t.Fatalf("metric delta after detail view = %#v, want detail view count", store.metricDelta)
	}
```

Add this block after the phone contact assertion:

```go
	wechatRec := httptest.NewRecorder()
	wechatReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"wechat"}`))
	router.ServeHTTP(wechatRec, wechatReq)
	wechatData := decodeEnvelopeData(t, wechatRec, http.StatusOK)
	if wechatData["message"] != "联系行为已记录" {
		t.Fatalf("wechat data = %#v, want message", wechatData)
	}
	if store.contactInput.Action != "wechat" || store.metricDelta.WechatCopyCount != 1 {
		t.Fatalf("contact input = %#v metric = %#v, want wechat metric", store.contactInput, store.metricDelta)
	}
```

Add this block before the my resources request:

```go
	metricsRec := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/resource-1/metrics", nil)
	router.ServeHTTP(metricsRec, metricsReq)
	metricsData := decodeEnvelopeData(t, metricsRec, http.StatusOK)
	summary := metricsData["summary"].(map[string]interface{})
	if summary["detailViewCount"] != float64(1) || summary["phoneClickCount"] != float64(1) || summary["wechatCopyCount"] != float64(1) {
		t.Fatalf("metrics summary = %#v, want detail, phone and wechat counts", summary)
	}
```

Update the my resources assertion:

```go
	myItems := myData["items"].([]interface{})
	if len(myItems) != 1 {
		t.Fatalf("my items = %#v, want one item", myData["items"])
	}
	myItem := myItems[0].(map[string]interface{})
	myMetrics := myItem["metrics"].(map[string]interface{})
	if myMetrics["detailViewCount"] != float64(1) || myMetrics["phoneClickCount"] != float64(1) || myMetrics["wechatCopyCount"] != float64(1) {
		t.Fatalf("my metrics = %#v, want updated loop metrics", myMetrics)
	}
```

Add this fake store method:

```go
func (s *fakeResourceAPIStore) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error) {
	return model.ResourceMetricsResult{
		ResourceID: resourceID,
		Summary: model.ResourceMetricsSummary{
			DetailViewCount: 1,
			PhoneClickCount: 1,
			WechatCopyCount: 1,
		},
		Daily: []model.ResourceMetricDailyItem{{
			Date: "2026-06-28", DetailViewCount: 1, PhoneClickCount: 1, WechatCopyCount: 1,
		}},
	}, nil
}
```

Update `ListMyResources` fake metrics:

```go
Metrics: model.MyResourceMetrics{DetailViewCount: 1, PhoneClickCount: 1, WechatCopyCount: 1},
```

- [ ] **Step 2: Run test to verify it fails or exposes missing behavior**

Run: `go test ./app/internal/server -run TestResourceAPIRouterRunsPublishReviewSearchContactFlow -count=1`

Expected if a gap exists: FAIL with either 404 for `GET /api/v1/resources/{resourceId}/metrics`, missing interface implementation, or mismatched metrics.

Expected if behavior already exists: PASS, which means the missing work was test coverage only.

- [ ] **Step 3: Implement the minimal backend fix if the test fails**

If the test fails because `fakeResourceAPIStore` does not satisfy `MetricsQueryAPIStore`, add `func (s *fakeResourceAPIStore) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error)` to `backend/app/internal/server/resource_api_test.go` with detail, phone, and WeChat counts all set to `1`.

If the route is not registered for stores that satisfy `ResourceAPIStore` and `MetricsQueryAPIStore`, adjust only `registerOptionalDomainRoutes` or `NewAPIRouter` so `registerMetricsRoutes` is called when the store implements `MetricsQueryAPIStore`.

If response field names differ, keep the API contract field names from `backend/app/api/metrics.api`: `detailViewCount`, `phoneClickCount`, `wechatCopyCount`.

- [ ] **Step 4: Run focused test to verify green**

Run: `go test ./app/internal/server -run TestResourceAPIRouterRunsPublishReviewSearchContactFlow -count=1`

Expected: PASS.

### Task 2: Verify Backend and Update MVP TODO

**Files:**
- Modify: `docs/superpowers/plans/2026-06-27-apparel-platform-current-mvp-todo.md`

- [ ] **Step 1: Run backend tests**

Run: `go test ./...`

Expected: PASS for all backend packages. If the command cannot run because local services or sandbox permissions are missing, capture the exact blocker and do not mark the manual API verification complete.

- [ ] **Step 2: Update the first development slice checkbox only after green verification**

In `docs/superpowers/plans/2026-06-27-apparel-platform-current-mvp-todo.md`, change:

```markdown
- [ ] 验证：发布资源 -> 待审核 -> 后台通过 -> 出现在搜索结果 -> 进入详情 -> 触发联系动作 -> 我的发布看到指标变化。
```

to:

```markdown
- [x] 验证：发布资源 -> 待审核 -> 后台通过 -> 出现在搜索结果 -> 进入详情 -> 触发联系动作 -> 我的发布看到指标变化。
```

- [ ] **Step 3: Run focused test again after docs update**

Run: `go test ./app/internal/server -run TestResourceAPIRouterRunsPublishReviewSearchContactFlow -count=1`

Expected: PASS.

- [ ] **Step 4: Review changed files**

Run: `git diff -- backend/app/internal/server/resource_api_test.go docs/superpowers/plans/2026-06-27-apparel-platform-current-mvp-todo.md`

Expected: Diff contains only the route test coverage and the single MVP TODO checkbox update.
