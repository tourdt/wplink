import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/topic/index.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('topic empty state is centered in the list display area', () => {
  assert.match(cssBlock('.topic-page'), /display:\s*flex;/)
  assert.match(cssBlock('.topic-page'), /flex-direction:\s*column;/)
  assert.match(source, /<view v-else class="empty-card">/)
  assert.match(cssBlock('.empty-card'), /flex:\s*1;/)
  assert.match(cssBlock('.empty-card'), /align-content:\s*center;/)
  assert.match(cssBlock('.empty-card'), /justify-items:\s*center;/)
  assert.match(cssBlock('.empty-card'), /min-height:\s*420rpx;/)
  assert.match(cssBlock('.empty-card'), /padding-bottom:\s*80rpx;/)
  assert.match(source, /<text class="empty-desc">专题供应更新中，可先浏览供需页。<\/text>/)
  assert.doesNotMatch(source, /可以先去供需页按类型继续浏览，平台会持续更新专题供应。/)
})
