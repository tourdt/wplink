# 供需详情参数合并 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除详情页自动组合标题和核心事实卡片，将数量/面积、价格/预算与类型属性合并展示在“详细参数”区块。

**Architecture:** `resourceDetailState.js` 保留供需方向和类型名称的展示状态，并新增统一参数构建函数，按核心参数在前、类型属性在后的顺序输出可供页面直接渲染的参数项。`detail.vue` 删除标题与事实卡片模板，继续通过该工具决定需求/供应的价格标签和长文本整行样式。

**Tech Stack:** Vue 3 `<script setup>`、uni-app、Node.js 内置测试运行器。

## Global Constraints

- 不改变发布端自动生成 `title` 的逻辑或已存数据。
- 不调整图片、地址、商家资料、推荐、联系、收藏和管理功能。
- 不增加接口、配置、数据库字段或迁移。
- 描述仅在真实填写时以“补充说明”展示，空值不占位。

---

### Task 1: 合并核心参数并精简详情摘要

**Files:**
- Modify: `wxapp/common/resourceDetailState.js:5-32`
- Modify: `wxapp/common/resourceDetailState.test.mjs:8-55`
- Modify: `wxapp/pages/resource/detail.vue:35-66,326`
- Modify: `wxapp/pages/resource/detail.test.mjs:71-105,228-245`

**Interfaces:**
- Consumes: `buildResourceDetailPresentation(resource)`、`buildDetailSpecItems(attributeItems)` 和非地址 `attributeSpecItems`。
- Produces: `buildResourceDetailSpecItems(resource, attributeItems)`，返回已过滤空值、按数量/价格优先的 `{ label, value, fullWidth }[]`。

- [ ] **Step 1: 写入失败的工具与页面回归测试**

在 `resourceDetailState.test.mjs` 中将自动标题/事实卡片断言替换为方向和类型断言，并添加参数合并行为：

```js
test('puts demand quantity and budget before configured specs', () => {
  const items = buildResourceDetailSpecItems(
    { direction: 'demand', quantityText: '5000 件', priceText: '面议' },
    [{ label: '交期要求', value: '20 天内完成' }],
  )

  assert.deepEqual(items, [
    { label: '数量/面积', value: '5000 件', fullWidth: false },
    { label: '预算/报价', value: '面议', fullWidth: false },
    { label: '交期要求', value: '20 天内完成', fullWidth: true },
  ])
})
```

在 `detail.test.mjs` 中断言详情页不含 `summary-title`、`summary-facts`、`detailPresentation.headline` 和 `detailPresentation.facts`，并断言：

```js
assert.match(source, /const specItems = computed\(\(\) => buildResourceDetailSpecItems\(resource\.value, attributeSpecItems\.value\)\)/)
assert.match(source, /<text class="section-title">补充说明<\/text>/)
```

- [ ] **Step 2: 运行聚焦测试并确认失败**

Run: `node --test wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.test.mjs`

Expected: 参数合并函数尚未导出，且详情页仍渲染自动组合标题和核心事实卡片，因此新断言失败。

- [ ] **Step 3: 编写最小实现**

在 `resourceDetailState.js` 中将展示状态收敛为：

```js
return {
  isDemand,
  noun: isDemand ? '需求' : '供应',
  typeName,
}
```

新增并导出：

```js
export function buildResourceDetailSpecItems(resource = {}, attributeItems = []) {
  const isDemand = isDemandResource(resource)
  return buildDetailSpecItems([
    { label: '数量/面积', value: resource.quantityText },
    { label: isDemand ? '预算/报价' : '价格/报价', value: resource.priceText },
    ...attributeItems,
  ])
}
```

在 `detail.vue` 中：

```vue
<view class="summary-kicker">
  <text :class="['direction-badge', detailPresentation.isDemand ? 'demand' : '']">{{ resourceNoun }}</text>
  <text class="summary-type">{{ detailPresentation.typeName }}</text>
</view>
```

删除 `summary-title`、`summary-facts`、`summary-fact*` 相关模板和样式；将参数计算替换为：

```js
const specItems = computed(() => buildResourceDetailSpecItems(resource.value, attributeSpecItems.value))
```

并从导入中加入 `buildResourceDetailSpecItems`。

- [ ] **Step 4: 运行聚焦测试并确认通过**

Run: `node --test wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.test.mjs`

Expected: 工具行为和详情页回归测试均通过；需求使用“预算/报价”，供应使用“价格/报价”，空数量/价格不生成参数项。

- [ ] **Step 5: 运行完整前端验证**

Run: `npm run check`

Expected: 页面校验、流程校验、Node 测试和微信小程序构建均退出码为 0。

- [ ] **Step 6: 提交实现**

```bash
git add wxapp/common/resourceDetailState.js wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat: unify resource detail specs"
```
