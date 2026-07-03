# Sourcing Map Gesture Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化小程序拿货地图页面，让默认筛选区更紧凑，并让地图支持原生拖动和双指缩放。

**Architecture:** 仅修改 `wxapp/pages/sourcing-map/index.vue` 的页面结构、交互状态和样式，并通过 `wxapp/pages/sourcing-map/index.test.mjs` 做静态回归验证。地图坐标系、后端接口和对象数据结构不变，视口参数由 `movable-view` 位移和缩放换算。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、微信小程序 `movable-area/movable-view`、Node.js `node:test` 静态验证。

---

## 文件结构

- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
  - 增加紧凑搜索筛选区和 `movable-area/movable-view` 手势地图的静态验证。
- 修改：`wxapp/pages/sourcing-map/index.vue`
  - 将筛选区默认改为紧凑搜索行和横向快捷筛选。
  - 将地图主交互容器从 `scroll-view` 改为 `movable-area/movable-view`。
  - 新增 `mapMoveX/mapMoveY`、`handleMapMove`、`handleMapScale`、`resetMapViewport` 等状态与方法。
  - 移除顶部标题区和地图缩放工具条，增加地图内浮动服务按钮。
  - 调整视口参数换算、点位样式缩放和底部详情浮层样式。

### Task 1: 写失败测试

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.test.mjs`

- [ ] **Step 1: 增加紧凑筛选区测试**

在页面测试中加入源码断言，要求存在以下标识：

```js
test('sourcing map page keeps search filters compact by default', () => {
  for (const token of [
    'search-shell',
    'compact-filter-row',
    'filter-toggle-button',
    'filtersExpanded',
    'toggleFiltersExpanded',
    'quickFilterItems',
    'activeFilterSummary',
    'clearMapConditions',
    '更多筛选',
    '收起筛选',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /<view v-if="filtersExpanded" class="filter-panel">/)
})
```

- [ ] **Step 2: 增加手势地图测试**

在页面测试中加入源码断言，要求地图使用 `movable-area/movable-view`：

```js
test('sourcing map page uses movable view for map gestures', () => {
  for (const token of [
    'movable-area',
    'movable-view',
    'map-gesture-area',
    'map-movable',
    'direction="all"',
    'scale',
    'inertia',
    ':scale-value="mapScale"',
    ':scale-min="MAP_MIN_SCALE"',
    ':scale-max="MAP_MAX_SCALE"',
    ':x="mapMoveX"',
    ':y="mapMoveY"',
    '@change="handleMapMove"',
    '@scale="handleMapScale"',
    'mapMoveX',
    'mapMoveY',
    'renderStageScale',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.doesNotMatch(source, /<scroll-view class="map-scroll"/)
})
```

- [ ] **Step 3: 运行测试确认失败**

Run: `node wxapp/pages/sourcing-map/index.test.mjs`

Expected: FAIL，失败原因是当前页面还没有 `search-shell`、`movable-area`、`handleMapScale` 等新标识。

### Task 2: 实现紧凑筛选区

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.vue`
- Test: `wxapp/pages/sourcing-map/index.test.mjs`

- [ ] **Step 1: 新增筛选展开状态和快捷筛选数据**

新增 `filtersExpanded`、`quickFilterItems`、`activeFilterSummary`、`toggleFiltersExpanded`、`clearMapConditions`。

- [ ] **Step 2: 改造模板**

将默认筛选区改为：

```vue
<view class="search-panel">
  <view class="search-shell">搜索框、筛选按钮、搜索按钮</view>
  <scroll-view class="compact-filter-row" scroll-x>快捷筛选 chip</scroll-view>
  <view v-if="keyword || hasActiveFilters" class="active-filter-summary">当前条件和清除按钮</view>
  <view v-if="filtersExpanded" class="filter-panel">完整筛选组</view>
  <scroll-view v-if="sceneTabsVisible" class="scene-tabs" scroll-x>场景 tabs</scroll-view>
</view>
```

- [ ] **Step 3: 调整样式**

压缩 `search-panel`、`search-shell`、`compact-filter-row`、`filter-chip` 的高度和间距，确保默认状态不再纵向展示四组筛选。

### Task 3: 实现手势地图容器

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.vue`
- Test: `wxapp/pages/sourcing-map/index.test.mjs`

- [ ] **Step 1: 替换地图容器模板**

将 `<scroll-view class="map-scroll">` 替换为：

```vue
<movable-area class="map-gesture-area">
  <movable-view
    class="map-movable"
    direction="all"
    scale
    inertia
    :scale-min="MAP_MIN_SCALE"
    :scale-max="MAP_MAX_SCALE"
    :scale-value="mapScale"
    :x="mapMoveX"
    :y="mapMoveY"
    :style="movableStageStyle"
    @change="handleMapMove"
    @scale="handleMapScale"
  >
    <view class="map-stage" :style="stageStyle">原有底图和点位</view>
  </movable-view>
</movable-area>
```

- [ ] **Step 2: 新增位移状态和缩放换算**

新增 `mapMoveX/mapMoveY`，用 `renderStageScale` 渲染底图和点位，用 `effectiveStageScale` 计算真实视口缩放。

- [ ] **Step 3: 改造视口加载**

`buildViewportQueryParams` 使用 `-mapMoveX/-mapMoveY` 和 `effectiveStageScale` 换算 `minX/minY/maxX/maxY`。

- [ ] **Step 4: 改造聚焦和地图服务按钮**

`focusMapCenter` 设置 `mapMoveX/mapMoveY`；移除放大、缩小和比例展示；新增 `refreshCurrentMapData` 和 `resetMapViewport`，并通过地图内浮动按钮触发刷新和归位。

### Task 4: 验证

**Files:**
- Test: `wxapp/pages/sourcing-map/index.test.mjs`
- Test: `wxapp/scripts/validate-pages.mjs`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 跑页面测试**

Run: `node wxapp/pages/sourcing-map/index.test.mjs`

Expected: PASS。

- [ ] **Step 2: 跑页面注册验证**

Run: `npm --prefix wxapp run validate:pages`

Expected: PASS。

- [ ] **Step 3: 跑流程验证**

Run: `npm --prefix wxapp run validate:flows`

Expected: PASS。

## 自检

- 设计目标均对应到任务：筛选区压缩在 Task 2，地图手势在 Task 3，回归验证在 Task 4。
- 计划不涉及后端、后台和 API 改动，符合精准改动原则。
- 没有新增跨页面抽象，所有变更收敛在拿货地图页面。
