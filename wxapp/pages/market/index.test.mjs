import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/market/index.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('market page supports pull refresh and load more pagination', () => {
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

test('market page uses category groups instead of direction tabs', () => {
  for (const token of [
    'groupResourceTypes',
    'categoryGroups',
    'groupFilterOptions',
    'channelTitle',
    'selectedGroupName',
    'filters.groupCode',
    'selectGroup',
    'applyCurrentGroupTypes',
    '全部类目',
    '全部分类',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const filters = reactive\(\{[\s\S]*cityCode: DEFAULT_CITY_CODE,[\s\S]*groupCode: '',[\s\S]*typeCode: '',[\s\S]*\}\)/)
  assert.match(source, /const selectedGroupName = computed\(\(\) =>/)
  assert.match(source, /const channelTitle = computed\(\(\) => filters\.groupCode \? selectedGroupName\.value : PAGE_TITLE\)/)
  assert.match(source, /listCityResourceTypes\(filters\.cityCode\)/)
  assert.match(source, /categoryGroups\.value = groupResourceTypes\(resp\.items \|\| \[\]\)/)
  assert.match(source, /const selectedGroup = categoryGroups\.value\.find\(\(item\) => item\.code === filters\.groupCode\)/)
  assert.match(source, /groupCode: filters\.groupCode/)
  assert.match(source, /typeCode: filters\.typeCode/)
  assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*showGroupDrawer\.value = false[\s\S]*filters\.groupCode = groupCode[\s\S]*filters\.typeCode = ''[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
  assert.doesNotMatch(source, /class="filter-row group-row"/)
  assert.doesNotMatch(source, /class="group-select-button"/)
  assert.doesNotMatch(source, />常用分类<\/text>/)

  for (const removedToken of [
    'directionTabs',
    'activeDirection',
    'selectDirection',
    'directionStateCache',
    'title-direction-tabs',
    "direction: activeDirection.value",
    'v-if="activeDirection === RESOURCE_DIRECTION_DEMAND"',
  ]) {
    assert.doesNotMatch(source, new RegExp(removedToken.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('market page keeps a mixed feed without direction filter state', () => {
  assert.match(source, /const RESOURCE_DIRECTION_DEMAND = 'demand'/)
  assert.doesNotMatch(source, /RESOURCE_DIRECTION_SUPPLY/)
  assert.doesNotMatch(source, /direction-filter-row/)
  assert.doesNotMatch(source, /direction-filter-button/)
  assert.doesNotMatch(source, /directionFilterOptions/)
  assert.doesNotMatch(source, /chooseResourceDirection/)
  assert.doesNotMatch(source, /direction:\s*filters\.direction/)
  assert.doesNotMatch(source, /direction:\s*'',/)
  assert.doesNotMatch(source, /filters\.direction/)
})

test('market page renders mixed supply and demand result cards from item direction', () => {
  for (const token of [
    'DemandCard',
    'ResourceCard',
    'RESOURCE_DIRECTION_DEMAND',
    '<template v-for="item in rows" :key="item.id">',
    'v-if="item.direction === RESOURCE_DIRECTION_DEMAND"',
    '<ResourceCard v-else',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('market page uses the custom title bar as the primary category channel switcher', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/market/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /const PAGE_TITLE = '供需市场'/)
  assert.match(source, /<view class="resource-nav" :style="resourceNavStyle">/)
  assert.match(source, /<view class="resource-title-bar" :style="resourceTitleBarStyle">/)
  assert.match(source, /<button class="channel-title-button" @click="openGroupDrawer">/)
  assert.match(source, /<text class="channel-title-text">\{\{ channelTitle \}\}<\/text>/)
  assert.match(source, /<text class="channel-title-arrow"><\/text>/)
  assert.match(cssBlock('.resource-title-bar'), /justify-content:\s*center;/)
  assert.match(cssBlock('.channel-title-button'), /background:\s*transparent;/)
  assert.match(cssBlock('.channel-title-button::after'), /border:\s*0;/)
  assert.match(source, /const resourceNavStyle = computed/)
  assert.match(source, /getMenuButtonBoundingClientRect/)
})

test('market page expands secondary categories inline from a compact arrow control', () => {
  for (const token of [
    'class="channel-title-button"',
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
    'enhanced',
    ':show-scrollbar="false"',
    'showTypePanel',
    'openTypePanel',
    'closeTypePanel',
    'toggleTypePanel',
    'showAllTypeButton',
    'type-panel-toggle',
    'type-panel-arrow',
    'type-panel',
    'type-panel-grid',
    ':aria-expanded="showTypePanel"',
    "showTypePanel ? '收起全部分类' : '展开全部分类'",
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /const showAllTypeButton = computed\(\(\) => resourceTypes\.value\.length - 1 > 3\)/)
  assert.match(source, /<view class="resource-page" :style="resourcePageStyle" @click="closeTypePanel">/)
  assert.match(source, /<view :class="\['filter-shell', showAllTypeButton \? 'has-type-panel-button' : ''\]" @click\.stop>/)
  assert.match(source, /<button[\s\S]*v-if="showAllTypeButton"[\s\S]*class="type-panel-toggle"[\s\S]*@click\.stop="toggleTypePanel"/)
  assert.match(source, /<scroll-view[\s\S]*v-if="showTypePanel"[\s\S]*class="type-panel"[\s\S]*scroll-y/)
  assert.match(source, /v-for="item in resourceTypes"[\s\S]*:class="\['type-panel-button', item\.value === filters\.typeCode \? 'active' : ''\]"/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypePanel\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
  assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*showTypePanel\.value = false[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*\}/)
  assert.doesNotMatch(source, /class="all-type-button"/)
  assert.doesNotMatch(source, /class="type-drawer-mask"/)
  assert.doesNotMatch(source, />\s*全部分类\s*</)
  assert.doesNotMatch(source, /:scroll-left="typeScrollLeft"/)
  assert.doesNotMatch(source, /@scroll="handleTypeScroll"/)
  assert.doesNotMatch(source, /const typeScrollLeft = ref\(0\)/)
  assert.doesNotMatch(source, /function handleTypeScroll/)
  assert.match(cssBlock('.filter-shell'), /height:\s*80rpx;/)
  assert.match(cssBlock('.filter-shell'), /padding:\s*4rpx;/)
  assert.match(cssBlock('.filter-shell.has-type-panel-button'), /grid-template-columns:\s*minmax\(0,\s*1fr\) 72rpx;/)
  assert.match(cssBlock('.filter-row'), /overflow-x:\s*auto;/)
  assert.match(cssBlock('.filter-row'), /-webkit-overflow-scrolling:\s*touch;/)
  assert.match(cssBlock('.type-panel-toggle'), /width:\s*72rpx;/)
  assert.match(cssBlock('.type-panel-toggle'), /height:\s*72rpx;/)
  assert.match(cssBlock('.type-panel'), /max-height:\s*360rpx;/)
  assert.match(cssBlock('.type-panel-grid'), /grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\);/)
  assert.match(cssBlock('.type-panel-button'), /height:\s*72rpx;/)
})

test('market page opens search with category filters and concise placeholder copy', () => {
  assert.match(source, /const searchPlaceholder = computed\(\(\) => filters\.groupCode[\s\S]*`在\$\{selectedGroupName\.value\}中搜索`[\s\S]*'搜供应、需求、场地或服务'[\s\S]*\)/)
  assert.match(source, /const searchOptions = \{[\s\S]*keyword,[\s\S]*groupCode: filters\.groupCode,[\s\S]*typeCode: filters\.typeCode,[\s\S]*cityCode: filters\.cityCode,[\s\S]*\}/)
  assert.match(source, /if \(keyword \|\| filters\.groupCode \|\| filters\.typeCode \|\| filters\.cityCode !== DEFAULT_CITY_CODE\)/)
  assert.match(source, /uni\.setStorageSync\(SEARCH_KEY, searchOptions\)/)
  assert.doesNotMatch(source, /direction: filters\.direction/)
  assert.doesNotMatch(source, /搜索找现货、找库存、找工厂、找服务/)
  assert.doesNotMatch(source, /搜索库存清仓、现货货源、工厂接单、配套服务/)
})

test('market page does not restore retired standalone demand pages', () => {
  for (const removedText of ['提交采购需求', 'openDemand', '/pages/demand/index']) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('market empty state uses concise copy without direction-specific text', () => {
  assert.match(source, /<text class="empty-title">\{\{ recommendationEmptyTitle \}\}<\/text>/)
  assert.match(source, /const recommendationEmptyTitle = '暂无推荐内容'/)
  assert.match(source, /<text class="empty-desc">换个类型或搜索关键词。<\/text>/)
  assert.doesNotMatch(source, /暂无推荐供应/)
  assert.doesNotMatch(source, /暂无推荐需求/)
})

test('market empty state is visually subdued', () => {
  assert.match(cssBlock('.resource-page'), /display:\s*flex;/)
  assert.match(cssBlock('.resource-page'), /flex-direction:\s*column;/)
  assert.match(cssBlock('.empty-card'), /flex:\s*1;/)
  assert.match(cssBlock('.empty-card'), /align-content:\s*center;/)
  assert.match(cssBlock('.empty-card'), /min-height:\s*420rpx;/)
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.match(cssBlock('.empty-card'), /padding-bottom:\s*88rpx;/)
  assert.match(cssBlock('.empty-visual'), /width:\s*148rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*104rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
