<template>
  <view class="merchant-page">
    <view class="merchant-hero-card">
      <view class="merchant-identity-row">
        <view class="merchant-main">
          <image v-if="merchantLogo" class="merchant-logo" :src="merchantLogo" mode="aspectFill" />
          <view v-else class="merchant-logo logo-placeholder">{{ merchantInitial }}</view>
          <view class="merchant-copy">
            <text class="merchant-name">{{ merchant.name || '商家' }}</text>
            <text class="merchant-summary">{{ merchantSubtitle }}</text>
          </view>
        </view>
        <button v-if="isOwnMerchant" class="follow-button" @click="openMerchantEditor">编辑</button>
        <button v-else class="follow-button" @click="toggleFollow">{{ followed ? '已关注' : '关注' }}</button>
      </view>

      <view class="hero-stats">
        <view v-for="item in statCards" :key="item.label" class="hero-stat-item">
          <text class="stat-value">{{ item.value }}</text>
          <text class="stat-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="profile-panel">
      <view class="section-head">
        <text class="section-title">商家档案</text>
      </view>
      <view class="profile-chip-row">
        <text v-if="merchantCategoryTags.length === 0" class="profile-chip muted">主营待补充</text>
        <text v-for="category in merchantCategoryTags" :key="category" class="profile-chip category">{{ category }}</text>
      </view>
      <text class="profile-description">{{ profileDescription }}</text>
    </view>

    <view class="section media-section" v-if="merchantImages.length">
      <text class="section-title">实拍图片</text>
      <scroll-view class="merchant-gallery" scroll-x>
        <image
          v-for="url in merchantImages"
          :key="url"
          class="merchant-image"
          :src="url"
          mode="aspectFill"
          @click="previewMerchantImage(url)"
        />
      </scroll-view>
    </view>

    <view class="section merchant-address-section" v-if="merchantAddressLocation">
      <view class="section-head">
        <text class="section-title">地址</text>
        <button v-if="merchantAddressLocation.hasGps" class="address-action" @click="openMerchantLocation">导航</button>
        <button v-else class="address-action secondary" @click="copyMerchantAddress()">复制</button>
      </view>
      <text class="merchant-address-text">{{ merchantAddressLocation.address }}</text>
      <map
        v-if="merchantAddressLocation.hasGps"
        class="merchant-address-map"
        :latitude="merchantAddressLocation.latitude"
        :longitude="merchantAddressLocation.longitude"
        :markers="merchantAddressLocation.markers"
        :scale="17"
        @tap="openMerchantLocation"
      />
    </view>

    <view class="section trust-note-section">
      <text class="section-title">温馨提醒</text>
      <text class="section-content">联系前请确认实物、价格和交期。</text>
      <text class="section-tip">电话和微信见供应详情。</text>
    </view>

    <view class="section resource-list-section">
      <view class="section-head">
        <text class="section-title">公开供应</text>
        <text class="section-link" v-if="merchantResourceCountText">{{ merchantResourceCountText }}</text>
      </view>
      <ResourceList
        exposure-source="merchant"
        :resources="merchantResources"
        :empty-text="merchantResourcesEmptyText"
        :loading="merchantResourcesLoading"
        :has-more="hasMoreMerchantResources"
        load-more-text="查看更多供应"
        @open="openResource"
        @load-more="loadMerchantResources"
      />
    </view>

  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import ResourceList from '../../components/ResourceList.vue'
import { getMerchantFollowState, setMerchantFollow } from '../../api/favorite'
import { getMerchant } from '../../api/merchant'
import { listResources } from '../../api/resource'
import { getSession } from '../../store/session'

const merchant = ref({})
const currentMerchantId = ref('')
const merchantResources = ref([])
const merchantResourcePage = ref(0)
const merchantResourcePageSize = 10
const merchantResourceTotal = ref(0)
const merchantResourcesLoading = ref(false)
const followed = ref(false)
const ownMerchantId = ref('')
const merchantTypeText = {
  individual: '个人',
  rental_provider: '场地/设备方',
  factory: '源头工厂',
  stall: '现货档口',
  stockist: '库存货源',
  service_provider: '配套服务',
  buyer: '采购',
}
const merchantLogo = computed(() => merchant.value.logoUrl || '')
const merchantImages = computed(() => merchant.value.images || [])
const merchantLocation = computed(() => merchant.value.location || {})
const hasMerchantLocation = computed(() => hasValidLocation(merchantLocation.value))
const merchantAddressLocation = computed(() => buildMerchantAddressLocation())
const resourcesSummary = computed(() => merchant.value.resourcesSummary || {})
const isOwnMerchant = computed(() => Boolean(merchant.value.id) && merchant.value.id === ownMerchantId.value)
const merchantInitial = computed(() => String(merchant.value.name || '商').slice(0, 1))
const merchantCategoryTags = computed(() => merchant.value.mainCategories || [])
const merchantTypeLabel = computed(() => merchantTypeText[merchant.value.merchantType] || merchant.value.merchantType || '')
const profileDescription = computed(() => merchant.value.description || '暂无介绍')
const merchantSubtitle = computed(() => {
  const categories = merchantCategoryTags.value.join('、')
  const identity = [merchantTypeLabel.value, categories].filter(Boolean).join(' · ')
  return identity || merchant.value.description || '服装供应商家'
})
const merchantResourceTotalCount = computed(() => (
  merchantResourcePage.value > 0
    ? merchantResourceTotal.value
    : resourcesSummary.value.publishedCount || merchantResources.value.length || 0
))
const merchantResourcesEmptyText = computed(() => (
  isOwnMerchant.value ? '暂无公开供应，发布后会展示在这里' : '暂无公开供应，可先关注商家'
))
const hasMoreMerchantResources = computed(() => merchantResourceTotal.value > merchantResources.value.length)
const merchantResourceCountText = computed(() => {
  const total = merchantResourceTotalCount.value
  if (!total) return ''
  return `${merchantResources.value.length}/${total} 条`
})
const statCards = computed(() => [
  {
    label: '公开供应',
    value: resourcesSummary.value.publishedCount || merchantResourceTotal.value || merchantResources.value.length || 0,
  },
  {
    label: '历史发布',
    value: resourcesSummary.value.totalCount || resourcesSummary.value.publishedCount || merchantResourceTotal.value || merchantResources.value.length || 0,
  },
  {
    label: '热度',
    value: merchant.value.heatScore || 0,
  },
])

onLoad(async (options) => {
  if (!options.id) return
  currentMerchantId.value = options.id
  ownMerchantId.value = getSession().merchantId
  resetMerchantResources()
  merchant.value = await getMerchant(options.id)
  await loadFollowState(options.id)
  await loadMerchantResources()
})

onReachBottom(() => {
  if (hasMoreMerchantResources.value) {
    loadMerchantResources()
  }
})

async function loadFollowState(merchantId) {
  if (!getSession().token || merchantId === ownMerchantId.value) return
  try {
    const resp = await getMerchantFollowState(merchantId)
    followed.value = Boolean(resp.followed)
  } catch (err) {
    followed.value = false
  }
}

async function loadMerchantResources() {
  const merchantId = merchant.value.id || currentMerchantId.value
  if (!merchantId || merchantResourcesLoading.value) return
  const nextPage = merchantResourcePage.value + 1
  merchantResourcesLoading.value = true
  try {
    const resp = await listResources({
      merchantId,
      page: nextPage,
      pageSize: merchantResourcePageSize,
    })
    const items = resp.items || []
    merchantResourcePage.value = resp.page || nextPage
    merchantResourceTotal.value = resp.total || merchantResources.value.length + items.length
    merchantResources.value = nextPage === 1 ? items : [...merchantResources.value, ...items]
  } catch (err) {
    uni.showToast({ title: err.message || '供应加载失败，请稍后重试', icon: 'none' })
  } finally {
    merchantResourcesLoading.value = false
  }
}

function resetMerchantResources() {
  merchantResources.value = []
  merchantResourcePage.value = 0
  merchantResourceTotal.value = 0
}

async function toggleFollow() {
  if (!merchant.value.id) return
  try {
    // 关注商家用于后续复访和提醒，当前只改变关注列表，不触发营销消息。
    const resp = await setMerchantFollow(merchant.value.id, !followed.value)
    followed.value = Boolean(resp.followed)
    uni.showToast({ title: followed.value ? '已关注' : '已取消关注', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '操作失败，请稍后重试', icon: 'none' })
  }
}

function openMerchantEditor() {
  if (!merchant.value.id) return
  uni.navigateTo({ url: `/pages/merchant/profile?merchantId=${merchant.value.id}` })
}

function openResource(resource) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${resource.id}` })
}

function openMerchantLocation() {
  const location = merchantAddressLocation.value
  if (!location?.hasGps) {
    copyMerchantAddress()
    return
  }
  uni.openLocation({
    latitude: location.latitude,
    longitude: location.longitude,
    name: location.name || merchant.value.name || '商家位置',
    address: location.address,
    scale: 18,
    fail() {
      if (location.address) {
        copyMerchantAddress('导航打开失败，已复制地址')
      }
    },
  })
}

function copyMerchantAddress(title = '地址已复制') {
  const location = merchantAddressLocation.value
  if (!location?.address) return
  uni.setClipboardData({ data: location.address })
  uni.showToast({ title, icon: 'none' })
}

function previewMerchantImage(url) {
  if (!url || merchantImages.value.length === 0) return
  uni.previewImage({
    urls: merchantImages.value,
    current: url,
  })
}

function buildMerchantAddressLocation() {
  const location = merchantLocation.value || {}
  const latitude = Number(location.latitude ?? location.lat)
  const longitude = Number(location.longitude ?? location.lng)
  const hasGps = Number.isFinite(latitude) && Number.isFinite(longitude)
  const address = String(merchant.value.addressText || location.address || location.name || (hasGps ? '商家位置' : '')).trim()
  if (!address) return null
  const name = String(location.name || merchant.value.name || '商家位置').trim()
  const item = {
    address,
    name,
    hasGps,
  }
  if (!hasGps) return item
  return {
    ...item,
    latitude,
    longitude,
    markers: [{
      id: 1,
      latitude,
      longitude,
      title: address,
    }],
  }
}

function hasValidLocation(location) {
  if (!location) return false
  const latitude = Number(location.latitude ?? location.lat)
  const longitude = Number(location.longitude ?? location.lng)
  return Number.isFinite(latitude) && Number.isFinite(longitude)
}
</script>

<style lang="scss" scoped>
.merchant-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.merchant-hero-card,
.profile-panel,
.section {
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.merchant-hero-card {
  display: grid;
  gap: 24rpx;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.08), transparent 46%),
    $wplink-primary;
  box-shadow: 0 16rpx 48rpx rgba(6, 22, 37, 0.12);
}

.merchant-identity-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 132rpx;
  gap: 16rpx;
  align-items: start;
}

.merchant-main {
  display: grid;
  grid-template-columns: 104rpx minmax(0, 1fr);
  gap: 18rpx;
  align-items: start;
  min-width: 0;
}

.merchant-copy {
  display: grid;
  gap: 10rpx;
  min-width: 0;
}

.merchant-logo {
  width: 104rpx;
  height: 104rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.24);
  border-radius: 12rpx;
  background: $wplink-card;
}

.logo-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
}

.merchant-name {
  display: block;
  color: $wplink-card;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.25;
  word-break: break-word;
}

.merchant-summary {
  display: block;
  color: rgba(255, 255, 255, 0.76);
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}

.profile-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.profile-chip {
  display: inline-flex;
  align-items: center;
  min-height: 40rpx;
  padding: 0 14rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  line-height: 1.25;
}

.hero-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
}

.hero-stat-item {
  display: grid;
  gap: 6rpx;
  padding: 18rpx 10rpx;
  border-radius: 10rpx;
  background: rgba(255, 255, 255, 0.1);
  text-align: center;
}

.stat-value {
  color: $wplink-card;
  font-size: 34rpx;
  font-weight: 700;
}

.stat-label {
  color: rgba(255, 255, 255, 0.7);
  font-size: 24rpx;
}

.follow-button {
  height: 64rpx;
  border-radius: 10rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 700;
}

.section-title {
  display: block;
  margin-bottom: 12rpx;
  color: $wplink-muted;
  font-size: 26rpx;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12rpx;
}

.section-head .section-title {
  margin-bottom: 0;
}

.profile-panel {
  display: grid;
  gap: 16rpx;
}

.profile-panel .section-head {
  margin-bottom: 0;
}

.profile-chip {
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-weight: 700;
}

.profile-chip.category {
  min-height: 48rpx;
  padding: 0 18rpx;
  border: 1rpx solid rgba($wplink-warning, 0.18);
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 700;
}

.profile-chip.muted {
  background: #f8fafc;
  color: $wplink-muted;
  font-weight: 600;
}

.profile-description {
  color: $wplink-primary;
  font-size: 30rpx;
  line-height: 1.6;
  word-break: break-word;
}

.merchant-address-section {
  display: grid;
  gap: 14rpx;
}

.merchant-address-section .section-head {
  margin-bottom: 0;
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

.address-action.secondary {
  background: #eef2f7;
  color: #364152;
}

.address-action::after {
  border: 0;
}

.merchant-address-text {
  display: block;
  color: $wplink-primary;
  font-size: 30rpx;
  line-height: 1.55;
  word-break: break-word;
}

.merchant-address-map {
  width: 100%;
  height: 260rpx;
  border-radius: 10rpx;
  overflow: hidden;
  background: #e2e8f0;
}

.section-content {
  color: $wplink-primary;
  font-size: 30rpx;
  line-height: 1.6;
  word-break: break-word;
}

.section-link {
  color: $wplink-muted;
  font-size: 26rpx;
}

.trust-note-section {
  background: $wplink-warning-soft;
}

.section-tip {
  display: block;
  margin-top: 10rpx;
  color: #7c5a22;
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}

.media-section {
  overflow: hidden;
}

.merchant-gallery {
  width: 100%;
  white-space: nowrap;
}

.merchant-image {
  display: inline-block;
  width: 280rpx;
  height: 180rpx;
  margin-right: 12rpx;
  border-radius: 10rpx;
  background: #e3e8ef;
}

</style>
