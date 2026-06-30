import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/messages/index.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

test('messages page removes top copy and keeps only effective status filters', () => {
  assert.doesNotMatch(source, /message-hero/)
  assert.doesNotMatch(source, /hero-title/)
  assert.doesNotMatch(source, /hero-desc/)
  assert.doesNotMatch(source, /消息和效果/)
  assert.doesNotMatch(source, /关注审核、过期、需求跟进和资源表现/)
  assert.doesNotMatch(source, /商家本周效果/)
  assert.doesNotMatch(source, /effect-card/)
  assert.doesNotMatch(source, /查看我的资源/)
  assert.match(source, /const messageTabs = \[[\s\S]*\{ label: '全部', status: '' \}[\s\S]*\{ label: '未读', status: 'unread' \}[\s\S]*\{ label: '已读', status: 'read' \}[\s\S]*\]/)
  assert.doesNotMatch(source, /\{ label: '审核'/)
  assert.doesNotMatch(source, /\{ label: '效果'/)
})

test('messages page follows my resources fixed compact filter style', () => {
  assert.match(source, /<view class="filter-row">[\s\S]*v-for="item in messageTabs"/)
  assert.doesNotMatch(source, /<scroll-view class="filter-row" scroll-x>/)
  assert.match(source, /\.messages-page \{[\s\S]*padding-top: 132rpx;/)
  assert.match(source, /\.messages-page \{[\s\S]*overflow-x: hidden;/)
  assert.match(source, /\.filter-row \{[\s\S]*position: fixed;[\s\S]*top: 0;[\s\S]*right: 0;[\s\S]*left: 0;[\s\S]*z-index: 10;[\s\S]*display: grid;[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);[\s\S]*padding: 24rpx 24rpx 16rpx;[\s\S]*overflow: hidden;[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 8rpx 20rpx rgba\(15, 23, 42, 0\.06\);/)
  assert.match(source, /\.filter-button \{[\s\S]*background: #f4f7fd;/)
  assert.match(source, /\.filter-button\.active \{[\s\S]*border-color: \$wplink-primary;[\s\S]*background: \$wplink-primary;[\s\S]*color: \$wplink-card;/)
})

test('messages page supports pull refresh and load more pagination', () => {
  for (const token of [
    'onPullDownRefresh',
    'onReachBottom',
    'uni.stopPullDownRefresh',
    'loadRows({ reset: true })',
    'loadRows({ reset: false })',
    'const page = ref(1)',
    'const pageSize = 20',
    'const total = ref(0)',
    'const hasMore = ref(true)',
    'const loading = ref(false)',
    "loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了'",
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /rows\.value = reset \? items : \[\.\.\.rows\.value, \.\.\.items\]/)
  assert.match(source, /hasMore\.value = rows\.value\.length < total\.value/)
  assert.match(source, /function selectStatus\(status\) \{[\s\S]*loadRows\(\{ reset: true \}\)[\s\S]*\}/)
})

test('messages page shows empty placeholder when current list has no rows', () => {
  assert.match(source, /v-if="!loading && !rows\.length" class="empty-placeholder"/)
  assert.match(source, /<text class="empty-title">\{\{ emptyTitle \}\}<\/text>/)
  assert.match(source, /const emptyTitle = computed\(\(\) => \{[\s\S]*if \(filters\.status === 'unread'\) return '暂无未读消息'[\s\S]*if \(filters\.status === 'read'\) return '暂无已读消息'[\s\S]*return '暂无消息'[\s\S]*\}\)/)
  assert.match(source, /\.empty-placeholder \{[\s\S]*min-height: 360rpx;[\s\S]*align-items: center;[\s\S]*justify-content: center;[\s\S]*background: \$wplink-card;/)
  assert.match(source, /\.empty-title \{[\s\S]*color: \$wplink-muted;[\s\S]*font-size: 28rpx;/)
})

test('messages page enables native pull down refresh in page config', () => {
  const page = pagesConfig.pages.find((entry) => entry.path === 'pages/messages/index')

  assert.equal(page?.style?.enablePullDownRefresh, true)
})

test('messages page builds resource detail target from resource message trigger id', () => {
  assert.match(source, /const resourceMessageTypes = new Set\(\[[\s\S]*'resource_review'[\s\S]*'resource_lifecycle'[\s\S]*'resource_expired'[\s\S]*'resource_expiring'[\s\S]*'effect_feedback'[\s\S]*\]\)/)
  assert.match(source, /function buildMessageTargetUrl\(item\) \{[\s\S]*if \(resourceMessageTypes\.has\(item\.messageType\) && item\.triggerId\)[\s\S]*return `\/pages\/resource\/detail\?id=\$\{encodeURIComponent\(item\.triggerId\)\}&merchantId=\$\{encodeURIComponent\(merchantId\)\}&from=my-resources`[\s\S]*return item\.targetUrl[\s\S]*\}/)
  assert.match(source, /const targetUrl = normalizeTargetUrl\(buildMessageTargetUrl\(item\)\)/)
})
