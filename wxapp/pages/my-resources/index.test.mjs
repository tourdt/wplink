import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

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
  assert.match(source, /async function loadRowsOnce\(\{ reset = true \} = \{\}\) \{[\s\S]*if \(!\(await ensurePageMerchantProfile\(\)\)\) return[\s\S]*const resp = await listMyResources/)
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

function ref(value) {
  return { value }
}

function reactive(value) {
  return value
}

function computed(getter) {
  return { get value() { return getter() } }
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function flushAsyncWork() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function resourceActionButton(clickExpression) {
  const { descriptor } = parse(source, { filename: 'pages/my-resources/index.vue' })
  const buttons = descriptor.template.content.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) || []
  const button = buttons.find((item) => item.includes(`@click="${clickExpression}"`))
  assert.ok(button, `应找到 ${clickExpression} 操作按钮`)
  return button
}

async function renderResourceActionButton(clickExpression, context) {
  const renderContext = {
    isActivePublished: () => true,
    canRepost: () => true,
    canDeleteTakenDown: () => true,
    ...context,
  }
  if (!renderContext.isResourceActionLoading) {
    renderContext.isResourceActionLoading = (item, action) => renderContext.isResourceAction(item, action)
  }
  if (!renderContext.resourceActionLabel) {
    renderContext.resourceActionLabel = (item, action, idleLabel, writingLabel) => (
      renderContext.isResourceAction(item, action) ? writingLabel : idleLabel
    )
  }
  const compiled = compileTemplate({
    source: resourceActionButton(clickExpression),
    filename: 'pages/my-resources/index.vue',
    id: 'my-resource-action-loading',
    compilerOptions: { mode: 'function' },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const Button = {
    props: Object.keys(renderContext),
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({ render: () => h(Button, renderContext) }))
}

function openingButtonTag(html) {
  return html.match(/<button\b[^>]*>/)?.[0] || ''
}

function assertBusyResourceButton(html, label) {
  const openingTag = openingButtonTag(html)
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, `${label}进行中应禁用`)
  assert.match(openingTag, /\bloading(?:=|\s|>)/, `${label}进行中应展示原生 loading`)
  assert.match(html, new RegExp(label), `进行中按钮应展示${label}`)
}

function assertLockedResourceButton(html, label) {
  const openingTag = openingButtonTag(html)
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, `${label}在另一写操作进行时应禁用`)
  assert.doesNotMatch(openingTag, /\bloading="true"/, `${label}不应显示另一操作的 loading`)
}

function assertResourceAction(page, resourceId, action, label) {
  assert.equal(page.resourceAction.value.resourceId, resourceId, `${label}应记录对应资源`)
  assert.equal(page.resourceAction.value.action, action, `${label}应记录对应动作`)
}

function assertResourceActionCleared(page, label) {
  assertResourceAction(page, '', '', label)
  assert.equal(page.resourceActionPhase.value, '', `${label}应清空阶段状态`)
}

test('my resource write buttons render only the active action as busy and preserve details', async () => {
  const noop = () => {}
  const activeItem = { id: 'resource-1', status: 'published' }
  const repostItem = { id: 'resource-1', status: 'expired' }
  const deletedItem = { id: 'resource-1', status: 'taken_down' }
  const actionContracts = [
    { action: 'refresh', click: 'refresh(item)', item: activeItem, busyText: '刷新中' },
    { action: 'top', click: 'topResource(item)', item: activeItem, busyText: '置顶中' },
    { action: 'takeDown', click: 'takeDown(item)', item: activeItem, busyText: '下架中' },
    { action: 'repost', click: 'repost(item)', item: repostItem, busyText: '准备中' },
    { action: 'delete', click: 'deleteTakenDown(item)', item: deletedItem, busyText: '删除中' },
  ]
  const page = loadMyResourcesPage()

  for (const contract of actionContracts) {
    page.resourceAction.value = { resourceId: 'resource-1', action: contract.action }
    const html = await renderResourceActionButton(contract.click, {
      item: contract.item,
      resourceActionBusy: true,
      isResourceAction: page.isResourceAction,
      isResourceActionLoading: page.isResourceActionLoading,
      resourceActionLabel: (item, action, idleLabel, writingLabel) => page.isResourceAction(item, action) ? writingLabel : idleLabel,
      refresh: noop,
      topResource: noop,
      takeDown: noop,
      repost: noop,
      deleteTakenDown: noop,
    })
    assertBusyResourceButton(html, contract.busyText)

    const otherResourceHtml = await renderResourceActionButton(contract.click, {
      item: { ...contract.item, id: 'resource-2' },
      resourceActionBusy: true,
      isResourceAction: page.isResourceAction,
      isResourceActionLoading: page.isResourceActionLoading,
      resourceActionLabel: (item, action, idleLabel, writingLabel) => page.isResourceAction(item, action) ? writingLabel : idleLabel,
      refresh: noop,
      topResource: noop,
      takeDown: noop,
      repost: noop,
      deleteTakenDown: noop,
    })
    assertLockedResourceButton(otherResourceHtml, contract.busyText)
  }

  const detailsHtml = await renderResourceActionButton('openResource(item)', {
    item: activeItem,
    resourceActionBusy: true,
    openResource: noop,
  })
  assert.doesNotMatch(openingButtonTag(detailsHtml), /\bdisabled(?:=|\s|>)/, '写操作进行时仍可查看详情')
})

function loadMyResourcesPage(additions = {}) {
  const { descriptor } = parse(source, { filename: 'pages/my-resources/index.vue' })
  const script = descriptor.scriptSetup.content.replace(/^import\s+[\s\S]*?\s+from\s+['"][^'"]+['"]\s*$/gm, '')
  const sandbox = {
    console,
    Promise,
    Set,
    Date,
    String,
    Number,
    Boolean,
    Math,
    ref,
    reactive,
    computed,
    onLoad: () => {},
    onPullDownRefresh: () => {},
    onReachBottom: () => {},
    requireLogin: () => true,
    ensureMerchantProfileReady: async () => true,
    getMerchantId: () => '',
    saveMerchantId: () => {},
    getMe: async () => ({ managedMerchants: [{ id: 'merchant-1' }] }),
    listTopVouchers: async () => ({ items: [] }),
    redeemTopVoucher: async () => ({}),
    deleteTakenDownResource: async () => ({}),
    getOwnResource: async () => ({}),
    listMyResources: async () => ({ items: [], total: 0 }),
    refreshResource: async () => ({}),
    takeDownResource: async () => ({}),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-1' }),
    createVIPPayment: async () => ({ status: 'paid' }),
    listQuotaPacks: async () => ({ items: [{ code: 'top_1d', benefits: { topVoucherCount: 1, topDurationHours: 24 } }] }),
    formatDateToDay: (value) => value,
    resourceTypeLabel: () => '',
    QUOTA_TYPE_REFRESH: 'refresh',
    buildQuotaPurchaseUrl: () => '/pages/vip/index',
    confirmQuotaPurchase: async () => false,
    uni: {
      showToast: () => {},
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
      requestPayment: ({ success }) => success({}),
      navigateTo: () => {},
      setStorageSync: () => {},
      removeStorageSync: () => {},
      switchTab: () => {},
      stopPullDownRefresh: () => {},
    },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.myResourcesPage = { merchantId, rows, resourceAction, resourceActionPhase: typeof resourceActionPhase === 'undefined' ? undefined : resourceActionPhase, resourceActionBusy, isResourceAction, isResourceActionLoading: typeof isResourceActionLoading === 'undefined' ? undefined : isResourceActionLoading, resourceActionLabel: typeof resourceActionLabel === 'undefined' ? undefined : resourceActionLabel, loadRows, refresh, topResource, takeDown, repost, deleteTakenDown }`, sandbox, { filename: 'pages/my-resources/index.vue' })
  return sandbox.myResourcesPage
}

test('my resource writes share one lock and clear it after success or rejection', async () => {
  const refreshRequest = deferred()
  let refreshCalls = 0
  let takeDownCalls = 0
  let repostCalls = 0
  const page = loadMyResourcesPage({
    refreshResource: () => {
      refreshCalls += 1
      return refreshRequest.promise
    },
    takeDownResource: async () => { takeDownCalls += 1 },
    getOwnResource: async () => { repostCalls += 1; return {} },
  })
  page.merchantId.value = 'merchant-1'
  const publishedItem = { id: 'resource-1', status: 'published' }
  const expiredItem = { id: 'resource-2', status: 'expired' }
  const firstRefresh = page.refresh(publishedItem)
  const duplicateRefresh = page.refresh(publishedItem)
  const concurrentTakeDown = page.takeDown(publishedItem)
  const concurrentRepost = page.repost(expiredItem)

  assert.equal(refreshCalls, 1, '连续刷新只应发起一次服务端请求')
  assert.equal(takeDownCalls, 0, '刷新期间不应下架其他资源')
  assert.equal(repostCalls, 0, '刷新期间不应读取其他资源以再发')
  assertResourceAction(page, 'resource-1', 'refresh', '刷新期间')
  refreshRequest.resolve({})
  await Promise.all([firstRefresh, duplicateRefresh, concurrentTakeDown, concurrentRepost])
  assertResourceActionCleared(page, '刷新成功和列表刷新后')

  const failedPage = loadMyResourcesPage({ refreshResource: async () => { throw new Error('网络异常') } })
  failedPage.merchantId.value = 'merchant-1'
  await assert.rejects(failedPage.refresh(publishedItem), /网络异常/)
  assertResourceActionCleared(failedPage, '刷新失败后')
})

test('top entitlement flow stays busy from voucher lookup through redemption and list refresh', async () => {
  const voucherRequest = deferred()
  const redemptionRequest = deferred()
  let voucherCalls = 0
  let redemptionCalls = 0
  let refreshCalls = 0
  const page = loadMyResourcesPage({
    listTopVouchers: () => {
      voucherCalls += 1
      return voucherRequest.promise
    },
    redeemTopVoucher: () => {
      redemptionCalls += 1
      return redemptionRequest.promise
    },
    refreshResource: async () => { refreshCalls += 1 },
  })
  page.merchantId.value = 'merchant-1'
  const item = { id: 'resource-1', status: 'published' }
  const firstTop = page.topResource(item)
  const duplicateTop = page.topResource(item)
  const concurrentRefresh = page.refresh(item)

  assert.equal(voucherCalls, 1, '置顶权益查询不应重复发起')
  assert.equal(refreshCalls, 0, '权益查询期间不应开始其他写操作')
  assertResourceAction(page, 'resource-1', 'top', '权益查询阶段')
  voucherRequest.resolve({ items: [{ id: 'voucher-1', remainingAmount: 1, topDurationHours: 24 }] })
  await flushAsyncWork()
  assert.equal(redemptionCalls, 1, '确认使用权益后应兑换一次')
  assertResourceAction(page, 'resource-1', 'top', '兑换和刷新期间')
  redemptionRequest.resolve({})
  await Promise.all([firstTop, duplicateTop, concurrentRefresh])
  assertResourceActionCleared(page, '置顶完成并刷新列表后')
})

test('top purchase flow keeps the same lock through order creation, payment and list refresh', async () => {
  const orderRequest = deferred()
  let voucherCalls = 0
  let orderCalls = 0
  let refreshCalls = 0
  const page = loadMyResourcesPage({
    listTopVouchers: async () => {
      voucherCalls += 1
      return { items: [] }
    },
    createQuotaPackOrder: () => {
      orderCalls += 1
      return orderRequest.promise
    },
    refreshResource: async () => { refreshCalls += 1 },
  })
  page.merchantId.value = 'merchant-1'
  const item = { id: 'resource-1', status: 'published' }
  const firstTop = page.topResource(item)
  const duplicateTop = page.topResource(item)
  const concurrentRefresh = page.refresh(item)

  await flushAsyncWork()
  assert.equal(voucherCalls, 1, '购买置顶前的权益查询不应重复发起')
  assert.equal(orderCalls, 1, '确认购买后只应创建一次置顶订单')
  assert.equal(refreshCalls, 0, '购买置顶期间不应开始其他写操作')
  assertResourceAction(page, 'resource-1', 'top', '创建置顶订单和支付期间')
  orderRequest.resolve({ orderId: 'top-order-1' })
  await Promise.all([firstTop, duplicateTop, concurrentRefresh])
  assertResourceActionCleared(page, '支付完成并刷新列表后')
})

test('top purchase cancellation preserves its toast and clears the resource action', async () => {
  const toasts = []
  const page = loadMyResourcesPage({
    listTopVouchers: async () => ({ items: [] }),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-1' }),
    createVIPPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'n', package: 'p', paySign: 's' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      showModal: ({ success }) => success({ confirm: true }),
      requestPayment: ({ fail }) => fail(new Error('用户取消支付')),
    },
  })
  page.merchantId.value = 'merchant-1'

  await page.topResource({ id: 'resource-1', status: 'published' })

  assert.equal(toasts.at(-1)?.title, '用户取消支付', '支付取消应保留原有友好提示')
  assertResourceActionCleared(page, '支付取消后')
})

test('top purchase payment failures use a friendly toast and clear the resource action', async () => {
  const toasts = []
  const page = loadMyResourcesPage({
    listTopVouchers: async () => ({ items: [] }),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-1' }),
    createVIPPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'n', package: 'p', paySign: 's' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      showModal: ({ success }) => success({ confirm: true }),
      requestPayment: ({ fail }) => fail({ errMsg: 'requestPayment:fail system error' }),
    },
  })
  page.merchantId.value = 'merchant-1'

  await page.topResource({ id: 'resource-1', status: 'published' })

  assert.equal(toasts.at(-1)?.title, '置顶服务购买失败，请稍后重试', '支付系统错误不应直接暴露给用户')
  assertResourceActionCleared(page, '支付失败后')
})

test('write refresh waits for an in-flight list request before replacing rows and unlocking', async () => {
  const oldListRequest = deferred()
  const refreshedListRequest = deferred()
  let listCalls = 0
  const page = loadMyResourcesPage({
    listMyResources: () => {
      listCalls += 1
      return listCalls === 1 ? oldListRequest.promise : refreshedListRequest.promise
    },
  })
  page.merchantId.value = 'merchant-1'
  const initialLoad = page.loadRows({ reset: true })
  await flushAsyncWork()
  assert.equal(listCalls, 1, '页面已有列表请求应先保持进行')

  const refresh = page.refresh({ id: 'resource-1', status: 'published' })
  await flushAsyncWork()
  assertResourceAction(page, 'resource-1', 'refresh', '写操作等待旧列表请求时')
  assert.equal(listCalls, 1, '写操作完成前不应与旧列表请求并发刷新')

  oldListRequest.resolve({ items: [{ id: 'stale-resource' }], total: 1 })
  await initialLoad
  await flushAsyncWork()
  assert.equal(listCalls, 2, '旧请求结束后必须补发一次强制刷新')
  assertResourceAction(page, 'resource-1', 'refresh', '写后强制刷新进行时')

  refreshedListRequest.resolve({ items: [{ id: 'fresh-resource' }], total: 1 })
  await refresh
  assert.equal(page.rows.value[0]?.id, 'fresh-resource', '写后刷新结果不能被旧列表响应覆盖')
  assertResourceActionCleared(page, '写后强制刷新完成后')
})

test('delete waits for confirmation before becoming busy and unlocks after deletion', async () => {
  let deleteCalls = 0
  const cancelledPage = loadMyResourcesPage({
    uni: { showModal: ({ success }) => success({ confirm: false }) },
    deleteTakenDownResource: async () => { deleteCalls += 1 },
  })
  cancelledPage.merchantId.value = 'merchant-1'
  const item = { id: 'resource-1', status: 'taken_down' }
  await cancelledPage.deleteTakenDown(item)
  assert.equal(deleteCalls, 0, '取消删除确认不应调用服务端')
  assertResourceActionCleared(cancelledPage, '取消删除确认后')

  const deletionRequest = deferred()
  const confirmedPage = loadMyResourcesPage({
    deleteTakenDownResource: () => {
      deleteCalls += 1
      return deletionRequest.promise
    },
  })
  confirmedPage.merchantId.value = 'merchant-1'
  const deletion = confirmedPage.deleteTakenDown(item)
  assertResourceAction(confirmedPage, '', '', '确认弹窗尚未结束时')
  await flushAsyncWork()
  assert.equal(deleteCalls, 1, '确认删除后应调用一次服务端')
  assertResourceAction(confirmedPage, 'resource-1', 'delete', '确认删除后')
  deletionRequest.resolve({})
  await deletion
  assertResourceActionCleared(confirmedPage, '删除完成和列表刷新后')
})

test('my resource top keeps its write lock while querying and prompting without showing a write spinner', async () => {
  const voucherRequest = deferred()
  let modalCallbacks
  let redeemCalls = 0
  let voucherCalls = 0
  const page = loadMyResourcesPage({
    listTopVouchers: () => { voucherCalls += 1; return voucherRequest.promise },
    redeemTopVoucher: async () => { redeemCalls += 1 },
    uni: {
      showModal: (options) => { modalCallbacks = options },
      showToast: () => {},
    },
  })
  page.merchantId.value = 'merchant-1'
  const item = { id: 'resource-1', status: 'published' }

  const top = page.topResource(item)
  const duplicateQuerying = page.topResource(item)
  assert.equal(page.resourceActionPhase?.value, 'querying', '权益查询期间应进入查询阶段')
  assert.equal(voucherCalls, 1, '查询阶段连续点击不应重复查询权益')
  const queryingHtml = await renderResourceActionButton('topResource(item)', {
    item,
    resourceActionBusy: true,
    isResourceAction: page.isResourceAction,
    isResourceActionLoading: page.isResourceActionLoading,
    resourceActionLabel: page.resourceActionLabel,
    topResource: () => {},
  })
  assertBusyResourceButton(queryingHtml, '查询中')

  voucherRequest.resolve({ items: [{ id: 'voucher-1', remainingAmount: 1, topDurationHours: 24 }] })
  await flushAsyncWork()
  assert.equal(page.resourceActionPhase?.value, 'prompting', '确认使用权益时应保持锁但切换为确认阶段')
  assert.equal(redeemCalls, 0, '用户确认前不应核销置顶券')
  await page.topResource(item)
  assert.equal(voucherCalls, 1, '确认阶段连续点击不应重复查询权益')
  const promptingHtml = await renderResourceActionButton('topResource(item)', {
    item,
    resourceActionBusy: true,
    isResourceAction: page.isResourceAction,
    isResourceActionLoading: page.isResourceActionLoading,
    resourceActionLabel: page.resourceActionLabel,
    topResource: () => {},
  })
  assert.match(openingButtonTag(promptingHtml), /\bdisabled(?:=|\s|>)/, '确认期间按钮应保持禁用')
  assert.doesNotMatch(openingButtonTag(promptingHtml), /\bloading="true"/, '确认期间不应显示置顶写入动画')
  assert.match(promptingHtml, />置顶</, '确认期间应恢复空闲文案')

  modalCallbacks.success({ confirm: false })
  await Promise.all([top, duplicateQuerying])
  assert.equal(redeemCalls, 0, '取消确认后不得进入写入阶段')
  assertResourceActionCleared(page, '取消确认后')
})

test('my resource top purchase keeps query feedback for vouchers and packs, then writes only after purchase confirmation', async () => {
  const voucherRequest = deferred()
  const packsRequest = deferred()
  const orderRequest = deferred()
  let actionSheetCallbacks
  let modalCallbacks
  const orderCalls = []
  const page = loadMyResourcesPage({
    listTopVouchers: () => voucherRequest.promise,
    listQuotaPacks: () => packsRequest.promise,
    createQuotaPackOrder: (...args) => { orderCalls.push(args); return orderRequest.promise },
    uni: {
      showActionSheet: (options) => { actionSheetCallbacks = options },
      showModal: (options) => { modalCallbacks = options },
      showToast: () => {},
    },
  })
  page.merchantId.value = 'merchant-1'
  const item = { id: 'resource-1', status: 'published' }

  const top = page.topResource(item)
  assert.equal(page.resourceActionPhase?.value, 'querying', '查询置顶券期间应展示查询反馈')
  voucherRequest.resolve({ items: [] })
  await flushAsyncWork()
  assert.equal(page.resourceActionPhase?.value, 'querying', '查询可购套餐期间仍应展示查询反馈')
  packsRequest.resolve({ items: [
    { code: 'top_1d', benefits: { topVoucherCount: 1, topDurationHours: 24 }, name: '1天置顶服务', salePriceCent: 100 },
    { code: 'top_3d', benefits: { topVoucherCount: 1, topDurationHours: 72 }, name: '3天置顶服务', salePriceCent: 200 },
  ] })
  await flushAsyncWork()
  assert.equal(page.resourceActionPhase?.value, 'prompting', '选择套餐时应保持锁但停止旋转')
  actionSheetCallbacks.success({ tapIndex: 0 })
  await flushAsyncWork()
  assert.equal(page.resourceActionPhase?.value, 'prompting', '购买确认弹窗期间不应提前进入写入')
  assert.equal(orderCalls.length, 0, '购买确认前不得创建订单')
  modalCallbacks.success({ confirm: true })
  await flushAsyncWork()
  assert.equal(page.resourceActionPhase?.value, 'writing', '确认购买后应进入写入阶段')
  assert.equal(orderCalls.length, 1, '确认购买后只应创建一笔订单')
  await page.topResource(item)
  assert.equal(orderCalls.length, 1, '写入阶段连续点击不应重复创建订单')
  orderRequest.resolve({ orderId: 'top-order-1' })
  await top
  assertResourceActionCleared(page, '购买流程结束后')
})

test('my resource top query failure clears both the write lock and phase', async () => {
  const page = loadMyResourcesPage({
    listTopVouchers: async () => { throw new Error('权益查询失败') },
  })
  page.merchantId.value = 'merchant-1'

  await assert.rejects(page.topResource({ id: 'resource-1', status: 'published' }), /权益查询失败/)

  assertResourceActionCleared(page, '权益查询失败后')
})

test('my resource cancels top purchase before creating an order and clears both states', async () => {
  let orderCalls = 0
  const page = loadMyResourcesPage({
    listTopVouchers: async () => ({ items: [] }),
    listQuotaPacks: async () => ({ items: [{ code: 'top_1d', benefits: { topVoucherCount: 1, topDurationHours: 24 } }] }),
    createQuotaPackOrder: async () => { orderCalls += 1 },
    uni: {
      showModal: ({ success }) => success({ confirm: false }),
      showToast: () => {},
    },
  })
  page.merchantId.value = 'merchant-1'

  await page.topResource({ id: 'resource-1', status: 'published' })

  assert.equal(orderCalls, 0, '取消购买确认后不得创建订单')
  assertResourceActionCleared(page, '取消置顶购买确认后')
})
