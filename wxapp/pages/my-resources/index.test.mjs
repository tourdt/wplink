import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')

test('my resources list uses only a compact cover image for item recognition', () => {
  assert.match(source, /<image class="resource-thumb" :src="item\.coverUrl \|\| DEFAULT_RESOURCE_COVER" mode="aspectFill" @error="handleResourceCoverError\(item\)" \/>/)
  assert.match(source, /\.resource-thumb \{[\s\S]*width: 112rpx;[\s\S]*height: 112rpx;/)
  assert.match(source, /\.resource-summary \{[\s\S]*display: grid;[\s\S]*grid-template-columns: 112rpx minmax\(0, 1fr\);/)
  assert.doesNotMatch(source, /width: 168rpx;[\s\S]*class="resource-thumb"/)
})

test('my resources list displays Chinese resource type text instead of raw type code', () => {
  assert.match(source, /import \{ resourceTypeText \} from '\.\.\/\.\.\/common\/enums'/)
  assert.match(source, /function displayResourceTypeText\(item\) \{[\s\S]*return resourceTypeText\[item\.typeCode\] \|\| item\.typeCode \|\| '资源'[\s\S]*\}/)
  assert.match(source, /\{\{ item\.category \}\} · \{\{ displayResourceTypeText\(item\) \}\}/)
  assert.doesNotMatch(source, /\{\{ item\.category \}\} · \{\{ item\.typeCode \}\}/)
})

test('my resources list falls back to a default resource image', () => {
  assert.match(source, /const DEFAULT_RESOURCE_COVER = '\/static\/resource\/default-resource-cover\.png'/)
  assert.match(source, /function handleResourceCoverError\(item\) \{[\s\S]*item\.coverUrl = ''[\s\S]*\}/)
  assert.equal(fs.existsSync(path.join(root, 'static/resource/default-resource-cover.png')), true)
  assert.doesNotMatch(source, /resource-thumb-placeholder/)
  assert.doesNotMatch(source, /\{\{ item\.category \|\| item\.typeCode \}\}/)
})

test('my resources page pins status filters and uses a compact publish action', () => {
  assert.doesNotMatch(source, /resource-manager-head/)
  assert.doesNotMatch(source, /manager-title/)
  assert.doesNotMatch(source, /manager-desc/)
  assert.match(source, /<button class="publish-fab" @click="openPublish">发布<\/button>/)
  assert.match(source, /\.my-resources-page \{[\s\S]*overflow-x: hidden;/)
  assert.match(source, /\.my-resources-page \{[\s\S]*padding-top: 132rpx;/)
  assert.match(source, /\.filter-row \{[\s\S]*position: fixed;[\s\S]*top: 0;[\s\S]*right: 0;[\s\S]*left: 0;[\s\S]*z-index: 10;[\s\S]*padding: 24rpx 24rpx 16rpx;[\s\S]*overflow: hidden;[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 8rpx 20rpx rgba\(15, 23, 42, 0\.06\);/)
  assert.match(source, /\.filter-button \{[\s\S]*background: #f4f7fd;/)
  assert.doesNotMatch(source, /position: sticky;/)
  assert.match(source, /\.publish-fab \{[\s\S]*position: fixed;[\s\S]*right: 24rpx;[\s\S]*bottom: calc\(32rpx \+ env\(safe-area-inset-bottom\)\);/)
})

test('my resources card actions avoid a visible toolbar frame', () => {
  assert.match(source, /<button v-if="isActivePublished\(item\)" class="primary-action" @click="refresh\(item\)">刷新<\/button>/)
  assert.match(source, /\.action-row \{[\s\S]*gap: 10rpx;[\s\S]*padding-top: 14rpx;[\s\S]*border-top: 1rpx solid #eef2f7;/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*border-radius:/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*background:/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*box-shadow:/)
  assert.match(source, /\.action-row button \{[\s\S]*min-width: 108rpx;[\s\S]*height: 62rpx;[\s\S]*border: 1rpx solid \$wplink-line;[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 2rpx 4rpx rgba\(15, 23, 42, 0\.04\);/)
  assert.match(source, /\.action-row button::after \{[\s\S]*border: 0;/)
  assert.match(source, /\.action-row \.primary-action \{[\s\S]*border-color: #b8c4d4;[\s\S]*background: \$wplink-primary-soft;[\s\S]*color: \$wplink-primary;/)
  assert.match(source, /\.action-row \.danger-button \{[\s\S]*border-color: #fecdd3;[\s\S]*background: #fff8f8;/)
  assert.doesNotMatch(source, /\.action-row button \{[\s\S]*background: #edf2f7;/)
  assert.doesNotMatch(source, /\.action-row \.primary-action \{[\s\S]*background: \$wplink-primary;/)
})

test('my resources hides the date row when no publish or expiry date exists', () => {
  assert.match(source, /<text v-if="shouldShowResourceDates\(item\)" class="resource-meta">发布 \{\{ displayDateOrPlaceholder\(item\.publishedAt\) \}\} · 到期 \{\{ displayDateOrPlaceholder\(item\.expiresAt\) \}\}<\/text>/)
  assert.match(source, /function shouldShowResourceDates\(item\) \{[\s\S]*return Boolean\(item\.publishedAt \|\| item\.expiresAt\)[\s\S]*\}/)
  assert.match(source, /function displayDateOrPlaceholder\(value\) \{[\s\S]*return value \? formatDateToDay\(value\) : '-'[\s\S]*\}/)
  assert.doesNotMatch(source, /<text class="resource-meta">发布 \{\{ formatDateToDay\(item\.publishedAt\) \}\} · 到期 \{\{ formatDateToDay\(item\.expiresAt\) \}\}<\/text>/)
})

test('my resources publish action opens the standalone publish editor', () => {
  assert.match(source, /function openPublish\(\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}` \}\)[\s\S]*\}/)
  assert.doesNotMatch(source, /function openPublish\(\) \{[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)[\s\S]*\}/)
})
