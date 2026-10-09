import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary } from '../../common/merchantBusinessProfile.js'
import { getPublicContact, getStorefrontPlace } from '../../common/merchantStorefront.js'

const source = fs.readFileSync(path.join(path.dirname(new URL(import.meta.url).pathname), 'detail.vue'), 'utf8')

function page({ session = { merchantId: '' }, getMerchant = async () => ({ id: 'm', name: '童装店', ownerVisible: true }) } = {}) {
  const calls = []
  const hooks = {}
  const sandbox = {
    getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary,
    ref: (value) => ({ value }),
    computed: (getter) => ({ get value() { return getter() } }),
    onLoad: (fn) => { hooks.load = fn },
    onShow: (fn) => { hooks.show = fn },
    onShareAppMessage: (fn) => { hooks.share = fn },
    onShareTimeline: (fn) => { hooks.timeline = fn },
    getMerchant, getSession: () => session, getPublicContact, getStorefrontPlace,
    uni: {
      openLocation: (options) => calls.push(['location', options]),
      showActionSheet: (options) => calls.push(['sheet', options]),
      makePhoneCall: (options) => calls.push(['phone', options]),
      setClipboardData: (options) => calls.push(['copy', options]),
      showToast: (options) => calls.push(['toast', options]),
      navigateTo: (options) => calls.push(['navigate', options.url]),
      switchTab: (options) => calls.push(['tab', options.url]),
      previewImage: () => {},
    },
  }
  const script = source.match(/<script setup>([\s\S]*?)<\/script>/)[1].replace(/^import .*$/gm, '')
  vm.runInNewContext(`${script}\nglobalThis.p = { loadMerchant, openNavigation, openContact, callPhone, copyWechat, openEditor, loadState, merchant, isOwner, canShare, publicContact, place, mapMarkers, businessProfile, categoryText, businessRoleText }`, sandbox)
  return { ...sandbox.p, calls, hooks }
}

test('storefront contains real data sections, map, public contact actions and no resource or follow requests', () => {
  assert.match(source, /merchantImages/)
  assert.match(source, /getBusinessProfileSummary\(merchant\.value\)/)
  assert.match(source, /merchant\.description/)
  assert.match(source, /<map v-if="place\.canNavigate"/)
  assert.match(source, /@click="openNavigation"/)
  assert.match(source, /@click="openContact"/)
  assert.match(source, /class="bottom-bar"[\s\S]*@click="openNavigation"/)
  assert.match(source, /v-if="place\.canNavigate" class="bar-primary"/)
  assert.match(source, /iconPath: '\/static\/map\/marker-selected\.png'/)
  assert.match(source, /onShareAppMessage\(sharePayload\)/)
  assert.doesNotMatch(source, /listResources|setMerchantFollow|getMerchantFollowState|ResourceList/)
})

test('visitor opens without login and only available valid coordinates enable native navigation', async () => {
  const detail = { id: 'm', name: '童装店', ownerVisible: true, mapVisible: true, addressText: '织里商城 A-18', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1 }, contact: { phone: '13800138000' } }
  const visitor = page({ getMerchant: async () => detail })
  await visitor.hooks.load({ id: 'm' })
  assert.equal(visitor.loadState.value, 'ready')
  assert.equal(visitor.isOwner.value, false)
  visitor.openNavigation()
  visitor.openContact()
  assert.equal(visitor.calls[0][0], 'location')
  assert.equal(visitor.calls[0][1].latitude, 30.8)
  assert.deepEqual(Array.from(visitor.calls[1][1].itemList), ['拨打电话'])
  visitor.calls[1][1].success({ tapIndex: 0 })
  assert.deepEqual(JSON.parse(JSON.stringify(visitor.calls[2])), ['phone', { phoneNumber: '13800138000' }])
  assert.equal(visitor.hooks.share().path, '/pages/merchant/detail?id=m')

  const hidden = page({ getMerchant: async () => ({ ...detail, location: { ...detail.location, available: false } }) })
  await hidden.hooks.load({ id: 'm' })
  hidden.openNavigation()
  assert.equal(hidden.calls.length, 0)
})

test('storefront accepts merchantId links from profile preview and share', async () => {
  const requested = []
  const visitor = page({ getMerchant: async (id) => { requested.push(id); return { id, name: '童装店', ownerVisible: true } } })
  await visitor.hooks.load({ merchantId: 'profile-merchant' })
  assert.deepEqual(requested, ['profile-merchant'])
  assert.equal(visitor.loadState.value, 'ready')
  assert.equal(visitor.merchant.value.id, 'profile-merchant')
})

test('paused owner can preview but cannot share and sees only public contact', async () => {
  const owner = page({ session: { merchantId: 'm' }, getMerchant: async () => ({ id: 'm', name: '童装店', ownerVisible: false, publicPhone: false, publicWechat: true, contact: { phone: '13800138000', wechat: 'shop-a' } }) })
  await owner.hooks.load({ id: 'm' })
  assert.equal(owner.loadState.value, 'ready')
  assert.equal(owner.isOwner.value, true)
  assert.equal(owner.canShare.value, false)
  assert.equal(owner.publicContact.value.phone, '')
  assert.equal(owner.publicContact.value.wechat, 'shop-a')
  assert.deepEqual(JSON.parse(JSON.stringify(owner.hooks.share())), { title: '找档口', path: '/pages/sourcing-map/index' })
  assert.equal(owner.hooks.timeline().query, 'from=merchant-paused')
  owner.openEditor()
  assert.deepEqual(owner.calls[0], ['navigate', '/pages/merchant/profile?merchantId=m'])
})

test('contact actions use public values and clipboard feedback waits for success', async () => {
  const detail = { id: 'm', ownerVisible: true, publicPhone: true, publicWechat: true, contact: { phone: '13800138000', wechat: 'shop-a' } }
  const owner = page({ session: { merchantId: 'm' }, getMerchant: async () => detail })
  await owner.hooks.load({ id: 'm' })
  owner.openContact()
  assert.deepEqual(Array.from(owner.calls[0][1].itemList), ['拨打电话', '复制微信'])
  owner.calls[0][1].success({ tapIndex: 1 })
  assert.equal(owner.calls[1][0], 'copy')
  assert.equal(owner.calls.some(([type]) => type === 'toast'), false)
  owner.calls[1][1].success()
  assert.equal(owner.calls[2][1].title, '微信号已复制')
})

test('failed navigation offers address copy or retry', async () => {
  const detail = { id: 'm', name: '童装店', mapVisible: true, addressText: '织里商城 A-18', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1 } }
  const visitor = page({ getMerchant: async () => detail })
  await visitor.hooks.load({ id: 'm' })
  visitor.openNavigation()
  visitor.calls[0][1].fail()
  assert.deepEqual(Array.from(visitor.calls[1][1].itemList), ['复制地址', '重试导航'])
  visitor.calls[1][1].success({ tapIndex: 0 })
  assert.equal(visitor.calls[2][1].data, '织里商城 A-18')
  visitor.calls[2][1].success()
  assert.equal(visitor.calls[3][1].title, '地址已复制')
})

test('unavailable and network failures remain distinct and retryable', async () => {
  const unavailable = page({ getMerchant: async () => { throw { statusCode: 404 } } })
  await unavailable.hooks.load({ id: 'm' })
  assert.equal(unavailable.loadState.value, 'unavailable')
  const network = page({ getMerchant: async () => { throw Error('offline') } })
  await network.hooks.load({ id: 'm' })
  assert.equal(network.loadState.value, 'error')
  const codeOnly = page({ getMerchant: async () => { const error = Error('店铺不可用'); error.code = 'MERCHANT_NOT_FOUND'; throw error } })
  await codeOnly.hooks.load({ id: 'm' })
  assert.equal(codeOnly.loadState.value, 'unavailable')
  assert.match(source, /@click="loadMerchant"/)
})

test('landlord homepage uses rental business details and hides private map locations', async () => {
  const detail = { id: 'm', name: '张先生出租', ownerVisible: true, mapVisible: false, addressText: '不公开的住址', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1 }, businessProfile: { schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord'], propertyTypes: ['住房', '厂房'], serviceArea: '织里镇', processingMethods: [], serviceItems: [] } }
  const owner = page({ session: { merchantId: 'm' }, getMerchant: async () => detail })
  await owner.hooks.load({ id: 'm' })
  assert.equal(owner.loadState.value, 'ready')
  assert.match(owner.categoryText.value, /住房.*厂房/)
  assert.match(owner.businessRoleText.value, /出租/)
  assert.equal(owner.businessProfile.value.serviceArea, '织里镇')
  assert.equal(owner.place.value.address, '')
  assert.equal(owner.place.value.canNavigate, false)
  owner.openNavigation()
  assert.equal(owner.calls.length, 0)
  assert.equal(owner.hooks.share().path, '/pages/merchant/detail?id=m')
})
