import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const componentPath = path.join(root, 'components/HomeRecentMerchantCard.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('recent merchant item composes the shared list style with onboarding context', () => {
  assert.equal(fs.existsSync(componentPath), true)
  assert.match(source, /import MerchantListItem from '\.\/MerchantListItem\.vue'/)
  assert.match(source, /<MerchantListItem/)
  assert.match(source, /:tags="visibleCategories"/)
  assert.match(source, /新入驻/)
  assert.match(source, /merchantTypeText/)
  assert.match(source, /formatListFreshnessDate/)
  assert.match(source, /logoUrl/)
  assert.match(source, /merchant-initial/)
  assert.match(source, /\.onboarded-date\s*\{[\s\S]*margin-left:\s*auto/)
  assert.doesNotMatch(source, /addressText/)
})

test('recent merchant item preserves its public click contract', () => {
  assert.match(source, /defineEmits\(\['open'\]\)/)
  assert.match(source, /@activate="\$emit\('open', merchant\)"/)
  assert.doesNotMatch(source, /phone|wechat|vip/i)
})
