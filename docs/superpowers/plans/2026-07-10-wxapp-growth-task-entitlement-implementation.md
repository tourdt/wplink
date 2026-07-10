# 小程序成长任务权益页实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把小程序“新手权益”从活动规则说明页优化为“新手主线任务 + 每日曝光任务”的成长任务页，并通过后端接口返回任务进度、奖励和下一步动作。

**Architecture:** 后端新增成长任务汇总 logic，复用现有增长活动规则、发放记录、权益余额、资源和联系事件数据，只读计算任务进度，不在任务接口里发放权益。路由挂在 `GET /api/v1/merchants/{merchantId}/growth-tasks`，沿用商家管理权限校验。小程序新增 `getGrowthTasks` API，重构 `pages/my/growth-entitlement.vue` 为余额概览、主线任务和每日曝光任务三块展示。

**Tech Stack:** Go、go-zero 风格手写 HTTP router、PostgreSQL SQL、uni-app Vue 3、小程序 Node.js 文本测试、Go 单元测试。

---

## 文件结构

- Create: `backend/app/internal/logic/growth/growth_task_logic.go`
  - 负责成长任务接口的请求校验、任务分组、状态计算、奖励和按钮文案组装。
- Create: `backend/app/internal/logic/growth/growth_task_logic_test.go`
  - 覆盖无活动、无资源、待审核、已到账、每日任务进度和余额汇总。
- Modify: `backend/app/internal/model/growth_campaign_model.go`
  - 新增只读查询结构和方法：商家发放记录、商家资源进度、每日有效联系和分享浏览统计。
- Modify: `backend/app/internal/model/growth_campaign_model_test.go`
  - 校验新增 SQL 包含核心过滤条件，避免统计口径回退。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 将成长任务接口接入用户 token 和商家权限校验。
- Modify: `backend/app/internal/server/remaining_api_test.go`
  - 覆盖路由注册、权限校验和响应字段。
- Modify: `wxapp/api/growthCampaign.js`
  - 新增 `getGrowthTasks(merchantId, options)`。
- Modify: `wxapp/pages/my/growth-entitlement.vue`
  - 使用任务汇总接口渲染余额、主线任务和每日任务，保留旧接口失败时的友好空态。
- Modify: `wxapp/pages/my/growth-entitlement.test.mjs`
  - 更新页面文本测试，确保任务页结构、接口和按钮动作存在。

## Task 1: 后端成长任务 logic

**Files:**
- Create: `backend/app/internal/logic/growth/growth_task_logic_test.go`
- Create: `backend/app/internal/logic/growth/growth_task_logic.go`

- [ ] **Step 1: 写失败测试**

在 `backend/app/internal/logic/growth/growth_task_logic_test.go` 新增测试，构造 fake store，覆盖：

```go
func TestGetGrowthTasksBuildsStarterAndDailyTasks(t *testing.T) {
	store := &fakeGrowthTaskStore{
		campaigns: []model.PublicGrowthCampaign{{
			Code: "starter_growth_2026_q3", Name: "新手成长权益活动", Title: "新手发布权益", Hint: "发布优质资源、有效分享可获得更多曝光权益",
			Rules: []model.PublicGrowthRule{
				{RuleCode: "first_login_publish_quota", RuleName: "首次登录赠送发布次数", TriggerEvent: model.GrowthEventUserFirstLogin, RewardType: model.EntitlementTypePublishQuota, RewardAmount: 3, ValidDays: 30, PerUserLimit: 1},
				{RuleCode: "first_resource_approved_publish_quota", RuleName: "首条资源审核通过奖励", TriggerEvent: model.GrowthEventResourceFirstApproved, RewardType: model.EntitlementTypePublishQuota, RewardAmount: 5, ValidDays: 30, PerUserLimit: 1},
				{RuleCode: "three_approved_resources_publish_quota", RuleName: "连续优质发布奖励发布次数", TriggerEvent: model.GrowthEventResourceApprovedCountReached, RewardType: model.EntitlementTypePublishQuota, RewardAmount: 5, ValidDays: 30, PerUserLimit: 1, Conditions: model.JSONMap{"approvedCount": float64(3), "windowDays": float64(30)}},
				{RuleCode: "share_contact_refresh_quota", RuleName: "分享带来有效联系奖励", TriggerEvent: model.GrowthEventResourceShareEffectiveContact, RewardType: model.EntitlementTypeRefreshQuota, RewardAmount: 1, ValidDays: 15, PerUserDailyLimit: 3},
			},
		}},
		entitlements: []model.MerchantEntitlement{{Type: model.EntitlementTypePublishQuota, RemainingAmount: 4}, {Type: model.EntitlementTypeRefreshQuota, RemainingAmount: 2}},
		progress: model.GrowthTaskProgress{TotalResourceCount: 1, PendingResourceCount: 1, PublishedResourceCount: 1, ApprovedWithinWindowCount: 1, TodayEffectiveContactCount: 1},
		grants: []model.MerchantGrowthRewardGrant{{RuleCode: "first_login_publish_quota", Status: "granted"}},
	}
	resp, err := NewGrowthTaskLogic(store).GetGrowthTasks(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Summary.PublishQuotaRemaining != 4 || resp.Summary.RefreshQuotaRemaining != 2 {
		t.Fatalf("summary = %#v, want quota totals", resp.Summary)
	}
	if resp.Summary.StarterTotalCount != 3 || resp.Summary.StarterCompletedCount != 1 {
		t.Fatalf("starter summary = %#v, want one completed of three", resp.Summary)
	}
	if len(resp.Tasks) != 4 {
		t.Fatalf("tasks length = %d, want 4", len(resp.Tasks))
	}
	assertGrowthTask(t, resp.Tasks, "first_login_publish_quota", "starter", "granted", 1, 1, "去发布")
	assertGrowthTask(t, resp.Tasks, "first_resource_approved_publish_quota", "starter", "pending_review", 1, 1, "查看发布")
	assertGrowthTask(t, resp.Tasks, "three_approved_resources_publish_quota", "starter", "in_progress", 1, 3, "去发布")
	assertGrowthTask(t, resp.Tasks, "share_contact_refresh_quota", "daily", "in_progress", 1, 3, "去分享")
}
```

同时增加 `TestGetGrowthTasksReturnsEmptyWhenNoActiveCampaign` 和 `TestGetGrowthTasksRejectsEmptyMerchantID`。

- [ ] **Step 2: 运行测试确认失败**

Run from `backend`: `go test ./app/internal/logic/growth`

Expected: FAIL，报 `undefined: NewGrowthTaskLogic` 或缺少响应类型。

- [ ] **Step 3: 实现最小 logic**

在 `growth_task_logic.go` 定义：

```go
type GrowthTaskStore interface {
	ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error)
	ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error)
	GetGrowthTaskProgress(ctx context.Context, merchantID string) (model.GrowthTaskProgress, error)
	ListMerchantGrowthRewardGrants(ctx context.Context, merchantID string) ([]model.MerchantGrowthRewardGrant, error)
}
```

实现 `GetGrowthTasks(ctx, merchantID)`：

- trim merchantID，空值返回 `errx.New(errx.CodeValidationFailed, "商家不存在")`。
- 无活动返回空 `tasks` 和余额汇总。
- 从第一条 active campaign 读取规则。
- `per_user_limit = 1` 且发布类/资源类规则归入 `starter`。
- 存在 `per_user_daily_limit` 或 `per_resource_daily_limit` 的分享类规则归入 `daily`。
- `first_login` 已发放为 `granted`，否则 `todo`。
- `resource_first_approved` 有 grant 为 `granted`；无 grant 但有 pending 资源为 `pending_review`；有发布资源但未到账为 `completed`；无资源为 `todo`。
- `resource_approved_count_reached` 目标从 `conditions.approvedCount` 读取，进度用 `ApprovedWithinWindowCount`。
- `resource_share_effective_contact` 目标优先用 `PerUserDailyLimit`，进度用 `TodayEffectiveContactCount`。
- `resource_share_effective_view` 目标从 `conditions.viewThreshold` 读取，进度用 `ShareViewWindowCount`。

- [ ] **Step 4: 运行 logic 测试通过**

Run from `backend`: `go test ./app/internal/logic/growth`

Expected: PASS。

## Task 2: 后端模型查询

**Files:**
- Modify: `backend/app/internal/model/growth_campaign_model.go`
- Modify: `backend/app/internal/model/growth_campaign_model_test.go`

- [ ] **Step 1: 写失败测试**

在 `growth_campaign_model_test.go` 新增字符串级 SQL 测试，要求模型包含：

```go
func TestGrowthTaskProgressSQLUsesResourceAndContactSignals(t *testing.T) {
	checks := []string{
		"COUNT(*) FILTER (WHERE status = 'published')",
		"COUNT(*) FILTER (WHERE status = 'pending')",
		"created_at >= date_trunc('day', now())",
		"action IN ('phone', 'wechat')",
		"action = 'share_view'",
	}
	for _, snippet := range checks {
		if !strings.Contains(growthTaskProgressSQL, snippet) {
			t.Fatalf("growthTaskProgressSQL missing %q:\n%s", snippet, growthTaskProgressSQL)
		}
	}
}
```

并检查发放记录 SQL：

```go
func TestMerchantGrowthRewardGrantsSQLFiltersGrantedByMerchant(t *testing.T) {
	for _, snippet := range []string{"FROM growth_reward_grants", "merchant_id = $1", "status = 'granted'"} {
		if !strings.Contains(listMerchantGrowthRewardGrantsSQL, snippet) {
			t.Fatalf("listMerchantGrowthRewardGrantsSQL missing %q:\n%s", snippet, listMerchantGrowthRewardGrantsSQL)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run from `backend`: `go test ./app/internal/model -run 'TestGrowthTask|TestMerchantGrowth'`

Expected: FAIL，缺少 SQL 常量和模型类型。

- [ ] **Step 3: 实现模型类型和查询**

在 `growth_campaign_model.go` 新增：

```go
type GrowthTaskProgress struct {
	TotalResourceCount       int64
	PendingResourceCount     int64
	PublishedResourceCount   int64
	ApprovedWithinWindowCount int64
	TodayEffectiveContactCount int64
	ShareViewWindowCount     int64
}

type MerchantGrowthRewardGrant struct {
	RuleCode string
	Status   string
}
```

新增 SQL 常量和方法：

- `GetGrowthTaskProgress(ctx, merchantID string) (GrowthTaskProgress, error)`
- `ListMerchantGrowthRewardGrants(ctx, merchantID string) ([]MerchantGrowthRewardGrant, error)`

查询只读，不写发放记录。内部错误原样返回给 logic/router 的统一响应层。

- [ ] **Step 4: 运行模型测试通过**

Run from `backend`: `go test ./app/internal/model -run 'TestGrowthTask|TestMerchantGrowth'`

Expected: PASS。

## Task 3: 后端路由接入

**Files:**
- Modify: `backend/app/internal/server/domain_routes.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`

- [ ] **Step 1: 写失败测试**

在 `remaining_api_test.go` 新增 `TestAPIRouterRequiresMerchantPermissionForGrowthTasks`：

```go
func TestAPIRouterRequiresMerchantPermissionForGrowthTasks(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-2/growth-tasks", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/growth-tasks", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	data := decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if _, ok := data["tasks"].([]interface{}); !ok {
		t.Fatalf("data = %#v, want tasks array", data)
	}
}
```

同时给 `fakeFullAPIStore` 增加 fake 数据和接口方法。

- [ ] **Step 2: 运行测试确认失败**

Run from `backend`: `go test ./app/internal/server -run TestAPIRouterRequiresMerchantPermissionForGrowthTasks`

Expected: FAIL，路由返回 404 或 store 未实现接口。

- [ ] **Step 3: 接入路由和接口**

在 `domain_routes.go`：

- 新增独立的 `GrowthTaskAPIStore`，由它包含 `growthlogic.GrowthTaskStore`，避免公开活动接口和私有商家任务接口混在同一个接口里。
- 新增独立的 `registerGrowthTaskRoutes(mux, store, tokenService, adminTokenService, permissionStore)`，并在 `registerOptionalDomainRoutes` 的增长活动分支中调用。
- 新增：

```go
mux.HandleFunc("GET /api/v1/merchants/{merchantId}/growth-tasks", func(w http.ResponseWriter, r *http.Request) {
	merchantID := r.PathValue("merchantId")
	if err := requireMerchantPermission(r, tokenService, permissionStore, merchantID); err != nil {
		response.JSON(w, nil, err)
		return
	}
	resp, err := growthlogic.NewGrowthTaskLogic(store).GetGrowthTasks(r.Context(), merchantID)
	response.JSON(w, resp, err)
})
```

- [ ] **Step 4: 运行 server 测试通过**

Run from `backend`: `go test ./app/internal/server -run TestAPIRouterRequiresMerchantPermissionForGrowthTasks`

Expected: PASS。

## Task 4: 小程序 API 与页面改造

**Files:**
- Modify: `wxapp/api/growthCampaign.js`
- Modify: `wxapp/pages/my/growth-entitlement.test.mjs`
- Modify: `wxapp/pages/my/growth-entitlement.vue`

- [ ] **Step 1: 写失败测试**

更新 `growth-entitlement.test.mjs`，断言：

```js
assert.match(apiSource, /getGrowthTasks/)
assert.match(apiSource, /\/api\/v1\/merchants\/\$\{merchantId\}\/growth-tasks/)
assert.match(source, /新手成长进度/)
assert.match(source, /新手主线任务/)
assert.match(source, /每日曝光任务/)
assert.match(source, /taskStatusText/)
assert.match(source, /openTaskAction/)
assert.match(source, /starterTasks/)
assert.match(source, /dailyTasks/)
assert.doesNotMatch(source, /可继续获得/)
```

- [ ] **Step 2: 运行测试确认失败**

Run from `wxapp`: `node --test pages/my/growth-entitlement.test.mjs`

Expected: FAIL，缺少新 API 和页面文案。

- [ ] **Step 3: 实现小程序 API**

在 `wxapp/api/growthCampaign.js` 添加：

```js
export function getGrowthTasks(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/growth-tasks`,
    method: 'GET',
    ...options,
  })
}
```

- [ ] **Step 4: 改造页面数据流和模板**

在 `growth-entitlement.vue`：

- 引入 `getGrowthTasks`。
- 加载任务汇总接口，保留 `getMerchantEntitlements` 作为接口失败或历史权益补充。
- 新增 computed：`taskSummary`、`starterTasks`、`dailyTasks`、`quotaCards`。
- 模板改成余额概览、主线任务、每日曝光任务、说明四块。
- `openTaskAction(task)` 根据 `actionType` 跳转：
  - `publish` -> `/pages/publish/index`
  - `share`、`manage_resources` -> `/pages/my-resources/index`
  - `use_entitlement` -> 发布次数走 `/pages/publish/index`，刷新次数走 `/pages/my-resources/index`

- [ ] **Step 5: 运行小程序页面测试通过**

Run from `wxapp`: `node --test pages/my/growth-entitlement.test.mjs`

Expected: PASS。

## Task 5: 集成验证与提交

**Files:**
- All touched files.

- [ ] **Step 1: 格式化 Go 文件**

Run: `gofmt -w backend/app/internal/logic/growth/growth_task_logic.go backend/app/internal/logic/growth/growth_task_logic_test.go backend/app/internal/model/growth_campaign_model.go backend/app/internal/model/growth_campaign_model_test.go backend/app/internal/server/domain_routes.go backend/app/internal/server/remaining_api_test.go`

Expected: 命令无输出。

- [ ] **Step 2: 运行后端相关测试**

Run from `backend`: `go test ./app/internal/logic/growth ./app/internal/model ./app/internal/server`

Expected: PASS。

- [ ] **Step 3: 运行小程序相关测试**

Run from `wxapp`: `node --test pages/my/growth-entitlement.test.mjs pages/my/index.test.mjs`

Expected: PASS。

- [ ] **Step 4: 查看 diff**

Run: `git diff --stat`

Expected: 只包含成长任务接口、增长模型查询、小程序成长权益页和本计划文件相关变更；已有未提交的旧改动不回退。

- [ ] **Step 5: 暂存并提交本次实现**

Run:

```bash
git add docs/superpowers/plans/2026-07-10-wxapp-growth-task-entitlement-implementation.md backend/app/internal/logic/growth/growth_task_logic.go backend/app/internal/logic/growth/growth_task_logic_test.go backend/app/internal/model/growth_campaign_model.go backend/app/internal/model/growth_campaign_model_test.go backend/app/internal/server/domain_routes.go backend/app/internal/server/remaining_api_test.go wxapp/api/growthCampaign.js wxapp/pages/my/growth-entitlement.vue wxapp/pages/my/growth-entitlement.test.mjs
git commit -m "feat: add wxapp growth task entitlement progress"
```

Expected: commit succeeds.
