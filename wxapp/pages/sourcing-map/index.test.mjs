import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = read('pages/sourcing-map/index.vue')
const legacySource = read('pages/sourcing-map/legacy-canvas.vue')
const apiSource = read('api/sourcingMap.js')
const pagesConfig = JSON.parse(read('pages.json'))

function read(file) {
  const fullPath = path.join(root, file)
  return fs.existsSync(fullPath) ? fs.readFileSync(fullPath, 'utf8') : ''
}

function expectTokens(target, tokens) {
  for (const token of tokens) {
    assert.match(target, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
}

test('sourcing map remains a stable tab and home banner target', () => {
  const tabs = pagesConfig.tabBar.list.map((item) => [item.pagePath, item.text])
  assert.deepEqual(tabs, [
    ['pages/home/index', '首页'],
    ['pages/sourcing-map/index', '拿货地图'],
    ['pages/publish/index', '发布'],
    ['pages/market/index', '供需'],
    ['pages/my/index', '我的'],
  ])
  const homeSource = read('pages/home/index.vue')
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
  assert.match(homeSource, /const tabPages = \[[^\]]*'\/pages\/sourcing-map\/index'/)
  assert.doesNotMatch(homeSource, /const tabPages = \[[^\]]*'\/pages\/messages\/index'/)
})

test('sourcing map uses merchant directory endpoint with list as default view', () => {
  expectTokens(apiSource, ['listMerchantPlaces', '/api/v1/map/merchant-places'])
  expectTokens(source, [
    "const viewMode = ref('list')",
    'listMerchantPlaces',
    'buildMerchantPlaceQuery',
    'MerchantPlaceCard',
    'merchant-list',
    'onReachBottom',
    '暂无匹配商家',
    '商家列表加载失败，请重试',
  ])
})

test('sourcing map search only presents merchant places with source filters', () => {
  expectTokens(source, [
    '搜索商家、档口号或市场',
    'submitSearch',
    'sourceFilters',
    '全部商家',
    '已入驻',
    '待认领',
    'selectSourceFilter',
    'listMapCategories',
  ])
  assert.doesNotMatch(source, /ResourceCard|DemandCard|searchResources/)
  assert.doesNotMatch(source, /拨打电话|复制微信|makePhoneCall|setClipboardData/)
})

test('map view uses native Tencent map and explicit search-this-area behavior', () => {
  expectTokens(source, [
    '<map',
    'merchantTencentMap',
    ':markers="markers"',
    '@markertap="handleMarkerTap"',
    '@regionchange="handleRegionChange"',
    '搜索此区域',
    'searchCurrentMapRegion',
    'uni.createMapContext',
    'uni.getLocation',
    'uni.openLocation',
    'const MAP_PAGE_SIZE = 100',
    "viewMode.value === 'map' ? MAP_PAGE_SIZE : LIST_PAGE_SIZE",
    "['drag', 'scale'].includes(event.causedBy)",
  ])
  assert.doesNotMatch(source, /<canvas|canvas-id=|createSourcingMapRenderer|handleCanvasTouch/)
})

test('selected native marker opens a custom merchant card and prelisted place can be claimed', () => {
  expectTokens(source, [
    'selectedPlace',
    'marker-card',
    'handleMarkerTap',
    'openPlaceClaim',
    '/pages/merchant/map-binding?objectId=',
    '这是我的档口',
    '平台预录',
  ])
})

test('legacy canvas implementation is preserved outside the registered user path', () => {
  assert.match(legacySource, /canvas-id="sourcingMapCanvas"/)
  assert.match(legacySource, /createSourcingMapRenderer/)
  assert.ok(!pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/legacy-canvas'))
})
