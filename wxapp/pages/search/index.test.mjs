import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/search/index.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('search page keeps the main tools and removes explanatory copy', () => {
  for (const token of [
    'class="search-bar"',
    'directionTabs',
    'activeDirection',
    'selectDirection',
    '资源',
    '需求',
    'class="filter-row"',
    'class="hot-row"',
    'ResourceCard',
    'DemandCard',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  for (const removedText of [
    '提交采购需求',
    'openDemand',
    '/pages/demand/index',
    'empty-visual-label',
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

test('search page matches recommendation category browsing controls', () => {
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
  assert.match(source, /listCityResourceTypes\(filters\.cityCode,\s*\{ direction: activeDirection\.value \}\)/)
  assert.match(source, /direction: activeDirection\.value/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await search\(\)[\s\S]*\}/)
})

test('search page moves direction switch into custom title bar', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/search/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /<view class="search-nav" :style="searchNavStyle">/)
  assert.match(source, /<view class="search-title-bar" :style="searchTitleBarStyle">/)
  assert.match(source, /<button class="nav-back-button" @click="goBack">/)
  assert.match(source, /<button class="nav-back-button" @click="goBack">\s*<view class="nav-back-icon"><\/view>\s*<\/button>/)
  assert.doesNotMatch(source, /nav-back-text/)
  assert.doesNotMatch(source, /<text class="nav-back-icon">/)
  assert.match(source, /<view class="title-direction-tabs">[\s\S]*v-for="item in directionTabs"[\s\S]*@click="selectDirection\(item\.value\)"/)
  assert.match(cssBlock('.search-title-bar'), /justify-content:\s*center;/)
  assert.match(cssBlock('.search-title-bar'), /position:\s*relative;/)
  assert.match(cssBlock('.nav-back-button'), /position:\s*absolute;/)
  assert.match(cssBlock('.nav-back-button'), /background:\s*transparent;/)
  assert.match(cssBlock('.nav-back-button'), /width:\s*64rpx;/)
  assert.match(cssBlock('.nav-back-button'), /min-width:\s*64rpx;/)
  assert.match(cssBlock('.nav-back-button'), /color:\s*#111827;/)
  assert.match(cssBlock('.nav-back-button::after'), /border:\s*0;/)
  assert.match(cssBlock('.nav-back-icon'), /border-left:\s*4rpx solid currentColor;/)
  assert.match(cssBlock('.nav-back-icon'), /border-bottom:\s*4rpx solid currentColor;/)
  assert.match(cssBlock('.nav-back-icon'), /transform:\s*rotate\(45deg\);/)
  assert.doesNotMatch(cssBlock('.search-title-bar'), /grid-template-columns:\s*72rpx minmax\(0, 320rpx\) 190rpx;/)
  assert.doesNotMatch(cssBlock('.nav-back-button'), /border-radius:\s*999rpx;/)
  assert.doesNotMatch(cssBlock('.nav-back-button'), /box-shadow:\s*inset 0 0 0 1rpx \$wplink-line;/)
  assert.doesNotMatch(cssBlock('.nav-back-button'), /gap:/)
  assert.doesNotMatch(cssBlock('.nav-back-button'), /color:\s*\$wplink-primary;/)
  assert.match(source, /const searchNavStyle = computed/)
  assert.match(source, /getMenuButtonBoundingClientRect/)
  assert.match(source, /function goBack\(\) \{[\s\S]*uni\.navigateBack\(\)[\s\S]*\}/)
  assert.doesNotMatch(source, /<view class="search-toolbar">[\s\S]*<view class="direction-tabs">[\s\S]*<\/view>[\s\S]*<view class="filter-shell">/)
})

test('search page keeps search and category controls sticky', () => {
  assert.match(source, /<view class="search-toolbar" :style="searchToolbarStyle">[\s\S]*<view class="search-bar">[\s\S]*<view class="filter-shell">[\s\S]*<\/view>\s*<view v-if="hotKeywords\.length" class="hot-row">/)
  assert.match(source, /const searchToolbarStyle = computed\(\(\) => `top: \$\{headerMetrics\.value\.headerHeight\}px;`\)/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*sticky;/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*-webkit-sticky;/)
  assert.match(cssBlock('.search-toolbar'), /z-index:\s*20;/)
  assert.match(cssBlock('.search-toolbar'), /background:\s*\$wplink-bg;/)
  assert.match(cssBlock('.filter-shell'), /margin-bottom:\s*0;/)
})

test('search hot keywords come from server config', () => {
  assert.match(source, /import \{ loadHotSearchKeywords \} from '..\/..\/common\/hotSearchKeywords'/)
  assert.match(source, /const hotKeywords = ref\(\[\]\)/)
  assert.match(source, /<view v-if="hotKeywords\.length" class="hot-row">/)
  assert.match(source, /async function loadHotKeywordOptions\(\) \{[\s\S]*hotKeywords\.value = await loadHotSearchKeywords\(filters\.cityCode\)[\s\S]*\}/)
  assert.doesNotMatch(source, /const hotKeywords = \[[\s\S]*夏款现货[\s\S]*\]/)
})

test('search page runs default search when opened without search conditions', () => {
  assert.match(source, /const routeSearched = await applyRouteSearch\(options\)/)
  assert.match(source, /if \(!routeSearched && !hasPendingSearch\(\)\) \{[\s\S]*await search\(\)[\s\S]*\}/)
  assert.match(source, /async function applyRouteSearch\(options = \{\}\) \{[\s\S]*routeDirection[\s\S]*if \(!routeKeyword && !routeTypeCode && !routeDirection\) return false[\s\S]*await search\(\)[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /function hasPendingSearch\(\) \{[\s\S]*return Boolean\(uni\.getStorageSync\(SEARCH_KEY\)\)[\s\S]*\}/)
})

test('search page supports load more pagination', () => {
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
  assert.match(source, /async function search\(\{ reset = true, force = false \} = \{\}\) \{[\s\S]*if \(loading\.value && !force\) return[\s\S]*if \(!reset && !hasMore\.value\) return[\s\S]*const nextPage = reset \? 1 : page\.value \+ 1[\s\S]*page: nextPage,[\s\S]*pageSize,[\s\S]*rows\.value = reset \? items : \[\.\.\.rows\.value, \.\.\.items\][\s\S]*page\.value = nextPage[\s\S]*total\.value = resp\.total \|\| rows\.value\.length[\s\S]*hasMore\.value = rows\.value\.length < total\.value[\s\S]*\}/)
})

test('search page preserves filters and scroll progress when switching direction', () => {
  assert.match(source, /const searchRequestSeq = ref\(0\)/)
  assert.match(source, /import \{ onLoad, onPageScroll, onReachBottom, onShow \} from '@dcloudio\/uni-app'/)
  assert.match(source, /<scroll-view[\s\S]*:scroll-left="typeScrollLeft"[\s\S]*@scroll="handleTypeScroll"/)
  assert.match(source, /const directionStateCache = reactive\(\{[\s\S]*\[RESOURCE_DIRECTION_SUPPLY\]: createDirectionResultState\(\)[\s\S]*\[RESOURCE_DIRECTION_DEMAND\]: createDirectionResultState\(\)[\s\S]*\}\)/)
  assert.match(source, /function createDirectionResultState\(\) \{[\s\S]*keyword: ''[\s\S]*typeCode: ''[\s\S]*typeScrollLeft: 0[\s\S]*pageScrollTop: 0[\s\S]*loaded: false[\s\S]*\}/)
  assert.match(source, /function saveCurrentDirectionState\(direction = activeDirection\.value\) \{[\s\S]*cachedState\.keyword = keyword\.value[\s\S]*cachedState\.typeCode = filters\.typeCode[\s\S]*cachedState\.typeScrollLeft = typeScrollLeft\.value[\s\S]*cachedState\.pageScrollTop = pageScrollTop\.value[\s\S]*\}/)
  assert.match(source, /function applyDirectionState\(direction\) \{[\s\S]*keyword\.value = cachedState\.keyword[\s\S]*filters\.typeCode = cachedState\.typeCode[\s\S]*typeScrollLeft\.value = cachedState\.typeScrollLeft[\s\S]*pageScrollTop\.value = cachedState\.pageScrollTop[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /onPageScroll\(handlePageScroll\)/)
  assert.match(source, /function handlePageScroll\(event = \{\}\) \{[\s\S]*pageScrollTop\.value = scrollTop[\s\S]*directionStateCache\[activeDirection\.value\]\.pageScrollTop = scrollTop[\s\S]*\}/)
  assert.match(source, /function handleTypeScroll\(event = \{\}\) \{[\s\S]*typeScrollLeft\.value = scrollLeft[\s\S]*directionStateCache\[activeDirection\.value\]\.typeScrollLeft = scrollLeft[\s\S]*\}/)
  assert.match(source, /async function restorePageScroll\(scrollTop = pageScrollTop\.value\) \{[\s\S]*uni\.pageScrollTo\(\{ scrollTop, duration: 0 \}\)[\s\S]*\}/)
  assert.match(source, /async function selectDirection\(direction\) \{[\s\S]*const inheritedPageScrollTop = pageScrollTop\.value[\s\S]*saveCurrentDirectionState\(activeDirection\.value\)[\s\S]*cancelPendingSearchRequests\(\)[\s\S]*activeDirection\.value = direction[\s\S]*if \(applyDirectionState\(direction\)\) \{[\s\S]*await restorePageScroll\(\)[\s\S]*return[\s\S]*\}[\s\S]*prepareFreshDirectionSearchState\(\)[\s\S]*await loadResourceTypes\(\)[\s\S]*await search\(\{ force: true \}\)[\s\S]*await restorePageScroll\(inheritedPageScrollTop\)[\s\S]*\}/)
  assert.match(source, /async function search\(\{ reset = true, force = false \} = \{\}\) \{[\s\S]*if \(loading\.value && !force\) return[\s\S]*const requestSeq = searchRequestSeq\.value \+ 1[\s\S]*searchRequestSeq\.value = requestSeq[\s\S]*if \(requestSeq !== searchRequestSeq\.value\) return[\s\S]*if \(requestSeq === searchRequestSeq\.value\) \{[\s\S]*loading\.value = false[\s\S]*\}/)
  assert.doesNotMatch(source, /function resetResultStateForDirectionSwitch/)
  assert.doesNotMatch(source, /async function selectDirection\(direction\) \{[\s\S]*filters\.typeCode = ''[\s\S]*\}/)
})

test('search page uses recommendation category font size', () => {
  assert.match(cssBlock('.filter-button'), /font-size:\s*26rpx;/)
})

test('search page keeps search button visually aligned with input height', () => {
  assert.match(cssBlock('.search-bar'), /grid-template-columns:\s*1fr 116rpx;/)
  assert.match(cssBlock('.search-bar'), /align-items:\s*center;/)
  assert.match(cssBlock('.search-button'), /height:\s*76rpx;/)
  assert.match(cssBlock('.search-button'), /font-size:\s*26rpx;/)
  assert.match(cssBlock('.search-button'), /font-weight:\s*700;/)
})

test('search page uses short search placeholder copy', () => {
  assert.match(source, /const searchPlaceholder = computed\(\(\) => \([\s\S]*\? '搜采购\/找厂\/服务'[\s\S]*: '搜现货\/库存\/工厂'[\s\S]*\)\)/)
  assert.doesNotMatch(source, /搜索找现货、找库存、找工厂、找服务/)
  assert.doesNotMatch(source, /搜索库存清仓、现货货源、工厂接单、配套服务/)
})

test('search empty state gives search-aware recovery actions', () => {
  const resetSearchConditionsBody = source.match(/async function resetSearchConditions\(\) \{([\s\S]*?)\n\}/)?.[1] || ''

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
  assert.doesNotMatch(resetSearchConditionsBody, /rows\.value = \[\]/)
  assert.doesNotMatch(resetSearchConditionsBody, /searched\.value = false/)
})

test('search empty state keeps copy short and illustration text-free', () => {
  assert.match(source, /const emptyTitle = '暂无匹配资源'/)
  assert.match(source, /const emptyDesc = '换个关键词或分类试试。'/)
  assert.doesNotMatch(source, /暂未找到「/)
  assert.doesNotMatch(source, /平台资源会保持更新/)
})

test('search empty state is visually subdued', () => {
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.doesNotMatch(cssBlock('.empty-card'), /box-shadow/)
  assert.match(cssBlock('.empty-visual'), /width:\s*168rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*112rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
