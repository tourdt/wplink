import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary } from '../../common/merchantBusinessProfile.js'
import { getMyStorefrontState, getStorefrontPlace } from '../../common/merchantStorefront.js'

const source = fs.readFileSync(path.join(path.dirname(new URL(import.meta.url).pathname), 'index.vue'), 'utf8')

function page({ session = { token: 'token', userId: 'u', merchantId: 'm' }, getMerchant = async () => ({ id: 'm', name: '童装店', ownerVisible: true }), setMerchantVisibility = async () => {}, draft = null } = {}) {
  const calls = []
  const hooks = {}
  const sandbox = {
    getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary,
    ref: (value) => ({ value }),
    computed: (getter) => ({ get value() { return getter() } }),
    onLoad: (fn) => { hooks.load = fn },
    onShow: (fn) => { hooks.show = fn },
    onShareAppMessage: (fn) => { hooks.share = fn },
    buildLoginUrl: (url) => `/pages/login/index?redirect=${encodeURIComponent(url)}`,
    getMerchantDraft: () => draft,
    getMyStorefrontState,
    getStorefrontPlace,
    getSession: () => session,
    getMerchant,
    setMerchantVisibility,
    uni: {
      navigateTo: (options) => calls.push(['navigate', options.url]),
      showToast: (options) => calls.push(['toast', options.title]),
      setClipboardData: (options) => calls.push(['copy', options.data]),
    },
  }
  const script = source.match(/<script setup>([\s\S]*?)<\/script>/)[1].replace(/^import .*$/gm, '')
  vm.runInNewContext(`${script}\nglobalThis.p = { syncSession, loadMerchantProfile, openMerchantEditor, openMerchantHome, toggleVisibility, openAccountSettings, pageState, merchantProfile, hasLocalDraft, visibilityBusy }`, sandbox)
  return { ...sandbox.p, hooks, calls }
}

test('my page renders distinct management states and has no launch-excluded service requests', () => {
  for (const state of ['guest', 'loading', 'error', 'unregistered', 'incomplete', 'paused', 'profile_only', 'location_error']) {
    assert.match(source, new RegExp(`pageState === '${state}'`))
  }
  assert.match(source, /@click="loadMerchantProfile"/)
  assert.doesNotMatch(source, /当前状态不会被当作未入驻/)
  assert.match(source, /@click="openMerchantEditor"/)
  assert.match(source, /@click="toggleVisibility"/)
  assert.match(source, /open-type="share"/)
  assert.doesNotMatch(source, /getMerchantEntitlements|listResources|getActiveGrowthCampaigns|getMerchantFollowState/)
})

test('guest never requests merchant data; failed owner request shows retry instead of onboarding', async () => {
  let requests = 0
  const guest = page({ session: { token: '', userId: '', merchantId: '' }, getMerchant: async () => { requests++; return {} } })
  await guest.syncSession()
  assert.equal(guest.pageState.value, 'guest')
  guest.openMerchantEditor()
  assert.match(guest.calls[0][1], /pages\/login/)
  assert.equal(requests, 0)

  const failed = page({ getMerchant: async () => { throw Error('offline') } })
  await failed.syncSession()
  assert.equal(failed.pageState.value, 'error')
  assert.equal(failed.merchantProfile.value.id, undefined)
})

test('owner visibility action sends one request, refreshes server state, and guards duplicate taps', async () => {
  let resolveVisibility
  const pending = new Promise((resolve) => { resolveVisibility = resolve })
  const calls = []
  let visible = true
  const owner = page({
    getMerchant: async () => ({ id: 'm', ownerVisible: visible, mapVisible: true, location: { status: 'visible', available: true } }),
    setMerchantVisibility: async (id, next) => { calls.push([id, next]); await pending; visible = next },
  })
  await owner.syncSession()
  const first = owner.toggleVisibility()
  const duplicate = owner.toggleVisibility()
  assert.equal(owner.visibilityBusy.value, true)
  assert.deepEqual(calls, [['m', false]])
  resolveVisibility()
  await Promise.all([first, duplicate])
  assert.equal(owner.pageState.value, 'paused')
  assert.equal(owner.visibilityBusy.value, false)
  assert.deepEqual(JSON.parse(JSON.stringify(owner.hooks.share())), { title: '找档口', path: '/pages/sourcing-map/index' })
  owner.openMerchantHome()
  assert.deepEqual(owner.calls.find(([type]) => type === 'navigate'), ['navigate', '/pages/merchant/detail?id=m'])
})

test('local draft is shown only for current signed-in account and edit uses current merchant', async () => {
  const owner = page({ draft: { name: '本机草稿' } })
  await owner.syncSession()
  assert.equal(owner.hasLocalDraft.value, true)
  owner.openMerchantEditor()
  assert.deepEqual(owner.calls[0], ['navigate', '/pages/merchant/profile?merchantId=m'])
  const noUser = page({ session: { token: 'token', userId: '', merchantId: '' }, draft: { name: 'old' } })
  await noUser.syncSession()
  assert.equal(noUser.hasLocalDraft.value, false)
})

test('a landlord without GPS can view and share a public profile and pause it', async () => {
  const calls = []
  let visible = true
  const owner = page({
    getMerchant: async () => ({ id: 'm', name: '张先生出租', profileStatus: 'completed', ownerVisible: visible, mapVisible: false, businessProfile: { schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord'], propertyTypes: ['住房'], serviceArea: '织里镇', processingMethods: [], serviceItems: [] }, location: { status: 'hidden', available: false } }),
    setMerchantVisibility: async (id, next) => { calls.push([id, next]); visible = next },
  })
  await owner.syncSession()
  assert.equal(owner.pageState.value, 'profile_only')
  owner.openMerchantHome()
  assert.equal(owner.hooks.share().path, '/pages/merchant/detail?id=m')
  assert.deepEqual(owner.calls[0], ['navigate', '/pages/merchant/detail?id=m'])
  await owner.toggleVisibility()
  assert.deepEqual(calls, [['m', false]])
  assert.equal(owner.pageState.value, 'paused')
})
