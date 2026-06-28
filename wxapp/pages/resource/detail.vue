<template>
  <view class="resource-page">
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
      <MerchantBadge :merchant="merchantInfo" />
      <text class="merchant-hint">查看商家认证、发布记录和信用信息</text>
    </view>

    <view v-if="relatedResources.length" class="related-section">
      <view class="section-head">
        <text class="section-title">同类推荐</text>
        <text class="section-link" @click="openSearch">查看更多</text>
      </view>
      <ResourceCard v-for="item in relatedResources" :key="item.id" :resource="item" @open="openRelatedResource" />
    </view>

    <view class="contact-bar">
      <button @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">联系商家</button>
      <button open-type="share" @click="shareResource">分享</button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShareAppMessage } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import ResourceCard from '../../components/ResourceCard.vue'
import { getResourceFavoriteState, setResourceFavorite } from '../../api/favorite'
import { getResource, listResources, recordResourceContact, recordResourceDetailView } from '../../api/resource'
import { getSession } from '../../store/session'

const resource = ref({})
const relatedResources = ref([])
const favorited = ref(false)
const SEARCH_KEY = 'wplink_pending_search_keyword'
const statusText = {
  published: '已发布',
  pending: '待审核',
  expired: '已过期',
  dealt: '已成交',
}
const isVerifiedMerchant = computed(() => (resource.value.merchant || {}).verificationStatus === 'verified')
const merchantInfo = computed(() => resource.value.merchant || {})
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
  resource.value = await getResource(options.id)
  await recordResourceDetailView(options.id)
  await loadFavoriteState(options.id)
  await loadRelatedResources()
})

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
  if (!resource.value.id) return
  await recordResourceContact(resource.value.id, action)
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
  uni.switchTab({ url: '/pages/search/index' })
}

async function callPhone() {
  await recordContact('phone')
  const phone = (resource.value.contact || {}).phoneMasked || ''
  if (phone && !phone.includes('*')) {
    uni.makePhoneCall({ phoneNumber: phone })
    return
  }
  uni.showToast({ title: '已记录联系，完整电话由平台保护', icon: 'none' })
}

async function copyWechat() {
  await recordContact('wechat')
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
  title: resource.value.title || '服链通资源',
  path: resource.value.id ? `/pages/resource/detail?id=${resource.value.id}` : '/pages/home/index',
}))
</script>

<style scoped>
.resource-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 150rpx;
  background: #f4f6f8;
}

.resource-card,
.merchant-card,
.trust-card,
.related-section {
  display: grid;
  gap: 12rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
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
  color: #ffffff;
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
  background: #e6f4f1;
  color: #0f766e;
}

.title-row {
  display: grid;
  grid-template-columns: 1fr 136rpx;
  gap: 16rpx;
  align-items: start;
}

.title {
  color: #1f2933;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.35;
  min-width: 0;
  word-break: break-word;
}

.favorite-button {
  height: 64rpx;
  border-radius: 10rpx;
  background: #fff7e6;
  color: #b7791f;
  font-size: 24rpx;
}

.meta,
.desc,
.merchant-status,
.section-content,
.merchant-hint,
.section-link {
  color: #697586;
  font-size: 28rpx;
  line-height: 1.55;
}

.price {
  color: #c2410c;
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
  color: #697586;
  font-size: 24rpx;
}

.spec-value {
  color: #1f2933;
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
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.section-link {
  color: #0f766e;
}

.contact-bar {
  position: fixed;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16rpx;
}

.contact-bar button {
  height: 88rpx;
  border-radius: 12rpx;
  background: #ffffff;
  color: #364152;
  font-size: 28rpx;
  line-height: 1.25;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
