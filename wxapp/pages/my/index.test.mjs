import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)

test('my page shows merchant logo and name when merchant profile exists', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ getMerchant \} from '\.\.\/\.\.\/api\/merchant'/)
  assert.match(source, /const merchantProfile = ref\(\{\}\)/)
  assert.match(source, /const merchantLogo = computed\(\(\) => merchantProfile\.value\.logoUrl \|\| ''\)/)
  assert.match(source, /const merchantName = computed\(\(\) => merchantProfile\.value\.name \|\| ''\)/)
  assert.match(source, /<image v-if="merchantLogo" class="avatar avatar-image" :src="merchantLogo" mode="aspectFill" \/>/)
  assert.match(source, /const accountName = computed\(\(\) => merchantName\.value \|\| \(isLoggedIn\.value \? '我的账号' : '未登录'\)\)/)
  assert.match(source, /await Promise\.all\(\[loadMerchantProfile\(\), loadVerificationStatus\(\), loadMerchantMetricsSummary\(\)\]\)/)
  assert.match(source, /async function loadMerchantProfile\(\)/)
  assert.match(source, /merchantProfile\.value = await getMerchant\(merchantId\.value, \{ suppressErrorToast: true \}\)/)
})

test('my page exposes native customer service entry without login gate', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /open-type="contact"/)
  assert.match(source, /<text class="action-title">联系客服<\/text>/)
  assert.match(source, /<text class="action-meta">平台问题和使用咨询<\/text>/)
  assert.doesNotMatch(source, /function openCustomerService\(\)[\s\S]*?requireLogin/)
})
