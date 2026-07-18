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

function functionBlock(name) {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`(?:async\\s+)?function ${escapedName}\\([^)]*\\) \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('search page keeps the main tools and removes explanatory copy', () => {
  for (const token of [
    'class="search-bar"',
    'class="channel-title-button"',
    'class="hot-row"',
    'ResourceCard',
    'DemandCard',
    'groupResourceTypes',
    'groupFilterOptions',
    'channelTitle',
    'selectedGroupName',
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
    'showGroupDrawer',
    'openGroupDrawer',
    'closeGroupDrawer',
    'group-drawer-mask',
    'drawer-group-list',
    'visibleResourceTypes',
    'scrollIntoTypeId',
    'scrollToSelectedType',
    'getTypeButtonId',
    'scroll-into-view',
    'scroll-with-animation',
    'showTypeDrawer',
    'openTypeDrawer',
    'closeTypeDrawer',
    'showAllTypeButton',
    'type-drawer-mask',
    'type-drawer-panel',
    '全部分类',
    'drawer-type-grid',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const filters = reactive\(\{[\s\S]*cityCode: DEFAULT_CITY_CODE,[\s\S]*groupCode: '',[\s\S]*typeCode: '',[\s\S]*\}\)/)
  assert.match(source, /const selectedGroupName = computed\(\(\) =>/)
  assert.match(source, /const channelTitle = computed\(\(\) => filters\.groupCode \? selectedGroupName\.value : '供需搜索'\)/)
  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /const showAllTypeButton = computed\(\(\) => resourceTypes\.value\.length - 1 > 3\)/)
  assert.match(source, /<view :class="\['filter-shell', showAllTypeButton \? 'has-all-type-button' : ''\]">/)
  assert.match(source, /<button[\s\S]*v-if="showAllTypeButton"[\s\S]*class="all-type-button"[\s\S]*全部分类/)
  assert.match(source, /listCityResourceTypes\(filters\.cityCode\)/)
  assert.match(source, /categoryGroups\.value = groupResourceTypes\(resp\.items \|\| \[\]\)/)
  assert.match(source, /const selectedGroup = categoryGroups\.value\.find\(\(item\) => item\.code === filters\.groupCode\)/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*showGroupDrawer\.value = false[\s\S]*filters\.groupCode = groupCode[\s\S]*filters\.typeCode = ''[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*await search\(\)[\s\S]*\}/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await search\(\)[\s\S]*\}/)
  assert.doesNotMatch(source, /class="filter-row group-row"/)
  assert.doesNotMatch(source, /class="group-select-button"/)
  assert.doesNotMatch(source, />常用分类<\/text>/)
})

test('search page supports configured resource tag filters', () => {
  for (const token of [
    'tags: []',
    'currentGroupResourceTypeItems',
    'selectedResourceType',
    'searchTagOptions',
    'fieldSchema?.tagOptions',
    'normalizeSearchTagOptions',
    'normalizeSearchTags',
    'parseSearchTags',
    'syncSelectedTagsWithOptions',
    'toggleSearchTag',
    'clearSearchTags',
    'isSearchTagSelected',
    'MAX_SEARCH_TAGS = 8',
    'class="tag-filter-row"',
    '@click="toggleSearchTag(tag)"',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /<scroll-view[\s\S]*v-if="searchTagOptions\.length"[\s\S]*class="tag-filter-row"/)
  assert.match(source, /:class="\['tag-filter-button', isSearchTagSelected\(tag\) \? 'active' : ''\]"/)
  assert.doesNotMatch(source, /@click="clearSearchTags"/)
  assert.doesNotMatch(source, />\s*不限标签\s*</)
  assert.match(source, /if \(!selectedResourceType\.value\) return \[\]/)
  assert.match(source, /normalizeSearchTagOptions\(selectedResourceType\.value\.fieldSchema\?\.tagOptions \|\| \[\]\)/)
  assert.doesNotMatch(source, /sourceItems\.flatMap/)
  assert.match(source, /filters\.tags = normalizeSearchTags\(filters\.tags, searchTagOptions\.value\)/)
  assert.match(source, /uni\.showToast\(\{ title: `最多选择\$\{MAX_SEARCH_TAGS\}个标签`, icon: 'none' \}\)/)
  assert.match(cssBlock('.tag-filter-row'), /overflow-x:\s*auto;/)
  assert.match(cssBlock('.tag-filter-button'), /height:\s*60rpx;/)
  assert.match(cssBlock('.tag-filter-button.active'), /background:\s*\$wplink-primary-soft;/)
})

test('search page uses the custom title bar as the primary category channel switcher', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/search/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /<view class="search-nav" :style="searchNavStyle">/)
  assert.match(source, /<view class="search-title-bar" :style="searchTitleBarStyle">/)
  assert.match(source, /<button class="nav-back-button" @click="goBack">/)
  assert.match(source, /<button class="channel-title-button" @click="openGroupDrawer">/)
  assert.match(source, /<text class="channel-title-text">\{\{ channelTitle \}\}<\/text>/)
  assert.match(source, /<text class="channel-title-arrow"><\/text>/)
  assert.match(cssBlock('.search-title-bar'), /justify-content:\s*center;/)
  assert.match(cssBlock('.search-title-bar'), /position:\s*relative;/)
  assert.match(cssBlock('.channel-title-button'), /background:\s*transparent;/)
  assert.match(cssBlock('.channel-title-button::after'), /border:\s*0;/)
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
  assert.match(source, /<view class="search-toolbar" :style="searchToolbarStyle">[\s\S]*<view class="search-bar">[\s\S]*<view :class="\['filter-shell', showAllTypeButton \? 'has-all-type-button' : ''\]">/)
  assert.match(source, /const searchToolbarStyle = computed\(\(\) => `top: \$\{headerMetrics\.value\.headerHeight\}px;`\)/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*sticky;/)
  assert.match(cssBlock('.search-toolbar'), /position:\s*-webkit-sticky;/)
  assert.match(cssBlock('.search-toolbar'), /z-index:\s*20;/)
  assert.match(cssBlock('.filter-shell'), /margin-bottom:\s*0;/)
  assert.match(cssBlock('.filter-shell'), /grid-template-columns:\s*minmax\(0,\s*1fr\);/)
  assert.match(cssBlock('.filter-shell.has-all-type-button'), /grid-template-columns:\s*minmax\(0,\s*1fr\) 156rpx;/)
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
  assert.match(source, /const routeTags = parseSearchTags\(options\.tags \|\| ''\)/)
  assert.match(source, /if \(!routeKeyword && !routeGroupCode && !routeTypeCode && !routeTags\.length && routeCityCode === DEFAULT_CITY_CODE\) return false/)
  assert.match(source, /filters\.groupCode = routeGroupCode/)
  assert.match(source, /filters\.typeCode = routeTypeCode/)
  assert.match(source, /filters\.tags = routeTags/)
  assert.match(source, /filters\.groupCode = pendingSearch\.groupCode \|\| ''/)
  assert.match(source, /filters\.typeCode = pendingSearch\.typeCode \|\| ''/)
  assert.match(source, /filters\.tags = parseSearchTags\(pendingSearch\.tags \|\| \[\]\)/)
  assert.match(source, /function hasPendingSearch\(\) \{[\s\S]*return Boolean\(uni\.getStorageSync\(SEARCH_KEY\)\)[\s\S]*\}/)
  assert.doesNotMatch(source, /routeDirection/)
  assert.doesNotMatch(source, /directionStateCache/)
})

test('search placeholder follows the active primary channel', () => {
  assert.match(source, /const searchPlaceholder = computed\(\(\) => filters\.groupCode[\s\S]*`在\$\{selectedGroupName\.value\}中搜索`[\s\S]*'搜供应、需求、场地或服务'[\s\S]*\)/)
  assert.match(source, /<input v-model="keyword" class="search-input" :placeholder="searchPlaceholder" @confirm="search" \/>/)
})

test('search page submits group and type filters to search API', () => {
  assert.match(source, /searchResources\(\{[\s\S]*\.\.\.filters,[\s\S]*tags: filters\.tags\.join\(','\),[\s\S]*keyword: keyword\.value\.trim\(\),[\s\S]*page: nextPage,[\s\S]*pageSize,[\s\S]*\}\)/)
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

test('search page keeps horizontal category scroll uncontrolled during gestures', () => {
  assert.match(source, /const searchRequestSeq = ref\(0\)/)
  assert.match(source, /import \{ onLoad, onPageScroll, onReachBottom, onShow \} from '@dcloudio\/uni-app'/)
  assert.match(source, /<scroll-view[\s\S]*scroll-x[\s\S]*scroll-with-animation[\s\S]*enhanced[\s\S]*:show-scrollbar="false"[\s\S]*:scroll-into-view="scrollIntoTypeId"/)
  assert.match(source, /const pageScrollTop = ref\(0\)/)
  assert.match(source, /onPageScroll\(handlePageScroll\)/)
  assert.match(source, /function handlePageScroll\(event = \{\}\) \{[\s\S]*pageScrollTop\.value = scrollTop[\s\S]*\}/)
  assert.doesNotMatch(source, /:scroll-left="typeScrollLeft"/)
  assert.doesNotMatch(source, /@scroll="handleTypeScroll"/)
  assert.doesNotMatch(source, /const typeScrollLeft = ref\(0\)/)
  assert.doesNotMatch(source, /function handleTypeScroll/)
  assert.match(cssBlock('.filter-row'), /overflow-x:\s*auto;/)
  assert.match(cssBlock('.filter-row'), /-webkit-overflow-scrolling:\s*touch;/)
  assert.match(source, /async function restorePageScroll\(scrollTop = pageScrollTop\.value\) \{[\s\S]*uni\.pageScrollTo\(\{ scrollTop, duration: 0 \}\)[\s\S]*\}/)
})

test('search page uses concise placeholder and empty state copy', () => {
  assert.match(source, /const searchPlaceholder = computed/)
  assert.match(source, /const emptyTitle = '暂无匹配内容'/)
  assert.match(source, /const emptyDesc = computed/)
  assert.match(source, /return '换个关键词或分类试试。'/)
  assert.doesNotMatch(source, /暂无匹配供应/)
  assert.doesNotMatch(source, /暂无匹配需求/)
  assert.doesNotMatch(source, /搜索找现货、找库存、找工厂、找服务/)
  assert.doesNotMatch(source, /搜索库存清仓、现货货源、工厂接单、配套服务/)
})

test('search reset keeps the active primary category channel', () => {
  const resetBlock = functionBlock('resetSearchConditions')

  assert.match(resetBlock, /keyword\.value = ''/)
  assert.match(resetBlock, /filters\.typeCode = ''/)
  assert.match(resetBlock, /filters\.tags = \[\]/)
  assert.doesNotMatch(resetBlock, /filters\.groupCode = ''/)
  assert.match(resetBlock, /applyCurrentGroupTypes\(\)/)
  assert.match(resetBlock, /await search\(\)/)
})

test('search empty state actions keep the primary channel while relaxing narrow filters', () => {
  const clearKeywordBlock = functionBlock('clearSearchKeyword')
  const primaryActionBlock = functionBlock('handleEmptyPrimaryAction')
  const resetBlock = functionBlock('resetSearchConditions')

  assert.match(source, /const emptyDesc = computed\(\(\) => \{[\s\S]*if \(trimmedKeyword\.value\)[\s\S]*当前关键词暂无匹配。[\s\S]*if \(filters\.tags\.length\) return '当前标签暂无结果。'[\s\S]*if \(filters\.groupCode && filters\.typeCode\)[\s\S]*当前分类暂无结果。[\s\S]*if \(filters\.groupCode\) return '该频道暂无内容。'[\s\S]*换个关键词或分类试试。[\s\S]*\}\)/)
  assert.match(source, /const emptyPrimaryActionLabel = computed\(\(\) => \{[\s\S]*if \(trimmedKeyword\.value\) return '清空关键词'[\s\S]*if \(filters\.tags\.length\) return '清空标签'[\s\S]*if \(filters\.groupCode && filters\.typeCode\)[\s\S]*`查看\$\{selectedGroupName\.value\}全部`[\s\S]*return ''[\s\S]*\}\)/)
  assert.match(source, /<view v-if="emptyPrimaryActionLabel" class="empty-actions">/)
  assert.match(source, /<button v-if="emptyPrimaryActionLabel" class="primary-empty-button" @click="handleEmptyPrimaryAction">/)
  assert.doesNotMatch(source, /emptySecondaryActionLabel/)
  assert.doesNotMatch(source, /class="secondary-button"/)
  assert.doesNotMatch(source, /换个条件<\/button>/)

  assert.match(clearKeywordBlock, /keyword\.value = ''/)
  assert.doesNotMatch(clearKeywordBlock, /filters\.groupCode = ''/)
  assert.doesNotMatch(clearKeywordBlock, /filters\.typeCode = ''/)
  assert.match(clearKeywordBlock, /await search\(\)/)

  assert.match(primaryActionBlock, /if \(trimmedKeyword\.value\)[\s\S]*await clearSearchKeyword\(\)[\s\S]*return/)
  assert.match(primaryActionBlock, /if \(filters\.tags\.length\)[\s\S]*await clearSearchTags\(\)[\s\S]*return/)
  assert.match(primaryActionBlock, /if \(filters\.groupCode && filters\.typeCode\)[\s\S]*await resetSearchConditions\(\)/)
  assert.match(resetBlock, /filters\.typeCode = ''/)
  assert.match(resetBlock, /filters\.tags = \[\]/)
  assert.doesNotMatch(resetBlock, /filters\.groupCode = ''/)
})

test('search page keeps compact control sizing and subdued empty state', () => {
  assert.match(cssBlock('.search-page'), /display:\s*flex;/)
  assert.match(cssBlock('.search-page'), /flex-direction:\s*column;/)
  assert.match(cssBlock('.filter-button'), /font-size:\s*26rpx;/)
  assert.match(cssBlock('.search-bar'), /grid-template-columns:\s*1fr 116rpx;/)
  assert.match(cssBlock('.search-button'), /height:\s*76rpx;/)
  assert.match(cssBlock('.search-button'), /font-weight:\s*700;/)
  assert.match(cssBlock('.empty-card'), /flex:\s*1;/)
  assert.match(cssBlock('.empty-card'), /align-content:\s*center;/)
  assert.match(cssBlock('.empty-card'), /min-height:\s*420rpx;/)
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.match(cssBlock('.empty-card'), /padding-bottom:\s*88rpx;/)
  assert.doesNotMatch(cssBlock('.empty-card'), /box-shadow/)
  assert.match(cssBlock('.empty-visual'), /width:\s*168rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
