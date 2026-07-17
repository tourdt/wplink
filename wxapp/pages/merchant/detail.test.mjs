import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/merchant/detail.vue')

test('merchant detail page does not show verification wording', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, />认证</)
  assert.doesNotMatch(source, /已认证/)
  assert.doesNotMatch(source, /已核实/)
  assert.doesNotMatch(source, /核验项/)
  assert.doesNotMatch(source, /主体资质、经营场地/)
  assert.doesNotMatch(source, /有效期/)
  assert.doesNotMatch(source, /showVerificationInfo/)
  assert.doesNotMatch(source, /merchantVerification/)
  assert.doesNotMatch(source, /verification-info/)
  assert.doesNotMatch(source, /formatDateToDay/)
  assert.doesNotMatch(source, /licenseUrl/)
  assert.doesNotMatch(source, /socialCreditCode/)
  assert.doesNotMatch(source, /businessName/)
})
