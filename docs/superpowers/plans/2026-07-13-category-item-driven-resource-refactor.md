# 二级类目驱动供需发布重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将现有供需类型重构为“一级类目分组 + 二级类目配置”，由二级类目决定供需方向、发布字段、必填规则和列表摘要。

**Architecture:** 复用现有 `resource_type_configs` 作为唯一配置源，不新增并行类目表。`type_code/type_name` 表示二级类目，`direction` 表示该二级类目的供需方向，`display_template.group` 表示一级类目，`field_schema/required_fields/display_template.summary` 表示独立属性和摘要映射。前端首页、发布、搜索、市场、我的发布统一从该配置读取类目。

**Tech Stack:** Go/go-zero 风格后端逻辑、PostgreSQL JSONB 配置、uni-app/Vue 小程序、Node 静态验证测试、Go 单元测试。

---

### Task 1: 后端列表与搜索不再默认供应方向

**Files:**
- Modify: `backend/app/internal/logic/resource/list_resources_logic.go`
- Modify: `backend/app/internal/logic/resource/list_resources_logic_test.go`
- Modify: `backend/app/internal/logic/resource/search_resources_logic_test.go`

- [ ] **Step 1: Write the failing test**

在 `list_resources_logic_test.go` 增加：

```go
func TestListResourcesDoesNotDefaultToSupplyWhenDirectionOmitted(t *testing.T) {
	store := &fakeListResourcesStore{}
	logic := NewListResourcesLogic(store)

	_, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if store.filter.Direction != "" {
		t.Fatalf("direction = %q, want empty direction for category-driven mixed results", store.filter.Direction)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend/app/internal/logic/resource -run TestListResourcesDoesNotDefaultToSupplyWhenDirectionOmitted`
Expected: FAIL because omitted direction currently normalizes to `supply`.

- [ ] **Step 3: Write minimal implementation**

Change `normalizeListDirection` so blank input returns `""`; keep rejecting non-empty invalid values.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend/app/internal/logic/resource -run TestListResourcesDoesNotDefaultToSupplyWhenDirectionOmitted`
Expected: PASS.

### Task 2: 资源列表返回动态二级类目名称

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/logic/resource/list_resources_logic.go`
- Modify: `backend/app/internal/logic/resource/list_resources_logic_test.go`

- [ ] **Step 1: Write the failing test**

在 `TestListResourcesRequestsPublishedOnly` 的 fake item 中设置 `TypeName: "尾货/库存出售"`，断言响应 item 返回同名字段。

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend/app/internal/logic/resource -run TestListResourcesRequestsPublishedOnly`
Expected: FAIL because list response currently has no `typeName`.

- [ ] **Step 3: Write minimal implementation**

给 model 和 logic 的 `ResourceListItem` 增加 `TypeName`；`listResourcesSQL` join `resource_type_configs rtc` 并扫描 `rtc.type_name`。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend/app/internal/logic/resource -run TestListResourcesRequestsPublishedOnly`
Expected: PASS.

### Task 3: 迁移与演示数据改成二级类目配置

**Files:**
- Modify: `backend/migrations/000003_seed_zhili.up.sql`
- Modify: `backend/migrations/000014_resource_type_display_names.up.sql`
- Modify: `backend/migrations/000015_find_rental_resource_type.up.sql`
- Modify: `backend/scripts/seed_demo_data.sql`
- Modify: `backend/scripts/validate_migrations.test.mjs`

- [ ] **Step 1: Write the failing migration assertions**

更新 `validate_migrations.test.mjs`，断言种子迁移包含 `display_template.group`、8 个一级类目、核心二级类目 `stock_clearance`、`buy_kids_goods`、`job_hiring`、`seek_factory_warehouse`，并断言旧类型 `inventory/goods/find_rental` 不再作为默认类目配置插入。

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test backend/scripts/validate_migrations.test.mjs`
Expected: FAIL because迁移还使用旧供需类型。

- [ ] **Step 3: Rewrite seed configs**

将 `000003_seed_zhili.up.sql` 的 `resource_type_configs` VALUES 改为二级类目清单；每个配置写入 `direction`、`field_schema`、`required_fields`、`display_template.group`、`display_template.summary`、`display_template.list/detail`。将 `000014` 和 `000015` 改为不再新增旧类目，只保留必要方向 schema 防护。同步 `backend/scripts/seed_demo_data.sql` 的演示资源 `type_code` 和属性。

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test backend/scripts/validate_migrations.test.mjs`
Expected: PASS.

### Task 4: 前端增加资源类目工具

**Files:**
- Create: `wxapp/common/resourceCategories.js`
- Create: `wxapp/common/resourceCategories.test.mjs`

- [ ] **Step 1: Write the failing test**

测试 `groupResourceTypes` 能按 `displayTemplate.group.sort` 生成一级分组，二级类目保留 `direction`，缺失分组时进入“其他类目”。

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test wxapp/common/resourceCategories.test.mjs`
Expected: FAIL because helper does not exist.

- [ ] **Step 3: Implement helper**

实现 `normalizeResourceTypeOption`、`groupResourceTypes`、`flattenGroupedResourceTypes`、`resourceTypeLabel`，只依赖普通 JS。

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test wxapp/common/resourceCategories.test.mjs`
Expected: PASS.

### Task 5: 发布入口改为一级分组 + 二级类目

**Files:**
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/components/ResourcePublishForm.vue`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write failing static assertions**

将“发布 tab supports supply and demand entry selection”更新为“发布 tab uses category item entry selection”，断言入口加载 `listCityResourceTypes`、使用 `groupResourceTypes`、展示 `童装批发/厂房仓库/本地服务`，不再展示 `我能提供/我想寻找`。

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`
Expected: FAIL because发布入口仍按方向展示。

- [ ] **Step 3: Implement publish entry**

发布页加载全部二级类目，按一级分组渲染按钮；点击二级类目写入 `typeCode` 后进入编辑页。编辑表单不再按方向请求类型，选择二级类目后从配置同步 `form.direction`。

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`
Expected: PASS.

### Task 6: 搜索与市场改为类目筛选

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/logic/resource/list_resources_logic.go`
- Modify: `backend/app/internal/logic/resource/search_resources_logic.go`
- Modify: `backend/app/internal/server/api.go`
- Modify: `wxapp/pages/search/index.vue`
- Modify: `wxapp/pages/market/index.vue`
- Modify: `wxapp/pages/home/index.vue`

- [ ] **Step 1: Write failing tests**

新增 Go 测试断言 `GroupCode` 进入 `ListResourcesFilter`；更新前端静态测试断言搜索/市场没有方向 tab，而是有一级类目和二级类目筛选。

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./backend/app/internal/logic/resource -run GroupCode`
Run: `node --test wxapp/scripts/validate-flows.test.mjs`
Expected: FAIL.

- [ ] **Step 3: Implement filter**

后端 `ListResourcesFilter` 增加 `GroupCode`，SQL 通过 `rtc.display_template #>> '{group,code}'` 过滤；前端搜索/市场加载全部类型并用 helper 分组，列表按每条资源 `direction` 选择 `ResourceCard` 或 `DemandCard`。

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./backend/app/internal/logic/resource`
Run: `node --test wxapp/scripts/validate-flows.test.mjs`
Expected: PASS.

### Task 7: 我的发布和卡片展示使用动态类型名

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/logic/resource/my_resource_logic.go`
- Modify: `wxapp/components/ResourceCard.vue`
- Modify: `wxapp/components/DemandCard.vue`
- Modify: `wxapp/pages/my-resources/index.vue`
- Modify: `wxapp/common/enums.js`

- [ ] **Step 1: Write failing assertions**

更新卡片测试断言优先使用 `resource.typeName`；更新我的发布逻辑测试断言返回 `direction/typeName`。

- [ ] **Step 2: Run tests to verify they fail**

Run: `node --test wxapp/components/ResourceCard.test.mjs`
Run: `go test ./backend/app/internal/logic/resource -run MyResources`
Expected: FAIL.

- [ ] **Step 3: Implement dynamic labels**

后端我的发布列表返回 `direction/typeName`；前端卡片优先显示 `resource.typeName`，枚举只保留兜底。

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./backend/app/internal/logic/resource`
Run: `node --test wxapp/components/ResourceCard.test.mjs`
Expected: PASS.

### Task 8: Final verification

Run:

```bash
go test ./backend/app/internal/logic/resource ./backend/app/internal/logic/city ./backend/app/internal/server
node --test backend/scripts/validate_migrations.test.mjs
npm --prefix wxapp run validate:flows
npm --prefix wxapp run validate:pages
node --test wxapp/common/resourceCategories.test.mjs wxapp/components/ResourceCard.test.mjs
```

Expected: all selected checks pass, or any environment limitation is recorded with exact command output.
