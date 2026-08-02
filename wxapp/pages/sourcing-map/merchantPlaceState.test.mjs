import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantTagLabels,
  buildMerchantPlaceQuery,
  hasMerchantDetail,
  hasValidLocation,
  merchantDetailPath,
  merchantPlaceSourceLabel,
  normalizeMerchantPlace,
} from './merchantPlaceState.js'

test('merchant tag labels translate configured codes and retain unmapped labels', () => {
  const labels = buildMerchantTagLabels([
    { code: 'girl', name: '女童' },
    { code: 'spot', name: '现货' },
  ])

  assert.deepEqual(labels(['girl', 'spot', '新标签']), ['女童', '现货', '新标签'])
})

test('merchant place query trims filters and keeps list and map pagination explicit', () => {
  assert.deepEqual(buildMerchantPlaceQuery({
    cityCode: ' zhili ',
    keyword: ' 童装 ',
    categories: ['girl', '', 'girl'],
    merchantTypes: ['factory'],
    claimed: 'claimed',
    page: 2,
    pageSize: 20,
  }), {
    cityCode: 'zhili',
    keyword: '童装',
    categories: 'girl',
    merchantTypes: 'factory',
    claimed: 'claimed',
    page: 2,
    pageSize: 20,
  })
})

test('merchant place location rejects blank, non-finite, and out-of-range coordinates', () => {
  const cases = [
    { place: { lat: '30.89912', lng: '120.20482' }, expected: true },
    { place: { lat: '', lng: '120.20482' }, expected: false },
    { place: { lat: '   ', lng: '120.20482' }, expected: false },
    { place: { lat: '\t', lng: '120.20482' }, expected: false },
    { place: { lat: 'NaN', lng: '120.20482' }, expected: false },
    { place: { lat: Number.NaN, lng: '120.20482' }, expected: false },
    { place: { lat: Number.POSITIVE_INFINITY, lng: '120.20482' }, expected: false },
    { place: { lat: '30.89912', lng: Number.NEGATIVE_INFINITY }, expected: false },
    { place: { lat: '91', lng: '120.20482' }, expected: false },
    { place: { lat: '-91', lng: '120.20482' }, expected: false },
    { place: { lat: '30.89912', lng: '181' }, expected: false },
    { place: { lat: '30.89912', lng: '-181' }, expected: false },
  ]

  for (const { place, expected } of cases) {
    assert.equal(hasValidLocation(place), expected)
  }
})

test('merchant places normalize claimed and prelisted labels without contact fields', () => {
  const claimed = normalizeMerchantPlace({
    objectId: '1',
    merchantId: '9',
    name: '小鹿童装',
    code: 'A001',
    sourceType: 'merchant_claimed',
    phone: '18800000001',
    wechat: 'xiaolu',
  })
  const prelisted = normalizeMerchantPlace({
    objectId: '2',
    name: 'B008 档口',
    code: 'B008',
    sourceType: 'platform_prelisted',
  })

  assert.equal(claimed.claimed, true)
  assert.equal(claimed.phone, undefined)
  assert.equal(claimed.wechat, undefined)
  assert.equal(merchantPlaceSourceLabel(claimed), '已入驻')
  assert.equal(prelisted.claimed, false)
  assert.equal(merchantPlaceSourceLabel(prelisted), '待认领')
})

test('merchant detail path only accepts claimed places with a merchant id', () => {
  const claimed = { claimed: true, merchantId: ' merchant/9 ' }

  assert.equal(hasMerchantDetail(claimed), true)
  assert.equal(merchantDetailPath(claimed), '/pages/merchant/detail?id=merchant%2F9')
  assert.equal(hasMerchantDetail({ claimed: true, merchantId: '' }), false)
  assert.equal(hasMerchantDetail({ claimed: false, merchantId: '9' }), false)
})
