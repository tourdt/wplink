# 小程序首页内容标签轻量化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将首页“新入驻商家 / 近期供需”从厚重的等宽分段按钮改为左对齐栏目标签，同时保持现有数据与切换行为不变。

**架构：** 不新增组件或状态，只修改首页模板的选项卡语义和同文件内的 SCSS。现有 `activeHomeFeedTab`、`selectHomeFeedTab()`、`homeFeedState` 与 `v-if / v-else` 内容渲染保持原样，测试通过读取组件源码验证模板与样式约束。

**技术栈：** Vue 3、uni-app、SCSS、微信小程序、Node.js `node:test`

## 全局约束

- 两个标签左对齐排列，间距固定为 `36rpx`。
- 标签区底部使用浅色分隔线，选中标签使用主色文字和 `4rpx` 高的文字同宽指示线。
- 移除外层灰色底、描边、圆角分段块和选中阴影。
- 每个标签继续保留至少 `88rpx` 的点击高度。
- 按下态只调整透明度，不增加缩放、位移或切换动画。
- 使用 `role="tablist"`、`role="tab"` 和动态 `aria-selected` 表达选项卡状态。
- 不修改标签文案、接口、数据排序、数量、默认标签、降级逻辑、卡片样式或跳转。
- 先写失败测试并确认因目标视觉行为缺失而失败，再修改生产代码。

## 文件结构

- 修改 `wxapp/pages/home/index.test.mjs`：验证选项卡语义、左对齐布局、轻量选中态和触控尺寸。
- 修改 `wxapp/pages/home/index.vue`：为现有标签补充语义，并将等宽分段样式替换为左对齐栏目样式。

---

### 任务 1：首页双标签改为左对齐轻量栏目

**文件：**

- 修改：`wxapp/pages/home/index.test.mjs:41-54`
- 修改：`wxapp/pages/home/index.vue:89-99`
- 修改：`wxapp/pages/home/index.vue:728-762`

**接口：**

- 输入：现有 `activeHomeFeedTab: Ref<'merchants' | 'resources' | ''>` 与 `homeFeedState.showSwitcher`。
- 输出：模板继续调用 `selectHomeFeedTab('merchants' | 'resources')`；标签容器提供 `role="tablist"`，标签按钮提供 `role="tab"` 和布尔 `aria-selected`。
- 不变：`selectHomeFeedTab()`、接口请求、列表渲染与跳转函数均不修改。

- [ ] **步骤 1：编写轻量栏目样式的失败测试**

在 `wxapp/pages/home/index.test.mjs` 的 `home combines recent merchants and resources into a local tabbed feed` 测试中，在已有行为断言后加入以下内容：

```js
  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const merchantTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', \{ active: activeHomeFeedTab === 'merchants' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const resourceTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', \{ active: activeHomeFeedTab === 'resources' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(merchantTab, /role="tab"/)
  assert.match(merchantTab, /:aria-selected="activeHomeFeedTab === 'merchants'"/)
  assert.match(merchantTab, /selectHomeFeedTab\('merchants'\)/)
  assert.match(resourceTab, /role="tab"/)
  assert.match(resourceTab, /:aria-selected="activeHomeFeedTab === 'resources'"/)
  assert.match(resourceTab, /selectHomeFeedTab\('resources'\)/)

  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabStyle = source.match(/\.home-feed-tab\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeTabStyle = source.match(/\.home-feed-tab\.active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const indicatorStyle = source.match(/\.home-feed-tab::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeIndicatorStyle = source.match(/\.home-feed-tab\.active::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedTabStyle = source.match(/\.home-feed-tab:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /gap:\s*36rpx/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid/)
  assert.doesNotMatch(tabsStyle, /grid-template-columns|background:|border-radius:|box-shadow:/)
  assert.match(tabStyle, /position:\s*relative/)
  assert.match(tabStyle, /width:\s*auto/)
  assert.match(tabStyle, /min-height:\s*88rpx/)
  assert.match(tabStyle, /background:\s*transparent/)
  assert.match(activeTabStyle, /color:\s*\$wplink-primary/)
  assert.doesNotMatch(activeTabStyle, /background:\s*#ffffff|box-shadow:\s*0\s+6rpx/)
  assert.match(indicatorStyle, /height:\s*4rpx/)
  assert.match(indicatorStyle, /right:\s*2rpx/)
  assert.match(indicatorStyle, /left:\s*2rpx/)
  assert.match(indicatorStyle, /background:\s*transparent/)
  assert.match(activeIndicatorStyle, /background:\s*\$wplink-primary/)
  assert.match(pressedTabStyle, /opacity:\s*0\.72/)
  assert.doesNotMatch(`${tabStyle}\n${activeTabStyle}\n${pressedTabStyle}`, /transform:|transition:|animation:/)
```

删除该测试中原来单独检查 `.home-feed-tab` 高度的这一行，避免重复断言：

```js
  assert.match(source, /\.home-feed-tab\s*\{[\s\S]*min-height:\s*88rpx/)
```

- [ ] **步骤 2：运行定向测试并确认失败原因正确**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：FAIL。首次失败应指出模板缺少 `role="tablist"`，或样式仍是 `display: grid` 而不是 `display: flex`；不能是语法错误或测试文件路径错误。

- [ ] **步骤 3：补充选项卡语义并实现最小样式改动**

将 `wxapp/pages/home/index.vue` 的双标签模板改为：

```vue
        <view
          v-if="homeFeedState.showSwitcher"
          class="home-feed-tabs"
          role="tablist"
          aria-label="首页内容"
        >
          <button
            :class="['home-feed-tab', { active: activeHomeFeedTab === 'merchants' }]"
            role="tab"
            :aria-selected="activeHomeFeedTab === 'merchants'"
            @click="selectHomeFeedTab('merchants')"
          >新入驻商家</button>
          <button
            :class="['home-feed-tab', { active: activeHomeFeedTab === 'resources' }]"
            role="tab"
            :aria-selected="activeHomeFeedTab === 'resources'"
            @click="selectHomeFeedTab('resources')"
          >近期供需</button>
        </view>
```

将现有 `.home-feed-tabs`、`.home-feed-tab`、`.home-feed-tab.active` 和按钮伪元素规则替换为：

```scss
.home-feed-tabs {
  display: flex;
  align-items: stretch;
  gap: 36rpx;
  margin: 0 0 20rpx;
  border-bottom: 1rpx solid rgba(148, 163, 184, 0.28);
}

.home-feed-tab {
  position: relative;
  width: auto;
  min-height: 88rpx;
  margin: 0;
  padding: 16rpx 2rpx;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: $wplink-muted;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 1.3;
  box-shadow: none;
}

.home-feed-tab.active {
  background: transparent;
  color: $wplink-primary;
  box-shadow: none;
}

.home-feed-tab::after {
  position: absolute;
  right: 2rpx;
  bottom: -1rpx;
  left: 2rpx;
  height: 4rpx;
  border: 0;
  border-radius: 999rpx 999rpx 0 0;
  background: transparent;
  content: '';
}

.home-feed-tab.active::after {
  background: $wplink-primary;
}

.home-feed-tab:active {
  opacity: 0.72;
}

.home-feed-more::after {
  border: 0;
}
```

不要修改 `selectHomeFeedTab()`、`homeFeedState`、列表组件或单内容标题。

- [ ] **步骤 4：运行定向测试并确认通过**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：PASS，首页测试全部通过，无错误和警告。

- [ ] **步骤 5：运行小程序全量校验**

在 `wxapp` 目录运行：

```bash
npm run check
```

预期：页面校验、流程校验、全量测试和 `mp-weixin` 构建全部通过。

- [ ] **步骤 6：检查改动范围并提交**

运行：

```bash
git diff --check
git diff -- wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git status --short
```

确认只修改首页组件和对应测试后提交：

```bash
git add wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git commit -m "style: 轻量化首页内容标签"
```
