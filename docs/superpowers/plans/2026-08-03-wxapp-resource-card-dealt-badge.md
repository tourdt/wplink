# 供需列表已成交角标实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 用图片右上角“已成交”角标替换供需列表的整卡印章水印，使资源状态更适合快速扫读。

**架构：** 只修改 `ResourceFeedCard` 模板、SCSS 和源码断言。继续使用现有 `cardModel.isCompleted` 作为已成交判断，不改数据模型；已成交时渲染右上角角标，并收窄左上类型标签的最大宽度避免重叠。

**技术栈：** Vue 3、uni-app、SCSS、Node.js `node:test`

## 全局约束

- `dealtAt` 有值时才显示“已成交”角标；未成交资源不显示状态角标。
- 删除 `feed-completed-stamp` 及所有印章子元素和样式。
- 不修改后端接口、状态机、页面调用、资源点击或类型标签数据来源。
- 类型标签必须在已成交时省略过长文案，且不得与右上角角标重叠。
- 先写失败测试并确认因新角标尚未实现而失败，再写最小实现。

## 文件结构

- 修改 `wxapp/components/ResourceFeedCard.test.mjs`：验证已成交角标、类型标签避让和旧印章移除。
- 修改 `wxapp/components/ResourceFeedCard.vue`：移除印章，恢复封面右上“已成交”角标并定义相应样式。

---

### 任务 1：替换已成交视觉状态为角标

**文件：**

- 修改：`wxapp/components/ResourceFeedCard.test.mjs`
- 修改：`wxapp/components/ResourceFeedCard.vue`

**接口：**

- 输入：现有 `cardModel.isCompleted: boolean`。
- 输出：已成交资源渲染 `<text class="feed-dealt-badge">已成交</text>`；类型标签带 `with-dealt` 类以收窄空间。
- 兼容性：继续使用 `<ResourceFeedCard :resource="item" @open="openResource" />` 和既有 `open(resource)` 事件。

- [ ] **步骤 1：编写失败测试**

将当前印章测试替换为：

```js
test('resource feed card gives dealt resources one prominent cover badge without a card watermark', () => {
  assert.match(
    source,
    /:class="\['feed-type-badge', \{ demand: cardModel\.isDemand, 'with-dealt': cardModel\.isCompleted \}\]"/,
  )
  assert.match(source, /<text v-if="cardModel\.isCompleted" class="feed-dealt-badge">已成交<\\/text>/)
  assert.match(source, /\.feed-type-badge\.with-dealt \{[\s\S]*max-width: calc\(100% - 99rpx\);/)
  assert.match(source, /\.feed-dealt-badge \{[\s\S]*height: 42rpx;[\s\S]*font-size: 21rpx;/)
  assert.doesNotMatch(source, /feed-completed-stamp/)
})
```

- [ ] **步骤 2：运行测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test components/ResourceFeedCard.test.mjs
```

预期：FAIL，错误指出新的已成交角标模板或样式不存在；不得是测试语法或路径错误。

- [ ] **步骤 3：实现最小角标模板与样式**

删除 `feed-completed-stamp` 模板和全部相关 SCSS。将类型角标改为：

```vue
<text
  :class="['feed-type-badge', { demand: cardModel.isDemand, 'with-dealt': cardModel.isCompleted }]"
>
  {{ cardModel.resourceTypeLabel || cardModel.directionLabel }}
</text>
<text v-if="cardModel.isCompleted" class="feed-dealt-badge">已成交</text>
```

在 `.feed-type-badge` 增加 `box-sizing: border-box`，并新增：

```scss
.feed-type-badge.with-dealt {
  max-width: calc(100% - 99rpx);
}

.feed-dealt-badge {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  height: 42rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  background: rgba(51, 65, 85, 0.94);
  color: #ffffff;
  font-size: 21rpx;
  font-weight: 700;
  line-height: 1;
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
git add wxapp/components/ResourceFeedCard.vue wxapp/components/ResourceFeedCard.test.mjs docs/superpowers/plans/2026-08-03-wxapp-resource-card-dealt-badge.md
git commit -m "feat: 显示供需已成交角标"
```
