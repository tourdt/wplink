# 供需列表已完成印章水印实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将共用供需列表卡片的普通已完成大字水印替换为浅色印章，并移除封面右上角的重复状态文字。

**架构：** 仅修改 `ResourceFeedCard` 及其源码断言。完成状态仍由 `cardModel.isCompleted` 驱动；模板以圆环、星形装饰和斜向矩形章面组合出纯 SCSS 印章，卡片内容保持在印章之上。

**技术栈：** Vue 3、uni-app、SCSS、Node.js `node:test`

## 全局约束

- 仅 `dealtAt` 有值的资源显示印章；未完成资源不显示印章。
- 删除封面右上角“已完成”文字及类型角标的完成状态宽度收窄逻辑。
- 印章使用 `pointer-events: none`，不得阻断既有 `open(resource)` 点击事件。
- 不新增图片资源、接口、数据字段或页面调用变更。
- 先写失败测试并确认因印章结构尚未实现而失败，再写最小实现。

## 文件结构

- 修改 `wxapp/components/ResourceFeedCard.test.mjs`：替换旧大字水印断言，约束印章结构、层级和已移除的右上角角标。
- 修改 `wxapp/components/ResourceFeedCard.vue`：渲染浅色圆章和斜向“已完成”章面，移除旧角标及无用样式。

---

### 任务 1：替换为浅色完成印章

**文件：**

- 修改：`wxapp/components/ResourceFeedCard.test.mjs`
- 修改：`wxapp/components/ResourceFeedCard.vue`

**接口：**

- 输入：现有 `cardModel.isCompleted: boolean`。
- 输出：完成资源渲染 `feed-completed-stamp`，其中包含圆环、星形装饰和“已完成”章面；未完成资源不渲染该元素。
- 兼容性：保留 `open(resource)`、`feed-type-badge` 与所有页面消费方式。

- [ ] **步骤 1：编写失败测试**

将 `ResourceFeedCard.test.mjs` 中现有水印测试替换为：

```js
test('resource feed card renders a subtle completed stamp without a duplicate cover badge', () => {
  assert.match(source, /<view v-if="cardModel\.isCompleted" class="feed-completed-stamp">/)
  assert.match(source, /class="feed-completed-stamp-ring"/)
  assert.match(source, /class="feed-completed-stamp-stars">✦<\/text>/)
  assert.match(source, /class="feed-completed-stamp-label">已完成<\/text>/)
  assert.match(source, /\.feed-completed-stamp \{[\s\S]*pointer-events: none;/)
  assert.match(source, /\.feed-completed-stamp-label \{[\s\S]*transform: rotate\(-13deg\);/)
  assert.doesNotMatch(source, /feed-completed-badge/)
  assert.doesNotMatch(source, /with-completed/)
})
```

- [ ] **步骤 2：运行测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test components/ResourceFeedCard.test.mjs
```

预期：FAIL，错误指出新的印章模板或样式不存在；不得是测试语法或路径错误。

- [ ] **步骤 3：实现最小印章模板与样式**

在卡片开头保留完成状态条件，并替换为：

```vue
<view v-if="cardModel.isCompleted" class="feed-completed-stamp">
  <view class="feed-completed-stamp-ring"></view>
  <text class="feed-completed-stamp-stars">✦</text>
  <text class="feed-completed-stamp-label">已完成</text>
</view>
```

移除模板中的 `feed-completed-badge`、类型标签上的 `'with-completed': cardModel.isCompleted` 和对应旧样式。类型标签保留：

```vue
:class="['feed-type-badge', { demand: cardModel.isDemand }]"
```

新增印章样式，使整体为淡灰蓝色、卡片中部偏右、不可交互；圆环位于章面后方，章面使用斜向矩形边框：

```scss
.feed-completed-stamp {
  position: absolute;
  top: 50%;
  left: 57%;
  z-index: 0;
  width: 164rpx;
  height: 122rpx;
  color: rgba(71, 85, 105, 0.24);
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.feed-completed-stamp-ring {
  position: absolute;
  top: 10rpx;
  left: 44rpx;
  width: 104rpx;
  height: 104rpx;
  border: 4rpx solid currentColor;
  border-radius: 50%;
}

.feed-completed-stamp-stars {
  position: absolute;
  top: 18rpx;
  left: 68rpx;
  font-size: 18rpx;
  letter-spacing: 18rpx;
}

.feed-completed-stamp-label {
  position: absolute;
  top: 40rpx;
  left: 0;
  display: grid;
  width: 164rpx;
  height: 56rpx;
  place-items: center;
  border: 4rpx solid currentColor;
  border-radius: 8rpx;
  background: rgba(255, 255, 255, 0.72);
  font-size: 40rpx;
  font-weight: 800;
  letter-spacing: 4rpx;
  transform: rotate(-13deg);
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
git add wxapp/components/ResourceFeedCard.vue wxapp/components/ResourceFeedCard.test.mjs docs/superpowers/plans/2026-08-02-wxapp-resource-card-completed-stamp.md
git commit -m "feat: 优化供需完成印章"
```
