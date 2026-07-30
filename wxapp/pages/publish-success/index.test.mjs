import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/publish-success/index.vue'), 'utf8')

test('publish success page prompts for merchant profile before opening my resources', () => {
  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /async function openMyResources\(\) \{[\s\S]*const merchantId = getMerchantId\(\)[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\)\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/my-resources\/index\?merchantId=\$\{merchantId\}` \}\)[\s\S]*\}/)
})

test('publish success page switches copy for demand submissions', () => {
  assert.match(source, /import \{ onLoad \} from '@dcloudio\/uni-app'/)
  assert.match(source, /const RESOURCE_DIRECTION_DEMAND = 'demand'/)
  assert.match(source, /onLoad\(\(options = \{\}\) => \{[\s\S]*publishDirection\.value = normalizePublishDirection\(options\.direction\)[\s\S]*resultStatus\.value = normalizeResultStatus\(options\.status\)[\s\S]*\}\)/)
  assert.match(source, /<text class="success-title">\{\{ successCopy\.title \}\}<\/text>/)
  assert.match(source, /需求已提交/)
  assert.match(source, /系统正在自动安全检测/)
  assert.match(source, /搜索、推荐和需求列表/)
})

test('publish success page uses supply copy instead of resource copy', () => {
  assert.match(source, /title: '供应已提交'/)
  assert.match(source, /系统正在自动安全检测/)
  assert.doesNotMatch(source, /供应已提交审核/)
})

test('resource publish form passes direction to success page', () => {
  const formSource = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.match(formSource, /function openPublishSuccess\(result = \{\}\) \{[\s\S]*const publishDirection = normalizePublishDirection\(form\.direction\) \|\| RESOURCE_DIRECTION_SUPPLY[\s\S]*`status=\$\{encodeURIComponent\(result\.status \|\| 'pending'\)\}`[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish-success\/index\?\$\{query\}` \}\)[\s\S]*\}/)
  assert.match(formSource, /const resp = await submitResource\(editingResourceId\.value, form\.merchantId\)[\s\S]*openPublishSuccess\(resp\)/)
  assert.match(formSource, /const images = await uploadPendingResourceImages\(\)[\s\S]*const resp = await createResource\(buildResourcePublishPayload\(images\)\)[\s\S]*openPublishSuccess\(resp\)/)
})

test('publish success page normalizes automatic safety states for users', () => {
  assert.match(source, /<text class="result-value">\{\{ auditResultText \}\}<\/text>/)
  assert.match(source, /resultStatus\.value === 'pending'\) return '自动检测中'/)
  assert.match(source, /resultStatus\.value === 'audit_retry'\) return '自动检测重试中'/)
  assert.match(source, /resultStatus\.value === 'manual_review'\) return '异常处理中'/)
})
