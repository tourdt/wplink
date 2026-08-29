import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const VISITOR_STORAGE_KEY = 'wplink_directory_visitor_key'
const FEEDBACK_STORAGE_KEY = 'wplink_directory_navigation_feedback'

function loadAnalytics({ record = async () => ({ recorded: true }) } = {}) {
  const source = fs.readFileSync(path.join(root, 'common/merchantDirectoryAnalytics.js'), 'utf8')
    .replace(/^import .*$/gm, '')
    .replaceAll('export function ', 'function ')
  const sandbox = { recordMerchantDirectoryEvent: record, uni: globalThis.uni, Date, Math, Promise, Set, String }
  vm.runInNewContext(`${source}\nglobalThis.analytics = {
    trackMerchantDirectoryEvent,
    saveNavigationFeedback,
    claimNavigationFeedback: typeof claimNavigationFeedback === 'function' ? claimNavigationFeedback : undefined,
    completeNavigationFeedback: typeof completeNavigationFeedback === 'function' ? completeNavigationFeedback : undefined,
    releaseNavigationFeedback: typeof releaseNavigationFeedback === 'function' ? releaseNavigationFeedback : undefined,
    clearNavigationFeedback,
  }`, sandbox)
  return sandbox.analytics
}

function installUni(storage = new Map()) {
  const writes = []
  const removals = []
  globalThis.uni = {
    getStorageSync(key) { return storage.get(key) || '' },
    setStorageSync(key, value) { storage.set(key, value); writes.push([key, value]) },
    removeStorageSync(key) { storage.delete(key); removals.push(key) },
  }
  return { storage, writes, removals }
}

function plain(value) {
  return JSON.parse(JSON.stringify(value))
}

test('directory analytics accepts all nine events and reuses bounded visitor and session identifiers', async () => {
  const uniMock = installUni()
  const payloads = []
  const analytics = loadAnalytics({ record: async (payload) => { payloads.push(payload); return { recorded: true } } })
  const eventTypes = [
    'directory_exposure', 'directory_merchant_click', 'location_view', 'nearby_marker_click',
    'nearby_list_item_click', 'nearby_merchant_click', 'nearby_drawer_open', 'navigation_click',
    'location_report_submit',
  ]

  for (const eventType of eventTypes) {
    analytics.trackMerchantDirectoryEvent({
      merchantId: ' merchant-main ', targetMerchantId: ' target-2 ', eventType, source: 'merchant_location',
    })
  }
  await new Promise((resolve) => setImmediate(resolve))

  assert.deepEqual(payloads.map((item) => item.eventType), eventTypes)
  assert.equal(new Set(payloads.map((item) => item.visitorKey)).size, 1)
  assert.equal(new Set(payloads.map((item) => item.sessionId)).size, 1)
  assert.ok(payloads[0].visitorKey.length <= 96)
  assert.ok(payloads[0].sessionId.length <= 96)
  assert.equal(payloads[0].merchantId, 'merchant-main')
  assert.equal(payloads[0].targetMerchantId, 'target-2')
  assert.equal(uniMock.writes.filter(([key]) => key === VISITOR_STORAGE_KEY).length, 1)
})

test('directory analytics rejects unknown events and sources before touching storage', async () => {
  const uniMock = installUni()
  const payloads = []
  const analytics = loadAnalytics({ record: async (payload) => payloads.push(payload) })

  analytics.trackMerchantDirectoryEvent({ merchantId: '1', eventType: 'location_entry_click', source: 'merchant_location' })
  analytics.trackMerchantDirectoryEvent({ merchantId: '1', eventType: 'location_view', source: 'legacy_canvas' })
  analytics.trackMerchantDirectoryEvent({ merchantId: ' ', eventType: 'location_view', source: 'merchant_location' })
  await new Promise((resolve) => setImmediate(resolve))

  assert.deepEqual(payloads, [])
  assert.deepEqual(uniMock.writes, [])
})

test('directory analytics failure is silent and is attempted once without retry', async () => {
  installUni()
  let calls = 0
  const analytics = loadAnalytics({ record: async () => { calls += 1; throw new Error('network unavailable') } })

  const result = analytics.trackMerchantDirectoryEvent({
    merchantId: '1', eventType: 'navigation_click', source: 'merchant_location',
  })
  await new Promise((resolve) => setImmediate(resolve))

  assert.equal(result, undefined)
  assert.equal(calls, 1)
})

test('navigation feedback stays stored while one page owns the shared claim and is removed only after completion', () => {
  const uniMock = installUni()
  const analytics = loadAnalytics()

  assert.equal(typeof analytics.claimNavigationFeedback, 'function')
  assert.equal(typeof analytics.completeNavigationFeedback, 'function')
  assert.equal(typeof analytics.releaseNavigationFeedback, 'function')

  assert.equal(analytics.saveNavigationFeedback({
    merchantId: ' merchant-1 ', name: ' 晨星童装 ', objectId: 'legacy-object', claimed: true,
  }), true)
  assert.deepEqual(plain(uniMock.storage.get(FEEDBACK_STORAGE_KEY)), { merchantId: 'merchant-1', name: '晨星童装' })

  const directoryClaim = analytics.claimNavigationFeedback()
  const locationClaim = analytics.claimNavigationFeedback()
  assert.deepEqual(plain(directoryClaim), { merchantId: 'merchant-1', name: '晨星童装' })
  assert.equal(locationClaim, null)
  assert.deepEqual(plain(uniMock.storage.get(FEEDBACK_STORAGE_KEY)), { merchantId: 'merchant-1', name: '晨星童装' })
  assert.equal(uniMock.removals.filter((key) => key === FEEDBACK_STORAGE_KEY).length, 0)

  assert.equal(analytics.releaseNavigationFeedback(directoryClaim), true)
  const retriedClaim = analytics.claimNavigationFeedback()
  assert.deepEqual(plain(retriedClaim), { merchantId: 'merchant-1', name: '晨星童装' })
  assert.equal(analytics.completeNavigationFeedback(retriedClaim), true)
  assert.equal(uniMock.storage.has(FEEDBACK_STORAGE_KEY), false)
  assert.equal(uniMock.removals.filter((key) => key === FEEDBACK_STORAGE_KEY).length, 1)
})

test('directory and location page claims allow at most one concurrent feedback request', async () => {
  installUni(new Map([[FEEDBACK_STORAGE_KEY, { merchantId: 'merchant-1', name: '晨星童装' }]]))
  const analytics = loadAnalytics()
  assert.equal(typeof analytics.claimNavigationFeedback, 'function')
  let requests = 0
  let finishRequest
  const pendingRequest = new Promise((resolve) => { finishRequest = resolve })

  async function runPageShow() {
    const claim = analytics.claimNavigationFeedback()
    if (!claim) return false
    requests += 1
    await pendingRequest
    analytics.completeNavigationFeedback(claim)
    return true
  }

  const directoryShow = runPageShow()
  const locationShow = runPageShow()
  assert.equal(requests, 1)
  finishRequest()
  assert.deepEqual(await Promise.all([directoryShow, locationShow]), [true, false])
})

test('invalid navigation feedback is never persisted and clear removes an armed target', () => {
  const uniMock = installUni()
  const analytics = loadAnalytics()
  assert.equal(typeof analytics.claimNavigationFeedback, 'function')

  assert.equal(analytics.saveNavigationFeedback({ merchantId: ' ', name: '无效商家' }), false)
  assert.equal(uniMock.storage.has(FEEDBACK_STORAGE_KEY), false)
  analytics.saveNavigationFeedback({ merchantId: '2', name: '商家二' })
  analytics.clearNavigationFeedback()
  assert.equal(analytics.claimNavigationFeedback(), null)
})
