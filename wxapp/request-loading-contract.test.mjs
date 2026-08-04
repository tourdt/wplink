import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

const root = path.resolve(new URL('.', import.meta.url).pathname)

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8')
}

function templateButton(relativePath, clickExpression) {
  const { descriptor } = parse(read(relativePath), { filename: relativePath })
  const buttons = descriptor.template.content.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) || []
  const button = buttons.find((item) => item.includes(`@click="${clickExpression}"`))
  assert.ok(button, `${relativePath} should contain the ${clickExpression} button`)
  return button
}

async function renderBusyButton(contract) {
  const source = templateButton(contract.file, contract.click)
  const compiled = compileTemplate({
    source,
    filename: contract.file,
    id: 'request-loading-contract',
    compilerOptions: { mode: 'function' },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const buttonComponent = {
    props: Object.keys(contract.context),
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({ render: () => h(buttonComponent, contract.context) }))
}

function assertNativeBusyFeedback(html, contract) {
  const openingTag = html.match(/<button\b[^>]*>/)?.[0] || ''
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, `${contract.file}: busy button should be disabled`)
  assert.match(openingTag, /\bloading="true"/, `${contract.file}: busy button should use native loading=true`)
  assert.match(html, new RegExp(contract.busyText), `${contract.file}: busy button should show action text`)
}

test('busy labels replace idle labels for reload, search, refresh, correction and reporting actions', async () => {
  const noop = () => {}
  const contracts = [
    {
      file: 'pages/merchant/location.vue', click: 'loadLocationContext()', busyText: '加载中', idleText: '重新加载',
      busyContext: { loading: true, loadLocationContext: noop }, idleContext: { loading: false, loadLocationContext: noop },
    },
    {
      file: 'pages/search/index.vue', click: 'search', busyText: '搜索中', idleText: '搜索',
      busyContext: { loading: true, search: noop }, idleContext: { loading: false, search: noop },
    },
    {
      file: 'pages/sourcing-map/index.vue', click: 'submitSearch', busyText: '搜索中', idleText: '搜索',
      busyContext: { loading: true, submitSearch: noop }, idleContext: { loading: false, submitSearch: noop },
    },
    {
      file: 'pages/sourcing-map/index.vue', click: 'loadPlaces({ reset: true })', busyText: '加载中', idleText: '重新加载',
      busyContext: { loading: true, loadPlaces: noop }, idleContext: { loading: false, loadPlaces: noop },
    },
    {
      file: 'pages/sourcing-map/legacy-canvas.vue', click: 'refreshCurrentMapData', busyText: '刷新中', idleText: '刷新',
      busyContext: { loading: true, objectLoading: false, refreshCurrentMapData: noop }, idleContext: { loading: false, objectLoading: false, refreshCurrentMapData: noop },
    },
    {
      file: 'pages/sourcing-map/legacy-canvas.vue', click: 'loadScenes', busyText: '加载中', idleText: '重新加载',
      busyContext: { loading: true, loadScenes: noop }, idleContext: { loading: false, loadScenes: noop },
    },
    {
      file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSearch', busyText: '搜索中', idleText: '搜索',
      busyContext: { objectLoading: true, submitSearch: noop }, idleContext: { objectLoading: false, submitSearch: noop },
    },
    {
      file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectLocationCorrection', busyText: '纠错中', idleText: '位置纠错',
      busyContext: { reportSubmitting: true, reportAction: 'location', submitSelectedObjectLocationCorrection: noop }, idleContext: { reportSubmitting: false, reportAction: '', submitSelectedObjectLocationCorrection: noop },
    },
    {
      file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectRiskReport', busyText: '举报中', idleText: '举报问题',
      busyContext: { reportSubmitting: true, reportAction: 'risk', submitSelectedObjectRiskReport: noop }, idleContext: { reportSubmitting: false, reportAction: '', submitSelectedObjectRiskReport: noop },
    },
  ]

  for (const contract of contracts) {
    const busyHtml = await renderBusyButton({ ...contract, context: contract.busyContext })
    const idleHtml = await renderBusyButton({ ...contract, context: contract.idleContext })
    assert.match(busyHtml, new RegExp(contract.busyText), `${contract.file}: busy text should be ${contract.busyText}`)
    assert.match(idleHtml, new RegExp(contract.idleText), `${contract.file}: idle text should be ${contract.idleText}`)
    assert.doesNotMatch(busyHtml, new RegExp(`>${contract.idleText}<`), `${contract.file}: busy button should not retain its idle text`)
  }
})

test('existing foreground request states render native busy button feedback', async () => {
  const noop = () => {}
  const contracts = [
    { file: 'pages/login/index.vue', click: 'loginWithWechatAccount', context: { loggingIn: true, loginWithWechatAccount: noop }, busyText: '登录中' },
    { file: 'pages/account/settings.vue', click: 'confirmDeleteAccount', context: { deleting: true, confirmDeleteAccount: noop }, busyText: '正在注销' },
    { file: 'pages/resource/report.vue', click: 'submitReport', context: { submitting: true, submitReport: noop }, busyText: '提交中' },
    { file: 'pages/vip/index.vue', click: 'openSelectedPlan', context: { purchaseBusy: true, paying: true, selectedPlanCode: 'yearly', openSelectedPlan: noop }, busyText: '正在开通' },
    { file: 'pages/vip/index.vue', click: 'openQuotaPack(item)', context: { purchaseBusy: true, item: { code: 'publish_5', actionText: '购买' }, payingPackCode: 'publish_5', openQuotaPack: noop, packActionText: (item) => item.actionText }, busyText: '购买中' },
    { file: 'pages/merchant/profile.vue', click: 'submitMerchantProfile', context: { submitting: true, submitMerchantProfile: noop, saveButtonText: '保存资料' }, busyText: '保存中' },
    { file: 'pages/merchant/map-binding.vue', click: 'searchCandidates', context: { candidateLoading: true, searchCandidates: noop }, busyText: '搜索中' },
    { file: 'pages/merchant/map-binding.vue', click: 'submitBindingRequest', context: { submitting: true, selectedObjectId: 'object-1', submitBindingRequest: noop }, busyText: '绑定中' },
    { file: 'pages/merchant/location.vue', click: 'loadLocationContext()', context: { loading: true, loadLocationContext: noop }, busyText: '加载中' },
    { file: 'pages/merchant/location.vue', click: 'retryNearby', context: { loading: true, retryNearby: noop }, busyText: '加载中' },
    { file: 'pages/publish/index.vue', click: 'loadPublishCategories', context: { loadingCategories: true, loadPublishCategories: noop }, busyText: '刷新中' },
    { file: 'pages/search/index.vue', click: 'search', context: { loading: true, search: noop }, busyText: '搜索中' },
    { file: 'pages/sourcing-map/index.vue', click: 'submitSearch', context: { loading: true, submitSearch: noop }, busyText: '搜索中' },
    { file: 'pages/sourcing-map/index.vue', click: 'loadPlaces({ reset: true })', context: { loading: true, loadPlaces: noop }, busyText: '加载中' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSearch', context: { objectLoading: true, submitSearch: noop }, busyText: '搜索中' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'refreshCurrentMapData', context: { loading: true, objectLoading: false, refreshCurrentMapData: noop }, busyText: '刷新中' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'loadScenes', context: { loading: true, loadScenes: noop }, busyText: '加载中' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectLocationCorrection', context: { reportSubmitting: true, reportAction: 'location', submitSelectedObjectLocationCorrection: noop }, busyText: '纠错中' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectRiskReport', context: { reportSubmitting: true, reportAction: 'risk', submitSelectedObjectRiskReport: noop }, busyText: '举报中' },
    { file: 'components/ResourceList.vue', click: "emit('load-more')", context: { hasMore: true, loading: true, emit: noop, loadingText: '加载中...', loadMoreText: '加载更多' }, busyText: '加载中' },
  ]

  for (const contract of contracts) {
    assertNativeBusyFeedback(await renderBusyButton(contract), contract)
  }
})

function ref(value) {
  return { value }
}

function reactive(value) {
  return value
}

function computed(getter) {
  return { get value() { return getter() } }
}

function loadPage(relativePath, additions, exports) {
  const { descriptor } = parse(read(relativePath), { filename: relativePath })
  const script = descriptor.scriptSetup.content.replace(/^import\s+[\s\S]*?\s+from\s+['"][^'"]+['"]\s*$/gm, '')
  const sandbox = {
    console,
    Promise,
    setTimeout,
    clearTimeout,
    ref,
    reactive,
    computed,
    onLoad: () => {},
    onShow: () => {},
    onPullDownRefresh: () => {},
    onReachBottom: () => {},
    onReady: () => {},
    onUnmounted: () => {},
    watch: () => {},
    getCurrentInstance: () => null,
    nextTick: () => Promise.resolve(),
    uni: { login: ({ success }) => success({ code: 'wechat-code' }), showToast: () => {}, redirectTo: () => {}, switchTab: () => {} },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.page = { ${exports.join(', ')} }`, sandbox, { filename: relativePath })
  return sandbox.page
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => { resolve = nextResolve; reject = nextReject })
  return { promise, resolve, reject }
}

function flushAsyncWork() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

test('foreground handlers ignore duplicate programmatic requests and recover their busy state', async () => {
  const loginRequest = deferred()
  let loginCalls = 0
  const loginPage = loadPage('pages/login/index.vue', {
    DEFAULT_CITY_CODE: 'huzhou', PRIVACY_POLICY_VERSION: 'v1', USER_AGREEMENT_VERSION: 'v1',
    wechatLogin: () => { loginCalls += 1; return loginRequest.promise },
    getMe: async () => ({}), saveMerchantId: () => {}, saveToken: () => {}, saveUserId: () => {},
  }, ['loggingIn', 'agreedToPolicies', 'loginWithWechatAccount'])
  loginPage.agreedToPolicies.value = true
  const firstLogin = loginPage.loginWithWechatAccount()
  const duplicateLogin = loginPage.loginWithWechatAccount()
  await flushAsyncWork()
  assert.equal(loginCalls, 1, 'duplicate login should not create another server request')
  loginRequest.resolve({ token: '', managedMerchants: [] })
  await Promise.all([firstLogin, duplicateLogin])
  assert.equal(loginPage.loggingIn.value, false, 'login busy state should reset after completion')

  const vipRequest = deferred()
  let vipCalls = 0
  const vipPage = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }),
    createVIPOrder: () => { vipCalls += 1; return vipRequest.promise },
    createQuotaPackOrder: async () => ({}), createVIPPayment: async () => ({ status: 'paid' }),
    listVIPPlans: async () => ({ items: [] }), listQuotaPacks: async () => ({ items: [] }),
  }, ['merchantId', 'paying', 'openSelectedPlan'])
  vipPage.merchantId.value = 'merchant-1'
  const firstVip = vipPage.openSelectedPlan()
  const duplicateVip = vipPage.openSelectedPlan()
  await flushAsyncWork()
  assert.equal(vipCalls, 1, 'duplicate VIP purchase should not create another order')
  vipRequest.resolve({ orderId: 'order-1' })
  await Promise.all([firstVip, duplicateVip])
  assert.equal(vipPage.paying.value, false, 'VIP busy state should reset after completion')

  const profileRequest = deferred()
  let profileCalls = 0
  const profilePage = loadPage('pages/merchant/profile.vue', {
    DEFAULT_CITY_CODE: 'huzhou', MERCHANT_PROFILE_IMAGE_MAX_COUNT: 9,
    validateMerchantName: () => '', getMerchantImagePreviewUrl: () => '', getStoredMerchantImageUrls: () => [],
    getMerchantImageUrlsForPreview: () => [], createStoredMerchantImageEntry: () => ({}),
    appendMerchantImageFiles: () => [], removeMerchantImageEntry: () => [], resolveImageCompressionOptions: () => ({}),
    getMerchantId: () => '', updateMerchant: () => { profileCalls += 1; return profileRequest.promise },
    getMerchant: async () => ({}), bindWechatPhone: async () => ({}), reverseGeocodeLocation: async () => ({}),
    createImageFileFromPath: async () => ({}), uploadSelectedImage: async () => '',
  }, ['merchantId', 'submitting', 'mainCategoriesText', 'form', 'submitMerchantProfile'])
  profilePage.merchantId.value = 'merchant-1'
  profilePage.mainCategoriesText.value = '童装'
  profilePage.form.name = '童装档口'
  const firstProfile = profilePage.submitMerchantProfile()
  const duplicateProfile = profilePage.submitMerchantProfile()
  await flushAsyncWork()
  assert.equal(profileCalls, 1, 'duplicate profile save should not create another update')
  profileRequest.resolve({})
  await Promise.all([firstProfile, duplicateProfile])
  assert.equal(profilePage.submitting.value, false, 'profile busy state should reset after completion')

  const candidateRequest = deferred()
  let candidateCalls = 0
  const bindingPage = loadPage('pages/merchant/map-binding.vue', {
    DEFAULT_CITY_CODE: 'huzhou', getMerchantId: () => '', ensureMerchantProfileReady: async () => true,
    getMerchantMapBinding: async () => ({}), listMapScenes: async () => ({ items: [] }),
    listMapBindCandidates: () => { candidateCalls += 1; return candidateRequest.promise },
    submitMapBindRequest: async () => ({}),
  }, ['merchantId', 'selectedObjectId', 'candidateLoading', 'submitting', 'searchCandidates', 'submitBindingRequest'])
  bindingPage.merchantId.value = 'merchant-1'
  const firstCandidateSearch = bindingPage.searchCandidates()
  const duplicateCandidateSearch = bindingPage.searchCandidates()
  await flushAsyncWork()
  assert.equal(candidateCalls, 1, 'duplicate candidate search should not create another server request')
  candidateRequest.resolve({ items: [] })
  await Promise.all([firstCandidateSearch, duplicateCandidateSearch])
  assert.equal(bindingPage.candidateLoading.value, false, 'candidate search busy state should reset after completion')

  const bindingRequest = deferred()
  let bindingCalls = 0
  const submitPage = loadPage('pages/merchant/map-binding.vue', {
    DEFAULT_CITY_CODE: 'huzhou', getMerchantId: () => '', ensureMerchantProfileReady: async () => true,
    getMerchantMapBinding: async () => ({}), listMapScenes: async () => ({ items: [] }), listMapBindCandidates: async () => ({ items: [] }),
    submitMapBindRequest: () => { bindingCalls += 1; return bindingRequest.promise },
  }, ['merchantId', 'selectedObjectId', 'submitting', 'submitBindingRequest'])
  submitPage.merchantId.value = 'merchant-1'
  submitPage.selectedObjectId.value = 'object-1'
  const firstBindingSubmit = submitPage.submitBindingRequest()
  const duplicateBindingSubmit = submitPage.submitBindingRequest()
  await flushAsyncWork()
  assert.equal(bindingCalls, 1, 'duplicate binding submit should not create another request')
  bindingRequest.resolve({})
  await Promise.all([firstBindingSubmit, duplicateBindingSubmit])
  assert.equal(submitPage.submitting.value, false, 'binding busy state should reset after completion')
})

test('VIP failures, payment cancellation and quota-pack duplicates restore their request states', async () => {
  const toasts = []
  const failedPlanPage = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }),
    createVIPOrder: async () => { throw new Error('支付服务暂不可用') },
    createQuotaPackOrder: async () => ({}), createVIPPayment: async () => ({ status: 'paid' }),
    listVIPPlans: async () => ({ items: [] }), listQuotaPacks: async () => ({ items: [] }),
    uni: { showToast: (options) => toasts.push(options), requestPayment: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['merchantId', 'paying', 'openSelectedPlan'])
  failedPlanPage.merchantId.value = 'merchant-1'
  await failedPlanPage.openSelectedPlan()
  assert.equal(failedPlanPage.paying.value, false, 'failed VIP order should clear paying')
  assert.equal(toasts.at(-1)?.title, '支付服务暂不可用', 'failed VIP order should preserve the friendly error')

  const quotaOrder = deferred()
  let quotaCalls = 0
  const quotaToasts = []
  const quotaPage = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }), createVIPOrder: async () => ({}),
    createQuotaPackOrder: () => { quotaCalls += 1; return quotaOrder.promise },
    createVIPPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'n', package: 'p', paySign: 's' } }),
    listVIPPlans: async () => ({ items: [] }), listQuotaPacks: async () => ({ items: [] }),
    uni: {
      showToast: (options) => quotaToasts.push(options), redirectTo: () => {}, switchTab: () => {},
      requestPayment: ({ fail }) => fail(new Error('用户取消支付')),
    },
  }, ['merchantId', 'payingPackCode', 'openQuotaPack'])
  quotaPage.merchantId.value = 'merchant-1'
  const pack = { code: 'publish_5' }
  const firstQuota = quotaPage.openQuotaPack(pack)
  const duplicateQuota = quotaPage.openQuotaPack(pack)
  await flushAsyncWork()
  assert.equal(quotaCalls, 1, 'duplicate quota purchase should not create another order')
  quotaOrder.resolve({ orderId: 'quota-order-1' })
  await Promise.all([firstQuota, duplicateQuota])
  assert.equal(quotaPage.payingPackCode.value, '', 'cancelled quota payment should clear its pack loading state')
  assert.equal(quotaToasts.at(-1)?.title, '用户取消支付', 'cancelled quota payment should preserve the friendly error')
})

test('paid VIP orders refresh benefits, keep the success feedback and clear paying', async () => {
  const toasts = []
  let planRefreshes = 0
  let quotaPackRefreshes = 0
  const page = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }),
    createVIPOrder: async () => ({ orderId: 'paid-order-1' }), createQuotaPackOrder: async () => ({}),
    createVIPPayment: async () => ({ status: 'paid' }),
    listVIPPlans: async () => { planRefreshes += 1; return { items: [] } },
    listQuotaPacks: async () => { quotaPackRefreshes += 1; return { items: [] } },
    uni: { showToast: (options) => toasts.push(options), requestPayment: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['merchantId', 'paying', 'openSelectedPlan'])
  page.merchantId.value = 'merchant-1'

  await page.openSelectedPlan()

  assert.equal(planRefreshes, 1, 'paid VIP order should refresh VIP plans')
  assert.equal(quotaPackRefreshes, 1, 'paid VIP order should refresh quota packs')
  assert.equal(toasts.at(-1)?.title, '支付已完成，权益到账后会自动更新', 'paid VIP order should keep the success feedback')
  assert.equal(page.paying.value, false, 'paid VIP order should clear paying')
})

test('VIP and quota-pack purchases share one lock while only the active action spins', async () => {
  const vipOrder = deferred()
  let vipOrderCalls = 0
  let packOrderCalls = 0
  const vipPage = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }),
    createVIPOrder: () => { vipOrderCalls += 1; return vipOrder.promise },
    createQuotaPackOrder: async () => { packOrderCalls += 1; return { orderId: 'pack-order' } },
    createVIPPayment: async () => ({ status: 'paid' }),
    listVIPPlans: async () => ({ items: [] }), listQuotaPacks: async () => ({ items: [] }),
  }, ['merchantId', 'selectedPlanCode', 'activeTab', 'paying', 'payingPackCode', 'purchaseBusy', 'selectPlan', 'switchTab', 'openSelectedPlan', 'openQuotaPack'])
  vipPage.merchantId.value = 'merchant-1'

  const buyingVip = vipPage.openSelectedPlan()
  vipPage.openQuotaPack({ code: 'publish_5' })
  vipPage.openSelectedPlan()
  vipPage.selectPlan('monthly')
  vipPage.switchTab('addons')
  await flushAsyncWork()
  assert.equal(vipOrderCalls, 1, 'VIP 购买中同类连续点击只应创建一个订单')
  assert.equal(packOrderCalls, 0, 'VIP 购买中不得并发创建次数包订单')
  assert.equal(vipPage.purchaseBusy.value, true, 'VIP 购买中应持有共享购买锁')
  assert.equal(vipPage.selectedPlanCode.value, 'yearly', '购买中不得切换 VIP 套餐')
  assert.equal(vipPage.activeTab.value, 'vip', '购买中不得切换 tab')

  const vipButtonHtml = await renderBusyButton({
    file: 'pages/vip/index.vue', click: 'openSelectedPlan',
    context: { purchaseBusy: true, paying: true, selectedPlanCode: 'yearly', openSelectedPlan: () => {} },
  })
  const lockedPackHtml = await renderBusyButton({
    file: 'pages/vip/index.vue', click: 'openQuotaPack(item)',
    context: { purchaseBusy: true, item: { code: 'publish_5', actionText: '购买' }, payingPackCode: '', openQuotaPack: () => {}, packActionText: (item) => item.actionText },
  })
  assert.match(vipButtonHtml, /loading="true"/, '当前 VIP 动作应显示 spinner')
  assert.match(lockedPackHtml, /disabled(?:=|\s|>)/, '非当前次数包按钮也应锁定')
  assert.match(lockedPackHtml, /loading="false"/, '非当前次数包按钮不应显示 VIP spinner')

  vipOrder.resolve({ orderId: 'vip-order' })
  await buyingVip
  assert.equal(vipPage.purchaseBusy.value, false, 'VIP 链路完成后应释放共享锁')

  const packOrder = deferred()
  vipOrderCalls = 0
  packOrderCalls = 0
  const packPage = loadPage('pages/vip/index.vue', {
    requireLogin: () => true, getSession: () => ({ merchantId: '' }),
    createVIPOrder: async () => { vipOrderCalls += 1; return { orderId: 'vip-order' } },
    createQuotaPackOrder: () => { packOrderCalls += 1; return packOrder.promise },
    createVIPPayment: async () => ({ status: 'paid' }),
    listVIPPlans: async () => ({ items: [] }), listQuotaPacks: async () => ({ items: [] }),
  }, ['merchantId', 'paying', 'payingPackCode', 'purchaseBusy', 'openSelectedPlan', 'openQuotaPack'])
  packPage.merchantId.value = 'merchant-1'

  const buyingPack = packPage.openQuotaPack({ code: 'publish_5' })
  packPage.openQuotaPack({ code: 'refresh_10' })
  packPage.openSelectedPlan()
  await flushAsyncWork()
  assert.equal(packOrderCalls, 1, '次数包购买中同类不同套餐只应创建一个订单')
  assert.equal(vipOrderCalls, 0, '次数包购买中不得并发创建 VIP 订单')
  assert.equal(packPage.purchaseBusy.value, true, '次数包购买中应持有共享购买锁')

  const lockedVipHtml = await renderBusyButton({
    file: 'pages/vip/index.vue', click: 'openSelectedPlan',
    context: { purchaseBusy: true, paying: false, selectedPlanCode: 'yearly', openSelectedPlan: () => {} },
  })
  const currentPackHtml = await renderBusyButton({
    file: 'pages/vip/index.vue', click: 'openQuotaPack(item)',
    context: { purchaseBusy: true, item: { code: 'publish_5', actionText: '购买' }, payingPackCode: 'publish_5', openQuotaPack: () => {}, packActionText: (item) => item.actionText },
  })
  assert.match(lockedVipHtml, /disabled(?:=|\s|>)/, '次数包购买中 VIP 按钮也应锁定')
  assert.match(lockedVipHtml, /loading="false"/, '非当前 VIP 按钮不应显示次数包 spinner')
  assert.match(currentPackHtml, /loading="true"/, '当前次数包动作应显示 spinner')

  packOrder.resolve({ orderId: 'pack-order' })
  await buyingPack
  assert.equal(packPage.purchaseBusy.value, false, '次数包链路完成后应释放共享锁')
})

test('map request entry points reject duplicate programmatic loading', async () => {
  const placesRequest = deferred()
  let placeCalls = 0
  const directoryPage = loadPage('pages/sourcing-map/index.vue', {
    DEFAULT_CITY_CODE: 'zhili', requireLogin: () => true, getMerchantId: () => '', trackMerchantMapEvent: () => {},
    buildMerchantTagLabels: () => () => [], buildMerchantPlaceQuery: (query) => query,
    hasMerchantDetail: () => false, hasValidLocation: () => false, merchantDetailPath: () => '', normalizeMerchantPlace: (item) => item,
    listMapCategories: async () => ({ items: [] }), submitMapLocationCorrection: async () => {},
    listMerchantPlaces: () => { placeCalls += 1; return placesRequest.promise },
    uni: { showModal: () => {}, showToast: () => {}, navigateTo: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['loading', 'loadPlaces', 'submitSearch'])
  const firstPlaces = directoryPage.loadPlaces({ reset: true })
  const duplicatePlaces = directoryPage.submitSearch()
  await flushAsyncWork()
  assert.equal(placeCalls, 1, 'duplicate directory reset should not create another request')
  placesRequest.resolve({ items: [], total: 0 })
  await Promise.all([firstPlaces, duplicatePlaces])
  assert.equal(directoryPage.loading.value, false, 'directory loading should clear after the request finishes')

  const scenesRequest = deferred()
  let sceneCalls = 0
  const legacyScenePage = loadPage('pages/sourcing-map/legacy-canvas.vue', {
    DEFAULT_CITY_CODE: 'zhili', requireLogin: () => true,
    listMapScenes: () => { sceneCalls += 1; return scenesRequest.promise },
    listMapCategories: async () => ({ items: [] }), listMapObjects: async () => ({ items: [], total: 0 }), searchMapObjects: async () => ({ items: [], total: 0 }),
    listNearbyPois: async () => ({ items: [] }), getMapObject: async () => ({}), submitMapLocationCorrection: async () => ({}), submitMapRiskReport: async () => ({}),
    uni: { showToast: () => {}, showActionSheet: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['loading', 'loadScenes'])
  const firstScenes = legacyScenePage.loadScenes()
  const duplicateScenes = legacyScenePage.loadScenes()
  await flushAsyncWork()
  assert.equal(sceneCalls, 1, 'duplicate scene loading should not create another request')
  scenesRequest.resolve({ items: [] })
  await Promise.all([firstScenes, duplicateScenes])
  assert.equal(legacyScenePage.loading.value, false, 'scene loading should clear after the request finishes')

  const objectsRequest = deferred()
  let objectCalls = 0
  const legacyObjectPage = loadPage('pages/sourcing-map/legacy-canvas.vue', {
    DEFAULT_CITY_CODE: 'zhili', requireLogin: () => true, listMapScenes: async () => ({ items: [] }), listMapCategories: async () => ({ items: [] }),
    listMapObjects: () => { objectCalls += 1; return objectsRequest.promise }, searchMapObjects: async () => ({ items: [], total: 0 }),
    listNearbyPois: async () => ({ items: [] }), getMapObject: async () => ({}), submitMapLocationCorrection: async () => ({}), submitMapRiskReport: async () => ({}),
    uni: { showToast: () => {}, showActionSheet: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['selectedSceneCode', 'objectLoading', 'loadSceneObjects'])
  legacyObjectPage.selectedSceneCode.value = 'scene-1'
  const firstObjects = legacyObjectPage.loadSceneObjects()
  const duplicateObjects = legacyObjectPage.loadSceneObjects()
  await flushAsyncWork()
  assert.equal(objectCalls, 1, 'duplicate visible object loading should not create another request')
  objectsRequest.resolve({ items: [], total: 0 })
  await Promise.all([firstObjects, duplicateObjects])
  assert.equal(legacyObjectPage.objectLoading.value, false, 'object loading should clear after the request finishes')
})

test('legacy canvas condition changes start a latest-wins object request', async () => {
  const firstRequest = deferred()
  const latestRequest = deferred()
  const queries = []
  const page = loadPage('pages/sourcing-map/legacy-canvas.vue', {
    DEFAULT_CITY_CODE: 'zhili', requireLogin: () => true,
    listMapScenes: async () => ({ items: [] }), listMapCategories: async () => ({ items: [] }),
    listMapObjects: async () => ({ items: [], total: 0 }),
    searchMapObjects: (query) => {
      queries.push(structuredClone(query))
      return queries.length === 1 ? firstRequest.promise : latestRequest.promise
    },
    listNearbyPois: async () => ({ items: [] }), getMapObject: async () => ({}),
    submitMapLocationCorrection: async () => ({}), submitMapRiskReport: async () => ({}),
    uni: { showToast: () => {}, showActionSheet: () => {}, redirectTo: () => {}, switchTab: () => {} },
  }, ['selectedSceneCode', 'keyword', 'activeFilters', 'rawMapObjects', 'objectLoading', 'submitSearch', 'toggleFilter'])
  page.selectedSceneCode.value = 'scene-1'
  page.keyword.value = '童装'

  const firstLoad = page.submitSearch()
  page.submitSearch()
  await flushAsyncWork()
  assert.equal(queries.length, 1, '相同显式搜索在请求中仍应防重复')

  const latestLoad = page.toggleFilter('categories', 'girl')
  await flushAsyncWork()
  assert.equal(queries.length, 2, '搜索中的筛选变化必须启动新的对象请求')
  assert.equal(queries[1].categories, 'girl')
  assert.equal(page.objectLoading.value, true, '最新对象请求完成前应持续显示 loading')

  latestRequest.resolve({ items: [{ id: 'latest' }], total: 1 })
  await latestLoad
  firstRequest.resolve({ items: [{ id: 'old' }], total: 1 })
  await firstLoad
  assert.deepEqual(page.rawMapObjects.value.map((item) => item.id), ['latest'], '迟到的旧响应不得覆盖最新筛选结果')
  assert.equal(page.objectLoading.value, false, '最新请求完成后应释放 loading')
})
