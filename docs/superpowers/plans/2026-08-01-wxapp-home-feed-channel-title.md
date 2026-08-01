# 小程序首页栏目标题式切换实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将首页“新入驻商家 / 近期供需”从织带式双标签改为“当前栏目标题 + 另一频道入口”，并在单内容状态下使用同一套静态栏目标题语言。

**架构：** 不新增组件、状态或图片资源，只修改 `wxapp/pages/home/index.vue` 的栏目头模板和同文件 SCSS。双内容状态直接按视觉顺序渲染当前 Tab 和另一 Tab，标签文案与点击目标由 `activeHomeFeedTab` 内联决定；现有 `homeFeedState`、`selectHomeFeedTab()`、请求流程和列表渲染保持不变。

**技术栈：** Vue 3、uni-app、SCSS、微信小程序、Node.js `node:test`

## 全局约束

- 栏目眉题固定为 `织里商机 · 持续更新`，字号 `22rpx`、字重 `600`、字间距 `1rpx`、颜色 `$wplink-muted`，与栏目标题行间距 `8rpx`。
- 标题行使用横向 `flex` 和两端对齐，最小高度 `88rpx`，底部为 `1rpx solid $wplink-line`，与首个内容项间距 `20rpx`。
- 当前栏目始终在左侧，字号 `32rpx`、字重 `800`、颜色 `$wplink-primary`，左侧内边距 `14rpx`，并显示 `4rpx × 32rpx` 的 `$wplink-accent` 左标记。
- 另一栏目始终在右侧，字号 `25rpx`、字重 `700`、颜色 `$wplink-muted`，名称后显示 `→`，间距 `8rpx`。
- 切换后两个栏目名称交换位置；模板先输出当前 Tab，再输出另一 Tab，视觉顺序与 Tab 语义顺序一致。
- 不使用胶囊、卡片底、实色大色块、织带切角、滑块、阴影、`clip-path`、缩放、位移或切换动画。
- 仅另一频道入口在按下时使用 `opacity: 0.72`；当前标题重复点击不触发请求或额外视觉反馈。
- 双内容状态保留 `role="tablist"`、`role="tab"`、`aria-selected` 和原有 `selectHomeFeedTab()`；切换只更新本地状态。
- 单内容状态使用静态 `view` 栏目标题，不显示右侧入口、箭头、点击事件或 Tab 语义。
- 不修改接口、请求时机、排序、数量、默认栏目、降级逻辑、列表卡片、“查看更多”入口和跳转。
- 沿用仓库现有首页源码断言方式完成 TDD，不引入新的组件渲染测试基础设施。
- 全量校验允许仓库既有 Node.js ExperimentalWarning、Dart Sass 弃用提示和 uni-app 版本提示，但不得新增其他警告。

## 文件结构

- 修改 `wxapp/pages/home/index.test.mjs`：验证动态栏目换位、双 Tab 语义、静态单栏目、标题式视觉和禁止项。
- 修改 `wxapp/pages/home/index.vue`：将织带标签替换为栏目标题与另一频道入口，保留现有数据和内容模板。

---

### Task 1：首页内容区接入栏目标题式切换

**文件：**

- 修改：`wxapp/pages/home/index.test.mjs:41-117`
- 修改：`wxapp/pages/home/index.vue:89-114`
- 修改：`wxapp/pages/home/index.vue:739-823`

**接口：**

- 输入：现有 `homeFeedState.showSwitcher`、`homeFeedState.hasMerchants` 和 `activeHomeFeedTab`。
- 输出：双内容时继续调用 `selectHomeFeedTab('merchants' | 'resources')`；单内容时只输出静态栏目名称。
- 视觉类：`.home-feed-channel-title` 表达当前栏目，`.home-feed-channel-switch` 表达另一频道入口，`.home-feed-channel-arrow` 表达同级切换箭头。
- 不变：`selectHomeFeedTab()`、接口请求、列表 `v-if / v-else`、详情与更多入口均不修改。

- [ ] **Step 1：编写栏目标题式切换的失败测试**

将 `wxapp/pages/home/index.test.mjs` 中的 `home presents recent merchants and resources as a woven business channel` 测试替换为：

```js
test('home presents the active feed as an editorial channel title', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.match(source, />织里商机 · 持续更新</)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list"/)
  assert.match(source, /v-else class="home-resource-panel"/)
  assert.doesNotMatch(source, /v-show="activeHomeFeedTab ===/)
  assert.match(source, /\.home-feed-more\s*\{[\s\S]*min-height:\s*88rpx/)

  const tabListTag = source.match(/<view\s+[^>]*class="home-feed-tabs"[^>]*>/)?.[0] || ''
  const activeTab = source.match(
    /<button\n\s+class="home-feed-tab home-feed-channel-title"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const switchTab = source.match(
    /<button\n\s+class="home-feed-tab home-feed-channel-switch"[\s\S]*?<\/button>/,
  )?.[0] || ''
  const singleHead = source.match(
    /<view v-else class="home-feed-single-head">[\s\S]*?<\/view>\n\s*<\/view>/,
  )?.[0] || ''

  assert.match(tabListTag, /role="tablist"/)
  assert.match(tabListTag, /aria-label="首页内容"/)
  assert.match(activeTab, /role="tab"/)
  assert.match(activeTab, /aria-selected="true"/)
  assert.match(activeTab, /selectHomeFeedTab\(activeHomeFeedTab\)/)
  assert.match(activeTab, /activeHomeFeedTab === 'merchants' \? '新入驻商家' : '近期供需'/)
  assert.match(switchTab, /role="tab"/)
  assert.match(switchTab, /aria-selected="false"/)
  assert.match(
    switchTab,
    /selectHomeFeedTab\(activeHomeFeedTab === 'merchants' \? 'resources' : 'merchants'\)/,
  )
  assert.match(switchTab, /activeHomeFeedTab === 'merchants' \? '近期供需' : '新入驻商家'/)
  assert.match(switchTab, /class="home-feed-channel-arrow" aria-hidden="true">→<\/text>/)
  assert.match(singleHead, /class="home-feed-channel-title"/)
  assert.match(singleHead, /homeFeedState\.hasMerchants \? '新入驻商家' : '近期供需'/)
  assert.doesNotMatch(singleHead, /role="tab"|aria-selected|@click|home-feed-channel-arrow/)
  assert.doesNotMatch(source, /home-feed-woven-label/)

  const kickerStyle = source.match(/\.home-feed-kicker\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabsStyle = source.match(/\.home-feed-tabs\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const tabStyle = source.match(/\.home-feed-tab\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const titleStyle = source.match(/\.home-feed-channel-title\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const markerStyle = source.match(/\.home-feed-channel-title::before\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const switchStyle = source.match(/\.home-feed-channel-switch\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const arrowStyle = source.match(/\.home-feed-channel-arrow\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const pressedSwitchStyle = source.match(/\.home-feed-channel-switch:active\s*\{([\s\S]*?)\n\}/)?.[1] || ''
  const singleHeadStyle = source.match(/\.home-feed-single-head\s*\{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(kickerStyle, /margin-bottom:\s*8rpx/)
  assert.match(kickerStyle, /font-size:\s*22rpx/)
  assert.match(kickerStyle, /font-weight:\s*600/)
  assert.match(kickerStyle, /letter-spacing:\s*1rpx/)
  assert.match(kickerStyle, /color:\s*\$wplink-muted/)
  assert.match(tabsStyle, /display:\s*flex/)
  assert.match(tabsStyle, /align-items:\s*stretch/)
  assert.match(tabsStyle, /justify-content:\s*space-between/)
  assert.match(tabsStyle, /gap:\s*18rpx/)
  assert.match(tabsStyle, /min-height:\s*88rpx/)
  assert.match(tabsStyle, /margin:\s*0 0 20rpx/)
  assert.match(tabsStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(tabsStyle, /border-top:|background:|border-radius:|box-shadow:/)
  assert.match(tabStyle, /display:\s*inline-flex/)
  assert.match(tabStyle, /min-height:\s*88rpx/)
  assert.match(tabStyle, /white-space:\s*nowrap/)
  assert.match(titleStyle, /position:\s*relative/)
  assert.match(titleStyle, /padding:\s*0 0 0 14rpx/)
  assert.match(titleStyle, /font-size:\s*32rpx/)
  assert.match(titleStyle, /font-weight:\s*800/)
  assert.match(titleStyle, /color:\s*\$wplink-primary/)
  assert.match(markerStyle, /top:\s*28rpx/)
  assert.match(markerStyle, /left:\s*0/)
  assert.match(markerStyle, /width:\s*4rpx/)
  assert.match(markerStyle, /height:\s*32rpx/)
  assert.match(markerStyle, /background:\s*\$wplink-accent/)
  assert.match(switchStyle, /justify-content:\s*flex-end/)
  assert.match(switchStyle, /gap:\s*8rpx/)
  assert.match(switchStyle, /font-size:\s*25rpx/)
  assert.match(switchStyle, /font-weight:\s*700/)
  assert.match(switchStyle, /color:\s*\$wplink-muted/)
  assert.match(arrowStyle, /font-size:\s*24rpx/)
  assert.match(pressedSwitchStyle, /opacity:\s*0\.72/)
  assert.match(singleHeadStyle, /display:\s*flex/)
  assert.match(singleHeadStyle, /min-height:\s*88rpx/)
  assert.match(singleHeadStyle, /margin:\s*0 0 20rpx/)
  assert.match(singleHeadStyle, /border-bottom:\s*1rpx solid \$wplink-line/)
  assert.doesNotMatch(
    `${tabsStyle}\n${tabStyle}\n${titleStyle}\n${switchStyle}\n${pressedSwitchStyle}`,
    /transform:|transition:|animation:|box-shadow:|clip-path:/,
  )
})
```

该测试要捕获的生产缺陷是：首页仍使用织带式双标签，无法表达“当前内容是栏目标题、另一项才是切换入口”的信息层级。

- [ ] **Step 2：运行定向测试并确认失败原因正确**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：FAIL。首次失败应指出缺少 `.home-feed-channel-title`、`.home-feed-channel-switch` 或仍存在 `home-feed-woven-label`，不能是语法错误或测试文件路径错误。

- [ ] **Step 3：实现动态栏目标题、另一频道入口和静态单栏目**

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
            class="home-feed-tab home-feed-channel-title"
            role="tab"
            aria-selected="true"
            @click="selectHomeFeedTab(activeHomeFeedTab)"
          >
            <text>{{ activeHomeFeedTab === 'merchants' ? '新入驻商家' : '近期供需' }}</text>
          </button>
          <button
            class="home-feed-tab home-feed-channel-switch"
            role="tab"
            aria-selected="false"
            @click="selectHomeFeedTab(activeHomeFeedTab === 'merchants' ? 'resources' : 'merchants')"
          >
            <text>{{ activeHomeFeedTab === 'merchants' ? '近期供需' : '新入驻商家' }}</text>
            <text class="home-feed-channel-arrow" aria-hidden="true">→</text>
          </button>
        </view>

        <view v-else class="home-feed-single-head">
          <view class="home-feed-channel-title">
            <text>{{ homeFeedState.hasMerchants ? '新入驻商家' : '近期供需' }}</text>
          </view>
        </view>
```

保留该区块后续的商家和供需列表模板不变。不要修改任何脚本逻辑。

将现有 `.home-feed-kicker`、`.home-feed-tabs`、织带标签、伪元素、按下态和单标题规则替换为：

```scss
.home-feed-kicker {
  display: block;
  margin-bottom: 8rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: 1rpx;
}

.home-feed-tabs {
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  gap: 18rpx;
  min-height: 88rpx;
  margin: 0 0 20rpx;
  border-bottom: 1rpx solid $wplink-line;
}

.home-feed-tab {
  display: inline-flex;
  align-items: center;
  width: auto;
  min-height: 88rpx;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  line-height: 1.2;
  white-space: nowrap;
}

.home-feed-channel-title {
  position: relative;
  display: inline-flex;
  align-items: center;
  min-height: 88rpx;
  padding: 0 0 0 14rpx;
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 800;
  line-height: 1.2;
  white-space: nowrap;
}

.home-feed-channel-title::before {
  position: absolute;
  top: 28rpx;
  left: 0;
  width: 4rpx;
  height: 32rpx;
  background: $wplink-accent;
  content: '';
}

.home-feed-channel-switch {
  justify-content: flex-end;
  gap: 8rpx;
  color: $wplink-muted;
  font-size: 25rpx;
  font-weight: 700;
  text-align: right;
}

.home-feed-channel-arrow {
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1;
}

.home-feed-channel-switch:active {
  opacity: 0.72;
}

.home-feed-more::after {
  border: 0;
}

.home-feed-single-head {
  display: flex;
  min-height: 88rpx;
  margin: 0 0 20rpx;
  border-bottom: 1rpx solid $wplink-line;
}
```

`top: 28rpx` 根据 `88rpx` 标题高度和 `32rpx` 标记高度计算，使左标记垂直居中且不使用 `transform`。不要为当前标题添加按下态；不要保留任何 `.home-feed-woven-label` 规则或右侧 CSS 三角。

- [ ] **Step 4：运行定向测试并确认通过**

在 `wxapp` 目录运行：

```bash
npm test -- pages/home/index.test.mjs
```

预期：PASS，首页定向测试全部通过；除仓库既有 Node.js ExperimentalWarning 外不出现新增警告。

- [ ] **Step 5：运行小程序全量校验**

在 `wxapp` 目录运行：

```bash
npm run check
```

预期：页面校验、流程校验、291 项以上测试和 `mp-weixin` 构建全部通过；仅允许全局约束中列明的既有提示。

- [ ] **Step 6：检查改动范围并提交**

运行：

```bash
git diff --check
git diff -- wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git status --short
```

确认只修改首页组件和对应测试后提交：

```bash
git add wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git commit -m "style: 优化首页栏目标题式切换"
```
