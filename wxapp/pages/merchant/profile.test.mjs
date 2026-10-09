import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import vm from 'node:vm'
import * as location from '../../common/merchantLocationForm.js'
import * as onboarding from '../../common/merchantOnboarding.js'
import * as business from '../../common/merchantBusinessProfile.js'
import { isMerchantDraftSupported } from '../../common/merchantDraft.js'
import * as categories from '../../common/merchantCategories.js'

const stall = { ...business.normalizeBusinessProfile(), entityKind: 'individual', primaryRole: 'stall', roles: ['stall'] }

const source = fs.readFileSync(new URL('./profile.vue', import.meta.url), 'utf8')
const script = source.match(/<script setup>([\s\S]*?)<\/script>/)[1]
  .replace(/import[\s\S]*?from\s+['"][^'"]+['"]\s*/g, '')

function page({ getMerchant = async () => ({ businessProfile: business.normalizeBusinessProfile(), mapVisible: false, name: '', profileStatus: 'incomplete', mainCategories: [], images: [], contact: {}, location: { status: 'hidden' } }), updateMerchant = async () => {}, saveMerchantDraft = () => true } = {}) {
  const actions = []
  const sandbox = {
    ...location, ...onboarding, ...categories, ...business, isMerchantDraftSupported,
    DEFAULT_CITY_CODE: 'zhili', MERCHANT_PROFILE_IMAGE_MAX_COUNT: 9,
    reactive: (value) => value, ref: (value) => ({ value }), computed: (getter) => ({ get value() { return getter() } }),
    onLoad: () => {}, onShareAppMessage: () => {},
    getSession: () => ({ token: 'token', userId: 'u1', merchantId: 'm1' }),
    getMerchant, updateMerchant, getMerchantDraft: () => null,
    saveMerchantDraft, clearMerchantDraft: () => true,
    createStoredMerchantImageEntry: (url) => ({ id: url, kind: 'stored', url }),
    getStoredMerchantImageUrls: (entries) => entries.filter((entry) => entry.kind === 'stored').map((entry) => entry.url),
    getMerchantImagePreviewUrl: (entry) => entry.url,
    getMerchantImageUrlsForPreview: (entries) => entries.map((entry) => entry.url),
    uni: { showToast: (options) => actions.push(['toast', options.title]), showModal: () => {}, hideLoading() {} },
    console: { warn() {} },
  }
  vm.runInNewContext(`${script}\nglobalThis.page = { loadMerchant, submitMerchantProfile, saveLocalDraft, restoreDraft, selectPrimaryRole, toggleBusinessRole, clearMerchantLocation, handleAddressTextInput, state: { form, merchantLoadError, fieldErrors, merchantId, userId, savedState } }`, sandbox)
  sandbox.page.state.merchantId.value = 'm1'
  sandbox.page.state.userId.value = 'u1'
  return { ...sandbox.page, actions }
}

test('failed load blocks submission and exposes retry state', async () => {
  let updates = 0
  const p = page({ getMerchant: async () => { throw new Error('internal phone data') }, updateMerchant: async () => { updates += 1 } })
  await p.loadMerchant()
  await p.submitMerchantProfile()
  assert.equal(p.state.merchantLoadError.value, true)
  assert.equal(updates, 0)
  assert.ok(p.actions.some(([type, message]) => type === 'toast' && message === '资料加载失败，请重试'))
})

test('first submission shows field errors before update', async () => {
  let updates = 0
  const p = page({ updateMerchant: async () => { updates += 1 } })
  await p.loadMerchant()
  await p.submitMerchantProfile()
  assert.equal(updates, 0)
  assert.deepEqual(Object.keys(p.state.fieldErrors).sort(), ['entityKind', 'name', 'primaryRole', 'roles'])
})

test('saving local draft never calls profile update', async () => {
  let updates = 0
  let saved
  const p = page({ updateMerchant: async () => { updates += 1 }, saveMerchantDraft: (_user, _merchant, draft) => { saved = draft; return true } })
  await p.loadMerchant()
  p.state.form.name = '织里小鹿童装'
  await p.saveLocalDraft()
  assert.equal(updates, 0)
  assert.equal(saved.name, '织里小鹿童装')
  assert.ok(p.actions.some(([type, message]) => type === 'toast' && message === '草稿已存到本机'))
})

test('valid first submission sends contact consent and shows server paused state', async () => {
  let reads = 0
  let patch
  const p = page({
    getMerchant: async () => {
      reads += 1
      return reads === 1
        ? { businessProfile: business.normalizeBusinessProfile(), mapVisible: false, name: '', profileStatus: 'incomplete', mainCategories: [], images: [], contact: {}, location: { status: 'hidden' } }
        : { businessProfile: stall, mapVisible: false, name: '织里小鹿童装', profileStatus: 'complete', ownerVisible: false, mainCategories: ['童装'], images: ['https://example.com/shop.jpg'], contact: {}, location: { status: 'visible', latitude: 30.8, longitude: 120.2 } }
    },
    updateMerchant: async (_id, value) => { patch = value },
  })
  await p.loadMerchant()
  p.restoreDraft({ mapVisible: false, businessProfile: { ...business.normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'stall', roles: ['stall'] }, name: '织里小鹿童装', mainCategories: ['童装'], images: ['https://example.com/shop.jpg'], addressText: '织里童装城 A108', latitude: 30.8, longitude: 120.2, contactPhone: '13800138000', publicPhone: false, publicWechat: true })
  await p.submitMerchantProfile()
  assert.equal(patch.publicPhone, false)
  assert.equal(patch.publicWechat, false)
  assert.equal(patch.logoUrl, 'https://example.com/shop.jpg')
  assert.equal(p.state.savedState.value, 'paused')
  assert.equal(reads, 2)
})

test('restoring a same-version visible draft cannot bypass the server risk-hidden location decision', async () => {
  let updates = 0
  const p = page({
    getMerchant: async () => ({ businessProfile: stall, name: '织里小鹿童装', profileStatus: 'complete', mapVisible: true, mainCategories: ['童装'], images: ['https://example.com/shop.jpg'], contact: {}, location: { status: 'risk_hidden', latitude: 30.8, longitude: 120.2 } }),
    updateMerchant: async () => { updates += 1 },
  })
  await p.loadMerchant()
  p.restoreDraft({ mapVisible: true, businessProfile: { ...business.normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'stall', roles: ['stall'] }, name: '织里小鹿童装', mainCategories: ['童装'], images: ['https://example.com/shop.jpg'], latitude: 30.8, longitude: 120.2, locationStatus: 'visible', description: '只改简介' })
  await p.submitMerchantProfile()
  assert.equal(updates, 0)
  assert.match(p.state.fieldErrors.location, /重新选择地图位置/)
})

const landlord = { ...business.normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord'], propertyTypes: ['厂房'] }
test('landlord completes without categories photos GPS and defaults off map', async () => {
  let patch
  const p = page({ updateMerchant: async (_id, value) => { patch = value } })
  await p.loadMerchant()
  p.state.form.name = '小鹿厂房出租'
  p.state.form.businessProfile = landlord
  await p.submitMerchantProfile()
  assert.equal(patch.mapVisible, false)
  assert.equal(patch.locationAction, 'keep')
  assert.equal(patch.businessProfile.primaryRole, 'landlord')
  assert.equal('merchantType' in patch, false)
  assert.equal(patch.images.length, 0)
})
test('restoring draft never opts a new profile into map publication', async () => {
  const p = page()
  await p.loadMerchant()
  p.restoreDraft({ businessProfile: landlord, mapVisible: true, latitude: 30.8, longitude: 120.2 })
  assert.equal(p.state.form.mapVisible, false)
})
test('switching roles preserves professional data and validates secondary roles', async () => {
  const p = page()
  await p.loadMerchant()
  p.state.form.businessProfile = { ...landlord, roles: ['landlord'], serviceItems: ['维修'] }
  p.selectPrimaryRole('service_provider')
  assert.deepEqual([...p.state.form.businessProfile.roles], ['landlord', 'service_provider'])
  p.toggleBusinessRole('landlord')
  assert.deepEqual(p.state.form.businessProfile.propertyTypes, ['厂房'])
})
test('map-disabled risk draft saves other data using keep without clearing private location', async () => {
  let patch
  const p = page({ getMerchant: async () => ({ name: '小鹿出租', profileStatus: 'complete', mapVisible: false, businessProfile: landlord, location: { status: 'risk_hidden', latitude: 30.8, longitude: 120.2 } }), updateMerchant: async (_id, value) => { patch = value } })
  await p.loadMerchant()
  p.restoreDraft({ businessProfile: landlord, name: '小鹿出租', latitude: 31, longitude: 121, mapVisible: true, description: '更新业务' })
  await p.submitMerchantProfile()
  assert.equal(patch.locationAction, 'keep')
  assert.equal(patch.mapVisible, false)
  assert.equal(p.state.form.latitude, 30.8)
  assert.equal(p.state.form.longitude, 120.2)
})
test('ordinary risk baseline saves unchanged location using keep', async () => {
  let patch
  const p = page({ getMerchant: async () => ({ name: '小鹿出租', profileStatus: 'complete', mapVisible: true, businessProfile: landlord, addressText: '示例路1号', location: { status: 'risk_hidden', latitude: 30.8, longitude: 120.2 } }), updateMerchant: async (_id, value) => { patch = value } })
  await p.loadMerchant()
  await p.submitMerchantProfile()
  assert.equal(patch.locationAction, 'keep')
})
test('explicit clear submits clear while retaining map-disabled profile', async () => {
  let patch
  const p = page({ getMerchant: async () => ({ name: '小鹿出租', profileStatus: 'complete', mapVisible: false, businessProfile: landlord, location: { status: 'visible', latitude: 30.8, longitude: 120.2 } }), updateMerchant: async (_id, value) => { patch = value } })
  await p.loadMerchant()
  p.clearMerchantLocation()
  await p.submitMerchantProfile()
  assert.equal(patch.locationAction, 'clear')
})

test('map enable requires actual coordinates and address even for completed profiles', async () => {
  let updates = 0
  const p = page({ updateMerchant: async () => { updates += 1 } })
  await p.loadMerchant()
  Object.assign(p.state.form, { name: '小鹿出租', businessProfile: landlord, mapVisible: true })
  await p.submitMerchantProfile()
  assert.equal(updates, 0)
  assert.ok(p.state.fieldErrors.addressText)
  assert.ok(p.state.fieldErrors.location)
})
test('editing location details cannot silently lose the edit via keep', async () => {
  let updates = 0
  const p = page({ getMerchant: async () => ({ name: '小鹿出租', profileStatus: 'complete', mapVisible: false, businessProfile: landlord, location: { status: 'visible', latitude: 30.8, longitude: 120.2 } }), updateMerchant: async () => { updates += 1 } })
  await p.loadMerchant()
  p.state.form.boothNo = 'A108'
  await p.submitMerchantProfile()
  assert.equal(updates, 0)
  assert.match(p.state.fieldErrors.location, /重新选择地图位置/)
})

test('location detail edits after restoring a different draft cannot be lost through keep', async () => {
  let updates = 0
  const p = page({
    getMerchant: async () => ({ name: '小鹿出租', profileStatus: 'complete', mapVisible: false, businessProfile: landlord, addressText: '示例路1号', location: { status: 'visible', latitude: 30.8, longitude: 120.2, boothNo: 'A101' } }),
    updateMerchant: async () => { updates += 1 },
  })
  await p.loadMerchant()
  p.restoreDraft({ businessProfile: landlord, mapVisible: false, addressText: '草稿路2号', latitude: 31, longitude: 121 })
  assert.equal(p.state.form.boothNo, 'A101', 'different draft location should restore the server baseline')
  p.state.form.boothNo = 'A102'
  await p.submitMerchantProfile()
  assert.equal(updates, 0, 'a changed location detail must not be silently ignored by keep')
  assert.match(p.state.fieldErrors.location, /重新选择地图位置/)
  assert.equal(p.state.form.boothNo, 'A102', 'keep the unsaved detail visible for correction')
})

test('unsupported or missing server profile and map switch block editing and submission', async () => {
  for (const detail of [{}, { businessProfile: business.normalizeBusinessProfile() }, { businessProfile: { ...business.normalizeBusinessProfile(), schemaVersion: 2 }, mapVisible: false }]) {
    let updates = 0
    const p = page({ getMerchant: async () => detail, updateMerchant: async () => { updates += 1 } })
    await p.loadMerchant()
    await p.submitMerchantProfile()
    assert.equal(p.state.merchantLoadError.value, true)
    assert.equal(updates, 0)
    assert.ok(p.actions.some(([, message]) => message === '业务资料格式或版本不受支持，请刷新后重试'))
  }
})
test('restoring a draft without the current business structure is ignored', async () => {
  const p = page()
  await p.loadMerchant()
  p.state.form.name = '当前编辑'
  p.restoreDraft({ name: '旧草稿' })
  assert.equal(p.state.form.name, '当前编辑')
})
