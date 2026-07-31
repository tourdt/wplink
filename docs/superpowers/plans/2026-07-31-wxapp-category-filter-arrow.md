# 小程序分类过滤条箭头展开 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将供需市场页和供需搜索页右侧的“全部分类”文字按钮改为单箭头按钮，并在过滤条下方原位展开三列分类面板。

**Architecture:** 继续复用现有 `resourceTypes`、`filters.typeCode`、`selectType` 和资源加载逻辑，只把二级分类底部抽屉替换为工具栏内的原位面板。两个页面统一使用 `showTypePanel`、`toggleTypePanel`、`openTypePanel`、`closeTypePanel`，页面根节点点击负责关闭面板，过滤控件内部阻止冒泡，避免箭头刚展开就被页面事件关闭。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、SCSS、Node.js `node:test` 静态页面测试。

## Global Constraints

- 所有实施说明和代码注释使用中文。
- 不修改后端接口、分类数据结构、查询参数和现有资源加载逻辑。
- 不调整一级类目标题栏和一级类目底部抽屉。
- 过滤条外层高度 `80rpx`，边框 `1rpx`，圆角 `10rpx`，内边距 `4rpx`。
- 箭头按钮尺寸 `72rpx × 72rpx`。
- 原位分类面板使用三列网格，最大高度 `360rpx`，分类项高度 `72rpx`，文案最多两行。
- 搜索页和供需市场页使用相同的状态、方法和样式命名。
- 不提交或修改当前工作区中与本需求无关的后台构建产物变更。

---

## 文件结构

- `wxapp/pages/market/index.vue`：供需市场页分类箭头、原位面板、状态与样式。
- `wxapp/pages/market/index.test.mjs`：市场页模板、状态、交互关闭规则和尺寸断言。
- `wxapp/pages/search/index.vue`：供需搜索页同款分类箭头、原位面板、状态与样式。
- `wxapp/pages/search/index.test.mjs`：搜索页模板、状态、交互关闭规则和尺寸断言。
- `wxapp/scripts/validate-flows.test.mjs`：只运行回归验证，不修改既有业务流程断言。

### Task 1: 供需市场页改为箭头原位展开

**Files:**
- Modify: `wxapp/pages/market/index.test.mjs:122-163`
- Modify: `wxapp/pages/market/index.vue:1-105`
- Modify: `wxapp/pages/market/index.vue:135-305`
- Modify: `wxapp/pages/market/index.vue:452-501`
- Modify: `wxapp/pages/market/index.vue:565-645`

**Interfaces:**
- Consumes: `resourceTypes: Ref<Array<{label: string, value: string}>>`、`filters.typeCode: string`、`selectType(typeCode: string): Promise<void>`。
- Produces: `showTypePanel: Ref<boolean>`、`openTypePanel(): void`、`closeTypePanel(): void`、`toggleTypePanel(): void`，供模板和市场页静态测试使用。

- [ ] **Step 1: 更新市场页测试表达箭头与原位面板**

将“market page shows all secondary categories and scrolls selected category into view”测试中的底部抽屉断言替换为以下核心断言：

```js
for (const token of [
  'showTypePanel',
  'openTypePanel',
  'closeTypePanel',
  'toggleTypePanel',
  'type-panel-toggle',
  'type-panel-arrow',
  'type-panel',
  'type-panel-grid',
  ':aria-expanded="showTypePanel"',
  "showTypePanel ? '收起全部分类' : '展开全部分类'",
]) {
  assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
}

assert.match(source, /<view class="resource-page" :style="resourcePageStyle" @click="closeTypePanel">/)
assert.match(source, /<view :class="\['filter-shell', showAllTypeButton \? 'has-type-panel-button' : ''\]" @click\.stop>/)
assert.match(source, /<scroll-view[\s\S]*v-if="showTypePanel"[\s\S]*class="type-panel"[\s\S]*scroll-y/)
assert.match(source, /v-for="item in resourceTypes"[\s\S]*class="type-panel-button"/)
assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypePanel\.value = false[\s\S]*scrollToSelectedType\(typeCode\)/)
assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*showTypePanel\.value = false[\s\S]*applyCurrentGroupTypes\(\)/)
assert.doesNotMatch(source, /class="all-type-button"/)
assert.doesNotMatch(source, /class="type-drawer-mask"/)
assert.doesNotMatch(source, />\s*全部分类\s*</)
assert.match(cssBlock('.filter-shell'), /height:\s*80rpx;/)
assert.match(cssBlock('.filter-shell'), /padding:\s*4rpx;/)
assert.match(cssBlock('.filter-shell.has-type-panel-button'), /grid-template-columns:\s*minmax\(0,\s*1fr\) 72rpx;/)
assert.match(cssBlock('.type-panel-toggle'), /width:\s*72rpx;/)
assert.match(cssBlock('.type-panel-toggle'), /height:\s*72rpx;/)
assert.match(cssBlock('.type-panel'), /max-height:\s*360rpx;/)
assert.match(cssBlock('.type-panel-grid'), /grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\);/)
assert.match(cssBlock('.type-panel-button'), /height:\s*72rpx;/)
```

保留横滑、选中项定位和 `showAllTypeButton` 的既有断言。

- [ ] **Step 2: 运行市场页测试并确认失败**

Run:

```bash
node --test wxapp/pages/market/index.test.mjs
```

Expected: FAIL，失败信息指出缺少 `showTypePanel`、`type-panel-toggle` 或原位面板样式。

- [ ] **Step 3: 实现市场页箭头和原位分类面板**

先把根节点开始标签精确改为：

```vue
<view class="resource-page" :style="resourcePageStyle" @click="closeTypePanel">
```

再用以下完整结构替换现有 `.filter-shell` 和二级分类底部抽屉：

```vue
<view :class="['filter-shell', showAllTypeButton ? 'has-type-panel-button' : '']" @click.stop>
  <scroll-view
    class="filter-row"
    scroll-x
    scroll-with-animation
    enhanced
    :show-scrollbar="false"
    :scroll-into-view="scrollIntoTypeId"
  >
    <button
      v-for="item in visibleResourceTypes"
      :key="item.value"
      :id="getTypeButtonId(item.value)"
      :class="['filter-button', item.value === filters.typeCode ? 'active' : '']"
      @click="selectType(item.value)"
    >
      {{ item.label }}
    </button>
  </scroll-view>
  <button
    v-if="showAllTypeButton"
    class="type-panel-toggle"
    :aria-label="showTypePanel ? '收起全部分类' : '展开全部分类'"
    :aria-expanded="showTypePanel"
    @click.stop="toggleTypePanel"
  >
    <text :class="['type-panel-arrow', showTypePanel ? 'expanded' : '']"></text>
  </button>
</view>
<scroll-view
  v-if="showTypePanel"
  class="type-panel"
  scroll-y
  enhanced
  :show-scrollbar="false"
  @click.stop
>
  <view class="type-panel-grid">
    <button
      v-for="item in resourceTypes"
      :key="item.value"
      :class="['type-panel-button', item.value === filters.typeCode ? 'active' : '']"
      @click="selectType(item.value)"
    >
      <text class="type-panel-button-text">{{ item.label }}</text>
    </button>
  </view>
</scroll-view>
```

状态和方法改为：

```js
const showTypePanel = ref(false)

function openTypePanel() {
  showGroupDrawer.value = false
  showTypePanel.value = true
}

function closeTypePanel() {
  showTypePanel.value = false
}

function toggleTypePanel() {
  if (showTypePanel.value) {
    closeTypePanel()
    return
  }
  openTypePanel()
}
```

`selectGroup`、`selectType`、`openGroupDrawer` 中统一写入 `showTypePanel.value = false`。删除二级分类底部抽屉模板和只由它使用的样式；保留一级类目抽屉的 `.type-drawer-head`、`.type-drawer-title`、`.type-drawer-close`。

新增样式：

```scss
.filter-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: center;
  gap: 4rpx;
  height: 80rpx;
  padding: 4rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
}

.filter-shell.has-type-panel-button {
  grid-template-columns: minmax(0, 1fr) 72rpx;
}

.type-panel-toggle {
  width: 72rpx;
  height: 72rpx;
  border-radius: 8rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.type-panel-arrow {
  width: 14rpx;
  height: 14rpx;
  border-right: 3rpx solid currentColor;
  border-bottom: 3rpx solid currentColor;
  transform: rotate(45deg) translate(-2rpx, -2rpx);
}

.type-panel-arrow.expanded {
  transform: rotate(225deg) translate(-1rpx, -1rpx);
}

.type-panel {
  max-height: 360rpx;
  margin-top: 8rpx;
  padding: 12rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
  box-shadow: 0 12rpx 28rpx rgba(6, 22, 37, 0.12);
}

.type-panel-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}

.type-panel-button {
  height: 72rpx;
  padding: 0 10rpx;
  border-radius: 10rpx;
  background: $wplink-bg;
  color: #364152;
  font-size: 24rpx;
}

.type-panel-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
  font-weight: 700;
}

.type-panel-button-text {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
```

- [ ] **Step 4: 运行市场页测试并确认通过**

Run:

```bash
node --test wxapp/pages/market/index.test.mjs
```

Expected: PASS。

- [ ] **Step 5: 提交市场页改动**

```bash
git add wxapp/pages/market/index.vue wxapp/pages/market/index.test.mjs
git commit -m "feat: 优化市场页分类展开入口"
```

### Task 2: 供需搜索页使用同款箭头原位展开

**Files:**
- Modify: `wxapp/pages/search/index.test.mjs:61-102`
- Modify: `wxapp/pages/search/index.vue:1-148`
- Modify: `wxapp/pages/search/index.vue:175-215`
- Modify: `wxapp/pages/search/index.vue:408-542`
- Modify: `wxapp/pages/search/index.vue:721-767`
- Modify: `wxapp/pages/search/index.vue:955-1045`

**Interfaces:**
- Consumes: `resourceTypes`、`filters.typeCode`、`selectType`、`searchTagOptions`。
- Produces: 与市场页相同的 `showTypePanel`、`openTypePanel`、`closeTypePanel`、`toggleTypePanel` 和 `type-panel-*` 模板/样式命名。

- [ ] **Step 1: 更新搜索页测试表达同款交互**

将“search page matches market category browsing controls”中的底部抽屉断言替换为与 Task 1 相同的箭头、原位面板、关闭行为和尺寸断言，并额外保留：

```js
assert.match(source, /<scroll-view[\s\S]*v-if="searchTagOptions\.length"[\s\S]*class="tag-filter-row"/)
assert.match(source, /<scroll-view[\s\S]*v-if="showTypePanel"[\s\S]*class="type-panel"[\s\S]*<\/scroll-view>[\s\S]*<scroll-view[\s\S]*v-if="searchTagOptions\.length"/)
```

这条断言保证搜索标签仍位于原位分类面板之后。

- [ ] **Step 2: 运行搜索页测试并确认失败**

Run:

```bash
node --test wxapp/pages/search/index.test.mjs
```

Expected: FAIL，失败信息指出仍使用 `showTypeDrawer` 或缺少原位分类面板。

- [ ] **Step 3: 实现搜索页箭头和原位分类面板**

按 Task 1 的同名模板、状态、方法和 SCSS 实现搜索页；差异仅为根节点类名：

```vue
<view class="search-page" :style="searchPageStyle" @click="closeTypePanel">
```

原位分类面板放在 `.filter-shell` 之后、`.tag-filter-row` 之前。`selectGroup`、`selectType`、`openGroupDrawer`、路由搜索初始化中原有关闭二级抽屉的代码全部改为关闭 `showTypePanel`。

删除二级分类底部抽屉模板和只由它使用的样式，一级类目抽屉样式保留。

- [ ] **Step 4: 运行搜索页测试并确认通过**

Run:

```bash
node --test wxapp/pages/search/index.test.mjs
```

Expected: PASS。

- [ ] **Step 5: 提交搜索页改动**

```bash
git add wxapp/pages/search/index.vue wxapp/pages/search/index.test.mjs
git commit -m "feat: 优化搜索页分类展开入口"
```

### Task 3: 回归验证分类筛选主流程

**Files:**
- Test: `wxapp/pages/market/index.test.mjs`
- Test: `wxapp/pages/search/index.test.mjs`
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Verify: `wxapp/package.json`

**Interfaces:**
- Consumes: Task 1 和 Task 2 完成的页面模板、状态和样式。
- Produces: 可通过小程序构建和现有流程校验的分类过滤条实现。

- [ ] **Step 1: 运行两个页面的聚焦测试**

Run:

```bash
node --test wxapp/pages/market/index.test.mjs wxapp/pages/search/index.test.mjs
```

Expected: PASS。

- [ ] **Step 2: 运行流程验证测试**

Run:

```bash
node --test wxapp/scripts/validate-flows.test.mjs
```

Expected: PASS。

- [ ] **Step 3: 运行小程序完整检查**

Run:

```bash
npm --prefix wxapp run check
```

Expected: 页面校验、流程校验、全部 Node 测试和 `build:mp-weixin` 均通过。

- [ ] **Step 4: 检查精准改动范围**

Run:

```bash
git status --short
git diff --check
git diff --stat HEAD~2
```

Expected: 新增提交只包含市场页、搜索页及其测试；工作区中原有后台构建产物变更保持未提交且未被修改。
