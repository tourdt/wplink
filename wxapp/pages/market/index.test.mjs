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
    'filters.groupCode',
    'selectGroup',
    'applyCurrentGroupTypes',
    '全部类目',
    '全部分类',
    '常用分类',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const filters = reactive\(\{[\s\S]*cityCode: DEFAULT_CITY_CODE,[\s\S]*groupCode: '',[\s\S]*typeCode: '',[\s\S]*\}\)/)
  assert.match(source, /listCityResourceTypes\(filters\.cityCode\)/)
  assert.match(source, /categoryGroups\.value = groupResourceTypes\(resp\.items \|\| \[\]\)/)
  assert.match(source, /const selectedGroup = categoryGroups\.value\.find\(\(item\) => item\.code === filters\.groupCode\)/)
  assert.match(source, /groupCode: filters\.groupCode/)
  assert.match(source, /typeCode: filters\.typeCode/)
  assert.match(source, /async function selectGroup\(groupCode\) \{[\s\S]*filters\.groupCode = groupCode[\s\S]*filters\.typeCode = ''[\s\S]*applyCurrentGroupTypes\(\)[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)

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

test('market page keeps the custom title bar without a direction switch', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/market/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /const PAGE_TITLE = '供需市场'/)
  assert.match(source, /<view class="resource-nav" :style="resourceNavStyle">/)
  assert.match(source, /<view class="resource-title-bar" :style="resourceTitleBarStyle">/)
  assert.match(source, /<text class="resource-title">供需市场<\/text>/)
  assert.match(cssBlock('.resource-title-bar'), /justify-content:\s*center;/)
  assert.match(source, /const resourceNavStyle = computed/)
  assert.match(source, /getMenuButtonBoundingClientRect/)
})

test('market page shows all secondary categories and scrolls selected category into view', () => {
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
    'drawer-type-grid',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /visibleResourceTypes = computed\(\(\) => resourceTypes\.value\)/)
  assert.match(source, /v-for="item in visibleResourceTypes"[\s\S]*:id="getTypeButtonId\(item\.value\)"/)
  assert.match(source, /async function selectType\(typeCode\) \{[\s\S]*showTypeDrawer\.value = false[\s\S]*scrollToSelectedType\(typeCode\)[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*\}/)
})

test('market page opens search with category filters and concise placeholder copy', () => {
  assert.match(source, /const searchPlaceholder = '搜供应、需求、场地或服务'/)
  assert.match(source, /const searchOptions = \{[\s\S]*keyword,[\s\S]*groupCode: filters\.groupCode,[\s\S]*typeCode: filters\.typeCode,[\s\S]*cityCode: filters\.cityCode,[\s\S]*\}/)
  assert.match(source, /uni\.setStorageSync\(SEARCH_KEY, searchOptions\)/)
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
  assert.match(cssBlock('.empty-card'), /padding:\s*32rpx 24rpx;/)
  assert.match(cssBlock('.empty-visual'), /width:\s*148rpx;/)
  assert.match(cssBlock('.empty-visual'), /height:\s*104rpx;/)
  assert.match(cssBlock('.empty-title'), /font-size:\s*28rpx;/)
  assert.match(cssBlock('.empty-title'), /color:\s*\$wplink-text;/)
  assert.match(cssBlock('.empty-desc'), /font-size:\s*24rpx;/)
})
