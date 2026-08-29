import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildMerchantDirectoryQuery,
  hasValidMerchantLocation,
  merchantDirectoryCoordinateText,
  merchantDirectoryDetailPath,
  mergeMerchantDirectoryItems,
  normalizeMerchantDirectoryItem,
  parseMerchantCategories,
} from './merchantDirectoryState.js'

test('merchant directory query contains only city keyword categories and pagination', () => {
  assert.deepEqual(buildMerchantDirectoryQuery({
    cityCode: ' zhili ',
    keyword: ' A108 ',
    categories: ' 童装, 女童，童装 ',
    merchantTypes: ['factory'],
    claimed: 'claimed',
    page: 2,
    pageSize: 20,
  }), {
    cityCode: 'zhili',
    keyword: 'A108',
    categories: '童装,女童',
    page: 2,
    pageSize: 20,
  })
})

test('merchant categories are clearable comma-separated text', () => {
  assert.deepEqual(parseMerchantCategories('童装, 女童，童装, ,现货'), ['童装', '女童', '现货'])
  assert.deepEqual(parseMerchantCategories('   '), [])
})

test('directory items expose only merchant identity and public directory fields', () => {
  const item = normalizeMerchantDirectoryItem({
    merchantId: ' 1 ', merchantNo: 'M001', name: '晨星童装', mainCategories: ['童装', '童装'],
    boothNo: ' A108 ', latitude: '30.8732', longitude: '120.2255', objectId: 'object-1',
    claimed: true, sourceType: 'merchant_claimed', phone: '18800000000', wechat: 'secret',
  })

  assert.deepEqual(item, {
    merchantId: '1', merchantNo: 'M001', name: '晨星童装', merchantType: '', mainCategories: ['童装'],
    logoUrl: '', addressText: '', marketName: '', buildingName: '', floorNo: '', boothNo: 'A108',
    latitude: 30.8732, longitude: 120.2255, locationName: '', lastResourceActiveAt: '', distanceMeters: 0,
  })
  for (const legacyOrPrivateField of ['objectId', 'claimed', 'sourceType', 'phone', 'wechat']) {
    assert.equal(legacyOrPrivateField in item, false)
  }
})

test('merchant directory coordinates reject blanks non-finite values and out-of-range numbers', () => {
  const cases = [
    [{ latitude: 30.8732, longitude: 120.2255 }, true],
    [{ latitude: '', longitude: 120.2255 }, false],
    [{ latitude: '  ', longitude: 120.2255 }, false],
    [{ latitude: Number.NaN, longitude: 120.2255 }, false],
    [{ latitude: Number.POSITIVE_INFINITY, longitude: 120.2255 }, false],
    [{ latitude: 91, longitude: 120.2255 }, false],
    [{ latitude: 30.8732, longitude: 181 }, false],
  ]
  for (const [item, expected] of cases) assert.equal(hasValidMerchantLocation(item), expected)
})

test('merchant coordinate strip follows the market building floor booth order', () => {
  assert.equal(merchantDirectoryCoordinateText({
    marketName: '织里童装城', buildingName: '一期', floorNo: '1F', boothNo: 'A108', addressText: '织里镇',
  }), '织里童装城 · 一期 · 1F · A108')
  assert.equal(merchantDirectoryCoordinateText({ addressText: '织里镇富民路' }), '织里镇富民路')
  assert.equal(merchantDirectoryCoordinateText({}), '档口地址待完善')
})

test('merchant detail navigation uses merchantId as its only identity', () => {
  assert.equal(merchantDirectoryDetailPath({ merchantId: ' merchant/a?b ' }), '/pages/merchant/detail?id=merchant%2Fa%3Fb')
  assert.equal(merchantDirectoryDetailPath({ merchantId: '' }), '')
})

test('stable pagination preserves first occurrence and deduplicates by merchantId', () => {
  const current = [normalizeMerchantDirectoryItem({ merchantId: '1', name: '第一页商家', latitude: 30, longitude: 120 })]
  const incoming = [
    normalizeMerchantDirectoryItem({ merchantId: '1', name: '重复商家', latitude: 30, longitude: 120 }),
    normalizeMerchantDirectoryItem({ merchantId: '2', name: '第二页商家', latitude: 30, longitude: 120 }),
  ]
  assert.deepEqual(mergeMerchantDirectoryItems(current, incoming).map((item) => [item.merchantId, item.name]), [
    ['1', '第一页商家'], ['2', '第二页商家'],
  ])
})
