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
  assert.match(source, /await Promise\.all\(\[loadMerchantProfile\(\), loadMerchantEntitlements\(\), loadGrowthCampaigns\(\)\]\)/)
  assert.match(source, /async function loadMerchantProfile\(\)/)
  assert.match(source, /merchantProfile\.value = await getMerchant\(merchantId\.value, \{ suppressErrorToast: true \}\)/)
})

test('my page shows merchant number below account name and copies it from the whole row', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /const merchantNo = computed\(\(\) => merchantProfile\.value\.merchantNo \|\| ''\)/)
  assert.match(source, /const merchantNoVisible = computed\(\(\) => Boolean\(isLoggedIn\.value && merchantNo\.value\)\)/)
  assert.match(source, /<view v-if="merchantNoVisible" class="account-number-row" @click\.stop="copyMerchantNo">/)
  assert.match(source, /<text class="account-number-label">编号：<\/text>/)
  assert.match(source, /<text class="account-number-value">\{\{ merchantNo \}\}<\/text>/)
  assert.match(source, /<text class="account-number-copy">复制<\/text>/)
  assert.match(source, /uni\.setClipboardData\(\{[\s\S]*data: merchantNo\.value,[\s\S]*success: \(\) => \{[\s\S]*uni\.showToast\(\{ title: '编号已复制', icon: 'none' \}\)/)
  assert.doesNotMatch(source, /<text class="account-desc">\{\{ accountDesc \}\}<\/text>/)
  assert.doesNotMatch(source, /const accountDesc = computed/)
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
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善发布者资料'/)
})

test('my page shows compact entitlement overview with entitlement center access', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ getMerchantEntitlements \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /const merchantEntitlements = ref\(\[\]\)/)
  assert.match(source, /<view v-if="benefitOverviewVisible" class="benefit-overview-card section-card" @click="openBenefitOverview">/)
  assert.match(source, /<text class="benefit-title">我的权益<\/text>/)
  assert.match(source, /<text class="benefit-desc">\{\{ benefitOverviewDesc \}\}<\/text>/)
  assert.match(source, /<text class="benefit-action">查看<\/text>/)
  assert.match(source, /<text class="benefit-value">\{\{ publishQuotaDisplay \}\}<\/text>/)
  assert.match(source, /<text class="benefit-label">发布次数<\/text>/)
  assert.match(source, /<text class="benefit-value">\{\{ refreshQuotaDisplay \}\}<\/text>/)
  assert.match(source, /<text class="benefit-label">刷新次数<\/text>/)
  assert.match(source, /const benefitOverviewVisible = computed\(\(\) => Boolean\(isLoggedIn\.value\)\)/)
  assert.match(source, /const benefitOverviewDesc = computed/)
  assert.match(source, /function entitlementRemaining\(type\)/)
  assert.match(source, /async function loadMerchantEntitlements\(\)/)
  assert.match(source, /getMerchantEntitlements\(merchantId\.value/)
  assert.doesNotMatch(source, /<text class="action-title">VIP 权益<\/text>/)
  assert.doesNotMatch(source, /<text class="action-meta">查看额度和限时特价<\/text>/)
  assert.doesNotMatch(source, /function openVIP\(\)/)
  assert.match(source, /async function openBenefitOverview\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*uni\.navigateTo\(\{ url: '\/pages\/vip\/index' \}\)/)
  assert.doesNotMatch(source, /quota-summary/)
  assert.doesNotMatch(source, /entitlement-section/)
  assert.doesNotMatch(source, /免费额度/)
  assert.doesNotMatch(source, /完善资料后每月/)
  assert.doesNotMatch(source, /profileStatus === 'completed' \? 10 : 3/)
  assert.doesNotMatch(source, /getMerchantEntitlementUsageRecords/)
  assert.doesNotMatch(source, /selectedEntitlement/)
  assert.doesNotMatch(source, /使用记录/)
  assert.doesNotMatch(source, /<text class="action-title">商家认证<\/text>/)
  assert.doesNotMatch(source, /openMerchantVerification/)
  assert.doesNotMatch(source, /getLatestVerification/)
})

test('my page only highlights quota purchases after a trustworthy entitlement response', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ QUOTA_TYPE_PUBLISH, QUOTA_TYPE_REFRESH, buildQuotaPurchaseUrl \} from '\.\.\/\.\.\/common\/entitlementPurchase'/)
  assert.match(source, /const QUOTA_LOW_THRESHOLD = 2/)
  assert.match(source, /const entitlementLoadState = ref\('idle'\)/)
  assert.match(source, /const entitlementRequestId = ref\(0\)/)
  assert.match(source, /const entitlementQuotaReady = computed\(\(\) => Boolean\(isLoggedIn\.value && merchantId\.value && entitlementLoadState\.value === 'loaded'\)\)/)
  assert.match(source, /const publishQuotaDisplay = computed\(\(\) => entitlementQuotaReady\.value \? publishQuotaRemaining\.value : '--'\)/)
  assert.match(source, /const refreshQuotaDisplay = computed\(\(\) => entitlementQuotaReady\.value \? refreshQuotaRemaining\.value : '--'\)/)
  assert.match(source, /const benefitExpiryReminder = computed\(\(\) => \{[\s\S]*if \(!entitlementQuotaReady\.value\) return ''/)
  assert.match(source, /const publishQuotaLow = computed\(\(\) => publishQuotaRemaining\.value > 0 && publishQuotaRemaining\.value <= QUOTA_LOW_THRESHOLD\)/)
  assert.match(source, /const refreshQuotaLow = computed\(\(\) => refreshQuotaRemaining\.value > 0 && refreshQuotaRemaining\.value <= QUOTA_LOW_THRESHOLD\)/)
  assert.match(source, /v-if="entitlementQuotaReady && publishQuotaRemaining === 0"[\s\S]*购买发布次数/)
  assert.match(source, /v-else-if="entitlementQuotaReady && publishQuotaLow"[\s\S]*即将用完 · 去补充/)
  assert.match(source, /v-if="entitlementQuotaReady && refreshQuotaRemaining === 0"[\s\S]*购买刷新次数/)
  assert.match(source, /v-else-if="entitlementQuotaReady && refreshQuotaLow"[\s\S]*即将用完 · 去补充/)
  assert.match(source, /if \(!token\.value \|\| !merchantId\.value\) \{[\s\S]*entitlementLoadState\.value = 'unavailable'/)
  assert.match(source, /const requestId = entitlementRequestId\.value \+ 1[\s\S]*entitlementRequestId\.value = requestId/)
  assert.match(source, /entitlementLoadState\.value = 'loading'/)
  assert.match(source, /const resp = await getMerchantEntitlements\(merchantId\.value, \{ suppressErrorToast: true \}\)[\s\S]*if \(requestId !== entitlementRequestId\.value\) return[\s\S]*merchantEntitlements\.value = resp\.items \|\| \[\][\s\S]*entitlementLoadState\.value = 'loaded'/)
  assert.match(source, /catch \(err\) \{[\s\S]*if \(requestId !== entitlementRequestId\.value\) return[\s\S]*merchantEntitlements\.value = \[\][\s\S]*entitlementLoadState\.value = 'error'/)
  assert.match(source, /function openQuotaPurchase\(quotaType\)[\s\S]*buildQuotaPurchaseUrl\(quotaType\)/)
  assert.doesNotMatch(source, /暂无可领取权益/)
})

test('my page keeps messages reachable after messages leaves the tab bar', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /<text class="action-title">消息<\/text>/)
  assert.match(source, /function openMessages\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*uni\.navigateTo\(\{ url: '\/pages\/messages\/index' \}\)/)
  assert.doesNotMatch(source, /uni\.switchTab\(\{ url: '\/pages\/messages\/index' \}\)/)
})

test('my page shows nearest 7-day expiring entitlement reminder', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ formatDateToDay \} from '\.\.\/\.\.\/common\/date'/)
  assert.match(source, /const BENEFIT_EXPIRY_SOON_DAYS = 7/)
  assert.match(source, /<text v-if="benefitExpiryReminder" class="benefit-expiry">\{\{ benefitExpiryReminder \}\}<\/text>/)
  assert.match(source, /const benefitExpiryReminder = computed/)
  assert.match(source, /Date\.parse\(item\.expiresAt\)/)
  assert.match(source, /expiresAtTime - Date\.now\(\) <= BENEFIT_EXPIRY_SOON_DAYS \* 24 \* 60 \* 60 \* 1000/)
  assert.match(source, /最近到期：\$\{entitlementLabel\(nearest\.type\)\} \$\{amount\} 次，\$\{formatDateToDay\(nearest\.expiresAt, ''\)\} 到期/)
  assert.match(source, /function entitlementLabel\(type\)/)
  assert.match(source, /if \(type === 'refresh_quota'\) return '刷新次数'/)
  assert.match(source, /if \(type === 'publish_quota'\) return '发布次数'/)
})

test('my page uses growth campaign data in compact entitlement overview', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')
  const loadGrowthStart = source.indexOf('async function loadGrowthCampaigns()')
  const loadGrowthEnd = source.indexOf('\n}\n\nfunction entitlementRemaining', loadGrowthStart)
  const loadGrowthSource = source.slice(loadGrowthStart, loadGrowthEnd)

  assert.match(source, /import \{ getActiveGrowthCampaigns \} from '\.\.\/\.\.\/api\/growthCampaign'/)
  assert.match(source, /const growthCampaigns = ref\(\[\]\)/)
  assert.match(source, /const activeGrowthCampaign = computed\(\(\) => growthCampaigns\.value\[0\] \|\| \{\}\)/)
  assert.match(source, /return activeGrowthCampaign\.value\.hint \|\| '完成新手任务可获得更多发布和刷新次数'/)
  assert.match(source, /async function loadGrowthCampaigns\(\)/)
  assert.match(loadGrowthSource, /if \(!token\.value\) \{/)
  assert.doesNotMatch(loadGrowthSource, /merchantId\.value/)
  assert.match(source, /getActiveGrowthCampaigns\(\{ suppressErrorToast: true \}\)/)
  assert.match(source, /v-if="activeGrowthCampaign\.code" class="benefit-growth-action" @click\.stop="openGrowthEntitlement">免费获得<\/text>/)
  assert.match(source, /async function openGrowthEntitlement\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/my\/growth-entitlement\?merchantId=\$\{merchantId\.value\}` \}\)/)
})

test('my page hides merchant weekly effect section for now', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.doesNotMatch(source, /getMerchantMetricsSummary/)
  assert.doesNotMatch(source, /merchantMetricsSummary/)
  assert.doesNotMatch(source, /merchantEffectVisible/)
  assert.doesNotMatch(source, /merchantEffectItems/)
  assert.doesNotMatch(source, /商家本周效果/)
  assert.doesNotMatch(source, /近 7 天/)
  assert.doesNotMatch(source, /merchant-effect-card/)
  assert.doesNotMatch(source, /merchant-effect-grid/)
  assert.doesNotMatch(source, /merchant-effect-item/)
})
