import assert from 'node:assert/strict'
import test from 'node:test'
import { normalizeBusinessProfile } from './merchantBusinessProfile.js'
import { buildMerchantContactVisibility, getMerchantOnboardingErrors, getMerchantSavedState } from './merchantOnboarding.js'

const landlord = { ...normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord'], propertyTypes: ['厂房'] }
test('first submission needs explicit business setup but no photo or GPS', () => {
  assert.deepEqual(getMerchantOnboardingErrors({ name: '小鹿出租', businessProfile: landlord, mapVisible: false }), {})
  const errors = getMerchantOnboardingErrors({ name: '小鹿出租', businessProfile: landlord, mapVisible: true })
  assert.deepEqual(Object.keys(errors).sort(), ['addressText', 'location'])
})

test('phone authorization alone does not publish contact and empty values cannot be public', () => {
  assert.deepEqual(buildMerchantContactVisibility({ contactPhone: '13800138000', contactWechat: '', publicPhone: false, publicWechat: true }), { publicPhone: false, publicWechat: false })
  assert.deepEqual(buildMerchantContactVisibility({ contactPhone: '13800138000', contactWechat: 'shop', publicPhone: true, publicWechat: true }), { publicPhone: true, publicWechat: true })
})

test('saved status uses server visibility and location state', () => {
  assert.equal(getMerchantSavedState({ ownerVisible: false }), 'paused')
  assert.equal(getMerchantSavedState({ ownerVisible: true, location: { status: 'risk_hidden' } }), 'location_error')
  assert.equal(getMerchantSavedState({ ownerVisible: true, location: { status: 'visible', available: true } }), 'public')
  assert.equal(getMerchantSavedState({ ownerVisible: true, location: { status: 'visible', available: false } }), 'location_error')
  assert.equal(getMerchantSavedState({ ownerVisible: true }), 'location_error')
})

test('map disabled is a normal public profile state', () => { assert.equal(getMerchantSavedState({ ownerVisible: true, mapVisible: false }), 'profile_public') })

test('save validation requires an explicit map visibility choice and complete business structure', () => {
  assert.ok(getMerchantOnboardingErrors({ name: '小鹿出租', businessProfile: landlord }).mapVisible)
  assert.ok(getMerchantOnboardingErrors({ name: '小鹿出租', mapVisible: false, businessProfile: { schemaVersion: 1 } }).businessProfile)
})
