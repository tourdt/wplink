import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import * as merchantPlaceState from '../sourcing-map/merchantPlaceState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/merchant/detail.vue')

function plain(value) {
  return JSON.parse(JSON.stringify(value))
}

function loadMerchantDetailPage({ trackMerchantMapEvent, timeline }) {
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
    },
  }
  vm.runInNewContext(`${script}\nglobalThis.merchantDetailPage = { merchant, openMerchantLocation }`, sandbox)
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
