import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'components/ResourceCard.vue'), 'utf8')
const demandSource = fs.readFileSync(path.join(root, 'components/DemandCard.vue'), 'utf8')

test('resource and demand cards display dynamic resource type names', () => {
  assert.match(source, /import \{ resourceTypeLabel as resolveResourceTypeLabel \} from '\.\.\/common\/resourceCategories'/)
  assert.match(source, /const resourceTypeLabel = computed/)
  assert.match(source, /resolveResourceTypeLabel\(props\.resource\)/)
  assert.match(source, /<text v-if="resourceTypeLabel" class="type-badge">\{\{ resourceTypeLabel \}\}<\/text>/)
  assert.match(demandSource, /import \{ resourceTypeLabel as resolveResourceTypeLabel \} from '\.\.\/common\/resourceCategories'/)
  assert.match(demandSource, /resolveResourceTypeLabel\(props\.resource\)/)
  assert.match(demandSource, /<text v-if="resourceTypeLabel" class="type-badge">\{\{ resourceTypeLabel \}\}<\/text>/)
  assert.equal(source.includes('resourceTypeText[props.resource.typeCode]'), false)
  assert.equal(demandSource.includes('resourceTypeText[props.resource.typeCode]'), false)
  assert.equal(source.includes('{{ resource.typeCode }}'), false)
})

test('resource card uses the default cover when resource image is missing', () => {
  assert.match(source, /const DEFAULT_RESOURCE_COVER = '\/static\/resource\/default-resource-cover\.png'/)
  assert.match(source, /<image class="resource-thumb" :src="coverUrl \|\| DEFAULT_RESOURCE_COVER" mode="aspectFill" \/>/)
  assert.match(demandSource, /const DEFAULT_RESOURCE_COVER = '\/static\/resource\/default-resource-cover\.png'/)
  assert.match(demandSource, /<image class="resource-thumb" :src="coverUrl \|\| DEFAULT_RESOURCE_COVER" mode="aspectFill" \/>/)
  assert.match(demandSource, /const coverUrl = computed\(\(\) => \{[\s\S]*return props\.resource\.coverUrl \|\| images\[0\] \|\| ''[\s\S]*\}/)
  assert.equal(source.includes('placeholder-thumb'), false)
  assert.equal(source.includes('placeholderLabel'), false)
  assert.equal(source.includes('供需信息图片'), false)
  assert.equal(source.includes('props.resource.typeCode || props.resource.category'), false)
  assert.equal(source.includes('type-corner'), false)
})

test('resource card image slot keeps a fixed square size instead of stretching with content', () => {
  assert.match(source, /\.thumb-wrap \{[\s\S]*align-self: flex-start;[\s\S]*width: 168rpx;[\s\S]*height: 168rpx;[\s\S]*min-height: 0;/)
  assert.match(source, /\.resource-card-home \.thumb-wrap \{[\s\S]*width: 160rpx;[\s\S]*height: 160rpx;[\s\S]*min-height: 0;/)
  assert.match(source, /\.resource-card-compact \.thumb-wrap \{[\s\S]*width: 144rpx;[\s\S]*height: 144rpx;[\s\S]*min-height: 0;/)
  assert.equal(source.includes('min-height: 168rpx'), false)
  assert.equal(source.includes('min-height: 160rpx'), false)
  assert.equal(source.includes('min-height: 144rpx'), false)
})

test('resource type label stays visually secondary in the card header', () => {
  assert.match(source, /\.direction-badge,[\s\S]*\.type-badge \{[\s\S]*height: 36rpx;[\s\S]*border-radius: 8rpx;[\s\S]*font-size: 22rpx;/)
  assert.match(source, /\.type-badge \{[\s\S]*max-width: 168rpx;[\s\S]*text-overflow: ellipsis;/)
  assert.match(source, /\.resource-card-home \.direction-badge,[\s\S]*\.resource-card-home \.type-badge \{[\s\S]*font-size: 20rpx;/)
  assert.match(source, /\.resource-card-compact \.direction-badge,[\s\S]*\.resource-card-compact \.type-badge \{[\s\S]*font-size: 18rpx;/)
})

test('resource card uses a readable four-line content layout', () => {
  assert.match(source, /<view class="card-head">[\s\S]*<text class="direction-badge supply">供应<\/text>[\s\S]*<text v-if="resourceTypeLabel" class="type-badge">\{\{ resourceTypeLabel \}\}<\/text>[\s\S]*<text v-if="freshnessText" class="refresh-time">\{\{ freshnessText \}\}<\/text>[\s\S]*<\/view>/)
  assert.match(source, /<text class="resource-title">\{\{ resource\.title \|\| '供应标题待完善' \}\}<\/text>[\s\S]*<text class="resource-meta">\{\{ resourceSummaryText \}\}<\/text>[\s\S]*<view v-if="resource\.priceText \|\| locationText" class="value-line">[\s\S]*<text v-if="resource\.priceText" class="resource-price">\{\{ resource\.priceText \}\}<\/text>[\s\S]*<text v-if="locationText" class="location-text">\{\{ locationText \}\}<\/text>[\s\S]*<view class="merchant-line">/)
  assert.match(source, /const resourceSummaryText = computed/)
  assert.match(source, /buildResourceSummaryText/)
  assert.match(source, /const locationText = computed\(\(\) => String\(props\.resource\.district \|\| ''\)\.trim\(\)\)/)
  assert.match(source, /<view class="merchant-line">[\s\S]*<text class="merchant-name">\{\{ merchantName \}\}<\/text>[\s\S]*<\/view>/)
  assert.match(demandSource, /<view class="thumb-wrap">[\s\S]*<view class="card-main">[\s\S]*<text class="direction-badge">需求<\/text>[\s\S]*<text v-if="resourceTypeLabel" class="type-badge">\{\{ resourceTypeLabel \}\}<\/text>/)
  assert.equal(source.includes('品类待沟通'), false)
  assert.equal(source.includes('数量待沟通'), false)
  assert.equal(source.includes('价格面议'), false)
  assert.equal(source.includes('已认证'), false)
  assert.equal(source.includes('isVIPMerchant'), false)
  assert.equal(source.includes('vip-badge'), false)
  assert.equal(source.includes('VIP'), false)
  assert.equal(demandSource.includes('已认证'), false)
  assert.equal(demandSource.includes('isVerifiedMerchant'), false)
  assert.equal(demandSource.includes('verificationStatus'), false)
  assert.equal(demandSource.includes('verified-badge'), false)
  assert.equal(demandSource.includes('VIP'), false)
  assert.equal(source.includes('meta-price-line'), false)
  assert.equal(source.includes('平台核实'), false)
  assert.equal(source.includes('hasCreditTags'), false)
  assert.equal(source.includes('查看详情'), false)
  assert.equal(source.includes('decision-tip'), false)
  assert.equal(source.includes('card-foot'), false)
  assert.equal(source.includes('tag-row'), false)
  assert.equal(source.includes('merchant-row'), false)
})

test('legacy detailed cards delegate compact market rendering to ResourceFeedCard', () => {
  assert.doesNotMatch(source, /isMarketVariant|resource-card-market|market-card-main|market-type-badge/)
  assert.doesNotMatch(demandSource, /isMarketVariant|demand-card-market|market-card-main|market-type-badge/)
})

test('resource card uses short freshness date in resource list', () => {
  assert.match(source, /import \{ formatListFreshnessDate \} from '\.\.\/common\/date'/)
  assert.match(source, /const freshnessText = computed\(\(\) => formatRefreshedAt\(props\.resource\.refreshedAt\)\)/)
  assert.match(source, /function formatRefreshedAt\(value\) \{[\s\S]*return formatListFreshnessDate\(value\)[\s\S]*\}/)
  assert.equal(source.includes("value.slice(0, 10)"), false)
  assert.equal(source.includes("'近期更新'"), false)
})

test('resource card displays selected publish tags without changing the type badge', () => {
  assert.match(source, /const resourceLabels = computed\(\(\) => normalizeResourceLabels\(props\.resource\.tags\)\.slice\(0, 3\)\)/)
  assert.match(source, /<view v-if="resourceLabels\.length" class="resource-labels">/)
  assert.match(source, /v-for="label in resourceLabels"/)
  assert.match(source, /class="resource-label"/)
  assert.match(source, /function normalizeResourceLabels\(tags = \[\]\)/)
  assert.match(source, /\.resource-labels \{[\s\S]*max-height: 52rpx;[\s\S]*overflow: hidden;/)
  assert.match(source, /\.resource-label \{[\s\S]*text-overflow: ellipsis;/)
})
