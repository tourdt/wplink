import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const componentPath = path.join(root, 'components/HomeRecentMerchantCard.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('recent merchant card shows public onboarding identity and compact business context', () => {
  assert.equal(fs.existsSync(componentPath), true, 'HomeRecentMerchantCard.vue should exist')
  assert.match(source, /新入驻/)
  assert.match(source, /merchantTypeText/)
  assert.match(source, /mainCategories/)
  assert.match(source, /formatListFreshnessDate/)
  assert.match(source, /入驻/)
  assert.match(source, /addressText/)
  assert.match(source, /logoUrl/)
  assert.match(source, /merchant-initial/)
})

test('recent merchant card emits open and does not expose contact or vip data', () => {
  assert.match(source, /defineEmits\(\['open'\]\)/)
  assert.match(source, /\$emit\('open', merchant\)/)
  assert.doesNotMatch(source, /phone/i)
  assert.doesNotMatch(source, /wechat/i)
  assert.doesNotMatch(source, /vip/i)
})
