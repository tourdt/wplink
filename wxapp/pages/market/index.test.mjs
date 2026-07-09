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

test('supply demand page supports pull refresh and load more pagination', () => {
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

test('supply demand page shows direction tabs and category controls', () => {
  for (const token of [
    'directionTabs',
    'activeDirection',
    'selectDirection',
    '供给',
    '需求',
    "direction: activeDirection.value",
    'DemandCard',
    'v-if="activeDirection === RESOURCE_DIRECTION_DEMAND"',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /const directionTabs = \[[\s\S]*\{ label: '供给', value: RESOURCE_DIRECTION_SUPPLY \}[\s\S]*\{ label: '需求', value: RESOURCE_DIRECTION_DEMAND \}[\s\S]*\]/)
  assert.doesNotMatch(source, /label: '找资源'/)
  assert.doesNotMatch(source, /label: '看需求'/)
  assert.match(source, /listCityResourceTypes\(filters\.cityCode,\s*\{ direction: activeDirection\.value \}\)/)
  assert.match(source, /const PAGE_TITLE = '供需市场'/)
})

test('supply demand page moves direction switch into custom title bar', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/market/index')

  assert.equal(page?.style?.navigationStyle, 'custom')
  assert.match(source, /<view class="resource-nav" :style="resourceNavStyle">/)
  assert.match(source, /<view class="resource-title-bar" :style="resourceTitleBarStyle">/)
  assert.match(source, /<view class="title-direction-tabs">[\s\S]*v-for="item in directionTabs"[\s\S]*@click="selectDirection\(item\.value\)"/)
  assert.doesNotMatch(source, /<text class="resource-page-title">供需<\/text>/)
  assert.doesNotMatch(source, /resource-page-title/)
  assert.match(cssBlock('.resource-title-bar'), /justify-content:\s*center;/)
  assert.doesNotMatch(cssBlock('.resource-title-bar'), /grid-template-columns:\s*112rpx minmax\(0, 320rpx\) 190rpx;/)
  assert.match(source, /const resourceNavStyle = computed/)
  assert.match(source, /getMenuButtonBoundingClientRect/)
  assert.doesNotMatch(source, /<view class="resource-toolbar">[\s\S]*<view class="direction-tabs">[\s\S]*<\/view>[\s\S]*<view class="filter-shell">/)
})

test('supply demand page restores cached direction state without refetching when switching back', () => {
  for (const token of [
    'directionStateCache',
    'createDirectionState',
    'saveCurrentDirectionState',
    'applyDirectionState',
    'prefetchDirectionState',
    'getOppositeDirection',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /async function selectDirection\(direction\) \{[\s\S]*saveCurrentDirectionState\(activeDirection\.value\)[\s\S]*activeDirection\.value = direction[\s\S]*if \(applyDirectionState\(direction\)\) \{[\s\S]*return[\s\S]*\}/)
  assert.match(source, /async function initResourcePage\(\) \{[\s\S]*await loadRecommendedResources\(\{ reset: true \}\)[\s\S]*saveCurrentDirectionState\(activeDirection\.value\)[\s\S]*prefetchDirectionState\(getOppositeDirection\(activeDirection\.value\)\)[\s\S]*\}/)
  assert.match(source, /function applyDirectionState\(direction\) \{[\s\S]*if \(!cachedState\.loaded\) return false[\s\S]*resourceTypes\.value = \[\.\.\.cachedState\.resourceTypes\][\s\S]*rows\.value = \[\.\.\.cachedState\.rows\][\s\S]*return true[\s\S]*\}/)
  assert.match(source, /async function prefetchDirectionState\(direction\) \{[\s\S]*if \(cachedState\.loaded \|\| cachedState\.loading\) return[\s\S]*listCityResourceTypes\(filters\.cityCode,\s*\{ direction \}\)[\s\S]*listResources\(\{[\s\S]*direction,[\s\S]*page: 1,[\s\S]*pageSize,[\s\S]*\}\)[\s\S]*cachedState\.loaded = true[\s\S]*\}/)
})

test('supply demand page preserves filters and scroll progress when switching direction', () => {
  assert.match(source, /import \{ onLoad, onPageScroll, onPullDownRefresh, onReachBottom \} from '@dcloudio\/uni-app'/)
  assert.match(source, /<scroll-view[\s\S]*:scroll-left="typeScrollLeft"[\s\S]*@scroll="handleTypeScroll"/)
  assert.match(source, /const typeScrollLeft = ref\(0\)/)
  assert.match(source, /const pageScrollTop = ref\(0\)/)
  assert.match(source, /onPageScroll\(handlePageScroll\)/)
  assert.match(source, /function createDirectionState\(\) \{[\s\S]*typeCode: ''[\s\S]*scrollIntoTypeId: ''[\s\S]*typeScrollLeft: 0[\s\S]*pageScrollTop: 0[\s\S]*loaded: false[\s\S]*\}/)
  assert.match(source, /function saveCurrentDirectionState\(direction\) \{[\s\S]*cachedState\.typeCode = filters\.typeCode[\s\S]*cachedState\.scrollIntoTypeId = scrollIntoTypeId\.value[\s\S]*cachedState\.typeScrollLeft = typeScrollLeft\.value[\s\S]*cachedState\.pageScrollTop = pageScrollTop\.value[\s\S]*\}/)
  assert.match(source, /function applyDirectionState\(direction\) \{[\s\S]*filters\.typeCode = cachedState\.typeCode[\s\S]*scrollIntoTypeId\.value = cachedState\.scrollIntoTypeId[\s\S]*typeScrollLeft\.value = cachedState\.typeScrollLeft[\s\S]*pageScrollTop\.value = cachedState\.pageScrollTop[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /function handlePageScroll\(event = \{\}\) \{[\s\S]*pageScrollTop\.value = scrollTop[\s\S]*directionStateCache\[activeDirection\.value\]\.pageScrollTop = scrollTop[\s\S]*\}/)
  assert.match(source, /function handleTypeScroll\(event = \{\}\) \{[\s\S]*typeScrollLeft\.value = scrollLeft[\s\S]*directionStateCache\[activeDirection\.value\]\.typeScrollLeft = scrollLeft[\s\S]*\}/)
  assert.match(source, /async function restorePageScroll\(scrollTop = pageScrollTop\.value\) \{[\s\S]*uni\.pageScrollTo\(\{ scrollTop, duration: 0 \}\)[\s\S]*\}/)
  assert.match(source, /async function selectDirection\(direction\) \{[\s\S]*saveCurrentDirectionState\(activeDirection\.value\)[\s\S]*activeDirection\.value = direction[\s\S]*if \(applyDirectionState\(direction\)\) \{[\s\S]*await restorePageScroll\(\)[\s\S]*return[\s\S]*\}/)
})

test('supply demand page uses short search placeholder copy', () => {
  assert.match(source, /const searchPlaceholder = computed\(\(\) => \([\s\S]*\? '搜采购\/找厂\/服务'[\s\S]*: '搜现货\/库存\/工厂'[\s\S]*\)\)/)
  assert.doesNotMatch(source, /搜索找现货、找库存、找工厂、找服务/)
  assert.doesNotMatch(source, /搜索库存清仓、现货货源、工厂接单、配套服务/)
})

test('supply demand page shows all categories and scrolls selected category into view', () => {
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

test('supply demand page does not restore retired standalone demand pages', () => {
  for (const removedText of ['提交采购需求', 'openDemand', '/pages/demand/index']) {
    assert.doesNotMatch(source, new RegExp(removedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('supply recommendation empty state uses concise copy without image text', () => {
  assert.match(source, /<text class="empty-title">\{\{ recommendationEmptyTitle \}\}<\/text>/)
  assert.match(source, /const recommendationEmptyTitle = computed\(\(\) => activeDirection\.value === RESOURCE_DIRECTION_DEMAND \? '暂无推荐需求' : '暂无推荐供给'\)/)
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
