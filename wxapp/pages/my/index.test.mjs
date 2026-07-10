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

test('my page shows compact entitlement overview and hides duplicated quota sections', () => {
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  assert.match(source, /import \{ getMerchantEntitlements \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /const merchantEntitlements = ref\(\[\]\)/)
  assert.match(source, /<view v-if="benefitOverviewVisible" class="benefit-overview-card section-card" @click="openBenefitOverview">/)
  assert.match(source, /<text class="benefit-title">我的权益<\/text>/)
  assert.match(source, /<text class="benefit-desc">\{\{ benefitOverviewDesc \}\}<\/text>/)
  assert.match(source, /<text class="benefit-action">查看<\/text>/)
  assert.match(source, /<text class="benefit-value">\{\{ publishQuotaRemaining \}\}<\/text>/)
  assert.match(source, /<text class="benefit-label">发布<\/text>/)
  assert.match(source, /<text class="benefit-value">\{\{ refreshQuotaRemaining \}\}<\/text>/)
  assert.match(source, /<text class="benefit-label">刷新<\/text>/)
  assert.match(source, /const benefitOverviewVisible = computed\(\(\) => Boolean\(isLoggedIn\.value\)\)/)
  assert.match(source, /const benefitOverviewDesc = computed/)
  assert.match(source, /function entitlementRemaining\(type\)/)
  assert.match(source, /async function loadMerchantEntitlements\(\)/)
  assert.match(source, /getMerchantEntitlements\(merchantId\.value/)
  assert.match(source, /<text class="action-title">VIP 权益<\/text>/)
  assert.match(source, /<text class="action-meta">查看额度和限时特价<\/text>/)
  assert.match(source, /async function openBenefitOverview\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*if \(activeGrowthCampaign\.value\.code\) \{[\s\S]*await openGrowthEntitlement\(\)[\s\S]*return[\s\S]*await openVIP\(\)/)
  assert.match(source, /async function openVIP\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/vip\/index\?merchantId=\$\{merchantId\.value\}` \}\)/)
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
