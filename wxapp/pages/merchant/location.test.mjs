import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import * as locationState from './locationState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = read('pages/merchant/location.vue')
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

function loadSourcingMapApi(request) {
  const executableSource = apiSource
    .replace("import request from './request'", '')
    .replaceAll('export function ', 'function ')
  const sandbox = { request }
  vm.runInNewContext(`${executableSource}\nglobalThis.sourcingMapApi = { getMerchantLocationContext }`, sandbox)
  return sandbox.sourcingMapApi
}

function loadLocationPage({ getMerchantLocationContext, trackMerchantMapEvent, timeline = [] }) {
  const script = source
    .match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
    .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '') || ''
  let loadHook
  const sandbox = {
    ...locationState,
    computed(getter) {
      return { get value() { return getter() } }
    },
    console: { error() {}, warn() {} },
    getMerchantLocationContext,
    nextTick(callback) {
      callback()
    },
    onLoad(callback) {
      loadHook = callback
    },
    ref(value) {
      return { value }
    },
    trackMerchantMapEvent,
    uni: {
      createMapContext() {
        return {
          includePoints() {},
          moveToLocation() {},
        }
      },
      navigateBack() {},
      navigateTo(options) {
        timeline.push(['navigateTo', options.url])
      },
      openLocation() {
        timeline.push(['openLocation'])
      },
      showToast() {},
      switchTab() {},
    },
  }
  vm.runInNewContext(`${script}\nglobalThis.locationPage = {
    closeNearbyDrawer,
    handleMarkerTap,
    openCurrentLocation,
    openNearbyDrawer,
    openNearbyMerchant,
    retryNearby,
  }`, sandbox)
  sandbox.locationPage.loadHook = (...args) => loadHook(...args)
  return sandbox.locationPage
}

function locationContext() {
  return {
    current: {
      claimed: true,
      sourceType: 'merchant_claimed',
      merchantId: 'merchant-main',
      name: '主商家',
      lat: '30.89912',
      lng: '120.20482',
    },
    nearby: [{
      claimed: true,
      sourceType: 'merchant_claimed',
      merchantId: ' nearby-2 ',
      name: '周边商家',
      lat: '30.90012',
      lng: '120.20582',
      distanceMeters: 180,
    }],
    nearbyAvailable: true,
    radiusMeters: 3000,
  }
}

test('merchant location page is registered with its business title', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/merchant/location')

  assert.deepEqual(page, {
    path: 'pages/merchant/location',
    style: { navigationBarTitleText: '店铺位置' },
  })
})

test('merchant location page loads the context and keeps the current shop as map focus', () => {
  expectTokens(apiSource, [
    'getMerchantLocationContext',
    'suppressErrorToast: true',
  ])
  expectTokens(source, [
    'getMerchantLocationContext',
    'normalizeMerchantLocationContext',
    'buildMerchantLocationMarkers',
    ':scale="mapScale"',
    'const mapScale = ref(16)',
    '@markertap="handleMarkerTap"',
    '导航到店',
  ])
  assert.doesNotMatch(source, /show-location|uni\.getLocation|搜索此区域/)
})

test('merchant location API trims and encodes the merchant id as one path segment', async () => {
  let requestOptions
  const { getMerchantLocationContext } = loadSourcingMapApi((options) => {
    requestOptions = options
    return Promise.resolve({})
  })

  await getMerchantLocationContext(' merchant/a?b#c% ')

  assert.match(
    requestOptions.url,
    /\/api\/v1\/map\/merchants\/merchant%2Fa%3Fb%23c%25\/location-context$/,
  )
  assert.equal(requestOptions.method, 'GET')
  assert.equal(requestOptions.suppressErrorToast, true)
})

test('merchant location page distinguishes full context failure from nearby degradation', () => {
  expectTokens(source, [
    '该商家暂时无法查看',
    '该商家位置待完善',
    '重新加载',
    '返回',
    'nearbyAvailable',
    '周边商家加载失败，请重试',
  ])
})

test('merchant location page exposes current shop details and a safe navigation failure', () => {
  expectTokens(source, [
    '当前档口',
    '店铺地址',
    '主营',
    'uni.openLocation',
    '导航打开失败，请稍后重试',
  ])
})

test('merchant location page exposes the collapsed and half-screen nearby drawer states', () => {
  expectTokens(source, [
    '周边已入驻商家',
    'nearby-drawer',
    'openNearbyDrawer',
    'closeNearbyDrawer',
    'scroll-into-view',
    '附近暂无其他入驻商家',
    '周边商家加载失败，请重试',
  ])
  assert.match(source, /<scroll-view[\s\S]*scroll-y[\s\S]*:scroll-into-view="scrollIntoViewId"/)
  assert.match(source, /max-height:\s*55vh/)
  assert.match(source, /\.nearby-drawer\.expanded\s*\{[^}]*\n\s*height:\s*55vh/)
  assert.match(source, /\.nearby-list\s*\{[^}]*\n\s*height:\s*calc\(55vh - 88rpx\)/)
})

test('merchant location page links a nearby marker, list item, and merchant detail without replacing current context', () => {
  expectTokens(source, [
    'id="merchantLocationMap"',
    'selectNearbyMerchant',
    'openNearbyMerchant',
    "uni.createMapContext('merchantLocationMap')",
    'includePoints',
    'scrollIntoViewId',
    'merchantMainTags(place)',
    'nearbyMerchantDomId(place.merchantId)',
    'scrollIntoViewId.value = nearbyMerchantDomId(normalizedMerchantId)',
    '/pages/merchant/detail?id=',
    'encodeURIComponent(normalizedMerchantId)',
    'mapScale.value = 16',
  ])
  assert.match(source, /:latitude="mapCenter\.latitude"[\s\S]*:longitude="mapCenter\.longitude"/)
  assert.doesNotMatch(source, /:id="`nearby-\$\{place\.merchantId\}`"/)
  assert.doesNotMatch(source, /platformTags/)
})

test('merchant location page records a view only after a valid location context succeeds', async () => {
  const events = []
  const page = loadLocationPage({
    getMerchantLocationContext: async () => locationContext(),
    trackMerchantMapEvent(event) {
      events.push(event)
    },
  })

  await page.loadHook({ merchantId: ' merchant-main ' })

  assert.deepEqual(plain(events), [{
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  }])

  const invalidEvents = []
  const invalidPage = loadLocationPage({
    getMerchantLocationContext: async () => ({
      current: { merchantId: 'merchant-main', name: '无效坐标', lat: '', lng: '120.20482' },
      nearby: [],
      nearbyAvailable: true,
    }),
    trackMerchantMapEvent(event) {
      invalidEvents.push(event)
    },
  })
  await invalidPage.loadHook({ merchantId: 'merchant-main' })
  assert.deepEqual(invalidEvents, [])
})

test('merchant location page records one view across nearby retries but still records after initial recovery', async () => {
  const events = []
  const page = loadLocationPage({
    getMerchantLocationContext: async () => locationContext(),
    trackMerchantMapEvent(event) {
      events.push(event)
    },
  })

  await page.loadHook({ merchantId: 'merchant-main' })
  page.retryNearby()
  await settlePageRequests()
  page.retryNearby()
  await settlePageRequests()

  assert.deepEqual(plain(events), [{
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  }])

  let recoveryAttempt = 0
  const recoveryEvents = []
  const recoveryPage = loadLocationPage({
    getMerchantLocationContext: async () => {
      recoveryAttempt += 1
      if (recoveryAttempt === 1) throw new Error('context unavailable')
      return locationContext()
    },
    trackMerchantMapEvent(event) {
      recoveryEvents.push(event)
    },
  })

  await recoveryPage.loadHook({ merchantId: 'merchant-main' })
  assert.deepEqual(recoveryEvents, [])
  recoveryPage.retryNearby()
  await settlePageRequests()
  recoveryPage.retryNearby()
  await settlePageRequests()

  assert.deepEqual(plain(recoveryEvents), [{
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  }])
})

test('merchant location page records navigation immediately before opening the map without awaiting analytics', async () => {
  const timeline = []
  const page = loadLocationPage({
    getMerchantLocationContext: async () => locationContext(),
    trackMerchantMapEvent(event) {
      timeline.push(['track', event])
      return new Promise(() => {})
    },
    timeline,
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  timeline.length = 0

  page.openCurrentLocation()

  assert.deepEqual(plain(timeline), [
    ['track', { merchantId: 'merchant-main', eventType: 'navigation_click', source: 'merchant_location' }],
    ['openLocation'],
  ])
})

test('merchant location page records drawer transition, nearby marker, and nearby merchant jump at their real actions', async () => {
  const timeline = []
  const page = loadLocationPage({
    getMerchantLocationContext: async () => locationContext(),
    trackMerchantMapEvent(event) {
      timeline.push(['track', event])
      return new Promise(() => {})
    },
    timeline,
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  timeline.length = 0

  page.openNearbyDrawer()
  page.openNearbyDrawer()
  assert.deepEqual(plain(timeline), [[
    'track',
    { merchantId: 'merchant-main', eventType: 'nearby_drawer_open', source: 'merchant_location' },
  ]])

  page.closeNearbyDrawer()
  timeline.length = 0
  page.handleMarkerTap({ detail: { markerId: 2 } })
  assert.deepEqual(plain(timeline), [
    ['track', {
      merchantId: 'merchant-main',
      targetMerchantId: 'nearby-2',
      eventType: 'nearby_marker_click',
      source: 'merchant_location',
    }],
    ['track', {
      merchantId: 'merchant-main',
      eventType: 'nearby_drawer_open',
      source: 'merchant_location',
    }],
  ])

  timeline.length = 0
  page.openNearbyMerchant({ merchantId: ' nearby-2 ' })
  assert.deepEqual(plain(timeline), [
    ['track', {
      merchantId: 'merchant-main',
      targetMerchantId: 'nearby-2',
      eventType: 'nearby_merchant_click',
      source: 'merchant_location',
    }],
    ['navigateTo', '/pages/merchant/detail?id=nearby-2'],
  ])
})

function settlePageRequests() {
  return new Promise((resolve) => setImmediate(resolve))
}
