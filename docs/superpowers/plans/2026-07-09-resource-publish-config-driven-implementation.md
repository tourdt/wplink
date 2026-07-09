# 发布字段配置化重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将发布资源/发布需求从固定“品类、数量/产能、价格/采购要求”表单改为按资源类型配置渲染，并用配置生成列表和详情所需的摘要字段。

**Architecture:** 后端继续保留 `resources.category`、`quantity_text`、`price_text` 作为检索和卡片摘要字段，但它们不再要求用户在所有类型上直接填写，而是从 `resource_type_configs.display_template.summary` 指定的动态字段派生。wxapp 发布页只固定渲染类型、标题、描述、图片、联系信息，业务字段全部来自 `field_schema.fields` 和 `required_fields`。

**Tech Stack:** Go/go-zero 业务逻辑、PostgreSQL JSONB 迁移种子、Vue 3 + uni-app 小程序、Node.js `node:test` 静态流校验。

---

### Task 1: 前端发布页改为配置驱动字段

**Files:**
- Modify: `wxapp/components/ResourcePublishForm.vue`
- Test: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write the failing test**

在 `validate-flows.test.mjs` 中新增断言，要求发布表单不再固定渲染品类、数量/产能、价格/采购要求，并要求提交 payload 通过 `displayTemplate.summary` 生成摘要字段。

```js
test('publish form renders type fields from schema instead of fixed category purchase fields', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.equal(source.includes('<text class="field-label">品类</text>'), false)
  assert.equal(source.includes("detailTitle: '采购要求'"), false)
  assert.equal(source.includes("quantityLabel: '需求数量'"), false)
  assert.equal(source.includes("priceLabel: '预算描述'"), false)
  assert.match(source, /buildResourcePublishPayload/)
  assert.match(source, /applySummaryFieldsToPayload/)
  assert.match(source, /displayTemplate\?\.summary/)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL，因为当前模板仍固定包含品类、需求数量、预算描述，且没有摘要 payload 构建函数。

- [ ] **Step 3: Write minimal implementation**

将 `ResourcePublishForm.vue` 的固定字段区改为只展示描述，删除 `category/quantityText/priceText` 输入；`requiredFields` 默认项去掉 `category`；新增 `buildResourcePublishPayload(images)` 和 `applySummaryFieldsToPayload(payload, resourceType)`，按 `displayTemplate.summary` 从 `attributes` 或基础字段取值后写入摘要字段。

```js
function buildResourcePublishPayload(images) {
  const payload = {
    ...clonePublishForm(),
    images,
  }
  applySummaryFieldsToPayload(payload, currentResourceType.value)
  return payload
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS。

### Task 2: 后端按 summary 派生摘要字段

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/logic/resource/create_resource_logic.go`
- Test: `backend/app/internal/logic/resource/create_resource_logic_test.go`

- [ ] **Step 1: Write the failing test**

新增测试：请求只提交动态字段，配置用 `display_template.summary` 映射到 `category/quantityText/priceText`，最终 `CreateResourceInput` 写入摘要字段。

```go
func TestCreateResourceDerivesSummaryFieldsFromDisplayTemplate(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-find-rental",
			TypeCode:       "find_rental",
			Direction:      model.ResourceDirectionDemand,
			RequiredFields: []string{"title", "rentalNeedType", "contactPhone"},
			FieldSchema: model.JSONMap{"fields": []interface{}{
				map[string]interface{}{"key": "rentalNeedType", "label": "场地类型", "type": "select", "options": []interface{}{"档口", "仓库"}},
				map[string]interface{}{"key": "expectedAreaText", "label": "面积需求", "type": "text"},
				map[string]interface{}{"key": "budgetRentText", "label": "预算租金", "type": "text"},
			}},
			DisplayTemplate: model.JSONMap{"summary": map[string]interface{}{
				"category":     "rentalNeedType",
				"quantityText": "expectedAreaText",
				"priceText":    "budgetRentText",
			}},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "find_rental",
		Title:       "急找童装城附近档口",
		Attributes:  model.JSONMap{"rentalNeedType": "档口", "expectedAreaText": "80-120 平", "budgetRentText": "8000 元/月以内"},
		Contact:     ResourceContactReq{Name: "李老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if store.input.Category != "档口" || store.input.QuantityText != "80-120 平" || store.input.PriceText != "8000 元/月以内" {
		t.Fatalf("input summary = %#v, want derived from dynamic attributes", store.input)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend/app/internal/logic/resource`

Expected: FAIL，因为 `ResourcePublishConfig` 还没有读取 `DisplayTemplate`，创建逻辑也不会派生摘要。

- [ ] **Step 3: Write minimal implementation**

在 `ResourcePublishConfig` 增加 `DisplayTemplate model.JSONMap`，查询 `display_template`；在 `buildResourceInput` 通过 `deriveResourceSummaryFields` 合并用户直接提交值和配置映射值，`category` 为空时用 `待沟通` 兜底以满足数据库非空约束。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend/app/internal/logic/resource`

Expected: PASS。

### Task 3: 管理端配置校验支持 summary 映射

**Files:**
- Modify: `backend/app/internal/logic/admin/resource_type_config_logic.go`
- Test: `backend/app/internal/logic/admin/resource_type_config_logic_test.go`

- [ ] **Step 1: Write the failing test**

新增测试：`displayTemplate.summary` 只允许目标字段为 `category/quantityText/priceText`，映射值必须引用基础字段或动态字段。

```go
func TestUpdateResourceTypeConfigRejectsInvalidSummaryTarget(t *testing.T) {
	store := &fakeResourceTypeConfigStore{}
	logic := NewResourceTypeConfigLogic(store)
	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		FieldSchema: map[string]interface{}{"fields": []interface{}{map[string]interface{}{"key": "serviceType", "label": "服务类型", "type": "text"}}},
		RequiredFields: []string{"title", "serviceType", "contactPhone"},
		DisplayTemplate: map[string]interface{}{"summary": map[string]interface{}{"unknown": "serviceType"}},
		DefaultValidDays: 30,
		Status: "active",
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend/app/internal/logic/admin`

Expected: FAIL，因为现有校验忽略 `summary`。

- [ ] **Step 3: Write minimal implementation**

新增 `validateDisplaySummaryTemplate`，在 `validateResourceTypeConfigPatch` 中校验 summary 目标字段和值引用。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend/app/internal/logic/admin`

Expected: PASS。

### Task 4: 种子配置补齐每个类型的 summary

**Files:**
- Modify: `backend/migrations/000003_seed_zhili.up.sql`
- Modify: `backend/migrations/000014_resource_type_display_names.up.sql`
- Modify: `backend/migrations/000015_find_rental_resource_type.up.sql`
- Test: `backend/scripts/validate_migrations.test.mjs`

- [ ] **Step 1: Write the failing test**

在迁移校验中要求所有供给和需求类型的 `display_template` 包含 `summary`，并且 `required_fields` 不再把 `category/quantityText/priceText` 作为跨类型默认必填。

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test backend/scripts/validate_migrations.test.mjs`

Expected: FAIL，因为现有种子配置缺少 summary，且仍有多处固定必填摘要字段。

- [ ] **Step 3: Write minimal implementation**

更新各类型 `display_template`：

```json
{
  "summary": {
    "category": "serviceType",
    "quantityText": "serviceArea",
    "priceText": "leadTime"
  }
}
```

每个类型按产品文档选择对应动态字段；只保留用户真正需要填写的动态字段和联系字段为必填。

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test backend/scripts/validate_migrations.test.mjs`

Expected: PASS。

### Task 5: 列表和详情展示去掉固定品类/数量/价格兜底

**Files:**
- Modify: `wxapp/components/ResourceCard.vue`
- Modify: `wxapp/components/DemandCard.vue`
- Modify: `wxapp/pages/resource/detail.vue`
- Test: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write the failing test**

新增断言：卡片不再输出“品类待沟通/数量待沟通/预算面议”的固定跨类型兜底，详情不再固定显示品类、数量、价格三项。

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL。

- [ ] **Step 3: Write minimal implementation**

卡片通过 `resourceSummaryText` 拼接已有摘要值，价格/预算为空时隐藏；详情优先展示后端 `attributeItems`，摘要字段只在没有与动态属性重复时作为补充。

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS。

### Task 6: Full verification

**Files:**
- No production edits

- [ ] **Step 1: Run backend logic tests**

Run: `go test ./backend/app/internal/logic/resource ./backend/app/internal/logic/admin`

Expected: PASS。

- [ ] **Step 2: Run migration tests**

Run: `node --test backend/scripts/validate_migrations.test.mjs`

Expected: PASS。

- [ ] **Step 3: Run wxapp flow validation**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS。

- [ ] **Step 4: Run existing wxapp validation scripts**

Run: `npm --prefix wxapp run validate:flows`

Expected: PASS。

Run: `npm --prefix wxapp run validate:pages`

Expected: PASS。
