import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

import { hasValidMerchantLocation, merchantDirectoryCoordinateText } from '../pages/sourcing-map/merchantDirectoryState.js'

const source = fs.readFileSync(path.resolve(new URL('.', import.meta.url).pathname, 'MerchantPlaceCard.vue'), 'utf8')

test('merchant card uses the compact market building floor booth coordinate strip', () => {
  assert.equal(merchantDirectoryCoordinateText({
    marketName: '织里童装城', buildingName: '一期', floorNo: '1F', boothNo: 'A108',
  }), '织里童装城 · 一期 · 1F · A108')
  assert.match(source, /coordinate-strip/)
  assert.match(source, /merchantDirectoryCoordinateText/)
  assert.equal(source.match(/coordinateText/g)?.length, 2, '坐标只应由计算属性生成并在签名坐标条渲染一次')
})

test('merchant card has only homepage and valid-location actions', () => {
  assert.match(source, /defineEmits\(\['detail', 'location'\]\)/)
  assert.match(source, />进入主页<\/button>/)
  assert.match(source, />查看位置<\/button>/)
  assert.equal(hasValidMerchantLocation({ latitude: 30.8732, longitude: 120.2255 }), true)
  assert.equal(hasValidMerchantLocation({ latitude: '', longitude: 120.2255 }), false)
  assert.doesNotMatch(source, /待认领|已入驻|认领|claim|claimed|objectId|sourceType|navigate/)
})

test('merchant card renders public main categories without contact actions', () => {
  assert.match(source, /:tags="merchant\.mainCategories"/)
  assert.doesNotMatch(source, /电话|微信|makePhoneCall|setClipboardData/)
})
