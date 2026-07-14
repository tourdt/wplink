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
    'class="filter-row group-row"',
    'class="hot-row"',
    'ResourceCard',
    'DemandCard',
    'groupResourceTypes',
    'groupFilterOptions',
    'selectGroup',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  for (const removedText of [
    'directionTabs',
    'activeDirection',
    'selectDirection',
    'title-direction-tabs',
    '提交采购需求',
    'openDemand',
    '/pages/demand/index',
    'empty-visual-label',
    '输入关键词或先选热门条件',
    '刷新保存',
    '推广供需信息均需审核通过',
    'search-guide',
    'promotion-note',
    'saveCurrentSearch',
    'savedSearches.length',
    '保存搜索',
    '已保存搜索',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('search page matches market category browsing controls', () => {
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

  assert.match(source, /const filters = reactive\(\{[\s\S]*cityCode: DEFAULT_CITY_CODE,[\s\S]*groupCode: '',[\s\S]*typeCode: '',[\s\S]*\}\)/)
  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /listCityResourceTypes\(filters\.cityCode\)/)
  assert.match(source, /categoryGroups\.value = groupResourceTypes\(resp\.items \|\| \[\]\)/)
  assert.match(source, /const selectedGroup = categoryGroups\.value\.find\(\(item\) => item\.code === filters\.groupCode\)/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*filters\.groupCode = groupCode[\s\S]*filters\.typeCode = ''[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*await search\(\)[\s\S]*\}/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await search\(\)[\s\S]*\}/)
})

test('search page moves only the title and back button into the custom title bar', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/search/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /<view class="search-nav" :style="searchNavStyle">/)
  assert.match(source, /<view class="search-title-bar" :style="searchTitleBarStyle">/)
  assert.match(source, /<button class="nav-back-button" @click="goBack">/)
  assert.match(source, /<text class="search-title">供需搜索<\/text>/)
  assert.match(cssBlock('.search-title-bar'), /justify-content:\s*center;/)
  assert.match(cssBlock('.search-title-bar'), /position:\s*relative;/)
  assert.match(cssBlock('.nav-back-button'), /position:\s*absolute;/)
  assert.match(cssBlock('.nav-back-button'), /background:\s*transparent;/)
  assert.match(cssBlock('.nav-back-button'), /width:\s*64rpx;/)
  assert.match(cssBlock('.nav-back-button::after'), /border:\s*0;/)
  assert.match(cssBlock('.nav-back-icon'), /border-left:\s*4rpx solid currentColor;/)
  assert.match(cssBlock('.nav-back-icon'), /transform:\s*rotate\(45deg\);/)
  assert.match(source, /const searchNavStyle = computed/)
  assert.match(source, /getMenuButtonBoundingClientRect/)
  assert.match(source, /function goBack\(\) \{[\s\S]*uni\.navigateBack\(\)[\s\S]*\}/)
})

test('search page keeps search and category controls sticky', () => {
  assert.match(source, /<view class="search-toolbar" :style="searchToolbarStyle">[\s\S]*<view class="search-bar">[\s\S]*<view class="filter-shell">/)
  assert.match(source, /const searchToolbarStyle = computed\(\(\) => `top: \$\{headerMetrics\.value\.headerHeight\}px;`\)/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*sticky;/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*-webkit-sticky;/)
  assert.match(cssBlock('.search-toolbar'), /z-index:\s*20;/)
  assert.match(cssBlock('.filter-shell'), /margin-bottom:\s*0;/)
})

test('search hot keywords come from server config', () => {
  assert.match(source, /import \{ loadHotSearchKeywords \} from '..\/..\/common\/hotSearchKeywords'/)
  assert.match(source, /const hotKeywords = ref\(\[\]\)/)
  assert.match(source, /<view v-if="hotKeywords\.length" class="hot-row">/)
  assert.match(source, /async function loadHotKeywordOptions\(\) \{[\s\S]*hotKeywords\.value = await loadHotSearchKeywords\(filters\.cityCode\)[\s\S]*\}/)
  assert.doesNotMatch(source, /const hotKeywords = \[[\s\S]*夏款现货[\s\S]*\]/)
})

test('search page applies route and pending category filters without direction state', () => {
  assert.match(source, /const routeGroupCode = decodeSearchValue\(options\.groupCode \|\| ''\)/)
  assert.match(source, /const routeTypeCode = decodeSearchValue\(options\.typeCode \|\| ''\)/)
  assert.match(source, /if \(!routeKeyword && !routeGroupCode && !routeTypeCode && routeCityCode === DEFAULT_CITY_CODE\) return false/)
  assert.match(source, /filters\.groupCode = routeGroupCode/)
  assert.match(source, /filters\.typeCode = routeTypeCode/)
  assert.match(source, /filters\.groupCode = pendingSearch\.groupCode \|\| ''/)
  assert.match(source, /filters\.typeCode = pendingSearch\.typeCode \|\| ''/)
  assert.match(source, /function hasPendingSearch\(\) \{[\s\S]*return Boolean\(uni\.getStorageSync\(SEARCH_KEY\)\)[\s\S]*\}/)
  assert.doesNotMatch(source, /routeDirection/)
  assert.doesNotMatch(source, /directionStateCache/)
})

test('search page submits group and type filters to search API', () => {
  assert.match(source, /searchResources\(\{[\s\S]*\.\.\.filters,[\s\S]*keyword: keyword\.value\.trim\(\),[\s\S]*page: nextPage,[\s\S]*pageSize,[\s\S]*\}\)/)
  assert.match(source, /rows\.value = reset \? items : \[\.\.\.rows\.value, \.\.\.items\]/)
  assert.match(source, /searched\.value = true/)
  assert.doesNotMatch(source, /direction: activeDirection\.value/)
})

test('search page renders mixed supply and demand result cards from item direction', () => {
  for (const token of [
    '<template v-for="item in rows" :key="item.id">',
    'v-if="item.direction === RESOURCE_DIRECTION_DEMAND"',
    '<ResourceCard v-else',
    'RESOURCE_DIRECTION_DEMAND',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
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
  assert.match(source, /async function search\(\{ reset = true, force = false \} = \{\}\) \{[\s\S]*if \(loading\.value && !force\) return[\s\S]*if \(!reset && !hasMore\.value\) return[\s\S]*const nextPage = reset \? 1 : page\.value \+ 1[\s\S]*page\.value = nextPage[\s\S]*total\.value = resp\.total \|\| rows\.value\.length[\s\S]*hasMore\.value = rows\.value\.length < total\.value[\s\S]*\}/)
})

test('search page tracks scroll progress without per-direction cache', () => {
  assert.match(source, /const searchRequestSeq = ref\(0\)/)
  assert.match(source, /import \{ onLoad, onPageScroll, onReachBottom, onShow \} from '@dcloudio\/uni-app'/)
  assert.match(source, /<scroll-view[\s\S]*:scroll-left="typeScrollLeft"[\s\S]*@scroll="handleTypeScroll"/)
  assert.match(source, /const typeScrollLeft = ref\(0\)/)
  assert.match(source, /const pageScrollTop = ref\(0\)/)
  assert.match(source, /onPageScroll\(handlePageScroll\)/)
  assert.match(source, /function handlePageScroll\(event = \{\}\) \{[\s\S]*pageScrollTop\.value = scrollTop[\s\S]*\}/)
  assert.match(source, /function handleTypeScroll\(event = \{\}\) \{[\s\S]*typeScrollLeft\.value = scrollLeft[\s\S]*\}/)
  assert.match(source, /async function restorePageScroll\(scrollTop = pageScrollTop\.value\) \{[\s\S]*uni\.pageScrollTo\(\{ scrollTop, duration: 0 \}\)[\s\S]*\}/)
})

test('search page uses concise placeholder and empty state copy', () => {
  assert.match(source, /const searchPlaceholder = '搜供应、需求、场地或服务'/)
  assert.match(source, /const emptyTitle = '暂无匹配内容'/)
  assert.match(source, /const emptyDesc = '换个关键词或分类试试。'/)
  assert.match(source, /async function resetSearchConditions\(\) \{[\s\S]*keyword\.value = ''[\s\S]*filters\.groupCode = ''[\s\S]*filters\.typeCode = ''[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*await search\(\)[\s\S]*\}/)
  assert.doesNotMatch(source, /暂无匹配供应/)
  assert.doesNotMatch(source, /暂无匹配需求/)
  assert.doesNotMatch(source, /搜索找现货、找库存、找工厂、找服务/)
  assert.doesNotMatch(source, /搜索库存清仓、现货货源、工厂接单、配套服务/)
})

test('search page keeps compact control sizing and subdued empty state', () => {
  assert.match(cssBlock('.filter-button'), /font-size:\s*26rpx;/)
  assert.match(cssBlock('.search-bar'), /grid-template-columns:\s*1fr 116rpx;/)
  assert.match(cssBlock('.search-button'), /height:\s*76rpx;/)
  assert.match(cssBlock('.search-button'), /font-weight:\s*700;/)
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.doesNotMatch(cssBlock('.empty-card'), /box-shadow/)
  assert.match(cssBlock('.empty-visual'), /width:\s*168rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
