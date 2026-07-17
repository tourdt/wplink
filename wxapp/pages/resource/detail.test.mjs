import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

test('resource detail gallery uses banner swiper and full screen preview', () => {
  assert.match(source, /const selectedGalleryIndex = ref\(0\)/)
  assert.match(source, /<swiper[\s\S]*v-if="galleryImages\.length > 1"[\s\S]*duration="450"[\s\S]*easing-function="easeInOutCubic"[\s\S]*@change="handleGalleryChange"/)
  assert.doesNotMatch(source, /:current="selectedGalleryIndex"/)
  assert.match(source, /<swiper-item[\s\S]*v-for="\(\s*url,\s*index\s*\) in galleryImages"/)
  assert.match(source, /@click="previewGalleryImage\(index\)"/)
  assert.match(source, /v-else-if="mainImage"[\s\S]*@click="previewGalleryImage\(0\)"/)
  assert.match(source, /function handleGalleryChange\(event\) \{[\s\S]*selectedGalleryIndex\.value = current[\s\S]*\}/)
  assert.match(source, /function previewGalleryImage\(index = selectedGalleryIndex\.value\) \{[\s\S]*uni\.previewImage\(\{[\s\S]*current,[\s\S]*urls: galleryImages\.value[\s\S]*\}\)/)
  assert.equal(source.includes('gallery-strip'), false)
  assert.equal(source.includes('gallery-thumb'), false)
  assert.equal(source.includes('selectGalleryImage'), false)
})

test('resource detail replaces empty image area with an informative no-image cover', () => {
  assert.match(source, /import \{ resourceTypeLabel as resolveResourceTypeLabel \} from '\.\.\/\.\.\/common\/resourceCategories'/)
  assert.match(source, /<view v-else :class="\['gallery-main', 'gallery-placeholder', isDemandResource \? 'demand' : ''\]">/)
  assert.match(source, /<text class="placeholder-badge">\{\{ noImageBadgeText \}\}<\/text>/)
  assert.match(source, /<text class="placeholder-type">\{\{ resourceTypeDisplay \}\}<\/text>/)
  assert.match(source, /<text class="placeholder-title">\{\{ noImageStateTitle \}\}<\/text>/)
  assert.match(source, /<button v-if="canEditOwnResourceWithoutImage" class="placeholder-edit-button" @click\.stop="openPublishEditor">补充图片<\/button>/)
  assert.match(source, /const resourceTypeDisplay = computed\(\(\) => resolveResourceTypeLabel\(resource\.value\) \|\| resource\.value\.category \|\| '供需信息'\)/)
  assert.match(source, /const isDemandResource = computed\(\(\) => \{[\s\S]*direction === 'demand'[\s\S]*\/\^\(buy_\|find_\|seek_\)\/[\s\S]*typeCode === 'job_seeking'[\s\S]*\}\)/)
  assert.match(source, /const noImageStateTitle = computed\(\(\) => \(isDemandResource\.value \? '需求暂无图片' : '暂无实拍图片'\)\)/)
  assert.match(source, /重点需求信息已整理在下方详情中/)
  assert.match(source, /重点供需信息已整理在下方详情中/)
  assert.match(source, /\.gallery-placeholder \{[\s\S]*height: auto;[\s\S]*min-height: 240rpx;[\s\S]*\}/)
  assert.equal(source.includes('noImageSummaryItems'), false)
  assert.equal(source.includes('placeholder-summary'), false)
  assert.equal(source.includes('可先联系发布方确认品类、数量、预算和交付时间'), false)
  assert.equal(source.includes('联系商家前，建议确认实物、数量、价格和交付方式'), false)
  assert.equal(source.includes("{{ resource.category || '供应实拍' }}"), false)
})

test('resource detail updates navigation title by supply or demand direction', () => {
  const detailPage = pagesConfig.pages.find((item) => item.path === 'pages/resource/detail')

  assert.equal(detailPage?.style?.navigationBarTitleText, '供应详情')
  assert.equal(source.includes('供需详情'), false)
  assert.equal(JSON.stringify(detailPage).includes('供需详情'), false)
  assert.match(source, /function updateNavigationTitle\(\) \{[\s\S]*uni\.setNavigationBarTitle\(\{[\s\S]*title: isDemandResource\.value \? '需求详情' : '供应详情'[\s\S]*\}\)[\s\S]*\}/)
  assert.match(source, /resource\.value = isOwnResource\.value \? await getOwnResource[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /async function loadOwnResourceIfCurrentMerchant\(resourceId\) \{[\s\S]*resource\.value = await getOwnResource[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /async function reloadOwnResource\(\) \{[\s\S]*resource\.value = await getOwnResource[\s\S]*updateNavigationTitle\(\)/)
  assert.match(source, /onReady\(\(\) => \{[\s\S]*if \(resource\.value\.id\) updateNavigationTitle\(\)[\s\S]*\}\)/)
})

test('resource detail keeps contact reminder friendly and visually quiet', () => {
  assert.match(source, /<text class="section-title">友情提示<\/text>/)
  assert.match(source, /<text class="section-content contact-tip-content">联系商家前，建议先确认实物、价格、数量和交付方式。<\/text>/)
  assert.match(source, /\.contact-tip-content \{[\s\S]*font-size: 26rpx;[\s\S]*line-height: 1\.5;[\s\S]*\}/)
  assert.equal(source.includes('平台已记录联系行为'), false)
  assert.equal(source.includes('<text class="section-title">联系提示</text>'), false)
})

test('resource detail only shows merchant home entry after merchant profile is completed', () => {
  assert.match(source, /const showMerchantHomeEntry = computed\(\(\) => \{[\s\S]*merchantInfo\.value\.profileStatus === 'completed'[\s\S]*\}\)/)
  assert.match(source, /<view v-if="showMerchantHomeEntry" class="merchant-card" @click="openMerchant">/)
  assert.match(source, /function openMerchant\(\) \{[\s\S]*if \(!showMerchantHomeEntry\.value\) return[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/merchant\/detail\?id=\$\{merchantId\}` \}\)/)
})

test('resource detail shows vip merchant tag without certification endorsement copy', () => {
  assert.match(source, /const isVIPMerchant = computed\(\(\) => \(resource\.value\.merchant \|\| \{\}\)\.vipStatus === 'active'\)/)
  assert.match(source, /<text v-if="isVIPMerchant" class="tag vip">VIP<\/text>/)
  assert.doesNotMatch(source, /平台核实/)
  assert.doesNotMatch(source, /认证商家/)
})

test('resource detail only shows publish status for own resource', () => {
  assert.match(source, /<text v-if="isOwnResource && resource\.status" class="tag">\{\{ statusText\[resource\.status\] \|\| resource\.status \}\}<\/text>/)
  assert.doesNotMatch(source, /<text v-if="resource\.status" class="tag">/)
})

test('resource detail displays publish tags in the top tag row', () => {
  assert.match(source, /const resourceFeatureTags = computed\(\(\) => normalizeResourceFeatureTags\(resource\.value\.tags\)\)/)
  assert.match(source, /<text v-for="tag in resourceFeatureTags" :key="tag" class="tag feature">\{\{ tag \}\}<\/text>/)
  assert.match(source, /function normalizeResourceFeatureTags\(tags = \[\]\)/)
  assert.match(source, /\.tag\.feature \{[\s\S]*background: #f8fafc;[\s\S]*color: \$wplink-muted;/)
})

test('own resource detail keeps share and management actions in the bottom bar', () => {
  assert.match(source, /<view v-if="isOwnResource" class="owner-action-bar">/)
  assert.match(source, /<button class="share-button" @click="shareOwnResource" :open-type="canShareOwnResource \? 'share' : ''">分享<\/button>/)
  assert.match(source, /const canShareOwnResource = computed\(\(\) => resource\.value\.status === 'published' && !isExpiredResource\.value && !resource\.value\.dealtAt\)/)
  assert.match(source, /<button class="primary-button" @click="openManagementSheet">管理<\/button>/)
  assert.match(source, /<view v-else class="contact-bar">/)
  assert.doesNotMatch(source, /这是你发布的供需信息，可在我的发布中管理/)
})

test('pending own resource management sheet only explains review state', () => {
  assert.match(source, /const contentAuditStatuses = new Set\(\['pending', 'manual_review', 'audit_retry'\]\)/)
  assert.match(source, /const managementNotice = computed\(\(\) => \{[\s\S]*isContentAuditStatus\(resource\.value\.status\)[\s\S]*供应正在内容审核中，审核通过后会公开展示。当前暂不能刷新、下架或分享。[\s\S]*\}\)/)
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
  const contactBar = source.match(/<view v-else class="contact-bar">[\s\S]*?<\/view>/)?.[0] || ''

  assert.match(source, /const showContactMoreSheet = ref\(false\)/)
  assert.match(source, /<view v-if="showContactMoreSheet" class="sheet-mask" @click="closeContactMoreSheet">/)
  assert.match(source, /<button class="management-action primary" open-type="share" @click="shareResourceFromMore">分享给朋友<\/button>/)
  assert.match(source, /<button class="management-action danger" @click="reportResourceFromMore">举报<\/button>/)
  assert.match(source, /async function shareResourceFromMore\(\) \{[\s\S]*await shareResource\(\)[\s\S]*closeContactMoreSheet\(\)[\s\S]*\}/)
  assert.match(source, /async function reportResourceFromMore\(\) \{[\s\S]*closeContactMoreSheet\(\)[\s\S]*openResourceReportPage\(\)[\s\S]*\}/)
  assert.match(contactBar, /@click="copyWechat"/)
  assert.match(contactBar, /@click="callPhone"/)
  assert.match(contactBar, /@click="openContactMoreSheet"/)
  assert.equal(contactBar.includes('open-type="share"'), false)
  assert.equal(contactBar.includes('reportCurrentResource'), false)
  assert.match(source, /grid-template-columns: minmax\(0, 1fr\) minmax\(0, 1\.35fr\) 104rpx;/)
  assert.equal(source.includes('grid-template-columns: repeat(4, 1fr);'), false)
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
  assert.match(source, /<canvas[\s\S]*canvas-id="resourceShareCoverCanvas"[\s\S]*class="share-cover-canvas"[\s\S]*:width="shareCoverCanvasSize\.width"[\s\S]*:height="shareCoverCanvasSize\.height"/)
  assert.match(source, /uni\.showShareMenu\(\{[\s\S]*menus: \['shareAppMessage', 'shareTimeline'\][\s\S]*\}\)/)
  assert.match(source, /uni\.canvasToTempFilePath\(\{[\s\S]*canvasId: RESOURCE_SHARE_COVER_CANVAS_ID[\s\S]*success: \(res\) => resolve\(res\.tempFilePath \|\| ''\)/)
  assert.match(source, /onShareAppMessage\(\(shareEvent\) => \{[\s\S]*buildResourceSharePayload\(resource\.value, shareImageUrl\.value\)[\s\S]*\}\)/)
  assert.match(source, /onShareTimeline\(\(\) => \{[\s\S]*buildResourceTimelinePayload\(resource\.value, shareImageUrl\.value\)[\s\S]*\}\)/)
  assert.match(source, /\.share-cover-canvas \{[\s\S]*position: fixed;[\s\S]*left: -9999px;[\s\S]*width: 600px;[\s\S]*height: 480px;/)
})

test('resource detail renders configured attribute items as specs', () => {
  assert.match(source, /const attributeSpecItems = computed\(\(\) =>/)
  assert.match(source, /resource\.value\.attributeItems \|\| \[\]/)
  assert.match(source, /label: item\.label/)
  assert.match(source, /value: item\.value/)
  assert.match(source, /const summarySpecItems = computed\(\(\) =>/)
  assert.match(source, /const attributeSpecValues = computed\(\(\) =>/)
  assert.match(source, /const specItems = computed\(\(\) => \[/)
  assert.match(source, /\.\.\.attributeSpecItems\.value/)
  assert.match(source, /\.\.\.summarySpecItems\.value/)
  assert.equal(source.includes("{ label: '品类', value: resource.value.category || '待沟通' }"), false)
  assert.equal(source.includes("{ label: '数量', value: resource.value.quantityText || '待沟通' }"), false)
  assert.equal(source.includes("{ label: '价格', value: resource.value.priceText || '面议' }"), false)
})

test('resource detail shows map navigation only for address attributes with gps', () => {
  assert.match(source, /<view v-if="resourceAddressLocations\.length" class="resource-address-list">/)
  assert.match(source, /<map[\s\S]*class="resource-address-map"[\s\S]*:latitude="item\.latitude"[\s\S]*:longitude="item\.longitude"[\s\S]*@tap="openResourceAddressLocation\(item\)"/)
  assert.match(source, /const resourceAddressLocations = computed\(\(\) => \{[\s\S]*Object\.entries\(attributes\)[\s\S]*buildResourceAddressLocation/)
  assert.match(source, /function buildResourceAddressLocation\(key, value, index\) \{[\s\S]*typeof value !== 'object'[\s\S]*Number\.isFinite\(latitude\)[\s\S]*Number\.isFinite\(longitude\)[\s\S]*return null[\s\S]*markers/)
  assert.match(source, /function openResourceAddressLocation\(item\) \{[\s\S]*uni\.openLocation\(\{[\s\S]*latitude: item\.latitude[\s\S]*longitude: item\.longitude[\s\S]*scale: 18/)
  assert.match(source, /导航打开失败，已复制地址/)
})

test('resource detail restores top voucher management action', () => {
  assert.match(source, /import \{ listTopVouchers, redeemTopVoucher \} from '\.\.\/\.\.\/api\/entitlement'/)
  assert.match(source, /import \{ createQuotaPackOrder, createVIPPayment, listQuotaPacks \} from '\.\.\/\.\.\/api\/vip'/)
  assert.match(source, /供应展示中，可按需刷新、置顶或下架。/)
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
