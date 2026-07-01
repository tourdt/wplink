import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/result.vue'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('search result page keeps the main tools and removes explanatory copy', () => {
  for (const token of [
    'class="search-bar"',
    'class="filter-row"',
    'class="hot-row"',
    'ResourceCard',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  for (const removedText of [
    '提交采购需求',
    'openDemand',
    '/pages/demand/index',
    'empty-visual-label',
    '找货',
    '输入关键词或先选热门条件',
    '刷新保存',
    '推广资源均需审核通过',
    '平台运营会继续留意库存',
    'search-guide',
    'promotion-note',
    'saveCurrentSearch',
    'savedSearches.length',
    'createSavedSearch',
    'listSavedSearches',
    'applySavedSearch',
    '保存搜索',
    '已保存搜索',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('search result page matches recommendation category browsing controls', () => {
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
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await search\(\)[\s\S]*\}/)
})

test('search result page keeps search and category controls sticky', () => {
  assert.match(source, /<view class="search-toolbar">[\s\S]*<view class="search-bar">[\s\S]*<view class="filter-shell">[\s\S]*<\/view>\s*<view v-if="hotKeywords\.length" class="hot-row">/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*sticky;/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*-webkit-sticky;/)
  assert.match(cssBlock('.search-toolbar'), /top:\s*0;/)
  assert.match(cssBlock('.search-toolbar'), /z-index:\s*20;/)
  assert.match(cssBlock('.search-toolbar'), /background:\s*\$wplink-bg;/)
  assert.match(cssBlock('.filter-shell'), /margin-bottom:\s*0;/)
})

test('search result hot keywords come from server config', () => {
  assert.match(source, /import \{ loadHotSearchKeywords \} from '..\/..\/common\/hotSearchKeywords'/)
  assert.match(source, /const hotKeywords = ref\(\[\]\)/)
  assert.match(source, /<view v-if="hotKeywords\.length" class="hot-row">/)
  assert.match(source, /async function loadHotKeywordOptions\(\) \{[\s\S]*hotKeywords\.value = await loadHotSearchKeywords\(filters\.cityCode\)[\s\S]*\}/)
  assert.doesNotMatch(source, /const hotKeywords = \[[\s\S]*夏款现货[\s\S]*\]/)
})

test('search result page runs default search when opened without search conditions', () => {
  assert.match(source, /const routeSearched = await applyRouteSearch\(options\)/)
  assert.match(source, /if \(!routeSearched && !hasPendingSearch\(\)\) \{[\s\S]*await search\(\)[\s\S]*\}/)
  assert.match(source, /async function applyRouteSearch\(options = \{\}\) \{[\s\S]*if \(!routeKeyword && !routeTypeCode\) return false[\s\S]*await search\(\)[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /function hasPendingSearch\(\) \{[\s\S]*return Boolean\(uni\.getStorageSync\(SEARCH_KEY\)\)[\s\S]*\}/)
})

test('search result page supports load more pagination', () => {
  for (const token of [
    'onReachBottom',
    'const page = ref(1)',
    'const pageSize = 20',
    'const total = ref(0)',
    'const hasMore = ref(true)',
    'const loading = ref(false)',
    "loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了'",
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /onReachBottom\(\(\) => \{[\s\S]*search\(\{ reset: false \}\)[\s\S]*\}\)/)
  assert.match(source, /async function search\(\{ reset = true \} = \{\}\) \{[\s\S]*if \(loading\.value\) return[\s\S]*if \(!reset && !hasMore\.value\) return[\s\S]*const nextPage = reset \? 1 : page\.value \+ 1[\s\S]*page: nextPage,[\s\S]*pageSize,[\s\S]*rows\.value = reset \? items : \[\.\.\.rows\.value, \.\.\.items\][\s\S]*page\.value = nextPage[\s\S]*total\.value = resp\.total \|\| rows\.value\.length[\s\S]*hasMore\.value = rows\.value\.length < total\.value[\s\S]*\}/)
})

test('search result page uses recommendation category font size', () => {
  assert.match(cssBlock('.filter-button'), /font-size:\s*26rpx;/)
})

test('search result page keeps search button visually aligned with input height', () => {
  assert.match(cssBlock('.search-bar'), /grid-template-columns:\s*1fr 116rpx;/)
  assert.match(cssBlock('.search-bar'), /align-items:\s*center;/)
  assert.match(cssBlock('.search-button'), /height:\s*76rpx;/)
  assert.match(cssBlock('.search-button'), /font-size:\s*26rpx;/)
  assert.match(cssBlock('.search-button'), /font-weight:\s*700;/)
})

test('search result empty state gives search-aware recovery actions', () => {
  for (const token of [
    'emptyTitle',
    'emptyDesc',
    'emptySuggestions',
    'empty-actions',
    'empty-suggestions',
    '换个条件',
    'resetSearchConditions',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const emptyTitle = '暂无匹配资源'/)
  assert.match(source, /const emptySuggestions = computed\(\(\) => hotKeywords\.value[\s\S]*slice\(0, 3\)\)/)
  assert.match(source, /async function resetSearchConditions\(\) \{[\s\S]*keyword\.value = ''[\s\S]*filters\.typeCode = ''[\s\S]*await scrollToSelectedType\(''\)[\s\S]*await search\(\)[\s\S]*\}/)
  assert.doesNotMatch(source, /async function resetSearchConditions\(\) \{[\s\S]*rows\.value = \[\][\s\S]*\}/)
  assert.doesNotMatch(source, /async function resetSearchConditions\(\) \{[\s\S]*searched\.value = false[\s\S]*\}/)
})

test('search result empty state keeps copy short and illustration text-free', () => {
  assert.match(source, /const emptyTitle = '暂无匹配资源'/)
  assert.match(source, /const emptyDesc = '换个关键词或分类试试。'/)
  assert.doesNotMatch(source, /暂未找到「/)
  assert.doesNotMatch(source, /平台资源会保持更新/)
})

test('search result empty state is visually subdued', () => {
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.doesNotMatch(cssBlock('.empty-card'), /box-shadow/)
  assert.match(cssBlock('.empty-visual'), /width:\s*168rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*112rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
