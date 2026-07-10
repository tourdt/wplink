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
  assert.match(source, /await Promise\.all\(\[loadMerchantProfile\(\), loadMerchantMetricsSummary\(\), loadMerchantEntitlements\(\)\]\)/)
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

test('my page prompts before opening merchant-only entries without profile', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /async function openMyResources\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/my-resources\/index\?merchantId=\$\{merchantId\.value\}` \}\)/)
  assert.match(source, /async function openMerchantHome\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/merchant\/detail\?id=\$\{merchantId\.value\}` \}\)/)
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善商家资料'/)
})

test('my page shows remaining quota summary and hides merchant verification main path', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ getMerchantEntitlements, getMerchantEntitlementUsageRecords \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /const merchantEntitlements = ref\(\[\]\)/)
  assert.match(source, /const selectedEntitlement = ref\(null\)/)
  assert.match(source, /const entitlementUsageRecords = ref\(\[\]\)/)
  assert.match(source, /<view v-if="quotaSummaryVisible" class="quota-summary">/)
  assert.match(source, /<text class="quota-title">\{\{ quotaSummaryTitle \}\}<\/text>/)
  assert.match(source, /<text class="quota-desc">\{\{ quotaSummaryDesc \}\}<\/text>/)
  assert.match(source, /<view v-if="entitlementCards\.length" class="entitlement-section section-card">/)
  assert.match(source, /<text class="section-title">权益余额<\/text>/)
  assert.match(source, /<text class="detail-label">过期时间<\/text>/)
  assert.match(source, /<text class="usage-title">使用记录<\/text>/)
  assert.match(source, /const quotaSummaryTitle = computed/)
  assert.match(source, /const quotaSummaryDesc = computed/)
  assert.doesNotMatch(source, /const topVoucherRemaining = computed\(\(\) => entitlementRemaining\('top_voucher'\)\)/)
  assert.match(source, /const entitlementCards = computed\(\(\) => visibleEntitlements\.value\.map/)
  assert.match(source, /const visibleEntitlements = computed\(\(\) => merchantEntitlements\.value\.filter\(\(item\) => item\.type !== 'top_voucher'\)\)/)
  assert.match(source, /function entitlementRemaining\(type\)/)
  assert.match(source, /async function loadMerchantEntitlements\(\)/)
  assert.match(source, /async function openEntitlementDetail\(item\)/)
  assert.match(source, /getMerchantEntitlementUsageRecords\(merchantId\.value, item\.id, \{ suppressErrorToast: true \}\)/)
  assert.match(source, /function usageActionText\(actionType\)/)
  assert.match(source, /getMerchantEntitlements\(merchantId\.value/)
  assert.match(source, /<text class="action-title">VIP 权益<\/text>/)
  assert.match(source, /<text class="action-meta">查看额度和限时特价<\/text>/)
  assert.match(source, /return `VIP 权益：发布 \$\{publishQuotaRemaining\.value\} 条 · 刷新 \$\{refreshQuotaRemaining\.value\} 次`/)
  assert.match(source, /return '置顶功能暂未开放，当前可使用发布和刷新权益'/)
  assert.match(source, /async function openVIP\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/vip\/index\?merchantId=\$\{merchantId\.value\}` \}\)/)
  assert.doesNotMatch(source, /<text class="action-title">商家认证<\/text>/)
  assert.doesNotMatch(source, /openMerchantVerification/)
  assert.doesNotMatch(source, /getLatestVerification/)
})
