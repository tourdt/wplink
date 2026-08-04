# 小程序「我的权益」卡片冷启动优化实施计划

> **供执行代理使用：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，逐项执行本计划。所有步骤使用复选框跟踪。

**目标：** 将「我的权益」卡片改为“先看余额、再做免费任务、仅在不足时购买”的冷启动结构，并保持现有权益数据、成长任务和次数包路由不变。

**架构：** 继续由 `wxapp/pages/my/index.vue` 汇总商户权益和成长活动，只调整卡片模板、少量派生状态和局部 SCSS。源码行为测试与流程校验同步锁定文案、点击边界和可信余额保护，不新增组件、接口或全局样式。

**技术栈：** Vue 3 `<script setup>`、uni-app、微信小程序、SCSS、Node.js `node:test`

## 全局约束

- 所有产品文案、实现注释和计划说明使用中文；代码标识、路径与命令保留英文。
- 只修改 `wxapp/pages/my/index.vue`、对应源码测试和流程校验，不改后端接口、权益规则、成长任务页、购买页或「我的」页面其他区域。
- 余额大于 2 时隐藏购买入口；余额为 1–2 时展示弱补充入口；余额为 0 时展示对应类型的购买入口。
- 只有 `entitlementQuotaReady` 为真时才能展示低余额或零余额状态；加载中、加载失败和无商户身份时不得按零余额营销。
- 绿色只表示免费任务；橙色只表示低余额或零余额补充；余额和基础信息保持蓝灰色。
- 整张权益卡、数字和空白区域不得绑定跳转；只有「权益明细」「去完成」和对应购买文案可以点击。
- 继续使用 `buildQuotaPurchaseUrl(QUOTA_TYPE_PUBLISH)` 和 `buildQuotaPurchaseUrl(QUOTA_TYPE_REFRESH)`，客户端不拼价格、不创建新购买协议。
- 不新增依赖、不拆分新组件、不修改 `wxapp/uni.scss`。

## 文件结构

- 修改：`wxapp/pages/my/index.vue`——权益卡模板、派生文案、显式点击入口和局部样式。
- 修改：`wxapp/pages/my/index.test.mjs`——锁定信息层级、可信余额状态、点击边界和路由行为。
- 修改：`wxapp/scripts/validate-flows.mjs`——把「我的」页流程标记从旧文案和旧整卡点击更新为新结构标记。
- 修改：`wxapp/scripts/validate-flows.test.mjs`——同步流程校验测试中的必备和移除标记。

---

### 任务 1：重构权益卡信息层级与交互状态

**文件：**

- 修改：`wxapp/pages/my/index.test.mjs:54-113`
- 修改：`wxapp/scripts/validate-flows.test.mjs:452-503`
- 修改：`wxapp/pages/my/index.vue:27-53, 121-181, 285-312`
- 修改：`wxapp/scripts/validate-flows.mjs:290-341`

**接口：**

- 使用：现有 `entitlementQuotaReady`、`publishQuotaRemaining`、`refreshQuotaRemaining`、`publishQuotaLow`、`refreshQuotaLow`、`activeGrowthCampaign`、`openQuotaPurchase(quotaType)`、`openGrowthEntitlement()`。
- 产生：`benefitGrowthTitle: ComputedRef<string>`，值只会是 `做任务，免费得次数` 或 `也可以免费获取`。
- 保持：`publishQuotaDisplay`、`refreshQuotaDisplay` 在不可信状态下返回 `--`。
- 移除：`benefitOverviewDesc`、`openBenefitOverview()`、权益卡根节点的统一 `@click`。

- [ ] **步骤 1：把权益卡结构测试改为新信息层级，并让测试先失败**

在 `wxapp/pages/my/index.test.mjs` 中替换 `my page shows compact entitlement overview with entitlement center access` 用例的权益卡断言，保留既有数据加载断言，新增以下具体断言：

```js
assert.match(source, /<view v-if="benefitOverviewVisible" class="benefit-overview-card section-card">/)
assert.doesNotMatch(source, /class="benefit-overview-card section-card" @click=/)
assert.match(source, /<text class="benefit-title">我的权益<\/text>/)
assert.match(source, /<text class="benefit-desc">剩余次数可用于发布和刷新供需信息<\/text>/)
assert.match(source, /<text class="benefit-detail-action" @click="openGrowthEntitlement">权益明细 ›<\/text>/)
assert.match(source, /<text class="benefit-label">可发布<\/text>/)
assert.match(source, /<text class="benefit-label">可刷新<\/text>/)
assert.match(source, /<text v-if="entitlementQuotaReady" class="benefit-unit">次<\/text>/)
assert.match(source, /<text v-else-if="entitlementQuotaReady" class="benefit-status">可正常使用<\/text>/)
assert.doesNotMatch(source, /const benefitOverviewDesc = computed/)
assert.doesNotMatch(source, /function openBenefitOverview\(\)/)
```

在同一文件的可信额度用例中补充状态类和免费任务断言，并移除对 `quota-purchase-primary` 的旧断言：

```js
assert.match(source, /'benefit-stat-low': entitlementQuotaReady && publishQuotaLow/)
assert.match(source, /'benefit-stat-empty': entitlementQuotaReady && publishQuotaRemaining === 0/)
assert.match(source, /'benefit-stat-low': entitlementQuotaReady && refreshQuotaLow/)
assert.match(source, /'benefit-stat-empty': entitlementQuotaReady && refreshQuotaRemaining === 0/)
assert.match(source, /v-if="entitlementQuotaReady && publishQuotaRemaining === 0"[\s\S]*购买发布次数 ›/)
assert.match(source, /v-else-if="entitlementQuotaReady && publishQuotaLow"[\s\S]*即将用完 · 去补充 ›/)
assert.match(source, /v-if="entitlementQuotaReady && refreshQuotaRemaining === 0"[\s\S]*购买刷新次数 ›/)
assert.match(source, /v-else-if="entitlementQuotaReady && refreshQuotaLow"[\s\S]*即将用完 · 去补充 ›/)
assert.match(source, /const benefitGrowthTitle = computed\(\(\) => \{/)
assert.match(source, /if \(!entitlementQuotaReady\.value\) return '做任务，免费得次数'/)
assert.match(source, /return publishQuotaRemaining\.value === 0 \|\| refreshQuotaRemaining\.value === 0[\s\S]*\? '也可以免费获取'[\s\S]*: '做任务，免费得次数'/)
assert.match(source, /v-if="activeGrowthCampaign\.code" class="benefit-growth-banner"/)
assert.match(source, /\{\{ benefitGrowthTitle \}\}/)
assert.match(source, /完成任务，奖励自动到账/)
assert.match(source, /class="benefit-growth-button" @click="openGrowthEntitlement">去完成<\/text>/)
assert.doesNotMatch(source, /quota-purchase-primary/)
```

- [ ] **步骤 2：运行页面测试，确认它因旧模板失败**

运行：

```bash
node --test wxapp/pages/my/index.test.mjs
```

预期：FAIL；失败信息至少包含找不到 `权益明细 ›`、`benefit-stat-low` 或 `benefit-growth-banner`，证明测试确实约束了新设计。

- [ ] **步骤 3：用最小模板和派生状态实现新结构**

将 `wxapp/pages/my/index.vue` 中第 27–53 行权益卡替换为以下结构。根节点不得增加点击事件：

```vue
<view v-if="benefitOverviewVisible" class="benefit-overview-card section-card">
  <view class="benefit-head">
    <view class="benefit-title-wrap">
      <text class="benefit-title">我的权益</text>
      <text class="benefit-desc">剩余次数可用于发布和刷新供需信息</text>
    </view>
    <text class="benefit-detail-action" @click="openGrowthEntitlement">权益明细 ›</text>
  </view>
  <view class="benefit-stats">
    <view
      class="benefit-stat"
      :class="{
        'benefit-stat-low': entitlementQuotaReady && publishQuotaLow,
        'benefit-stat-empty': entitlementQuotaReady && publishQuotaRemaining === 0,
      }"
    >
      <text class="benefit-label">可发布</text>
      <view class="benefit-value-row">
        <text class="benefit-value">{{ publishQuotaDisplay }}</text>
        <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
      </view>
      <text v-if="entitlementQuotaReady && publishQuotaRemaining === 0" class="quota-purchase-action" @click="openQuotaPurchase(QUOTA_TYPE_PUBLISH)">购买发布次数 ›</text>
      <text v-else-if="entitlementQuotaReady && publishQuotaLow" class="quota-purchase-action quota-purchase-low" @click="openQuotaPurchase(QUOTA_TYPE_PUBLISH)">即将用完 · 去补充 ›</text>
      <text v-else-if="entitlementQuotaReady" class="benefit-status">可正常使用</text>
    </view>
    <view
      class="benefit-stat"
      :class="{
        'benefit-stat-low': entitlementQuotaReady && refreshQuotaLow,
        'benefit-stat-empty': entitlementQuotaReady && refreshQuotaRemaining === 0,
      }"
    >
      <text class="benefit-label">可刷新</text>
      <view class="benefit-value-row">
        <text class="benefit-value">{{ refreshQuotaDisplay }}</text>
        <text v-if="entitlementQuotaReady" class="benefit-unit">次</text>
      </view>
      <text v-if="entitlementQuotaReady && refreshQuotaRemaining === 0" class="quota-purchase-action" @click="openQuotaPurchase(QUOTA_TYPE_REFRESH)">购买刷新次数 ›</text>
      <text v-else-if="entitlementQuotaReady && refreshQuotaLow" class="quota-purchase-action quota-purchase-low" @click="openQuotaPurchase(QUOTA_TYPE_REFRESH)">即将用完 · 去补充 ›</text>
      <text v-else-if="entitlementQuotaReady" class="benefit-status">可正常使用</text>
    </view>
  </view>
  <view v-if="activeGrowthCampaign.code" class="benefit-growth-banner">
    <view class="benefit-growth-main">
      <text class="benefit-growth-title">{{ benefitGrowthTitle }}</text>
      <text class="benefit-growth-desc">完成任务，奖励自动到账</text>
    </view>
    <text class="benefit-growth-button" @click="openGrowthEntitlement">去完成</text>
  </view>
  <text v-if="benefitExpiryReminder" class="benefit-expiry">{{ benefitExpiryReminder }}</text>
</view>
```

在 `activeGrowthCampaign` 之后增加派生标题；该计算必须先检查可信余额，避免加载失败时把 `0` 当成真实余额：

```js
const benefitGrowthTitle = computed(() => {
  // 权益未知时使用中性免费任务文案，不能把接口故障误表达为“次数已用完”。
  if (!entitlementQuotaReady.value) return '做任务，免费得次数'
  return publishQuotaRemaining.value === 0 || refreshQuotaRemaining.value === 0
    ? '也可以免费获取'
    : '做任务，免费得次数'
})
```

删除不再使用的 `benefitOverviewDesc` 和 `openBenefitOverview()`。保留 `openGrowthEntitlement()` 中的登录保护、商户资料保护和现有成长权益路由。

- [ ] **步骤 4：运行页面测试，确认新结构与既有功能同时通过**

运行：

```bash
node --test wxapp/pages/my/index.test.mjs
```

预期：PASS，全部用例通过；不存在未使用的 `benefitOverviewDesc` 或 `openBenefitOverview` 源码标记。

- [ ] **步骤 5：更新流程校验标记并让流程测试先失败**

在 `wxapp/scripts/validate-flows.test.mjs` 的「我的」页 token 列表中：

- 删除：`openBenefitOverview`、`/pages/vip/index`、`quota-purchase-primary`。
- 增加：`权益明细`、`剩余次数可用于发布和刷新供需信息`、`benefitGrowthTitle`、`做任务，免费得次数`、`完成任务，奖励自动到账`、`benefit-stat-low`、`benefit-stat-empty`、`benefit-growth-banner`、`openGrowthEntitlement`。
- 在 `hiddenToken` 数组中增加：`benefitOverviewDesc`、`openBenefitOverview`、`quota-purchase-primary`。

运行：

```bash
node --test wxapp/scripts/validate-flows.test.mjs
```

预期：FAIL，因为 `wxapp/scripts/validate-flows.mjs` 仍要求旧 token。

- [ ] **步骤 6：同步生产流程校验清单**

在 `wxapp/scripts/validate-flows.mjs` 的 `pages/my/index.vue` 检查项中应用与步骤 5 完全相同的 token 变更，并将 `absentChecks` 更新为：

```js
absentChecks: [
  '暂无可领取权益',
  'benefitOverviewDesc',
  'openBenefitOverview',
  'quota-purchase-primary',
],
```

- [ ] **步骤 7：运行页面测试和流程测试**

运行：

```bash
node --test wxapp/pages/my/index.test.mjs
node --test wxapp/scripts/validate-flows.test.mjs
npm --prefix wxapp run validate:flows
```

预期：三条命令均 PASS；流程校验输出不再要求旧的整卡购买入口。

- [ ] **步骤 8：提交信息层级与交互改动**

```bash
git add wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs wxapp/scripts/validate-flows.mjs wxapp/scripts/validate-flows.test.mjs
git commit -m "feat: 优化我的权益卡交互层级"
```

---

### 任务 2：落实余额状态与免费任务视觉语义

**文件：**

- 修改：`wxapp/pages/my/index.test.mjs:54-113`
- 修改：`wxapp/pages/my/index.vue:456-554`

**接口：**

- 使用：任务 1 已产生的 `.benefit-stat-low`、`.benefit-stat-empty`、`.benefit-value-row`、`.benefit-unit`、`.benefit-status`、`.benefit-growth-banner`、`.benefit-growth-main`、`.benefit-growth-title`、`.benefit-growth-desc`、`.benefit-growth-button`。
- 产生：上述选择器的局部 SCSS 定义；不产生跨文件样式接口。
- 保持：现有 `$wplink-primary-soft`、`$wplink-success`、`$wplink-success-soft`、`$wplink-warning`、`$wplink-warning-soft`、`$wplink-muted` 变量语义。

- [ ] **步骤 1：增加视觉选择器测试并确认先失败**

在 `wxapp/pages/my/index.test.mjs` 的权益卡用例中增加：

```js
assert.match(source, /\.benefit-detail-action \{[\s\S]*min-height: 72rpx/)
assert.match(source, /\.benefit-stat \{[\s\S]*border: 1rpx solid transparent/)
assert.match(source, /\.benefit-stat-low \{[\s\S]*rgba\(194, 58, 0, 0\.04\)/)
assert.match(source, /\.benefit-stat-empty \{[\s\S]*\$wplink-warning-soft/)
assert.match(source, /\.benefit-growth-banner \{[\s\S]*\$wplink-success-soft/)
assert.match(source, /\.benefit-growth-button \{[\s\S]*min-height: 72rpx/)
assert.doesNotMatch(source, /\.benefit-actions \{/)
assert.doesNotMatch(source, /\.benefit-growth-action \{/)
```

运行：

```bash
node --test wxapp/pages/my/index.test.mjs
```

预期：FAIL；新选择器尚未实现或仍存在旧 `.benefit-actions`、`.benefit-growth-action`。

- [ ] **步骤 2：替换权益卡局部样式**

在 `wxapp/pages/my/index.vue` 的 scoped SCSS 中删除 `.benefit-action`、`.benefit-actions`、`.benefit-growth-action` 和 `.quota-purchase-primary`，并将权益卡相关样式调整为：

```scss
.benefit-overview-card {
  display: grid;
  gap: 20rpx;
}

.benefit-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18rpx;
  min-width: 0;
}

.benefit-title-wrap {
  display: grid;
  flex: 1 1 auto;
  gap: 6rpx;
  min-width: 0;
}

.benefit-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.benefit-detail-action {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  min-height: 72rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  font-weight: 700;
}

.benefit-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.benefit-stat {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 180rpx;
  padding: 18rpx;
  border: 1rpx solid transparent;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
}

.benefit-stat-low {
  border-color: rgba(194, 58, 0, 0.16);
  background: rgba(194, 58, 0, 0.04);
}

.benefit-stat-empty {
  border-color: rgba(194, 58, 0, 0.24);
  background: $wplink-warning-soft;
}

.benefit-value-row {
  display: flex;
  align-items: baseline;
  gap: 6rpx;
  margin-top: 8rpx;
}

.benefit-value {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 800;
  line-height: 1.1;
}

.benefit-unit,
.benefit-label,
.benefit-status {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.3;
}

.benefit-status,
.quota-purchase-action {
  margin-top: auto;
  padding-top: 12rpx;
}

.quota-purchase-action {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  min-height: 56rpx;
  color: $wplink-warning;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.4;
}

.quota-purchase-low {
  color: rgba(194, 58, 0, 0.78);
}

.benefit-growth-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: $wplink-success-soft;
}

.benefit-growth-main {
  display: grid;
  flex: 1 1 auto;
  gap: 6rpx;
  min-width: 0;
}

.benefit-growth-title {
  color: $wplink-success;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
}

.benefit-growth-desc {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.4;
}

.benefit-growth-button {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  min-height: 72rpx;
  padding: 0 20rpx;
  border-radius: 999rpx;
  background: $wplink-success;
  color: $wplink-card;
  font-size: 22rpx;
  font-weight: 700;
}
```

保留现有 `.benefit-expiry` 样式，确保到期提醒继续位于免费任务入口下方。

- [ ] **步骤 3：运行页面测试并修复仅限权益卡范围的样式问题**

运行：

```bash
node --test wxapp/pages/my/index.test.mjs
```

预期：PASS；不得通过放宽或删除步骤 1 的选择器断言来规避失败。

- [ ] **步骤 4：执行小程序构建，验证 Vue 模板和 SCSS 可编译**

运行：

```bash
npm --prefix wxapp run build:mp-weixin
```

预期：PASS；无 Vue 模板编译错误、SCSS 变量错误或未闭合标签。

- [ ] **步骤 5：执行完整小程序回归**

运行：

```bash
npm --prefix wxapp run validate:pages
npm --prefix wxapp run validate:flows
npm --prefix wxapp test
```

预期：三条命令均 PASS；「我的」页、成长任务、权益购买和登录态相关既有测试无回归。

- [ ] **步骤 6：人工检查四种关键状态**

在微信开发者工具或本地可预览环境中检查：

1. 权益加载中：两个余额显示 `--`，不出现橙色购买文案。
2. 余额分别为 `8 / 5`：两个余额块均为蓝灰色，只突出绿色免费任务。
3. 余额分别为 `2 / 5`：发布次数块显示弱提醒，刷新次数块仍显示「可正常使用」。
4. 余额分别为 `3 / 0`：刷新次数块显示浅橙背景和「购买刷新次数 ›」，底部显示「也可以免费获取」。
5. 点击卡片空白、数字和标签无跳转；点击「权益明细」「去完成」和购买入口分别进入正确页面。
6. 长文案、窄屏和最近到期提醒出现时，数字、单位和操作文案不重叠、不被截断。

- [ ] **步骤 7：提交视觉样式与最终验证结果**

```bash
git add wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs
git commit -m "style: 完善权益卡余额状态视觉"
```
