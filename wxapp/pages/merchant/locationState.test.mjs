import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantLocationMarkers,
  merchantIdFromMarker,
  normalizeMerchantLocationContext,
} from './locationState.js'

function merchantPlace(overrides = {}) {
  return {
    objectId: 'object-1',
    merchantId: 'merchant-1',
    name: '小熊星球童装',
    code: 'A101',
    categoryCodes: ['children'],
    serviceTags: [],
    platformTags: [],
    claimed: true,
    sourceType: 'merchant_claimed',
    lat: '30.8700000',
    lng: '120.1200000',
    ...overrides,
  }
}

test('location context rejects an invalid current merchant coordinate', () => {
  const context = normalizeMerchantLocationContext({
    current: merchantPlace({ lat: 'NaN' }),
    nearby: [merchantPlace({ objectId: 'object-2', merchantId: 'merchant-2' })],
    radiusMeters: 1000,
    nearbyAvailable: true,
  })

  assert.equal(context.current, null)
})

test('location context keeps valid nearby merchants and filters every invalid coordinate', () => {
  const context = normalizeMerchantLocationContext({
    current: merchantPlace(),
    nearby: [
      merchantPlace({ objectId: 'object-2', merchantId: 'merchant-2', distanceMeters: 56 }),
      merchantPlace({ objectId: 'object-3', merchantId: 'merchant-3', lat: '   ' }),
      merchantPlace({ objectId: 'object-4', merchantId: 'merchant-4', lng: Number.POSITIVE_INFINITY }),
      merchantPlace({ objectId: 'object-5', merchantId: 'merchant-5', lat: '91' }),
    ],
    radiusMeters: 1000,
    nearbyAvailable: true,
  })

  assert.equal(context.current.merchantId, 'merchant-1')
  assert.deepEqual(context.nearby.map((item) => [item.merchantId, item.distanceMeters]), [
    ['merchant-2', 56],
  ])
  assert.equal(context.radiusMeters, 1000)
  assert.equal(context.nearbyAvailable, true)
})

test('location context preserves server order while excluding current, deduplicating, and limiting nearby merchants', () => {
  const nearby = [
    merchantPlace({ objectId: 'current-copy', merchantId: 'merchant-1', distanceMeters: 1 }),
    merchantPlace({ objectId: 'object-2-first', merchantId: 'merchant-2', distanceMeters: 20 }),
    merchantPlace({ objectId: 'object-2-later', merchantId: 'merchant-2', distanceMeters: 21 }),
    ...Array.from({ length: 21 }, (_, index) => merchantPlace({
      objectId: `object-${index + 3}`,
      merchantId: `merchant-${index + 3}`,
      distanceMeters: index + 30,
    })),
  ]

  const context = normalizeMerchantLocationContext({
    current: merchantPlace(),
    nearby,
    nearbyAvailable: true,
  })

  assert.deepEqual(
    context.nearby.map((item) => item.merchantId),
    [
      'merchant-2', 'merchant-3', 'merchant-4', 'merchant-5', 'merchant-6',
      'merchant-7', 'merchant-8', 'merchant-9', 'merchant-10', 'merchant-11',
      'merchant-12', 'merchant-13', 'merchant-14', 'merchant-15', 'merchant-16',
      'merchant-17', 'merchant-18', 'merchant-19', 'merchant-20', 'merchant-21',
    ],
  )
  assert.equal(context.nearby[0].objectId, 'object-2-first')
})

test('location markers keep the current merchant as the fixed orange focus', () => {
  const context = normalizeMerchantLocationContext({
    current: merchantPlace(),
    nearby: [merchantPlace({ objectId: 'object-2', merchantId: 'merchant-2' })],
    nearbyAvailable: true,
  })

  const markers = buildMerchantLocationMarkers(context)

  assert.equal(markers[0].id, 1)
  assert.equal(markers[0].merchantId, 'merchant-1')
  assert.equal(markers[0].iconPath, '/static/map/marker-selected.png')
  assert.equal(markers[0].zIndex, 10)
  assert.equal(markers[0].callout.display, 'ALWAYS')
  assert.ok(markers[0].width > markers[1].width)
  assert.ok(markers[0].height > markers[1].height)
})

test('location markers use the temporary selected asset only for the tapped nearby merchant', () => {
  const context = normalizeMerchantLocationContext({
    current: merchantPlace(),
    nearby: [
      merchantPlace({ objectId: 'object-2', merchantId: 'merchant-2' }),
      merchantPlace({ objectId: 'object-3', merchantId: 'merchant-3' }),
    ],
    nearbyAvailable: true,
  })

  const markers = buildMerchantLocationMarkers(context, 'merchant-2')

  assert.equal(markers[1].iconPath, '/static/map/marker-nearby-selected.png')
  assert.equal(markers[2].iconPath, '/static/map/marker-nearby.png')
  assert.equal(merchantIdFromMarker(markers[1].id, markers), 'merchant-2')
  assert.equal(merchantIdFromMarker('missing', markers), '')
})
