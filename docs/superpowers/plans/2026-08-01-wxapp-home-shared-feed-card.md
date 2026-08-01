# 小程序首页商家与供需共用列表项重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 让首页“新入驻商家”和“近期供需”分别复用拿货地图与供需市场的列表视觉，并用单一紧凑供需组件消除供应、需求和首页之间的重复实现。

**架构：** 保留现有 `MerchantListItem`、`HomeRecentMerchantCard` 和 `MerchantPlaceCard` 组合关系；新增 `ResourceFeedCard`，由组件内部根据 `resource.direction` 统一整理并渲染供应或需求。首页和供需市场直接消费新组件，旧 `ResourceCard`、`DemandCard` 退出 `market` 场景，但保留其他页面正在使用的详细卡片能力。

**技术栈：** Vue 3、uni-app、SCSS、微信小程序、Node.js `node:test`

## 全局约束

- 保留首页现有“新入驻商家 / 近期供需”双标签、默认标签和单内容降级逻辑。
- 不修改后端接口、数据库结构、接口参数、排序规则和展示数量。
- 不修改拿货地图的筛选、地图、详情、导航、认领和选中行为。
- 不修改供需市场的筛选、分页、曝光来源和详情跳转。
- 供需列表不得新增认证、平台核实、VIP、会员或权益标识。
- 先写失败测试并确认因目标行为缺失而失败，再写生产代码。
- 只修改与本次重构直接相关的小程序组件、页面和测试。

## 文件结构

- 新建 `wxapp/components/ResourceFeedCard.vue`：统一的紧凑供需列表项，负责方向差异、字段整理、兜底和视觉。
- 新建 `wxapp/components/ResourceFeedCard.test.mjs`：验证共用组件接口、方向差异、字段兜底和样式约束。
- 修改 `wxapp/pages/market/index.vue`：供需市场统一使用 `ResourceFeedCard`。
- 修改 `wxapp/pages/market/index.test.mjs`：验证供需市场不再维护方向卡片分支。
- 修改 `wxapp/pages/home/index.vue`：近期供需统一使用 `ResourceFeedCard`，商家继续使用 `HomeRecentMerchantCard`。
- 修改 `wxapp/pages/home/index.test.mjs`：验证首页共用组件、曝光包装和原有双标签行为。
- 修改 `wxapp/components/ResourceCard.vue`：移除只服务供需市场的 `market` 模板、计算和样式。
- 修改 `wxapp/components/DemandCard.vue`：移除只服务供需市场的 `market` 模板、属性、计算和样式。
- 修改 `wxapp/components/ResourceCard.test.mjs`：删除旧 `market` 分支断言，验证旧组件不再包含紧凑市场实现。

---

### 任务 1：建立统一紧凑供需列表项

**文件：**

- 新建：`wxapp/components/ResourceFeedCard.test.mjs`
- 新建：`wxapp/components/ResourceFeedCard.vue`

**接口：**

- 输入：`resource: Object`，必填；使用现有字段 `direction`、`images`、`coverUrl`、`typeCode`、`typeName`、`title`、`quantityText`、`priceText`、`merchant`、`refreshedAt`、`dealtAt`。
- 输出：`open(resource)` 事件。
- 供后续使用：`<ResourceFeedCard :resource="item" @open="openResource" />`。

- [ ] **步骤 1：编写不存在组件的失败测试**

新建 `wxapp/components/ResourceFeedCard.test.mjs`：

```js
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const componentPath = path.resolve(new URL('.', import.meta.url).pathname, 'ResourceFeedCard.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('shared resource feed card owns one compact layout for supply and demand', () => {
  assert.equal(fs.existsSync(componentPath), true)
  assert.match(source, /const isDemand = computed\(\(\) => props\.resource\.direction === 'demand'\)/)
  assert.match(source, /const directionLabel = computed\(\(\) => isDemand\.value \? '需求' : '供应'\)/)
  assert.match(source, /:class="\['resource-feed-card', \{ demand: isDemand \}\]"/)
  assert.match(source, /:class="\['feed-type-badge', \{ demand: isDemand \}\]"/)
  assert.match(source, /\{\{ directionLabel \}\}/)
  assert.doesNotMatch(source, /<template v-if="isDemand">/)
})

test('shared resource feed card exposes the market decision fields and safe fallbacks', () => {
  for (const token of [
    'DEFAULT_RESOURCE_COVER',
    'resourceTypeLabel',
    'coverUrl',
    'titleText',
    'quantityText',
    'priceText',
    'merchantName',
    'freshnessText',
    'isCompleted',
    '交易信息待完善',
    '已完成',
  ]) {
    assert.match(source, new RegExp(token))
  }
  assert.match(source, /defineEmits\(\['open'\]\)/)
  assert.match(source, /@click="\$emit\('open', resource\)"/)
  assert.match(source, /isDemand\.value \? '需求标题待完善' : '供应标题待完善'/)
  assert.match(source, /isDemand\.value \? '采购方待确认' : '商家待确认'/)
})

test('shared resource feed card keeps a fixed compact image and aligned three-line body', () => {
  assert.match(source, /\.feed-thumb-wrap \{[\s\S]*width: 152rpx;[\s\S]*height: 152rpx;/)
  assert.match(source, /\.feed-card-main \{[\s\S]*align-content: space-between;/)
  assert.match(source, /\.feed-type-badge\.demand \{[\s\S]*background: \$wplink-warning;/)
  assert.match(source, /\.feed-type-badge:not\(\.demand\) \{[\s\S]*background: \$wplink-primary;/)
})
```

- [ ] **步骤 2：运行测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --test components/ResourceFeedCard.test.mjs
```

预期：FAIL，首个失败为 `ResourceFeedCard.vue` 不存在，不是测试语法或路径错误。

- [ ] **步骤 3：实现最小可用的共用组件**

新建 `wxapp/components/ResourceFeedCard.vue`，模板使用一套结构，不按方向复制模板：

```vue
<template>
  <view
    :class="['resource-feed-card', { demand: isDemand }]"
    @click="$emit('open', resource)"
  >
    <view class="feed-thumb-wrap">
      <image class="feed-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
      <text
        class="feed-type-badge"
        :class="{ demand: isDemand, 'with-completed': isCompleted }"
      >
        {{ resourceTypeLabel || directionLabel }}
      </text>
      <text v-if="isCompleted" class="feed-completed-badge">已完成</text>
    </view>

    <view class="feed-card-main">
      <text class="feed-title">{{ titleText }}</text>
      <view v-if="hasTradeInfo" class="feed-decision-line">
        <text v-if="quantityText" class="feed-quantity">{{ quantityText }}</text>
        <text v-if="priceText" class="feed-price">{{ priceText }}</text>
      </view>
      <text v-else class="feed-trade-empty">交易信息待完善</text>
      <view class="feed-merchant-line">
        <text class="feed-merchant-name">{{ merchantName }}</text>
        <text v-if="freshnessText" class="feed-refresh-time">{{ freshnessText }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

import { formatListFreshnessDate } from '../common/date'
import { resourceTypeLabel as resolveResourceTypeLabel } from '../common/resourceCategories'

const DEFAULT_RESOURCE_COVER = '/static/resource/default-resource-cover.png'

const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
})

defineEmits(['open'])

const isDemand = computed(() => props.resource.direction === 'demand')
const directionLabel = computed(() => isDemand.value ? '需求' : '供应')
const coverUrl = computed(() => {
  const images = props.resource.images || []
  return props.resource.coverUrl || images[0] || ''
})
const resourceTypeLabel = computed(() => resolveResourceTypeLabel(props.resource))
const titleText = computed(() => (
  props.resource.title || (isDemand.value ? '需求标题待完善' : '供应标题待完善')
))
const quantityText = computed(() => String(props.resource.quantityText || '').trim())
const priceText = computed(() => String(props.resource.priceText || '').trim())
const hasTradeInfo = computed(() => Boolean(quantityText.value || priceText.value))
const merchantName = computed(() => (
  (props.resource.merchant || {}).name || (isDemand.value ? '采购方待确认' : '商家待确认')
))
const freshnessText = computed(() => formatListFreshnessDate(props.resource.refreshedAt))
const isCompleted = computed(() => Boolean(props.resource.dealtAt))
</script>
```

在同一文件加入以下样式。颜色差异只放在类型标识，容器和信息层级保持一致：

```scss
<style lang="scss" scoped>
.resource-feed-card {
  display: flex;
  align-items: flex-start;
  gap: 12rpx;
  padding: 20rpx;
  overflow: hidden;
  border: 1rpx solid rgba(148, 163, 184, 0.22);
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.feed-thumb-wrap {
  position: relative;
  box-sizing: border-box;
  flex: 0 0 152rpx;
  width: 152rpx;
  height: 152rpx;
  overflow: hidden;
  border-radius: 10rpx;
  background: #edf2f7;
}

.feed-thumb {
  display: block;
  width: 100%;
  height: 100%;
}

.feed-type-badge,
.feed-completed-badge {
  position: absolute;
  top: 8rpx;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  color: #ffffff;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1;
}

.feed-type-badge {
  left: 8rpx;
  max-width: calc(100% - 16rpx);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feed-type-badge.with-completed {
  max-width: 68rpx;
}

.feed-type-badge:not(.demand) {
  background: $wplink-primary;
}

.feed-type-badge.demand {
  background: $wplink-warning;
}

.feed-completed-badge {
  right: 8rpx;
  background: rgba(71, 85, 105, 0.92);
}

.feed-card-main {
  display: grid;
  flex: 1;
  align-self: stretch;
  align-content: space-between;
  min-width: 0;
}

.feed-title,
.feed-quantity,
.feed-price,
.feed-trade-empty,
.feed-merchant-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feed-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.feed-decision-line,
.feed-merchant-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  min-width: 0;
}

.feed-quantity {
  min-width: 0;
  color: #475569;
  font-size: 25rpx;
  font-weight: 600;
}

.feed-price {
  min-width: 0;
  margin-left: auto;
  color: $wplink-warning;
  font-size: 28rpx;
  font-weight: 700;
  text-align: right;
}

.feed-trade-empty,
.feed-merchant-name,
.feed-refresh-time {
  color: $wplink-muted;
  font-size: 24rpx;
}

.feed-merchant-name {
  flex: 1;
  min-width: 0;
}

.feed-refresh-time {
  flex: 0 0 auto;
}
</style>
```

- [ ] **步骤 4：运行组件测试并确认通过**

运行：

```bash
cd wxapp
node --test components/ResourceFeedCard.test.mjs
```

预期：PASS，3 个测试全部通过。

- [ ] **步骤 5：提交共用组件**

```bash
git add wxapp/components/ResourceFeedCard.vue wxapp/components/ResourceFeedCard.test.mjs
git commit -m "feat: 新增共用供需列表项"
```

---

### 任务 2：供需市场接入共用组件并移除旧市场分支

**文件：**

- 修改：`wxapp/pages/market/index.test.mjs`
- 修改：`wxapp/pages/market/index.vue`
- 修改：`wxapp/components/ResourceCard.test.mjs`
- 修改：`wxapp/components/ResourceCard.vue`
- 修改：`wxapp/components/DemandCard.vue`

**接口：**

- 使用任务 1 产出的 `ResourceFeedCard(resource: Object)` 和 `open(resource)`。
- 保留 `ResourceExposure :resource-id="item.id" source="list"`。
- 保留页面方法 `openResource(item)`。

- [ ] **步骤 1：把供需市场测试改为期望单一共用组件**

将 `wxapp/pages/market/index.test.mjs` 中“renders mixed supply and demand”测试替换为：

```js
test('market page renders mixed supply and demand with the shared feed card', () => {
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(
    source,
    /<ResourceExposure[\s\S]*:resource-id="item\.id"[\s\S]*source="list"[\s\S]*<ResourceFeedCard[\s\S]*:resource="item"[\s\S]*@open="openResource"/,
  )
  assert.doesNotMatch(source, /import DemandCard/)
  assert.doesNotMatch(source, /import ResourceCard/)
  assert.doesNotMatch(source, /RESOURCE_DIRECTION_DEMAND/)
  assert.doesNotMatch(source, /variant="market"/)
})
```

将“market-only card layout does not leak”测试改为检查旧组件已经退出市场模式：

```js
test('legacy detailed cards no longer own the market-only layout', () => {
  const resourceCardSource = fs.readFileSync(path.join(root, 'components/ResourceCard.vue'), 'utf8')
  const demandCardSource = fs.readFileSync(path.join(root, 'components/DemandCard.vue'), 'utf8')

  assert.equal(resourceCardSource.includes("props.variant === 'market'"), false)
  assert.equal(resourceCardSource.includes('resource-card-market'), false)
  assert.equal(demandCardSource.includes("props.variant === 'market'"), false)
  assert.equal(demandCardSource.includes('demand-card-market'), false)
})
```

在 `wxapp/components/ResourceCard.test.mjs` 删除两个旧 `market` 布局测试，增加：

```js
test('legacy detailed cards delegate compact market rendering to ResourceFeedCard', () => {
  assert.doesNotMatch(source, /isMarketVariant|resource-card-market|market-card-main|market-type-badge/)
  assert.doesNotMatch(demandSource, /isMarketVariant|demand-card-market|market-card-main|market-type-badge/)
})
```

- [ ] **步骤 2：运行定向测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --test pages/market/index.test.mjs components/ResourceCard.test.mjs
```

预期：FAIL，原因是市场页仍导入旧卡片，且旧卡片仍包含 `market` 分支。

- [ ] **步骤 3：市场页改用共用组件**

将 `wxapp/pages/market/index.vue` 的列表改为：

```vue
<view v-if="rows.length" class="result-list">
  <ResourceExposure
    v-for="item in rows"
    :key="item.id"
    :resource-id="item.id"
    source="list"
  >
    <ResourceFeedCard :resource="item" @open="openResource" />
  </ResourceExposure>
  <text class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
</view>
```

脚本导入调整为：

```js
import ResourceFeedCard from '../../components/ResourceFeedCard.vue'
import ResourceExposure from '../../components/ResourceExposure.vue'
```

删除 `DemandCard`、`ResourceCard` 导入和不再使用的 `RESOURCE_DIRECTION_DEMAND` 常量。筛选、分页和 `openResource` 不改。

- [ ] **步骤 4：删除旧卡片中的 `market` 实现**

在 `ResourceCard.vue` 中执行以下精准收敛：

- 模板只保留当前 `v-else` 的详细供应卡结构，移除外层 `<template v-if="isMarketVariant">` / `<template v-else>`。
- `variantClass` 只处理 `home`、`compact`，不再返回 `resource-card-market`。
- 删除 `isMarketVariant`、`quantityText`、`hasMarketTradeInfo`。
- 删除 `.resource-card-market`、`.market-type-badge`、`.market-completed-badge`、`.market-card-main`、`.market-title`、`.market-decision-line`、`.market-merchant-line`、`.market-quantity`、`.market-price` 和 `.market-trade-empty` 样式。

在 `DemandCard.vue` 中执行以下精准收敛：

- 模板只保留当前 `v-else` 的详细需求卡结构。
- 删除 `variant` 属性、`variantClass`、`isMarketVariant`、`quantityText`、`hasMarketTradeInfo`。
- 根元素固定为 `class="demand-card"`。
- 删除 `.demand-card-market` 和全部 `.market-*` 样式。

不得修改详细卡片的认证/VIP禁用规则、默认封面、详情字段或点击事件。

- [ ] **步骤 5：运行供需市场和旧卡片测试**

运行：

```bash
cd wxapp
node --test pages/market/index.test.mjs components/ResourceCard.test.mjs components/ResourceFeedCard.test.mjs
```

预期：PASS，市场列表、旧详细卡和新共用组件测试全部通过。

- [ ] **步骤 6：提交供需市场接入**

```bash
git add wxapp/pages/market/index.vue wxapp/pages/market/index.test.mjs wxapp/components/ResourceCard.vue wxapp/components/ResourceCard.test.mjs wxapp/components/DemandCard.vue
git commit -m "refactor: 供需市场复用统一列表项"
```

---

### 任务 3：首页近期供需接入并完成全量回归

**文件：**

- 修改：`wxapp/pages/home/index.test.mjs`
- 修改：`wxapp/pages/home/index.vue`
- 验证：`wxapp/components/HomeRecentMerchantCard.vue`
- 验证：`wxapp/components/MerchantListItem.vue`
- 验证：`wxapp/components/MerchantPlaceCard.vue`

**接口：**

- 使用任务 1 产出的 `ResourceFeedCard(resource: Object)` 和 `open(resource)`。
- 保留 `ResourceExposure :resource-id="item.id" source="home"`。
- 保留 `HomeRecentMerchantCard`、`homeFeedState`、`openMerchant`、`openResource`、`openSourcingMap` 和 `openSearch`。

- [ ] **步骤 1：为首页共用组件接入编写失败测试**

在 `wxapp/pages/home/index.test.mjs` 新增：

```js
test('home reuses map merchant rows and the shared compact resource feed card', () => {
  assert.match(source, /import HomeRecentMerchantCard from '\.\.\/\.\.\/components\/HomeRecentMerchantCard\.vue'/)
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(
    source,
    /<ResourceExposure[\s\S]*:resource-id="item\.id"[\s\S]*source="home"[\s\S]*<ResourceFeedCard[\s\S]*:resource="item"[\s\S]*@open="openResource"/,
  )
  assert.doesNotMatch(source, /import ResourceCard/)
  assert.doesNotMatch(source, /variant="home"/)
})
```

在已有商家测试中保留并强化以下断言，确保本次没有把地图专属操作带入首页：

```js
const merchantCardSource = fs.readFileSync(path.join(root, 'components/HomeRecentMerchantCard.vue'), 'utf8')
assert.match(merchantCardSource, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
assert.doesNotMatch(merchantCardSource, /导航|待认领|这是我的档口/)
```

- [ ] **步骤 2：运行首页测试并确认失败原因正确**

运行：

```bash
cd wxapp
node --test pages/home/index.test.mjs
```

预期：FAIL，原因是首页仍导入 `ResourceCard` 并使用 `variant="home"`。

- [ ] **步骤 3：首页改用共用供需列表项**

将首页供需列表中的组件替换为：

```vue
<view v-if="homeResources.length" class="home-resource-list">
  <ResourceExposure
    v-for="item in homeResources"
    :key="item.id"
    :resource-id="item.id"
    source="home"
  >
    <ResourceFeedCard :resource="item" @open="openResource" />
  </ResourceExposure>
</view>
```

把脚本导入从：

```js
import ResourceCard from '../../components/ResourceCard.vue'
```

替换为：

```js
import ResourceFeedCard from '../../components/ResourceFeedCard.vue'
```

将 `.recent-merchant-list` 和 `.home-resource-list` 的列表间距统一为供需市场与拿货地图使用的 `18rpx`；不修改标签控件、推荐卡和“查看更多”按钮样式。

- [ ] **步骤 4：运行首页与相关组件定向测试**

运行：

```bash
cd wxapp
node --test pages/home/index.test.mjs pages/home/homeFeedState.test.mjs components/HomeRecentMerchantCard.test.mjs components/MerchantListItem.test.mjs components/MerchantPlaceCard.test.mjs components/ResourceFeedCard.test.mjs pages/sourcing-map/index.test.mjs
```

预期：PASS，首页双标签、商家共用组件、供需共用组件和拿货地图行为全部通过。

- [ ] **步骤 5：运行完整小程序测试**

运行：

```bash
cd wxapp
npm test
```

预期：PASS，无失败测试、未处理异常或新增警告。

- [ ] **步骤 6：构建微信小程序**

运行：

```bash
cd wxapp
npm run build:mp-weixin
```

预期：构建成功，输出到 `wxapp/dist/mp-weixin`，没有 Vue 模板或 SCSS 编译错误。

- [ ] **步骤 7：检查改动边界并提交首页接入**

运行：

```bash
git diff --check
git status --short
git diff --stat
```

确认只包含本计划列出的源文件、测试文件和构建产生的既有忽略文件；不得提交 `wxapp/dist` 或无关改动。

提交：

```bash
git add wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git commit -m "refactor: 首页复用统一供需列表项"
```

预期：提交完成，工作区不存在本任务遗留的未提交源代码。
