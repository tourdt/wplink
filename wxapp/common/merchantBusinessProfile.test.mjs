import assert from 'node:assert/strict'
import test from 'node:test'
import { getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary, getBusinessProfileErrors, normalizeBusinessProfile } from './merchantBusinessProfile.js'

test('only explicit v1 structure can be read as a supported profile', () => {
  assert.notEqual(getBusinessProfile({ merchantType: 'factory' }).schemaVersion, 1)
  assert.equal(getBusinessProfile({ merchantType: 'factory' }).primaryRole, '')
  assert.notEqual(getBusinessProfile({ businessProfile: {} }).schemaVersion, 1)
  assert.notEqual(getBusinessProfile({ businessProfile: { schemaVersion: 2 } }).schemaVersion, 1)
  const empty = normalizeBusinessProfile()
  assert.equal(getBusinessProfile({ businessProfile: empty }).schemaVersion, 1)
  assert.equal(getBusinessProfile({ businessProfile: empty }).primaryRole, '')
  assert.equal('needsSetup' in getBusinessProfile({ businessProfile: empty }), false)
})
test('all selected roles require relevant professional data but no GPS or photos', () => {
  const profile = { ...normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord', 'service_provider'], propertyTypes: ['厂房'], serviceItems: [] }
  assert.deepEqual(getBusinessProfileErrors(profile), { serviceItems: '请填写服务项目' })
  profile.serviceItems = ['维修']
  assert.deepEqual(getBusinessProfileErrors(profile), {})
  assert.deepEqual(getBusinessRoleLabels(profile), ['房东 / 出租方', '配套服务'])
  assert.equal(getBusinessProfileSummary({ businessProfile: profile }), '出租：厂房；服务：维修')
})
test('invalid versions roles and oversized values are rejected', () => {
  assert.ok(getBusinessProfileErrors({ schemaVersion: 2 }).businessProfile)
  assert.ok(getBusinessProfileErrors({ schemaVersion: 1, entityKind: 'individual', primaryRole: 'bad', roles: ['bad'] }).primaryRole)
})

test('explicit unsupported versions are not reinterpreted as legacy profiles', () => {
  assert.equal(getBusinessProfile({ businessProfile: { schemaVersion: 0 } }).schemaVersion, 0)
})
test('normalization trims and deduplicates arrays with bounded professional data', () => {
  const profile = { ...normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'organization', primaryRole: 'factory', roles: ['factory', 'factory'], processingMethods: [' 缝制 ', '缝制', ''] }
  assert.deepEqual(getBusinessProfile({ businessProfile: profile }).processingMethods, ['缝制'])
  assert.deepEqual(getBusinessProfileErrors(profile, ['童装']), {})
  assert.ok(getBusinessProfileErrors({ ...profile, processingMethods: ['工'.repeat(51)] }, ['童装']).processingMethods)
  assert.ok(getBusinessProfileErrors({ ...profile, serviceArea: '织'.repeat(201) }, ['童装']).serviceArea)
})

test('old types and categories never supply business identity or summary', () => {
  assert.equal(getBusinessProfileSummary({ merchantType: 'individual', mainCategories: ['童装'] }), '')
})

test('explicit landlord profile does not show retained apparel categories', () => {
  const detail = { mainCategories: ['童装'], businessProfile: { ...normalizeBusinessProfile(), schemaVersion: 1, entityKind: 'individual', primaryRole: 'landlord', roles: ['landlord'], propertyTypes: ['厂房'] } }
  assert.equal(getBusinessProfileSummary(detail), '出租：厂房')
})
