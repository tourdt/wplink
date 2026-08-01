import assert from 'node:assert/strict'
import test from 'node:test'

import * as merchantPlaceState from './merchantPlaceState.js'
import {
  buildMerchantPlaceQuery,
  hasValidLocation,
  merchantPlaceSourceLabel,
  normalizeMerchantPlace,
} from './merchantPlaceState.js'

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

test('merchant place location accepts only finite Tencent map coordinates', () => {
  assert.equal(hasValidLocation({ lat: '30.89912', lng: '120.20482' }), true)
  assert.equal(hasValidLocation({ lat: '', lng: '120.20482' }), false)
  assert.equal(hasValidLocation({ lat: '91', lng: '120.20482' }), false)
  assert.equal(hasValidLocation({ lat: 'NaN', lng: '120.20482' }), false)
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

test('merchant detail path is available only for claimed places with a merchant identity', () => {
  assert.equal(
    merchantPlaceState.merchantDetailPath?.({ claimed: true, merchantId: ' 8020000000000000001 ' }),
    '/pages/merchant/detail?id=8020000000000000001',
  )
  assert.equal(
    merchantPlaceState.merchantDetailPath?.({ claimed: false, merchantId: '8020000000000000001' }),
    '',
  )
  assert.equal(
    merchantPlaceState.merchantDetailPath?.({ claimed: true, merchantId: '' }),
    '',
  )
  assert.equal(
    merchantPlaceState.hasMerchantDetail?.({ claimed: true, merchantId: ' 8020000000000000001 ' }),
    true,
  )
  assert.equal(
    merchantPlaceState.hasMerchantDetail?.({ claimed: true, merchantId: '   ' }),
    false,
  )
})
