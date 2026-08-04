import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

import * as merchantPlaceState from '../sourcing-map/merchantPlaceState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/merchant/detail.vue')

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

function createTrackedRefFactory() {
  const writes = new WeakMap()
  return {
    ref(initialValue) {
      let currentValue = initialValue
      const state = {}
      writes.set(state, [])
      Object.defineProperty(state, 'value', {
        get() {
          return currentValue
        },
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

async function renderMerchantDetailTemplate(context) {
  const source = fs.readFileSync(sourcePath, 'utf8')
  const { descriptor } = parse(source, { filename: 'pages/merchant/detail.vue' })
  const compiled = compileTemplate({
    source: descriptor.template.content,
    filename: 'pages/merchant/detail.vue',
    id: 'merchant-detail-follow-loading',
    compilerOptions: {
      mode: 'function',
      isCustomElement: (tag) => ['view', 'text', 'image', 'scroll-view', 'button', 'ResourceList'].includes(tag),
    },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const Page = {
    props: Object.keys(context),
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({ render: () => h(Page, context) }))
}

function loadMerchantDetailPage(additions = {}) {
  const { trackMerchantMapEvent = () => {}, timeline = [] } = additions
  const source = fs.readFileSync(sourcePath, 'utf8')
  const script = source
    .match(/<script setup>([\s\S]*?)<\/script>/)?.[1]
    .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '') || ''
  const sandbox = {
    buildMerchantAddressLocation: merchantPlaceState.buildMerchantAddressLocation,
    computed(getter) {
      return { get value() { return getter() } }
    },
    getMerchant: async () => ({}),
    getMerchantFollowState: async () => ({ followed: false }),
    getSession: () => ({ merchantId: '', token: '' }),
    listResources: async () => ({ items: [], total: 0 }),
    onLoad() {},
    onReachBottom() {},
    ref(value) {
      return { value }
    },
    setMerchantFollow: async () => ({ followed: false }),
    trackMerchantMapEvent,
    uni: {
      navigateTo(options) {
        timeline.push(['navigateTo', options.url])
      },
      showToast() {},
    },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.merchantDetailPage = { merchant, ownMerchantId, followed, followBusy: typeof followBusy === 'undefined' ? undefined : followBusy, isOwnMerchant, toggleFollow, openMerchantLocation }`, sandbox)
  return sandbox.merchantDetailPage
}

test('merchant detail page does not show verification wording', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, />认证</)
  assert.doesNotMatch(source, /已认证/)
  assert.doesNotMatch(source, /已核实/)
  assert.doesNotMatch(source, /核验项/)
  assert.doesNotMatch(source, /主体资质、经营场地/)
  assert.doesNotMatch(source, /有效期/)
  assert.doesNotMatch(source, /showVerificationInfo/)
  assert.doesNotMatch(source, /merchantVerification/)
  assert.doesNotMatch(source, /verification-info/)
  assert.doesNotMatch(source, /formatDateToDay/)
  assert.doesNotMatch(source, /licenseUrl/)
  assert.doesNotMatch(source, /socialCreditCode/)
  assert.doesNotMatch(source, /businessName/)
})

test('merchant detail page keeps full resources at the bottom without overview card', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /class="supply-overview-panel"/)
  assert.doesNotMatch(source, />供应概览</)
  assert.doesNotMatch(source, /scrollToResourceList/)
  assert.doesNotMatch(source, /uni\.pageScrollTo/)
  assert.match(source, /label: '公开供应'/)
  assert.doesNotMatch(source, /label: '在售供应'/)
  assert.match(source, /class="section resource-list-section"/)
  assert.match(source, />公开供应</)
  assert.doesNotMatch(source, />全部公开供应</)
  assert.match(source, /<ResourceList[\s\S]*exposure-source="merchant"[\s\S]*variant="feed"[\s\S]*:resources="merchantResources"/)
  assert.match(source, /:empty-text="merchantResourcesEmptyText"/)

  const profileIndex = source.indexOf('class="profile-panel"')
  const trustNoteIndex = source.indexOf('class="section trust-note-section"')
  const resourceListIndex = source.indexOf('class="section resource-list-section"')

  assert.ok(profileIndex > -1)
  assert.ok(trustNoteIndex > profileIndex)
  assert.ok(resourceListIndex > trustNoteIndex)
})

test('merchant detail page renders address as a lightweight single-address block', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /<view class="section merchant-address-section" v-if="merchantAddressLocation">/)
  assert.doesNotMatch(source, /merchant-address-card/)
  assert.doesNotMatch(source, /merchant-address-head/)
  assert.doesNotMatch(source, />经营地址</)
  assert.doesNotMatch(source, /merchant-address-title/)
  assert.match(source, /<view class="section-head">[\s\S]*<text class="section-title">地址<\/text>[\s\S]*<button/)
  assert.match(source, /<button v-if="merchantAddressLocation\.hasGps" class="address-action" @click="openMerchantLocation">查看位置<\/button>/)
  assert.match(source, /<button v-else class="address-action secondary" @click="copyMerchantAddress\(\)">复制<\/button>/)
  assert.match(source, /<text class="merchant-address-text">\{\{ merchantAddressLocation\.address \}\}<\/text>/)
  assert.doesNotMatch(source, /<map|merchant-address-map/)
  assert.match(source, /import \{ buildMerchantAddressLocation \} from '\.\.\/sourcing-map\/merchantPlaceState'/)
  assert.match(source, /const merchantAddressLocation = computed\(\(\) => buildMerchantAddressLocation\(merchant\.value\)\)/)
  assert.doesNotMatch(source, /function buildMerchantAddressLocation\(/)
  assert.match(source, /function openMerchantLocation\(\) \{[\s\S]*merchant\.value\.id[\s\S]*merchantAddressLocation\.value\?\.hasGps[\s\S]*\/pages\/merchant\/location\?merchantId=\$\{encodeURIComponent\(merchant\.value\.id\)\}/)
  assert.doesNotMatch(source, /uni\.openLocation/)
  assert.match(source, /function copyMerchantAddress\(title = '地址已复制'\) \{[\s\S]*uni\.setClipboardData\(\{ data: location\.address \}\)/)
  assert.match(source, /\.merchant-address-section \{[\s\S]*display: grid;[\s\S]*gap: 14rpx;/)
  assert.match(source, /\.merchant-address-section \.section-head \{[\s\S]*margin-bottom: 0;/)
  assert.match(source, /\.merchant-address-text \{[\s\S]*display: block;[\s\S]*font-size: 30rpx;[\s\S]*line-height: 1\.55;[\s\S]*word-break: break-word;/)
})

test('merchant detail address builder rejects blank, non-finite, and out-of-range coordinates', () => {
  assert.equal(typeof merchantPlaceState.buildMerchantAddressLocation, 'function')
  const buildMerchantAddressLocation = merchantPlaceState.buildMerchantAddressLocation
  const cases = [
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: '30.89912', lng: '120.20482' } },
      expected: { address: '织里商城 A-101', hasGps: true, latitude: 30.89912, longitude: 120.20482 },
    },
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: '   ', lng: '120.20482' } },
      expected: { address: '织里商城 A-101', hasGps: false },
    },
    { merchant: { location: { lat: '\t', lng: '120.20482' } }, expected: null },
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: Number.NaN, lng: '120.20482' } },
      expected: { address: '织里商城 A-101', hasGps: false },
    },
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: Number.POSITIVE_INFINITY, lng: '120.20482' } },
      expected: { address: '织里商城 A-101', hasGps: false },
    },
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: '91', lng: '120.20482' } },
      expected: { address: '织里商城 A-101', hasGps: false },
    },
    {
      merchant: { addressText: '织里商城 A-101', location: { lat: '30.89912', lng: '-181' } },
      expected: { address: '织里商城 A-101', hasGps: false },
    },
  ]

  for (const { merchant, expected } of cases) {
    assert.deepEqual(buildMerchantAddressLocation(merchant), expected)
  }
})

test('merchant detail page only directs users to contact details in published supply and demand', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /merchant\.contact/)
  assert.match(source, /联系方式仅随有效供需信息展示/)
})

test('merchant detail records its valid location entry immediately before navigation', () => {
  const timeline = []
  const page = loadMerchantDetailPage({
    trackMerchantMapEvent(event) {
      timeline.push(['track', event])
      return new Promise(() => {})
    },
    timeline,
  })
  page.merchant.value = {
    id: 'merchant-detail',
    addressText: '织里商城 A-101',
    location: { lat: '30.89912', lng: '120.20482' },
  }

  page.openMerchantLocation()

  assert.deepEqual(plain(timeline), [
    ['track', {
      merchantId: 'merchant-detail',
      eventType: 'location_entry_click',
      source: 'merchant_detail',
    }],
    ['navigateTo', '/pages/merchant/location?merchantId=merchant-detail'],
  ])

  timeline.length = 0
  page.merchant.value.location.lat = ''
  page.openMerchantLocation()
  assert.deepEqual(timeline, [])
})

test('merchant follow action shows native loading, rejects duplicates, and restores feedback state', async () => {
  const busyHtml = await renderMerchantDetailTemplate({
    merchantLogo: '', merchantInitial: '商', merchant: { name: '示例商家' }, merchantSubtitle: '',
    isOwnMerchant: false, followed: false, followBusy: true, toggleFollow: () => {}, statCards: [],
    merchantCategoryTags: [], profileDescription: '', merchantImages: [], merchantAddressLocation: null,
    merchantResourceCountText: '', merchantResources: [], merchantResourcesEmptyText: '',
    merchantResourcesLoading: false, hasMoreMerchantResources: false, openMerchantEditor: () => {},
    previewMerchantImage: () => {}, openMerchantLocation: () => {}, copyMerchantAddress: () => {},
    openResource: () => {}, loadMerchantResources: () => {},
  })
  const followButton = busyHtml.match(/<button[^>]*>(?:关注|已关注|处理中)<\/button>/)?.[0] || ''
  assert.match(followButton, /loading="true"/, '关注请求期间按钮应显示原生 loading')
  assert.match(followButton, /disabled(?:=|\s|>)/, '关注请求期间按钮应禁用')

  const request = deferred()
  const calls = []
  const toasts = []
  const page = loadMerchantDetailPage({
    setMerchantFollow(merchantId, nextFollowed) {
      calls.push({ merchantId, nextFollowed })
      return request.promise
    },
    uni: { showToast: (options) => toasts.push(options) },
  })
  page.merchant.value = { id: 'merchant-expected' }
  const first = page.toggleFollow()
  const duplicate = page.toggleFollow()
  assert.deepEqual(calls, [{ merchantId: 'merchant-expected', nextFollowed: true }], '连续点击只能提交一次关注请求')
  assert.equal(page.followBusy.value, true, '请求未结束时应保持 busy')
  request.resolve({ followed: true })
  await Promise.all([first, duplicate])
  assert.equal(page.followBusy.value, false, '关注成功后应恢复 busy')
  assert.equal(page.followed.value, true, '成功后应采用服务端关注状态')
  assert.equal(toasts.at(-1)?.title, '已关注', '成功提示应保持')

  const failureToasts = []
  const failedPage = loadMerchantDetailPage({
    setMerchantFollow: async () => { throw new Error('关注服务暂不可用') },
    uni: { showToast: (options) => failureToasts.push(options) },
  })
  failedPage.merchant.value = { id: 'merchant-expected' }
  await failedPage.toggleFollow()
  assert.equal(failedPage.followBusy.value, false, '关注失败后应恢复 busy')
  assert.equal(failureToasts.at(-1)?.title, '关注服务暂不可用', '失败提示应保持')
})

test('merchant follow local branches never enter busy state', async () => {
  const tracker = createTrackedRefFactory()
  let requests = 0
  const page = loadMerchantDetailPage({
    ref: tracker.ref,
    setMerchantFollow: async () => { requests += 1; return { followed: true } },
    getSession: () => ({ merchantId: 'merchant-own', token: 'token' }),
  })
  await page.toggleFollow()
  assert.equal(page.followBusy.value, false, '缺少商家时不应进入 busy')
  assert.deepEqual(tracker.writesFor(page.followBusy), [], '缺少商家时不应短暂写入 busy')
  page.merchant.value = { id: 'merchant-own' }
  page.ownMerchantId.value = 'merchant-own'
  await page.toggleFollow()
  assert.equal(page.followBusy.value, false, '自己的商家不应进入 busy')
  assert.deepEqual(tracker.writesFor(page.followBusy), [], '自己的商家时不应短暂写入 busy')
  assert.equal(requests, 0, '本地分支不应请求关注接口')
})
