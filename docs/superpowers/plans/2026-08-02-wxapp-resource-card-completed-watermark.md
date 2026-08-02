# 供需列表已完成水印实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 为首页和供需市场共用列表卡片中的已完成资源添加整卡低透明度水印，同时保留现有封面角标。

**架构：** 仅修改共用的 `ResourceFeedCard`。模板在 `cardModel.isCompleted` 时渲染一个不接收点击事件的水印元素；SCSS 将卡片设为定位容器，水印置于内容层下方，现有封面与信息层保持可读且可点击。

**技术栈：** Vue 3、uni-app、SCSS、Node.js `node:test`

## 全局约束

- 仅影响 `dealtAt` 有值的供给和需求资源；未完成资源不显示水印。
- 保留封面右上角的“已完成”角标、既有数据模型、点击事件和页面调用方式。
- 水印必须使用 `pointer-events: none`，不得拦截卡片点击。
- 不修改后端接口、数据库、资源状态规则、首页和供需市场页面逻辑。
- 先写失败测试并确认因水印尚未实现而失败，再写最小实现。

## 文件结构

- 修改 `wxapp/components/ResourceFeedCard.test.mjs`：约束完成状态水印的模板条件、层级和关键样式。
- 修改 `wxapp/components/ResourceFeedCard.vue`：在完成状态下渲染整卡水印，并以 scoped SCSS 定义低透明度斜向视觉和内容层级。

---

### 任务 1：为共用供需卡片添加已完成水印

**文件：**

- 修改：`wxapp/components/ResourceFeedCard.test.mjs`
- 修改：`wxapp/components/ResourceFeedCard.vue`

**接口：**

- 输入：现有 `resource` 对象，通过 `buildResourceFeedCardModel(resource).isCompleted` 识别 `dealtAt` 是否存在。
- 输出：完成资源的模板额外渲染 `<text class="feed-completed-watermark">已完成</text>`；未完成资源不渲染该元素。
- 兼容性：保留现有 `open(resource)` 事件、`feed-completed-badge` 封面角标及页面调用方式。

- [ ] **步骤 1：编写失败测试**

在 `wxapp/components/ResourceFeedCard.test.mjs` 末尾新增：

```js
test('resource feed card overlays completed resources with a readable non-interactive watermark', () => {
  assert.match(
    source,
    /<text v-if="cardModel\.isCompleted" class="feed-completed-watermark">已完成<\\/text>/,
  )
  assert.match(source, /\.resource-feed-card \{[\s\S]*position: relative;/)
  assert.match(source, /\.feed-completed-watermark \{[\s\S]*position: absolute;[\s\S]*pointer-events: none;/)
  assert.match(source, /\.feed-completed-watermark \{[\s\S]*font-size: 76rpx;[\s\S]*transform: translate\(-50%, -50%\) rotate\(-18deg\);/)
  assert.match(source, /\.feed-thumb-wrap,[\s\S]*\.feed-card-main \{[\s\S]*position: relative;[\s\S]*z-index: 1;/)
})
```

- [ ] **步骤 2：运行测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test components/ResourceFeedCard.test.mjs
```

预期：FAIL，错误指出水印模板或对应样式不存在；不得是 Node、测试路径或解析错误。

- [ ] **步骤 3：实现最小水印结构与样式**

在 `ResourceFeedCard.vue` 的顶层 `resource-feed-card` 内、`feed-thumb-wrap` 前加入：

```vue
<text v-if="cardModel.isCompleted" class="feed-completed-watermark">已完成</text>
```

将 `.resource-feed-card` 增加定位上下文：

```scss
position: relative;
```

为封面和信息内容建立高于水印的层级：

```scss
.feed-thumb-wrap,
.feed-card-main {
  position: relative;
  z-index: 1;
}
```

新增水印样式：

```scss
.feed-completed-watermark {
  position: absolute;
  top: 50%;
  left: 50%;
  z-index: 0;
  color: rgba(71, 85, 105, 0.14);
  font-size: 76rpx;
  font-weight: 800;
  line-height: 1;
  pointer-events: none;
  transform: translate(-50%, -50%) rotate(-18deg);
  white-space: nowrap;
}
```

- [ ] **步骤 4：运行组件测试确认通过**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test components/ResourceFeedCard.test.mjs
```

预期：PASS，所有 `ResourceFeedCard` 测试通过。

- [ ] **步骤 5：运行小程序完整检查**

运行：

```bash
cd wxapp
npm run check
```

预期：退出码为 `0`；页面静态校验、流程校验、Node 测试和微信小程序构建均通过。

- [ ] **步骤 6：提交实现**

```bash
git add wxapp/components/ResourceFeedCard.vue wxapp/components/ResourceFeedCard.test.mjs docs/superpowers/plans/2026-08-02-wxapp-resource-card-completed-watermark.md
git commit -m "feat: 突出供需已完成状态"
```
