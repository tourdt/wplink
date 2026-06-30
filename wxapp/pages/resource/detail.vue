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
        <image v-if="mainImage" class="gallery-main" :src="mainImage" mode="aspectFill" />
        <view v-else class="gallery-main gallery-placeholder">
          <text>{{ resource.category || '资源实拍' }}</text>
        </view>
        <scroll-view v-if="galleryImages.length > 1" class="gallery-strip" scroll-x>
          <image v-for="url in galleryImages" :key="url" class="gallery-thumb" :src="url" mode="aspectFill" />
        </scroll-view>
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
        <text class="section-title">联系提示</text>
        <text class="section-content">平台已记录联系行为，电话和微信可能因隐私保护展示为脱敏信息。联系时建议确认实物、价格和交付方式。</text>
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

      <view class="contact-bar">
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
import { getResourceFavoriteState, setResourceFavorite } from '../../api/favorite'
import { getMerchant } from '../../api/merchant'
import { getOwnResource, getResource, listResources, recordResourceContact, recordResourceDetailView } from '../../api/resource'
import { getSession } from '../../store/session'

const resource = ref({})
const merchantProfile = ref({})
const relatedResources = ref([])
const favorited = ref(false)
const isOwnResource = ref(false)
const ownerMerchantId = ref('')
const resourceUnavailable = ref(false)
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
const mainImage = computed(() => galleryImages.value[0] || '')
const specItems = computed(() => [
  { label: '品类', value: resource.value.category || '待沟通' },
  { label: '数量', value: resource.value.quantityText || '待沟通' },
  { label: '价格', value: resource.value.priceText || '面议' },
  { label: '刷新', value: resource.value.refreshedAt || '近期更新' },
])

onLoad(async (options) => {
  if (!options.id) return
  // 从“我的发布”进入时允许查看待审核、草稿、已下架等非公开状态，避免误提示资源已下架。
  ownerMerchantId.value = options.merchantId || ''
  isOwnResource.value = options.from === 'my-resources' || Boolean(ownerMerchantId.value)
  resourceUnavailable.value = false
  try {
    resource.value = isOwnResource.value ? await getOwnResource(options.id, ownerMerchantId.value, { suppressErrorToast: true }) : await getResource(options.id, { suppressErrorToast: true })
  } catch (err) {
    if (!isOwnResource.value && await loadOwnResourceIfCurrentMerchant(options.id)) {
      return
    }
    resourceUnavailable.value = true
    resource.value = {}
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
    return true
  } catch (err) {
    return false
  }
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
    if (action !== 'merchant_home') {
      uni.showToast({ title: '这是你发布的资源，可在我的发布中管理', icon: 'none' })
    }
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

.gallery-strip {
  width: 100%;
  white-space: nowrap;
}

.gallery-thumb {
  display: inline-block;
  width: 132rpx;
  height: 132rpx;
  margin-right: 12rpx;
  border-radius: 10rpx;
  background: #edf2f7;
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

.contact-bar {
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

.contact-bar button::after {
  border: 0;
}

.contact-bar .primary-button {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.14);
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}
</style>
