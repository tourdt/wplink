import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/messages/index.vue'), 'utf8')

test('messages page removes top copy and keeps only effective status filters', () => {
  assert.doesNotMatch(source, /message-hero/)
  assert.doesNotMatch(source, /hero-title/)
  assert.doesNotMatch(source, /hero-desc/)
  assert.doesNotMatch(source, /消息和效果/)
  assert.doesNotMatch(source, /关注审核、过期、需求跟进和资源表现/)
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
