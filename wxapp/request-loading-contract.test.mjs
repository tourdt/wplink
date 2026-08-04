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
  assert.match(openingTag, /\bloading(?:=|\s|>)/, `${contract.file}: busy button should use native loading`)
  assert.match(html, new RegExp(contract.busyText), `${contract.file}: busy button should show action text`)
}

test('existing foreground request states render native busy button feedback', async () => {
  const noop = () => {}
  const contracts = [
    { file: 'pages/login/index.vue', click: 'loginWithWechatAccount', context: { loggingIn: true, loginWithWechatAccount: noop }, busyText: '登录中' },
    { file: 'pages/account/settings.vue', click: 'confirmDeleteAccount', context: { deleting: true, confirmDeleteAccount: noop }, busyText: '正在注销' },
    { file: 'pages/resource/report.vue', click: 'submitReport', context: { submitting: true, submitReport: noop }, busyText: '提交中' },
    { file: 'pages/vip/index.vue', click: 'openSelectedPlan', context: { paying: true, selectedPlanCode: 'yearly', openSelectedPlan: noop }, busyText: '正在开通' },
    { file: 'pages/vip/index.vue', click: 'openQuotaPack(item)', context: { item: { code: 'publish_5', actionText: '购买' }, payingPackCode: 'publish_5', openQuotaPack: noop, packActionText: (item) => item.actionText }, busyText: '购买中' },
    { file: 'pages/merchant/profile.vue', click: 'submitMerchantProfile', context: { submitting: true, submitMerchantProfile: noop, saveButtonText: '保存资料' }, busyText: '保存中' },
    { file: 'pages/merchant/map-binding.vue', click: 'searchCandidates', context: { candidateLoading: true, searchCandidates: noop }, busyText: '搜索中' },
    { file: 'pages/merchant/map-binding.vue', click: 'submitBindingRequest', context: { submitting: true, selectedObjectId: 'object-1', submitBindingRequest: noop }, busyText: '绑定中' },
    { file: 'pages/merchant/location.vue', click: 'loadLocationContext()', context: { loading: true, loadLocationContext: noop }, busyText: '重新加载' },
    { file: 'pages/merchant/location.vue', click: 'retryNearby', context: { loading: true, retryNearby: noop }, busyText: '加载中' },
    { file: 'pages/publish/index.vue', click: 'loadPublishCategories', context: { loadingCategories: true, loadPublishCategories: noop }, busyText: '刷新中' },
    { file: 'pages/search/index.vue', click: 'search', context: { loading: true, search: noop }, busyText: '搜索' },
    { file: 'pages/sourcing-map/index.vue', click: 'submitSearch', context: { loading: true, submitSearch: noop }, busyText: '搜索' },
    { file: 'pages/sourcing-map/index.vue', click: 'loadPlaces({ reset: true })', context: { loading: true, loadPlaces: noop }, busyText: '重新加载' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSearch', context: { objectLoading: true, submitSearch: noop }, busyText: '搜索' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'refreshCurrentMapData', context: { loading: true, objectLoading: false, refreshCurrentMapData: noop }, busyText: '刷新' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'loadScenes', context: { loading: true, loadScenes: noop }, busyText: '重新加载' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectLocationCorrection', context: { reportSubmitting: true, reportAction: 'location', submitSelectedObjectLocationCorrection: noop }, busyText: '位置纠错' },
    { file: 'pages/sourcing-map/legacy-canvas.vue', click: 'submitSelectedObjectRiskReport', context: { reportSubmitting: true, reportAction: 'risk', submitSelectedObjectRiskReport: noop }, busyText: '举报问题' },
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
  const promise = new Promise((next) => { resolve = next })
  return { promise, resolve }
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
