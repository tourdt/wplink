# 供需页面移除方向筛选实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除供需首页和搜索页“全部 / 供应 / 需求”筛选栏，使两个页面固定展示供需混合流。

**Architecture:** 调整 `pages/market/index.vue` 和 `pages/search/index.vue` 的展示与查询状态，不修改接口协议。通过页面测试和流程校验约束两个页面不再持有方向筛选、请求不再传递 `direction`，同时确认供应卡片和需求卡片仍并存。

**Tech Stack:** Vue 3、uni-app、Node.js `node:test`

## Global Constraints

- 仅修改供需首页、搜索页及其直接相关测试和校验。
- 首页和搜索页省略 `direction` 参数，由现有服务端默认行为返回供需混合列表。
- 搜索页继续保留关键词、类目、标签和分页筛选能力。
- 不修改后端接口协议。

---

### Task 1: 移除供需首页方向筛选

**Files:**
- Modify: `wxapp/pages/market/index.test.mjs`
- Modify: `wxapp/scripts/validate-flows.test.mjs`
- Modify: `wxapp/scripts/validate-flows.mjs`
- Modify: `wxapp/pages/market/index.vue`

**Interfaces:**
- Consumes: `listResources({ cityCode, groupCode, typeCode, page, pageSize })`
- Produces: 无方向筛选状态的供需首页混合列表

- [x] **Step 1: 修改页面测试并验证失败**

在 `wxapp/pages/market/index.test.mjs` 中将方向筛选断言改为：

```js
assert.doesNotMatch(source, /direction-filter-row|directionFilterOptions|chooseResourceDirection/)
assert.doesNotMatch(source, /direction:\s*filters\.direction/)
assert.match(source, /DemandCard v-if="item\.direction === RESOURCE_DIRECTION_DEMAND"/)
assert.match(source, /<ResourceCard v-else/)
```

同时更新 `wxapp/scripts/validate-flows.test.mjs` 和 `wxapp/scripts/validate-flows.mjs`，使流程校验不再要求首页包含方向筛选。

运行：

```bash
cd wxapp && node --test pages/market/index.test.mjs scripts/validate-flows.test.mjs
```

预期：供需首页仍包含方向筛选实现，因此测试失败。

- [x] **Step 2: 实现最小页面改动**

在 `wxapp/pages/market/index.vue` 中：

```js
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  groupCode: '',
  typeCode: '',
})
```

删除方向筛选模板、样式、`directionFilterOptions`、`chooseResourceDirection`，并从 `listResources`、`applyCurrentGroupTypes` 和 `openSearchPage` 中移除方向条件。

- [x] **Step 3: 运行定向测试**

```bash
cd wxapp && node --test pages/market/index.test.mjs scripts/validate-flows.test.mjs
```

预期：全部通过。

- [x] **Step 4: 运行完整检查**

```bash
cd wxapp && npm run check
```

预期：页面校验、流程校验、全部测试和微信小程序构建通过。

- [x] **Step 5: 检查差异并提交**

```bash
git diff --check
git status --short
git add docs/superpowers/plans/2026-07-31-wxapp-market-remove-direction-filter.md \
  wxapp/pages/market/index.vue \
  wxapp/pages/market/index.test.mjs \
  wxapp/scripts/validate-flows.mjs \
  wxapp/scripts/validate-flows.test.mjs
git commit -m "feat: 精简供需首页方向筛选"
```

### Task 2: 移除搜索页方向筛选

**Files:**
- Modify: `wxapp/pages/search/index.test.mjs`
- Modify: `wxapp/scripts/validate-flows.test.mjs`
- Modify: `wxapp/scripts/validate-flows.mjs`
- Modify: `wxapp/pages/search/index.vue`

**Interfaces:**
- Consumes: `searchResources({ cityCode, groupCode, typeCode, tags, keyword, page, pageSize })`
- Produces: 无方向筛选状态的供需混合搜索结果

- [ ] **Step 1: 修改搜索页测试并验证失败**

在 `wxapp/pages/search/index.test.mjs` 中断言搜索页不再包含方向筛选模板、状态、方法和样式，且路由、待搜索条件、接口请求均不再读取或传递 `direction`。同步更新流程校验，但保留混合供应卡片和需求卡片断言。

```js
assert.doesNotMatch(source, /direction-filter-row|directionFilterOptions|chooseResourceDirection/)
assert.doesNotMatch(source, /filters\.direction|normalizeResourceDirection|routeDirection/)
assert.match(source, /DemandCard v-if="item\.direction === RESOURCE_DIRECTION_DEMAND"/)
assert.match(source, /<ResourceCard v-else/)
```

运行：

```bash
cd wxapp && node --test pages/search/index.test.mjs scripts/validate-flows.test.mjs
```

预期：搜索页仍包含方向筛选实现，因此测试失败。

- [ ] **Step 2: 实现最小页面改动**

从 `wxapp/pages/search/index.vue` 删除方向筛选模板、样式、状态、方法、路由读取和待搜索条件读取；保留结果卡片根据 `item.direction` 选择 `DemandCard` 或 `ResourceCard`。

```js
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  groupCode: '',
  typeCode: '',
  tags: [],
})

const items = currentGroupResourceTypeItems.value.map((item) => ({
  label: item.typeName,
  value: item.typeCode,
}))
```

- [ ] **Step 3: 运行定向测试**

```bash
cd wxapp && node --test pages/search/index.test.mjs scripts/validate-flows.test.mjs
```

预期：全部通过。

- [ ] **Step 4: 运行完整检查**

```bash
cd wxapp && npm run check
```

预期：页面校验、流程校验、全部测试和微信小程序构建通过。

- [ ] **Step 5: 检查差异并提交**

```bash
git diff --check
git add docs/superpowers/specs/2026-07-31-wxapp-market-remove-direction-filter-design.md \
  docs/superpowers/plans/2026-07-31-wxapp-market-remove-direction-filter.md \
  wxapp/pages/search/index.vue \
  wxapp/pages/search/index.test.mjs \
  wxapp/scripts/validate-flows.mjs \
  wxapp/scripts/validate-flows.test.mjs
git commit -m "feat: 移除供需搜索方向筛选"
```
