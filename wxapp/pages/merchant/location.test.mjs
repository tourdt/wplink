import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import * as locationState from './locationState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/merchant/location.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => { resolve = nextResolve; reject = nextReject })
  return { promise, resolve, reject }
}

function flush() {
  return new Promise((resolve) => setImmediate(resolve))
}

function plain(value) {
  return JSON.parse(JSON.stringify(value))
}

function contextResponse() {
  return {
    current: {
      merchantId: 'merchant-main', name: '主商家', mainCategories: ['童装'], marketName: '织里童装城',
      buildingName: '一期', floorNo: '1F', boothNo: 'A108', addressText: '富民路',
      latitude: 30.8732, longitude: 120.2255,
    },
    nearby: [{ merchantId: 'nearby-2', name: '周边商家', mainCategories: ['女童'], latitude: 30.874, longitude: 120.226, distanceMeters: 180 }],
    nearbyAvailable: true,
    radiusMeters: 1000,
  }
}

function loadPage({
  getMerchantLocationContext = async () => contextResponse(),
  submitMerchantLocationReport = async () => ({ idempotent: false }),
  trackMerchantDirectoryEvent = () => {},
  saveNavigationFeedback = () => true,
  clearNavigationFeedback = () => {},
  consumeNavigationFeedback = () => null,
  requireLogin = () => true,
  timeline = [],
  uni: uniOverrides = {},
} = {}) {
  const script = source.match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
    .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '') || ''
  let loadHook
  let showHook
  const sandbox = {
    ...locationState,
    clearNavigationFeedback,
    computed(getter) { return { get value() { return getter() } } },
    consumeNavigationFeedback,
    getMerchantLocationContext,
    nextTick(callback) { callback() },
    onLoad(callback) { loadHook = callback },
    onShow(callback) { showHook = callback },
    ref(value) { return { value } },
    requireLogin,
    saveNavigationFeedback,
    submitMerchantLocationReport,
    trackMerchantDirectoryEvent,
    uni: {
      createMapContext() { return { includePoints() {}, moveToLocation() {} } },
      navigateBack() {}, switchTab() {},
      navigateTo(options) { timeline.push(['navigateTo', options.url]) },
      openLocation(options) { timeline.push(['openLocation', options]); uniOverrides.openLocation?.(options) },
      showModal(options) { timeline.push(['showModal', options]); options.success?.({ confirm: true }) },
      showToast(options) { timeline.push(['showToast', options.title]) },
      showLoading() {}, hideLoading() {},
      ...uniOverrides,
    },
    console: { error() {}, warn() {} },
    Promise, String, Number, Boolean,
  }
  vm.runInNewContext(`${script}\nglobalThis.page = {
    loadLocationContext, retryNearby, openCurrentLocation, handleMarkerTap, openNearbyDrawer,
    closeNearbyDrawer, selectNearbyMerchant, openNearbyMerchant,
    promptNavigationFeedback, submitNavigationFeedback,
    state: { merchantId, context, loading, errorText, reportSubmitting },
  }`, sandbox)
  sandbox.page.loadHook = (...args) => loadHook(...args)
  sandbox.page.showHook = (...args) => showHook(...args)
  return sandbox.page
}

test('merchant location page remains registered and uses only new directory modules', () => {
  assert.deepEqual(pagesConfig.pages.find((item) => item.path === 'pages/merchant/location'), {
    path: 'pages/merchant/location', style: { navigationBarTitleText: '店铺位置' },
  })
  assert.match(source, /api\/merchantDirectory/)
  assert.match(source, /common\/merchantDirectoryAnalytics/)
  assert.doesNotMatch(source, /sourcingMap|merchantMapAnalytics|merchantPlaceState|objectId|claimed|prelisted/)
})

test('location page records a view only after current merchant context is valid', async () => {
  const events = []
  const page = loadPage({ trackMerchantDirectoryEvent: (event) => events.push(plain(event)) })
  await page.loadHook({ merchantId: ' merchant-main ' })

  assert.deepEqual(events, [{ merchantId: 'merchant-main', eventType: 'location_view', source: 'merchant_location' }])
  assert.equal(page.state.context.value.current.merchantId, 'merchant-main')

  const invalidEvents = []
  const invalidPage = loadPage({
    getMerchantLocationContext: async () => ({ ...contextResponse(), current: { ...contextResponse().current, latitude: '' } }),
    trackMerchantDirectoryEvent: (event) => invalidEvents.push(event),
  })
  await invalidPage.loadHook({ merchantId: 'merchant-main' })
  assert.deepEqual(invalidEvents, [])
  assert.equal(invalidPage.state.errorText.value, '该商家位置待完善')
})

test('nearby retry failure preserves current navigation context and marks nearby unavailable', async () => {
  let calls = 0
  const timeline = []
  const page = loadPage({
    timeline,
    getMerchantLocationContext: async () => {
      calls += 1
      if (calls === 1) return contextResponse()
      throw new Error('nearby database unavailable')
    },
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  page.retryNearby()
  await flush()
  await flush()

  assert.equal(page.state.context.value.current.merchantId, 'merchant-main')
  assert.equal(page.state.context.value.nearbyAvailable, false)
  assert.ok(timeline.some((item) => item[0] === 'showToast' && item[1] === '周边商家加载失败，请重试'))
})

test('navigation stores merchant-only feedback and records navigation immediately before opening Tencent map', async () => {
  const timeline = []
  const saved = []
  const page = loadPage({
    timeline,
    saveNavigationFeedback(target) { saved.push(plain(target)); return true },
    trackMerchantDirectoryEvent(event) { timeline.push(['track', event]) },
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  timeline.length = 0

  page.openCurrentLocation()

  assert.deepEqual(saved, [{ merchantId: 'merchant-main', name: '主商家' }])
  assert.equal(timeline[0][0], 'track')
  assert.equal(timeline[0][1].eventType, 'navigation_click')
  assert.equal(timeline[1][0], 'openLocation')
  assert.equal(timeline[1][1].latitude, 30.8732)
  assert.equal(timeline[1][1].longitude, 120.2255)
})

test('navigation open failure clears pending feedback without losing current context', async () => {
  let clears = 0
  const timeline = []
  const page = loadPage({
    timeline,
    clearNavigationFeedback() { clears += 1 },
    uni: { openLocation(options) { options.fail(new Error('open failed')) } },
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  page.openCurrentLocation()

  assert.equal(clears, 1)
  assert.equal(page.state.context.value.current.merchantId, 'merchant-main')
  assert.ok(timeline.some((item) => item[0] === 'showToast' && item[1] === '导航打开失败，请稍后重试'))
})

test('location onShow submits one logged-in idempotent feedback and records submit event', async () => {
  const report = deferred()
  let reportCalls = 0
  let consumeCalls = 0
  const timeline = []
  const page = loadPage({
    timeline,
    consumeNavigationFeedback() {
      consumeCalls += 1
      return consumeCalls === 1 ? { merchantId: 'merchant-main', name: '主商家' } : null
    },
    submitMerchantLocationReport() { reportCalls += 1; return report.promise },
    trackMerchantDirectoryEvent(event) { timeline.push(['track', event]) },
  })

  page.showHook()
  page.showHook()
  await flush()
  assert.equal(reportCalls, 1)
  assert.equal(page.state.reportSubmitting.value, true)
  report.resolve({ idempotent: true })
  await flush()
  await flush()

  assert.equal(page.state.reportSubmitting.value, false)
  assert.ok(timeline.some((item) => item[0] === 'showToast' && item[1] === '已收到你的反馈'))
  assert.ok(timeline.some((item) => item[0] === 'track' && item[1].eventType === 'location_report_submit'))
})

test('nearby marker list and merchant actions keep distinct directory events', async () => {
  const events = []
  const timeline = []
  const page = loadPage({
    timeline,
    trackMerchantDirectoryEvent(event) { events.push(plain(event)) },
  })
  await page.loadHook({ merchantId: 'merchant-main' })
  events.length = 0

  page.handleMarkerTap({ detail: { markerId: 2 } })
  page.selectNearbyMerchant('nearby-2')
  page.openNearbyMerchant({ merchantId: 'nearby-2' })

  assert.deepEqual(events.map((item) => item.eventType), [
    'nearby_marker_click', 'nearby_drawer_open', 'nearby_list_item_click', 'nearby_merchant_click',
  ])
  assert.ok(timeline.some((item) => item[0] === 'navigateTo' && item[1] === '/pages/merchant/detail?id=nearby-2'))
})
