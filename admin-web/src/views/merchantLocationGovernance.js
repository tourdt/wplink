const locationStatusPresentation = {
  visible: { statusText: '已公开', statusType: 'success' },
  hidden: { statusText: '已隐藏', statusType: 'info' },
  risk_hidden: { statusText: '风险隐藏', statusType: 'danger' },
}

export function buildMerchantListParams(filters, pagination) {
  return {
    ...filters,
    page: pagination.page,
    pageSize: pagination.pageSize,
  }
}

export function merchantLocationSignal(merchant = {}) {
  const presentation = locationStatusPresentation[merchant.locationStatus]
    || { statusText: merchant.locationStatus || '未设置', statusType: 'info' }
  const boothSummary = [merchant.marketName, merchant.buildingName, merchant.floorNo, merchant.boothNo]
    .filter(Boolean)
    .join(' · ')
  return {
    ...presentation,
    boothSummary: boothSummary || '未填写档口信息',
    pendingText: `待处理 ${Number(merchant.pendingLocationReportCount) || 0}`,
  }
}

export function createMerchantLocationGovernance(api, notify = () => {}, makeReactive = (value) => value) {
  const state = makeReactive({
    merchantFilters: {
      cityCode: 'zhili',
      merchantType: '',
      status: '',
      locationStatus: '',
    },
    merchantPage: { page: 1, pageSize: 20, total: 0 },
    merchants: [],
    merchantsLoading: false,
    merchantError: '',
    reportFilters: { status: 'pending' },
    reportPage: { page: 1, pageSize: 20, total: 0 },
    reports: [],
    reportsLoading: false,
    reportError: '',
    reportDrawerOpen: false,
    detail: null,
    detailDrawerOpen: false,
    actions: [],
  })

  let disposed = false
  let merchantRequestGeneration = 0
  let reportRequestGeneration = 0
  let detailRequestGeneration = 0

  async function loadMerchants(page = state.merchantPage.page) {
    if (disposed) return false
    state.merchantPage.page = page
    const generation = ++merchantRequestGeneration
    state.merchantsLoading = true
    state.merchantError = ''
    try {
      const response = await api.listMerchants(buildMerchantListParams(state.merchantFilters, state.merchantPage))
      if (disposed || generation !== merchantRequestGeneration) return false
      state.merchants = response.items || []
      state.merchantPage.total = response.total || 0
      return true
    } catch (error) {
      if (disposed || generation !== merchantRequestGeneration) return false
      state.merchantError = backendMessage(error) || '商家列表加载失败，请重试'
      return false
    } finally {
      if (!disposed && generation === merchantRequestGeneration) state.merchantsLoading = false
    }
  }

  async function loadReports(page = state.reportPage.page) {
    if (disposed) return false
    state.reportPage.page = page
    const generation = ++reportRequestGeneration
    state.reportsLoading = true
    state.reportError = ''
    try {
      const response = await api.listMerchantLocationReports({
        status: state.reportFilters.status,
        page: state.reportPage.page,
        pageSize: state.reportPage.pageSize,
      })
      if (disposed || generation !== reportRequestGeneration || !state.reportDrawerOpen) return false
      state.reports = response.items || []
      state.reportPage.total = response.total || 0
      return true
    } catch (error) {
      if (disposed || generation !== reportRequestGeneration || !state.reportDrawerOpen) return false
      state.reportError = backendMessage(error) || '位置反馈列表加载失败，请重试'
      return false
    } finally {
      if (!disposed && generation === reportRequestGeneration) state.reportsLoading = false
    }
  }

  function openReports(page = 1) {
    if (disposed) return false
    state.reportDrawerOpen = true
    return loadReports(page)
  }

  function closeReports() {
    state.reportDrawerOpen = false
    state.reports = []
    state.reportPage.total = 0
    state.reportsLoading = false
    reportRequestGeneration += 1
  }

  function openDetail(row) {
    if (disposed) return false
    detailRequestGeneration += 1
    state.detail = merchantListRowDetail(row)
    state.detailDrawerOpen = true
    return true
  }

  function closeDetail() {
    state.detailDrawerOpen = false
    state.detail = null
    detailRequestGeneration += 1
  }

  async function reviewReport(report, status, reviewNote = '') {
    if (disposed) return false
    const key = actionKey('report', report.reportId)
    if (hasAction(state.actions, key)) return false
    state.actions.push({ key, action: status })
    try {
      const response = await api.reviewMerchantLocationReport(report.reportId, { status, reviewNote })
      if (disposed) return false
      notify('success', response.message || '位置反馈已处理')
      await Promise.all([
        loadMerchants(state.merchantPage.page),
        state.reportDrawerOpen ? loadReports(state.reportPage.page) : Promise.resolve(),
      ])
      return !disposed
    } catch (error) {
      if (disposed) return false
      notify('error', backendMessage(error) || '位置反馈处理失败，请重试')
      return false
    } finally {
      removeAction(state.actions, key)
    }
  }

  async function updateLocation(merchant, action, reason = '') {
    if (disposed) return false
    const key = actionKey('merchant', merchant.id)
    if (hasAction(state.actions, key)) return false
    state.actions.push({ key, action })
    try {
      const response = await api.updateMerchantLocationStatus(merchant.id, { action, reason })
      if (disposed) return false
      notify('success', response.message || (action === 'restore' ? '位置已恢复' : '位置已隐藏'))
      await loadMerchants(state.merchantPage.page)
      return !disposed
    } catch (error) {
      if (disposed) return false
      notify('error', backendMessage(error) || '位置状态更新失败，请重试')
      return false
    } finally {
      removeAction(state.actions, key)
    }
  }

  function isActionLoading(scope, id, action = '') {
    const current = state.actions.find((item) => item.key === actionKey(scope, id))
    return Boolean(current && (!action || current.action === action))
  }

  function dispose() {
    disposed = true
    merchantRequestGeneration += 1
    reportRequestGeneration += 1
    detailRequestGeneration += 1
    state.merchantsLoading = false
    state.reportsLoading = false
    state.reportDrawerOpen = false
    state.detailDrawerOpen = false
    state.detail = null
    state.actions.splice(0)
  }

  return {
    state,
    loadMerchants,
    loadReports,
    openReports,
    closeReports,
    openDetail,
    closeDetail,
    reviewReport,
    updateLocation,
    isActionLoading,
    dispose,
  }
}

function merchantListRowDetail(row = {}) {
  const fields = [
    'id', 'name', 'merchantType', 'status', 'locationStatus', 'marketName', 'buildingName',
    'floorNo', 'boothNo', 'pendingLocationReportCount', 'lastActiveAt',
  ]
  return Object.fromEntries(fields.map((field) => [field, row[field]]).filter(([, value]) => value !== undefined))
}

function hasAction(actions, key) {
  return actions.some((item) => item.key === key)
}

function removeAction(actions, key) {
  const index = actions.findIndex((item) => item.key === key)
  if (index >= 0) actions.splice(index, 1)
}

function actionKey(scope, id) {
  return `${scope}:${id}`
}

function backendMessage(error) {
  return error?.response?.data?.message || error?.response?.data?.msg || ''
}
