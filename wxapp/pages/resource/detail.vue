<template>
  <view class="resource-page">
    <view v-if="resourceUnavailable" class="unavailable-state">
      <text class="unavailable-title">内容暂不可查看</text>
      <text class="unavailable-desc">该内容可能正在审核、已下架或已过期。你可以继续搜索同类内容，或返回首页查看平台推荐。</text>
      <view class="unavailable-actions">
        <button class="primary-button" @click="openSearch">去找其他内容</button>
        <button @click="backHome">返回首页</button>
      </view>
    </view>

    <view v-else class="detail-content">
      <view class="detail-gallery">
        <swiper
          v-if="galleryImages.length > 1"
          class="gallery-main gallery-swiper"
          indicator-dots
          indicator-color="rgba(255, 255, 255, 0.55)"
          indicator-active-color="#ffffff"
          duration="450"
          easing-function="easeInOutCubic"
          @change="handleGalleryChange"
        >
          <swiper-item v-for="(url, index) in galleryImages" :key="`${url}-${index}`">
            <image class="gallery-slide-image" :src="url" mode="aspectFill" @click="previewGalleryImage(index)" />
          </swiper-item>
        </swiper>
        <image
          v-else-if="mainImage"
          class="gallery-main"
          :src="mainImage"
          mode="aspectFill"
          @click="previewGalleryImage(0)"
        />
        <view v-else :class="['gallery-main', 'gallery-placeholder', isDemandResource ? 'demand' : '']">
          <view class="placeholder-copy">
            <view class="placeholder-head">
              <text class="placeholder-badge">{{ noImageBadgeText }}</text>
              <text class="placeholder-type">{{ resourceTypeDisplay }}</text>
            </view>
            <text class="placeholder-title">{{ noImageStateTitle }}</text>
            <text class="placeholder-desc">{{ noImageHintText }}</text>
          </view>
          <button v-if="canEditOwnResourceWithoutImage" class="placeholder-edit-button" @click.stop="openPublishEditor">补充图片</button>
        </view>
      </view>

      <view class="resource-card">
        <view class="tag-row">
          <text v-if="isVIPMerchant" class="tag vip">VIP</text>
          <text v-if="isOwnResource && resource.status" class="tag">{{ statusText[resource.status] || resource.status }}</text>
          <text v-if="resource.refreshedAt" class="tag">{{ resource.refreshedAt }}</text>
        </view>
        <view class="title-row">
          <text class="title">{{ resource.title }}</text>
          <button class="favorite-button" @click="toggleFavorite">{{ favorited ? '已收藏' : '收藏' }}</button>
        </view>
        <text v-if="resource.priceText" class="price">{{ resource.priceText }}</text>
        <view class="spec-list">
          <view v-for="item in specItems" :key="item.label" class="spec-item">
            <text class="spec-label">{{ item.label }}</text>
            <text class="spec-value">{{ item.value }}</text>
          </view>
        </view>
        <view v-if="resourceAddressLocations.length" class="resource-address-list">
          <view v-for="item in resourceAddressLocations" :key="item.key" class="resource-address-card">
            <view class="resource-address-head">
              <text class="resource-address-title">{{ item.label }}</text>
              <button class="address-action" @click="openResourceAddressLocation(item)">导航</button>
            </view>
            <map
              class="resource-address-map"
              :latitude="item.latitude"
              :longitude="item.longitude"
              :markers="item.markers"
              :scale="17"
              @tap="openResourceAddressLocation(item)"
            />
            <text class="resource-address-text">{{ item.address }}</text>
          </view>
        </view>
        <text class="desc">{{ resource.description || '商家暂未填写详细描述，建议联系前确认数量、尺码、看样方式和交付时间。' }}</text>
      </view>

      <view class="trust-card">
        <text class="section-title">友情提示</text>
        <text class="section-content contact-tip-content">联系商家前，建议先确认实物、价格、数量和交付方式。</text>
      </view>

      <view v-if="showMerchantHomeEntry" class="merchant-card" @click="openMerchant">
        <image v-if="merchantAvatarUrl" class="merchant-avatar" :src="merchantAvatarUrl" mode="aspectFill" />
        <view v-else class="merchant-avatar merchant-avatar-placeholder">
          <text>{{ merchantAvatarText }}</text>
        </view>
        <view class="merchant-info">
          <MerchantBadge :merchant="merchantInfo" />
          <text class="merchant-hint">{{ merchantBusinessText }}</text>
        </view>
        <text class="merchant-arrow">›</text>
      </view>

      <view v-if="relatedResources.length" class="related-section">
        <view class="section-head">
          <text class="section-title">同类推荐</text>
          <text class="section-link" @click="openSearch">查看更多</text>
        </view>
        <ResourceList
          :resources="relatedResources"
          variant="compact"
          empty-text="暂无同类供应"
          @open="openRelatedResource"
        />
      </view>

      <view v-if="showManagementSheet" class="sheet-mask" @click="closeManagementSheet">
        <view class="management-sheet" @click.stop>
          <view class="sheet-head">
            <view class="sheet-copy">
              <text class="sheet-title">{{ managementTitle }}</text>
              <text v-if="!managementActions.length" class="sheet-desc">{{ managementNotice }}</text>
            </view>
            <button class="sheet-close" @click="closeManagementSheet">关闭</button>
          </view>
          <view v-if="managementActions.length" class="management-actions">
            <button
              v-for="action in managementActions"
              :key="action.key"
              :class="['management-action', action.primary ? 'primary' : '', action.danger ? 'danger' : '']"
              @click="handleManagementAction(action.key)"
            >
              {{ action.label }}
            </button>
          </view>
          <text v-else class="empty-management">暂无可操作功能</text>
        </view>
      </view>

      <view v-if="showContactMoreSheet" class="sheet-mask" @click="closeContactMoreSheet">
        <view class="management-sheet contact-more-sheet" @click.stop>
          <view class="sheet-head">
            <view class="sheet-copy">
              <text class="sheet-title">更多操作</text>
              <text class="sheet-desc">分享给同行或反馈问题资源。</text>
            </view>
            <button class="sheet-close" @click="closeContactMoreSheet">关闭</button>
          </view>
          <view class="management-actions">
            <button class="management-action primary" open-type="share" @click="shareResourceFromMore">分享给朋友</button>
            <button class="management-action danger" @click="reportResourceFromMore">举报</button>
          </view>
        </view>
      </view>

      <view v-if="isOwnResource" class="owner-action-bar">
        <button class="share-button" @click="shareOwnResource" :open-type="canShareOwnResource ? 'share' : ''">分享</button>
        <button class="primary-button" @click="openManagementSheet">管理</button>
      </view>

      <view v-else class="contact-bar">
        <button @click="copyWechat">复制微信</button>
        <button class="primary-button" @click="callPhone">{{ contactButtonText }}</button>
        <button class="more-button" @click="openContactMoreSheet">更多</button>
      </view>

      <canvas
        canvas-id="resourceShareCoverCanvas"
        id="resourceShareCoverCanvas"
        class="share-cover-canvas"
        :width="shareCoverCanvasSize.width"
        :height="shareCoverCanvasSize.height"
      />
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onReady, onShareAppMessage, onShareTimeline } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import ResourceList from '../../components/ResourceList.vue'
import { listTopVouchers, redeemTopVoucher } from '../../api/entitlement'
import { getResourceFavoriteState, setResourceFavorite } from '../../api/favorite'
import { getMerchant } from '../../api/merchant'
import {
  createContactUnlockOrder,
  createContactUnlockPayment,
  deleteTakenDownResource,
  getOwnResource,
  getResource,
  listResources,
  recordResourceContact,
  recordResourceDetailView,
  refreshResource,
  takeDownResource,
} from '../../api/resource'
import { createQuotaPackOrder, createVIPPayment, listQuotaPacks } from '../../api/vip'
import { requireLogin } from '../../common/auth'
import { resourceTypeLabel as resolveResourceTypeLabel } from '../../common/resourceCategories'
import {
  RESOURCE_SHARE_COVER_CANVAS_ID,
  RESOURCE_SHARE_COVER_SIZE,
  buildResourceSharePayload,
  buildResourceSharePosterModel,
  buildResourceTimelinePayload,
  getResourceShareCoverSource,
} from '../../common/resourceShare'
import { getSession } from '../../store/session'

const resource = ref({})
const merchantProfile = ref({})
const relatedResources = ref([])
const favorited = ref(false)
const isOwnResource = ref(false)
const ownerMerchantId = ref('')
const resourceUnavailable = ref(false)
const selectedGalleryIndex = ref(0)
const showManagementSheet = ref(false)
const managementBusy = ref(false)
const topServicePacks = ref([])
const shareImageUrl = ref('')
const shareCoverCanvasSize = RESOURCE_SHARE_COVER_SIZE
// 底部只保留高频联系动作，分享和举报收进更多操作，减少详情页主路径干扰。
const showContactMoreSheet = ref(false)
const SEARCH_KEY = 'wplink_pending_search_keyword'
const fallbackTopServicePacks = [
  { code: 'top_1d', name: '1天置顶服务', standardPriceCent: 10000, salePriceCent: 10000, description: '购买后可置顶 1 天', saleLabel: '置顶 1 天', benefits: { topVoucherCount: 1, topDurationHours: 24 } },
  { code: 'top_3d', name: '3天置顶服务', standardPriceCent: 20000, salePriceCent: 20000, description: '购买后可置顶 3 天', saleLabel: '置顶 3 天', benefits: { topVoucherCount: 1, topDurationHours: 72 } },
  { code: 'top_5d', name: '5天置顶服务', standardPriceCent: 30000, salePriceCent: 30000, description: '购买后可置顶 5 天', saleLabel: '置顶 5 天', benefits: { topVoucherCount: 1, topDurationHours: 120 } },
  { code: 'top_7d', name: '7天置顶服务', standardPriceCent: 40000, salePriceCent: 40000, description: '购买后可置顶 7 天', saleLabel: '置顶 7 天', benefits: { topVoucherCount: 1, topDurationHours: 168 } },
  { code: 'top_15d', name: '15天置顶服务', standardPriceCent: 60000, salePriceCent: 60000, description: '购买后可置顶 15 天', saleLabel: '置顶 15 天', benefits: { topVoucherCount: 1, topDurationHours: 360 } },
  { code: 'top_30d', name: '30天置顶服务', standardPriceCent: 90000, salePriceCent: 90000, description: '购买后可置顶 30 天', saleLabel: '置顶 30 天', benefits: { topVoucherCount: 1, topDurationHours: 720 } },
]
let shareCanvasReady = false
let shareCoverRenderTimer = null
let shareCoverRendering = false
const merchantTypeText = {
  individual: '个人',
  rental_provider: '场地/设备方',
  factory: '源头工厂',
  stall: '现货档口',
  stockist: '库存货源',
  service_provider: '配套服务',
  buyer: '采购',
}
const statusText = {
  draft: '草稿',
  rejected: '已驳回',
  published: '已发布',
  pending: '内容审核中',
  manual_review: '内容审核中',
  audit_retry: '内容审核中',
  expired: '已过期',
  dealt: '已成交',
  taken_down: '已下架',
}
const contentAuditStatuses = new Set(['pending', 'manual_review', 'audit_retry'])
const isVIPMerchant = computed(() => (resource.value.merchant || {}).vipStatus === 'active')
const contactAccess = computed(() => resource.value.contactAccess || {})
// 底部主按钮执行的是电话解锁和拨号，按钮文案保持动作导向，避免展示“登录后免费查看”等规则说明。
const contactButtonText = computed(() => '拨打电话')
const merchantInfo = computed(() => ({
  ...(resource.value.merchant || {}),
  ...(merchantProfile.value || {}),
}))
const showMerchantHomeEntry = computed(() => {
  const merchantId = (merchantInfo.value || {}).id
  return Boolean(merchantId) && merchantInfo.value.profileStatus === 'completed'
})
const merchantAvatarUrl = computed(() => merchantProfile.value.logoUrl || merchantInfo.value.logoUrl || merchantInfo.value.avatarUrl || '')
const merchantAvatarText = computed(() => {
  const name = merchantInfo.value.name || '商家'
  return name.slice(0, 1)
})
const merchantBusinessText = computed(() => {
  const mainCategories = merchantInfo.value.mainCategories || []
  if (mainCategories.length > 0) return mainCategories.join('、')
  return merchantTypeText[merchantInfo.value.merchantType] || merchantInfo.value.merchantType || '主营品类待补充'
})
const galleryImages = computed(() => {
  const images = resource.value.images || []
  const cover = resource.value.coverUrl ? [resource.value.coverUrl] : []
  return [...cover, ...images].filter(Boolean)
})
const mainImage = computed(() => galleryImages.value[selectedGalleryIndex.value] || galleryImages.value[0] || '')
const resourceTypeDisplay = computed(() => resolveResourceTypeLabel(resource.value) || resource.value.category || '供需信息')
const isDemandResource = computed(() => {
  const direction = String(resource.value.direction || '').trim()
  if (direction === 'demand') return true
  const typeCode = String(resource.value.typeCode || '').trim()
  // 公开详情兼容历史响应可能缺少 direction 的情况，避免求购/求租类无图时仍显示供应口径。
  return /^(buy_|find_|seek_)/.test(typeCode) || typeCode.endsWith('_buy') || typeCode === 'job_seeking'
})
const noImageBadgeText = computed(() => (isDemandResource.value ? '需求信息' : '供需信息'))
const noImageStateTitle = computed(() => (isDemandResource.value ? '需求暂无图片' : '暂无实拍图片'))
const noImageHintText = computed(() => {
  if (isOwnResource.value) return '当前未上传图片，补充后详情展示会更完整。'
  if (isDemandResource.value) return '重点需求信息已整理在下方详情中。'
  return '重点供需信息已整理在下方详情中。'
})
const attributeSpecItems = computed(() => (resource.value.attributeItems || [])
  .filter((item) => item?.label && item?.value !== undefined && item?.value !== '')
  .map((item) => ({
    label: item.label,
    value: item.value,
  })))
const attributeLabelByKey = computed(() => {
  const labels = {}
  for (const item of resource.value.attributeItems || []) {
    if (item?.key && item?.label) labels[item.key] = item.label
  }
  return labels
})
const resourceAddressLocations = computed(() => {
  const attributes = resource.value.attributes || {}
  return Object.entries(attributes)
    .map(([key, value], index) => buildResourceAddressLocation(key, value, index))
    .filter(Boolean)
})
const attributeSpecValues = computed(() => new Set(attributeSpecItems.value.map((item) => String(item.value))))
const summarySpecItems = computed(() => [
  { label: '分类摘要', value: resource.value.category },
  { label: '数量摘要', value: resource.value.quantityText },
  { label: '价格摘要', value: resource.value.priceText },
].filter((item) => item.value && !attributeSpecValues.value.has(String(item.value))))
const specItems = computed(() => [
  ...attributeSpecItems.value,
  ...summarySpecItems.value,
  { label: '刷新', value: resource.value.refreshedAt || '近期更新' },
])
const isExpiredResource = computed(() => {
  if (resource.value.status === 'expired') return true
  if (!resource.value.expiresAt) return false
  const expiresAt = Date.parse(resource.value.expiresAt)
  return !Number.isNaN(expiresAt) && expiresAt <= Date.now()
})
const isDealtResource = computed(() => resource.value.status === 'dealt' || Boolean(resource.value.dealtAt))
const canShareOwnResource = computed(() => resource.value.status === 'published' && !isExpiredResource.value && !resource.value.dealtAt)
const canEditOwnResourceWithoutImage = computed(() => isOwnResource.value && ['draft', 'rejected'].includes(resource.value.status))
const managementTitle = computed(() => statusText[resource.value.status] || '供应管理')
const managementNotice = computed(() => {
  if (isContentAuditStatus(resource.value.status)) {
    return '供应正在内容审核中，审核通过后会公开展示。当前暂不能刷新、下架或分享。'
  }
  if (resource.value.status === 'draft') return '草稿可继续编辑，完善后再提交审核。'
  if (resource.value.status === 'rejected') return resource.value.rejectReason ? `驳回原因：${resource.value.rejectReason}` : '供应已被驳回，可编辑后重新提交审核。'
  if (isExpiredResource.value) return '供应已过期，建议再发类似供应后重新提交审核。'
  if (isDealtResource.value) return '供应已成交，不再公开展示，可再发类似供应。'
  if (resource.value.status === 'taken_down') return '供应已下架，不再公开展示。'
  return '供应展示中，可按需刷新、置顶或下架。'
})
const managementActions = computed(() => {
  if (isContentAuditStatus(resource.value.status)) return []
  if (resource.value.status === 'draft' || resource.value.status === 'rejected') {
    return [{ key: 'edit', label: '编辑', primary: true }]
  }
  if (resource.value.status === 'taken_down') {
    return [
      { key: 'repost', label: '再发类似', primary: true },
      { key: 'delete', label: '删除', danger: true },
    ]
  }
  if (isExpiredResource.value || isDealtResource.value) {
    return [{ key: 'repost', label: '再发类似', primary: true }]
  }
  if (resource.value.status === 'published') {
    return [
      { key: 'refresh', label: '刷新', primary: true },
      { key: 'top', label: '置顶', primary: true },
      { key: 'take-down', label: '下架', danger: true },
    ]
  }
  return []
})

function isContentAuditStatus(status) {
  return contentAuditStatuses.has(status)
}

function updateNavigationTitle() {
  if (typeof uni.setNavigationBarTitle !== 'function') return
  uni.setNavigationBarTitle({
    title: isDemandResource.value ? '需求详情' : '供应详情',
  })
}

onLoad(async (options) => {
  if (!options.id) return
  // 从“我的发布”进入时允许查看待审核、草稿、已下架等非公开状态，避免误提示供应已下架。
  ownerMerchantId.value = options.merchantId || ''
  isOwnResource.value = options.from === 'my-resources' || Boolean(ownerMerchantId.value)
  resourceUnavailable.value = false
  selectedGalleryIndex.value = 0
  try {
    resource.value = isOwnResource.value ? await getOwnResource(options.id, ownerMerchantId.value, { suppressErrorToast: true }) : await getResource(options.id, { suppressErrorToast: true })
    updateNavigationTitle()
  } catch (err) {
    if (!isOwnResource.value && await loadOwnResourceIfCurrentMerchant(options.id)) {
      return
    }
    resourceUnavailable.value = true
    resource.value = {}
    selectedGalleryIndex.value = 0
    return
  }
  await loadMerchantProfile()
  enableShareMenu()
  scheduleShareCoverRender()
  if (!isOwnResource.value) {
    await recordResourceDetailView(options.id)
    await loadFavoriteState(options.id)
    await loadRelatedResources()
  }
})

onReady(() => {
  shareCanvasReady = true
  if (resource.value.id) updateNavigationTitle()
  scheduleShareCoverRender()
})

async function loadOwnResourceIfCurrentMerchant(resourceId) {
  const session = getSession()
  if (!session.merchantId) return false
  try {
    ownerMerchantId.value = session.merchantId
    resource.value = await getOwnResource(resourceId, session.merchantId, { suppressErrorToast: true })
    isOwnResource.value = true
    resourceUnavailable.value = false
    selectedGalleryIndex.value = 0
    updateNavigationTitle()
    await loadMerchantProfile()
    enableShareMenu()
    scheduleShareCoverRender()
    return true
  } catch (err) {
    return false
  }
}

async function reloadOwnResource() {
  if (!resource.value.id || !ownerMerchantId.value) return
  resource.value = await getOwnResource(resource.value.id, ownerMerchantId.value, { suppressErrorToast: true })
  updateNavigationTitle()
  await loadMerchantProfile()
}

function handleGalleryChange(event) {
  const current = Number(event.detail?.current) || 0
  selectedGalleryIndex.value = current
}

function previewGalleryImage(index = selectedGalleryIndex.value) {
  if (!galleryImages.value.length) return
  const current = galleryImages.value[index] || galleryImages.value[0]
  uni.previewImage({
    current,
    urls: galleryImages.value,
  })
}

async function loadMerchantProfile() {
  const merchantId = (resource.value.merchant || {}).id
  if (!merchantId) {
    merchantProfile.value = {}
    return
  }
  try {
    merchantProfile.value = await getMerchant(merchantId, { suppressErrorToast: true })
  } catch (err) {
    merchantProfile.value = {}
  }
}

async function loadFavoriteState(resourceId) {
  if (!getSession().token) return
  try {
    const resp = await getResourceFavoriteState(resourceId)
    favorited.value = Boolean(resp.favorited)
  } catch (err) {
    favorited.value = false
  }
}

async function toggleFavorite() {
  if (!resource.value.id) return
  if (isOwnResource.value) {
    uni.showToast({ title: '不能收藏自己发布的供应', icon: 'none' })
    return
  }
  try {
    // 收藏状态以服务端返回为准，避免弱网下本地乐观更新和真实状态不一致。
    const resp = await setResourceFavorite(resource.value.id, !favorited.value)
    favorited.value = Boolean(resp.favorited)
    uni.showToast({ title: favorited.value ? '已收藏供应' : '已取消收藏', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '收藏失败，请稍后重试', icon: 'none' })
  }
}

async function recordContact(action) {
  if (!resource.value.id) return false
  if (isOwnResource.value) {
    return false
  }
  if (isContactUnlockAction(action) && !requireLogin()) return false
  try {
    const resp = await recordResourceContact(resource.value.id, action)
    return resp || {}
  } catch (err) {
    if (err?.code === 'PAYMENT_REQUIRED' && isContactUnlockAction(action)) {
      return unlockPaidContact(action)
    }
    return false
  }
}

function isContactUnlockAction(action) {
  return action === 'phone' || action === 'wechat'
}

async function unlockPaidContact(action) {
  if (!resource.value.id || !requireLogin()) return false
  try {
    // 订单价格和权限由后端按二级分类商业规则计算，前端只负责支付和支付后的再次解锁。
    const order = await createContactUnlockOrder(resource.value.id, { action })
    if (order.alreadyUnlocked) {
      const unlocked = await recordResourceContact(resource.value.id, action)
      return unlocked || {}
    }
    if (!order.orderId) {
      uni.showToast({ title: order.message || '暂时无法创建查看订单', icon: 'none' })
      return false
    }
    const payment = await createContactUnlockPayment(resource.value.id, order.orderId, {})
    if (payment.payment?.package) {
      await requestContactUnlockPayment(payment.payment)
    }
    const resp = await recordResourceContact(resource.value.id, action)
    return resp || {}
  } catch (err) {
    if (err?.message) {
      uni.showToast({ title: err.message, icon: 'none' })
    }
    return false
  }
}

function requestContactUnlockPayment(payment) {
  return new Promise((resolve, reject) => {
    if (!payment?.package || typeof uni.requestPayment !== 'function') {
      resolve()
      return
    }
    uni.requestPayment({
      timeStamp: payment.timeStamp,
      nonceStr: payment.nonceStr,
      package: payment.package,
      signType: payment.signType,
      paySign: payment.paySign,
      success: resolve,
      fail: reject,
    })
  })
}

async function openMerchant() {
  if (!showMerchantHomeEntry.value) return
  const merchantId = (resource.value.merchant || {}).id
  if (!merchantId) return
  await recordContact('merchant_home')
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId}` })
}

function buildResourceAddressLocation(key, value, index) {
  if (!value || typeof value !== 'object') return null
  const latitude = Number(value.latitude)
  const longitude = Number(value.longitude)
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return null
  const address = normalizeResourceAddressText(value)
  if (!address) return null
  const label = attributeLabelByKey.value[key] || '地址'
  return {
    key,
    label,
    address,
    name: String(value.name || label || resource.value.title || '').trim(),
    latitude,
    longitude,
    markers: [{
      id: index + 1,
      latitude,
      longitude,
      title: address,
    }],
  }
}

function normalizeResourceAddressText(value) {
  if (typeof value === 'string') return value.trim()
  if (!value || typeof value !== 'object') return ''
  return String(value.address || value.name || '').trim()
}

function openResourceAddressLocation(item) {
  if (!item) return
  uni.openLocation({
    latitude: item.latitude,
    longitude: item.longitude,
    name: item.name || resource.value.title || item.label,
    address: item.address,
    scale: 18,
    fail() {
      if (item.address) {
        uni.setClipboardData({ data: item.address })
        uni.showToast({ title: '导航打开失败，已复制地址', icon: 'none' })
      }
    },
  })
}

async function loadRelatedResources() {
  if (!resource.value.typeCode) return
  const resp = await listResources({ typeCode: resource.value.typeCode, page: 1, pageSize: 4 })
  relatedResources.value = (resp.items || []).filter((item) => item.id !== resource.value.id).slice(0, 3)
}

function openRelatedResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openSearch() {
  if (resource.value.category) {
    uni.setStorageSync(SEARCH_KEY, resource.value.category)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.navigateTo({ url: '/pages/search/index' })
}

function backHome() {
  uni.switchTab({ url: '/pages/home/index' })
}

function openManagementSheet() {
  showManagementSheet.value = true
}

function closeManagementSheet() {
  showManagementSheet.value = false
}

function openContactMoreSheet() {
  showContactMoreSheet.value = true
}

function closeContactMoreSheet() {
  showContactMoreSheet.value = false
}

function shareOwnResource() {
  if (canShareOwnResource.value) return
  uni.showToast({ title: '供应审核通过后可分享', icon: 'none' })
}

async function handleManagementAction(action) {
  if (managementBusy.value) return
  managementBusy.value = true
  try {
    if (action === 'edit') {
      openPublishEditor()
      return
    }
    if (action === 'refresh') {
      await refreshOwnResource()
      return
    }
    if (action === 'top') {
      await topOwnResource()
      return
    }
    if (action === 'take-down') {
      await takeDownOwnResource()
      return
    }
    if (action === 'repost') {
      await repostOwnResource()
      return
    }
    if (action === 'delete') {
      await deleteOwnResource()
    }
  } finally {
    managementBusy.value = false
  }
}

function openPublishEditor() {
  if (!resource.value.id || !ownerMerchantId.value) return
  closeManagementSheet()
  uni.navigateTo({ url: `/pages/publish/edit?merchantId=${ownerMerchantId.value}&resourceId=${resource.value.id}` })
}

async function refreshOwnResource() {
  await refreshResource(resource.value.id, ownerMerchantId.value)
  uni.showToast({ title: '已刷新', icon: 'none' })
  closeManagementSheet()
  await reloadOwnResource()
}

async function topOwnResource() {
  const voucher = await getAvailableTopVoucher()
  if (!voucher) {
    await purchaseTopService()
    return
  }
  const confirmed = await confirmTopVoucherUse(voucher)
  if (!confirmed) return
  await redeemTopVoucher(voucher.id, resource.value.id, ownerMerchantId.value)
  uni.showToast({ title: '已置顶', icon: 'none' })
  closeManagementSheet()
  await reloadOwnResource()
}

async function takeDownOwnResource() {
  const confirmed = await confirmManagementAction({
    title: '下架供应',
    content: '下架后供应将不再公开展示，确认下架吗？',
    confirmText: '下架',
    confirmColor: '#c2410c',
  })
  if (!confirmed) return
  await takeDownResource(resource.value.id, ownerMerchantId.value, '商家主动下架')
  uni.showToast({ title: '已下架', icon: 'none' })
  closeManagementSheet()
  await reloadOwnResource()
}

async function getAvailableTopVoucher() {
  // 置顶券余额以服务端为准，避免用户在不同入口重复核销同一批次权益。
  const resp = await listTopVouchers(ownerMerchantId.value)
  return (resp.items || []).find(isAvailableTopVoucher)
}

function isAvailableTopVoucher(voucher) {
  if (Number(voucher.remainingAmount || 0) <= 0) return false
  if (!voucher.expiresAt) return true
  const expiresAt = Date.parse(voucher.expiresAt)
  return Number.isNaN(expiresAt) || expiresAt > Date.now()
}

function confirmTopVoucherUse(voucher) {
  return new Promise((resolve) => {
    uni.showModal({
      title: '置顶供应',
      content: `将消耗 1 张置顶券，置顶 ${topDurationText(voucher)}，确认使用吗？`,
      confirmText: '置顶',
      confirmColor: '#061625',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}

function topDurationText(voucher) {
  const hours = Number(voucher.topDurationHours || 24)
  if (hours > 0 && hours % 24 === 0) return `${hours / 24} 天`
  return `${hours || 24} 小时`
}

async function purchaseTopService() {
  const pack = await chooseTopServicePack()
  if (!pack) return
  const confirmed = await confirmTopServicePurchase(pack)
  if (!confirmed) return
  closeManagementSheet()
  try {
    const order = await createQuotaPackOrder(ownerMerchantId.value, pack.code, { resourceId: resource.value.id })
    await payTopServiceOrder(order)
    uni.showToast({ title: '置顶服务已购买，置顶生效中', icon: 'none' })
    await reloadOwnResource()
  } catch (err) {
    uni.showToast({ title: err?.message || '置顶服务购买失败，请稍后重试', icon: 'none' })
  }
}

async function chooseTopServicePack() {
  const packs = await loadTopServicePacks()
  if (!packs.length) {
    uni.showToast({ title: '暂无可购买的置顶服务', icon: 'none' })
    return null
  }
  if (packs.length === 1) return packs[0]
  return new Promise((resolve) => {
    uni.showActionSheet({
      itemList: packs.map(topServiceOptionText),
      success: (res) => resolve(packs[res.tapIndex] || null),
      fail: () => resolve(null),
    })
  })
}

async function loadTopServicePacks() {
  if (topServicePacks.value.length) return topServicePacks.value
  try {
    const resp = await listQuotaPacks()
    topServicePacks.value = (resp.items || []).filter(isTopServicePack)
  } catch (err) {
    topServicePacks.value = []
  }
  if (!topServicePacks.value.length) {
    topServicePacks.value = fallbackTopServicePacks
  }
  return topServicePacks.value
}

function isTopServicePack(item) {
  const benefits = item.benefits || {}
  return Number(benefits.topVoucherCount || 0) === 1 && Number(benefits.topDurationHours || 0) > 0
}

function topServiceOptionText(item) {
  return topServicePurchaseText(item).join(' · ')
}

function topServiceName(item) {
  return String(item.name || '置顶服务').replace(/置顶券/g, '置顶服务')
}

function topServiceSaleLabel(item) {
  return String(item.saleLabel || '').trim()
}

function topServicePurchaseText(item) {
  const parts = [topServiceName(item)]
  const saleLabel = topServiceSaleLabel(item)
  if (saleLabel) parts.push(saleLabel)
  parts.push(formatTopServicePrice(item))
  if (isTopServiceDiscounted(item)) {
    parts.push(`原价${formatTopServiceCent(item.standardPriceCent)}`)
  }
  return parts
}

function isTopServiceDiscounted(item) {
  const salePriceCent = Number(item.salePriceCent || 0)
  const standardPriceCent = Number(item.standardPriceCent || 0)
  return salePriceCent > 0 && standardPriceCent > 0 && salePriceCent < standardPriceCent
}

function formatTopServicePrice(item) {
  return formatTopServiceCent(item.salePriceCent || item.standardPriceCent)
}

function formatTopServiceCent(value) {
  const price = Number(value || 0) / 100
  return `¥${Number.isInteger(price) ? price.toFixed(0) : price.toFixed(1)}`
}

function confirmTopServicePurchase(pack) {
  return new Promise((resolve) => {
    uni.showModal({
      title: '购买置顶服务',
      content: `将购买 ${topServicePurchaseText(pack).join('，')}，支付成功后直接置顶当前供应，确认继续吗？`,
      confirmText: '购买',
      cancelText: '取消',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}

async function payTopServiceOrder(order) {
  const resp = await createVIPPayment(ownerMerchantId.value, order.orderId)
  if (resp.status === 'paid') return
  const payment = resp.payment || {}
  if (!payment.timeStamp || !payment.nonceStr || !payment.package || !payment.paySign) {
    throw new Error('支付参数无效，请稍后重试')
  }
  await requestWechatPayment(payment)
}

function requestWechatPayment(payment) {
  return new Promise((resolve, reject) => {
    uni.requestPayment({
      timeStamp: payment.timeStamp,
      nonceStr: payment.nonceStr,
      package: payment.package,
      signType: payment.signType || 'RSA',
      paySign: payment.paySign,
      success: resolve,
      fail: reject,
    })
  })
}

async function repostOwnResource() {
  const detail = await getOwnResource(resource.value.id, ownerMerchantId.value)
  uni.setStorageSync('publish:repost-initial-form', buildRepostInitialForm(detail))
  closeManagementSheet()
  uni.navigateTo({ url: `/pages/publish/edit?merchantId=${ownerMerchantId.value}&repost=1` })
}

function buildRepostInitialForm(detail) {
  return {
    merchantId: ownerMerchantId.value,
    cityCode: detail.cityCode || 'zhili',
    typeCode: detail.typeCode || '',
    title: detail.title || '',
    category: detail.category || '',
    quantityText: detail.quantityText || '',
    priceText: detail.priceText || '',
    description: detail.description || '',
    attributes: detail.attributes || {},
    tags: detail.tags || [],
    images: detail.images || [],
    contact: {
      name: detail.contact?.name || '',
      phone: detail.contact?.phone || detail.contact?.phoneMasked || '',
      wechat: detail.contact?.wechat || detail.contact?.wechatMasked || '',
    },
  }
}

async function deleteOwnResource() {
  const confirmed = await confirmManagementAction({
    title: '删除供应',
    content: '删除后将不再显示在我的发布中，确认删除吗？',
    confirmText: '删除',
    confirmColor: '#c2410c',
  })
  if (!confirmed) return
  await deleteTakenDownResource(resource.value.id, ownerMerchantId.value)
  uni.showToast({ title: '已删除', icon: 'none' })
  closeManagementSheet()
  resourceUnavailable.value = true
}

function confirmManagementAction(options) {
  return new Promise((resolve) => {
    uni.showModal({
      title: options.title,
      content: options.content,
      confirmText: options.confirmText,
      confirmColor: options.confirmColor || '#061625',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}

async function callPhone() {
  const resp = await recordContact('phone')
  if (!resp) return
  if (resp.phone) {
    uni.makePhoneCall({ phoneNumber: resp.phone })
    return
  }
  uni.showToast({ title: '商家暂未填写电话', icon: 'none' })
}

async function copyWechat() {
  const resp = await recordContact('wechat')
  if (!resp) return
  if (resp.wechat) {
    uni.setClipboardData({ data: resp.wechat })
    uni.showToast({ title: '微信号已复制', icon: 'none' })
    return
  }
  uni.showToast({ title: '商家暂未填写微信，可电话联系', icon: 'none' })
}

async function shareResource() {
  if (isOwnResource.value) return
  await recordContact('share')
}

async function shareResourceFromMore() {
  await shareResource()
  closeContactMoreSheet()
}

async function reportResourceFromMore() {
  closeContactMoreSheet()
  openResourceReportPage()
}

function openResourceReportPage() {
  if (!resource.value.id || isOwnResource.value) return
  if (!requireLogin()) return
  const params = [`resourceId=${encodeURIComponent(resource.value.id)}`]
  if (resource.value.title) {
    params.push(`title=${encodeURIComponent(resource.value.title)}`)
  }
  uni.navigateTo({ url: `/pages/resource/report?${params.join('&')}` })
}

function enableShareMenu() {
  if (typeof uni.showShareMenu !== 'function') return
  // 小程序页面内按钮只能直接转发给好友/微信群；朋友圈入口需要显式开放右上角菜单。
  uni.showShareMenu({
    withShareTicket: true,
    menus: ['shareAppMessage', 'shareTimeline'],
  })
}

function scheduleShareCoverRender() {
  if (!shareCanvasReady || !resource.value.id) return
  clearTimeout(shareCoverRenderTimer)
  // 等待供需信息主图和隐藏 canvas 完成一次视图更新，避免刚加载详情时导出空白封面。
  shareCoverRenderTimer = setTimeout(() => {
    shareCoverRenderTimer = null
    renderShareCover()
  }, 80)
}

async function renderShareCover() {
  if (shareCoverRendering || !shareCanvasReady || !resource.value.id) return
  shareCoverRendering = true
  try {
    const poster = buildResourceSharePosterModel(resource.value, merchantInfo.value)
    const ctx = uni.createCanvasContext(RESOURCE_SHARE_COVER_CANVAS_ID)
    await drawResourceShareCover(ctx, poster)
    const tempFilePath = await exportShareCoverImage()
    shareImageUrl.value = tempFilePath || getResourceShareCoverSource(resource.value)
  } catch (err) {
    console.warn('供需信息分享封面生成失败', {
      resourceId: resource.value.id,
      message: err?.message || String(err),
    })
    shareImageUrl.value = getResourceShareCoverSource(resource.value)
  } finally {
    shareCoverRendering = false
  }
}

async function drawResourceShareCover(ctx, poster) {
  const { width, height } = RESOURCE_SHARE_COVER_SIZE
  const coverHeight = 280
  ctx.setFillStyle('#f7fafc')
  ctx.fillRect(0, 0, width, height)

  const coverPath = await resolveCanvasImagePath(poster.coverSource)
  if (coverPath) {
    ctx.drawImage(coverPath, 0, 0, width, coverHeight)
    drawImageShade(ctx, width, coverHeight)
  } else {
    drawCoverPlaceholder(ctx, poster, width, coverHeight)
  }

  drawBadge(ctx, poster.typeLabel, 28, 26, { background: 'rgba(6, 22, 37, 0.86)', color: '#ffffff' })
  ctx.setFillStyle('rgba(255, 255, 255, 0.92)')
  ctx.setFontSize(24)
  ctx.fillText('衣货通', width - 104, 52)

  ctx.setFillStyle('#ffffff')
  ctx.fillRect(24, 246, width - 48, 206)
  ctx.setFillStyle('#c2410c')
  ctx.fillRect(24, 246, 8, 206)

  let badgeX = 46
  for (const badge of poster.badges.slice(0, 3)) {
    const badgeWidth = drawBadge(ctx, badge, badgeX, 270, { background: '#fff4ed', color: '#c2410c' })
    badgeX += badgeWidth + 10
  }

  ctx.setFillStyle('#061625')
  ctx.setFontSize(34)
  drawWrappedText(ctx, poster.title, 46, 340, width - 92, 42, 2)

  ctx.setFillStyle('#c2410c')
  ctx.setFontSize(28)
  drawWrappedText(ctx, poster.summaryLines.join(' · ') || '欢迎联系商家确认详情', 46, 418, width - 92, 34, 1)

  ctx.setFillStyle('#64748b')
  ctx.setFontSize(22)
  ctx.fillText(poster.merchantName, 46, height - 24)
  ctx.fillText(poster.footerText, width - 214, height - 24)

  await flushCanvas(ctx)
}

function resolveCanvasImagePath(src) {
  return new Promise((resolve) => {
    if (!src || typeof uni.getImageInfo !== 'function') {
      resolve('')
      return
    }
    uni.getImageInfo({
      src,
      success: (res) => resolve(res.path || src),
      fail: () => resolve(''),
    })
  })
}

function exportShareCoverImage() {
  return new Promise((resolve) => {
    if (typeof uni.canvasToTempFilePath !== 'function') {
      resolve('')
      return
    }
    const { width, height } = RESOURCE_SHARE_COVER_SIZE
    uni.canvasToTempFilePath({
      canvasId: RESOURCE_SHARE_COVER_CANVAS_ID,
      width,
      height,
      destWidth: width,
      destHeight: height,
      fileType: 'jpg',
      quality: 0.92,
      success: (res) => resolve(res.tempFilePath || ''),
      fail: () => resolve(''),
    })
  })
}

function drawImageShade(ctx, width, coverHeight) {
  const gradient = ctx.createLinearGradient(0, 120, 0, coverHeight)
  gradient.addColorStop(0, 'rgba(6, 22, 37, 0)')
  gradient.addColorStop(1, 'rgba(6, 22, 37, 0.48)')
  ctx.setFillStyle(gradient)
  ctx.fillRect(0, 120, width, coverHeight - 120)
}

function drawCoverPlaceholder(ctx, poster, width, coverHeight) {
  const gradient = ctx.createLinearGradient(0, 0, width, coverHeight)
  gradient.addColorStop(0, '#061625')
  gradient.addColorStop(1, '#d88a80')
  ctx.setFillStyle(gradient)
  ctx.fillRect(0, 0, width, coverHeight)
  ctx.setFillStyle('rgba(255, 255, 255, 0.16)')
  for (let x = -80; x < width; x += 120) {
    ctx.fillRect(x, 0, 42, coverHeight)
  }
  ctx.setFillStyle('#ffffff')
  ctx.setFontSize(42)
  drawWrappedText(ctx, poster.typeLabel, 36, 184, width - 72, 48, 1)
}

function drawBadge(ctx, text, x, y, options = {}) {
  const label = String(text || '').slice(0, 8)
  if (!label) return 0
  const width = Math.max(58, getTextWidth(ctx, label, 22) + 26)
  ctx.setFillStyle(options.background || '#eef2f7')
  ctx.fillRect(x, y, width, 34)
  ctx.setFillStyle(options.color || '#364152')
  ctx.setFontSize(20)
  ctx.fillText(label, x + 13, y + 24)
  return width
}

function drawWrappedText(ctx, text, x, y, maxWidth, lineHeight, maxLines) {
  const chars = Array.from(String(text || '').trim())
  const lines = []
  let line = ''
  let index = 0
  for (; index < chars.length; index += 1) {
    const char = chars[index]
    const nextLine = `${line}${char}`
    if (line && getTextWidth(ctx, nextLine) > maxWidth) {
      lines.push(line)
      if (lines.length === maxLines) break
      line = char
    } else {
      line = nextLine
    }
  }
  if (line && lines.length < maxLines) lines.push(line)
  if (index < chars.length && lines.length) {
    const lastIndex = Math.min(lines.length, maxLines) - 1
    lines[lastIndex] = fitText(ctx, lines[lastIndex], maxWidth, '…')
  }
  lines.slice(0, maxLines).forEach((item, index) => {
    ctx.fillText(item, x, y + index * lineHeight)
  })
}

function fitText(ctx, text, maxWidth, suffix = '') {
  let result = String(text || '')
  while (result && getTextWidth(ctx, `${result}${suffix}`) > maxWidth) {
    result = result.slice(0, -1)
  }
  return `${result}${suffix}`
}

function getTextWidth(ctx, text, fontSize = 28) {
  if (typeof ctx.measureText === 'function') return ctx.measureText(text).width
  return String(text || '').length * fontSize
}

function flushCanvas(ctx) {
  return new Promise((resolve) => {
    ctx.draw(false, resolve)
  })
}

function trackShareFromMenu() {
  if (isOwnResource.value || !resource.value.id) return
  recordContact('share').catch(() => {})
}

onShareAppMessage((shareEvent) => {
  if (shareEvent?.from === 'menu') trackShareFromMenu()
  return buildResourceSharePayload(resource.value, shareImageUrl.value)
})

onShareTimeline(() => {
  trackShareFromMenu()
  return buildResourceTimelinePayload(resource.value, shareImageUrl.value)
})
</script>

<style lang="scss" scoped>
.resource-page {
  min-height: 100vh;
  padding: 24rpx 24rpx calc(150rpx + env(safe-area-inset-bottom));
  background: $wplink-bg;
}

.resource-card,
.merchant-card,
.trust-card {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.unavailable-state {
  display: grid;
  gap: 22rpx;
  padding: 56rpx 32rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.unavailable-title {
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.3;
}

.unavailable-desc {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.6;
}

.unavailable-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
  margin-top: 8rpx;
}

.unavailable-actions button {
  height: 76rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  color: #364152;
  font-size: 26rpx;
  line-height: 1.25;
}

.detail-gallery {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.gallery-main {
  width: 100%;
  height: 420rpx;
  border-radius: 12rpx;
  background: #edf2f7;
  overflow: hidden;
}

.gallery-slide-image {
  width: 100%;
  height: 100%;
  display: block;
}

.gallery-placeholder {
  display: grid;
  gap: 22rpx;
  box-sizing: border-box;
  height: auto;
  min-height: 240rpx;
  padding: 28rpx;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.92), rgba(255, 247, 237, 0.86)),
    repeating-linear-gradient(135deg, rgba(194, 58, 0, 0.08) 0 14rpx, transparent 14rpx 30rpx),
    #f8fafc;
  border: 1rpx solid rgba(194, 58, 0, 0.16);
}

.gallery-placeholder.demand {
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.94), rgba(236, 253, 245, 0.78)),
    repeating-linear-gradient(135deg, rgba(21, 128, 61, 0.08) 0 14rpx, transparent 14rpx 30rpx),
    #f8fafc;
  border-color: rgba(21, 128, 61, 0.16);
}

.placeholder-copy {
  display: grid;
  gap: 14rpx;
  min-width: 0;
}

.placeholder-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12rpx;
  min-width: 0;
}

.placeholder-badge {
  flex: 0 0 auto;
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: rgba(6, 22, 37, 0.86);
  color: $wplink-card;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.35;
}

.placeholder-type {
  min-width: 0;
  color: #475569;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
  word-break: break-word;
}

.placeholder-title {
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.28;
  word-break: break-word;
}

.placeholder-desc {
  color: #64748b;
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}

.placeholder-edit-button {
  justify-self: start;
  min-width: 172rpx;
  height: 64rpx;
  padding: 0 22rpx;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.25;
}

.placeholder-edit-button::after {
  border: 0;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.tag {
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #edf2f7;
  color: #4a5568;
  font-size: 24rpx;
}

.tag.vip {
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
}

.title-row {
  display: grid;
  grid-template-columns: 1fr 136rpx;
  gap: 16rpx;
  align-items: start;
}

.title {
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.35;
  min-width: 0;
  word-break: break-word;
}

.favorite-button {
  height: 64rpx;
  border-radius: 10rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 24rpx;
}

.meta,
.desc,
.merchant-status,
.section-content,
.merchant-hint,
.section-link {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.55;
}

.price {
  color: $wplink-warning;
  font-size: 32rpx;
  font-weight: 700;
}

.spec-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.spec-item {
  display: grid;
  gap: 6rpx;
  padding: 16rpx;
  border-radius: 10rpx;
  background: #f8fafc;
}

.spec-label {
  color: $wplink-muted;
  font-size: 24rpx;
}

.spec-value {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.35;
  word-break: break-word;
}

.resource-address-list {
  display: grid;
  gap: 14rpx;
}

.resource-address-card {
  display: grid;
  gap: 12rpx;
  padding: 16rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #f8fafc;
}

.resource-address-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.resource-address-title {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
}

.address-action {
  flex: 0 0 auto;
  height: 52rpx;
  padding: 0 18rpx;
  border-radius: 8rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 52rpx;
}

.address-action::after {
  border: 0;
}

.resource-address-map {
  width: 100%;
  height: 260rpx;
  border-radius: 10rpx;
  overflow: hidden;
  background: #e2e8f0;
}

.resource-address-text {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
  word-break: break-word;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.section-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.section-link {
  color: $wplink-primary;
}

.contact-tip-content {
  font-size: 26rpx;
  line-height: 1.5;
}

.related-section {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.merchant-card {
  grid-template-columns: 88rpx minmax(0, 1fr) 28rpx;
  align-items: center;
  gap: 18rpx;
}

.merchant-avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 12rpx;
  background: #edf2f7;
}

.merchant-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 34rpx;
  font-weight: 700;
}

.merchant-info {
  display: grid;
  gap: 6rpx;
  min-width: 0;
}

.merchant-info :deep(.merchant-badge) {
  min-width: 0;
  flex-wrap: wrap;
}

.merchant-info :deep(.merchant-name) {
  min-width: 0;
  line-height: 1.35;
  word-break: break-word;
}

.merchant-arrow {
  color: $wplink-muted;
  font-size: 44rpx;
  line-height: 1;
  text-align: right;
}

.contact-bar,
.owner-action-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  display: grid;
  gap: 16rpx;
  box-sizing: border-box;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
  z-index: 20;
}

.contact-bar {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.35fr) 104rpx;
}

.owner-action-bar {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.contact-bar button {
  height: 88rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: #f8fafc;
  color: #364152;
  font-size: 28rpx;
  line-height: 1.25;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);
}

.owner-action-bar button {
  height: 88rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: #f8fafc;
  color: #364152;
  font-size: 28rpx;
  line-height: 1.25;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);
}

.contact-bar button::after {
  border: 0;
}

.owner-action-bar button::after {
  border: 0;
}

.contact-bar .primary-button {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.14);
}

.contact-bar .more-button {
  padding: 0;
  background: $wplink-card;
  color: $wplink-primary;
  font-weight: 700;
}

.owner-action-bar .primary-button {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.14);
}

.owner-action-bar .share-button {
  background: $wplink-card;
  color: $wplink-primary;
}

.share-cover-canvas {
  position: fixed;
  left: -9999px;
  top: -9999px;
  width: 600px;
  height: 480px;
  pointer-events: none;
  opacity: 0;
}

.sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.36);
}

.management-sheet {
  display: grid;
  gap: 24rpx;
  width: 100%;
  box-sizing: border-box;
  padding: 28rpx 24rpx calc(30rpx + env(safe-area-inset-bottom));
  border-radius: 18rpx 18rpx 0 0;
  background: $wplink-card;
  box-shadow: 0 -12rpx 36rpx rgba(15, 23, 42, 0.16);
}

.sheet-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120rpx;
  gap: 16rpx;
  align-items: start;
}

.sheet-copy {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.sheet-title {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.35;
}

.sheet-desc,
.empty-management {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.55;
}

.sheet-close {
  height: 62rpx;
  border-radius: 10rpx;
  background: #f4f7fd;
  color: #364152;
  font-size: 24rpx;
  line-height: 1.25;
}

.sheet-close::after {
  border: 0;
}

.management-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14rpx;
}

.management-action {
  min-width: 0;
  height: 76rpx;
  padding: 0 12rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #f8fafc;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.25;
}

.management-action::after {
  border: 0;
}

.management-action.primary {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
}

.management-action.danger {
  border-color: #fecdd3;
  background: #fff8f8;
  color: #be123c;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}
</style>
