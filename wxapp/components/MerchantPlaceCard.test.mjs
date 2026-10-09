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

test('merchant card has one whole-card homepage action and a real-image fallback', () => {
  assert.match(source, /defineEmits\(\['detail'\]\)/)
  assert.match(source, /@click="\$emit\('detail', merchant\)"/)
  assert.doesNotMatch(source, /<button/)
  assert.match(source, /:src="merchant.logoUrl"/)
  assert.match(source, /@error="failedLogo = merchant.logoUrl"/)
  assert.match(source, /v-else class="shop-cover shop-placeholder"/)
})

test('merchant card renders public main categories without contact actions', () => {
  assert.match(source, /merchant\.mainCategories\.slice\(0, 3\)/)
  assert.doesNotMatch(source, /电话|微信|makePhoneCall|setClipboardData/)
})
