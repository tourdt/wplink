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

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function flushAsyncWork() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function createTrackedRefFactory() {
  const writes = new WeakMap()
  return {
    ref(initialValue) {
      let currentValue = initialValue
      const state = {}
      writes.set(state, [])
      Object.defineProperty(state, 'value', {
        get: () => currentValue,
        set(nextValue) {
          writes.get(state).push(nextValue)
          currentValue = nextValue
        },
      })
      return state
    },
    writesFor(state) {
      return writes.get(state) || []
    },
  }
}

function ref(value) {
  return { value }
}

function reactive(value) {
  return value
}

function computed(getter) {
  return { get value() { return getter() } }
}

function loadLegacyCanvasPage(additions = {}) {
  const script = legacySource
    .match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
    .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '') || ''
  const sandbox = {
    console,
    Promise,
    Set,
    Date,
    String,
    Number,
    Boolean,
    Math,
    setTimeout,
    clearTimeout,
    ref,
    reactive,
    computed,
    watch: () => {},
    onLoad: () => {},
    onReady: () => {},
    onUnmounted: () => {},
    getCurrentInstance: () => null,
    DEFAULT_CITY_CODE: 'zhili',
    requireLogin: () => true,
    createInitialTransform: () => ({ scale: 1, offsetX: 0, offsetY: 0 }),
    endGesture: (state) => state.transform,
    moveGesture: () => ({}),
    screenToMap: () => ({}),
    startGesture: () => ({}),
    mapObjectCenter: () => null,
    hitTestMapObjects: () => null,
    isObjectInBounds: () => true,
    isRentableMapObject: () => false,
    isVerifiedMapObject: () => false,
    mapObjectIdentity: (item) => String(item?.id || ''),
    createSourcingMapRenderer: () => ({ dispose: () => {} }),
    listMapScenes: async () => ({ items: [] }),
    listMapCategories: async () => ({ items: [] }),
    listMapObjects: async () => ({ items: [], total: 0 }),
    searchMapObjects: async () => ({ items: [], total: 0 }),
    listNearbyPois: async () => ({ items: [] }),
    getMapObject: async () => ({}),
    submitMapLocationCorrection: async () => ({}),
    submitMapRiskReport: async () => ({}),
    uni: { showToast: () => {}, showActionSheet: () => {}, redirectTo: () => {}, switchTab: () => {} },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.legacyPage = { selectedScene, selectedSceneCode, keyword, activeFilters, rawMapObjects, objectLoading, selectScene, submitSearch, toggleFilter }`, sandbox)
  return sandbox.legacyPage
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
  submitMapLocationCorrection = async () => {},
  requireLogin = () => true,
  refFactory,
  uni: uniOverrides = {},
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
    ref: refFactory || ((value) => ({ value })),
    requireLogin,
    submitMapLocationCorrection,
    trackMerchantMapEvent,
    uni: {
      navigateTo(options) {
        timeline.push(['navigateTo', options.url])
      },
      ...uniOverrides,
    },
  }
  vm.runInNewContext(`${script}\nglobalThis.directoryPage = { loadCategories, loadPlaces, submitSearch, selectSourceFilter, toggleCategory, clearConditions, openMerchantLocation, submitLocationCorrection, state: { keyword, claimedFilter, categoryCodes, categories, errorText, places, loading, navigationCorrectionBusy: typeof navigationCorrectionBusy === 'undefined' ? undefined : navigationCorrectionBusy } }`, sandbox)
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

test('merchant directory condition changes are latest-wins while explicit duplicate search stays deduplicated', async () => {
  const firstRequest = deferred()
  const latestRequest = deferred()
  const queries = []
  const page = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    listMerchantPlaces(query) {
      queries.push(plain(query))
      return queries.length === 1 ? firstRequest.promise : latestRequest.promise
    },
  })

  page.state.keyword.value = '旧条件'
  const firstLoad = page.loadPlaces({ reset: true })
  page.submitSearch()
  await flushAsyncWork()
  assert.equal(queries.length, 1, '相同显式搜索在请求中仍应防重复')

  const latestLoad = page.selectSourceFilter('claimed')
  await flushAsyncWork()
  assert.equal(queries.length, 2, '筛选变化必须启动新的 reset 请求')
  assert.equal(queries[1].claimed, 'claimed')
  assert.equal(page.state.loading.value, true, '最新请求完成前应持续显示加载状态')

  firstRequest.resolve({ items: [{ objectId: 'old', name: '旧结果' }], total: 1 })
  await firstLoad
  assert.deepEqual(plain(page.state.places.value), [], '旧响应不得覆盖最新筛选结果')
  assert.equal(page.state.loading.value, true, '旧请求完成不得提前释放最新请求的 loading')

  latestRequest.resolve({ items: [{ objectId: 'latest', name: '最新结果' }], total: 1 })
  await latestLoad
  assert.deepEqual(plain(page.state.places.value).map((item) => item.objectId), ['latest'])
  assert.equal(page.state.loading.value, false, '最新请求完成后才释放 loading')
})

test('merchant directory reuses a pending request when the active source filter is clicked again', async () => {
  const request = deferred()
  let requestCalls = 0
  const page = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    listMerchantPlaces() {
      requestCalls += 1
      return request.promise
    },
  })

  const first = page.selectSourceFilter('claimed')
  const duplicate = page.selectSourceFilter('claimed')
  await flushAsyncWork()
  assert.equal(requestCalls, 1, '重复点击当前来源筛选应复用同签名 pending 请求')

  request.resolve({ items: [{ objectId: 'claimed-result' }], total: 1 })
  await Promise.all([first, duplicate])
  assert.deepEqual(plain(page.state.places.value).map((item) => item.objectId), ['claimed-result'])
})

test('legacy canvas reuses a pending request for the active scene', async () => {
  const request = deferred()
  let requestCalls = 0
  const page = loadLegacyCanvasPage({
    listMapObjects() {
      requestCalls += 1
      return request.promise
    },
  })
  const scene = { code: 'scene-1', width: 750, height: 1000 }

  const first = page.selectScene(scene)
  const duplicate = page.selectScene(scene)
  await flushAsyncWork()
  assert.equal(requestCalls, 1, '重复选择当前场景应复用同签名 pending 请求')

  request.resolve({ items: [], total: 0 })
  await Promise.all([first, duplicate])
  assert.equal(page.objectLoading.value, false)
})

test('legacy canvas still applies the latest result when the object query signature changes', async () => {
  const firstRequest = deferred()
  const latestRequest = deferred()
  const queries = []
  const page = loadLegacyCanvasPage({
    searchMapObjects(query) {
      queries.push(plain(query))
      return queries.length === 1 ? firstRequest.promise : latestRequest.promise
    },
  })
  page.selectedSceneCode.value = 'scene-1'
  page.keyword.value = '童装'

  const first = page.submitSearch()
  const latest = page.toggleFilter('categories', 'girl')
  await flushAsyncWork()
  assert.equal(queries.length, 2, '对象查询签名变化应启动新版本请求')
  assert.equal(queries[1].categories, 'girl')

  latestRequest.resolve({ items: [{ id: 'latest' }], total: 1 })
  await latest
  firstRequest.resolve({ items: [{ id: 'old' }], total: 1 })
  await first
  assert.deepEqual(plain(page.rawMapObjects.value).map((item) => item.id), ['latest'])
})

test('navigation correction shows local feedback, deduplicates requests, and clears state on success or failure', async () => {
  const request = deferred()
  const loadingCalls = []
  const toasts = []
  let requestCalls = 0
  const page = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    submitMapLocationCorrection() {
      requestCalls += 1
      return request.promise
    },
    uni: {
      showLoading: (options) => loadingCalls.push(['show', plain(options)]),
      hideLoading: () => loadingCalls.push(['hide']),
      showToast: (options) => toasts.push(options),
    },
  })

  const first = page.submitLocationCorrection('object-1')
  const duplicate = page.submitLocationCorrection('object-1')
  await flushAsyncWork()
  assert.ok(page.state.navigationCorrectionBusy, '页面应提供导航纠错局部 busy 状态')
  assert.equal(requestCalls, 1, '纠错提交期间重复调用只应发送一次请求')
  assert.equal(page.state.navigationCorrectionBusy.value, true, '纠错请求期间应保留局部 busy')
  assert.deepEqual(loadingCalls, [['show', { title: '提交中', mask: true }]], '纠错请求应显示带遮罩的局部反馈')

  request.resolve({})
  await Promise.all([first, duplicate])
  assert.equal(toasts.at(-1)?.title, '反馈已提交')
  assert.equal(page.state.navigationCorrectionBusy.value, false, '纠错成功后应释放局部 busy')
  assert.deepEqual(loadingCalls.at(-1), ['hide'], '纠错成功后应关闭局部 loading')

  const failedLoadingCalls = []
  const failedToasts = []
  const failedPage = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    submitMapLocationCorrection: async () => { throw new Error('网络异常') },
    uni: {
      showLoading: (options) => failedLoadingCalls.push(['show', plain(options)]),
      hideLoading: () => failedLoadingCalls.push(['hide']),
      showToast: (options) => failedToasts.push(options),
    },
  })
  await failedPage.submitLocationCorrection('object-2')
  assert.equal(failedToasts.at(-1)?.title, '网络异常', '纠错失败应保留中文友好提示')
  assert.equal(failedPage.state.navigationCorrectionBusy.value, false, '纠错失败后应释放局部 busy')
  assert.deepEqual(failedLoadingCalls.at(-1), ['hide'], '纠错失败后也应关闭局部 loading')
})

test('unauthenticated navigation correction performs zero busy writes, requests, or loading feedback', async () => {
  const tracker = createTrackedRefFactory()
  let requestCalls = 0
  let loadingCalls = 0
  const page = loadDirectoryPage({
    trackMerchantMapEvent() {},
    timeline: [],
    requireLogin: () => false,
    refFactory: tracker.ref,
    submitMapLocationCorrection: async () => { requestCalls += 1 },
    uni: {
      showLoading: () => { loadingCalls += 1 },
      hideLoading: () => { loadingCalls += 1 },
    },
  })

  await page.submitLocationCorrection('object-1')

  assert.ok(page.state.navigationCorrectionBusy, '未登录路径应观察真实的纠错 busy 状态')
  assert.deepEqual(tracker.writesFor(page.state.navigationCorrectionBusy), [], '未登录不得短暂写入纠错 busy')
  assert.equal(requestCalls, 0, '未登录不得提交纠错请求')
  assert.equal(loadingCalls, 0, '未登录不得闪现纠错 loading')
})
