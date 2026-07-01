import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/merchant/detail.vue')

test('merchant detail page shows a safe verification summary for verified merchants', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /class="section verification-info-section"/)
  assert.match(source, /v-if="showVerificationInfo"/)
  assert.match(source, /<text class="section-title">认证信息<\/text>/)
  assert.match(source, /平台已认证/)
  assert.match(source, /认证身份/)
  assert.match(source, /已核验/)
  assert.match(source, /主体资质、经营场地/)
  assert.match(source, /merchantVerificationReviewedDate/)
  assert.match(source, /formatDateToDay/)
  assert.doesNotMatch(source, /licenseUrl/)
  assert.doesNotMatch(source, /socialCreditCode/)
  assert.doesNotMatch(source, /businessName/)
})
