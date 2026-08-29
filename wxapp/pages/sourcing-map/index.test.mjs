import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import * as directoryState from './merchantDirectoryState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/sourcing-map/index.vue'), 'utf8')
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

function loadPage({
  listMerchantDirectory = async () => ({ items: [], total: 0, page: 1, pageSize: 20 }),
  submitMerchantLocationReport = async () => ({ idempotent: false }),
  trackMerchantDirectoryEvent = () => {},
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
    ...directoryState,
    DEFAULT_CITY_CODE: 'zhili',
    computed(getter) { return { get value() { return getter() } } },
    consumeNavigationFeedback,
    listMerchantDirectory,
    onLoad(callback) { loadHook = callback },
    onPullDownRefresh() {},
    onReachBottom() {},
    onShow(callback) { showHook = callback },
    ref(value) { return { value } },
    requireLogin,
    submitMerchantLocationReport,
    trackMerchantDirectoryEvent,
    uni: {
      navigateTo(options) { timeline.push(['navigateTo', options.url]) },
      showModal(options) { timeline.push(['showModal', options]); options.success?.({ confirm: true }) },
      showToast(options) { timeline.push(['showToast', options.title]) },
      showLoading() {}, hideLoading() {}, stopPullDownRefresh() {},
      ...uniOverrides,
    },
    console: { error() {} },
    Promise, Map, Set, JSON, String, Number, Boolean,
  }
  vm.runInNewContext(`${script}\nglobalThis.page = {
    loadMerchants, submitSearch, applyCategoryFilter, clearConditions,
    openMerchantDetail, openMerchantLocation, promptNavigationFeedback, submitNavigationFeedback,
    state: { keyword, categoriesText, merchants, total, page, loading, errorText, reportSubmitting, loadMoreText },
  }`, sandbox)
  sandbox.page.loadHook = (...args) => loadHook(...args)
  sandbox.page.showHook = (...args) => showHook(...args)
  return sandbox.page
}

function merchant(merchantId, name = `商家${merchantId}`) {
  return { merchantId, name, mainCategories: ['童装'], marketName: '织里童装城', boothNo: `A${merchantId}`, latitude: 30.8732, longitude: 120.2255 }
}

test('merchant directory remains the stable list-first tab', () => {
  const tab = pagesConfig.tabBar.list.find((item) => item.pagePath === 'pages/sourcing-map/index')
  assert.equal(tab.text, '拿货档口')
  assert.doesNotMatch(source, /<map\b|canvas|sourceFilters|claimedFilter/)
  assert.match(source, /v-model="categoriesText"/)
  assert.match(source, /:key="merchant\.merchantId"/)
})

test('directory keeps stable pagination and records each merchant exposure once per page lifetime', async () => {
  const queries = []
  const events = []
  const responses = [
    { items: [merchant('1'), merchant('2')], total: 3, page: 1, pageSize: 20 },
    { items: [merchant('2', '重复'), merchant('3')], total: 3, page: 2, pageSize: 20 },
    { items: [merchant('1'), merchant('2')], total: 3, page: 1, pageSize: 20 },
  ]
  const page = loadPage({
    listMerchantDirectory: async (query) => { queries.push(plain(query)); return responses.shift() },
    trackMerchantDirectoryEvent: (event) => events.push(plain(event)),
  })
  page.state.keyword.value = ' A108 '
  page.state.categoriesText.value = '童装, 女童'

  await page.loadMerchants({ reset: true })
  await page.loadMerchants({ reset: false })
  await page.loadMerchants({ reset: true })

  assert.deepEqual(queries, [
    { cityCode: 'zhili', keyword: 'A108', categories: '童装,女童', page: 1, pageSize: 20 },
    { cityCode: 'zhili', keyword: 'A108', categories: '童装,女童', page: 2, pageSize: 20 },
    { cityCode: 'zhili', keyword: 'A108', categories: '童装,女童', page: 1, pageSize: 20 },
  ])
  assert.deepEqual(page.state.merchants.value.map((item) => item.merchantId), ['1', '2'])
  assert.deepEqual(events.filter((item) => item.eventType === 'directory_exposure').map((item) => item.merchantId), ['1', '2', '3'])
})

test('same directory query shares one request while changed filters use latest-wins state', async () => {
  const first = deferred()
  const latest = deferred()
  const calls = []
  const page = loadPage({
    listMerchantDirectory(query) {
      calls.push(plain(query))
      return calls.length === 1 ? first.promise : latest.promise
    },
  })

  const original = page.submitSearch()
  const duplicate = page.submitSearch()
  await flush()
  assert.equal(calls.length, 1)

  page.state.keyword.value = '新条件'
  const changed = page.submitSearch()
  await flush()
  assert.equal(calls.length, 2)
  latest.resolve({ items: [merchant('2', '最新结果')], total: 1, page: 1, pageSize: 20 })
  await changed
  first.resolve({ items: [merchant('1', '迟到结果')], total: 1, page: 1, pageSize: 20 })
  await Promise.all([original, duplicate])

  assert.deepEqual(page.state.merchants.value.map((item) => item.name), ['最新结果'])
  assert.equal(page.state.loading.value, false)
})

test('directory failure clears a reset result and exposes one safe retry message', async () => {
  const page = loadPage({ listMerchantDirectory: async () => { throw new Error('sql: relation does not exist') } })
  page.state.merchants.value = [merchant('old')]

  await page.loadMerchants({ reset: true })

  assert.deepEqual(plain(page.state.merchants.value), [])
  assert.equal(page.state.total.value, 0)
  assert.equal(page.state.errorText.value, '商家列表加载失败，请稍后重试')
  assert.equal(page.state.loading.value, false)
})

test('load-more failure preserves prior merchants and a successful retry clears its error', async () => {
  let calls = 0
  const page = loadPage({
    listMerchantDirectory: async () => {
      calls += 1
      if (calls === 1) return { items: [merchant('1')], total: 2, page: 1, pageSize: 20 }
      if (calls === 2) throw new Error('page two unavailable')
      return { items: [merchant('2')], total: 2, page: 2, pageSize: 20 }
    },
  })

  await page.loadMerchants({ reset: true })
  await page.loadMerchants({ reset: false })
  assert.deepEqual(page.state.merchants.value.map((item) => item.merchantId), ['1'])
  assert.equal(page.state.errorText.value, '商家列表加载失败，请稍后重试')
  assert.equal(page.state.loadMoreText.value, '加载失败，继续上拉重试')

  await page.loadMerchants({ reset: false })
  assert.deepEqual(page.state.merchants.value.map((item) => item.merchantId), ['1', '2'])
  assert.equal(page.state.errorText.value, '')
  assert.equal(page.state.loadMoreText.value, '已展示全部商家')
})

test('directory homepage and location actions record clicks before merchantId-only navigation', () => {
  const timeline = []
  const page = loadPage({
    timeline,
    trackMerchantDirectoryEvent(event) { timeline.push(['track', event]) },
  })
  const item = merchant('merchant/a?b')

  page.openMerchantDetail(item)
  page.openMerchantLocation(item)

  assert.deepEqual(plain(timeline), [
    ['track', { merchantId: 'merchant/a?b', eventType: 'directory_merchant_click', source: 'directory' }],
    ['navigateTo', '/pages/merchant/detail?id=merchant%2Fa%3Fb'],
    ['track', { merchantId: 'merchant/a?b', eventType: 'directory_merchant_click', source: 'directory' }],
    ['navigateTo', '/pages/merchant/location?merchantId=merchant%2Fa%3Fb'],
  ])
})

test('directory onShow consumes navigation feedback once and idempotent report success is tracked', async () => {
  const report = deferred()
  const timeline = []
  let consumeCalls = 0
  let reportCalls = 0
  const page = loadPage({
    timeline,
    consumeNavigationFeedback() {
      consumeCalls += 1
      return consumeCalls === 1 ? { merchantId: 'merchant-1', name: '晨星童装' } : null
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
