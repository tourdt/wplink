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

test('my resources publish action opens the standalone publish editor', () => {
  assert.match(source, /function openPublish\(\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}` \}\)[\s\S]*\}/)
  assert.doesNotMatch(source, /function openPublish\(\) \{[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)[\s\S]*\}/)
})
