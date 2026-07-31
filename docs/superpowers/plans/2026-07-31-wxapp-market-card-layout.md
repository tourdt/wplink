# 小程序供需市场列表卡片精简实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 为供需市场页增加 C1 市场专用卡片布局，通过彩色业务类型角标和三行正文突出价格、预算与数量，同时移除地区和资源标签。

**架构：** 在现有 `ResourceCard` 和 `DemandCard` 中增加 `variant="market"` 分支，市场分支复用既有字段计算与点击行为，默认分支保持原样。供需市场页显式传入该变体，确保首页、搜索、收藏、专题和通用列表不受影响。

**技术栈：** uni-app、Vue 3 `<script setup>`、SCSS、Node.js `node:test` 静态结构测试、现有小程序构建流程。

## 全局约束

- 设计规格：`docs/superpowers/specs/2026-07-31-wxapp-market-card-layout-design.md`。
- 所有实施说明、计划和新增注释使用中文；代码标识、路径和命令保留英文。
- 只修改供需市场卡片展示，不修改接口、字段结构、分页、排序、曝光统计、详情跳转和筛选逻辑。
- 供需市场正文必须保持“标题、数量与价格、商户与更新时间”三行结构。
- 市场变体不得显示独立的“供应 / 需求”方向角标、地区、品类摘要和资源标签。
- 供应业务类型角标使用深色，需求业务类型角标使用橙色。
- 首页、搜索、收藏、专题和通用列表未传入 `variant="market"` 时必须保持现有布局。
- 不新增依赖、图片、图标或后端配置。
- 保留工作区中与本需求无关的既有改动，不格式化或提交无关文件。

---

## 文件结构

- 修改 `wxapp/components/ResourceCard.vue`：增加供应卡市场变体模板、计算状态和局部样式，保留默认、首页和紧凑变体。
- 修改 `wxapp/components/DemandCard.vue`：增加与供应卡一致的 `variant` 接口及需求卡市场变体模板、计算状态和局部样式。
- 修改 `wxapp/components/ResourceCard.test.mjs`：覆盖两个组件的市场结构、颜色、缺失交易信息兜底和默认结构回归。
- 修改 `wxapp/pages/market/index.vue`：仅在供需市场结果列表中为两个卡片传入 `variant="market"`。
- 修改 `wxapp/pages/market/index.test.mjs`：验证市场页显式启用市场变体，混合供需渲染和既有行为不变。

### Task 1：实现供应卡市场专用变体

**文件：**

- 修改：`wxapp/components/ResourceCard.test.mjs`
- 修改：`wxapp/components/ResourceCard.vue`

**接口：**

- 输入：`ResourceCard` 现有 `resource: Object` 与 `variant: String` 属性。
- 产出：`variant="market"`；计算状态 `isMarketVariant`、`quantityText`、`hasMarketTradeInfo`；市场专用类名 `market-type-badge supply`、`market-completed-badge`、`market-title`、`market-decision-line`、`market-merchant-line`。
- 保持：`variant=""`、`variant="home"`、`variant="compact"` 的既有结构和样式。

- [ ] **Step 1：编写供应卡市场变体失败测试**

在 `wxapp/components/ResourceCard.test.mjs` 增加测试，断言：

```js
test('resource card provides a compact market-only layout', () => {
  assert.match(source, /if \(props\.variant === 'market'\) return 'resource-card-market'/)
  assert.match(source, /const isMarketVariant = computed\(\(\) => props\.variant === 'market'\)/)
  assert.match(source, /const quantityText = computed\(\(\) => String\(props\.resource\.quantityText \|\| ''\)\.trim\(\)\)/)
  assert.match(source, /const hasMarketTradeInfo = computed/)
  assert.match(source, /<template v-if="isMarketVariant">[\s\S]*class="market-type-badge supply"[\s\S]*\{\{ resourceTypeLabel \}\}[\s\S]*class="market-title"[\s\S]*class="market-decision-line"[\s\S]*class="market-merchant-line"[\s\S]*<\/template>/)
  assert.match(source, /<text v-if="isCompleted" class="market-completed-badge">已完成<\/text>/)
  assert.match(source, /<text v-if="quantityText" class="market-quantity">\{\{ quantityText \}\}<\/text>/)
  assert.match(source, /<text v-if="resource\.priceText" class="market-price">\{\{ resource\.priceText \}\}<\/text>/)
  assert.match(source, /<text v-else class="market-trade-empty">交易信息待完善<\/text>/)
  assert.match(source, /\.resource-card-market \{[\s\S]*padding: 20rpx;/)
  assert.match(source, /\.resource-card-market \.thumb-wrap \{[\s\S]*width: 152rpx;[\s\S]*height: 152rpx;/)
  assert.match(source, /\.market-type-badge\.supply \{[\s\S]*background: \$wplink-primary;/)
})
```

保留现有默认结构测试，并补充断言，确保市场结构之外仍存在：

```js
assert.match(source, /<template v-else>[\s\S]*class="direction-badge supply"[\s\S]*class="resource-meta"[\s\S]*class="location-text"[\s\S]*class="resource-labels"/)
```

- [ ] **Step 2：运行供应卡测试并确认失败**

工作目录：`wxapp`

运行：

```bash
node --test components/ResourceCard.test.mjs
```

预期：新增的 `resource-card-market`、`isMarketVariant` 或市场模板断言失败；既有测试继续通过。

- [ ] **Step 3：实现供应卡市场模板和计算状态**

在 `ResourceCard.vue` 根节点内按变体分支渲染。把以下市场模板插入根节点开头：

```vue
<template v-if="isMarketVariant">
  <view class="thumb-wrap">
    <image class="resource-thumb" :src="coverUrl || DEFAULT_RESOURCE_COVER" mode="aspectFill" />
    <text v-if="resourceTypeLabel" class="market-type-badge supply">{{ resourceTypeLabel }}</text>
    <text v-if="isCompleted" class="market-completed-badge">已完成</text>
  </view>
  <view class="market-card-main">
    <text class="market-title">{{ resource.title || '供应标题待完善' }}</text>
    <view v-if="hasMarketTradeInfo" class="market-decision-line">
      <text v-if="quantityText" class="market-quantity">{{ quantityText }}</text>
      <text v-if="resource.priceText" class="market-price">{{ resource.priceText }}</text>
    </view>
    <text v-else class="market-trade-empty">交易信息待完善</text>
    <view class="market-merchant-line">
      <text class="merchant-name">{{ merchantName }}</text>
      <text v-if="freshnessText" class="refresh-time">{{ freshnessText }}</text>
    </view>
  </view>
</template>
```

然后把当前根节点中从 `<view class="thumb-wrap">` 到对应 `<view class="card-main">...</view>` 的两个既有兄弟节点整体包入 `<template v-else>...</template>`；节点内容、顺序和属性不变。

扩展现有计算属性：

```js
const isMarketVariant = computed(() => props.variant === 'market')

const variantClass = computed(() => {
  if (props.variant === 'home') return 'resource-card-home'
  if (props.variant === 'compact') return 'resource-card-compact'
  if (props.variant === 'market') return 'resource-card-market'
  return ''
})

const quantityText = computed(() => String(props.resource.quantityText || '').trim())
const hasMarketTradeInfo = computed(() => Boolean(quantityText.value || props.resource.priceText))
```

市场模板不得引用 `locationText`、`resourceSummaryText` 或 `resourceLabels`，但这些计算属性继续为默认模板服务，不能删除。

- [ ] **Step 4：实现供应卡市场局部样式**

在 `ResourceCard.vue` scoped SCSS 末尾增加以下市场样式。角标需要保证完成状态完整，业务类型在剩余空间内省略：

```scss
.resource-card-market {
  align-items: flex-start;
  gap: 12rpx;
  padding: 20rpx;
}

.resource-card-market .thumb-wrap {
  flex-basis: 152rpx;
  width: 152rpx;
  height: 152rpx;
}

.market-type-badge,
.market-completed-badge {
  position: absolute;
  top: 8rpx;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 8rpx;
  color: #fff;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1;
}

.market-type-badge {
  left: 8rpx;
  right: 66rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.market-type-badge.supply {
  background: $wplink-primary;
}

.market-completed-badge {
  right: 8rpx;
  background: rgba(71, 85, 105, 0.92);
}

.market-card-main {
  display: grid;
  flex: 1;
  align-self: stretch;
  align-content: space-between;
  min-width: 0;
}

.market-title,
.market-quantity,
.market-price,
.market-trade-empty {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.market-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.market-decision-line,
.market-merchant-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  min-width: 0;
}

.market-quantity {
  min-width: 0;
  color: #475569;
  font-size: 25rpx;
  font-weight: 600;
}

.market-price {
  min-width: 0;
  margin-left: auto;
  color: $wplink-warning;
  font-size: 28rpx;
  font-weight: 700;
  text-align: right;
}

.market-trade-empty {
  color: $wplink-muted;
  font-size: 24rpx;
}
```

`market-merchant-line` 复用现有 `.merchant-name` 和 `.refresh-time`，必要时为两者补充 `min-width` 与 `flex`，但不能改变非市场变体的排版。

- [ ] **Step 5：运行供应卡测试并确认通过**

工作目录：`wxapp`

运行：

```bash
node --test components/ResourceCard.test.mjs
```

预期：全部通过，供应市场结构和默认结构回归断言均为 PASS。

- [ ] **Step 6：提交供应卡市场变体**

```bash
git add wxapp/components/ResourceCard.vue wxapp/components/ResourceCard.test.mjs
git commit -m "feat: 增加供应市场精简卡片"
```

### Task 2：实现需求卡市场专用变体

**文件：**

- 修改：`wxapp/components/ResourceCard.test.mjs`
- 修改：`wxapp/components/DemandCard.vue`

**接口：**

- 输入：`DemandCard` 现有 `resource: Object`；新增 `variant: String`，默认值为空字符串。
- 产出：`variant="market"`；计算状态 `variantClass`、`isMarketVariant`、`quantityText`、`hasMarketTradeInfo`；市场专用类名与供应卡一致，需求角标额外使用 `demand` 修饰类。
- 保持：未传入 `variant` 时需求卡现有模板和样式不变。

- [ ] **Step 1：编写需求卡市场变体失败测试**

在 `ResourceCard.test.mjs` 增加：

```js
test('demand card provides the matching compact market-only layout', () => {
  assert.match(demandSource, /variant:\s*\{[\s\S]*type: String,[\s\S]*default: '',[\s\S]*\}/)
  assert.match(demandSource, /const variantClass = computed\(\(\) => props\.variant === 'market' \? 'demand-card-market' : ''\)/)
  assert.match(demandSource, /const isMarketVariant = computed\(\(\) => props\.variant === 'market'\)/)
  assert.match(demandSource, /const quantityText = computed\(\(\) => String\(props\.resource\.quantityText \|\| ''\)\.trim\(\)\)/)
  assert.match(demandSource, /<template v-if="isMarketVariant">[\s\S]*class="market-type-badge demand"[\s\S]*class="market-title"[\s\S]*class="market-decision-line"[\s\S]*class="market-merchant-line"[\s\S]*<\/template>/)
  assert.match(demandSource, /<text v-if="isCompleted" class="market-completed-badge">已完成<\/text>/)
  assert.match(demandSource, /\.demand-card-market \{[\s\S]*padding: 20rpx;/)
  assert.match(demandSource, /\.demand-card-market \.thumb-wrap \{[\s\S]*width: 152rpx;[\s\S]*height: 152rpx;/)
  assert.match(demandSource, /\.market-type-badge\.demand \{[\s\S]*background: \$wplink-warning;/)
  assert.match(demandSource, /<template v-else>[\s\S]*class="direction-badge"[\s\S]*class="demand-meta"[\s\S]*class="location-text"[\s\S]*class="resource-labels"/)
})
```

- [ ] **Step 2：运行需求卡测试并确认失败**

工作目录：`wxapp`

运行：

```bash
node --test components/ResourceCard.test.mjs
```

预期：需求卡缺少 `variant`、`demand-card-market` 或市场模板导致新增测试失败。

- [ ] **Step 3：实现需求卡属性、模板与计算状态**

将根节点改为：

```vue
<view :class="['demand-card', variantClass]" @click="$emit('open', resource)">
```

新增属性和计算状态：

```js
const props = defineProps({
  resource: {
    type: Object,
    required: true,
  },
  variant: {
    type: String,
    default: '',
  },
})

const variantClass = computed(() => props.variant === 'market' ? 'demand-card-market' : '')
const isMarketVariant = computed(() => props.variant === 'market')
const quantityText = computed(() => String(props.resource.quantityText || '').trim())
const hasMarketTradeInfo = computed(() => Boolean(quantityText.value || props.resource.priceText))
```

市场分支与供应卡保持相同结构，仅替换以下内容：

```vue
<text v-if="resourceTypeLabel" class="market-type-badge demand">{{ resourceTypeLabel }}</text>
<text class="market-title">{{ resource.title || '需求标题待完善' }}</text>
<text v-if="resource.priceText" class="market-price">{{ resource.priceText }}</text>
```

商户名称继续复用现有 `merchantName`，缺失时显示“采购方待确认”。默认需求模板完整放入 `v-else` 且不删除地区、摘要或标签逻辑。

- [ ] **Step 4：实现需求卡市场局部样式**

复制供应卡市场结构的尺寸、三行排版和截断规则到 `DemandCard.vue`，仅保留需求方向差异：

```scss
.demand-card-market {
  align-items: flex-start;
  gap: 12rpx;
  padding: 20rpx;
}

.demand-card-market .thumb-wrap {
  position: relative;
  flex-basis: 152rpx;
  width: 152rpx;
  height: 152rpx;
}

.market-type-badge.demand {
  background: $wplink-warning;
}
```

需求卡继续保留现有极浅橙色边框。其余 `.market-*` 样式值与 Task 1 一致，避免同一列表出现卡片高度、字号或间距差异。

- [ ] **Step 5：运行组件测试并确认通过**

工作目录：`wxapp`

运行：

```bash
node --test components/ResourceCard.test.mjs
```

预期：供应和需求市场变体、默认结构、封面、类型解析和更新时间测试全部通过。

- [ ] **Step 6：提交需求卡市场变体**

```bash
git add wxapp/components/DemandCard.vue wxapp/components/ResourceCard.test.mjs
git commit -m "feat: 增加需求市场精简卡片"
```

### Task 3：接入供需市场并完成回归验证

**文件：**

- 修改：`wxapp/pages/market/index.test.mjs`
- 修改：`wxapp/pages/market/index.vue`
- 验证：`wxapp/scripts/validate-flows.test.mjs`

**接口：**

- 输入：Task 1 和 Task 2 提供的 `ResourceCard variant="market"` 与 `DemandCard variant="market"`。
- 产出：供需市场混合列表显式启用精简卡片；其他页面不传入该变体。
- 保持：`ResourceExposure` 包裹、`RESOURCE_DIRECTION_DEMAND` 分流、`openResource` 事件、分页和曝光来源不变。

- [ ] **Step 1：编写市场页接入失败测试**

更新 `market page renders mixed supply and demand result cards from item direction`，增加精确断言：

```js
assert.match(
  source,
  /<DemandCard[\s\S]*v-if="item\.direction === RESOURCE_DIRECTION_DEMAND"[\s\S]*:resource="item"[\s\S]*variant="market"[\s\S]*@open="openResource"[\s\S]*\/>/,
)
assert.match(
  source,
  /<ResourceCard[\s\S]*v-else[\s\S]*:resource="item"[\s\S]*variant="market"[\s\S]*@open="openResource"[\s\S]*\/>/,
)
```

增加作用范围回归测试，读取 `pages/search/index.vue`、`pages/home/index.vue`、`pages/favorites/index.vue` 和 `pages/topic/index.vue`，断言这些页面不包含 `variant="market"`：

```js
for (const relativePath of [
  'pages/search/index.vue',
  'pages/home/index.vue',
  'pages/favorites/index.vue',
  'pages/topic/index.vue',
]) {
  const pageSource = fs.readFileSync(path.join(root, relativePath), 'utf8')
  assert.equal(pageSource.includes('variant="market"'), false, `${relativePath} should keep its existing card layout`)
}
```

- [ ] **Step 2：运行市场页测试并确认失败**

工作目录：`wxapp`

运行：

```bash
node --test pages/market/index.test.mjs
```

预期：市场页尚未传入 `variant="market"`，新增接入断言失败。

- [ ] **Step 3：为市场页两个卡片传入市场变体**

仅修改 `wxapp/pages/market/index.vue` 结果列表：

```vue
<ResourceExposure :resource-id="item.id" source="list">
  <DemandCard
    v-if="item.direction === RESOURCE_DIRECTION_DEMAND"
    :resource="item"
    variant="market"
    @open="openResource"
  />
  <ResourceCard
    v-else
    :resource="item"
    variant="market"
    @open="openResource"
  />
</ResourceExposure>
```

不得修改 `ResourceExposure`、列表键值、资源方向判断或 `openResource`。

- [ ] **Step 4：运行定向测试**

工作目录：`wxapp`

运行：

```bash
node --test components/ResourceCard.test.mjs pages/market/index.test.mjs scripts/validate-flows.test.mjs
```

预期：全部通过；市场页启用精简卡片，既有流程校验未出现回归。

- [ ] **Step 5：运行小程序完整检查**

工作目录：`wxapp`

运行：

```bash
npm run check
```

预期：

- 页面配置验证通过。
- 流程验证通过。
- 全部 Node 测试通过。
- `mp-weixin` 构建成功。
- 如只出现项目既有 Sass deprecation warning，可记录但不视为失败；不得忽略测试失败、编译错误或新增警告。

- [ ] **Step 6：检查精准改动范围**

运行：

```bash
git diff --check
git status --short
git diff -- wxapp/components/ResourceCard.vue wxapp/components/DemandCard.vue wxapp/components/ResourceCard.test.mjs wxapp/pages/market/index.vue wxapp/pages/market/index.test.mjs
```

确认：

- 没有空白错误。
- 本需求只改动计划列出的五个小程序文件。
- 工作区原有后台构建产物和商户地图相关改动未被暂存。
- 市场模板不引用 `locationText`、`resourceSummaryText` 或 `resourceLabels`。
- 默认模板仍保留地区、摘要和资源标签。

- [ ] **Step 7：提交市场页接入**

```bash
git add wxapp/pages/market/index.vue wxapp/pages/market/index.test.mjs
git commit -m "feat: 优化供需市场列表布局"
```

- [ ] **Step 8：最终提交与结果核对**

运行：

```bash
git log -3 --oneline
git status --short
```

预期最近三条提交依次包含：

- `feat: 增加供应市场精简卡片`
- `feat: 增加需求市场精简卡片`
- `feat: 优化供需市场列表布局`

最终状态允许存在用户原有未提交改动，但本需求文件不得残留未提交内容。
