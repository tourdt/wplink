import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'components/ResourceCard.vue'), 'utf8')

test('resource card displays type code as Chinese resource type text', () => {
  assert.match(source, /import \{ resourceTypeText \} from '\.\.\/common\/enums'/)
  assert.match(source, /const resourceTypeLabel = computed/)
  assert.match(source, /resourceTypeText\[props\.resource\.typeCode\]/)
  assert.match(source, /<text v-if="resourceTypeLabel" class="type-corner">\{\{ resourceTypeLabel \}\}<\/text>/)
  assert.equal(source.includes('{{ resource.typeCode }}'), false)
})

test('resource card uses the default cover when resource image is missing', () => {
  assert.match(source, /const DEFAULT_RESOURCE_COVER = '\/static\/resource\/default-resource-cover\.png'/)
  assert.match(source, /<image class="resource-thumb" :src="coverUrl \|\| DEFAULT_RESOURCE_COVER" mode="aspectFill" \/>/)
  assert.match(source, /<text v-if="resourceTypeLabel" class="type-corner">\{\{ resourceTypeLabel \}\}<\/text>/)
  assert.equal(source.includes('placeholder-thumb'), false)
  assert.equal(source.includes('placeholderLabel'), false)
  assert.equal(source.includes('资源图片'), false)
  assert.equal(source.includes('props.resource.typeCode || props.resource.category'), false)
})

test('resource card image slot keeps a fixed square size instead of stretching with content', () => {
  assert.match(source, /\.thumb-wrap \{[\s\S]*align-self: flex-start;[\s\S]*width: 168rpx;[\s\S]*height: 168rpx;[\s\S]*min-height: 0;/)
  assert.match(source, /\.resource-card-home \.thumb-wrap \{[\s\S]*width: 160rpx;[\s\S]*height: 160rpx;[\s\S]*min-height: 0;/)
  assert.match(source, /\.resource-card-compact \.thumb-wrap \{[\s\S]*width: 144rpx;[\s\S]*height: 144rpx;[\s\S]*min-height: 0;/)
  assert.equal(source.includes('min-height: 168rpx'), false)
  assert.equal(source.includes('min-height: 160rpx'), false)
  assert.equal(source.includes('min-height: 144rpx'), false)
})

test('resource type corner label stays visually secondary on the image', () => {
  assert.match(source, /\.type-corner \{[\s\S]*padding: 3rpx 8rpx;[\s\S]*border-radius: 7rpx;[\s\S]*font-size: 20rpx;/)
  assert.match(source, /\.resource-card-home \.type-corner \{[\s\S]*font-size: 20rpx;/)
  assert.match(source, /\.resource-card-compact \.type-corner \{[\s\S]*font-size: 18rpx;/)
})

test('resource card uses a readable four-line content layout', () => {
  assert.match(source, /<text class="resource-title">\{\{ resource\.title \|\| '资源标题待完善' \}\}<\/text>[\s\S]*<text class="resource-meta">\{\{ resource\.category \|\| '品类待沟通' \}\} · \{\{ resource\.quantityText \|\| '数量待沟通' \}\}<\/text>[\s\S]*<text class="resource-price">\{\{ resource\.priceText \|\| '价格面议' \}\}<\/text>[\s\S]*<view class="merchant-line">/)
  assert.match(source, /<view class="merchant-line">[\s\S]*<text v-if="isVerifiedMerchant" class="verified-badge">已认证<\/text>[\s\S]*<text class="merchant-name">\{\{ merchantName \}\}<\/text>[\s\S]*<text class="refresh-time">\{\{ formatRefreshedAt\(resource\.refreshedAt\) \}\}<\/text>[\s\S]*<\/view>/)
  assert.equal(source.includes('meta-price-line'), false)
  assert.equal(source.includes('平台核实'), false)
  assert.equal(source.includes('hasCreditTags'), false)
  assert.equal(source.includes('查看详情'), false)
  assert.equal(source.includes('decision-tip'), false)
  assert.equal(source.includes('card-foot'), false)
  assert.equal(source.includes('tag-row'), false)
  assert.equal(source.includes('merchant-row'), false)
})

test('resource card uses short freshness date in resource list', () => {
  assert.match(source, /import \{ formatListFreshnessDate \} from '\.\.\/common\/date'/)
  assert.match(source, /function formatRefreshedAt\(value\) \{[\s\S]*return formatListFreshnessDate\(value\)[\s\S]*\}/)
  assert.equal(source.includes("value.slice(0, 10)"), false)
  assert.equal(source.includes("'近期更新'"), false)
})
