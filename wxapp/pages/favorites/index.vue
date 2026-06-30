<template>
  <view class="favorites-page">
    <view class="filter-row">
      <button
        v-for="item in tabs"
        :key="item.value"
        :class="['filter-button', activeTab === item.value ? 'active' : '']"
        @click="selectTab(item.value)"
      >
        {{ item.label }}
      </button>
    </view>

    <view v-if="activeTab === 'resources' && favoriteResources.length" class="content-list">
      <ResourceCard v-for="item in favoriteResources" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-if="activeTab === 'merchants' && followedMerchants.length" class="content-list">
      <view v-for="item in followedMerchants" :key="item.id" class="merchant-item" @click="openMerchant(item)">
        <image v-if="merchantAvatarUrl(item)" class="merchant-avatar" :src="merchantAvatarUrl(item)" mode="aspectFill" />
        <view v-else class="merchant-avatar merchant-avatar-placeholder">
          <text>{{ merchantAvatarText(item) }}</text>
        </view>
        <view class="merchant-info">
          <MerchantBadge :merchant="item" />
          <text class="merchant-hint">{{ merchantBusinessText(item) }}</text>
        </view>
        <text class="merchant-arrow">›</text>
      </view>
    </view>

    <view v-if="!loading && currentRows.length === 0" class="empty-state">
      <view class="empty-visual">
        <text>{{ activeTab === 'resources' ? '藏' : '关' }}</text>
      </view>
      <text class="empty-title">{{ emptyTitle }}</text>
      <text class="empty-desc">{{ emptyDesc }}</text>
      <button class="empty-action" @click="openEmptyAction">{{ emptyActionText }}</button>
    </view>

    <text v-if="currentRows.length" class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import MerchantBadge from '../../components/MerchantBadge.vue'
import ResourceCard from '../../components/ResourceCard.vue'
import { listFavoriteResources, listFollowedMerchants } from '../../api/favorite'

const tabs = [
  { label: '资源', value: 'resources' },
  { label: '商家', value: 'merchants' },
]
const activeTab = ref('resources')
const favoriteResources = ref([])
const followedMerchants = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)
const merchantTypeText = {
  factory: '工厂',
  stall: '档口',
  stockist: '库存商',
  service_provider: '服务商',
  buyer: '采购商',
}
const currentRows = computed(() => (activeTab.value === 'resources' ? favoriteResources.value : followedMerchants.value))
const emptyTitle = computed(() => (activeTab.value === 'resources' ? '暂无收藏资源' : '暂无关注商家'))
const emptyDesc = computed(() => {
  if (activeTab.value === 'resources') return '看到合适的库存、货源或产能后点收藏，后续可在这里快速回看。'
  return '关注常合作或感兴趣的商家，后续可从这里快速进入商家主页。'
})
const emptyActionText = computed(() => (activeTab.value === 'resources' ? '去找资源' : '去找商家'))

onLoad(() => {
  loadRows({ reset: true })
})

onPullDownRefresh(async () => {
  try {
    await loadRows({ reset: true })
  } finally {
    uni.stopPullDownRefresh()
  }
})

onReachBottom(() => {
  loadRows({ reset: false })
})

function selectTab(value) {
  if (activeTab.value === value) return
  activeTab.value = value
  loadRows({ reset: true })
}

async function loadRows({ reset = true } = {}) {
  if (loading.value) return
  if (!reset && !hasMore.value) return
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = activeTab.value === 'resources'
      ? await listFavoriteResources({ page: nextPage, pageSize })
      : await listFollowedMerchants({ page: nextPage, pageSize })
    const items = resp.items || []
    const nextRows = reset ? items : [...currentRows.value, ...items]
    if (activeTab.value === 'resources') {
      favoriteResources.value = nextRows
    } else {
      followedMerchants.value = nextRows
    }
    page.value = nextPage
    total.value = resp.total || nextRows.length
    hasMore.value = nextRows.length < total.value
  } catch (err) {
    if (reset) {
      if (activeTab.value === 'resources') {
        favoriteResources.value = []
      } else {
        followedMerchants.value = []
      }
      total.value = 0
      hasMore.value = false
    }
  } finally {
    loading.value = false
  }
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openMerchant(item) {
  uni.navigateTo({ url: `/pages/merchant/detail?id=${item.id}` })
}

function openEmptyAction() {
  uni.switchTab({ url: '/pages/search/index' })
}

function merchantAvatarUrl(item) {
  return item.logoUrl || item.avatarUrl || ''
}

function merchantAvatarText(item) {
  const name = item.name || '商家'
  return name.slice(0, 1)
}

function merchantBusinessText(item) {
  const mainCategories = item.mainCategories || []
  if (mainCategories.length > 0) return mainCategories.join('、')
  return merchantTypeText[item.merchantType] || item.merchantType || '主营品类待补充'
}
</script>

<style lang="scss" scoped>
.favorites-page {
  min-height: 100vh;
  padding: 24rpx;
  padding-top: 132rpx;
  background: $wplink-bg;
}

.filter-row {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 10;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
  padding: 24rpx 24rpx 16rpx;
  overflow: hidden;
  background: $wplink-card;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);
}

.filter-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-width: 0;
  height: 72rpx;
  padding: 0 8rpx;
  border: 2rpx solid transparent;
  border-radius: 10rpx;
  background: #f4f7fd;
  color: #364152;
  font-size: 25rpx;
  line-height: 1.2;
  white-space: nowrap;
  transition: background 0.18s ease, color 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.filter-button.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
  box-shadow: 0 8rpx 18rpx rgba(194, 58, 0, 0.18);
}

.content-list {
  display: grid;
  gap: 16rpx;
}

.merchant-item {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr) 28rpx;
  align-items: center;
  gap: 18rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
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

.merchant-hint {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.55;
  word-break: break-word;
}

.merchant-arrow {
  color: $wplink-muted;
  font-size: 44rpx;
  line-height: 1;
  text-align: right;
}

.empty-state {
  display: grid;
  align-content: center;
  justify-items: center;
  gap: 16rpx;
  min-height: 420rpx;
  padding: 48rpx 28rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.empty-visual {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  border-radius: 999rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
}

.empty-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
}

.empty-desc {
  max-width: 540rpx;
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.empty-action {
  min-width: 180rpx;
  height: 72rpx;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
}

.load-more-text {
  display: block;
  padding: 24rpx 0 18rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
  text-align: center;
}

</style>
