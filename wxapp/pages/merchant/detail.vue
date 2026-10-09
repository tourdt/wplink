<template>
  <view class="merchant-page">
    <view v-if="loadState === 'loading'" class="feedback"><text>正在读取主页…</text></view>
    <view v-else-if="loadState === 'error'" class="feedback">
      <text class="feedback-title">主页暂时无法打开</text>
      <text>请检查网络后重试。</text>
      <button class="primary-button" @click="loadMerchant">重新加载</button>
    </view>
    <view v-else-if="loadState === 'unavailable'" class="feedback">
      <text class="feedback-title">主页暂不可查看</text>
      <text>经营者可能暂停了主页，或业务资料已调整。</text>
      <button class="outline-button" @click="openDirectory">继续找档口</button>
    </view>
    <view v-else-if="loadState === 'ready'" class="storefront">
      <view v-if="isOwner && merchant.ownerVisible === false" class="preview-banner">仅管理者可见 · 展示已暂停，分享链接无法供他人查看</view>
      <view class="hero">
        <view class="hero-photo-wrap" :class="{ empty: !heroPhoto }">
          <image v-if="heroPhoto" class="hero-photo" :src="heroPhoto" mode="aspectFill" @click="previewImage(heroPhoto)" />
          <view v-else class="hero-photo-empty">{{ merchantInitial }}</view>
        </view>
        <view class="hero-body">
          <text class="eyebrow">{{ businessRoleText || '业务主页' }}</text>
          <text class="merchant-name">{{ merchant.name }}</text>
          <text v-if="place.position" class="merchant-position">{{ place.position }}</text>
          <view class="hero-actions">
            <button v-if="isOwner" class="inline-action" @click="openEditor">编辑资料</button>
            <button v-if="canShare" class="inline-action" open-type="share">分享主页</button>
          </view>
        </view>
      </view>
      <view class="category-block">
        <text class="section-label">经营业务</text>
        <text class="category-text">{{ categoryText }}</text>
      </view>
      <view v-if="merchantImages.length > 1" class="section">
        <text class="section-label">业务实拍</text>
        <scroll-view class="gallery" scroll-x>
          <image v-for="url in merchantImages" :key="url" class="gallery-image" :src="url" mode="aspectFill" @click="previewImage(url)" />
        </scroll-view>
      </view>
      <view class="section">
        <text class="section-label">业务介绍</text>
        <text class="body-copy">{{ merchant.description || '暂未填写业务介绍。' }}</text>
      </view>
      <view v-if="businessProfile.serviceArea" class="section"><text class="section-label">服务区域</text><text class="body-copy">{{ businessProfile.serviceArea }}</text></view>
      <view v-if="place.address" class="section address-section">
        <view class="section-row"><text class="section-label">接待地址</text><text v-if="!place.canNavigate" class="location-note">{{ locationNote }}</text></view>
        <text class="body-copy address-copy">{{ place.address }}</text>
        <map v-if="place.canNavigate" class="store-map" :latitude="place.latitude" :longitude="place.longitude" :markers="mapMarkers" :show-location="false" />
        <button v-if="place.canNavigate" class="map-action" @click="openNavigation">导航前往 →</button>
      </view>
      <view class="section last-section">
        <text class="section-label">联系提醒</text>
        <text class="body-copy">请先联系确认业务内容；如需到访，请确认接待时间和地点。</text>
      </view>
      <view v-if="hasPublicContact || place.canNavigate" class="bottom-bar">
        <button v-if="hasPublicContact" class="bar-secondary" @click="openContact">联系经营者</button>
        <button v-if="place.canNavigate" class="bar-primary" @click="openNavigation">导航前往</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onShareAppMessage, onShareTimeline } from '@dcloudio/uni-app'
import { getMerchant } from '../../api/merchant'
import { getPublicContact, getStorefrontPlace } from '../../common/merchantStorefront'
import { getBusinessProfile, getBusinessRoleLabels, getBusinessProfileSummary } from '../../common/merchantBusinessProfile'
import { getSession } from '../../store/session'

const currentMerchantId = ref('')
const merchant = ref({})
const loadState = ref('loading')
const ownMerchantId = ref('')
const requestId = ref(0)
const isOwner = computed(() => Boolean(merchant.value.id && ownMerchantId.value && merchant.value.id === ownMerchantId.value))
const canShare = computed(() => loadState.value === 'ready' && merchant.value.ownerVisible !== false)
const merchantInitial = computed(() => String(merchant.value.name || '店').slice(0, 1))
const merchantImages = computed(() => (Array.isArray(merchant.value.images) ? merchant.value.images : []).filter(Boolean))
const heroPhoto = computed(() => merchantImages.value[0] || merchant.value.logoUrl || '')
const businessProfile = computed(() => getBusinessProfile(merchant.value))
const businessRoleText = computed(() => getBusinessRoleLabels(businessProfile.value).join(' · '))
const categoryText = computed(() => getBusinessProfileSummary(merchant.value) || '业务资料待补充')
const place = computed(() => getStorefrontPlace(merchant.value))
const locationNote = computed(() => merchant.value.location && merchant.value.location.status === 'risk_hidden' ? '位置待处理' : '暂不可导航')
const publicContact = computed(() => getPublicContact(merchant.value, isOwner.value))
const hasPublicContact = computed(() => Boolean(publicContact.value.phone || publicContact.value.wechat))
const mapMarkers = computed(() => place.value.canNavigate ? [{ id: 1, latitude: place.value.latitude, longitude: place.value.longitude, iconPath: '/static/map/marker-selected.png', width: 24, height: 30 }] : [])

onLoad((options) => {
  if (options?.from === 'merchant-paused') {
    openDirectory()
    return
  }
  currentMerchantId.value = String(options?.id || options?.merchantId || '').trim()
  ownMerchantId.value = getSession().merchantId
  return loadMerchant()
})
onShow(() => {
  if (!currentMerchantId.value) return
  ownMerchantId.value = getSession().merchantId
  return loadMerchant()
})

async function loadMerchant() {
  const id = ++requestId.value
  if (!currentMerchantId.value) {
    loadState.value = 'unavailable'
    return
  }
  loadState.value = 'loading'
  try {
    const detail = await getMerchant(currentMerchantId.value, { suppressErrorToast: true })
    if (id !== requestId.value) return
    if (!detail?.id) {
      loadState.value = 'unavailable'
      return
    }
    merchant.value = detail
    loadState.value = 'ready'
  } catch (error) {
    if (id !== requestId.value) return
    // 已暂停或不存在的店铺在非管理者侧可能返回不可用；其他网络错误保留重试入口。
    const status = Number(error?.statusCode || error?.status)
    loadState.value = error?.code === 'MERCHANT_NOT_FOUND' || [403, 404, 410].includes(status) ? 'unavailable' : 'error'
  }
}

function openDirectory() {
  uni.switchTab({ url: '/pages/sourcing-map/index' })
}

function openEditor() {
  if (!isOwner.value) return
  uni.navigateTo({ url: `/pages/merchant/profile?merchantId=${encodeURIComponent(merchant.value.id)}` })
}

function openNavigation() {
  if (!place.value.canNavigate) return
  uni.openLocation({
    latitude: place.value.latitude,
    longitude: place.value.longitude,
    name: merchant.value.name || '接待位置',
    address: place.value.address || place.value.position,
    fail: () => {
      // 原生地图可能因系统授权或服务故障打不开；保留可复制的准确地址和重试入口。
      uni.showActionSheet({
        itemList: ['复制地址', '重试导航'],
        success: ({ tapIndex }) => {
          if (tapIndex === 0) copyAddress()
          if (tapIndex === 1) openNavigation()
        },
      })
    },
  })
}

function copyAddress() {
  const address = place.value.address || place.value.position
  if (!address) return
  uni.setClipboardData({
    data: address,
    success: () => uni.showToast({ title: '地址已复制', icon: 'none' }),
  })
}

function openContact() {
  const choices = []
  if (publicContact.value.phone) choices.push({ label: '拨打电话', action: callPhone })
  if (publicContact.value.wechat) choices.push({ label: '复制微信', action: copyWechat })
  if (!choices.length) return
  uni.showActionSheet({
    itemList: choices.map((choice) => choice.label),
    success: ({ tapIndex }) => choices[tapIndex]?.action(),
  })
}

function previewImage(url) {
  if (!url || !merchantImages.value.length) return
  uni.previewImage({ urls: merchantImages.value, current: url })
}

function callPhone() {
  if (!publicContact.value.phone) return
  uni.makePhoneCall({ phoneNumber: publicContact.value.phone })
}

function copyWechat() {
  if (!publicContact.value.wechat) return
  uni.setClipboardData({
    data: publicContact.value.wechat,
    success: () => uni.showToast({ title: '微信号已复制', icon: 'none' }),
  })
}

function sharePayload() {
  if (!canShare.value) return { title: '找档口', path: '/pages/sourcing-map/index' }
  return {
    title: merchant.value.name || '业务主页',
    path: `/pages/merchant/detail?id=${encodeURIComponent(merchant.value.id)}`,
    imageUrl: heroPhoto.value || undefined,
  }
}
onShareAppMessage(sharePayload)
onShareTimeline(() => ({ title: sharePayload().title, query: canShare.value ? `id=${encodeURIComponent(merchant.value.id)}` : 'from=merchant-paused' }))
</script>

<style lang="scss" scoped>
.merchant-page { min-height: 100vh; background: #fff; color: #202020; }
.storefront { padding-bottom: calc(140rpx + env(safe-area-inset-bottom)); }
.preview-banner { padding: 20rpx 30rpx; background: #fff1e9; color: #a63708; font-size: 25rpx; line-height: 1.45; }
.hero-photo-wrap { width: 100%; height: 480rpx; background: #f0ebe8; }
.hero-photo-wrap.empty { height: 180rpx; }
.hero-photo, .hero-photo-empty { width: 100%; height: 100%; }
.hero-photo-empty { display: flex; align-items: center; justify-content: center; color: #c23a00; font-size: 120rpx; font-weight: 800; }
.hero-body { display: grid; gap: 14rpx; padding: 30rpx 30rpx 34rpx; border-bottom: 1rpx solid #e9e6e3; }
.eyebrow, .section-label { color: #a53306; font-size: 23rpx; font-weight: 700; letter-spacing: 2rpx; }
.merchant-name { color: #202020; font-size: 50rpx; font-weight: 800; line-height: 1.2; }
.merchant-position { color: #202020; font-size: 30rpx; font-weight: 700; line-height: 1.45; }
.hero-actions { display: flex; flex-wrap: wrap; gap: 20rpx; margin-top: 12rpx; }
.inline-action { margin: 0; padding: 0; background: transparent; color: #a53306; font-size: 25rpx; font-weight: 700; line-height: 1.5; }
.inline-action::after { border: 0; }
.category-block { display: grid; gap: 18rpx; margin: 28rpx 28rpx 0; padding: 30rpx; border-left: 8rpx solid #c23a00; background: #fff5ef; }
.category-text { color: #a53306; font-size: 42rpx; font-weight: 800; line-height: 1.35; }
.section { display: grid; gap: 22rpx; margin: 0 28rpx; padding: 40rpx 0; border-bottom: 1rpx solid #ece9e7; }
.section-row { display: flex; justify-content: space-between; gap: 16rpx; }
.body-copy { color: #303030; font-size: 29rpx; line-height: 1.7; word-break: break-word; }
.location-note { color: #a53306; font-size: 23rpx; }
.store-map { width: 100%; height: 320rpx; border-radius: 14rpx; }
.map-action { justify-self: start; margin: 0; padding: 0; background: transparent; color: #c23a00; font-size: 27rpx; font-weight: 700; }
.map-action::after, .bottom-bar button::after { border: 0; }
.gallery { width: 100%; white-space: nowrap; }
.gallery-image { display: inline-block; width: 280rpx; height: 220rpx; margin-right: 14rpx; border-radius: 12rpx; }
.last-section { border-bottom: 0; }
.bottom-bar { position: fixed; right: 0; bottom: 0; left: 0; display: flex; gap: 12rpx; padding: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom)); border-top: 1rpx solid #e9e6e3; background: #fff; }
.bottom-bar button { flex: 1; display: flex; align-items: center; justify-content: center; min-height: 78rpx; margin: 0; padding: 0 12rpx; border-radius: 10rpx; font-size: 25rpx; font-weight: 700; }
.bottom-bar button:only-child { flex: 0 0 100%; }
.bottom-bar button[disabled] { background: #e9e7e5; color: #888; }
.bar-secondary { background: #fff1e9; color: #a53306; }
.bar-primary, .primary-button { background: #c23a00; color: #fff; }
.feedback { display: grid; gap: 20rpx; justify-items: start; padding: 120rpx 34rpx; color: #666; font-size: 28rpx; }
.feedback-title { color: #202020; font-size: 40rpx; font-weight: 800; }
.feedback button { margin: 16rpx 0 0; padding: 0 30rpx; border-radius: 10rpx; font-size: 27rpx; }
.outline-button { border: 1rpx solid #c23a00; background: #fff; color: #c23a00; }
</style>
