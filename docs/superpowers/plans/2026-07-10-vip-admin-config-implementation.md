# VIP 管理后台运营配置 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在管理后台新增完整 VIP 配置能力，让运营可维护 VIP 套餐、次数包和优惠活动，并让小程序继续读取最新可售配置。

**Architecture:** 后端复用现有 `vip_plans`、`vip_plan_versions`、`vip_quota_packs`、`vip_promotions` 表，新增后台专用 logic/store 方法和 `/api/v1/admin/vip/*` 手写路由；套餐权益保存时插入新版本，历史订单继续依赖 `vip_orders.benefits_snapshot`。前端新增 `VIPConfigView`，以三个 tab 分别维护套餐、次数包、优惠，金额在页面显示为元、提交为分。

**Tech Stack:** Go 1.23、go-zero API contract/goctl generated types、PostgreSQL、Vue 3、Element Plus、Node.js `node:test`、Go test。

---

## 文件结构

- Modify: `backend/app/api/admin.api`
  - 增加后台 VIP 配置 DTO 和 `/api/v1/admin/vip/*` 合约。
- Modify: `backend/app/internal/types/types.go`
  - 由 goctl 生成同步后台 VIP 配置 DTO。
- Create: `backend/app/internal/logic/admin/vip_config_logic.go`
  - 后台 VIP 配置业务校验、输入归一化、响应映射。
- Create: `backend/app/internal/logic/admin/vip_config_logic_test.go`
  - 后台 VIP 配置 logic TDD 测试。
- Modify: `backend/app/internal/model/vip_model.go`
  - 增加后台列表和保存套餐、次数包、优惠的持久化方法。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 注册 `/api/v1/admin/vip/*` 路由，并从管理员 token 绑定操作人。
- Modify: `backend/app/internal/server/remaining_api_test.go`
  - 增加后台 VIP 配置路由测试和 fake store 方法。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 扩展 `VIPAPIStore` 或新增后台 VIP 配置 Store 接口。
- Modify: `backend/scripts/api_contract.test.mjs`
  - 增加后台 VIP 配置 API 合约断言。
- Create: `admin-web/src/api/vipConfig.js`
  - 管理后台 VIP 配置接口封装。
- Create: `admin-web/src/views/VIPConfigView.vue`
  - VIP 配置页面。
- Modify: `admin-web/src/router/index.js`
  - 注册 `VIPConfigView` 懒加载路由。
- Modify: `admin-web/src/layouts/AdminLayout.vue`
  - 增加“VIP 配置”菜单入口。
- Modify: `admin-web/scripts/feature-visibility.test.mjs`
  - 增加后台入口、tab 和 API 前缀静态断言。
- Modify: `admin-web/src/plugins/elementPlus.js`
  - 如页面使用新组件，则补充按需注册。

## Task 1: API 合约和 goctl 类型

**Files:**
- Modify: `backend/scripts/api_contract.test.mjs`
- Modify: `backend/app/api/admin.api`
- Modify: `backend/app/internal/types/types.go`

- [ ] **Step 1: Write the failing test**

在 `backend/scripts/api_contract.test.mjs` 增加测试：

```js
test('admin api contract exposes vip config endpoints', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type AdminVIPBenefitConfig',
    'type AdminVIPPlanConfigItem',
    'type AdminSaveVIPPlanConfigReq',
    'type AdminQuotaPackConfigItem',
    'type AdminSaveQuotaPackConfigReq',
    'type AdminVIPPromotionConfigItem',
    'type AdminSaveVIPPromotionConfigReq',
    'get /vip/plans returns (AdminListVIPPlansResp)',
    'post /vip/plans (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/plans/:planCode (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)',
    'get /vip/quota-packs returns (AdminListQuotaPacksResp)',
    'post /vip/quota-packs (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/quota-packs/:packCode (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)',
    'get /vip/promotions returns (AdminListVIPPromotionsResp)',
    'post /vip/promotions (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/promotions/:promotionCode (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)',
  ]) {
    assert(adminApiSource.includes(snippet), `admin.api should contain ${snippet}`)
  }

  for (const snippet of [
    'type AdminVIPPlanConfigItem struct',
    'type AdminSaveVIPPlanConfigReq struct',
    'type AdminQuotaPackConfigItem struct',
    'type AdminVIPPromotionConfigItem struct',
  ]) {
    assert(typesSource.includes(snippet), `types.go should contain ${snippet}`)
  }
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && node --test scripts/api_contract.test.mjs`

Expected: FAIL with missing `AdminVIPBenefitConfig` or missing `/vip/plans` admin route snippets.

- [ ] **Step 3: Update admin.api**

Add these DTOs before the `@server(prefix: /api/v1/admin group: adminconfig)` block:

```go
type AdminVIPBenefitConfig {
	PublishPolicy      string `json:"publishPolicy,optional"`
	PublishQuota       int64  `json:"publishQuota"`
	RefreshQuota       int64  `json:"refreshQuota"`
	TopVoucherCount    int64  `json:"topVoucherCount"`
	TopDurationHours   int64  `json:"topDurationHours"`
	HomepageImageLimit int64  `json:"homepageImageLimit,optional"`
}

type AdminVIPPlanConfigItem {
	Code              string                `json:"code"`
	Name              string                `json:"name"`
	DurationMonths    int64                 `json:"durationMonths"`
	StandardPriceCent int64                 `json:"standardPriceCent"`
	Status            string                `json:"status"`
	DisplayOrder      int64                 `json:"displayOrder"`
	Benefits          AdminVIPBenefitConfig `json:"benefits"`
	UpdatedAt         string                `json:"updatedAt,optional"`
}

type AdminListVIPPlansResp {
	Items []AdminVIPPlanConfigItem `json:"items"`
}

type AdminSaveVIPPlanConfigReq {
	Code              string                `json:"code,optional"`
	Name              string                `json:"name"`
	DurationMonths    int64                 `json:"durationMonths"`
	StandardPriceCent int64                 `json:"standardPriceCent"`
	Status            string                `json:"status,optional"`
	DisplayOrder      int64                 `json:"displayOrder,optional"`
	Benefits          AdminVIPBenefitConfig `json:"benefits"`
}

type AdminQuotaPackConfigItem {
	Code              string                `json:"code"`
	Name              string                `json:"name"`
	Description       string                `json:"description,optional"`
	StandardPriceCent int64                 `json:"standardPriceCent"`
	SalePriceCent     int64                 `json:"salePriceCent,optional"`
	SaleLabel         string                `json:"saleLabel,optional"`
	Status            string                `json:"status"`
	DisplayOrder      int64                 `json:"displayOrder"`
	Benefits          AdminVIPBenefitConfig `json:"benefits"`
	UpdatedAt         string                `json:"updatedAt,optional"`
}

type AdminListQuotaPacksResp {
	Items []AdminQuotaPackConfigItem `json:"items"`
}

type AdminSaveQuotaPackConfigReq {
	Code              string                `json:"code,optional"`
	Name              string                `json:"name"`
	Description       string                `json:"description,optional"`
	StandardPriceCent int64                 `json:"standardPriceCent"`
	SalePriceCent     int64                 `json:"salePriceCent,optional"`
	SaleLabel         string                `json:"saleLabel,optional"`
	Status            string                `json:"status,optional"`
	DisplayOrder      int64                 `json:"displayOrder,optional"`
	Benefits          AdminVIPBenefitConfig `json:"benefits"`
}

type AdminVIPPromotionConfigItem {
	Code          string `json:"code"`
	PlanCode      string `json:"planCode"`
	PlanName      string `json:"planName,optional"`
	PromotionType string `json:"promotionType"`
	SalePriceCent int64  `json:"salePriceCent"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt,optional"`
	QuotaLimit    int64  `json:"quotaLimit,optional"`
	UsedCount     int64  `json:"usedCount"`
	Status        string `json:"status"`
	UpdatedAt     string `json:"updatedAt,optional"`
}

type AdminListVIPPromotionsResp {
	Items []AdminVIPPromotionConfigItem `json:"items"`
}

type AdminSaveVIPPromotionConfigReq {
	Code          string `json:"code,optional"`
	PlanCode      string `json:"planCode"`
	PromotionType string `json:"promotionType"`
	SalePriceCent int64  `json:"salePriceCent"`
	StartsAt      string `json:"startsAt,optional"`
	EndsAt        string `json:"endsAt,optional"`
	QuotaLimit    int64  `json:"quotaLimit,optional"`
	Status        string `json:"status,optional"`
}

type AdminSaveVIPConfigResp {
	Code      string `json:"code"`
	UpdatedAt string `json:"updatedAt"`
}
```

Add a server block:

```go
@server(
	prefix: /api/v1/admin
	group: adminvipconfig
)
service wplink-api {
	@handler AdminListVIPPlans
	get /vip/plans returns (AdminListVIPPlansResp)

	@handler AdminCreateVIPPlan
	post /vip/plans (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)

	@handler AdminUpdateVIPPlan
	post /vip/plans/:planCode (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)

	@handler AdminListQuotaPacks
	get /vip/quota-packs returns (AdminListQuotaPacksResp)

	@handler AdminCreateQuotaPack
	post /vip/quota-packs (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)

	@handler AdminUpdateQuotaPack
	post /vip/quota-packs/:packCode (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)

	@handler AdminListVIPPromotions
	get /vip/promotions returns (AdminListVIPPromotionsResp)

	@handler AdminCreateVIPPromotion
	post /vip/promotions (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)

	@handler AdminUpdateVIPPromotion
	post /vip/promotions/:promotionCode (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)
}
```

- [ ] **Step 4: Generate types without unrelated goctl churn**

Run:

```bash
cd backend/app
tmpdir="$(mktemp -d /private/tmp/wplink-goctl-admin-vip.XXXXXX)"
goctl api go -api ./api/app.api -dir "$tmpdir"
cp "$tmpdir/internal/types/types.go" ./internal/types/types.go
```

Expected: `backend/app/internal/types/types.go` contains the new `AdminVIP*` DTOs. Do not copy generated handlers or logic from the temp directory.

- [ ] **Step 5: Run test to verify it passes**

Run:

```bash
cd backend
node --test scripts/api_contract.test.mjs
goctl api validate --api app/api/app.api
```

Expected: PASS.

## Task 2: 后台 VIP 配置 logic

**Files:**
- Create: `backend/app/internal/logic/admin/vip_config_logic_test.go`
- Create: `backend/app/internal/logic/admin/vip_config_logic.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/app/internal/logic/admin/vip_config_logic_test.go`:

```go
package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSaveVIPPlanRejectsInvalidTopVoucherDuration(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveVIPPlan(context.Background(), "", SaveVIPPlanConfigReq{
		Code:              "monthly",
		Name:              "VIP 月卡",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "active",
		Benefits: VIPBenefitConfig{
			PublishQuota:    80,
			RefreshQuota:    30,
			TopVoucherCount: 3,
		},
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveVIPPlan() error = %v, want validation error", err)
	}
}

func TestSaveVIPPlanPassesNormalizedInputToStore(t *testing.T) {
	store := &fakeVIPConfigAdminStore{saved: model.AdminVIPConfigSaveResult{Code: "monthly", UpdatedAt: "2026-07-10T10:00:00Z"}}
	logic := NewVIPConfigAdminLogic(store)

	resp, err := logic.SaveVIPPlan(context.Background(), " monthly ", SaveVIPPlanConfigReq{
		Name:              " VIP 月卡 ",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "",
		DisplayOrder:      10,
		Benefits: VIPBenefitConfig{
			PublishQuota:       80,
			RefreshQuota:       30,
			TopVoucherCount:    3,
			TopDurationHours:   24,
			HomepageImageLimit: 18,
		},
	}, " admin-1 ")
	if err != nil {
		t.Fatalf("SaveVIPPlan() error = %v", err)
	}
	if resp.Code != "monthly" || store.planInput.Code != "monthly" || store.planInput.Name != "VIP 月卡" || store.planInput.OperatorID != "admin-1" {
		t.Fatalf("resp = %#v input = %#v, want normalized values", resp, store.planInput)
	}
	if store.planInput.Status != "active" || store.planInput.Benefits.PublishPolicy != model.VIPPublishPolicyQuota {
		t.Fatalf("input = %#v, want default active quota policy", store.planInput)
	}
}

func TestSaveQuotaPackRejectsSalePriceAboveStandardPrice(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveQuotaPack(context.Background(), "", SaveQuotaPackConfigReq{
		Code:              "publish_5",
		Name:              "发布次数包",
		StandardPriceCent: 2500,
		SalePriceCent:     3000,
		Status:            "active",
		Benefits:          VIPBenefitConfig{PublishQuota: 5},
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveQuotaPack() error = %v, want validation error", err)
	}
}

func TestSaveVIPPromotionRejectsInvalidPeriod(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveVIPPromotion(context.Background(), "", SaveVIPPromotionConfigReq{
		Code:          "launch_monthly",
		PlanCode:      "monthly",
		PromotionType: "launch",
		SalePriceCent: 1990,
		StartsAt:      "2026-07-11T00:00:00Z",
		EndsAt:        "2026-07-10T00:00:00Z",
		Status:        "active",
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveVIPPromotion() error = %v, want validation error", err)
	}
}

type fakeVIPConfigAdminStore struct {
	planInput      model.SaveAdminVIPPlanInput
	quotaPackInput model.SaveAdminQuotaPackInput
	promotionInput model.SaveAdminVIPPromotionInput
	saved          model.AdminVIPConfigSaveResult
}

func (s *fakeVIPConfigAdminStore) ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error) {
	return []model.AdminVIPPlanConfig{{Code: "monthly", Name: "VIP 月卡", DurationMonths: 1, Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error) {
	s.planInput = input
	return s.saved, nil
}

func (s *fakeVIPConfigAdminStore) ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error) {
	return []model.AdminQuotaPackConfig{{Code: "publish_5", Name: "发布次数包", Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error) {
	s.quotaPackInput = input
	return s.saved, nil
}

func (s *fakeVIPConfigAdminStore) ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error) {
	return []model.AdminVIPPromotionConfig{{Code: "launch_monthly", PlanCode: "monthly", PromotionType: "launch", Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error) {
	s.promotionInput = input
	return s.saved, nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./app/internal/logic/admin -run 'TestSaveVIP' -count=1`

Expected: FAIL because `NewVIPConfigAdminLogic`, request structs, and model input structs are undefined.

- [ ] **Step 3: Implement logic and model DTO types**

Create `backend/app/internal/logic/admin/vip_config_logic.go` with:

```go
package admin

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type VIPConfigAdminStore interface {
	ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error)
	SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error)
	ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error)
	SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error)
	ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error)
	SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error)
}

type VIPConfigAdminLogic struct {
	store VIPConfigAdminStore
}

func NewVIPConfigAdminLogic(store VIPConfigAdminStore) *VIPConfigAdminLogic {
	return &VIPConfigAdminLogic{store: store}
}

type VIPBenefitConfig struct {
	PublishPolicy      string `json:"publishPolicy,omitempty"`
	PublishQuota       int64  `json:"publishQuota"`
	RefreshQuota       int64  `json:"refreshQuota"`
	TopVoucherCount    int64  `json:"topVoucherCount"`
	TopDurationHours   int64  `json:"topDurationHours"`
	HomepageImageLimit int64  `json:"homepageImageLimit,omitempty"`
}

type SaveVIPPlanConfigReq struct {
	Code              string           `json:"code,omitempty"`
	Name              string           `json:"name"`
	DurationMonths    int64            `json:"durationMonths"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	Status            string           `json:"status,omitempty"`
	DisplayOrder      int64            `json:"displayOrder,omitempty"`
	Benefits          VIPBenefitConfig `json:"benefits"`
}

type SaveQuotaPackConfigReq struct {
	Code              string           `json:"code,omitempty"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	SalePriceCent     int64            `json:"salePriceCent,omitempty"`
	SaleLabel         string           `json:"saleLabel,omitempty"`
	Status            string           `json:"status,omitempty"`
	DisplayOrder      int64            `json:"displayOrder,omitempty"`
	Benefits          VIPBenefitConfig `json:"benefits"`
}

type SaveVIPPromotionConfigReq struct {
	Code          string `json:"code,omitempty"`
	PlanCode      string `json:"planCode"`
	PromotionType string `json:"promotionType"`
	SalePriceCent int64  `json:"salePriceCent"`
	StartsAt      string `json:"startsAt,omitempty"`
	EndsAt        string `json:"endsAt,omitempty"`
	QuotaLimit    int64  `json:"quotaLimit,omitempty"`
	Status        string `json:"status,omitempty"`
}

type SaveVIPConfigResp struct {
	Code      string `json:"code"`
	UpdatedAt string `json:"updatedAt"`
}

func (l *VIPConfigAdminLogic) SaveVIPPlan(ctx context.Context, pathCode string, req SaveVIPPlanConfigReq, operatorID string) (SaveVIPConfigResp, error) {
	input := model.SaveAdminVIPPlanInput{
		Code:              configCode(pathCode, req.Code),
		Name:              strings.TrimSpace(req.Name),
		DurationMonths:    req.DurationMonths,
		StandardPriceCent: req.StandardPriceCent,
		Status:            normalizeActiveInactive(req.Status),
		DisplayOrder:      req.DisplayOrder,
		Benefits:          mapAdminVIPBenefits(req.Benefits),
		OperatorID:        strings.TrimSpace(operatorID),
	}
	if err := validateVIPPlanInput(input); err != nil {
		return SaveVIPConfigResp{}, err
	}
	result, err := l.store.SaveAdminVIPPlan(ctx, input)
	if err != nil {
		logx.Errorf("保存 VIP 套餐配置失败: operatorId=%s planCode=%s err=%+v", input.OperatorID, input.Code, err)
		return SaveVIPConfigResp{}, errx.New(errx.CodeInternal, "保存失败，请稍后重试")
	}
	return SaveVIPConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}
```

In the same file, implement these exact public methods and helpers:

- `ListVIPPlans(ctx context.Context) (ListVIPPlansResp, error)`
- `SaveQuotaPack(ctx context.Context, pathCode string, req SaveQuotaPackConfigReq, operatorID string) (SaveVIPConfigResp, error)`
- `ListQuotaPacks(ctx context.Context) (ListQuotaPacksResp, error)`
- `SaveVIPPromotion(ctx context.Context, pathCode string, req SaveVIPPromotionConfigReq, operatorID string) (SaveVIPConfigResp, error)`
- `ListVIPPromotions(ctx context.Context) (ListVIPPromotionsResp, error)`
- `configCode(pathCode string, bodyCode string) string`
- `normalizeActiveInactive(status string) string`
- `mapAdminVIPBenefits(req VIPBenefitConfig) model.VIPBenefitSnapshot`
- `validateVIPPlanInput(input model.SaveAdminVIPPlanInput) error`
- `validateQuotaPackInput(input model.SaveAdminQuotaPackInput) error`
- `validateVIPPromotionInput(input model.SaveAdminVIPPromotionInput) error`
- `validatePromotionPeriod(startsAt string, endsAt string) error`

Validators must return the Chinese errors from the design document and must not return raw database errors to the frontend.

Add these model types to `backend/app/internal/model/vip_model.go` near existing VIP structs:

```go
type AdminVIPPlanConfig struct {
	Code              string
	Name              string
	DurationMonths    int64
	StandardPriceCent int64
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	UpdatedAt         string
}

type SaveAdminVIPPlanInput struct {
	Code              string
	Name              string
	DurationMonths    int64
	StandardPriceCent int64
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	OperatorID        string
}

type AdminQuotaPackConfig struct {
	Code              string
	Name              string
	Description       string
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	UpdatedAt         string
}

type SaveAdminQuotaPackInput struct {
	Code              string
	Name              string
	Description       string
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	OperatorID        string
}

type AdminVIPPromotionConfig struct {
	Code          string
	PlanCode      string
	PlanName      string
	PromotionType string
	SalePriceCent int64
	StartsAt      string
	EndsAt        string
	QuotaLimit    int64
	UsedCount     int64
	Status        string
	UpdatedAt     string
}

type SaveAdminVIPPromotionInput struct {
	Code          string
	PlanCode      string
	PromotionType string
	SalePriceCent int64
	StartsAt      string
	EndsAt        string
	QuotaLimit    int64
	Status        string
	OperatorID    string
}

type AdminVIPConfigSaveResult struct {
	Code      string
	UpdatedAt string
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./app/internal/logic/admin -run 'TestSaveVIP|TestSaveQuota' -count=1`

Expected: PASS.

## Task 3: VIP 模型后台持久化

**Files:**
- Modify: `backend/app/internal/model/vip_model.go`
- Test: `backend/app/internal/model/vip_model_test.go`

- [ ] **Step 1: Write failing model-focused tests**

Append to `backend/app/internal/model/vip_model_test.go`:

```go
func TestVIPAdminConfigSaveInputsPreserveBenefitSnapshot(t *testing.T) {
	input := SaveAdminVIPPlanInput{
		Code:              "monthly",
		Name:              "VIP 月卡",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "active",
		DisplayOrder:      10,
		Benefits: VIPBenefitSnapshot{
			PublishPolicy:      VIPPublishPolicyQuota,
			PublishQuota:       80,
			RefreshQuota:       30,
			TopVoucherCount:    3,
			TopDurationHours:   24,
			HomepageImageLimit: 18,
		},
		OperatorID: "admin-1",
	}

	values := input.Benefits.ToJSONMap()
	if values["publishQuota"] != int64(80) || values["refreshQuota"] != int64(30) || values["topVoucherCount"] != int64(3) {
		t.Fatalf("benefits json = %#v, want quota snapshot values", values)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./app/internal/model -run TestVIPAdminConfigSaveInputsPreserveBenefitSnapshot -count=1`

Expected: FAIL because `SaveAdminVIPPlanInput` is undefined until Task 2 model types are added.

- [ ] **Step 3: Implement list/save methods**

Add methods to `backend/app/internal/model/vip_model.go`:

```go
func (m *VIPModel) ListAdminVIPPlans(ctx context.Context) ([]AdminVIPPlanConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  p.code,
  p.name,
  p.duration_months,
  p.standard_price_cent,
  p.status,
  p.display_order,
  COALESCE(v.benefits, '{}'::jsonb) AS benefits,
  p.updated_at
FROM vip_plans p
LEFT JOIN LATERAL (
  SELECT benefits
  FROM vip_plan_versions
  WHERE plan_id = p.id
  ORDER BY version DESC
  LIMIT 1
) v ON true
ORDER BY p.display_order ASC, p.duration_months ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminVIPPlanConfig, 0)
	for rows.Next() {
		var item AdminVIPPlanConfig
		var benefits JSONMap
		var updatedAt time.Time
		if err := rows.Scan(&item.Code, &item.Name, &item.DurationMonths, &item.StandardPriceCent, &item.Status, &item.DisplayOrder, &benefits, &updatedAt); err != nil {
			return nil, err
		}
		item.Benefits = VIPBenefitSnapshotFromJSON(benefits)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}
```

Implement:

- `SaveAdminVIPPlan(ctx, input)` in a transaction:
  - Query previous snapshot for operation log.
  - Upsert `vip_plans`.
  - Insert `vip_plan_versions` with `COALESCE(MAX(version), 0) + 1`.
  - Write `recordOperationLogTx` with `ObjectID: ""` because `operation_logs.object_id` is bigint; include `code` in snapshots.
- `ListAdminQuotaPacks(ctx)`
- `SaveAdminQuotaPack(ctx, input)`
- `ListAdminVIPPromotions(ctx)`
- `SaveAdminVIPPromotion(ctx, input)`

For promotion plan lookup, use `SELECT id FROM vip_plans WHERE code = $1 LIMIT 1`; if no row, return `sql.ErrNoRows` so logic can map to `请选择有效的 VIP 套餐`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./app/internal/model -run 'TestVIPAdminConfig|TestVIPBenefit' -count=1`

Expected: PASS.

## Task 4: 后台 VIP 配置路由

**Files:**
- Modify: `backend/app/internal/server/domain_routes.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`

- [ ] **Step 1: Write failing route tests**

In `backend/app/internal/server/remaining_api_test.go`, add:

```go
func TestAPIRouterServesAdminVIPConfigRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}}}))

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/vip/plans", nil)
	listReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(listRec, listReq)
	data := decodeEnvelopeData(t, listRec, http.StatusOK)
	items, ok := data["items"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("items = %#v, want vip plan items", data["items"])
	}

	saveRec := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/vip/plans/monthly", strings.NewReader(`{
		"name":"VIP 月卡",
		"durationMonths":1,
		"standardPriceCent":4900,
		"status":"active",
		"displayOrder":10,
		"operatorId":"attacker",
		"benefits":{"publishQuota":80,"refreshQuota":30,"topVoucherCount":3,"topDurationHours":24}
	}`))
	saveReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(saveRec, saveReq)
	decodeEnvelopeData(t, saveRec, http.StatusOK)
	if store.saveVIPPlanInput.OperatorID != "admin-1" {
		t.Fatalf("operatorID = %q, want token admin user", store.saveVIPPlanInput.OperatorID)
	}
}
```

Add fields and fake methods:

```go
saveVIPPlanInput      model.SaveAdminVIPPlanInput
saveQuotaPackInput    model.SaveAdminQuotaPackInput
saveVIPPromotionInput model.SaveAdminVIPPromotionInput
```

```go
func (s *fakeFullAPIStore) ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error) {
	return []model.AdminVIPPlanConfig{{Code: "monthly", Name: "VIP 月卡", DurationMonths: 1, StandardPriceCent: 4900, Status: "active", Benefits: model.VIPBenefitSnapshot{PublishQuota: 80, RefreshQuota: 30}}}, nil
}

func (s *fakeFullAPIStore) SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveVIPPlanInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}
```

Add these fake methods for quota packs and promotions:

```go
func (s *fakeFullAPIStore) ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error) {
	return []model.AdminQuotaPackConfig{{Code: "publish_5", Name: "发布次数包", StandardPriceCent: 2500, Status: "active", Benefits: model.VIPBenefitSnapshot{PublishQuota: 5}}}, nil
}

func (s *fakeFullAPIStore) SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveQuotaPackInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error) {
	return []model.AdminVIPPromotionConfig{{Code: "launch_monthly", PlanCode: "monthly", PromotionType: "launch", SalePriceCent: 1990, Status: "active"}}, nil
}

func (s *fakeFullAPIStore) SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveVIPPromotionInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./app/internal/server -run TestAPIRouterServesAdminVIPConfigRoutes -count=1`

Expected: FAIL with 404 or missing methods because routes are not registered.

- [ ] **Step 3: Register routes**

In `backend/app/internal/server/domain_routes.go`:

```go
type VIPAPIStore interface {
	viplogic.Store
	paymentlogic.VIPPaymentStore
	adminlogic.VIPConfigAdminStore
}
```

Add to `registerVIPRoutes`:

```go
mux.HandleFunc("GET /api/v1/admin/vip/plans", func(w http.ResponseWriter, r *http.Request) {
	resp, err := adminlogic.NewVIPConfigAdminLogic(store).ListVIPPlans(r.Context())
	response.JSON(w, resp, err)
})
mux.HandleFunc("POST /api/v1/admin/vip/plans", func(w http.ResponseWriter, r *http.Request) {
	var body adminlogic.SaveVIPPlanConfigReq
	if err := decodeJSONBody(r, &body); err != nil {
		response.JSON(w, nil, err)
		return
	}
	operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
	if err != nil {
		response.JSON(w, nil, err)
		return
	}
	resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPlan(r.Context(), "", body, operatorID)
	response.JSON(w, resp, err)
})
```

Register the matching update/list/create routes for quota packs and promotions. Use `r.PathValue("planCode")`, `r.PathValue("packCode")`, and `r.PathValue("promotionCode")` for updates.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./app/internal/server -run TestAPIRouterServesAdminVIPConfigRoutes -count=1`

Expected: PASS.

## Task 5: 管理后台入口和 API 封装

**Files:**
- Modify: `admin-web/scripts/feature-visibility.test.mjs`
- Create: `admin-web/src/api/vipConfig.js`
- Modify: `admin-web/src/router/index.js`
- Modify: `admin-web/src/layouts/AdminLayout.vue`

- [ ] **Step 1: Write failing frontend static test**

Add to `admin-web/scripts/feature-visibility.test.mjs`:

```js
test('admin ui exposes vip config management entry', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/vipConfig.js'), 'utf8')

  assert.match(routeSource, /const VIPConfigView = \(\) => import\('\.\.\/views\/VIPConfigView\.vue'\)/)
  assert.match(routeSource, /path: 'vip-configs'/)
  assert.match(layoutSource, /index="\/vip-configs"/)
  assert.match(layoutSource, /<span>VIP 配置<\/span>/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/plans/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/quota-packs/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/promotions/)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd admin-web && node --test scripts/feature-visibility.test.mjs`

Expected: FAIL because `vipConfig.js`, route, and menu do not exist.

- [ ] **Step 3: Add API file**

Create `admin-web/src/api/vipConfig.js`:

```js
import http from './http'

export function listVIPPlanConfigs() {
  return http.get('/api/v1/admin/vip/plans')
}

export function createVIPPlanConfig(payload) {
  return http.post('/api/v1/admin/vip/plans', payload)
}

export function updateVIPPlanConfig(planCode, payload) {
  return http.post(`/api/v1/admin/vip/plans/${planCode}`, payload)
}

export function listQuotaPackConfigs() {
  return http.get('/api/v1/admin/vip/quota-packs')
}

export function createQuotaPackConfig(payload) {
  return http.post('/api/v1/admin/vip/quota-packs', payload)
}

export function updateQuotaPackConfig(packCode, payload) {
  return http.post(`/api/v1/admin/vip/quota-packs/${packCode}`, payload)
}

export function listVIPPromotionConfigs() {
  return http.get('/api/v1/admin/vip/promotions')
}

export function createVIPPromotionConfig(payload) {
  return http.post('/api/v1/admin/vip/promotions', payload)
}

export function updateVIPPromotionConfig(promotionCode, payload) {
  return http.post(`/api/v1/admin/vip/promotions/${promotionCode}`, payload)
}
```

- [ ] **Step 4: Add route and menu**

In `admin-web/src/router/index.js`:

```js
const VIPConfigView = () => import('../views/VIPConfigView.vue')
```

Add child route:

```js
{ path: 'vip-configs', name: 'vipConfigs', component: VIPConfigView },
```

In `admin-web/src/layouts/AdminLayout.vue`, import `Medal` from `@element-plus/icons-vue` and add:

```vue
<el-menu-item index="/vip-configs">
  <el-icon><Medal /></el-icon>
  <span>VIP 配置</span>
</el-menu-item>
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd admin-web && node --test scripts/feature-visibility.test.mjs`

Expected: PASS.

## Task 6: VIPConfigView 页面

**Files:**
- Modify: `admin-web/scripts/feature-visibility.test.mjs`
- Create: `admin-web/src/views/VIPConfigView.vue`

- [ ] **Step 1: Write failing page structure test**

Add to `admin-web/scripts/feature-visibility.test.mjs`:

```js
test('vip config page provides plan quota pack and promotion tabs', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/VIPConfigView.vue'), 'utf8')

  assert.match(source, /<h2>VIP 配置<\/h2>/)
  assert.match(source, /<el-tab-pane label="VIP 套餐" name="plans">/)
  assert.match(source, /<el-tab-pane label="次数包" name="quotaPacks">/)
  assert.match(source, /<el-tab-pane label="优惠活动" name="promotions">/)
  assert.match(source, /openPlanEditor/)
  assert.match(source, /openQuotaPackEditor/)
  assert.match(source, /openPromotionEditor/)
  assert.match(source, /yuanToCent/)
  assert.match(source, /centToYuan/)
  assert.match(source, /buildPlanPayload/)
  assert.match(source, /buildQuotaPackPayload/)
  assert.match(source, /buildPromotionPayload/)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd admin-web && node --test scripts/feature-visibility.test.mjs`

Expected: FAIL because `VIPConfigView.vue` does not exist or lacks required tabs.

- [ ] **Step 3: Create page**

Create `admin-web/src/views/VIPConfigView.vue` with:

```vue
<template>
  <section>
    <div class="page-title">
      <h2>VIP 配置</h2>
      <el-button :loading="loading" plain @click="loadAll">刷新</el-button>
    </div>

    <section class="panel">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="VIP 套餐" name="plans">
          <div class="table-toolbar">
            <el-button type="primary" @click="openPlanEditor()">新增套餐</el-button>
          </div>
          <el-table v-loading="loading" :data="plans" stripe empty-text="暂无 VIP 套餐">
            <el-table-column prop="name" label="套餐" min-width="140" />
            <el-table-column prop="code" label="编码" width="130" />
            <el-table-column label="价格" width="120">
              <template #default="{ row }">{{ formatMoney(row.standardPriceCent) }}</template>
            </el-table-column>
            <el-table-column label="权益" min-width="220">
              <template #default="{ row }">{{ benefitSummary(row.benefits) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag></template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }"><el-button type="primary" link @click="openPlanEditor(row)">编辑</el-button></template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="次数包" name="quotaPacks">
          <div class="table-toolbar">
            <el-button type="primary" @click="openQuotaPackEditor()">新增次数包</el-button>
          </div>
          <el-table v-loading="loading" :data="quotaPacks" stripe empty-text="暂无次数包">
            <el-table-column prop="name" label="次数包" min-width="140" />
            <el-table-column prop="code" label="编码" width="130" />
            <el-table-column prop="description" label="描述" min-width="160" />
            <el-table-column label="价格" width="150">
              <template #default="{ row }">{{ formatMoney(row.salePriceCent || row.standardPriceCent) }}</template>
            </el-table-column>
            <el-table-column label="权益" min-width="220">
              <template #default="{ row }">{{ benefitSummary(row.benefits) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }"><el-button type="primary" link @click="openQuotaPackEditor(row)">编辑</el-button></template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="优惠活动" name="promotions">
          <div class="table-toolbar">
            <el-button type="primary" @click="openPromotionEditor()">新增优惠</el-button>
          </div>
          <el-table v-loading="loading" :data="promotions" stripe empty-text="暂无优惠活动">
            <el-table-column prop="code" label="编码" min-width="150" />
            <el-table-column prop="planCode" label="套餐" width="120" />
            <el-table-column prop="promotionType" label="类型" width="130" />
            <el-table-column label="优惠价" width="120">
              <template #default="{ row }">{{ formatMoney(row.salePriceCent) }}</template>
            </el-table-column>
            <el-table-column label="时间" min-width="220">
              <template #default="{ row }">{{ row.startsAt || '-' }} / {{ row.endsAt || '长期' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }"><el-button type="primary" link @click="openPromotionEditor(row)">编辑</el-button></template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </section>
  </section>
</template>
```

Add script setup that imports API methods, loads all three lists on mount, provides editor drawers/forms, and implements these functions:

```js
function centToYuan(value) {
  return Number(value || 0) / 100
}

function yuanToCent(value) {
  return Math.round(Number(value || 0) * 100)
}

function buildPlanPayload() {
  return {
    code: planForm.code.trim(),
    name: planForm.name.trim(),
    durationMonths: Number(planForm.durationMonths || 0),
    standardPriceCent: yuanToCent(planForm.standardPriceYuan),
    status: planForm.status,
    displayOrder: Number(planForm.displayOrder || 0),
    benefits: buildBenefitsPayload(planForm.benefits),
  }
}
```

Implement these page helpers in `VIPConfigView.vue`:

- `buildQuotaPackPayload()` returns `code`、`name`、`description`、`standardPriceCent`、`salePriceCent`、`saleLabel`、`status`、`displayOrder`、`benefits`。
- `buildPromotionPayload()` returns `code`、`planCode`、`promotionType`、`salePriceCent`、`startsAt`、`endsAt`、`quotaLimit`、`status`。
- `openPlanEditor(row)` fills `planForm` from a row or clears it for create mode.
- `openQuotaPackEditor(row)` fills `quotaPackForm` from a row or clears it for create mode.
- `openPromotionEditor(row)` fills `promotionForm` from a row or clears it for create mode.
- `savePlan()` chooses `createVIPPlanConfig` or `updateVIPPlanConfig` based on edit mode.
- `saveQuotaPack()` chooses `createQuotaPackConfig` or `updateQuotaPackConfig` based on edit mode.
- `savePromotion()` chooses `createVIPPromotionConfig` or `updateVIPPromotionConfig` based on edit mode.

Use Element Plus form controls in drawers. Keep all UI text short and operational.

- [ ] **Step 4: Run frontend static test**

Run: `cd admin-web && node --test scripts/feature-visibility.test.mjs`

Expected: PASS.

## Task 7: Verification and formatting

**Files:**
- All touched files

- [ ] **Step 1: Format Go files**

Run:

```bash
cd backend
gofmt -w app/internal/logic/admin/vip_config_logic.go app/internal/logic/admin/vip_config_logic_test.go app/internal/model/vip_model.go app/internal/model/vip_model_test.go app/internal/server/domain_routes.go app/internal/server/remaining_api_test.go
```

Expected: command exits 0.

- [ ] **Step 2: Run backend targeted tests**

Run:

```bash
cd backend
go test ./app/internal/logic/admin ./app/internal/model ./app/internal/server
```

Expected: PASS.

- [ ] **Step 3: Run API contract tests**

Run:

```bash
cd backend
node --test scripts/api_contract.test.mjs
goctl api validate --api app/api/app.api
```

Expected: PASS.

- [ ] **Step 4: Run admin web tests and build**

Run:

```bash
cd admin-web
node --test scripts/feature-visibility.test.mjs
npm run build
```

Expected: PASS and Vite build exits 0.

- [ ] **Step 5: Inspect diff**

Run:

```bash
git status --short
git diff --stat
git diff --check
```

Expected: only VIP admin config implementation files changed, and `git diff --check` exits 0.

- [ ] **Step 6: Commit implementation**

Run:

```bash
git add backend/app/api/admin.api backend/app/internal/types/types.go backend/app/internal/logic/admin/vip_config_logic.go backend/app/internal/logic/admin/vip_config_logic_test.go backend/app/internal/model/vip_model.go backend/app/internal/model/vip_model_test.go backend/app/internal/server/domain_routes.go backend/app/internal/server/remaining_api_test.go backend/scripts/api_contract.test.mjs admin-web/src/api/vipConfig.js admin-web/src/views/VIPConfigView.vue admin-web/src/router/index.js admin-web/src/layouts/AdminLayout.vue admin-web/scripts/feature-visibility.test.mjs admin-web/src/plugins/elementPlus.js docs/superpowers/plans/2026-07-10-vip-admin-config-implementation.md
git commit -m "feat: add vip admin config"
```

Expected: commit succeeds after all verification commands pass.
