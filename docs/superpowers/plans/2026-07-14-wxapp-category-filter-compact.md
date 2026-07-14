# 小程序频道化分类筛选 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将搜索页和供需市场页改为“标题栏一级频道 + 单行二级类目横滑”，突出用户从首页进入后的频道心智，并减少列表首屏筛选占用。

**Architecture:** 继续复用现有 `groupCode/typeCode` 过滤参数和 `groupResourceTypes` 分组结果，不改后端接口。自定义标题栏显示当前一级频道，点击标题栏打开一级类目抽屉；筛选区只保留当前频道内的二级类目横滑和“全部分类”二级抽屉。搜索框占位文案随一级频道变化，例如“在童装批发中搜索”。

**Tech Stack:** uni-app/Vue 小程序、现有 SCSS 变量、Node 静态测试。

---

### Task 1: 搜索页和市场页测试先表达频道标题栏交互

**Files:**
- Modify: `wxapp/pages/search/index.test.mjs`
- Modify: `wxapp/pages/market/index.test.mjs`

- [ ] **Step 1: Write the failing test**

更新现有分类筛选测试，断言页面包含：

```js
'class="channel-title-button"'
'channelTitle'
'selectedGroupName'
'showGroupDrawer'
'group-drawer-mask'
'drawer-group-list'
```

同时断言模板不再包含常驻一级类目横滑和筛选区一级按钮：

```js
'class="filter-row group-row"'
'class="group-select-button"'
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
node --test wxapp/pages/search/index.test.mjs wxapp/pages/market/index.test.mjs
```

Expected: FAIL，因为当前实现仍把一级类目放在筛选区。

### Task 2: 搜索页实现标题栏一级频道

**Files:**
- Modify: `wxapp/pages/search/index.vue`

- [ ] **Step 1: Write minimal implementation**

在 `search-title-bar` 中用 `channel-title-button` 替代固定“供需搜索”标题。`channelTitle` 在选中一级类目时显示 `selectedGroupName`，未选一级类目时显示“供需搜索”。点击标题栏打开 `showGroupDrawer`，抽屉中列出 `groupFilterOptions`，选择后复用现有 `selectGroup`。

- [ ] **Step 2: Preserve existing behavior**

从 `filter-shell` 移除一级类目按钮。保留 `typeCode` 横滑、`scrollToSelectedType`、二级“全部分类”抽屉和搜索参数提交逻辑不变。`searchPlaceholder` 在选中一级类目时显示“在{一级类目}中搜索”。

### Task 3: 市场页实现同款标题栏一级频道

**Files:**
- Modify: `wxapp/pages/market/index.vue`

- [ ] **Step 1: Write minimal implementation**

复用搜索页的命名和样式结构，在市场页加入 `channelTitle`、`selectedGroupName`、`showGroupDrawer`、`openGroupDrawer`、`closeGroupDrawer`。未选一级类目时 `channelTitle` 显示 `PAGE_TITLE`。

- [ ] **Step 2: Preserve existing behavior**

从市场页 `filter-shell` 移除一级类目按钮。保留供需市场的搜索入口、分页、下拉刷新、二级分类抽屉和列表加载逻辑不变。搜索入口文案随一级频道变化。

### Task 4: 验证

**Files:**
- Test: `wxapp/pages/search/index.test.mjs`
- Test: `wxapp/pages/market/index.test.mjs`
- Test: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Run focused tests**

```bash
node --test wxapp/pages/search/index.test.mjs wxapp/pages/market/index.test.mjs
```

Expected: PASS.

- [ ] **Step 2: Run flow validation**

```bash
node --test wxapp/scripts/validate-flows.test.mjs
```

Expected: PASS.
