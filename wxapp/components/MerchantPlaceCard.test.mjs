import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const source = fs.readFileSync(path.resolve(new URL('.', import.meta.url).pathname, 'MerchantPlaceCard.vue'), 'utf8')

test('merchant place card uses a booth doorplate hierarchy and distinguishes source state', () => {
  for (const token of [
    'doorplate-code',
    'source-badge',
    '已入驻',
    '待认领',
    'locationText',
    '位置待完善',
  ]) {
    assert.match(source, new RegExp(token))
  }
})

test('merchant place card separates claimed location viewing from prelisted navigation', () => {
  assert.match(source, /v-if="place\.claimed && hasLocation"[^>]*@click\.stop="\$emit\('location', place\)"[^>]*>查看位置<\/button>/)
  assert.match(source, /v-else-if="hasLocation"[^>]*@click\.stop="\$emit\('navigate', place\)"[^>]*>导航<\/button>/)
  assert.match(source, /v-if="!place\.claimed" class="claim-button"/)
  assert.match(source, /defineEmits\(\['select', 'detail', 'location', 'navigate', 'claim'\]\)/)
  assert.doesNotMatch(source, /拨打电话|复制微信|makePhoneCall|setClipboardData/)
})

test('merchant place card only shows the incomplete-location state without location actions', () => {
  assert.match(source, /hasLocation \? \(place\.distanceText \|\| '可导航到店'\) : '位置待完善'/)
  assert.doesNotMatch(source, /v-else(?!-if)[^>]*>导航<\/button>/)
})

test('merchant place card composes the shared list item and preserves map actions', () => {
  assert.match(source, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
  assert.match(source, /<MerchantListItem/)
  assert.match(source, /#leading/)
  assert.match(source, /#badge/)
  assert.match(source, /#meta/)
  assert.match(source, /#actions/)
  assert.match(source, /@activate="\$emit\('select', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('detail', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('location', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('navigate', place\)"/)
  assert.match(source, /@click\.stop="\$emit\('claim', place\)"/)
})

test('claimed merchant card exposes an explicit merchant homepage action', () => {
  assert.match(source, /v-if="hasMerchantDetail\(place\)" class="detail-button"/)
  assert.match(source, />进入主页<\/button>/)
  assert.match(source, /defineEmits\(\['select', 'detail', 'location', 'navigate', 'claim'\]\)/)
})
