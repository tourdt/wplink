# 小程序公开供需列表统一 ResourceFeedCard 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让搜索、收藏和专题页的公开供需信息统一使用 `ResourceFeedCard`。

**Architecture:** 三个页面继续保留各自的资源数据、曝光、筛选、分页、空状态和详情跳转流程，只替换渲染的卡片组件。`ResourceFeedCard` 通过既有 `resourceFeedCardState.js` 统一处理供应和需求展示，不增加页面专用数据转换。

**Tech Stack:** Vue 3、uni-app、Node.js 内置测试运行器。

## 全局约束

- 只覆盖搜索、收藏与专题的公开供需列表；“我的发布”管理卡片不改。
- 不修改 `ResourceFeedCard`、`resourceFeedCardState.js` 或资源接口。
- 保留 `ResourceExposure`、筛选、分页、空状态与详情跳转。
- 每一处页面变更均先更新测试并确认在旧卡片实现下失败。

---

### Task 1: 搜索页统一供应与需求卡片

**Files:**
- Modify: `wxapp/pages/search/index.test.mjs`
- Modify: `wxapp/pages/search/index.vue:124-129,166-170`

**Interfaces:**
- Consumes: `ResourceFeedCard` 的 `resource` 属性和 `open` 事件。
- Produces: 搜索结果在既有 `ResourceExposure` 内无方向分支地渲染 `ResourceFeedCard`。

- [x] **Step 1: 写入失败测试**

将“搜索页按方向渲染混合卡片”的断言替换为以下行为断言：

```js
test('search page renders mixed results with the shared feed card', () => {
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(source, /<ResourceExposure :resource-id="item\.id" source="search">[\s\S]*<ResourceFeedCard :resource="item" @open="openResource" \/>/)
  assert.doesNotMatch(source, /import DemandCard/)
  assert.doesNotMatch(source, /import ResourceCard/)
  assert.doesNotMatch(source, /RESOURCE_DIRECTION_DEMAND/)
})
```

- [x] **Step 2: 运行失败测试**

Run: `npm --prefix wxapp test -- pages/search/index.test.mjs`

Expected: FAIL，因为当前页面导入旧卡片，并按 `item.direction` 分支渲染。

- [x] **Step 3: 最小实现**

在 `wxapp/pages/search/index.vue`：

```vue
<ResourceExposure :resource-id="item.id" source="search">
  <ResourceFeedCard :resource="item" @open="openResource" />
</ResourceExposure>
```

将 `DemandCard`、`ResourceCard` 的导入替换为：

```js
import ResourceFeedCard from '../../components/ResourceFeedCard.vue'
```

并删除只服务于旧方向分支的 `RESOURCE_DIRECTION_DEMAND` 常量。

- [x] **Step 4: 运行搜索页测试**

Run: `npm --prefix wxapp test -- pages/search/index.test.mjs`

Expected: PASS，筛选、分页和统一卡片测试均通过。

### Task 2: 收藏页与专题页复用供需卡片

**Files:**
- Modify: `wxapp/pages/favorites/index.test.mjs`
- Modify: `wxapp/pages/favorites/index.vue:14-16,35-38`
- Modify: `wxapp/pages/topic/index.test.mjs`
- Modify: `wxapp/pages/topic/index.vue:26-29,37-40`

**Interfaces:**
- Consumes: `ResourceFeedCard(resource, @open)`。
- Produces: 收藏资源与专题资源保留现有循环和跳转，使用统一卡片。

- [x] **Step 1: 写入失败测试**

分别加入以下断言：

```js
test('favorites resource list uses the shared feed card', () => {
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(source, /<ResourceFeedCard v-for="item in favoriteResources" :key="item\.id" :resource="item" @open="openResource" \/>/)
  assert.doesNotMatch(source, /import ResourceCard/)
})

test('topic resources use the shared feed card inside exposure tracking', () => {
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(source, /<ResourceExposure v-for="item in rows" :key="item\.id" :resource-id="item\.id" source="topic">[\s\S]*<ResourceFeedCard :resource="item" @open="openResource" \/>/)
  assert.doesNotMatch(source, /import ResourceCard/)
})
```

- [x] **Step 2: 运行失败测试**

Run: `npm --prefix wxapp test -- pages/favorites/index.test.mjs pages/topic/index.test.mjs`

Expected: FAIL，因为两页仍导入和渲染 `ResourceCard`。

- [x] **Step 3: 最小实现**

将收藏页资源循环替换为：

```vue
<ResourceFeedCard v-for="item in favoriteResources" :key="item.id" :resource="item" @open="openResource" />
```

将专题页曝光容器内替换为：

```vue
<ResourceFeedCard :resource="item" @open="openResource" />
```

两页都将 `ResourceCard` 导入替换为 `ResourceFeedCard`；不改商家收藏区域和专题筛选/空状态。

- [x] **Step 4: 运行收藏与专题测试**

Run: `npm --prefix wxapp test -- pages/favorites/index.test.mjs pages/topic/index.test.mjs`

Expected: PASS，原有空状态测试与新增统一卡片测试通过。

### Task 3: 同步全量流程契约并验证

**Files:**
- Modify: `wxapp/scripts/validate-flows.test.mjs:220-240,330-350,420-435`

**Interfaces:**
- Consumes: 三个页面已使用 `ResourceFeedCard` 的公开列表结构。
- Produces: 全量流程测试不再把旧卡片作为搜索或专题页面的预期。

- [x] **Step 1: 写入失败契约测试**

更新流程测试的公开列表断言：搜索页期待 `ResourceFeedCard` 且排除 `DemandCard`、`ResourceCard`；专题页期待 `ResourceFeedCard`；收藏页资源标签期待 `ResourceFeedCard`，商家标签仍只校验商家列表。

- [x] **Step 2: 运行流程测试，确认旧断言失败**

Run: `npm --prefix wxapp test -- scripts/validate-flows.test.mjs`

Expected: FAIL，直到旧卡片相关契约全部替换为统一供需卡片契约。

- [x] **Step 3: 同步最小契约改动**

删除只验证 `ResourceCard` / `DemandCard` 旧结构的断言，改为验证页面导入并在现有循环或 `ResourceExposure` 内渲染 `ResourceFeedCard`；不改变其他 MVP 流程断言。

- [x] **Step 4: 运行验证**

Run: `npm --prefix wxapp run validate:pages && npm --prefix wxapp test`

Expected: 两个命令均成功，完整测试无失败。

- [x] **Step 5: 提交本任务文件**

```bash
git add docs/superpowers/plans/2026-08-03-wxapp-public-resource-lists-feed-card.md wxapp/pages/search/index.vue wxapp/pages/search/index.test.mjs wxapp/pages/favorites/index.vue wxapp/pages/favorites/index.test.mjs wxapp/pages/topic/index.vue wxapp/pages/topic/index.test.mjs wxapp/scripts/validate-flows.test.mjs
git commit -m "refactor: 统一公开供需列表卡片"
```
