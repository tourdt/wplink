import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

const VISITOR_STORAGE_KEY = 'wplink_map_visitor_key'
const root = path.resolve(new URL('..', import.meta.url).pathname)

async function loadAnalytics({ clearSession = () => {}, redirectToLogin = () => {} } = {}) {
  const requestSource = fs.readFileSync(path.join(root, 'api/request.js'), 'utf8')
    .replace(/^import .*$/gm, '')
    .replace('export default function request', 'function request')
  const metricsSource = fs.readFileSync(path.join(root, 'api/metrics.js'), 'utf8')
    .replace(/^import .*$/gm, '')
    .replaceAll('export function ', 'function ')
  const analyticsSource = fs.readFileSync(path.join(root, 'common/merchantMapAnalytics.js'), 'utf8')
    .replace(/^import .*$/gm, '')
    .replaceAll('export function ', 'function ')
  const sandbox = {
    API_BASE_URL: '',
    STORAGE_KEYS: { token: 'wplink_token' },
    buildApiUrl(baseUrl, apiPath) {
      return baseUrl ? `${baseUrl}${apiPath}` : apiPath
    },
    clearSession,
    redirectToLogin,
    uni: globalThis.uni,
  }
  vm.runInNewContext(`${requestSource}\n${metricsSource}\n${analyticsSource}\n` +
    'globalThis.analytics = { trackMerchantMapEvent }', sandbox)
  return sandbox.analytics
}

function installUniMock({ storage = new Map(), failRequest = false, unauthorizedRequest = false } = {}) {
  const requests = []
  const storageWrites = []
  const toasts = []
  globalThis.uni = {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    setStorageSync(key, value) {
      storage.set(key, value)
      storageWrites.push([key, value])
    },
    request(options) {
      requests.push(options)
      if (failRequest) {
        options.fail(new Error('network unavailable'))
        return
      }
      if (unauthorizedRequest) {
        options.success({
          statusCode: 401,
          data: { errorCode: 'UNAUTHORIZED', msg: '登录已过期，请重新登录' },
        })
        return
      }
      options.success({ statusCode: 204, data: {} })
    },
    showToast(options) {
      toasts.push(options)
    },
  }
  return { requests, storage, storageWrites, toasts }
}

function settleRequests() {
  return new Promise((resolve) => setImmediate(resolve))
}

function plain(value) {
  return JSON.parse(JSON.stringify(value))
}

test('map analytics persists one bounded visitor key and reuses one bounded in-process session id', async () => {
  const uniMock = installUniMock()
  const { trackMerchantMapEvent } = await loadAnalytics()

  trackMerchantMapEvent({
    merchantId: ' merchant-main ',
    eventType: 'location_view',
    source: 'merchant_location',
  })
  trackMerchantMapEvent({
    merchantId: 'merchant-main',
    targetMerchantId: ' nearby-2 ',
    eventType: 'nearby_marker_click',
    source: 'merchant_location',
  })
  await settleRequests()

  assert.equal(uniMock.requests.length, 2)
  assert.equal(uniMock.storageWrites.length, 1)
  assert.equal(uniMock.storageWrites[0][0], VISITOR_STORAGE_KEY)
  assert.equal(uniMock.storage.get(VISITOR_STORAGE_KEY), uniMock.requests[0].data.visitorKey)
  assert.ok(uniMock.requests[0].data.visitorKey.length > 0)
  assert.ok(uniMock.requests[0].data.visitorKey.length <= 96)
  assert.equal(uniMock.requests[1].data.visitorKey, uniMock.requests[0].data.visitorKey)
  assert.ok(uniMock.requests[0].data.sessionId.length > 0)
  assert.ok(uniMock.requests[0].data.sessionId.length <= 96)
  assert.equal(uniMock.requests[1].data.sessionId, uniMock.requests[0].data.sessionId)
  assert.equal(uniMock.requests[0].url.endsWith('/api/v1/metrics/merchant-map-events'), true)
  assert.equal(uniMock.requests[0].method, 'POST')
  assert.deepEqual(plain(uniMock.requests[1].data), {
    merchantId: 'merchant-main',
    targetMerchantId: 'nearby-2',
    eventType: 'nearby_marker_click',
    source: 'merchant_location',
    visitorKey: uniMock.requests[0].data.visitorKey,
    sessionId: uniMock.requests[0].data.sessionId,
  })
})

test('map analytics keeps using the persisted visitor key on a later module session', async () => {
  const storage = new Map([[VISITOR_STORAGE_KEY, 'persisted-map-visitor']])
  const uniMock = installUniMock({ storage })
  const { trackMerchantMapEvent } = await loadAnalytics()

  trackMerchantMapEvent({
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  })
  await settleRequests()

  assert.equal(uniMock.requests[0].data.visitorKey, 'persisted-map-visitor')
  assert.deepEqual(uniMock.storageWrites, [])
})

test('map analytics replaces a persisted visitor key that exceeds the backend limit', async () => {
  const storage = new Map([[VISITOR_STORAGE_KEY, 'v'.repeat(97)]])
  const uniMock = installUniMock({ storage })
  const { trackMerchantMapEvent } = await loadAnalytics()

  trackMerchantMapEvent({
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  })
  await settleRequests()

  assert.equal(uniMock.requests.length, 1)
  assert.ok(uniMock.requests[0].data.visitorKey.length <= 96)
  assert.notEqual(uniMock.requests[0].data.visitorKey, 'v'.repeat(97))
  assert.equal(uniMock.storageWrites.length, 1)
  assert.equal(uniMock.storageWrites[0][0], VISITOR_STORAGE_KEY)
})

test('map analytics does not request for a blank merchant or unknown event and source', async () => {
  const uniMock = installUniMock()
  const { trackMerchantMapEvent } = await loadAnalytics()

  trackMerchantMapEvent({ merchantId: '   ', eventType: 'location_view', source: 'merchant_location' })
  trackMerchantMapEvent({ merchantId: 'merchant-main', eventType: 'location_open', source: 'merchant_location' })
  trackMerchantMapEvent({ merchantId: 'merchant-main', eventType: 'location_view', source: 'merchant_list' })
  await settleRequests()

  assert.equal(uniMock.requests.length, 0)
  assert.equal(uniMock.storageWrites.length, 0)
})

test('map analytics failure stays silent and is attempted only once', async () => {
  const uniMock = installUniMock({ failRequest: true })
  const { trackMerchantMapEvent } = await loadAnalytics()

  const result = trackMerchantMapEvent({
    merchantId: 'merchant-main',
    eventType: 'navigation_click',
    source: 'merchant_location',
  })
  await settleRequests()

  assert.equal(result, undefined)
  assert.equal(uniMock.requests.length, 1)
  assert.deepEqual(uniMock.toasts, [])
})

test('map analytics keeps an expired optional login session untouched when backend returns unauthorized', async () => {
  const storage = new Map([
    ['wplink_token', 'expired-token'],
    [VISITOR_STORAGE_KEY, 'persisted-map-visitor'],
  ])
  const uniMock = installUniMock({ storage, unauthorizedRequest: true })
  const authSideEffects = []
  const { trackMerchantMapEvent } = await loadAnalytics({
    clearSession() {
      authSideEffects.push('clearSession')
    },
    redirectToLogin() {
      authSideEffects.push('redirectToLogin')
    },
  })

  trackMerchantMapEvent({
    merchantId: 'merchant-main',
    eventType: 'location_view',
    source: 'merchant_location',
  })
  await settleRequests()

  assert.equal(uniMock.requests.length, 1)
  assert.equal(uniMock.requests[0].header.Authorization, 'Bearer expired-token')
  assert.deepEqual(authSideEffects, [])
  assert.deepEqual(uniMock.toasts, [])
  assert.equal(storage.get('wplink_token'), 'expired-token')
})
