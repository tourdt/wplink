import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')

test('my resources list uses only a compact cover image for item recognition', () => {
  assert.match(source, /<image class="resource-thumb" :src="item\.coverUrl \|\| DEFAULT_RESOURCE_COVER" mode="aspectFill" @error="handleResourceCoverError\(item\)" \/>/)
  assert.match(source, /\.resource-thumb \{[\s\S]*width: 112rpx;[\s\S]*height: 112rpx;/)
  assert.match(source, /\.resource-summary \{[\s\S]*display: grid;[\s\S]*grid-template-columns: 112rpx minmax\(0, 1fr\);/)
  assert.doesNotMatch(source, /width: 168rpx;[\s\S]*class="resource-thumb"/)
})

test('my resources list displays Chinese resource type text instead of raw type code', () => {
  assert.match(source, /import \{ resourceTypeLabel \} from '\.\.\/\.\.\/common\/resourceCategories'/)
  assert.match(source, /function displayResourceTypeText\(item\) \{[\s\S]*return resourceTypeLabel\(item\) \|\| item\.typeCode \|\| '发布'[\s\S]*\}/)
  assert.match(source, /\{\{ item\.category \}\} · \{\{ displayResourceTypeText\(item\) \}\}/)
  assert.doesNotMatch(source, /\{\{ item\.category \}\} · \{\{ item\.typeCode \}\}/)
})

test('my resources list distinguishes automatic safety checks from exceptional handling', () => {
  assert.match(source, /const contentAuditStatuses = new Set\(\['pending', 'manual_review', 'audit_retry'\]\)/)
  assert.match(source, /pending: '自动检测中'/)
  assert.match(source, /manual_review: '异常处理中'/)
  assert.match(source, /audit_retry: '自动检测重试中'/)
  assert.match(source, /if \(isContentAuditStatus\(item\.status\)\) return 'pending'/)
  assert.match(source, /if \(isContentAuditStatus\(item\.status\)\) return statusText\[item\.status\]/)
})

test('my resources list falls back to a default resource image', () => {
  assert.match(source, /const DEFAULT_RESOURCE_COVER = '\/static\/resource\/default-resource-cover\.png'/)
  assert.match(source, /function handleResourceCoverError\(item\) \{[\s\S]*item\.coverUrl = ''[\s\S]*\}/)
  assert.equal(fs.existsSync(path.join(root, 'static/resource/default-resource-cover.png')), true)
  assert.doesNotMatch(source, /resource-thumb-placeholder/)
  assert.doesNotMatch(source, /\{\{ item\.category \|\| item\.typeCode \}\}/)
})

test('my resources page pins status filters and uses a compact publish action', () => {
  assert.doesNotMatch(source, /resource-manager-head/)
  assert.doesNotMatch(source, /manager-title/)
  assert.doesNotMatch(source, /manager-desc/)
  assert.match(source, /<button class="publish-fab" @click="openPublish">发布<\/button>/)
  assert.match(source, /\.my-resources-page \{[\s\S]*display: flex;[\s\S]*flex-direction: column;/)
  assert.match(source, /\.my-resources-page \{[\s\S]*overflow-x: hidden;/)
  assert.match(source, /\.my-resources-page \{[\s\S]*padding-top: 132rpx;/)
  assert.match(source, /<view class="filter-panel">[\s\S]*<view class="filter-row">/)
  assert.doesNotMatch(source, /<view class="direction-row">/)
  assert.match(source, /\.filter-panel \{[\s\S]*position: fixed;[\s\S]*top: 0;[\s\S]*right: 0;[\s\S]*left: 0;[\s\S]*z-index: 10;[\s\S]*padding: 24rpx 24rpx 16rpx;[\s\S]*overflow: hidden;[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 8rpx 20rpx rgba\(15, 23, 42, 0\.06\);/)
  assert.match(source, /\.filter-button \{[\s\S]*background: #f4f7fd;/)
  assert.doesNotMatch(source, /position: sticky;/)
  assert.match(source, /\.publish-fab \{[\s\S]*position: fixed;[\s\S]*right: 24rpx;[\s\S]*bottom: calc\(32rpx \+ env\(safe-area-inset-bottom\)\);/)
})

test('my resources empty state is centered in the list display area', () => {
  assert.match(source, /<view v-if="!loading && rows\.length === 0" class="empty-state">/)
  assert.match(source, /\.empty-state \{[\s\S]*flex: 1;[\s\S]*align-content: center;[\s\S]*min-height: 420rpx;/)
  assert.match(source, /<text class="empty-desc">发布后可查看审核和数据。<\/text>/)
  assert.match(source, /\.empty-state \{[\s\S]*padding-bottom: 88rpx;/)
  assert.doesNotMatch(source, /发布供应或需求后，可在这里查看审核进度、曝光数据和推广效果。/)
})

test('my resources removes supply and demand direction tabs from list filtering', () => {
  for (const token of [
    'directionOptions',
    '全部发布',
    '供应发布',
    '需求发布',
    'selectDirection',
    'RESOURCE_DIRECTION_DEMAND',
    'direction: filters.direction',
    'direction-row',
    'direction-button',
  ]) {
    assert.doesNotMatch(source, new RegExp(token))
  }

  assert.match(source, /const filters = reactive\(\{ status: '' \}\)/)
  assert.match(source, /listMyResources\(\{ merchantId: merchantId\.value, status: filters\.status, page: nextPage, pageSize \}\)/)
})

test('my resources card actions avoid a visible toolbar frame', () => {
  assert.match(source, /<button v-if="isActivePublished\(item\)" class="primary-action" @click="refresh\(item\)">刷新<\/button>/)
  assert.match(source, /\.action-row \{[\s\S]*gap: 10rpx;[\s\S]*padding-top: 14rpx;[\s\S]*border-top: 1rpx solid #eef2f7;/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*border-radius:/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*background:/)
  assert.doesNotMatch(source, /\.action-row \{[^}]*box-shadow:/)
  assert.match(source, /\.action-row button \{[\s\S]*min-width: 108rpx;[\s\S]*height: 62rpx;[\s\S]*border: 1rpx solid \$wplink-line;[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 2rpx 4rpx rgba\(15, 23, 42, 0\.04\);/)
  assert.match(source, /\.action-row button::after \{[\s\S]*border: 0;/)
  assert.match(source, /\.action-row \.primary-action \{[\s\S]*border-color: #b8c4d4;[\s\S]*background: \$wplink-primary-soft;[\s\S]*color: \$wplink-primary;/)
  assert.match(source, /\.action-row \.danger-button \{[\s\S]*border-color: #fecdd3;[\s\S]*background: #fff8f8;/)
  assert.doesNotMatch(source, /\.action-row button \{[\s\S]*background: #edf2f7;/)
  assert.doesNotMatch(source, /\.action-row \.primary-action \{[\s\S]*background: \$wplink-primary;/)
})

test('my resources hides the date row when no publish or expiry date exists', () => {
  assert.match(source, /<text v-if="shouldShowResourceDates\(item\)" class="resource-meta">发布 \{\{ displayDateOrPlaceholder\(item\.publishedAt\) \}\} · 到期 \{\{ displayDateOrPlaceholder\(item\.expiresAt\) \}\}<\/text>/)
  assert.match(source, /function shouldShowResourceDates\(item\) \{[\s\S]*return Boolean\(item\.publishedAt \|\| item\.expiresAt\)[\s\S]*\}/)
  assert.match(source, /function displayDateOrPlaceholder\(value\) \{[\s\S]*return value \? formatDateToDay\(value\) : '-'[\s\S]*\}/)
  assert.doesNotMatch(source, /<text class="resource-meta">发布 \{\{ formatDateToDay\(item\.publishedAt\) \}\} · 到期 \{\{ formatDateToDay\(item\.expiresAt\) \}\}<\/text>/)
})

test('my resources publish action opens the publish type selection tab', () => {
  assert.match(source, /async function openPublish\(\) \{[\s\S]*if \(!\(await ensurePageMerchantProfile\(\)\)\) return[\s\S]*uni\.removeStorageSync\('wplink_pending_publish_type_code'\)[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)[\s\S]*\}/)
  assert.doesNotMatch(source, /async function openPublish\(\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}` \}\)/)
})

test('my resources prompts for merchant profile before list and publish actions', () => {
  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /async function ensurePageMerchantProfile\(\) \{[\s\S]*if \(await ensureMerchantProfileReady\(merchantId\.value\)\) return true[\s\S]*rows\.value = \[\][\s\S]*return false[\s\S]*\}/)
  assert.match(source, /async function loadRows\(\{ reset = true \} = \{\}\) \{[\s\S]*if \(!\(await ensurePageMerchantProfile\(\)\)\) return[\s\S]*const resp = await listMyResources/)
  assert.match(source, /async function openPublish\(\) \{[\s\S]*if \(!\(await ensurePageMerchantProfile\(\)\)\) return[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)/)
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善发布者资料'/)
})

test('my resources reconciles merchant id with current account before listing', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(source, /import \{ getMerchantId, saveMerchantId \} from '\.\.\/\.\.\/store\/session'/)
  assert.match(source, /import \{ getMe \} from '\.\.\/\.\.\/api\/auth'/)
  assert.match(source, /const resolvedMerchantId = await resolveManagedMerchantId\(\)/)
  assert.match(source, /const me = await getMe\(\{ suppressErrorToast: true \}\)/)
  assert.match(source, /const matchedMerchant = managedMerchants\.find\(\(item\) => normalizeMerchantId\(item\.id\) === candidateMerchantId\)/)
  assert.match(source, /const selectedMerchant = matchedMerchant \|\| managedMerchants\[0\]/)
  assert.match(source, /saveMerchantId\(resolvedMerchantId\)/)
  assert.match(source, /路由或旧缓存不匹配时同步修正/)
})

test('my resources restores top voucher actions for merchants', () => {
  assert.match(source, /import \{ listTopVouchers, redeemTopVoucher \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /import \{ createQuotaPackOrder, createVIPPayment, listQuotaPacks \} from '\.\.\/\.\.\/api\/vip'/)
  assert.match(source, /@click="topResource\(item\)"/)
  assert.match(source, /async function topResource\(item\)/)
  assert.match(source, /listTopVouchers\(merchantId\.value\)/)
  assert.match(source, /redeemTopVoucher\(voucher\.id, item\.id, merchantId\.value\)/)
  assert.match(source, /async function purchaseTopService\(item\)/)
  assert.match(source, /createQuotaPackOrder\(merchantId\.value, pack\.code, \{ resourceId: item\.id \}\)/)
  assert.match(source, /createVIPPayment\(merchantId\.value, order\.orderId\)/)
  assert.match(source, /购买置顶服务/)
  assert.match(source, /saleLabel: '置顶 1 天'/)
  assert.match(source, /function topServiceSaleLabel\(item\)/)
  assert.match(source, /function topServicePurchaseText\(item\)/)
  assert.match(source, /function isTopServiceDiscounted\(item\)/)
  assert.match(source, /topServicePurchaseText\(item\)\.join\(' · '\)/)
  assert.match(source, /topServicePurchaseText\(pack\)\.join\('，'\)/)
  assert.doesNotMatch(source, /暂无可用置顶券，请先购买/)
  assert.doesNotMatch(source, /tab=top/)
})

test('my resources directs refresh quota shortages to purchase', () => {
  assert.match(source, /import \{ QUOTA_TYPE_REFRESH, buildQuotaPurchaseUrl, confirmQuotaPurchase \} from '\.\.\/\.\.\/common\/entitlementPurchase'/)
  assert.match(source, /async function handleRefreshQuotaError\(err\) \{[\s\S]*err\?\.code !== 'QUOTA_NOT_ENOUGH'[\s\S]*return false[\s\S]*await confirmQuotaPurchase\(QUOTA_TYPE_REFRESH\)[\s\S]*uni\.navigateTo\(\{ url: buildQuotaPurchaseUrl\(QUOTA_TYPE_REFRESH\) \}\)[\s\S]*return true[\s\S]*\}/)
  assert.match(source, /async function refresh\(item\) \{[\s\S]*try \{[\s\S]*catch \(err\) \{[\s\S]*if \(await handleRefreshQuotaError\(err\)\) return[\s\S]*throw err[\s\S]*\}/)
})
