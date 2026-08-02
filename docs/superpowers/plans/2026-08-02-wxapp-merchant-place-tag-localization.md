# 拿货档口标签中文化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将拿货档口卡片中的分类和服务机器码展示为中文运营标签。

**Architecture:** 保持接口的原始标签代码用于筛选和请求；在小程序端以已加载的分类配置构建代码到中文名称的映射。页面将转换后的展示标签传入卡片，卡片不再直接拼接原始代码。

**Tech Stack:** Vue 3、uni-app、Node.js 内置 test runner。

## Global Constraints

- 仅修改小程序拿货档口页面、其状态工具与定向测试；不改接口、数据库或筛选参数。
- 分类、服务与平台标签均使用运营配置返回的 `code` 和 `name`。
- 未匹配代码及原本中文标签必须原样保留，避免丢失新配置。
- 卡片最多展示三个标签的现有规则保持不变。

---

### Task 1: 标签展示转换函数

**Files:**

- Modify: `wxapp/pages/sourcing-map/merchantPlaceState.js`
- Test: `wxapp/pages/sourcing-map/merchantPlaceState.test.mjs`

**Interfaces:**

- Consumes: `categories: Array<{ code: string, name: string }>`、`tags: string[]`。
- Produces: `buildMerchantTagLabels(categories)`，返回接受标签数组并返回中文展示数组的函数。

- [x] **Step 1: 写出失败测试**

```js
test('merchant tag labels translate configured codes and retain unmapped labels', () => {
  const labels = buildMerchantTagLabels([
    { code: 'girl', name: '女童' },
    { code: 'spot', name: '现货' },
  ])

  assert.deepEqual(labels(['girl', 'spot', '新标签']), ['女童', '现货', '新标签'])
})
```

- [x] **Step 2: 运行测试并确认失败**

Run: `npm test -- pages/sourcing-map/merchantPlaceState.test.mjs`
Expected: FAIL，提示 `buildMerchantTagLabels` 未导出。

- [x] **Step 3: 实现最小转换函数**

```js
export function buildMerchantTagLabels(categories = []) {
  const names = new Map(categories.map((category) => [String(category.code || '').trim(), String(category.name || '').trim()]))
  return (tags = []) => cleanList(tags).map((tag) => names.get(tag) || tag)
}
```

- [x] **Step 4: 运行测试并确认通过**

Run: `npm test -- pages/sourcing-map/merchantPlaceState.test.mjs`
Expected: PASS。

- [x] **Step 5: 提交本任务**

```bash
git add wxapp/pages/sourcing-map/merchantPlaceState.js wxapp/pages/sourcing-map/merchantPlaceState.test.mjs
git commit -m "fix: 本地化档口标签"
```

### Task 2: 列表页面使用中文展示标签

**Files:**

- Modify: `wxapp/pages/sourcing-map/index.vue`
- Modify: `wxapp/components/MerchantPlaceCard.vue`
- Test: `wxapp/pages/sourcing-map/index.test.mjs`
- Test: `wxapp/components/MerchantPlaceCard.test.mjs`

**Interfaces:**

- Consumes: `buildMerchantTagLabels(categories)` 返回的标签转换函数。
- Produces: 传给 `MerchantPlaceCard` 的 `place.displayTags: string[]`，卡片仅渲染该字段。

- [x] **Step 1: 写出失败测试**

```js
test('merchant directory loads category, service, and platform names for card labels', () => {
  assert.match(source, /\['booth_category', 'booth_service', 'platform_tag'\]/)
  assert.match(source, /displayTags/)
})
```

```js
test('merchant place card renders localized display tags instead of raw tag codes', () => {
  assert.match(source, /:tags="place\.displayTags"/)
  assert.doesNotMatch(source, /props\.place\.categoryCodes/)
})
```

- [x] **Step 2: 运行测试并确认失败**

Run: `npm test -- pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs`
Expected: FAIL，提示分类类型数组或 `displayTags` 尚不存在。

- [x] **Step 3: 实现最小页面接入**

页面并发请求 `booth_category`、`booth_service`、`platform_tag`；仅 `booth_category` 作为筛选列表，三类配置共同构建转换函数。列表规范化后，将分类、服务与平台标签转换为 `displayTags` 并取前三个。

- [x] **Step 4: 运行定向测试并确认通过**

Run: `npm test -- pages/sourcing-map/merchantPlaceState.test.mjs pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs`
Expected: PASS。

- [x] **Step 5: 提交本任务**

```bash
git add wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs wxapp/components/MerchantPlaceCard.vue wxapp/components/MerchantPlaceCard.test.mjs
git commit -m "fix: 展示中文档口标签"
```

### Task 3: 回归验证

**Files:**

- Verify: `wxapp/pages/sourcing-map/merchantPlaceState.test.mjs`
- Verify: `wxapp/pages/sourcing-map/index.test.mjs`
- Verify: `wxapp/components/MerchantPlaceCard.test.mjs`

**Interfaces:**

- Consumes: 已完成的标签转换与页面接入。
- Produces: 通过的定向测试和小程序构建结果。

- [x] **Step 1: 运行定向测试**

Run: `npm test -- pages/sourcing-map/merchantPlaceState.test.mjs pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs`
Expected: PASS，0 failures。

- [x] **Step 2: 构建微信小程序**

Run: `npm run build:mp-weixin`
Expected: exit code 0。

- [x] **Step 3: 检查改动范围**

Run: `git diff --check && git status --short`
Expected: 无空白错误，仅包含计划中的小程序源码与测试改动。
