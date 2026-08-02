import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import * as merchantPlaceState from './merchantPlaceState.js'

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

function plain(value) {
  return JSON.parse(JSON.stringify(value))
}

function importedMerchantPlaceStateBindings() {
  const match = source.match(/import\s*\{([^}]*)\}\s*from\s*['"]\.\/merchantPlaceState['"]/)
  assert.ok(match, 'directory page should import merchant place state helpers')

  return Object.fromEntries(match[1]
    .split(',')
    .map((name) => name.trim())
    .filter(Boolean)
    .map((name) => [name, merchantPlaceState[name]]))
}

function loadDirectoryPage({
  trackMerchantMapEvent,
  timeline,
  listMapCategories = async () => ({ items: [] }),
  listMerchantPlaces = async () => ({ items: [], total: 0 }),
}) {
  const script = source
    .match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
    .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '') || ''
  const sandbox = {
    ...importedMerchantPlaceStateBindings(),
    DEFAULT_CITY_CODE: 'zhili',
    computed(getter) {
      return { get value() { return getter() } }
    },
    getMerchantId() {
      return ''
    },
    listMapCategories,
    listMerchantPlaces,
    onLoad() {},
    onPullDownRefresh() {},
    onReachBottom() {},
    onShow() {},
    ref(value) {
      return { value }
    },
    requireLogin() {
      return true
    },
    submitMapLocationCorrection: async () => {},
    trackMerchantMapEvent,
    uni: {
      navigateTo(options) {
        timeline.push(['navigateTo', options.url])
      },
    },
  }
  vm.runInNewContext(`${script}\nglobalThis.directoryPage = { loadCategories, loadPlaces, openMerchantLocation, state: { categories, errorText, places } }`, sandbox)
  return sandbox.directoryPage
}

test('merchant booth directory remains a stable tab and home entry target', () => {
  const tabs = pagesConfig.tabBar.list.map((item) => [item.pagePath, item.text])
  assert.deepEqual(tabs, [
    ['pages/home/index', '首页'],
    ['pages/sourcing-map/index', '拿货档口'],
    ['pages/publish/index', '发布'],
    ['pages/market/index', '供需'],
    ['pages/my/index', '我的'],
  ])
  const homeSource = read('pages/home/index.vue')
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
  assert.match(homeSource, /const tabPages = \[[^\]]*'\/pages\/sourcing-map\/index'/)
  assert.doesNotMatch(homeSource, /const tabPages = \[[^\]]*'\/pages\/messages\/index'/)
})

test('merchant booth directory uses the merchant endpoint as its only view', () => {
  expectTokens(apiSource, ['listMerchantPlaces', '/api/v1/map/merchant-places'])
  expectTokens(source, [
    'listMerchantPlaces',
    'buildMerchantPlaceQuery',
    'MerchantPlaceCard',
    'merchant-list',
    'onReachBottom',
    '暂无匹配商家',
    '商家列表加载失败，请重试',
  ])
  assert.doesNotMatch(source, /viewMode|merchantTencentMap|searchCurrentMapRegion|搜索此区域/)
})

test('merchant booth directory omits redundant result count while keeping pagination total', () => {
  assert.doesNotMatch(source, /\{\{\s*total\s*\}\}\s*个档口/)
  assert.doesNotMatch(source, /directory-summary|result-count/)
  expectTokens(source, [
    'const total = ref(0)',
    'const hasMore = computed(() => places.value.length < total.value)',
    'total.value = Number(resp.total || 0)',
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

test('merchant booth directory removes native map mode and stale map layout', () => {
  assert.doesNotMatch(source, /<map|native-map-shell|tencent-map|marker-card|map-loading-pill|map-empty-pill/)
  assert.doesNotMatch(source, /viewMode|markers|selectedPlace|selectedObjectId|mapCenter|mapScale|bounds|MAP_PAGE_SIZE/)
  assert.doesNotMatch(source, /buildTencentMapMarkers|fallbackMapCenter|placeIdFromMarker|uni\.createMapContext|uni\.getLocation/)
})

test('prelisted booth cards keep claim and direct navigation actions', () => {
  expectTokens(source, [
    'openPlaceClaim',
    '/pages/merchant/map-binding?objectId=',
    '@navigate="openPlaceLocation"',
    'uni.openLocation',
  ])
})

test('claimed merchant cards open the homepage or independent location page', () => {
  assert.equal((source.match(/@detail="openMerchantDetail"/g) || []).length, 1)
  expectTokens(source, [
    'merchantDetailPath',
    'function openMerchantDetail(place)',
    '@location="openMerchantLocation"',
    'function openMerchantLocation(place)',
    'if (!hasMerchantDetail(place) || !hasValidLocation(place)) return',
    "const merchantId = String(place.merchantId || '').trim()",
    '/pages/merchant/location?merchantId=',
    'encodeURIComponent(merchantId)',
    'uni.navigateTo({ url })',
  ])
})

test('legacy canvas implementation is preserved outside the registered user path', () => {
  assert.match(legacySource, /canvas-id="sourcingMapCanvas"/)
  assert.match(legacySource, /createSourcingMapRenderer/)
  assert.ok(!pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/legacy-canvas'))
})

test('merchant directory uses declared state imports and records its valid location entry immediately before navigation', () => {
  const timeline = []
  const page = loadDirectoryPage({
    trackMerchantMapEvent(event) {
      timeline.push(['track', event])
      return new Promise(() => {})
    },
    timeline,
  })
  const place = {
    claimed: true,
    merchantId: ' merchant-directory ',
    lat: '30.89912',
    lng: '120.20482',
  }

  page.openMerchantLocation(place)

  assert.deepEqual(plain(timeline), [
    ['track', {
      merchantId: 'merchant-directory',
      eventType: 'location_entry_click',
      source: 'directory',
    }],
    ['navigateTo', '/pages/merchant/location?merchantId=merchant-directory'],
  ])

  timeline.length = 0
  page.openMerchantLocation({ ...place, lat: '' })
  assert.deepEqual(timeline, [])
})

test('merchant directory translates configured tag codes before rendering cards', async () => {
  const categoryTypes = []
  const page = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    async listMapCategories({ type }) {
      categoryTypes.push(type)
      return {
        items: {
          booth_category: [{ code: 'girl', name: '女童' }],
          booth_service: [{ code: 'spot', name: '现货' }],
          platform_tag: [{ code: 'hot', name: '热门推荐' }],
        }[type] || [],
      }
    },
    async listMerchantPlaces() {
      return {
        items: [{
          objectId: 'A001',
          name: '晨星童装',
          categoryCodes: ['girl'],
          serviceTags: ['spot'],
          platformTags: ['hot'],
        }],
        total: 1,
      }
    },
  })

  await page.loadCategories()
  await page.loadPlaces({ reset: true })

  assert.deepEqual(plain(categoryTypes), ['booth_category', 'booth_service', 'platform_tag'])
  assert.equal(page.state.errorText.value, '')
  assert.deepEqual(plain(page.state.places.value[0].displayTags), ['女童', '现货', '热门推荐'])
})
