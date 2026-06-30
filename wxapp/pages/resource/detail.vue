<template>
  <view class="resource-page">
    <view v-if="resourceUnavailable" class="unavailable-state">
      <text class="unavailable-title">资源暂不可查看</text>
      <text class="unavailable-desc">该资源可能正在审核、已下架或已过期。你可以继续搜索同类资源，或返回首页查看平台推荐。</text>
      <view class="unavailable-actions">
        <button class="primary-button" @click="openSearch">去找其他资源</button>
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
          :current="selectedGalleryIndex"
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
        <view v-else class="gallery-main gallery-placeholder">
          <text>{{ resource.category || '资源实拍' }}</text>
        </view>
      </view>

      <view class="resource-card">
        <view class="tag-row">
          <text v-if="isVerifiedMerchant" class="tag verified">平台核实</text>
          <text v-if="isVerifiedMerchant" class="tag verified">认证商家</text>
          <text v-if="resource.status" class="tag">{{ statusText[resource.status] || resource.status }}</text>
          <text v-if="resource.refreshedAt" class="tag">{{ resource.refreshedAt }}</text>
        </view>
        <view class="title-row">
          <text class="title">{{ resource.title }}</text>
          <button class="favorite-button" @click="toggleFavorite">{{ favorited ? '已收藏' : '收藏' }}</button>
        </view>
        <text class="price">{{ resource.priceText || '价格面议' }}</text>
        <view class="spec-list">
          <view v-for="item in specItems" :key="item.label" class="spec-item">
            <text class="spec-label">{{ item.label }}</text>
            <text class="spec-value">{{ item.value }}</text>
          </view>
        </view>
        <text class="desc">{{ resource.description || '商家暂未填写详细描述，建议联系前确认数量、尺码、看样方式和交付时间。' }}</text>
      </view>

      <view class="trust-card">
        <text class="section-title">友情提示</text>
        <text class="section-content contact-tip-content">联系商家前，建议先确认实物、价格、数量和交付方式。</text>
      </view>

      <view class="merchant-card" @click="openMerchant">
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
          empty-text="暂无同类资源"
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

      <view v-if="isOwnResource" class="owner-action-bar">
        <button class="share-button" @click="shareOwnResource" :open-type="canShareOwnResource ? 'share' : ''">分享</button>
        <button class="primary-button" @click="openManagementSheet">管理</button>
      </view>

      <view v-else class="contact-bar">
        <button @click="copyWechat">复制微信</button>
        <button class="primary-button" @click="callPhone">联系商家</button>
        <button open-type="share" @click="shareResource">分享</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShareAppMessage } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import ResourceList from '../../components/ResourceList.vue'
import { listTopVouchers, redeemTopVoucher } from '../../api/entitlement'
import { getResourceFavoriteState, setResourceFavorite } from '../../api/favorite'
import { getMerchant } from '../../api/merchant'
import {
  deleteTakenDownResource,
  getOwnResource,
  getResource,
  listResources,
  recordResourceContact,
  recordResourceDetailView,
  refreshResource,
  takeDownResource,
} from '../../api/resource'
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
const SEARCH_KEY = 'wplink_pending_search_keyword'
const merchantTypeText = {
  factory: '工厂',
  stall: '档口',
  stockist: '库存商',
  service_provider: '服务商',
  buyer: '采购商',
}
const statusText = {
  draft: '草稿',
  rejected: '已驳回',
  published: '已发布',
  pending: '待审核',
  expired: '已过期',
  dealt: '已成交',
  taken_down: '已下架',
}
const isVerifiedMerchant = computed(() => (resource.value.merchant || {}).verificationStatus === 'verified')
const merchantInfo = computed(() => ({
  ...(resource.value.merchant || {}),
  ...(merchantProfile.value || {}),
}))
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
const specItems = computed(() => [
  { label: '品类', value: resource.value.category || '待沟通' },
  { label: '数量', value: resource.value.quantityText || '待沟通' },
  { label: '价格', value: resource.value.priceText || '面议' },
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
const managementTitle = computed(() => statusText[resource.value.status] || '资源管理')
const managementNotice = computed(() => {
  if (resource.value.status === 'pending') {
    return '资源正在审核，审核通过后会公开展示。当前暂不能刷新、置顶、下架或分享。'
  }
  if (resource.value.status === 'draft') return '草稿可继续编辑，完善后再提交审核。'
  if (resource.value.status === 'rejected') return resource.value.rejectReason ? `驳回原因：${resource.value.rejectReason}` : '资源已被驳回，可编辑后重新提交审核。'
  if (isExpiredResource.value) return '资源已过期，建议再发类似资源后重新提交审核。'
  if (isDealtResource.value) return '资源已成交，不再公开展示，可再发类似资源。'
  if (resource.value.status === 'taken_down') return '资源已下架，不再公开展示。'
  return '资源展示中，可按需刷新、置顶或下架。'
})
const managementActions = computed(() => {
  if (resource.value.status === 'pending') return []
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
      { key: 'top', label: '置顶' },
      { key: 'take-down', label: '下架', danger: true },
    ]
  }
  return []
})

onLoad(async (options) => {
  if (!options.id) return
  // 从“我的发布”进入时允许查看待审核、草稿、已下架等非公开状态，避免误提示资源已下架。
  ownerMerchantId.value = options.merchantId || ''
  isOwnResource.value = options.from === 'my-resources' || Boolean(ownerMerchantId.value)
  resourceUnavailable.value = false
  selectedGalleryIndex.value = 0
  try {
    resource.value = isOwnResource.value ? await getOwnResource(options.id, ownerMerchantId.value, { suppressErrorToast: true }) : await getResource(options.id, { suppressErrorToast: true })
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
  if (!isOwnResource.value) {
    await recordResourceDetailView(options.id)
    await loadFavoriteState(options.id)
    await loadRelatedResources()
  }
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
    return true
  } catch (err) {
    return false
  }
}

async function reloadOwnResource() {
  if (!resource.value.id || !ownerMerchantId.value) return
  resource.value = await getOwnResource(resource.value.id, ownerMerchantId.value, { suppressErrorToast: true })
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
    uni.showToast({ title: '不能收藏自己发布的资源', icon: 'none' })
    return
  }
  try {
    // 收藏状态以服务端返回为准，避免弱网下本地乐观更新和真实状态不一致。
    const resp = await setResourceFavorite(resource.value.id, !favorited.value)
    favorited.value = Boolean(resp.favorited)
    uni.showToast({ title: favorited.value ? '已收藏资源' : '已取消收藏', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '收藏失败，请稍后重试', icon: 'none' })
  }
}

async function recordContact(action) {
  if (!resource.value.id) return false
  if (isOwnResource.value) {
    return false
  }
  await recordResourceContact(resource.value.id, action)
  return true
}

async function openMerchant() {
  const merchantId = (resource.value.merchant || {}).id
  if (!merchantId) return
  await recordContact('merchant_home')
  uni.navigateTo({ url: `/pages/merchant/detail?id=${merchantId}` })
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
  uni.navigateTo({ url: '/pages/search/result' })
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

function shareOwnResource() {
  if (canShareOwnResource.value) return
  uni.showToast({ title: '资源审核通过后可分享', icon: 'none' })
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
  const resp = await listTopVouchers(ownerMerchantId.value)
  const voucher = (resp.items || []).find((entry) => entry.status === 'unused')
  if (!voucher) {
    uni.showToast({ title: '暂无可用置顶券', icon: 'none' })
    return
  }
  await redeemTopVoucher(voucher.id, resource.value.id, ownerMerchantId.value)
  uni.showToast({ title: '已置顶', icon: 'none' })
  closeManagementSheet()
}

async function takeDownOwnResource() {
  const confirmed = await confirmManagementAction({
    title: '下架资源',
    content: '下架后资源将不再公开展示，确认下架吗？',
    confirmText: '下架',
    confirmColor: '#c2410c',
  })
  if (!confirmed) return
  await takeDownResource(resource.value.id, ownerMerchantId.value, '商家主动下架')
  uni.showToast({ title: '已下架', icon: 'none' })
  closeManagementSheet()
  await reloadOwnResource()
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
    title: '删除资源',
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
  if (!(await recordContact('phone'))) return
  const phone = (resource.value.contact || {}).phoneMasked || ''
  if (phone && !phone.includes('*')) {
    uni.makePhoneCall({ phoneNumber: phone })
    return
  }
  uni.showToast({ title: '已记录联系，完整电话由平台保护', icon: 'none' })
}

async function copyWechat() {
  if (!(await recordContact('wechat'))) return
  const wechat = (resource.value.contact || {}).wechatMasked || ''
  if (wechat && !wechat.includes('*')) {
    uni.setClipboardData({ data: wechat })
    return
  }
  uni.showToast({ title: '已记录联系，完整微信由平台保护', icon: 'none' })
}

async function shareResource() {
  if (isOwnResource.value) return
  await recordContact('share')
}

onShareAppMessage(() => ({
  title: resource.value.title || '衣货通资源',
  path: resource.value.id ? `/pages/resource/detail?id=${resource.value.id}` : '/pages/home/index',
}))
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
  display: flex;
  align-items: flex-end;
  padding: 24rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.22), transparent 38%),
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.18) 0 14rpx, transparent 14rpx 28rpx),
    #d88a80;
  color: $wplink-card;
  font-size: 30rpx;
  font-weight: 700;
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

.tag.verified {
  background: $wplink-success-soft;
  color: $wplink-success;
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
  grid-template-columns: repeat(3, 1fr);
  gap: 16rpx;
  box-sizing: border-box;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
  z-index: 20;
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
