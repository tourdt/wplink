import assert from 'node:assert/strict'
import test from 'node:test'
import { getMyStorefrontState, getStorefrontPlace, getPublicContact } from './merchantStorefront.js'

test('my storefront distinguishes guest, loading, failure, onboarding and public states', () => {
  assert.equal(getMyStorefrontState({ token: '' }), 'guest')
  assert.equal(getMyStorefrontState({ token: 't', loading: true }), 'loading')
  assert.equal(getMyStorefrontState({ token: 't', error: true }), 'error')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: '' }), 'unregistered')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { profileStatus: 'incomplete' } }), 'incomplete')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { ownerVisible: false } }), 'paused')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { ownerVisible: true, mapVisible: true, location: { status: 'risk_hidden' } } }), 'location_error')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { ownerVisible: true, mapVisible: true, location: { status: 'hidden', available: false } } }), 'location_error')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { ownerVisible: true, mapVisible: true, location: { status: 'visible', available: true } } }), 'active')
})

test('navigation requires an available visible location with valid coordinates', () => {
  const base = { mapVisible: true, addressText: '织里商城 2F A-18', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1, marketName: '织里商城', floorNo: '2F', boothNo: 'A-18' } }
  assert.deepEqual(getStorefrontPlace(base), { address: '织里商城 2F A-18', position: '织里商城 · 2F · A-18', canNavigate: true, latitude: 30.8, longitude: 120.1 })
  for (const location of [{ ...base.location, available: false }, { ...base.location, status: 'risk_hidden' }, { ...base.location, latitude: 91 }, { ...base.location, longitude: '' }]) {
    assert.equal(getStorefrontPlace({ ...base, location }).canNavigate, false)
  }
})

test('contact exposes only authorized public fields', () => {
  assert.deepEqual(getPublicContact({ contactPhone: '13800138000', contactWechat: 'shop-a', contact: { name: '隐私姓名' } }), { phone: '', wechat: '' })
  assert.deepEqual(getPublicContact({ publicPhone: true, publicWechat: false, contact: { phone: '13800138000', wechat: 'shop-a' } }, true), { phone: '13800138000', wechat: '' })
  assert.deepEqual(getPublicContact({ contact: { phone: '13800138000', wechat: 'shop-a' } }), { phone: '13800138000', wechat: 'shop-a' })
})

test('a public profile without a public map location is a normal profile-only state', () => {
  const merchant = { profileStatus: 'completed', ownerVisible: true, mapVisible: false, location: { status: 'hidden', available: false } }
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant }), 'profile_only')
  assert.equal(getMyStorefrontState({ token: 't', merchantId: 'm', merchant: { ...merchant, ownerVisible: false } }), 'paused')
})

test('map opt-out hides private address and coordinates even in an owner preview', () => {
  const detail = { mapVisible: false, addressText: '私有接待地址', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1, marketName: '私有地址' } }
  assert.deepEqual(getStorefrontPlace(detail), { address: '', position: '', canNavigate: false, latitude: null, longitude: null })
})

// 未显式同意公开地点时，即使响应携带坐标也不能展示或导航。
test('missing map consent never exposes a location', () => {
  const detail = { addressText: '私有地址', location: { status: 'visible', available: true, latitude: 30.8, longitude: 120.1 } }
  assert.deepEqual(getStorefrontPlace(detail), { address: '', position: '', canNavigate: false, latitude: null, longitude: null })
})
