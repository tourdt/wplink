# 小程序分类过滤栏视觉精修 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将供需市场页和供需搜索页的分类过滤栏实施为已确认的 A1「深色锚点」样式，并完善展开面板的标题、当前状态和选中反馈。

**Architecture:** 保留两个页面现有的分类数据流、原位展开逻辑和三列网格结构，仅在页面内增加 `selectedTypeName` 计算值、面板标题模板与 CSS 对勾。两个页面继续使用同构模板和局部 SCSS，不为本次小范围视觉精修抽取新组件，避免扩大改动面。

**Tech Stack:** Vue 3 Composition API、uni-app、微信小程序、SCSS、Node.js `node:test`

## Global Constraints

- 只修改 `wxapp/pages/market/index.vue`、`wxapp/pages/market/index.test.mjs`、`wxapp/pages/search/index.vue`、`wxapp/pages/search/index.test.mjs`。
- 不改变分类接口、分类数据、筛选查询参数、资源加载逻辑和面板开关行为。
- 保留 `72rpx × 72rpx` 箭头点击热区、三列分类网格和 `360rpx` 面板最大高度。
- 面板标题固定使用“选择分类”，当前状态固定使用“当前：{分类名称}”。
- 视觉令牌沿用 `$wplink-primary: #061625`、`$wplink-bg: #f4f7fd`、`$wplink-card: #ffffff`、`$wplink-muted: #6b7280`、`$wplink-line: #d8e0ec`。
- 不新增图片、图标、字体、依赖或全局样式变量；对勾使用局部 CSS 绘制。
- 保留工作区内所有与本任务无关的现有修改，不暂存、不格式化、不覆盖。

## 设计校准

- **对象：** 在供需市场中快速扫选货源、需求和服务的用户。
- **单一任务：** 立即识别当前分类，并能快速打开全部分类切换。
- **文字层级：** 选中分类使用 `26rpx/700`；面板标题使用 `24rpx/700`；当前状态使用 `22rpx/400`；未选分类保持 `24rpx` 至 `26rpx` 的常规字重。
- **布局：** 横滑分类与箭头仍在一个 `80rpx` 组合控件中；面板以 `10rpx` 间距原位展开，标题行位于三列网格上方。
- **标志性元素：** 深墨色选中块在横滑分类和展开网格之间保持一致，像一个随选择移动的“分类锚点”；箭头、边框和阴影全部退为辅助层。
- **自我批评与修正：** 没有增加渐变、插图或额外动画，因为这些装饰与快速筛选任务无关；唯一承担辨识度的深色锚点来自项目现有品牌主色，也不会让控件变成通用模板式的彩色胶囊集合。

---

### Task 1: 精修供需市场页分类过滤栏

**Files:**
- Modify: `wxapp/pages/market/index.test.mjs`
- Modify: `wxapp/pages/market/index.vue`

**Interfaces:**
- Consumes: `resourceTypes: Ref<Array<{ label: string, value: string }>>` 与 `filters.typeCode: string`
- Produces: `selectedTypeName: ComputedRef<string>`，以及 `.type-panel-head`、`.type-panel-title`、`.type-panel-current`、`.type-panel-check` 视觉结构

- [ ] **Step 1: 写入市场页 A1 视觉契约的失败测试**

在 `market page expands secondary categories inline from a compact arrow control` 测试中增加以下断言：

```js
assert.match(source, /const selectedTypeName = computed\(\(\) => resourceTypes\.value\.find\(\(item\) => item\.value === filters\.typeCode\)\?\.label \|\| '全部'\)/)
assert.match(source, /<view class="type-panel-head">[\s\S]*<text class="type-panel-title">选择分类<\/text>[\s\S]*<text class="type-panel-current">当前：\{\{ selectedTypeName \}\}<\/text>[\s\S]*<\/view>/)
assert.match(source, /v-if="item\.value === filters\.typeCode"[\s\S]*class="type-panel-check"/)
assert.match(cssBlock('.filter-button.active'), /background:\s*\$wplink-primary;/)
assert.match(cssBlock('.filter-button.active'), /color:\s*\$wplink-card;/)
assert.match(cssBlock('.type-panel-toggle'), /border-left:\s*1rpx solid \$wplink-line;/)
assert.match(cssBlock('.type-panel-toggle'), /background:\s*transparent;/)
assert.match(cssBlock('.type-panel'), /border-radius:\s*12rpx;/)
assert.match(cssBlock('.type-panel-head'), /justify-content:\s*space-between;/)
assert.match(cssBlock('.type-panel-button.active'), /background:\s*\$wplink-primary;/)
assert.match(cssBlock('.type-panel-check'), /border-left:\s*2rpx solid currentColor;/)
```

这些断言分别防止当前状态来源、标题结构、选中对勾、深色锚点和弱化箭头在后续修改中丢失。

- [ ] **Step 2: 运行市场页测试并确认因 A1 结构缺失而失败**

Run:

```bash
cd wxapp
node --test pages/market/index.test.mjs
```

Expected: FAIL，首个失败指向缺少 `selectedTypeName`、`.type-panel-head` 或深色选中态，而不是语法或路径错误。

- [ ] **Step 3: 实施市场页最小模板与状态改动**

在 `visibleResourceTypes` 附近增加：

```js
const selectedTypeName = computed(() => resourceTypes.value.find((item) => item.value === filters.typeCode)?.label || '全部')
```

在 `.type-panel` 的 `.type-panel-grid` 之前增加：

```vue
<view class="type-panel-head">
  <text class="type-panel-title">选择分类</text>
  <text class="type-panel-current">当前：{{ selectedTypeName }}</text>
</view>
```

在每个 `.type-panel-button` 内、分类文字之后增加：

```vue
<text
  v-if="item.value === filters.typeCode"
  class="type-panel-check"
  aria-hidden="true"
></text>
```

- [ ] **Step 4: 实施市场页 A1 局部 SCSS**

保持现有尺寸约束，并将对应规则调整为：

```scss
.filter-shell {
  border-radius: 12rpx;
  box-shadow: 0 2rpx 8rpx rgba(6, 22, 37, 0.03);
}

.filter-button {
  margin-right: 8rpx;
  background: transparent;
}

.filter-button.active {
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-toggle {
  padding: 0;
  border-left: 1rpx solid $wplink-line;
  border-radius: 0 8rpx 8rpx 0;
  background: transparent;
}

.type-panel-toggle:active {
  background: rgba($wplink-primary, 0.04);
}

.type-panel {
  margin-top: 10rpx;
  padding: 16rpx;
  border-radius: 12rpx;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.08);
}

.type-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 14rpx;
}

.type-panel-title {
  flex: 0 0 auto;
  color: $wplink-text;
  font-size: 24rpx;
  font-weight: 700;
}

.type-panel-current {
  min-width: 0;
  overflow: hidden;
  color: $wplink-muted;
  font-size: 22rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-panel-button {
  position: relative;
  border: 1rpx solid transparent;
  background: $wplink-bg;
}

.type-panel-button.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-check {
  position: absolute;
  top: 10rpx;
  right: 12rpx;
  width: 10rpx;
  height: 6rpx;
  border-left: 2rpx solid currentColor;
  border-bottom: 2rpx solid currentColor;
  transform: rotate(-45deg);
}
```

- [ ] **Step 5: 运行市场页测试并提交**

Run:

```bash
cd wxapp
node --test pages/market/index.test.mjs
```

Expected: PASS，市场页测试文件零失败。

Commit:

```bash
git add wxapp/pages/market/index.test.mjs wxapp/pages/market/index.vue
git commit -m "feat: 精修市场页分类过滤栏"
```

---

### Task 2: 同步供需搜索页分类过滤栏

**Files:**
- Modify: `wxapp/pages/search/index.test.mjs`
- Modify: `wxapp/pages/search/index.vue`

**Interfaces:**
- Consumes: `resourceTypes: Ref<Array<{ label: string, value: string }>>` 与 `filters.typeCode: string`
- Produces: 与市场页同名、同语义的 `selectedTypeName` 和 A1 面板视觉结构

- [ ] **Step 1: 写入搜索页 A1 一致性契约的失败测试**

在 `search page matches the market inline category panel behavior` 测试中增加与 Task 1 相同的结构和样式断言：

```js
assert.match(source, /const selectedTypeName = computed\(\(\) => resourceTypes\.value\.find\(\(item\) => item\.value === filters\.typeCode\)\?\.label \|\| '全部'\)/)
assert.match(source, /<view class="type-panel-head">[\s\S]*<text class="type-panel-title">选择分类<\/text>[\s\S]*<text class="type-panel-current">当前：\{\{ selectedTypeName \}\}<\/text>[\s\S]*<\/view>/)
assert.match(source, /v-if="item\.value === filters\.typeCode"[\s\S]*class="type-panel-check"/)
assert.match(cssBlock('.filter-button.active'), /background:\s*\$wplink-primary;/)
assert.match(cssBlock('.filter-button.active'), /color:\s*\$wplink-card;/)
assert.match(cssBlock('.type-panel-toggle'), /border-left:\s*1rpx solid \$wplink-line;/)
assert.match(cssBlock('.type-panel-toggle'), /background:\s*transparent;/)
assert.match(cssBlock('.type-panel'), /border-radius:\s*12rpx;/)
assert.match(cssBlock('.type-panel-head'), /justify-content:\s*space-between;/)
assert.match(cssBlock('.type-panel-button.active'), /background:\s*\$wplink-primary;/)
assert.match(cssBlock('.type-panel-check'), /border-left:\s*2rpx solid currentColor;/)
```

移除搜索页旧断言中对浅橙选中态或浅蓝箭头背景的依赖；已有尺寸、网格、滚动和标签筛选顺序断言继续保留。

- [ ] **Step 2: 运行搜索页测试并确认因 A1 结构缺失而失败**

Run:

```bash
cd wxapp
node --test pages/search/index.test.mjs
```

Expected: FAIL，首个失败来自缺少 A1 标题、状态、对勾或深色样式。

- [ ] **Step 3: 实施搜索页最小模板、状态与 SCSS 改动**

加入 `selectedTypeName`、`.type-panel-head` 和 `.type-panel-check`。搜索页已有 `.tag-filter-row` 的模板位置和样式保持不变。

生产代码必须与以下契约一致：

```js
const selectedTypeName = computed(() => resourceTypes.value.find((item) => item.value === filters.typeCode)?.label || '全部')
```

```vue
<view class="type-panel-head">
  <text class="type-panel-title">选择分类</text>
  <text class="type-panel-current">当前：{{ selectedTypeName }}</text>
</view>
```

```vue
<text
  v-if="item.value === filters.typeCode"
  class="type-panel-check"
  aria-hidden="true"
></text>
```

将搜索页对应局部样式精确调整为：

```scss
.filter-shell {
  border-radius: 12rpx;
  box-shadow: 0 2rpx 8rpx rgba(6, 22, 37, 0.03);
}

.filter-button {
  margin-right: 8rpx;
  background: transparent;
}

.filter-button.active {
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-toggle {
  padding: 0;
  border-left: 1rpx solid $wplink-line;
  border-radius: 0 8rpx 8rpx 0;
  background: transparent;
}

.type-panel-toggle:active {
  background: rgba($wplink-primary, 0.04);
}

.type-panel {
  margin-top: 10rpx;
  padding: 16rpx;
  border-radius: 12rpx;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.08);
}

.type-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 14rpx;
}

.type-panel-title {
  flex: 0 0 auto;
  color: $wplink-text;
  font-size: 24rpx;
  font-weight: 700;
}

.type-panel-current {
  min-width: 0;
  overflow: hidden;
  color: $wplink-muted;
  font-size: 22rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-panel-button {
  position: relative;
  border: 1rpx solid transparent;
  background: $wplink-bg;
}

.type-panel-button.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-check {
  position: absolute;
  top: 10rpx;
  right: 12rpx;
  width: 10rpx;
  height: 6rpx;
  border-left: 2rpx solid currentColor;
  border-bottom: 2rpx solid currentColor;
  transform: rotate(-45deg);
}
```

- [ ] **Step 4: 运行搜索页测试和两页定向测试**

Run:

```bash
cd wxapp
node --test pages/search/index.test.mjs
node --test pages/market/index.test.mjs pages/search/index.test.mjs
```

Expected: 两条命令均 PASS，零失败。

- [ ] **Step 5: 提交搜索页同步改动**

```bash
git add wxapp/pages/search/index.test.mjs wxapp/pages/search/index.vue
git commit -m "feat: 同步搜索页分类过滤栏样式"
```

---

### Task 3: 完整验证与视觉自检

**Files:**
- Verify only: `wxapp/pages/market/index.vue`
- Verify only: `wxapp/pages/search/index.vue`

**Interfaces:**
- Consumes: Task 1 与 Task 2 已提交的页面和测试
- Produces: 可交付的 A1 分类过滤栏视觉精修

- [ ] **Step 1: 检查变更范围和格式**

Run:

```bash
git diff HEAD~2 --check
git diff HEAD~2 --stat
git status --short
```

Expected: 与本任务有关的代码只涉及两个页面和两个测试；其他用户改动仍保持原有未暂存状态。

- [ ] **Step 2: 运行小程序完整检查**

Run:

```bash
cd wxapp
npm run check
```

Expected: 页面校验、流程校验、全部 Node 测试和 `mp-weixin` 构建均通过；若仅出现项目已有 Sass deprecation warning，记录但不在本次范围内处理。

- [ ] **Step 3: 按规格逐项自检**

逐项确认：

```text
[ ] 横滑分类只有当前项使用深墨色背景与白色文字
[ ] 箭头常态透明且左侧有细分隔线
[ ] 展开面板显示“选择分类”和正确的“当前：分类名称”
[ ] 面板选中项使用深墨色并显示 CSS 对勾
[ ] 面板仍为三列、最大高度 360rpx、超出内部滚动
[ ] 搜索页标签筛选仍位于分类面板之后
[ ] 分类选择、面板收起、一级类目切换和资源加载逻辑没有变化
```
