import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'components/ResourceList.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('resource list empty state is centered in the list display area', () => {
  assert.match(source, /<view v-if="resources\.length === 0 && !loading" class="empty-text">\{\{ emptyText \}\}<\/view>/)
  assert.match(cssBlock('.empty-text'), /display:\s*grid;/)
  assert.match(cssBlock('.empty-text'), /place-items:\s*center;/)
  assert.match(cssBlock('.empty-text'), /min-height:\s*360rpx;/)
  assert.match(cssBlock('.empty-text'), /padding-bottom:\s*80rpx;/)
  assert.match(cssBlock('.empty-text'), /text-align:\s*center;/)
})
