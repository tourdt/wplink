import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantPlaceQuery,
  hasMerchantDetail,
  hasValidLocation,
  merchantDetailPath,
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

test('merchant detail path only accepts claimed places with a merchant id', () => {
  const claimed = { claimed: true, merchantId: ' merchant/9 ' }

  assert.equal(hasMerchantDetail(claimed), true)
  assert.equal(merchantDetailPath(claimed), '/pages/merchant/detail?id=merchant%2F9')
  assert.equal(hasMerchantDetail({ claimed: true, merchantId: '' }), false)
  assert.equal(hasMerchantDetail({ claimed: false, merchantId: '9' }), false)
})
