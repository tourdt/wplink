import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

function ref(value) {
  return { value }
}

function computed(getter) {
  return { get value() { return getter() } }
}

function createTrackedRefFactory() {
  const writes = new WeakMap()
  return {
    ref(initialValue) {
      let currentValue = initialValue
      const state = {}
      writes.set(state, [])
      Object.defineProperty(state, 'value', {
        get() {
          return currentValue
        },
        set(nextValue) {
          writes.get(state).push(nextValue)
          currentValue = nextValue
        },
      })
      return state
    },
    writesFor(state) {
      return writes.get(state) || []
    },
  }
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

function detailButton(clickExpression) {
  const { descriptor } = parse(source, { filename: 'pages/resource/detail.vue' })
  const buttons = descriptor.template.content.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) || []
  const button = buttons.find((item) => item.includes(`@click="${clickExpression}"`))
  assert.ok(button, `应找到 ${clickExpression} 操作按钮`)
  return button
}

async function renderDetailButton(clickExpression, context) {
  const compiled = compileTemplate({
    source: detailButton(clickExpression),
    filename: 'pages/resource/detail.vue',
    id: 'resource-detail-action-loading',
    compilerOptions: { mode: 'function' },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const Button = {
    props: Object.keys(context),
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({ render: () => h(Button, context) }))
}

function openingButtonTag(html) {
  return html.match(/<button\b[^>]*>/)?.[0] || ''
}

function assertBusyButton(html, busyText, label) {
  const openingTag = openingButtonTag(html)
  assert.match(openingTag, /\bloading="true"/, `${label}应展示原生 loading`)
  assert.match(openingTag, /\bdisabled(?:=|\s|>)/, `${label}应禁用`)
  assert.match(html, new RegExp(busyText), `${label}应展示“${busyText}”`)
}

function loadResourceDetailPage(additions = {}) {
  const { descriptor } = parse(source, { filename: 'pages/resource/detail.vue' })
  const script = descriptor.scriptSetup.content.replace(/^import\s+[\s\S]*?\s+from\s+['"][^'"]+['"]\s*$/gm, '')
  const auxiliaryLoader = {
    begin: (resourceId) => ({ resourceId: String(resourceId) }),
    isCurrent: () => true,
    run: async () => {},
    loadMerchantProfile: async () => {},
  }
  const sandbox = {
    console,
    Promise,
    Set,
    Date,
    String,
    Number,
    Boolean,
    Math,
    setTimeout,
    clearTimeout,
    ref,
    computed,
    onLoad: () => {},
    onReady: () => {},
    onShareAppMessage: () => {},
    onShareTimeline: () => {},
    createResourceDetailAuxiliaryLoader: () => auxiliaryLoader,
    createResourceShareCoverRenderer: () => ({ request: () => {} }),
    buildResourceDetailPresentation: (item) => ({ noun: item.direction === 'demand' ? '需求' : '供应', isDemand: item.direction === 'demand', typeName: item.typeName || '' }),
    buildResourceShareRenderContext: () => ({}),
    buildResourceSharePosterModel: () => ({}),
    buildResourceSharePayload: () => ({}),
    buildResourceTimelinePayload: () => ({}),
    getResourceShareCoverSource: () => '',
    RESOURCE_SHARE_COVER_CANVAS_ID: 'resourceShareCoverCanvas',
    RESOURCE_SHARE_COVER_SIZE: { width: 600, height: 480 },
    requireLogin: () => true,
    getSession: () => ({ token: 'token', merchantId: 'merchant-1' }),
    getResourceFavoriteState: async () => ({}),
    getMerchant: async () => ({}),
    listRelatedResources: async () => ({ items: [] }),
    recordResourceDetailView: async () => ({}),
    getOwnResource: async () => ({ id: 'resource-1', presentation: { fields: [], tags: [] } }),
    getResource: async () => ({}),
    setResourceFavorite: async () => ({ favorited: true }),
    recordResourceContact: async () => ({}),
    createContactUnlockOrder: async () => ({ alreadyUnlocked: true }),
    createContactUnlockPayment: async () => ({ status: 'paid' }),
    listTopVouchers: async () => ({ items: [] }),
    redeemTopVoucher: async () => ({}),
    refreshResource: async () => ({}),
    takeDownResource: async () => ({}),
    deleteTakenDownResource: async () => ({}),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-1' }),
    createVIPPayment: async () => ({ status: 'paid' }),
    listQuotaPacks: async () => ({ items: [{ code: 'top_1d', benefits: { topVoucherCount: 1, topDurationHours: 24 } }] }),
    uni: {
      showToast: () => {},
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
      requestPayment: ({ success }) => success({}),
      navigateTo: () => {},
      setStorageSync: () => {},
      setClipboardData: () => {},
      makePhoneCall: () => {},
    },
    ...additions,
  }
  vm.runInNewContext(`${script}\nglobalThis.resourceDetailPage = {
    resource, ownerMerchantId, isOwnResource, showManagementSheet, showContactMoreSheet, favorited,
    managementAction, managementBusy, favoriteBusy, contactAction, contactBusyText, managementActions,
    managementActionLabel, handleManagementAction, closeManagementSheet, toggleFavorite, favoriteResourceFromMore, callPhone, copyWechat,
  }`, sandbox, { filename: 'pages/resource/detail.vue' })
  return sandbox.resourceDetailPage
}

test('resource detail gallery uses banner swiper and full screen preview', () => {
  assert.match(source, /const selectedGalleryIndex = ref\(0\)/)
  assert.match(source, /<swiper[\s\S]*v-if="galleryImages\.length > 1"[\s\S]*duration="450"[\s\S]*easing-function="easeInOutCubic"[\s\S]*@change="handleGalleryChange"/)
  assert.doesNotMatch(source, /:current="selectedGalleryIndex"/)
  assert.match(source, /<swiper-item[\s\S]*v-for="\(\s*url,\s*index\s*\) in galleryImages"/)
  assert.match(source, /@click="previewGalleryImage\(index\)"/)
  assert.match(source, /<image v-else class="gallery-main" :src="mainImage" mode="aspectFill" @click="previewGalleryImage\(0\)" \/>/)
  assert.match(source, /function handleGalleryChange\(event\) \{[\s\S]*selectedGalleryIndex\.value = current[\s\S]*\}/)
  assert.match(source, /function previewGalleryImage\(index = selectedGalleryIndex\.value\) \{[\s\S]*uni\.previewImage\(\{[\s\S]*current,[\s\S]*urls: galleryImages\.value[\s\S]*\}\)/)
  assert.equal(source.includes('gallery-strip'), false)
  assert.equal(source.includes('gallery-thumb'), false)
  assert.equal(source.includes('selectGalleryImage'), false)
})

test('resource detail hides the gallery when no image is available', () => {
  assert.match(source, /<view v-if="galleryImages\.length" class="detail-gallery">/)
  assert.doesNotMatch(source, /gallery-placeholder/)
  assert.doesNotMatch(source, /noImageBadgeText/)
  assert.doesNotMatch(source, /canEditOwnResourceWithoutImage/)
})

test('resource detail updates navigation title by supply or demand direction', () => {
  const detailPage = pagesConfig.pages.find((item) => item.path === 'pages/resource/detail')

  assert.equal(detailPage?.style?.navigationBarTitleText, '供应详情')
  assert.equal(source.includes('供需详情'), false)
  assert.equal(JSON.stringify(detailPage).includes('供需详情'), false)
  assert.match(source, /function updateNavigationTitle\(\) \{[\s\S]*uni\.setNavigationBarTitle\(\{[\s\S]*title: isDemandResource\.value \? '需求详情' : '供应详情'[\s\S]*\}\)[\s\S]*\}/)
  assert.match(source, /const detail = isOwnResource\.value[\s\S]*await getOwnResource[\s\S]*resource\.value = detail[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /async function loadOwnResourceIfCurrentMerchant\(resourceId, loadContext\) \{[\s\S]*const detail = await getOwnResource[\s\S]*resource\.value = detail[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /async function reloadOwnResource\(\) \{[\s\S]*resource\.value = await getOwnResource[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /onReady\(\(\) => \{[\s\S]*if \(resource\.value\.id\) updateNavigationTitle\(\)[\s\S]*\}\)/)
})

test('resource detail keeps contact reminder friendly and visually quiet', () => {
  assert.match(source, /<text class="section-title">友情提示<\/text>/)
  assert.match(source, /<text class="section-content contact-tip-content">联系\{\{ resourceNoun \}\}方前，建议先确认实物、价格、数量和交付方式。<\/text>/)
  assert.match(source, /\.contact-tip-content \{[\s\S]*font-size: 26rpx;[\s\S]*line-height: 1\.5;[\s\S]*\}/)
  assert.equal(source.includes('平台已记录联系行为'), false)
  assert.equal(source.includes('<text class="section-title">联系提示</text>'), false)
})

test('resource detail places the friendly tip after merchant and before related resources', () => {
  const merchantIndex = source.indexOf('class="merchant-card"')
  const tipIndex = source.indexOf('class="trust-card"')
  const relatedIndex = source.indexOf('class="related-section"')

  assert.ok(merchantIndex >= 0)
  assert.ok(tipIndex > merchantIndex)
  assert.ok(relatedIndex > tipIndex)
})

test('resource detail only shows merchant home entry after merchant profile is completed', () => {
  assert.match(source, /const showMerchantHomeEntry = computed\(\(\) => \{[\s\S]*merchantInfo\.value\.profileStatus === 'completed'[\s\S]*\}\)/)
  assert.match(source, /<view v-if="showMerchantHomeEntry" class="merchant-card" @click="openMerchant">/)
  assert.match(source, /function openMerchant\(\) \{[\s\S]*if \(!showMerchantHomeEntry\.value\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/merchant\/detail\?id=\$\{merchantId\}` \}\)/)
})

test('resource detail does not show merchant endorsement copy in the description card', () => {
  assert.doesNotMatch(source, /const isVIPMerchant = computed/)
  assert.doesNotMatch(source, /<text v-if="isVIPMerchant" class="tag vip">VIP<\/text>/)
  assert.doesNotMatch(source, /平台核实/)
  assert.doesNotMatch(source, /认证商家/)
})

test('resource detail keeps publish status out of the description card', () => {
  assert.doesNotMatch(source, /<text v-if="isOwnResource && resource\.status" class="tag">/)
  assert.doesNotMatch(source, /<text v-if="resource\.status" class="tag">/)
  assert.match(source, /const managementTitle = computed\(\(\) => statusText\[resource\.value\.status\] \|\| `\$\{resourceNoun\.value\}管理`\)/)
})

test('resource detail places automatic summary before specs and merchant', () => {
  const summaryIndex = source.indexOf('class="detail-summary-card"')
  const merchantIndex = source.indexOf('class="merchant-card"')
  const specIndex = source.indexOf('详细参数')

  assert.ok(summaryIndex >= 0)
  assert.ok(specIndex > summaryIndex)
  assert.ok(merchantIndex > specIndex)
  assert.doesNotMatch(source, /summary-title/)
  assert.doesNotMatch(source, /summary-facts/)
  assert.doesNotMatch(source, /summary-fact/)
  assert.doesNotMatch(source, /detailPresentation\.headline/)
  assert.doesNotMatch(source, /detailPresentation\.facts/)
})

test('resource detail displays only backend presentation tags in the automatic summary card', () => {
  assert.match(source, /const resourceFeatureTags = computed\(\(\) => resource\.value\.presentation\.tags\)/)
  assert.doesNotMatch(source, /const hasDetailTags = computed/)
  assert.match(source, /<view class="detail-summary-card">[\s\S]*<view v-if="resourceFeatureTags\.length" class="tag-row">/)
  assert.match(source, /<text v-for="tag in resourceFeatureTags" :key="tag" class="tag feature">\{\{ tag \}\}<\/text>/)
  assert.doesNotMatch(source, /<text v-if="resource\.refreshedAt" class="tag">/)
  assert.doesNotMatch(source, /<text v-if="isOwnResource && resource\.status" class="tag">/)
  assert.doesNotMatch(source, /function normalizeResourceFeatureTags/)
  assert.match(source, /\.tag \{[\s\S]*border: 1rpx solid rgba\(100, 116, 139, 0\.22\);[\s\S]*font-weight: 700;/)
  assert.match(source, /\.tag\.feature \{[\s\S]*background: #fff7ed;[\s\S]*color: \$wplink-warning;/)
})

test('resource detail displays a non-empty user description after tags in the summary card', () => {
  const galleryIndex = source.indexOf('class="detail-gallery"')
  const summaryIndex = source.indexOf('class="detail-summary-card"')

  assert.ok(summaryIndex > galleryIndex)
  assert.match(source, /<view class="detail-summary-card">[\s\S]*<view v-if="resourceFeatureTags\.length" class="tag-row">[\s\S]*<\/view>[\s\S]*<text v-if="resource\.description" class="desc">\{\{ resource\.description \}\}<\/text>[\s\S]*<\/view>/)
  assert.doesNotMatch(source, /description-card/)
  assert.doesNotMatch(source, /补充说明/)
  assert.match(source, /\.desc \{[\s\S]*background: #f8fafc;[\s\S]*font-size: 28rpx;[\s\S]*line-height: 1\.6;/)
  assert.equal(source.includes('class="favorite-button"'), false)
  assert.equal(source.includes('.favorite-button'), false)
  assert.equal(source.includes('.detail-head-row'), false)
  assert.equal(source.includes('.title-row'), false)
  assert.equal(source.includes('.price {'), false)
})

test('own resource detail keeps share and management actions in the bottom bar', () => {
  assert.match(source, /<view v-if="isOwnResource" class="owner-action-bar">/)
  assert.match(source, /<button class="share-button" @click="shareOwnResource" :open-type="canShareOwnResource \? 'share' : ''">分享<\/button>/)
  assert.match(source, /const canShareOwnResource = computed\(\(\) => resource\.value\.status === 'published' && !isExpiredResource\.value && !resource\.value\.dealtAt\)/)
  assert.match(source, /<button class="primary-button" @click="openManagementSheet">管理<\/button>/)
  assert.match(source, /<view v-else-if="!isDealtResource" class="contact-bar">/)
  assert.match(source, /<view v-else class="completed-action-bar">/)
  assert.doesNotMatch(source, /这是你发布的供需信息，可在我的发布中管理/)
})

test('pending own resource management sheet explains automatic safety state', () => {
  assert.match(source, /const contentAuditStatuses = new Set\(\['pending', 'manual_review', 'audit_retry'\]\)/)
  assert.match(source, /const managementNotice = computed\(\(\) => \{[\s\S]*resource\.value\.status === 'pending'[\s\S]*\$\{resourceNoun\.value\}正在自动安全检测[\s\S]*\}\)/)
  assert.match(source, /resource\.value\.status === 'audit_retry'[\s\S]*系统正在自动重试安全检测/)
  assert.match(source, /resource\.value\.status === 'manual_review'[\s\S]*\$\{resourceNoun\.value\}处于异常处理中/)
  assert.match(source, /const managementActions = computed\(\(\) => \{[\s\S]*if \(isContentAuditStatus\(resource\.value\.status\)\) return \[\][\s\S]*\}\)/)
  assert.match(source, /<view v-if="showManagementSheet" class="sheet-mask" @click="closeManagementSheet">/)
  assert.match(source, /<text class="sheet-title">\{\{ managementTitle \}\}<\/text>/)
  assert.match(source, /<text v-if="!managementActions\.length" class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.match(source, /<view v-if="managementActions\.length" class="management-actions">/)
  assert.match(source, /<text v-else class="empty-management">暂无可操作功能<\/text>/)
  assert.doesNotMatch(source, /<text class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.doesNotMatch(source, /查看我的发布/)
  assert.doesNotMatch(source, /返回列表/)
})

test('own resource management sheet does not expose deal action', () => {
  assert.doesNotMatch(source, /标记成交/)
  assert.doesNotMatch(source, /key: 'deal'/)
  assert.doesNotMatch(source, /markOwnResourceDealt/)
  assert.doesNotMatch(source, /markResourceDeal/)
})

test('resource detail unlocks contact through backend before copy or call', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(source, /if \(isContactUnlockAction\(action\) && !requireLogin\(\)\) return false/)
  assert.match(source, /function isContactUnlockAction\(action\) \{[\s\S]*return action === 'phone' \|\| action === 'wechat'[\s\S]*\}/)
  assert.match(source, /const resp = await recordResourceContact\(resource\.value\.id, action\)/)
  assert.match(source, /return resp \|\| \{\}/)
  assert.match(source, /uni\.setClipboardData\(\{ data: resp\.wechat \}\)/)
  assert.match(source, /uni\.makePhoneCall\(\{ phoneNumber: resp\.phone \}\)/)
  assert.match(source, /微信号已复制/)
  assert.equal(source.includes('已记录联系，完整微信由平台保护'), false)
  assert.equal(source.includes('已记录联系，完整电话由平台保护'), false)
})

test('resource detail uses short phone action text in bottom bar', () => {
  assert.match(source, /const contactButtonText = computed\(\(\) => '拨打电话'\)/)
  assert.doesNotMatch(source, /contactAccess\.value\.actionText \|\| '联系商家'/)
  assert.doesNotMatch(source, />登录后免费查看</)
  assert.doesNotMatch(source, />查看联系方式</)
})

test('resource detail groups low-frequency contact actions behind more sheet', () => {
  const contactBar = source.match(/<view v-else-if="!isDealtResource" class="contact-bar">[\s\S]*?<\/view>/)?.[0] || ''
  const contactMoreSheet = source.match(/<view v-if="showContactMoreSheet"[\s\S]*?<view v-if="isOwnResource"/)?.[0] || ''

  assert.match(source, /const showContactMoreSheet = ref\(false\)/)
  assert.match(source, /<view v-if="showContactMoreSheet" class="sheet-mask" @click="closeContactMoreSheet">/)
  assert.doesNotMatch(contactMoreSheet, /收藏、分享给同行或反馈问题资源。/)
  assert.doesNotMatch(contactMoreSheet, /class="sheet-desc"/)
  assert.match(contactMoreSheet, /<button class="management-action danger" @click="reportResourceFromMore">[\s\S]*report\.svg[\s\S]*<text class="action-label">举报<\/text>[\s\S]*<\/button>/)
  assert.match(contactMoreSheet, /<button class="management-action" @click="favoriteResourceFromMore"[\s\S]*bookmark\.svg[\s\S]*<text class="action-label">[\s\S]*<\/text>[\s\S]*<\/button>/)
  assert.match(contactMoreSheet, /<button class="management-action" open-type="share" @click="shareResourceFromMore">[\s\S]*share\.svg[\s\S]*<text class="action-label">分享给朋友<\/text>[\s\S]*<\/button>/)
  assert.ok(contactMoreSheet.indexOf('reportResourceFromMore') < contactMoreSheet.indexOf('favoriteResourceFromMore'))
  assert.ok(contactMoreSheet.indexOf('favoriteResourceFromMore') < contactMoreSheet.indexOf('shareResourceFromMore'))
  assert.match(source, /\{ key: 'edit', label: '编辑', icon: actionIconPaths\.edit \}/)
  assert.match(source, /\{ key: 'repost', label: '再发类似', icon: actionIconPaths\.repost \}/)
  assert.match(source, /\{ key: 'delete', label: '删除', icon: actionIconPaths\.delete, danger: true \}/)
  assert.match(source, /\{ key: 'refresh', label: '刷新', icon: actionIconPaths\.refresh \}/)
  assert.match(source, /\{ key: 'top', label: '置顶', icon: actionIconPaths\.top \}/)
  assert.match(source, /\{ key: 'take-down', label: '下架', icon: actionIconPaths\.takeDown, danger: true \}/)
  assert.match(source, /async function favoriteResourceFromMore\(\) \{[\s\S]*const success = await toggleFavorite\(\)[\s\S]*if \(success\) closeContactMoreSheet\(\)[\s\S]*\}/)
  assert.match(source, /async function shareResourceFromMore\(\) \{[\s\S]*await shareResource\(\)[\s\S]*closeContactMoreSheet\(\)[\s\S]*\}/)
  assert.match(source, /async function reportResourceFromMore\(\) \{[\s\S]*closeContactMoreSheet\(\)[\s\S]*openResourceReportPage\(\)[\s\S]*\}/)
  assert.match(contactBar, /@click="copyWechat"/)
  assert.match(contactBar, /@click="callPhone"/)
  assert.match(contactBar, /@click="openContactMoreSheet"/)
  assert.ok(contactBar.indexOf('openContactMoreSheet') < contactBar.indexOf('copyWechat'))
  assert.ok(contactBar.indexOf('copyWechat') < contactBar.indexOf('callPhone'))
  assert.equal(contactBar.includes('open-type="share"'), false)
  assert.equal(contactBar.includes('reportCurrentResource'), false)
  assert.match(source, /grid-template-columns: 104rpx minmax\(0, 1fr\) minmax\(0, 1\.35fr\);/)
  assert.equal(source.includes('grid-template-columns: repeat(4, 1fr);'), false)
})

test('resource detail action sheet close buttons align with their title areas', () => {
  const sheetHead = source.match(/\.sheet-head \{[^}]*\}/)?.[0] || ''

  assert.match(sheetHead, /grid-template-columns: minmax\(0, 1fr\) 120rpx;/)
  assert.match(sheetHead, /align-items: center;/)
  assert.match(source, /<text v-if="!managementActions\.length" class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.doesNotMatch(sheetHead, /align-items: start;/)
})

test('resource detail action icons are bundled local svg assets', () => {
  for (const name of ['bookmark', 'share', 'report', 'edit', 'refresh', 'top', 'take-down', 'repost', 'delete']) {
    const icon = fs.readFileSync(path.join(root, 'static/action-icons', `${name}.svg`), 'utf8')
    assert.match(icon, /<svg/)
    assert.match(icon, /viewBox="0 0 24 24"/)
  }
})

test('resource detail action sheets use centered icon-over-label layout', () => {
  assert.match(source, /\.management-actions \{[\s\S]*display: flex;[\s\S]*justify-content: space-around;[\s\S]*gap: 14rpx;/)
  assert.match(source, /\.management-action \{[\s\S]*display: flex;[\s\S]*flex-direction: column;[\s\S]*width: 200rpx;[\s\S]*min-height: 132rpx;/)
  assert.match(source, /\.action-icon-wrap \{[\s\S]*width: 76rpx;[\s\S]*height: 76rpx;[\s\S]*border-radius: 50%;/)
  assert.match(source, /\.action-icon \{[\s\S]*width: 40rpx;[\s\S]*height: 40rpx;/)
  assert.match(source, /\.management-action\.danger \.action-icon-wrap \{[\s\S]*background: #fff7f8;[\s\S]*border-color: #f2ced3;/)
  assert.doesNotMatch(source, /\.management-action\.primary/)
})

test('resource detail renders related resources with the market feed card variant', () => {
  assert.match(source, /<view v-if="relatedResources\.length" class="related-section">[\s\S]*<ResourceList[\s\S]*:resources="relatedResources"[\s\S]*variant="feed"[\s\S]*@open="openRelatedResource"/)
})

test('resource detail delegates related resource lifecycle to the generation-aware loader', () => {
  assert.match(source, /listRelatedResources/)
  assert.doesNotMatch(source, /listResources\(\{ typeCode: resource\.value\.typeCode/)
  assert.match(source, /createResourceDetailAuxiliaryLoader/)
  assert.match(source, /const loadContext = detailAuxiliaryLoader\.begin\(options\.id\)/)
  assert.match(source, /await detailAuxiliaryLoader\.run\(loadContext, \{[\s\S]*initializeSharing: initializeResourceSharing,[\s\S]*isOwnResource: isOwnResource\.value/)
  assert.match(source, /detailAuxiliaryLoader\.isCurrent\(loadContext\)/)
  assert.match(source, /getFavoriteState: getResourceFavoriteState/)
  assert.match(source, /getMerchantProfile\(merchantId\) \{[\s\S]*getMerchant\(merchantId, \{ suppressErrorToast: true \}\)/)
  assert.match(source, /setFavorited\(value\) \{[\s\S]*favorited\.value = Boolean\(value\)/)
  assert.match(source, /setMerchantProfile\(profile\) \{[\s\S]*merchantProfile\.value = profile \|\| \{\}/)
  assert.match(source, /setShareImageUrl\(imageUrl\) \{[\s\S]*shareImageUrl\.value = imageUrl \|\| ''/)
  assert.doesNotMatch(source, /async function loadFavoriteState/)
  assert.doesNotMatch(source, /async function loadMerchantProfile/)
})

test('resource detail opens dedicated report page from more sheet', () => {
  assert.match(source, /function openResourceReportPage\(\) \{[\s\S]*if \(!resource\.value\.id \|\| isOwnResource\.value\) return[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*resourceId=\$\{encodeURIComponent\(resource\.value\.id\)\}[\s\S]*\/pages\/resource\/report\?\$\{params\.join\('&'\)\}[\s\S]*\}/)
  assert.doesNotMatch(source, /reportResource\(resource\.value\.id/)
  assert.doesNotMatch(source, /chooseResourceReportReason/)
  assert.doesNotMatch(source, /collectResourceReportContact/)
})

test('resource detail handles paid contact unlock flow', () => {
  const apiSource = fs.readFileSync(path.join(root, 'api/resource.js'), 'utf8')

  assert.match(source, /contactAccess/)
  assert.match(source, /createContactUnlockOrder/)
  assert.match(source, /createContactUnlockPayment/)
  assert.match(source, /PAYMENT_REQUIRED/)
  assert.match(source, /requestPayment/)
  assert.match(apiSource, /createContactUnlockOrder/)
  assert.match(apiSource, /createContactUnlockPayment/)
})

test('resource detail supports WeChat group and timeline sharing with generated cover image', () => {
  assert.match(source, /import \{ onLoad, onReady, onShareAppMessage, onShareTimeline \} from '@dcloudio\/uni-app'/)
  assert.match(source, /buildResourceSharePayload/)
  assert.match(source, /buildResourceTimelinePayload/)
  assert.match(source, /buildResourceSharePosterModel/)
  assert.match(source, /RESOURCE_SHARE_COVER_CANVAS_ID/)
  assert.match(source, /RESOURCE_SHARE_COVER_SIZE/)
  assert.match(source, /const shareImageUrl = ref\(''\)/)
  assert.match(source, /createResourceShareCoverRenderer/)
  assert.match(source, /buildResourceShareRenderContext/)
  assert.match(source, /scheduleShareCoverRender\(loadContext\)/)
  assert.match(source, /merchantId: \(resource\.value\.merchant \|\| \{\}\)\.id,[\s\S]*onMerchantProfileLoaded: scheduleShareCoverRender/)
  assert.match(source, /const renderContext = buildResourceShareRenderContext\([\s\S]*loadContext,[\s\S]*resource\.value,[\s\S]*merchantProfile\.value,[\s\S]*\)/)
  assert.match(source, /shareCoverRenderer\.request\(renderContext\)/)
  assert.match(source, /renderShareCoverForContext\(renderContext\)/)
  assert.match(source, /<canvas[\s\S]*canvas-id="resourceShareCoverCanvas"[\s\S]*class="share-cover-canvas"[\s\S]*:width="shareCoverCanvasSize\.width"[\s\S]*:height="shareCoverCanvasSize\.height"/)
  assert.match(source, /uni\.showShareMenu\(\{[\s\S]*menus: \['shareAppMessage', 'shareTimeline'\][\s\S]*\}\)/)
  assert.match(source, /uni\.canvasToTempFilePath\(\{[\s\S]*canvasId: RESOURCE_SHARE_COVER_CANVAS_ID[\s\S]*success: \(res\) => resolve\(res\.tempFilePath \|\| ''\)/)
  assert.match(source, /onShareAppMessage\(\(shareEvent\) => \{[\s\S]*buildResourceSharePayload\(resource\.value, shareImageUrl\.value\)[\s\S]*\}\)/)
  assert.match(source, /onShareTimeline\(\(\) => \{[\s\S]*buildResourceTimelinePayload\(resource\.value, shareImageUrl\.value\)[\s\S]*\}\)/)
  assert.match(source, /\.share-cover-canvas \{[\s\S]*position: fixed;[\s\S]*left: -9999px;[\s\S]*width: 600px;[\s\S]*height: 480px;/)
})

test('resource detail renders backend presentation fields without local deduplication or layout inference', () => {
  assert.match(source, /<view v-if="resource\.presentation\.fields\.length" class="resource-card">/)
  assert.match(source, /v-for="item in resource\.presentation\.fields" :key="item\.key"/)
  assert.match(source, /:class="\['spec-item', item\.layout === 'full' \? 'full-width' : ''\]"/)
  assert.doesNotMatch(source, /buildResourceDetailSpecItems/)
  assert.doesNotMatch(source, /attributeSpecItems/)
  assert.doesNotMatch(source, /addressAttributeKeys/)
  assert.doesNotMatch(source, /summarySourceKeys/)
  assert.doesNotMatch(source, /fullWidth/)
  assert.match(source, /\.spec-item\.full-width \{[\s\S]*grid-column: 1 \/ -1;/)
})

test('resource detail renders address attributes as a dedicated navigation section', () => {
  assert.match(source, /<view v-if="resourceAddressLocations\.length" class="address-section">/)
  assert.match(source, /<text class="section-title">地址<\/text>/)
  assert.equal(source.includes('位置与导航'), false)
  assert.match(source, /<view class="resource-address-main">[\s\S]*<view class="resource-address-copy">[\s\S]*<text class="resource-address-text">\{\{ item\.address \}\}<\/text>[\s\S]*<button v-if="item\.hasGps" class="address-action" @click="openResourceAddressLocation\(item\)">导航<\/button>/)
  assert.match(source, /<button v-if="item\.hasGps" class="address-action" @click="openResourceAddressLocation\(item\)">导航<\/button>/)
  assert.match(source, /<button v-else class="address-action secondary" @click="copyResourceAddress\(item\)">复制<\/button>/)
  assert.match(source, /<text class="resource-address-text">\{\{ item\.address \}\}<\/text>[\s\S]*<map[\s\S]*v-if="item\.hasGps"[\s\S]*class="resource-address-map"[\s\S]*:latitude="item\.latitude"[\s\S]*:longitude="item\.longitude"[\s\S]*@tap="openResourceAddressLocation\(item\)"/)
  assert.match(source, /const resourceAddressLocations = computed\(\(\) => \{[\s\S]*Object\.entries\(attributes\)[\s\S]*buildResourceAddressLocation/)
  assert.match(source, /function buildResourceAddressLocation\(key, value, index\) \{[\s\S]*isResourceAddressAttributeValue\(value\)[\s\S]*const hasGps = Number\.isFinite\(latitude\) && Number\.isFinite\(longitude\)[\s\S]*if \(!hasGps\) return item[\s\S]*markers/)
  assert.match(source, /function isResourceAddressAttributeValue\(value\) \{[\s\S]*\['address', 'name', 'latitude', 'longitude', 'lat', 'lng'\]/)
  assert.match(source, /function openResourceAddressLocation\(item\) \{[\s\S]*uni\.openLocation\(\{[\s\S]*latitude: item\.latitude[\s\S]*longitude: item\.longitude[\s\S]*scale: 18/)
  assert.match(source, /function copyResourceAddress\(item, title = '地址已复制'\)/)
  assert.match(source, /导航打开失败，已复制地址/)
  assert.match(source, /\.resource-address-main \{[\s\S]*grid-template-columns: minmax\(0, 1fr\) 96rpx;/)
  assert.match(source, /\.resource-address-map \{[\s\S]*height: 220rpx;/)
  assert.match(source, /\.resource-address-text \{[\s\S]*display: block;[\s\S]*font-size: 27rpx;[\s\S]*word-break: break-word;/)
})

test('resource detail renders action-specific loading for management, favorite, WeChat and phone buttons', async () => {
  const page = loadResourceDetailPage()
  const managementContracts = [
    { key: 'refresh', label: '刷新', busyText: '刷新中' },
    { key: 'top', label: '置顶', busyText: '置顶中' },
    { key: 'take-down', label: '下架', busyText: '下架中' },
    { key: 'repost', label: '再发类似', busyText: '准备中' },
    { key: 'delete', label: '删除', busyText: '删除中' },
  ]

  for (const action of managementContracts) {
    page.managementAction.value = action.key
    const html = await renderDetailButton('handleManagementAction(action.key)', {
      managementActions: [action],
      managementAction: action.key,
      managementBusy: true,
      managementActionLabel: page.managementActionLabel,
      handleManagementAction: () => {},
    })
    assertBusyButton(html, action.busyText, `${action.label}操作`)
  }

  page.managementAction.value = 'refresh'
  const lockedTopHtml = await renderDetailButton('handleManagementAction(action.key)', {
    managementActions: [{ key: 'top', label: '置顶' }],
    managementAction: 'refresh',
    managementBusy: true,
    managementActionLabel: page.managementActionLabel,
    handleManagementAction: () => {},
  })
  assert.match(openingButtonTag(lockedTopHtml), /\bdisabled(?:=|\s|>)/, '其他管理动作应被禁用')
  assert.doesNotMatch(openingButtonTag(lockedTopHtml), /\bloading="true"/, '其他管理动作不应冒充当前动作显示 loading')
  assert.match(lockedTopHtml, />置顶</, '其他管理动作应保留空闲文案')

  const favoriteHtml = await renderDetailButton('favoriteResourceFromMore', {
    favoriteBusy: true,
    favorited: false,
    favoriteResourceFromMore: () => {},
  })
  assertBusyButton(favoriteHtml, '收藏中', '收藏操作')

  const unfavoriteHtml = await renderDetailButton('favoriteResourceFromMore', {
    favoriteBusy: true,
    favorited: true,
    favoriteResourceFromMore: () => {},
  })
  assertBusyButton(unfavoriteHtml, '取消收藏中', '取消收藏操作')

  const wechatHtml = await renderDetailButton('copyWechat', {
    contactAction: 'wechat',
    contactBusyText: '查询中',
    copyWechat: () => {},
  })
  assertBusyButton(wechatHtml, '查询中', '微信查询')

  const phoneHtml = await renderDetailButton('callPhone', {
    contactAction: 'phone',
    contactBusyText: '解锁中',
    contactButtonText: '拨打电话',
    callPhone: () => {},
  })
  assertBusyButton(phoneHtml, '解锁中', '电话解锁')

  const lockedWechatHtml = await renderDetailButton('copyWechat', {
    contactAction: 'phone',
    contactBusyText: '解锁中',
    copyWechat: () => {},
  })
  assert.match(openingButtonTag(lockedWechatHtml), /\bdisabled(?:=|\s|>)/, '电话处理中应禁用微信按钮')
  assert.doesNotMatch(openingButtonTag(lockedWechatHtml), /\bloading="true"/, '微信按钮不应显示电话动作的 loading')
  assert.match(lockedWechatHtml, />复制微信</, '非当前联系方式按钮应保留空闲文案')
})

test('resource detail favorite action rejects duplicates and restores state after success or failure', async () => {
  const favoriteRequest = deferred()
  const favoriteCalls = []
  const page = loadResourceDetailPage({
    setResourceFavorite: (resourceId, nextFavorited) => {
      favoriteCalls.push({ resourceId, nextFavorited })
      return favoriteRequest.promise
    },
  })
  page.resource.value = { id: 'resource-expected', direction: 'supply', presentation: { fields: [], tags: [] } }
  page.showContactMoreSheet.value = true

  const firstFavorite = page.favoriteResourceFromMore()
  const duplicateFavorite = page.favoriteResourceFromMore()
  assert.deepEqual(favoriteCalls, [{ resourceId: 'resource-expected', nextFavorited: true }], '收藏必须使用当前资源且连续点击只请求一次')
  assert.equal(page.favoriteBusy.value, true, '收藏请求期间应保持 busy')
  favoriteRequest.resolve({ favorited: true })
  await Promise.all([firstFavorite, duplicateFavorite])
  assert.equal(page.favoriteBusy.value, false, '收藏成功后应恢复状态')
  assert.equal(page.favorited.value, true, '收藏状态应以服务端结果为准')
  assert.equal(page.showContactMoreSheet.value, false, '收藏成功后应保留关闭更多面板的行为')

  const toasts = []
  const failedPage = loadResourceDetailPage({
    setResourceFavorite: async () => { throw new Error('收藏服务暂不可用') },
    uni: { showToast: (options) => toasts.push(options) },
  })
  failedPage.resource.value = { id: 'resource-expected', direction: 'supply', presentation: { fields: [], tags: [] } }
  failedPage.showContactMoreSheet.value = true
  await failedPage.favoriteResourceFromMore()
  assert.equal(failedPage.favoriteBusy.value, false, '收藏失败后应恢复状态')
  assert.equal(failedPage.showContactMoreSheet.value, true, '收藏失败时应保留更多面板以便重试')
  assert.equal(toasts.at(-1)?.title, '收藏服务暂不可用', '收藏失败应保留原有友好提示')

  let unauthenticatedCalls = 0
  const unauthenticatedPage = loadResourceDetailPage({
    requireLogin: () => false,
    setResourceFavorite: async () => { unauthenticatedCalls += 1; return { favorited: true } },
  })
  unauthenticatedPage.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }
  await unauthenticatedPage.toggleFavorite()
  assert.equal(unauthenticatedCalls, 0, '登录校验失败不应请求收藏接口')
  assert.equal(unauthenticatedPage.favoriteBusy.value, false, '登录校验失败不应进入 busy')
})

test('resource detail validates contact and favorite actions before touching busy state', async () => {
  const contactCases = [
    { label: '缺少资源', resource: { presentation: { fields: [], tags: [] } }, own: false, loggedIn: true },
    { label: '自己的资源', resource: { id: 'resource-expected', presentation: { fields: [], tags: [] } }, own: true, loggedIn: true },
    { label: '未登录', resource: { id: 'resource-expected', presentation: { fields: [], tags: [] } }, own: false, loggedIn: false },
  ]

  for (const item of contactCases) {
    const tracker = createTrackedRefFactory()
    let contactApiCalls = 0
    const page = loadResourceDetailPage({
      ref: tracker.ref,
      requireLogin: () => item.loggedIn,
      recordResourceContact: async () => { contactApiCalls += 1; return {} },
    })
    page.resource.value = item.resource
    page.isOwnResource.value = item.own

    await page.callPhone()
    await page.copyWechat()

    assert.deepEqual(tracker.writesFor(page.contactAction), [], `${item.label}时电话和微信不应写入 busy 状态`)
    assert.equal(contactApiCalls, 0, `${item.label}时不应请求联系方式接口`)
  }

  for (const item of contactCases) {
    const tracker = createTrackedRefFactory()
    let favoriteApiCalls = 0
    const page = loadResourceDetailPage({
      ref: tracker.ref,
      requireLogin: () => item.loggedIn,
      setResourceFavorite: async () => { favoriteApiCalls += 1; return { favorited: true } },
      uni: { showToast: () => {} },
    })
    page.resource.value = item.resource
    page.isOwnResource.value = item.own

    await page.toggleFavorite()

    assert.deepEqual(tracker.writesFor(page.favoriteBusy), [], `${item.label}时收藏不应写入 busy 状态`)
    assert.equal(favoriteApiCalls, 0, `${item.label}时不应请求收藏接口`)
  }

  let missingResourceChecks = 0
  const missingResourcePage = loadResourceDetailPage()
  missingResourcePage.resource.value = {
    get id() {
      missingResourceChecks += 1
      return ''
    },
    presentation: { fields: [], tags: [] },
  }
  missingResourcePage.favoriteBusy.value = true
  await missingResourcePage.toggleFavorite()
  assert.equal(missingResourceChecks, 1, '收藏应先检查资源，再判断现有 busy 锁')

  const ownResourceToasts = []
  const ownBusyPage = loadResourceDetailPage({ uni: { showToast: (options) => ownResourceToasts.push(options) } })
  ownBusyPage.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }
  ownBusyPage.isOwnResource.value = true
  ownBusyPage.favoriteBusy.value = true
  await ownBusyPage.toggleFavorite()
  assert.equal(ownResourceToasts.at(-1)?.title, '不能收藏自己发布的供应', '收藏应先执行自有资源校验，再判断现有 busy 锁')

  let loginChecks = 0
  const busyPage = loadResourceDetailPage({
    requireLogin: () => { loginChecks += 1; return true },
    setResourceFavorite: async () => { throw new Error('busy 时不应请求收藏接口') },
  })
  busyPage.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }
  busyPage.favoriteBusy.value = true
  await busyPage.toggleFavorite()
  assert.equal(loginChecks, 1, '收藏应先完成登录校验，再判断现有 busy 锁')
  assert.equal(busyPage.favoriteBusy.value, true, '重复收藏不应改写正在进行的 busy 状态')
})

test('resource detail contact action stays locked through record, order, payment and re-unlock', async () => {
  const orderRequest = deferred()
  const paymentRequest = deferred()
  const unlockedContactRequest = deferred()
  const recordCalls = []
  const orderCalls = []
  const paymentCalls = []
  const clipboardValues = []
  let paymentCallbacks
  const page = loadResourceDetailPage({
    recordResourceContact: (resourceId, action) => {
      recordCalls.push({ resourceId, action })
      if (recordCalls.length === 1) {
        const err = new Error('需要解锁')
        err.code = 'PAYMENT_REQUIRED'
        throw err
      }
      return unlockedContactRequest.promise
    },
    createContactUnlockOrder: (resourceId, payload) => {
      orderCalls.push({ resourceId, payload })
      return orderRequest.promise
    },
    createContactUnlockPayment: (resourceId, orderId, payload) => {
      paymentCalls.push({ resourceId, orderId, payload })
      return paymentRequest.promise
    },
    uni: {
      showToast: () => {},
      setClipboardData: ({ data }) => clipboardValues.push(data),
      requestPayment: (callbacks) => { paymentCallbacks = callbacks },
    },
  })
  page.resource.value = {
    id: 'resource-expected',
    contactAccess: { mode: 'paid_or_vip', priceCent: 500, unlocked: false },
    presentation: { fields: [], tags: [] },
  }

  const firstCopy = page.copyWechat()
  const duplicateCopy = page.copyWechat()
  await flushAsyncWork()
  assert.deepEqual(recordCalls, [{ resourceId: 'resource-expected', action: 'wechat' }], '首次解锁应使用当前资源和微信动作且拒绝连续点击')
  assert.equal(orderCalls.length, 1, '解锁订单只应创建一次')
  assert.equal(orderCalls[0].resourceId, 'resource-expected', '解锁订单应绑定当前资源')
  assert.equal(orderCalls[0].payload.action, 'wechat', '解锁订单应携带微信动作')
  assert.equal(page.contactAction.value, 'wechat', '订单创建期间应保持微信 busy')

  orderRequest.resolve({ orderId: 'contact-order-expected' })
  await flushAsyncWork()
  assert.equal(paymentCalls.length, 1, '联系方式支付参数只应创建一次')
  assert.equal(paymentCalls[0].resourceId, 'resource-expected', '联系方式支付应绑定当前资源')
  assert.equal(paymentCalls[0].orderId, 'contact-order-expected', '支付必须使用刚创建的联系方式订单')
  assert.equal(page.contactAction.value, 'wechat', '支付参数创建期间应保持微信 busy')

  paymentRequest.resolve({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', signType: 'RSA', paySign: 'sign' } })
  await flushAsyncWork()
  assert.equal(typeof paymentCallbacks?.success, 'function', '应保留微信原生支付弹窗')
  assert.equal(page.contactAction.value, 'wechat', '原生支付弹窗期间应保持微信 busy')

  paymentCallbacks.success({})
  await flushAsyncWork()
  assert.deepEqual(recordCalls.at(-1), { resourceId: 'resource-expected', action: 'wechat' }, '支付后应再次解锁同一资源的微信')
  assert.equal(recordCalls.length, 2, '支付后只应再请求一次解锁')
  assert.equal(page.contactAction.value, 'wechat', '再次解锁期间应保持微信 busy')

  unlockedContactRequest.resolve({ wechat: 'wx-expected' })
  await Promise.all([firstCopy, duplicateCopy])
  assert.deepEqual(clipboardValues, ['wx-expected'], '成功解锁后应保留复制微信行为')
  assert.equal(page.contactAction.value, '', '联系方式完整链路结束后应恢复状态')
})

test('resource detail contact payment cancellation preserves neutral feedback and restores the action', async () => {
  const toasts = []
  let phoneCalls = 0
  const page = loadResourceDetailPage({
    recordResourceContact: async () => {
      const err = new Error('需要解锁')
      err.code = 'PAYMENT_REQUIRED'
      throw err
    },
    createContactUnlockOrder: async () => ({ orderId: 'contact-order-expected' }),
    createContactUnlockPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', paySign: 'sign' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      makePhoneCall: () => { phoneCalls += 1 },
      requestPayment: ({ fail }) => fail({ errMsg: 'requestPayment:fail cancel' }),
    },
  })
  page.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }

  await page.callPhone()

  assert.equal(page.contactAction.value, '', '支付取消后应恢复联系方式状态')
  assert.equal(phoneCalls, 0, '支付取消后不应拨打电话')
  assert.equal(toasts.at(-1)?.title, '已取消支付', '平台取消错误应转换为中性中文提示')
})

test('resource detail contact payment system errors use friendly feedback and restore the action', async () => {
  const toasts = []
  const page = loadResourceDetailPage({
    recordResourceContact: async () => {
      const err = new Error('需要解锁')
      err.code = 'PAYMENT_REQUIRED'
      throw err
    },
    createContactUnlockOrder: async () => ({ orderId: 'contact-order-expected' }),
    createContactUnlockPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', paySign: 'sign' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      requestPayment: ({ fail }) => fail({ errMsg: 'requestPayment:fail system error' }),
    },
  })
  page.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }

  await page.copyWechat()

  assert.equal(page.contactAction.value, '', '支付系统错误后应恢复联系方式状态')
  assert.equal(toasts.at(-1)?.title, '支付失败，请稍后重试', '支付系统 errMsg 不应静默或直接暴露')
})

test('resource detail contact unlock preserves a friendly API error message', async () => {
  const toasts = []
  const page = loadResourceDetailPage({
    recordResourceContact: async () => {
      const err = new Error('需要解锁')
      err.code = 'PAYMENT_REQUIRED'
      throw err
    },
    createContactUnlockOrder: async () => { throw new Error('订单已取消支付，请重新发起') },
    uni: { showToast: (options) => toasts.push(options) },
  })
  page.resource.value = { id: 'resource-expected', presentation: { fields: [], tags: [] } }

  await page.callPhone()

  assert.equal(page.contactAction.value, '', '联系方式 API 失败后应恢复状态')
  assert.equal(toasts.at(-1)?.title, '订单已取消支付，请重新发起', '非平台 error.message 应保持原有友好文案')
})

test('resource detail management action rejects concurrent writes and stays visible through top purchase refresh', async () => {
  const orderRequest = deferred()
  const paymentRequest = deferred()
  const reloadRequest = deferred()
  const voucherCalls = []
  const orderCalls = []
  const paymentCalls = []
  const reloadCalls = []
  let refreshCalls = 0
  let paymentCallbacks
  const page = loadResourceDetailPage({
    listTopVouchers: async (merchantId) => { voucherCalls.push(merchantId); return { items: [] } },
    createQuotaPackOrder: (merchantId, packCode, payload) => {
      orderCalls.push({ merchantId, packCode, payload })
      return orderRequest.promise
    },
    createVIPPayment: (merchantId, orderId) => {
      paymentCalls.push({ merchantId, orderId })
      return paymentRequest.promise
    },
    getOwnResource: (resourceId, merchantId) => {
      reloadCalls.push({ resourceId, merchantId })
      return reloadRequest.promise
    },
    refreshResource: async () => { refreshCalls += 1 },
    uni: {
      showToast: () => {},
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
      requestPayment: (callbacks) => { paymentCallbacks = callbacks },
    },
  })
  page.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  page.ownerMerchantId.value = 'merchant-expected'
  page.showManagementSheet.value = true

  const firstTop = page.handleManagementAction('top')
  const duplicateTop = page.handleManagementAction('top')
  const concurrentRefresh = page.handleManagementAction('refresh')
  await flushAsyncWork()
  assert.deepEqual(voucherCalls, ['merchant-expected'], '置顶券查询只应发起一次且使用当前商家')
  assert.equal(orderCalls.length, 1, '置顶订单只应创建一次')
  assert.equal(orderCalls[0].merchantId, 'merchant-expected', '置顶订单应绑定当前商家')
  assert.equal(orderCalls[0].packCode, 'top_1d', '置顶订单应使用选中的服务包')
  assert.equal(orderCalls[0].payload.resourceId, 'resource-expected', '置顶订单应绑定当前资源')
  assert.equal(refreshCalls, 0, '置顶期间不应穿透锁执行刷新')
  assert.equal(page.managementAction.value, 'top', '创建置顶订单期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '支付前管理面板应保持可见')
  page.closeManagementSheet()
  assert.equal(page.showManagementSheet.value, true, '置顶进行时用户点击遮罩或关闭按钮不应隐藏反馈')

  orderRequest.resolve({ orderId: 'top-order-expected' })
  await flushAsyncWork()
  assert.deepEqual(paymentCalls, [{ merchantId: 'merchant-expected', orderId: 'top-order-expected' }], '置顶支付必须使用刚创建的订单')
  assert.equal(page.managementAction.value, 'top', '创建支付参数期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '支付参数创建期间管理面板应保持可见')

  paymentRequest.resolve({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', paySign: 'sign' } })
  await flushAsyncWork()
  assert.equal(typeof paymentCallbacks?.success, 'function', '置顶购买应保留微信原生支付弹窗')
  assert.equal(page.managementAction.value, 'top', '原生支付弹窗期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '原生支付弹窗期间管理面板应保持可见')

  paymentCallbacks.success({})
  await flushAsyncWork()
  assert.deepEqual(reloadCalls, [{ resourceId: 'resource-expected', merchantId: 'merchant-expected' }], '支付后应刷新当前资源详情')
  assert.equal(page.managementAction.value, 'top', '支付后刷新详情期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '刷新详情期间管理面板应保持可见')

  reloadRequest.resolve({ id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } })
  await Promise.all([firstTop, duplicateTop, concurrentRefresh])
  assert.equal(page.managementAction.value, '', '支付和刷新全部完成后应恢复管理状态')
  assert.equal(page.showManagementSheet.value, false, '支付和刷新全部完成后才关闭管理面板')
})

test('resource detail voucher redemption remains busy and visible until the refreshed detail arrives', async () => {
  const redeemRequest = deferred()
  const reloadRequest = deferred()
  const redeemCalls = []
  const reloadCalls = []
  const page = loadResourceDetailPage({
    listTopVouchers: async () => ({ items: [{ id: 'voucher-expected', remainingAmount: 1, topDurationHours: 24 }] }),
    redeemTopVoucher: (voucherId, resourceId, merchantId) => {
      redeemCalls.push({ voucherId, resourceId, merchantId })
      return redeemRequest.promise
    },
    getOwnResource: (resourceId, merchantId) => {
      reloadCalls.push({ resourceId, merchantId })
      return reloadRequest.promise
    },
  })
  page.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  page.ownerMerchantId.value = 'merchant-expected'
  page.showManagementSheet.value = true

  const top = page.handleManagementAction('top')
  await flushAsyncWork()
  assert.deepEqual(redeemCalls, [{ voucherId: 'voucher-expected', resourceId: 'resource-expected', merchantId: 'merchant-expected' }], '置顶券应核销到当前资源和商家')
  assert.equal(page.managementAction.value, 'top', '置顶券核销期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '置顶券核销期间管理面板应保持可见')

  redeemRequest.resolve({})
  await flushAsyncWork()
  assert.deepEqual(reloadCalls, [{ resourceId: 'resource-expected', merchantId: 'merchant-expected' }], '核销后应刷新当前资源')
  assert.equal(page.managementAction.value, 'top', '核销后的详情刷新期间应保持 top 状态')
  assert.equal(page.showManagementSheet.value, true, '核销后的详情刷新期间管理面板应保持可见')

  reloadRequest.resolve({ id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } })
  await top
  assert.equal(page.managementAction.value, '', '置顶券核销和刷新结束后应恢复管理状态')
  assert.equal(page.showManagementSheet.value, false, '置顶券核销和刷新结束后才关闭管理面板')
})

test('resource detail management payment cancellation and failures restore state while edit stays synchronous', async () => {
  const toasts = []
  const navigations = []
  const page = loadResourceDetailPage({
    listTopVouchers: async () => ({ items: [] }),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-expected' }),
    createVIPPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', paySign: 'sign' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
      requestPayment: ({ fail }) => fail({ errMsg: 'requestPayment:fail cancel' }),
      navigateTo: (options) => navigations.push(options),
    },
  })
  page.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  page.ownerMerchantId.value = 'merchant-expected'
  page.showManagementSheet.value = true

  await page.handleManagementAction('top')
  assert.equal(page.managementAction.value, '', '支付取消后应恢复管理状态')
  assert.equal(page.showManagementSheet.value, true, '支付取消后应保留管理面板供重试')
  assert.equal(toasts.at(-1)?.title, '已取消支付', '置顶支付取消应显示中性中文提示')

  await page.handleManagementAction('edit')
  assert.equal(page.managementAction.value, '', '编辑仅跳转，不应进入管理 busy')
  assert.equal(navigations.length, 1, '编辑只应跳转一次')
  assert.equal(navigations[0].url, '/pages/publish/edit?merchantId=merchant-expected&resourceId=resource-expected', '编辑应保留原有跳转')

  const failedPage = loadResourceDetailPage({ refreshResource: async () => { throw new Error('刷新服务暂不可用') } })
  failedPage.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  failedPage.ownerMerchantId.value = 'merchant-expected'
  await assert.rejects(failedPage.handleManagementAction('refresh'), /刷新服务暂不可用/)
  assert.equal(failedPage.managementAction.value, '', '管理接口失败后应恢复状态')
})

test('resource detail top payment system errors use friendly feedback and restore management state', async () => {
  const toasts = []
  const page = loadResourceDetailPage({
    listTopVouchers: async () => ({ items: [] }),
    createQuotaPackOrder: async () => ({ orderId: 'top-order-expected' }),
    createVIPPayment: async () => ({ payment: { timeStamp: '1', nonceStr: 'nonce', package: 'package', paySign: 'sign' } }),
    uni: {
      showToast: (options) => toasts.push(options),
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
      requestPayment: ({ fail }) => fail({ errMsg: 'requestPayment:fail system error' }),
    },
  })
  page.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  page.ownerMerchantId.value = 'merchant-expected'
  page.showManagementSheet.value = true

  await page.handleManagementAction('top')

  assert.equal(page.managementAction.value, '', '置顶支付系统错误后应恢复管理状态')
  assert.equal(page.showManagementSheet.value, true, '置顶支付系统错误后应保留管理面板供重试')
  assert.equal(toasts.at(-1)?.title, '置顶服务购买失败，请稍后重试', '置顶支付系统 errMsg 不应直接暴露')
})

test('resource detail top purchase preserves a friendly API error message', async () => {
  const toasts = []
  const page = loadResourceDetailPage({
    listTopVouchers: async () => ({ items: [] }),
    createQuotaPackOrder: async () => { throw new Error('置顶订单创建失败，请稍后重试') },
    uni: {
      showToast: (options) => toasts.push(options),
      showModal: ({ success }) => success({ confirm: true }),
      showActionSheet: ({ success }) => success({ tapIndex: 0 }),
    },
  })
  page.resource.value = { id: 'resource-expected', status: 'published', presentation: { fields: [], tags: [] } }
  page.ownerMerchantId.value = 'merchant-expected'

  await page.handleManagementAction('top')

  assert.equal(page.managementAction.value, '', '置顶 API 失败后应恢复管理状态')
  assert.equal(toasts.at(-1)?.title, '置顶订单创建失败，请稍后重试', '置顶 API error.message 应保持原有友好文案')
})

test('resource detail restores top voucher management action', () => {
  assert.match(source, /import \{ listTopVouchers, redeemTopVoucher \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /import \{ createQuotaPackOrder, createVIPPayment, listQuotaPacks \} from '\.\.\/\.\.\/api\/vip'/)
  assert.match(source, /\$\{resourceNoun\.value\}展示中，可按需刷新、置顶或下架。/)
  assert.match(source, /key: 'top'/)
  assert.match(source, /label: '置顶'/)
  assert.match(source, /async function topOwnResource\(\)/)
  assert.match(source, /listTopVouchers\(ownerMerchantId\.value\)/)
  assert.match(source, /redeemTopVoucher\(voucher\.id, resource\.value\.id, ownerMerchantId\.value\)/)
  assert.match(source, /async function purchaseTopService\(\)/)
  assert.match(source, /createQuotaPackOrder\(ownerMerchantId\.value, pack\.code, \{ resourceId: resource\.value\.id \}\)/)
  assert.match(source, /createVIPPayment\(ownerMerchantId\.value, order\.orderId\)/)
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
