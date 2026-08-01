# 小程序首页织带式内容栏目实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将首页“新入驻商家 / 近期供需”从纯文字下划线标签改为带栏目眉题的织带式栏目签，并让单内容状态使用相同的静态栏目语言。

**架构：** 不新增组件、图片或状态，只修改 `wxapp/pages/home/index.vue` 的栏目头模板和同文件 SCSS。新增 `.home-feed-woven-label` 作为按钮标签与静态标签共享的视觉基础类，现有 `activeHomeFeedTab`、`selectHomeFeedTab()`、`homeFeedState` 和内容列表渲染保持不变。

**技术栈：** Vue 3、uni-app、SCSS、微信小程序、Node.js `node:test`

## 全局约束

- 栏目眉题固定为 `织里商机 · 持续更新`，字号 `22rpx`、字重 `600`、字间距 `1rpx`，颜色为 `$wplink-muted`。
- 标签组使用横向 `flex`，标签间距 `18rpx`，上下边框均为 `1rpx solid $wplink-line`，与列表间距 `20rpx`。
- 两个标签均使用 `min-height: 88rpx` 和 `padding: 0 28rpx`；选中状态不得修改 padding、字号或字重。
- 当前标签使用 `$wplink-primary` 背景、白色文字、`4rpx × 32rpx` 的 `$wplink-accent` 左侧缝线和 `14rpx` 右侧 CSS 边框三角形。
- 不使用 `clip-path`、图片资源、圆角外框、胶囊底、阴影、缩放、位移或循环动画。
- 按下态只将可点击标签透明度调整为 `0.82`。
- 双内容状态保留 `role="tablist"`、`role="tab"`、动态 `aria-selected` 和原有点击函数。
- 单内容状态使用静态 `view` 织带签，不提供点击事件或选项卡语义。
- 不修改接口、请求时机、排序、数量、默认标签、降级逻辑、列表卡片和跳转。
- 先写失败测试并确认因织带栏目行为缺失而失败，再修改生产代码。
- 全量校验允许仓库既有 Node.js ExperimentalWarning、Dart Sass 弃用提示和 uni-app 版本提示，但不得新增其他警告。

## 文件结构

- 修改 `wxapp/pages/home/index.test.mjs`：验证栏目眉题、双标签语义、织带视觉、固定几何和单内容静态标签。
- 修改 `wxapp/pages/home/index.vue`：增加栏目眉题，将双标签和单内容标题统一为织带式栏目签。

---

### 任务 1：首页内容区接入织带式栏目签

**文件：**

- 修改：`wxapp/pages/home/index.test.mjs:41-95`
- 修改：`wxapp/pages/home/index.vue:89-112`
- 修改：`wxapp/pages/home/index.vue:737-793`

**接口：**

- 输入：现有 `homeFeedState.showSwitcher`、`homeFeedState.hasMerchants` 和 `activeHomeFeedTab`。
- 输出：双内容时继续调用 `selectHomeFeedTab('merchants' | 'resources')`；单内容时只输出静态栏目名称。
- 共享视觉类：`.home-feed-woven-label`；当前状态类：`.active`。
- 不变：`selectHomeFeedTab()`、接口请求、列表 `v-if / v-else`、详情和更多入口均不修改。

- [ ] **步骤 1：编写织带栏目行为的失败测试**

将 `wxapp/pages/home/index.test.mjs` 中的 `home combines recent merchants and resources into a local tabbed feed` 测试替换为：

```js
test('home presents recent merchants and resources as a woven business channel', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.match(source, />织里商机 · 持续更新</)
  assert.match(source, />新入驻商家</)
  assert.match(source, />近期供需</)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-else class="home-resource-panel"/)
  assert.doesNotMatch(source, /v-show="activeHomeFeedTab ===/)
  assert.match(source, /\.home-feed-more\s*\{[\s\S]*min-height:\s*88rpx/)

  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const merchantTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', 'home-feed-woven-label', \{ active: activeHomeFeedTab === 'merchants' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const resourceTab = source.match(
    /<button\n\s+:class="\['home-feed-tab', 'home-feed-woven-label', \{ active: activeHomeFeedTab === 'resources' \}\]"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const singleHead = source.match(
    /<view v-else class="home-feed-single-head home-feed-woven-label active">[\s\S]*?<\/view>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(merchantTab, /role="tab"/)
  assert.match(merchantTab, /:aria-selected="activeHomeFeedTab === 'merchants'"/)
  assert.match(merchantTab, /selectHomeFeedTab\('merchants'\)/)
  assert.match(resourceTab, /role="tab"/)
  assert.match(resourceTab, /:aria-selected="activeHomeFeedTab === 'resources'"/)
  assert.match(resourceTab, /selectHomeFeedTab\('resources'\)/)
  assert.match(singleHead, /homeFeedState\.hasMerchants \? '新入驻商家' : '近期供需'/)
  assert.doesNotMatch(singleHead, /role="tab"|aria-selected|@click/)

  const kickerStyle = source.match(/\.home-feed-kicker\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const labelStyle = source.match(/\.home-feed-woven-label\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeStyle = source.match(/\.home-feed-woven-label\.active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const seamStyle = source.match(/\.home-feed-woven-label::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeSeamStyle = source.match(/\.home-feed-woven-label\.active::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const activeNotchStyle = source.match(/\.home-feed-woven-label\.active::after\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedTabStyle = source.match(/\.home-feed-tab:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(kickerStyle, /margin-bottom:\s*12rpx/)
  assert.match(kickerStyle, /font-size:\s*22rpx/)
  assert.match(kickerStyle, /font-weight:\s*600/)
  assert.match(kickerStyle, /letter-spacing:\s*1rpx/)
  assert.match(kickerStyle, /color:\s*\$wplink-muted/)
  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /gap:\s*18rpx/)
  assert.match(tabsStyle, /border-top:\s*1rpx solid \$wplink-line/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(tabsStyle, /background:|border-radius:|box-shadow:/)
  assert.match(labelStyle, /display:\s*inline-flex/)
  assert.match(labelStyle, /min-height:\s*88rpx/)
  assert.match(labelStyle, /padding:\s*0 28rpx/)
  assert.match(labelStyle, /font-size:\s*27rpx/)
  assert.match(labelStyle, /font-weight:\s*700/)
  assert.match(labelStyle, /overflow:\s*visible/)
  assert.match(labelStyle, /white-space:\s*nowrap/)
  assert.match(activeStyle, /background:\s*\$wplink-primary/)
  assert.match(activeStyle, /color:\s*#ffffff/)
  assert.doesNotMatch(activeStyle, /padding:|font-size:|font-weight:/)
  assert.match(seamStyle, /top:\s*28rpx/)
  assert.match(seamStyle, /width:\s*4rpx/)
  assert.match(seamStyle, /height:\s*32rpx/)
  assert.match(activeSeamStyle, /background:\s*\$wplink-accent/)
  assert.match(activeNotchStyle, /right:\s*-14rpx/)
  assert.match(activeNotchStyle, /border-top:\s*44rpx solid transparent/)
  assert.match(activeNotchStyle, /border-bottom:\s*44rpx solid transparent/)
  assert.match(activeNotchStyle, /border-left:\s*14rpx solid \$wplink-primary/)
  assert.doesNotMatch(activeNotchStyle, /clip-path:/)
  assert.match(pressedTabStyle, /opacity:\s*0\.82/)
  assert.doesNotMatch(
    `${labelStyle}\n${activeStyle}\n${pressedTabStyle}`,
    /transform:|transition:|animation:|box-shadow:/,
  )
})
```

- [ ] **步骤 2：运行定向测试并确认失败原因正确**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：FAIL。首次失败应指出缺少“织里商机 · 持续更新”或缺少 `.home-feed-woven-label`，不能是语法错误或测试文件路径错误。

- [ ] **步骤 3：实现栏目眉题、双织带标签和静态单标签**

将 `wxapp/pages/home/index.vue` 的首页内容区头部改为：

```vue
      <view v-if="homeFeedReady && homeFeedState.hasAnyContent" class="home-feed-section">
        <text class="home-feed-kicker">织里商机 · 持续更新</text>

        <view
          v-if="homeFeedState.showSwitcher"
          class="home-feed-tabs"
          role="tablist"
          aria-label="首页内容"
        >
          <button
            :class="['home-feed-tab', 'home-feed-woven-label', { active: activeHomeFeedTab === 'merchants' }]"
            role="tab"
            :aria-selected="activeHomeFeedTab === 'merchants'"
            @click="selectHomeFeedTab('merchants')"
          >新入驻商家</button>
          <button
            :class="['home-feed-tab', 'home-feed-woven-label', { active: activeHomeFeedTab === 'resources' }]"
            role="tab"
            :aria-selected="activeHomeFeedTab === 'resources'"
            @click="selectHomeFeedTab('resources')"
          >近期供需</button>
        </view>

        <view v-else class="home-feed-single-head home-feed-woven-label active">
          <text>{{ homeFeedState.hasMerchants ? '新入驻商家' : '近期供需' }}</text>
        </view>
```

保留该区块后续的商家和供需列表模板不变。

将现有 `.home-feed-tabs`、`.home-feed-tab`、选中态、伪元素和单标题规则替换为：

```scss
.home-feed-kicker {
  display: block;
  margin-bottom: 12rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: 1rpx;
}

.home-feed-tabs {
  display: flex;
  align-items: stretch;
  gap: 18rpx;
  margin: 0 0 20rpx;
  border-top: 1rpx solid $wplink-line;
  border-bottom: 1rpx solid $wplink-line;
}

.home-feed-woven-label {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: auto;
  min-height: 88rpx;
  margin: 0;
  padding: 0 28rpx;
  overflow: visible;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: $wplink-muted;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 1.3;
  white-space: nowrap;
  box-shadow: none;
}

.home-feed-woven-label.active {
  background: $wplink-primary;
  color: #ffffff;
}

.home-feed-woven-label::before,
.home-feed-woven-label::after {
  position: absolute;
  border: 0;
  content: '';
}

.home-feed-woven-label::before {
  top: 28rpx;
  left: 14rpx;
  width: 4rpx;
  height: 32rpx;
  border-radius: 999rpx;
  background: transparent;
}

.home-feed-woven-label.active::before {
  background: $wplink-accent;
}

.home-feed-woven-label.active::after {
  top: 0;
  right: -14rpx;
  width: 0;
  height: 0;
  border-top: 44rpx solid transparent;
  border-bottom: 44rpx solid transparent;
  border-left: 14rpx solid $wplink-primary;
}

.home-feed-tab:active {
  opacity: 0.82;
}

.home-feed-more::after {
  border: 0;
}

.home-feed-single-head {
  margin: 0 0 20rpx;
}
```

`top: 28rpx` 根据标签高度 `88rpx` 和缝线高度 `32rpx` 计算，使缝线垂直居中，同时完全避免 transform。不要修改任何脚本逻辑。

- [ ] **步骤 4：运行定向测试并确认通过**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：PASS，首页定向测试全部通过；除仓库既有 Node.js ExperimentalWarning 外不出现新增警告。

- [ ] **步骤 5：运行小程序全量校验**

在 `wxapp` 目录运行：

```bash
npm run check
```

预期：页面校验、流程校验、291 项以上测试和 `mp-weixin` 构建全部通过；仅允许全局约束中列明的既有提示。

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
git commit -m "style: 优化首页织带式内容栏目"
```
