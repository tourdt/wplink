import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizeBusinessProfile } from './merchantBusinessProfile.js'
import { clearMerchantDraft, getMerchantDraft, saveMerchantDraft } from './merchantDraft.js'

function storage() {
  const values = new Map()
  globalThis.uni = {
    getStorageSync: (key) => values.get(key),
    setStorageSync: (key, value) => values.set(key, value),
    removeStorageSync: (key) => values.delete(key),
  }
  return values
}

test('drafts are isolated by user and merchant and clearing one preserves the others', () => {
  storage()
  saveMerchantDraft('u1', 'm1', { businessProfile: normalizeBusinessProfile(), mapVisible: false, name: '甲', images: ['https://example.com/a.jpg'] })
  saveMerchantDraft('u1', 'm2', { businessProfile: normalizeBusinessProfile(), mapVisible: false, name: '乙' })
  saveMerchantDraft('u2', 'm1', { businessProfile: normalizeBusinessProfile(), mapVisible: false, name: '丙' })
  assert.equal(getMerchantDraft('u1', 'm1').name, '甲')
  assert.equal(getMerchantDraft('u1', 'm2').name, '乙')
  assert.equal(getMerchantDraft('u2', 'm1').name, '丙')
  clearMerchantDraft('u1', 'm1')
  assert.equal(getMerchantDraft('u1', 'm1'), null)
  assert.equal(getMerchantDraft('u2', 'm1').name, '丙')
})

test('draft storage rejects missing identity and ephemeral images', () => {
  const values = storage()
  assert.equal(saveMerchantDraft('', 'm1', { businessProfile: normalizeBusinessProfile(), mapVisible: false, name: '甲' }), false)
  assert.equal(saveMerchantDraft('u1', 'm1', { businessProfile: normalizeBusinessProfile(), mapVisible: false, images: ['wxfile://tmp/1.jpg'] }), false)
  assert.equal(values.size, 0)
})

test('v2 namespace ignores old storage and rejects missing or unknown profile structure', () => {
  const values = storage()
  values.set('wplink_merchant_draft_v1:u1:m1', { name: '旧草稿' })
  assert.equal(getMerchantDraft('u1', 'm1'), null)
  assert.equal(saveMerchantDraft('u1', 'm1', { name: '缺业务档案' }), false)
  assert.equal(saveMerchantDraft('u1', 'm1', { businessProfile: {}, mapVisible: false }), false)
  assert.equal(saveMerchantDraft('u1', 'm1', { businessProfile: { ...normalizeBusinessProfile(), schemaVersion: 2 }, mapVisible: false }), false)
  assert.equal(saveMerchantDraft('u1', 'm1', { businessProfile: normalizeBusinessProfile() }), false)
  assert.equal(saveMerchantDraft('u1', 'm1', { businessProfile: normalizeBusinessProfile(), mapVisible: false }), true)
  assert.ok(values.has('wplink_merchant_draft_v2:u1:m1'))
  values.set('wplink_merchant_draft_v2:u1:m1', { businessProfile: {}, mapVisible: false })
  assert.equal(getMerchantDraft('u1', 'm1'), null, 'invalid current-namespace data is also rejected')
  assert.ok(values.has('wplink_merchant_draft_v1:u1:m1'), 'old storage is neither read nor migrated')
})
