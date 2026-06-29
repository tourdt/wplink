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
            <view class="hero-tag-row">
              <text class="hero-tag verified" v-if="merchant.verificationStatus === 'verified'">已认证</text>
              <text class="hero-tag verified" v-if="creditTags.length">平台核实</text>
            </view>
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
        <text class="section-title">商家信息</text>
      </view>
      <view class="profile-chip-row">
        <text v-if="merchantCategoryTags.length === 0" class="profile-chip muted">主营品类待补充</text>
        <text v-for="category in merchantCategoryTags" :key="category" class="profile-chip category">{{ category }}</text>
      </view>
      <text class="profile-description">{{ profileDescription }}</text>
    </view>

    <view class="section" v-if="merchant.addressText || hasMerchantLocation">
      <view class="section-head">
        <text class="section-title">商家地址</text>
        <button v-if="hasMerchantLocation" class="address-action" @click="openMerchantLocation">地图导航</button>
      </view>
      <text class="section-content">{{ merchant.addressText || merchantLocation.address || merchantLocation.name }}</text>
    </view>

    <view class="section media-section" v-if="merchantImages.length">
      <text class="section-title">商家图片</text>
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

    <view class="section trust-note-section">
      <text class="section-title">联系提示</text>
      <text class="section-content">认证商家、运营推荐和置顶权益会影响资源曝光，但不会替代平台审核和买家自行确认。</text>
      <text class="section-tip">从资源详情进入可查看完整联系方式，便于平台记录浏览、电话和微信行为。</text>
    </view>

    <view class="section">
      <view class="section-head">
        <text class="section-title">发布记录</text>
        <text class="section-link" v-if="merchantResourceCountText">{{ merchantResourceCountText }}</text>
      </view>
      <ResourceList
        :resources="merchantResources"
        empty-text="暂无公开资源"
        :loading="merchantResourcesLoading"
        :has-more="hasMoreMerchantResources"
        load-more-text="查看更多发布记录"
        @open="openResource"
        @load-more="loadMerchantResources"
      />
    </view>

    <view class="contact-bar">
      <button class="contact-button" @click="copyWechat">复制微信</button>
      <button class="primary-button" @click="callPhone">拨打电话</button>
    </view>
    <view class="contact-spacer" />
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
  factory: '工厂',
  stall: '档口',
  stockist: '库存商',
  service_provider: '服务商',
  buyer: '采购商',
}
const creditTags = computed(() => merchant.value.creditTags || [])
const merchantLogo = computed(() => merchant.value.logoUrl || '')
const merchantImages = computed(() => merchant.value.images || [])
const merchantLocation = computed(() => merchant.value.location || {})
const hasMerchantLocation = computed(() => hasValidLocation(merchantLocation.value))
const resourcesSummary = computed(() => merchant.value.resourcesSummary || {})
const isOwnMerchant = computed(() => Boolean(merchant.value.id) && merchant.value.id === ownMerchantId.value)
const merchantInitial = computed(() => String(merchant.value.name || '商').slice(0, 1))
const merchantCategoryTags = computed(() => merchant.value.mainCategories || [])
const merchantTypeLabel = computed(() => merchantTypeText[merchant.value.merchantType] || merchant.value.merchantType || '')
const profileDescription = computed(() => merchant.value.description || '暂无简介')
const merchantSubtitle = computed(() => {
  const categories = merchantCategoryTags.value.join('、')
  const identity = [merchantTypeLabel.value, categories].filter(Boolean).join(' · ')
  return identity || merchant.value.description || '服装产业资源商家'
})
const hasMoreMerchantResources = computed(() => merchantResourceTotal.value > merchantResources.value.length)
const merchantResourceCountText = computed(() => {
  const total = merchantResourceTotal.value || resourcesSummary.value.publishedCount || merchantResources.value.length
  if (!total) return ''
  return `${merchantResources.value.length}/${total} 条`
})
const statCards = computed(() => [
  {
    label: '当前资源',
    value: resourcesSummary.value.publishedCount || merchantResourceTotal.value || merchantResources.value.length || 0,
  },
  {
    label: '历史发布',
    value: resourcesSummary.value.totalCount || resourcesSummary.value.publishedCount || merchantResourceTotal.value || merchantResources.value.length || 0,
  },
  {
    label: '商家热度',
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
    uni.showToast({ title: err.message || '发布记录加载失败，请稍后重试', icon: 'none' })
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
    uni.showToast({ title: followed.value ? '已关注商家' : '已取消关注', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '关注失败，请稍后重试', icon: 'none' })
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
  if (!hasMerchantLocation.value) return
  const location = merchantLocation.value
  uni.openLocation({
    latitude: Number(location.latitude),
    longitude: Number(location.longitude),
    name: location.name || merchant.value.name || '商家位置',
    address: location.address || merchant.value.addressText || '',
  })
}

function previewMerchantImage(url) {
  if (!url || merchantImages.value.length === 0) return
  uni.previewImage({
    urls: merchantImages.value,
    current: url,
  })
}

function callPhone() {
  uni.showToast({ title: '请在资源详情页查看完整电话', icon: 'none' })
}

function copyWechat() {
  uni.showToast({ title: '请在资源详情页查看完整微信', icon: 'none' })
}

function hasValidLocation(location) {
  if (!location) return false
  return Number.isFinite(Number(location.latitude)) && Number.isFinite(Number(location.longitude))
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

.hero-tag-row,
.profile-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.hero-tag,
.profile-chip {
  display: inline-flex;
  align-items: center;
  min-height: 40rpx;
  padding: 0 14rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
  line-height: 1.25;
}

.hero-tag {
  background: rgba(255, 255, 255, 0.12);
  color: $wplink-card;
}

.hero-tag.verified {
  background: rgba(22, 163, 106, 0.16);
  color: #b8f3d5;
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

.profile-chip.verified {
  background: $wplink-success-soft;
  color: $wplink-success;
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

.address-action {
  min-width: 132rpx;
  height: 56rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 8rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 56rpx;
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

.contact-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
}

.contact-button,
.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  font-size: 30rpx;
  line-height: 1.25;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.contact-spacer {
  height: calc(124rpx + env(safe-area-inset-bottom));
}
</style>
