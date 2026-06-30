import assert from 'node:assert/strict'
import test from 'node:test'

import { validateMerchantName } from './merchantName.js'

test('accepts normal merchant name', () => {
  assert.equal(validateMerchantName('织里晨星童装厂'), '')
})

test('rejects misleading merchant name keywords', () => {
  for (const name of ['织里官方童装厂', '织里认证工厂', '平台推荐童装厂']) {
    assert.equal(validateMerchantName(name), '商家名称不能包含认证、官方等容易误导的字样')
  }
})

test('rejects empty merchant name', () => {
  assert.equal(validateMerchantName('  '), '请填写商家名称')
})
