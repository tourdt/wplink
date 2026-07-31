# 小程序首页商家与供需双标签实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将首页“新入驻商家”和“近期供需”合并为双标签内容区，以静态列表展示最多 6 家新商家，并让首页与拿货地图复用同一商家列表基础组件。

**架构：** 新建无业务依赖的 `MerchantListItem` 负责商家列表行的共同结构和视觉规范，`HomeRecentMerchantCard` 与 `MerchantPlaceCard` 分别作为首页和地图的薄封装。首页通过纯函数 `getHomeFeedState` 计算可用内容、默认标签和是否显示切换控件，数据仍并行加载，切换标签不重新请求接口。

**技术栈：** Vue 3 `<script setup>`、uni-app、SCSS、Node.js `node:test`、微信小程序构建。

## 全局约束

- 首页新入驻商家固定最多展示 6 家，不自动滚动、不循环播放。
- 新入驻商家和近期供需都有内容时显示两个等宽标签；只有一类内容时隐藏切换控件并直接展示可用内容。
- 有新商家时默认展示新入驻商家；新商家为空或加载失败时展示近期供需。
- 切换标签只切换已加载内容，不重新请求商家或供需接口。
- 保持 `GET /api/v1/home/recent-merchants?cityCode=zhili` 的资格、排序、数量和响应结构不变。
- 首页商家行不展示详细地址、联系方式、VIP、评分、热度或地图专属操作。
- 拿货地图现有的选中、进入主页、导航和认领行为必须保持。
- 不新增第三方依赖，不调整后端代码，不重构与本需求无关的首页模块。
- 开始修改前必须检查 `MerchantPlaceCard.vue`、`MerchantPlaceCard.test.mjs`、`pages/sourcing-map/` 的工作区差异；保留已有“进入主页”等改动。若未提交改动与本计划重叠且无法分离暂存，停止实施并请求用户处理，不能覆盖或代为提交。

---

### 任务 1：建立商家列表基础组件并接入拿货地图

**文件：**

- 新建：`wxapp/components/MerchantListItem.vue`
- 新建：`wxapp/components/MerchantListItem.test.mjs`
- 修改：`wxapp/components/MerchantPlaceCard.vue`
- 修改：`wxapp/components/MerchantPlaceCard.test.mjs`

**接口：**

- `MerchantListItem` 属性：
  - `title: String`，必填；
  - `subtitle: String`，默认空字符串；
  - `tags: Array<String>`，默认空数组；
  - `selected: Boolean`，默认 `false`。
- `MerchantListItem` 插槽：`leading`、`badge`、`meta`、`actions`。
- `MerchantListItem` 事件：点击列表行时触发 `activate`。
- `MerchantPlaceCard` 对外属性和 `select/detail/navigate/claim` 事件保持不变。

- [ ] **步骤 1：记录并检查拿货地图现有未提交改动**

运行：

```bash
git status --short
git diff -- wxapp/components/MerchantPlaceCard.vue \
  wxapp/components/MerchantPlaceCard.test.mjs \
  wxapp/pages/sourcing-map/index.vue \
  wxapp/pages/sourcing-map/index.test.mjs \
  wxapp/pages/sourcing-map/merchantPlaceState.js \
  wxapp/pages/sourcing-map/merchantPlaceState.test.mjs
```

预期：明确现有“进入主页”等改动的来源和状态；后续改造必须保留 `detail` 事件、`hasMerchantDetail` 判断以及地图页的 `openMerchantDetail` 跳转。

- [ ] **步骤 2：为基础组件编写失败测试**

新建 `wxapp/components/MerchantListItem.test.mjs`：

```js
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const componentPath = path.resolve(new URL('.', import.meta.url).pathname, 'MerchantListItem.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('merchant list item provides shared content regions without scene-specific copy', () => {
  assert.equal(fs.existsSync(componentPath), true)
  for (const slot of ['leading', 'badge', 'meta', 'actions']) {
    assert.match(source, new RegExp(`name="${slot}"`))
  }
  assert.match(source, /defineEmits\(\['activate'\]\)/)
  assert.match(source, /title:\s*\{\s*type:\s*String,\s*required:\s*true/)
  assert.match(source, /subtitle:\s*\{\s*type:\s*String,\s*default:\s*''/)
  assert.match(source, /tags:\s*\{\s*type:\s*Array,\s*default:\s*\(\)\s*=>\s*\[\]/)
  assert.match(source, /selected:\s*\{\s*type:\s*Boolean,\s*default:\s*false/)
  assert.doesNotMatch(source, /新入驻|待认领|导航|入驻日期/)
})
```

在 `wxapp/components/MerchantPlaceCard.test.mjs` 增加：

```js
test('merchant place card composes the shared list item and preserves map actions', () => {
  assert.match(source, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
  assert.match(source, /<MerchantListItem/)
  assert.match(source, /#leading/)
  assert.match(source, /#badge/)
  assert.match(source, /#meta/)
  assert.match(source, /#actions/)
  assert.match(source, /@activate="\$emit\('select', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('detail', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('navigate', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('claim', place\)"/)
})
```

- [ ] **步骤 3：运行测试并确认失败**

运行：

```bash
cd wxapp
node --test components/MerchantListItem.test.mjs components/MerchantPlaceCard.test.mjs
```

预期：`MerchantListItem.vue` 不存在，新增测试失败；已有拿货地图测试继续反映当前行为。

- [ ] **步骤 4：实现最小基础组件**

新建 `wxapp/components/MerchantListItem.vue`，核心结构如下：

```vue
<template>
  <view
    :class="['merchant-list-item', { selected }]"
    role="button"
    @click="$emit('activate')"
  >
    <view class="merchant-list-leading">
      <slot name="leading" />
    </view>
    <view class="merchant-list-main">
      <view class="merchant-list-title-row">
        <text class="merchant-list-title">{{ title }}</text>
        <slot name="badge" />
      </view>
      <text v-if="subtitle" class="merchant-list-subtitle">{{ subtitle }}</text>
      <view v-if="tags.length" class="merchant-list-tags">
        <text v-for="tag in tags" :key="tag" class="merchant-list-tag">{{ tag }}</text>
      </view>
      <view class="merchant-list-foot">
        <slot name="meta" />
        <view class="merchant-list-actions">
          <slot name="actions" />
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  tags: { type: Array, default: () => [] },
  selected: { type: Boolean, default: false },
})

defineEmits(['activate'])
</script>

<style lang="scss" scoped>
.merchant-list-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 20rpx;
  padding: 24rpx;
  border: 1rpx solid rgba(148, 163, 184, 0.32);
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 22rpx rgba(15, 23, 42, 0.05);
}

.merchant-list-item.selected {
  border-color: rgba(194, 58, 0, 0.72);
  box-shadow: 0 10rpx 28rpx rgba(194, 58, 0, 0.12);
}

.merchant-list-main {
  min-width: 0;
}

.merchant-list-title-row,
.merchant-list-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14rpx;
}

.merchant-list-title {
  overflow: hidden;
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merchant-list-subtitle {
  display: block;
  margin-top: 10rpx;
  color: $wplink-muted;
  font-size: 23rpx;
  line-height: 1.55;
}

.merchant-list-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  margin-top: 14rpx;
}

.merchant-list-tag {
  padding: 5rpx 10rpx;
  border-radius: 6rpx;
  background: #f1f4f8;
  color: #526176;
  font-size: 20rpx;
}

.merchant-list-foot {
  min-height: 50rpx;
  margin-top: 16rpx;
}

.merchant-list-actions {
  display: flex;
  flex: none;
  gap: 8rpx;
}
</style>
```

调用方负责将标签截断到场景允许的数量；`actions` 内按钮必须继续使用 `.stop` 阻止冒泡。

- [ ] **步骤 5：将拿货地图卡片改为薄封装**

在 `MerchantPlaceCard.vue` 中：

```vue
<MerchantListItem
  :title="place.name"
  :subtitle="locationText"
  :tags="visibleTags"
  :selected="selected"
  @activate="$emit('select', place)"
>
  <template #leading>
    <view class="doorplate">
      <text class="doorplate-label">档口</text>
      <text class="doorplate-code">{{ place.code || '--' }}</text>
    </view>
  </template>
  <template #badge>
    <text :class="['source-badge', { claimed: place.claimed }]">{{ sourceLabel }}</text>
  </template>
  <template #meta>
    <text class="distance-text">{{ hasLocation ? (place.distanceText || '可导航到店') : '位置待完善' }}</text>
  </template>
  <template #actions>
    <button v-if="hasMerchantDetail(place)" class="detail-button" @click.stop="$emit('detail', place)">进入主页</button>
    <button v-if="hasLocation" class="navigate-button" @click.stop="$emit('navigate', place)">导航</button>
    <button v-if="!place.claimed" class="claim-button" @click.stop="$emit('claim', place)">这是我的档口</button>
  </template>
</MerchantListItem>
```

保留 `hasMerchantDetail`、`hasValidLocation`、`sourceLabel`、`visibleTags` 和 `locationText` 计算逻辑；删除已由基础组件承担的外层卡片、标题、位置、标签和底部布局样式，只保留门牌、状态徽标和按钮样式。

- [ ] **步骤 6：运行组件与地图定向测试**

运行：

```bash
cd wxapp
node --test \
  components/MerchantListItem.test.mjs \
  components/MerchantPlaceCard.test.mjs \
  pages/sourcing-map/merchantPlaceState.test.mjs \
  pages/sourcing-map/index.test.mjs
```

预期：全部通过；进入主页、导航和认领事件测试继续存在且通过。

- [ ] **步骤 7：提交基础组件改造**

仅在步骤 1 确认现有重叠改动已可安全纳入时运行：

```bash
git add \
  wxapp/components/MerchantListItem.vue \
  wxapp/components/MerchantListItem.test.mjs \
  wxapp/components/MerchantPlaceCard.vue \
  wxapp/components/MerchantPlaceCard.test.mjs
git commit -m "refactor: 共用商家列表基础组件"
```

预期：提交只包含基础组件和拿货地图卡片薄封装，不包含后台静态文件或其他页面改动。

---

### 任务 2：首页新入驻商家改为静态列表行

**文件：**

- 修改：`wxapp/components/HomeRecentMerchantCard.vue`
- 修改：`wxapp/components/HomeRecentMerchantCard.test.mjs`

**接口：**

- 继续接收 `merchant: Object`。
- 继续触发 `open(merchant)`。
- 消费任务 1 的 `MerchantListItem`。
- 不再读取或展示 `merchant.addressText`。

- [ ] **步骤 1：更新首页商家卡片失败测试**

将 `HomeRecentMerchantCard.test.mjs` 的展示断言调整为：

```js
test('recent merchant item composes the shared list style with onboarding context', () => {
  assert.equal(fs.existsSync(componentPath), true)
  assert.match(source, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
  assert.match(source, /<MerchantListItem/)
  assert.match(source, /:tags="visibleCategories"/)
  assert.match(source, /merchantTypeText/)
  assert.match(source, /formatListFreshnessDate/)
  assert.match(source, /新入驻/)
  assert.match(source, /logoUrl/)
  assert.match(source, /merchant-initial/)
  assert.doesNotMatch(source, /addressText/)
})

test('recent merchant item preserves its public click contract', () => {
  assert.match(source, /defineEmits\(\['open'\]\)/)
  assert.match(source, /@activate="\$emit\('open', merchant\)"/)
  assert.doesNotMatch(source, /phone|wechat|vip/i)
})
```

- [ ] **步骤 2：运行测试并确认失败**

运行：

```bash
cd wxapp
node --test components/HomeRecentMerchantCard.test.mjs
```

预期：因组件仍使用旧两列卡片结构并包含地址而失败。

- [ ] **步骤 3：改造首页商家薄封装**

将 `HomeRecentMerchantCard.vue` 改为：

```vue
<template>
  <MerchantListItem
    :title="merchant.name"
    :subtitle="merchantTypeLabel"
    :tags="visibleCategories"
    @activate="$emit('open', merchant)"
  >
    <template #leading>
      <image v-if="merchant.logoUrl" class="merchant-logo" :src="merchant.logoUrl" mode="aspectFill" />
      <view v-else class="merchant-initial">{{ merchantInitial }}</view>
    </template>
    <template #badge>
      <text class="new-badge">新入驻</text>
    </template>
    <template #meta>
      <text class="onboarded-date">{{ onboardedLabel }}</text>
    </template>
  </MerchantListItem>
</template>

<script setup>
import { computed } from 'vue'
import MerchantListItem from './MerchantListItem.vue'
import { formatListFreshnessDate } from '../common/date'
import { merchantTypeText } from '../common/enums'

const props = defineProps({
  merchant: { type: Object, required: true },
})

defineEmits(['open'])

const merchantInitial = computed(() => String(props.merchant.name || '商').trim().slice(0, 1) || '商')
const merchantTypeLabel = computed(() =>
  merchantTypeText[props.merchant.merchantType] || props.merchant.merchantType || '商家'
)
const visibleCategories = computed(() =>
  (props.merchant.mainCategories || []).filter(Boolean).slice(0, 2)
)
const onboardedLabel = computed(() =>
  `${formatListFreshnessDate(props.merchant.onboardedAt)}入驻`
)
</script>

<style lang="scss" scoped>
.merchant-logo,
.merchant-initial {
  width: 72rpx;
  height: 72rpx;
  border-radius: 12rpx;
}

.merchant-logo {
  display: block;
  background: #eef1f5;
}

.merchant-initial {
  display: grid;
  place-items: center;
  background: $wplink-primary;
  color: #ffffff;
  font-size: 32rpx;
  font-weight: 800;
}

.new-badge {
  padding: 5rpx 11rpx;
  border-radius: 5rpx;
  background: $wplink-warning;
  color: #ffffff;
  font-size: 19rpx;
  font-weight: 800;
}

.onboarded-date {
  color: $wplink-muted;
  font-family: "DIN Alternate", Arial, sans-serif;
  font-size: 20rpx;
  font-weight: 700;
}
</style>
```

不再定义基础组件已有的边框、标题、标签和布局样式。

- [ ] **步骤 4：运行首页商家组件测试**

运行：

```bash
cd wxapp
node --test components/MerchantListItem.test.mjs components/HomeRecentMerchantCard.test.mjs
```

预期：全部通过；源码中不再包含 `addressText`。

- [ ] **步骤 5：提交首页商家列表行**

```bash
git add \
  wxapp/components/HomeRecentMerchantCard.vue \
  wxapp/components/HomeRecentMerchantCard.test.mjs
git commit -m "refactor: 统一首页商家列表样式"
```

---

### 任务 3：首页增加商家与供需双标签

**文件：**

- 新建：`wxapp/pages/home/homeFeedState.js`
- 新建：`wxapp/pages/home/homeFeedState.test.mjs`
- 修改：`wxapp/pages/home/index.vue`
- 修改：`wxapp/pages/home/index.test.mjs`

**接口：**

- `getHomeFeedState({ merchantCount, resourceCount, hasRecommendCard })` 返回：

```js
{
  hasMerchants: Boolean,
  hasResources: Boolean,
  hasAnyContent: Boolean,
  showSwitcher: Boolean,
  defaultTab: 'merchants' | 'resources' | '',
}
```

- 首页本地状态 `activeHomeFeedTab` 只允许 `'merchants'` 或 `'resources'`。
- `selectHomeFeedTab(tab)` 只切换当前可用标签，不触发接口调用。

- [ ] **步骤 1：为内容状态纯函数编写失败测试**

新建 `wxapp/pages/home/homeFeedState.test.mjs`：

```js
import assert from 'node:assert/strict'
import test from 'node:test'
import { getHomeFeedState } from './homeFeedState.js'

test('both feeds show the switcher and prefer recent merchants', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 6, resourceCount: 8, hasRecommendCard: true }),
    {
      hasMerchants: true,
      hasResources: true,
      hasAnyContent: true,
      showSwitcher: true,
      defaultTab: 'merchants',
    }
  )
})

test('resources become the only visible feed when merchants are unavailable', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 0, resourceCount: 1, hasRecommendCard: false }),
    {
      hasMerchants: false,
      hasResources: true,
      hasAnyContent: true,
      showSwitcher: false,
      defaultTab: 'resources',
    }
  )
})

test('a recommendation card counts as resource content', () => {
  assert.equal(
    getHomeFeedState({ merchantCount: 0, resourceCount: 0, hasRecommendCard: true }).defaultTab,
    'resources'
  )
})

test('empty feeds hide the whole home feed section', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 0, resourceCount: 0, hasRecommendCard: false }),
    {
      hasMerchants: false,
      hasResources: false,
      hasAnyContent: false,
      showSwitcher: false,
      defaultTab: '',
    }
  )
})
```

- [ ] **步骤 2：运行纯函数测试并确认失败**

运行：

```bash
cd wxapp
node --test pages/home/homeFeedState.test.mjs
```

预期：模块不存在，测试失败。

- [ ] **步骤 3：实现内容状态纯函数**

新建 `wxapp/pages/home/homeFeedState.js`：

```js
export function getHomeFeedState({
  merchantCount = 0,
  resourceCount = 0,
  hasRecommendCard = false,
} = {}) {
  const hasMerchants = Number(merchantCount) > 0
  const hasResources = Number(resourceCount) > 0 || Boolean(hasRecommendCard)
  return {
    hasMerchants,
    hasResources,
    hasAnyContent: hasMerchants || hasResources,
    showSwitcher: hasMerchants && hasResources,
    defaultTab: hasMerchants ? 'merchants' : hasResources ? 'resources' : '',
  }
}
```

- [ ] **步骤 4：更新首页结构测试**

在 `wxapp/pages/home/index.test.mjs` 中：

1. 保留最多 6 家、并行请求、接口静默失败及跳转断言；
2. 将旧的“两个纵向区块顺序”测试替换为：

```js
test('home combines recent merchants and resources into a local tabbed feed', () => {
  assert.match(source, /import \{ getHomeFeedState \} from '\.\/homeFeedState'/)
  assert.match(source, /const activeHomeFeedTab = ref\(''\)/)
  assert.match(source, /class="home-feed-tabs"/)
  assert.match(source, />新入驻商家</)
  assert.match(source, />近期供需</)
  assert.match(source, /selectHomeFeedTab\('merchants'\)/)
  assert.match(source, /selectHomeFeedTab\('resources'\)/)
  assert.match(source, /v-if="activeHomeFeedTab === 'merchants'"/)
  assert.match(source, /v-else-if="activeHomeFeedTab === 'resources'"/)
})

test('home initializes the feed tab after parallel data loading without refetching on switch', () => {
  assert.match(source, /await Promise\.all\(\[loadHomeOperationConfig\(\), loadHomeRecentMerchants\(\), loadHomeResources\(\)\]\)/)
  assert.match(source, /activeHomeFeedTab\.value = homeFeedState\.value\.defaultTab/)
  assert.match(source, /function selectHomeFeedTab\(tab\)/)
  const switchFunction = source.match(/function selectHomeFeedTab\(tab\) \{[\s\S]*?\n\}/)?.[0] || ''
  assert.doesNotMatch(switchFunction, /loadHomeRecentMerchants|loadHomeResources|listHome/)
})

test('home degrades to a single available feed and hides an empty feed container', () => {
  assert.match(source, /v-if="homeFeedState\.hasAnyContent"/)
  assert.match(source, /v-if="homeFeedState\.showSwitcher"/)
  assert.match(source, /homeFeedState\.hasMerchants/)
  assert.match(source, /homeFeedState\.hasResources/)
})
```

- [ ] **步骤 5：运行首页测试并确认失败**

运行：

```bash
cd wxapp
node --test pages/home/homeFeedState.test.mjs pages/home/index.test.mjs
```

预期：状态纯函数通过，首页仍是两个独立纵向区块，因此首页结构测试失败。

- [ ] **步骤 6：实现首页标签状态**

在 `wxapp/pages/home/index.vue`：

```js
import { getHomeFeedState } from './homeFeedState'

const activeHomeFeedTab = ref('')
const homeFeedState = computed(() => getHomeFeedState({
  merchantCount: recentMerchants.value.length,
  resourceCount: homeResources.value.length,
  hasRecommendCard: Boolean(displayRecommendCard.value),
}))

async function loadHomeData() {
  await Promise.all([loadHomeOperationConfig(), loadHomeRecentMerchants(), loadHomeResources()])
  activeHomeFeedTab.value = homeFeedState.value.defaultTab
}

function selectHomeFeedTab(tab) {
  if (tab === 'merchants' && !homeFeedState.value.hasMerchants) return
  if (tab === 'resources' && !homeFeedState.value.hasResources) return
  activeHomeFeedTab.value = tab
}
```

不要在 `selectHomeFeedTab` 中调用任何接口。`loadHomeRecentMerchants` 继续使用 `RECENT_MERCHANT_LIMIT = 6` 截断响应。

- [ ] **步骤 7：合并首页两个内容区**

将快捷入口之后的两个独立区块替换为：

```vue
<view v-if="homeFeedState.hasAnyContent" class="home-feed-section">
  <view v-if="homeFeedState.showSwitcher" class="home-feed-tabs">
    <button
      :class="['home-feed-tab', { active: activeHomeFeedTab === 'merchants' }]"
      @click="selectHomeFeedTab('merchants')"
    >新入驻商家</button>
    <button
      :class="['home-feed-tab', { active: activeHomeFeedTab === 'resources' }]"
      @click="selectHomeFeedTab('resources')"
    >近期供需</button>
  </view>

  <view v-else class="section-head home-feed-single-head">
    <text class="section-title">
      {{ homeFeedState.hasMerchants ? '新入驻商家' : '近期供需' }}
    </text>
  </view>

  <view v-if="activeHomeFeedTab === 'merchants'" class="recent-merchant-list">
    <HomeRecentMerchantCard
      v-for="item in recentMerchants"
      :key="item.id"
      :merchant="item"
      @open="openMerchant"
    />
    <button class="home-feed-more" @click="openSourcingMap()">查看更多商家</button>
  </view>

  <view v-else-if="activeHomeFeedTab === 'resources'" class="home-resource-panel">
    <view class="recommend-card" v-if="displayRecommendCard" @click="openRecommendCard(displayRecommendCard)">
      <view>
        <text class="recommend-tag">{{ displayRecommendCard.tag || '热门场景' }}</text>
        <text class="recommend-title">{{ displayRecommendCard.title }}</text>
        <text v-if="displayRecommendCard.subtitle" class="recommend-desc">{{ displayRecommendCard.subtitle }}</text>
      </view>
      <text class="recommend-action">查看</text>
    </view>
    <view v-if="homeResources.length" class="home-resource-list">
      <ResourceExposure
        v-for="item in homeResources"
        :key="item.id"
        :resource-id="item.id"
        source="home"
      >
        <ResourceCard
          :resource="item"
          variant="home"
          @open="openResource"
        />
      </ResourceExposure>
    </view>
    <button class="home-feed-more" @click="openSearch()">查看更多供需</button>
  </view>
</view>
```

推荐卡和供需卡内部现有模板必须原样迁移，不改字段、事件或曝光组件。

新增样式：

```scss
.home-feed-section {
  margin-bottom: 12rpx;
}

.home-feed-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8rpx;
  margin: 0 0 20rpx;
  padding: 6rpx;
  border: 1rpx solid rgba(148, 163, 184, 0.28);
  border-radius: 16rpx;
  background: #e8eeee;
}

.home-feed-tab {
  margin: 0;
  padding: 16rpx 8rpx;
  border: 0;
  border-radius: 12rpx;
  background: transparent;
  color: $wplink-muted;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 1.3;
}

.home-feed-tab.active {
  background: #ffffff;
  color: $wplink-primary;
  box-shadow: 0 6rpx 18rpx rgba(15, 23, 42, 0.08);
}

.home-feed-tab::after,
.home-feed-more::after {
  border: 0;
}

.recent-merchant-list,
.home-resource-panel {
  display: grid;
  gap: 12rpx;
}

.home-feed-more {
  width: 100%;
  margin: 0;
  padding: 18rpx;
  border: 0;
  background: transparent;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.3;
}
```

删除旧 `.recent-merchant-grid` 两列样式；单内容降级继续复用现有 `.section-head` 和 `.section-title`。

- [ ] **步骤 8：运行首页及组件定向测试**

运行：

```bash
cd wxapp
node --test \
  pages/home/homeFeedState.test.mjs \
  pages/home/index.test.mjs \
  components/MerchantListItem.test.mjs \
  components/HomeRecentMerchantCard.test.mjs \
  components/MerchantPlaceCard.test.mjs \
  pages/sourcing-map/merchantPlaceState.test.mjs \
  pages/sourcing-map/index.test.mjs
```

预期：全部通过。

- [ ] **步骤 9：运行小程序全量验证**

运行：

```bash
cd wxapp
npm run check
```

预期：

- 页面配置校验通过；
- 流程校验通过；
- 所有 `node:test` 测试通过；
- `mp-weixin` 构建成功；
- 只允许出现项目已有的 Sass 弃用提醒或 uni-app 更新提示，不允许新增编译错误和运行时警告。

- [ ] **步骤 10：检查最终差异和改动范围**

运行：

```bash
git diff --check
git status --short
git diff -- \
  wxapp/pages/home/homeFeedState.js \
  wxapp/pages/home/homeFeedState.test.mjs \
  wxapp/pages/home/index.vue \
  wxapp/pages/home/index.test.mjs
```

预期：没有空白错误；后台静态文件和其他用户改动保持原状；首页差异只涉及标签状态、模板组合和对应样式。

- [ ] **步骤 11：提交首页双标签**

```bash
git add \
  wxapp/pages/home/homeFeedState.js \
  wxapp/pages/home/homeFeedState.test.mjs \
  wxapp/pages/home/index.vue \
  wxapp/pages/home/index.test.mjs
git commit -m "feat: 首页切换商家与近期供需"
```

提交后运行：

```bash
git status --short
git log -4 --oneline
```

预期：三个功能提交清晰可审查，工作区仅保留实施前已有的无关改动。
