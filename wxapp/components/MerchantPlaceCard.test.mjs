import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const source = fs.readFileSync(path.resolve(new URL('.', import.meta.url).pathname, 'MerchantPlaceCard.vue'), 'utf8')

test('merchant place card uses a booth doorplate hierarchy and distinguishes source state', () => {
  for (const token of [
    'merchant-place-card',
    'doorplate-code',
    'source-badge',
    '已入驻',
    '待认领',
    'market-location',
    '位置待完善',
  ]) {
    assert.match(source, new RegExp(token))
  }
})

test('merchant place card exposes navigation for valid coordinates and keeps claim available for prelisted booths', () => {
  assert.match(source, /v-if="hasLocation"/)
  assert.match(source, /@click\.stop="\$emit\('navigate', place\)"/)
  assert.match(source, /v-if="!place\.claimed" class="claim-button"/)
  assert.doesNotMatch(source, /v-else-if="!place\.claimed"/)
  assert.doesNotMatch(source, /拨打电话|复制微信|makePhoneCall|setClipboardData/)
})

test('claimed merchant card exposes an explicit merchant homepage action', () => {
  assert.match(source, /v-if="hasMerchantDetail\(place\)" class="detail-button"/)
  assert.match(source, /@click\.stop="\$emit\('detail', place\)"/)
  assert.match(source, />进入主页<\/button>/)
  assert.match(source, /defineEmits\(\['select', 'detail', 'navigate', 'claim'\]\)/)
})
