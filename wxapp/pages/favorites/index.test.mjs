import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/favorites/index.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('favorites empty state is centered in the list display area', () => {
  assert.match(cssBlock('.favorites-page'), /display:\s*flex;/)
  assert.match(cssBlock('.favorites-page'), /flex-direction:\s*column;/)
  assert.match(source, /<view v-if="!loading && currentRows\.length === 0" class="empty-state">/)
  assert.match(cssBlock('.empty-state'), /flex:\s*1;/)
  assert.match(cssBlock('.empty-state'), /align-content:\s*center;/)
  assert.match(cssBlock('.empty-state'), /min-height:\s*420rpx;/)
  assert.match(cssBlock('.empty-state'), /padding-bottom:\s*88rpx;/)
  assert.match(source, /if \(activeTab\.value === 'resources'\) return '收藏后可在这里快速回看。'/)
  assert.match(source, /return '关注后可快速进入商家主页。'/)
  assert.doesNotMatch(source, /看到合适的工厂直批、库存出售或加工生产信息后点收藏/)
  assert.doesNotMatch(source, /关注常合作或感兴趣的发布者/)
})

test('favorites resource list uses the shared feed card', () => {
  assert.match(source, /import ResourceFeedCard from '\.\.\/\.\.\/components\/ResourceFeedCard\.vue'/)
  assert.match(source, /<ResourceFeedCard v-for="item in favoriteResources" :key="item\.id" :resource="item" @open="openResource" \/>/)
  assert.doesNotMatch(source, /import ResourceCard/)
})
