import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/index.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

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

test('resource recommendation page shows all categories and scrolls selected category into view', () => {
  for (const token of [
    'visibleResourceTypes',
    'scrollIntoTypeId',
    'scrollToSelectedType',
    'getTypeButtonId',
    'scroll-into-view',
    'scroll-with-animation',
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

  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.doesNotMatch(source, /MAX_VISIBLE_RESOURCE_TYPES/)
  assert.doesNotMatch(source, /resourceTypes\.value\.slice/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
})

test('resource recommendation page does not expose demand submission in MVP', () => {
  for (const removedText of ['提交采购需求', 'openDemand', '/pages/demand/index']) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('resource recommendation empty state uses concise copy without image text', () => {
  assert.match(source, /<text class="empty-title">暂无推荐资源<\/text>/)
  assert.match(source, /<text class="empty-desc">换个类型或搜索关键词。<\/text>/)
  assert.doesNotMatch(source, /<text>资源<\/text>/)
  assert.doesNotMatch(source, /当前类型暂无推荐资源/)
  assert.doesNotMatch(source, /可以换个类型继续浏览/)
})

test('resource recommendation empty state is visually subdued', () => {
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.match(cssBlock('.empty-visual'), /width:\s*148rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*104rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
