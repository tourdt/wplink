import assert from 'node:assert/strict'
import test from 'node:test'
import { MERCHANT_CATEGORY_OPTIONS, mergeMerchantCategories, toggleMerchantCategory } from './merchantCategories.js'

test('shared categories are unique and preserve removable historical tags', () => {
  assert.ok(MERCHANT_CATEGORY_OPTIONS.includes('童装'))
  assert.equal(new Set(MERCHANT_CATEGORY_OPTIONS).size, MERCHANT_CATEGORY_OPTIONS.length)
  assert.deepEqual(mergeMerchantCategories(['童装', '旧分类', '童装']), ['童装', '旧分类'])
  assert.deepEqual(toggleMerchantCategory(['童装', '旧分类'], '旧分类'), ['童装'])
  assert.deepEqual(toggleMerchantCategory(['童装'], '女童装'), ['童装', '女童装'])
})
