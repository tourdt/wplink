# 供需详情同类推荐复用供需列表卡片实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让供需详情页的同类推荐使用供需市场同款 `ResourceFeedCard`。

**Architecture:** 详情页继续通过现有 `ResourceList` 渲染 `relatedResources`，只把展示变体由 `compact` 切换为 `feed`。`ResourceList` 已将 `feed` 变体映射到 `ResourceFeedCard`，因此无需修改推荐数据、跳转回调或公共组件。

**Tech Stack:** Vue 3、uni-app、Node.js 内置测试运行器。

## 全局约束

- 仅修改与同类推荐卡片样式直接相关的文件。
- 保留现有推荐查询、最多三条限制、`openRelatedResource` 回调和资源曝光行为。
- 不修改用户工作区内已存在的 `wxapp/components/ResourceFeedCard.vue` 与其测试改动。
- 测试先行：先确认新增断言在现状失败，再写最小页面改动使其通过。

---

### Task 1: 详情页切换同类推荐为供需市场卡片

**Files:**
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue:115-125`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

**Interfaces:**
- Consumes: `ResourceList` 的 `variant="feed"` 约定；该组件会渲染 `ResourceFeedCard`。
- Produces: 同类推荐区传入 `variant="feed"`，继续通过 `@open="openRelatedResource"` 打开推荐详情。

- [x] **Step 1: 写入失败测试**

在 `wxapp/pages/resource/detail.test.mjs` 的同类推荐相关断言旁新增测试，限定推荐区域使用 feed 变体并保留原有数据与跳转绑定：

```js
test('resource detail renders related resources with the market feed card variant', () => {
  assert.match(source, /<view v-if="relatedResources\\.length" class="related-section">[\\s\\S]*<ResourceList[\\s\\S]*:resources="relatedResources"[\\s\\S]*variant="feed"[\\s\\S]*@open="openRelatedResource"/)
})
```

- [x] **Step 2: 运行测试，确认当前失败**

Run: `npm --prefix wxapp test -- pages/resource/detail.test.mjs`

Expected: FAIL，因为详情页当前仍包含 `variant="compact"`，新增断言找不到 `variant="feed"`。

- [x] **Step 3: 写入最小页面改动**

在 `wxapp/pages/resource/detail.vue` 的同类推荐 `ResourceList` 上，将：

```vue
variant="compact"
```

替换为：

```vue
variant="feed"
```

不要更改 `:resources="relatedResources"`、`empty-text` 或 `@open="openRelatedResource"`。

- [x] **Step 4: 运行测试，确认通过**

Run: `npm --prefix wxapp test -- pages/resource/detail.test.mjs`

Expected: PASS，包含新增的同类推荐 feed 变体断言。

- [x] **Step 5: 运行页面结构校验**

Run: `npm --prefix wxapp run validate:pages`

Expected: PASS，详情页和页面配置校验无错误。

- [x] **Step 6: 提交本任务文件**

```bash
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "fix: 详情同类推荐复用供需卡片"
```
