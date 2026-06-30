import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/index.vue'), 'utf8')

test('resource recommendation page supports pull refresh and load more pagination', () => {
  for (const token of [
    'onPullDownRefresh',
    'onReachBottom',
    'uni.stopPullDownRefresh',
    "loadRecommendedResources({ reset: true })",
    "loadRecommendedResources({ reset: false })",
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
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
})

test('resource recommendation page keeps common categories compact and opens all categories drawer', () => {
  for (const token of [
    'visibleResourceTypes',
    'MAX_VISIBLE_RESOURCE_TYPES',
    'showTypeDrawer',
    'openTypeDrawer',
    'closeTypeDrawer',
    'type-drawer-mask',
    'type-drawer-panel',
    '全部分类',
    '常用分类',
    'drawer-type-grid',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /v-for="item in visibleResourceTypes"/)
  assert.match(source, /v-if="resourceTypes\.length > visibleResourceTypes\.length"/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
})
