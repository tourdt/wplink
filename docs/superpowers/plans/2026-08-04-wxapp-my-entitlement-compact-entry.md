# 小程序「我的权益」紧凑常驻入口实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将发布次数和刷新次数模块改为两行紧凑布局，并在可信余额状态下于各模块右上角常驻正确的补充或购买入口。

**架构：** 保留现有权益加载、可信状态守卫、低余额判断和购买路由，只调整 `pages/my/index.vue` 的模板层级与局部 SCSS。发布和刷新模块继续独立使用既有 `QUOTA_TYPE_PUBLISH`、`QUOTA_TYPE_REFRESH`，入口文案由各自余额直接决定，不新增接口、组件或依赖。

**技术栈：** uni-app、Vue 3 `<script setup>`、微信小程序、SCSS、Node.js `node:test` 源码行为测试。

## 全局约束

- 只有 `entitlementQuotaReady` 为真时才展示常驻获取入口；加载中、加载失败或商户身份无效时显示 `--` 并隐藏入口。
- 余额大于 0 时入口文案为 `补充 ›`；余额等于 0 时入口文案为 `购买 ›`。
- 余额大于 2 时入口为蓝灰色，余额 1–2 时为弱橙色，余额 0 时为橙色。
- 删除余额充足状态下的 `可正常使用`，不再展示 `即将用完 · 去补充 ›`、`购买发布次数 ›`、`购买刷新次数 ›`。
- 余额模块高度控制在 `132–140rpx`，本计划统一使用 `min-height: 136rpx`；右上角入口实际触摸高度不得低于 `64rpx`。
- 数字、标签、余额块和卡片空白区域不得绑定点击事件；发布和刷新入口必须继续使用各自正确的 `quotaType`。
- 免费任务、权益明细、最近到期提醒、登录保护及商户资料保护行为保持不变。
- 不新增依赖，不修改后端接口，不重构「我的」页面其他区域。

## 文件结构

- 修改：`wxapp/pages/my/index.vue`——调整两个余额模块的模板结构、常驻入口文案和局部样式。
- 修改：`wxapp/pages/my/index.test.mjs`——覆盖可信状态、入口常驻、短文案、路由绑定、点击边界和紧凑视觉规则。

---

### Task 1: 将发布与刷新入口改为可信状态下常驻

**Files:**
- Modify: `wxapp/pages/my/index.vue:35-68`
- Test: `wxapp/pages/my/index.test.mjs:90-132`

**Interfaces:**
- Consumes: `entitlementQuotaReady: ComputedRef<boolean>`、`publishQuotaRemaining: ComputedRef<number>`、`refreshQuotaRemaining: ComputedRef<number>`、`openQuotaPurchase(quotaType)`、`QUOTA_TYPE_PUBLISH`、`QUOTA_TYPE_REFRESH`。
- Produces: 两个 `.benefit-stat-head`；其中 `.quota-purchase-action` 仅在可信状态出现，发布入口固定传 `QUOTA_TYPE_PUBLISH`，刷新入口固定传 `QUOTA_TYPE_REFRESH`。

- [ ] **Step 1: 先修改行为测试，表达常驻入口和短文案规则**

在 `my page only highlights quota purchases after a trustworthy entitlement response` 中，用以下断言替换现有四条零余额/低余额长文案断言，并保留可信状态、独立余额状态和路由构建器断言：

```js
assert.match(source, /<view class="benefit-stat-head">[\s\S]*<text class="benefit-label">可发布<\/text>[\s\S]*v-if="entitlementQuotaReady"[^>]*class="quota-purchase-action"[^>]*@click="openQuotaPurchase\(QUOTA_TYPE_PUBLISH\)"/)
assert.match(source, /\{\{ publishQuotaRemaining === 0 \? '购买 ›' : '补充 ›' \}\}/)
assert.match(source, /<view class="benefit-stat-head">[\s\S]*<text class="benefit-label">可刷新<\/text>[\s\S]*v-if="entitlementQuotaReady"[^>]*class="quota-purchase-action"[^>]*@click="openQuotaPurchase\(QUOTA_TYPE_REFRESH\)"/)
assert.match(source, /\{\{ refreshQuotaRemaining === 0 \? '购买 ›' : '补充 ›' \}\}/)
assert.doesNotMatch(source, /可正常使用/)
assert.doesNotMatch(source, /即将用完 · 去补充 ›/)
assert.doesNotMatch(source, /购买发布次数 ›/)
assert.doesNotMatch(source, /购买刷新次数 ›/)
```

继续保留下列点击边界断言，确保常驻入口没有扩大为整块点击：

```js
assert.doesNotMatch(source, /<view\s+class="benefit-stat"[^>]*@click=/)
assert.doesNotMatch(source, /<text class="benefit-value"[^>]*@click=/)
assert.doesNotMatch(source, /<text class="benefit-label"[^>]*@click=/)
```

- [ ] **Step 2: 运行定向测试并确认新断言先失败**

Run: `node --test wxapp/pages/my/index.test.mjs`

Expected: FAIL；失败信息应指出缺少 `.benefit-stat-head`、可信状态常驻入口或新短文案，并命中仍存在的旧文案。

- [ ] **Step 3: 最小化修改两个余额模块模板**

将发布模块内部内容替换为：

```vue
<view class="benefit-stat-head">
  <text class="benefit-label">可发布</text>
  <text
    v-if="entitlementQuotaReady"
    class="quota-purchase-action"
    @click="openQuotaPurchase(QUOTA_TYPE_PUBLISH)"
  >{{ publishQuotaRemaining === 0 ? '购买 ›' : '补充 ›' }}</text>
</view>
<view class="benefit-value-row">
  <text class="benefit-value">{{ publishQuotaDisplay }}</text>
  <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
</view>
```

将刷新模块内部内容替换为：

```vue
<view class="benefit-stat-head">
  <text class="benefit-label">可刷新</text>
  <text
    v-if="entitlementQuotaReady"
    class="quota-purchase-action"
    @click="openQuotaPurchase(QUOTA_TYPE_REFRESH)"
  >{{ refreshQuotaRemaining === 0 ? '购买 ›' : '补充 ›' }}</text>
</view>
<view class="benefit-value-row">
  <text class="benefit-value">{{ refreshQuotaDisplay }}</text>
  <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
</view>
```

删除两个模块原有的三分支长文案和 `.benefit-status` 节点。不要修改 `benefit-stat-low`、`benefit-stat-empty` 的绑定条件，也不要修改 `openQuotaPurchase`。

- [ ] **Step 4: 运行页面测试并确认行为通过**

Run: `node --test wxapp/pages/my/index.test.mjs`

Expected: PASS，11 tests、0 failures。

- [ ] **Step 5: 提交常驻入口行为改动**

```bash
git add wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs
git commit -m "feat: 常驻展示权益补充入口"
```

---

### Task 2: 压缩余额模块并完成状态视觉验收

**Files:**
- Modify: `wxapp/pages/my/index.vue:507-575`
- Test: `wxapp/pages/my/index.test.mjs:134-148`

**Interfaces:**
- Consumes: Task 1 生成的 `.benefit-stat-head`、`.quota-purchase-action` 和现有 `.benefit-stat-low`、`.benefit-stat-empty` 状态类。
- Produces: `136rpx` 等高余额模块；正常状态蓝灰入口、低余额弱橙入口、零余额橙色入口；入口触摸高度至少 `64rpx`。

- [ ] **Step 1: 先增加紧凑布局和分级颜色的失败断言**

在 `my page gives quota states and free tasks distinct visual treatments` 中增加：

```js
assert.match(source, /\.benefit-stat \{[^}]*position: relative;[^}]*min-height: 136rpx/)
assert.match(source, /\.benefit-stat-head \{[^}]*display: flex;[^}]*padding-right: 74rpx/)
assert.match(source, /\.quota-purchase-action \{[^}]*position: absolute;[^}]*min-height: 64rpx;[^}]*color: \$wplink-muted/)
assert.match(source, /\.benefit-stat-low \.quota-purchase-action \{[^}]*color: rgba\(194, 58, 0, 0\.78\)/)
assert.match(source, /\.benefit-stat-empty \.quota-purchase-action \{[^}]*color: \$wplink-warning/)
assert.doesNotMatch(source, /\.benefit-status/)
assert.doesNotMatch(source, /\.quota-purchase-low/)
```

- [ ] **Step 2: 运行定向测试并确认样式断言先失败**

Run: `node --test wxapp/pages/my/index.test.mjs`

Expected: FAIL；失败信息应指出旧 `180rpx` 高度、旧 `56rpx` 点击高度或缺少状态后代选择器。

- [ ] **Step 3: 实现紧凑模块和右上角点击热区**

将余额区相关 SCSS 收敛为以下结构；保留现有低余额和零余额块背景：

```scss
.benefit-stat {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 136rpx;
  padding: 16rpx 18rpx;
  border: 1rpx solid transparent;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
}

.benefit-stat-head {
  display: flex;
  align-items: center;
  min-width: 0;
  min-height: 32rpx;
  padding-right: 74rpx;
}

.benefit-value-row {
  display: flex;
  align-items: baseline;
  gap: 6rpx;
  margin-top: 10rpx;
}

.benefit-unit,
.benefit-label {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.3;
}

.quota-purchase-action {
  position: absolute;
  top: 2rpx;
  right: 8rpx;
  display: inline-flex;
  align-items: center;
  min-height: 64rpx;
  padding: 0 10rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.4;
}

.benefit-stat-low .quota-purchase-action {
  color: rgba(194, 58, 0, 0.78);
}

.benefit-stat-empty .quota-purchase-action {
  color: $wplink-warning;
}
```

删除旧 `.benefit-status`、`.quota-purchase-low` 规则以及 `.benefit-status, .quota-purchase-action` 的底部布局规则。不要改变免费任务绿色、到期提醒蓝灰色或余额块背景色。

- [ ] **Step 4: 运行页面测试、流程校验和完整测试**

Run: `node --test wxapp/pages/my/index.test.mjs`

Expected: PASS，11 tests、0 failures。

Run: `npm --prefix wxapp run validate:pages`

Expected: PASS，输出 `wxapp pages ok`。

Run: `npm --prefix wxapp run validate:flows`

Expected: PASS，输出 `wxapp flows ok`。

Run: `npm --prefix wxapp test`

Expected: PASS，367 tests、0 failures；允许出现既有 Node VM Modules 实验性提示。

- [ ] **Step 5: 构建微信小程序并在开发者工具复查状态矩阵**

Run: `npm --prefix wxapp run build:mp-weixin`

Expected: PASS，输出 `DONE Build complete`；允许出现既有 Sass `legacy-js-api` 与 `@import` 弃用提示。

在微信开发者工具中使用临时 `wx.request` mock 和临时 session storage 依次验证：

1. 正常 `8 / 5`：两个模块右上角均显示蓝灰色 `补充 ›`，不显示 `可正常使用`，免费任务仍为绿色。
2. 低余额 `2 / 5`：发布模块 `补充 ›` 为弱橙色，刷新模块 `补充 ›` 保持蓝灰色，模块高度一致。
3. 零余额 `0 / 5`：发布模块显示橙色 `购买 ›`，刷新模块显示蓝灰色 `补充 ›`。
4. 权益失败：两个数字均为 `--`，两个模块均不显示 `补充 ›` 或 `购买 ›`。
5. 点击发布模块入口进入 `/pages/vip/index?tab=addons&quotaType=publish_quota`；点击刷新模块入口进入 `/pages/vip/index?tab=addons&quotaType=refresh_quota`。
6. 点击余额块、数字、标签和空白区域仍停留在 `/pages/my/index`。
7. 读取两个 `.benefit-stat` 的运行时高度，均应处于 `132–140rpx` 对应的像素范围且彼此相等；窄屏下入口不得遮挡标签或数字。
8. 验收结束后 restore `wx.request`，删除临时 token、merchantId，并确认账号文案恢复为 `未登录`。

- [ ] **Step 6: 检查 diff 并提交视觉改动**

Run: `git diff --check`

Expected: PASS，无输出。

Run: `git status --short`

Expected: 仅显示 `wxapp/pages/my/index.vue` 和 `wxapp/pages/my/index.test.mjs` 的预期修改。

```bash
git add wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs
git commit -m "style: 压缩权益余额模块高度"
```
