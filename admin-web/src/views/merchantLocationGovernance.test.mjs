import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantListParams,
  createMerchantLocationGovernance,
  merchantLocationSignal,
} from './merchantLocationGovernance.js'

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function apiFixture(overrides = {}) {
  return {
    listMerchants: async () => ({ items: [], total: 0 }),
    listMerchantLocationReports: async () => ({ items: [], total: 0 }),
    reviewMerchantLocationReport: async () => ({ message: '位置反馈已处理' }),
    updateMerchantLocationStatus: async () => ({ message: '商家位置状态已更新' }),
    ...overrides,
  }
}

test('merchant list keeps business status and location status independent across pagination', async () => {
  const requests = []
  const governance = createMerchantLocationGovernance(apiFixture({
    listMerchants: async (params) => {
      requests.push(params)
      return { items: [{ id: 'm-1' }], total: 41 }
    },
  }))
  Object.assign(governance.state.merchantFilters, {
    cityCode: 'zhili',
    merchantType: 'factory',
    status: 'active',
    locationStatus: 'risk_hidden',
  })

  await governance.loadMerchants(2)

  assert.deepEqual(requests, [{
    cityCode: 'zhili',
    merchantType: 'factory',
    status: 'active',
    locationStatus: 'risk_hidden',
    page: 2,
    pageSize: 20,
  }])
  assert.deepEqual(governance.state.merchants, [{ id: 'm-1' }])
  assert.equal(governance.state.merchantPage.total, 41)
  assert.deepEqual(buildMerchantListParams({ status: '', locationStatus: 'visible' }, { page: 3, pageSize: 10 }), {
    status: '',
    locationStatus: 'visible',
    page: 3,
    pageSize: 10,
  })
})

test('report drawer loads the selected status and page without locking merchant list', async () => {
  const pending = deferred()
  const requests = []
  const governance = createMerchantLocationGovernance(apiFixture({
    listMerchantLocationReports: (params) => {
      requests.push(params)
      return pending.promise
    },
  }))
  governance.state.reportFilters.status = 'invalid'

  const loading = governance.openReports(3)
  assert.equal(governance.state.reportDrawerOpen, true)
  assert.equal(governance.state.reportsLoading, true)
  assert.equal(governance.state.merchantsLoading, false)
  pending.resolve({ items: [{ reportId: 'r-3', status: 'invalid' }], total: 25 })
  await loading

  assert.deepEqual(requests, [{ status: 'invalid', page: 3, pageSize: 20 }])
  assert.deepEqual(governance.state.reports, [{ reportId: 'r-3', status: 'invalid' }])
  assert.equal(governance.state.reportPage.total, 25)
})

test('review action prevents duplicate submission and refreshes reports plus merchant pending counts', async () => {
  const review = deferred()
  let reviewCalls = 0
  let merchantRefreshes = 0
  let reportRefreshes = 0
  const notices = []
  const governance = createMerchantLocationGovernance(apiFixture({
    reviewMerchantLocationReport: () => {
      reviewCalls += 1
      return review.promise
    },
    listMerchants: async () => {
      merchantRefreshes += 1
      return { items: [], total: 0 }
    },
    listMerchantLocationReports: async () => {
      reportRefreshes += 1
      return { items: [], total: 0 }
    },
  }), (type, message) => notices.push({ type, message }))
  governance.state.reportDrawerOpen = true

  const first = governance.reviewReport({ reportId: 'r-9' }, 'valid')
  const duplicate = await governance.reviewReport({ reportId: 'r-9' }, 'valid')
  assert.equal(governance.isActionLoading('report', 'r-9'), true)
  assert.equal(duplicate, false)
  assert.equal(reviewCalls, 1)
  review.resolve({ message: '位置反馈已标记有效' })
  await first

  assert.equal(governance.isActionLoading('report', 'r-9'), false)
  assert.equal(merchantRefreshes, 1)
  assert.equal(reportRefreshes, 1)
  assert.deepEqual(notices, [{ type: 'success', message: '位置反馈已标记有效' }])
})

test('hide and restore share a merchant-scoped lock while other rows remain actionable', async () => {
  const firstUpdate = deferred()
  const actions = []
  const governance = createMerchantLocationGovernance(apiFixture({
    updateMerchantLocationStatus: (merchantId, payload) => {
      actions.push({ merchantId, payload })
      if (merchantId === 'm-1') return firstUpdate.promise
      return Promise.resolve({ message: '位置已隐藏' })
    },
  }))

  const hiding = governance.updateLocation({ id: 'm-1' }, 'hide')
  const duplicateRestore = await governance.updateLocation({ id: 'm-1' }, 'restore')
  const otherMerchant = await governance.updateLocation({ id: 'm-2' }, 'hide')
  assert.equal(governance.isActionLoading('merchant', 'm-1'), true)
  assert.equal(governance.isActionLoading('merchant', 'm-1', 'hide'), true)
  assert.equal(governance.isActionLoading('merchant', 'm-1', 'restore'), false)
  assert.equal(duplicateRestore, false)
  assert.equal(otherMerchant, true)
  assert.deepEqual(actions, [
    { merchantId: 'm-1', payload: { action: 'hide', reason: '' } },
    { merchantId: 'm-2', payload: { action: 'hide', reason: '' } },
  ])
  firstUpdate.resolve({ message: '位置已隐藏' })
  await hiding
  assert.equal(governance.isActionLoading('merchant', 'm-1'), false)
})

test('governance displays backend Chinese errors and uses the operation fallback otherwise', async () => {
  const notices = []
  const governance = createMerchantLocationGovernance(apiFixture({
    updateMerchantLocationStatus: async (merchantId) => {
      if (merchantId === 'm-invalid') {
        throw { response: { data: { message: '商家尚未设置有效地图位置，无法恢复' } } }
      }
      throw new Error('Network Error')
    },
  }), (type, message) => notices.push({ type, message }))

  await governance.updateLocation({ id: 'm-invalid' }, 'restore')
  await governance.updateLocation({ id: 'm-network' }, 'hide')

  assert.deepEqual(notices, [
    { type: 'error', message: '商家尚未设置有效地图位置，无法恢复' },
    { type: 'error', message: '位置状态更新失败，请重试' },
  ])
})

test('list failures expose backend Chinese messages or the list-specific fallback', async () => {
  const governance = createMerchantLocationGovernance(apiFixture({
    listMerchants: async () => {
      throw new Error('Network Error')
    },
    listMerchantLocationReports: async () => {
      throw { response: { data: { message: '位置反馈查询暂时不可用' } } }
    },
  }))

  await governance.loadMerchants(1)
  governance.state.reportDrawerOpen = true
  await governance.loadReports(1)

  assert.equal(governance.state.merchantError, '商家列表加载失败，请重试')
  assert.equal(governance.state.reportError, '位置反馈查询暂时不可用')
})

test('detail drawer uses only fields available from the admin merchant list row', () => {
  const governance = createMerchantLocationGovernance(apiFixture())

  governance.openDetail({
    id: 'm-1',
    name: '童装工厂',
    merchantType: 'factory',
    status: 'active',
    locationStatus: 'risk_hidden',
    marketName: '织里童装城',
    buildingName: 'A 区',
    floorNo: '2F',
    boothNo: '2088',
    pendingLocationReportCount: 3,
    lastActiveAt: '2026-08-29T10:00:00+08:00',
    contact: { name: '不应显示' },
    locationName: '不在后台列表契约',
  })

  assert.deepEqual(governance.state.detail, {
    id: 'm-1',
    name: '童装工厂',
    merchantType: 'factory',
    status: 'active',
    locationStatus: 'risk_hidden',
    marketName: '织里童装城',
    buildingName: 'A 区',
    floorNo: '2F',
    boothNo: '2088',
    pendingLocationReportCount: 3,
    lastActiveAt: '2026-08-29T10:00:00+08:00',
  })
  assert.equal(governance.state.detailDrawerOpen, true)
})

test('closing report drawer prevents its stale request from polluting state', async () => {
  const reportRequest = deferred()
  const governance = createMerchantLocationGovernance(apiFixture({
    listMerchantLocationReports: () => reportRequest.promise,
  }))

  const reports = governance.openReports(1)
  governance.closeReports()
  reportRequest.resolve({ items: [{ reportId: 'stale' }], total: 1 })
  await reports
  assert.deepEqual(governance.state.reports, [])
  assert.equal(governance.state.reportDrawerOpen, false)
})

test('dispose prevents stale merchant and report list responses from writing state', async () => {
  const merchantRequest = deferred()
  const reportRequest = deferred()
  const governance = createMerchantLocationGovernance(apiFixture({
    listMerchants: () => merchantRequest.promise,
    listMerchantLocationReports: () => reportRequest.promise,
  }))

  const merchants = governance.loadMerchants(1)
  const reports = governance.openReports(1)
  governance.dispose()
  merchantRequest.resolve({ items: [{ id: 'stale-merchant' }], total: 1 })
  reportRequest.resolve({ items: [{ reportId: 'stale-report' }], total: 1 })
  await Promise.all([merchants, reports])

  assert.deepEqual(governance.state.merchants, [])
  assert.deepEqual(governance.state.reports, [])
  assert.equal(governance.state.merchantsLoading, false)
  assert.equal(governance.state.reportsLoading, false)
  assert.equal(governance.state.reportDrawerOpen, false)
})

test('dispose suppresses update success toast refresh and action loading after await', async () => {
  const update = deferred()
  const notices = []
  let merchantRefreshes = 0
  const governance = createMerchantLocationGovernance(apiFixture({
    updateMerchantLocationStatus: () => update.promise,
    listMerchants: async () => {
      merchantRefreshes += 1
      return { items: [], total: 0 }
    },
  }), (type, message) => notices.push({ type, message }))

  const action = governance.updateLocation({ id: 'm-1' }, 'hide')
  governance.dispose()
  update.resolve({ message: '位置已隐藏' })
  await action

  assert.deepEqual(notices, [])
  assert.equal(merchantRefreshes, 0)
  assert.deepEqual(governance.state.actions, [])
})

test('dispose suppresses review success toast and both refreshes after await', async () => {
  const review = deferred()
  const notices = []
  let merchantRefreshes = 0
  let reportRefreshes = 0
  const governance = createMerchantLocationGovernance(apiFixture({
    reviewMerchantLocationReport: () => review.promise,
    listMerchants: async () => {
      merchantRefreshes += 1
      return { items: [], total: 0 }
    },
    listMerchantLocationReports: async () => {
      reportRefreshes += 1
      return { items: [], total: 0 }
    },
  }), (type, message) => notices.push({ type, message }))
  governance.state.reportDrawerOpen = true

  const action = governance.reviewReport({ reportId: 'r-1' }, 'valid')
  governance.dispose()
  review.resolve({ message: '位置反馈已标记有效' })
  await action

  assert.deepEqual(notices, [])
  assert.equal(merchantRefreshes, 0)
  assert.equal(reportRefreshes, 0)
  assert.deepEqual(governance.state.actions, [])
})

test('location governance signal renders status booth summary and pending count together', () => {
  assert.deepEqual(merchantLocationSignal({
    locationStatus: 'risk_hidden',
    marketName: '中国织里童装城',
    buildingName: 'A 区',
    floorNo: '2F',
    boothNo: '2088',
    pendingLocationReportCount: 3,
  }), {
    statusText: '风险隐藏',
    statusType: 'danger',
    boothSummary: '中国织里童装城 · A 区 · 2F · 2088',
    pendingText: '待处理 3',
  })
  assert.equal(merchantLocationSignal({ locationStatus: 'hidden' }).boothSummary, '未填写档口信息')
})
