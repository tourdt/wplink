# 小程序首页移除内容区眉题实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 移除首页内容发现区的“织里商机 · 持续更新”眉题，不改变栏目切换与内容加载行为。

**架构：** 该变更仅涉及 `home` 页面模板、仅服务该节点的样式，以及现有 Node 源码断言。删除 DOM 节点和孤立样式后，以负向断言防止文案回归；不改变任何状态、接口或组件边界。

**技术栈：** Vue 3（uni-app 单文件组件）、Node.js 内置 `node:test` / `node:assert`。

## 全局约束

- 只修改与首页眉题相关的模板、样式和测试。
- 保留 `homeFeedState`、`activeHomeFeedTab`、`selectHomeFeedTab()`、请求和卡片渲染逻辑。
- 不新增依赖、组件、接口或路由。

---

### 任务 1：首页内容区移除眉题并防回归

**文件：**
- 修改：`wxapp/pages/home/index.test.mjs:47-49, 108-118`
- 修改：`wxapp/pages/home/index.vue:89-91, 746-752`

**接口：**
- 消费：现有 `homeFeedReady && homeFeedState.hasAnyContent` 内容区条件。
- 产出：首页直接从栏目标题行开始渲染，源码中不存在 `织里商机 · 持续更新` 或 `.home-feed-kicker`。

- [ ] **步骤 1：编写失败测试**

将既有正向断言替换为以下断言，并添加样式类断言：

```js
assert.doesNotMatch(source, /织里商机 · 持续更新/)
assert.doesNotMatch(source, /home-feed-kicker/)
```

- [ ] **步骤 2：运行测试并确认失败**

运行：`node --test wxapp/pages/home/index.test.mjs`

预期：失败，断言会在仍存在的眉题文案或样式类上命中。

- [ ] **步骤 3：实现最小修改**

从内容区删除以下模板节点：

```vue
<text class="home-feed-kicker">织里商机 · 持续更新</text>
```

删除唯一引用 `.home-feed-kicker` 的样式块，保留其余 `home-feed-*` 样式和栏目标题行。

- [ ] **步骤 4：运行定向测试并确认通过**

运行：`node --test wxapp/pages/home/index.test.mjs`

预期：全部通过。

- [ ] **步骤 5：检查变更范围并提交**

运行：`git diff --check -- wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs`

运行：`git add wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs && git commit -m "fix: 移除首页内容区眉题"`

预期：仅提交首页模板和其定向测试；不包含工作区中已有的无关改动。
