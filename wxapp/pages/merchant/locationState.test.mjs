import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantLocationMarkers,
  merchantIdFromMarker,
  merchantMainTags,
  nearbyMerchantDomId,
  normalizeMerchantLocationContext,
} from './locationState.js'

function merchant(merchantId = '1', overrides = {}) {
  return {
    merchantId, merchantNo: `M${merchantId}`, name: `商家${merchantId}`, mainCategories: ['童装', '现货'],
    marketName: '织里童装城', buildingName: '一期', floorNo: '1F', boothNo: `A${merchantId}`,
    latitude: 30.8732, longitude: 120.2255, distanceMeters: 100, ...overrides,
  }
}

test('location context directly normalizes directory DTO without legacy identities', () => {
  const context = normalizeMerchantLocationContext({ current: merchant('1', { objectId: 'old', claimed: true }), nearby: [], radiusMeters: 1000, nearbyAvailable: true })
  assert.equal(context.current.merchantId, '1')
  assert.equal('objectId' in context.current, false)
  assert.equal('claimed' in context.current, false)
  assert.deepEqual(context.current.mainCategories, ['童装', '现货'])
})

test('location context rejects invalid current coordinates and filters invalid nearby merchants', () => {
  const invalid = normalizeMerchantLocationContext({ current: merchant('1', { latitude: ' ' }), nearby: [merchant('2')] })
  assert.equal(invalid.current, null)

  const context = normalizeMerchantLocationContext({
    current: merchant('1'),
    nearby: [merchant('2'), merchant('3', { longitude: 181 }), merchant('4', { latitude: Number.NaN })],
    nearbyAvailable: true,
  })
  assert.deepEqual(context.nearby.map((item) => item.merchantId), ['2'])
})

test('location context preserves server order deduplicates and never expands beyond 1000m or 20 merchants', () => {
  const context = normalizeMerchantLocationContext({
    current: merchant('1'),
    nearby: [merchant('1'), merchant('2'), merchant('2', { name: '重复' }), ...Array.from({ length: 25 }, (_, index) => merchant(String(index + 3)))],
    radiusMeters: 3000,
    nearbyAvailable: true,
  })
  assert.equal(context.radiusMeters, 1000)
  assert.equal(context.nearby.length, 20)
  assert.deepEqual(context.nearby.slice(0, 3).map((item) => item.merchantId), ['2', '3', '4'])
})

test('location markers use only current nearby and selected-nearby assets', () => {
  const context = normalizeMerchantLocationContext({ current: merchant('1'), nearby: [merchant('2'), merchant('3')], radiusMeters: 1000, nearbyAvailable: true })
  const markers = buildMerchantLocationMarkers(context, '2')
  assert.deepEqual(markers.map((item) => item.iconPath), [
    '/static/map/marker-selected.png', '/static/map/marker-nearby-selected.png', '/static/map/marker-nearby.png',
  ])
  assert.equal(merchantIdFromMarker(markers[1].id, markers), '2')
  assert.equal(merchantIdFromMarker('missing', markers), '')
})

test('nearby merchant helpers use trimmed merchant identity and public categories', () => {
  assert.equal(nearbyMerchantDomId(' merchant/a?b '), 'nearby-6d-65-72-63-68-61-6e-74-2f-61-3f-62')
  assert.deepEqual(merchantMainTags({ mainCategories: ['童装', '童装', '现货', '女童', '婴童'] }), ['童装', '现货', '女童'])
})
