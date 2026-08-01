<template>
  <view class="sourcing-map-page">
    <view class="directory-toolbar">
      <view class="search-row">
        <view class="search-field">
          <view class="search-icon" />
          <input
            v-model="keyword"
            placeholder="搜索商家、档口号或市场"
            confirm-type="search"
            @confirm="submitSearch"
          />
        </view>
        <button class="search-button" :disabled="loading" @click="submitSearch">搜索</button>
      </view>

      <view class="directory-summary">
        <text class="result-count">{{ total }} 个档口</text>
      </view>

      <scroll-view class="filter-scroll" scroll-x>
        <button
          v-for="filter in sourceFilters"
          :key="filter.value"
          :class="['filter-chip', { active: claimedFilter === filter.value }]"
          @click="selectSourceFilter(filter.value)"
        >
          {{ filter.label }}
        </button>
        <button
          v-for="category in categories"
          :key="category.code"
          :class="['filter-chip', { active: categoryCodes.includes(category.code) }]"
          @click="toggleCategory(category.code)"
        >
          {{ category.name }}
        </button>
      </scroll-view>
    </view>

    <view class="merchant-list">
      <view v-if="loading && !places.length" class="state-card">
        <text class="state-title">正在查找商家</text>
        <text class="state-desc">档口资料正在加载，请稍候。</text>
      </view>
      <view v-else-if="errorText && !places.length" class="state-card error">
        <text class="state-title">商家列表加载失败，请重试</text>
        <text class="state-desc">{{ errorText }}</text>
        <button class="state-button" @click="loadPlaces({ reset: true })">重新加载</button>
      </view>
      <view v-else-if="!places.length" class="state-card">
        <text class="state-title">暂无匹配商家</text>
        <text class="state-desc">换个商家名、档口号或减少筛选条件试试。</text>
        <button v-if="hasConditions" class="state-button secondary" @click="clearConditions">清除条件</button>
      </view>
      <template v-else>
        <MerchantPlaceCard
          v-for="place in places"
          :key="place.objectId"
          :place="place"
          @select="handlePlaceSelect"
          @detail="openMerchantDetail"
          @location="openMerchantLocation"
          @navigate="openPlaceLocation"
          @claim="openPlaceClaim"
        />
        <text class="load-more-text">{{ loadMoreText }}</text>
      </template>
    </view>

  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import MerchantPlaceCard from '../../components/MerchantPlaceCard.vue'
import { requireLogin } from '../../common/auth'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { getMerchantId } from '../../store/session'
import {
  listMapCategories,
  listMerchantPlaces,
  submitMapLocationCorrection,
} from '../../api/sourcingMap'
import {
  buildMerchantPlaceQuery,
  hasValidLocation,
  merchantDetailPath,
  normalizeMerchantPlace,
} from './merchantPlaceState'

const NAVIGATION_FEEDBACK_KEY = 'sourcing-map-navigation-feedback'
const LIST_PAGE_SIZE = 20

const keyword = ref('')
const claimedFilter = ref('all')
const categoryCodes = ref([])
const categories = ref([])
const places = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const errorText = ref('')
const loadedOnce = ref(false)
let requestVersion = 0
let shouldAskNavigationFeedback = false

const sourceFilters = [
  { label: '全部商家', value: 'all' },
  { label: '已入驻', value: 'claimed' },
  { label: '待认领', value: 'prelisted' },
]
const hasMore = computed(() => places.value.length < total.value)
const hasConditions = computed(() => Boolean(keyword.value.trim() || claimedFilter.value !== 'all' || categoryCodes.value.length))
const loadMoreText = computed(() => {
  if (loading.value) return '正在加载'
  if (hasMore.value) return '继续上拉查看更多'
  return places.value.length ? '已展示全部商家' : ''
})

onLoad(async () => {
  await Promise.all([loadCategories(), loadPlaces({ reset: true })])
  loadedOnce.value = true
})

onShow(() => {
  if (loadedOnce.value) promptNavigationFeedback()
})

onPullDownRefresh(async () => {
  await loadPlaces({ reset: true })
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (hasMore.value && !loading.value) {
    loadPlaces({ reset: false })
  }
})

async function loadCategories() {
  try {
    const resp = await listMapCategories({ type: 'booth_category' })
    categories.value = (resp.items || []).slice(0, 8)
  } catch (err) {
    // 分类配置失败不阻断商家浏览，用户仍可通过搜索和入驻状态查找。
    categories.value = []
  }
}

async function loadPlaces({ reset }) {
  if (loading.value && !reset) return
  const version = ++requestVersion
  const targetPage = reset ? 1 : page.value + 1
  loading.value = true
  if (reset) errorText.value = ''
  try {
    const resp = await listMerchantPlaces(buildMerchantPlaceQuery({
      cityCode: DEFAULT_CITY_CODE,
      keyword: keyword.value,
      categories: categoryCodes.value,
      claimed: claimedFilter.value,
      page: targetPage,
      pageSize: LIST_PAGE_SIZE,
    }))
    if (version !== requestVersion) return
    const nextItems = (resp.items || []).map(normalizeMerchantPlace)
    places.value = reset ? nextItems : [...places.value, ...nextItems]
    total.value = Number(resp.total || 0)
    page.value = targetPage
  } catch (err) {
    if (version !== requestVersion) return
    errorText.value = err.message || '网络连接不稳定，请稍后重试'
    if (reset) {
      places.value = []
      total.value = 0
    }
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function submitSearch() {
  loadPlaces({ reset: true })
}

function selectSourceFilter(value) {
  claimedFilter.value = value
  loadPlaces({ reset: true })
}

function toggleCategory(code) {
  categoryCodes.value = categoryCodes.value.includes(code)
    ? categoryCodes.value.filter((item) => item !== code)
    : [...categoryCodes.value, code]
  loadPlaces({ reset: true })
}

function clearConditions() {
  keyword.value = ''
  claimedFilter.value = 'all'
  categoryCodes.value = []
  loadPlaces({ reset: true })
}

function handlePlaceSelect(place) {
  if (merchantDetailPath(place)) {
    openMerchantDetail(place)
    return
  }
  uni.showModal({
    title: place.code || '平台预录档口',
    content: '这是平台预录的档口资料，尚未有商家完成认领。',
    confirmText: '这是我的档口',
    success: ({ confirm }) => {
      if (confirm) openPlaceClaim(place)
    },
  })
}

function openMerchantDetail(place) {
  const url = merchantDetailPath(place)
  if (!url) return
  // 档口卡片保留独立的商家主页入口，方便买家先看经营资料再决定是否联系。
  uni.navigateTo({ url })
}

function openMerchantLocation(place) {
  if (!place.claimed || !place.merchantId || !hasValidLocation(place)) return
  // 已入驻档口进入商家位置页；待认领点位没有可靠的商家上下文，只允许直接导航。
  uni.navigateTo({ url: `/pages/merchant/location?merchantId=${encodeURIComponent(place.merchantId)}` })
}

function openPlaceClaim(place) {
  if (!requireLogin()) return
  const merchantId = getMerchantId()
  const merchantQuery = merchantId ? `&merchantId=${merchantId}` : ''
  uni.navigateTo({ url: `/pages/merchant/map-binding?objectId=${place.objectId}${merchantQuery}` })
}

function openPlaceLocation(place) {
  if (!hasValidLocation(place)) {
    uni.showToast({ title: '该档口位置待完善，暂时无法导航', icon: 'none' })
    return
  }
  uni.setStorageSync(NAVIGATION_FEEDBACK_KEY, {
    objectId: place.objectId,
    name: place.name,
  })
  shouldAskNavigationFeedback = true
  uni.openLocation({
    latitude: Number(place.lat),
    longitude: Number(place.lng),
    name: place.name,
    address: [place.marketName, place.buildingName, place.floorNo, place.address].filter(Boolean).join(' · '),
    scale: 18,
    fail: () => {
      shouldAskNavigationFeedback = false
      uni.removeStorageSync(NAVIGATION_FEEDBACK_KEY)
      uni.showToast({ title: '导航打开失败，请稍后重试', icon: 'none' })
    },
  })
}

function promptNavigationFeedback() {
  if (!shouldAskNavigationFeedback) return
  shouldAskNavigationFeedback = false
  const target = uni.getStorageSync(NAVIGATION_FEEDBACK_KEY)
  uni.removeStorageSync(NAVIGATION_FEEDBACK_KEY)
  if (!target?.objectId) return
  uni.showModal({
    title: '位置是否准确？',
    content: `${target.name || '该档口'} 的导航位置是否准确？`,
    confirmText: '位置有误',
    cancelText: '位置准确',
    success: ({ confirm }) => {
      if (confirm) submitLocationCorrection(target.objectId)
    },
  })
}

async function submitLocationCorrection(objectId) {
  if (!requireLogin()) return
  try {
    await submitMapLocationCorrection(objectId, {
      reasonCode: 'navigation_inaccurate',
      description: '用户在腾讯地图导航后反馈位置不准确',
    })
    uni.showToast({ title: '反馈已提交', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '反馈提交失败，请重试', icon: 'none' })
  }
}

</script>

<style lang="scss" scoped>
.sourcing-map-page {
  min-height: 100vh;
  background: #f3f6fa;
  color: #172033;
}

.directory-toolbar {
  position: sticky;
  top: 0;
  z-index: 10;
  padding: 20rpx 24rpx 16rpx;
  border-bottom: 1rpx solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.97);
}

.search-row {
  display: flex;
  align-items: center;
  gap: 14rpx;
}

.search-field {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 14rpx;
  height: 76rpx;
  padding: 0 22rpx;
  border: 1rpx solid #cfd6e1;
  border-radius: 12rpx;
  background: #f8fafc;
}

.search-field input {
  flex: 1;
  font-size: 27rpx;
}

.search-icon {
  position: relative;
  width: 24rpx;
  height: 24rpx;
  border: 3rpx solid #65748a;
  border-radius: 50%;
}

.search-icon::after {
  position: absolute;
  right: -9rpx;
  bottom: -6rpx;
  width: 12rpx;
  height: 3rpx;
  background: #65748a;
  content: '';
  transform: rotate(45deg);
}

.search-button,
.state-button {
  margin: 0;
  border-radius: 10rpx;
  background: #172033;
  color: #ffffff;
  font-size: 25rpx;
}

.search-button {
  width: 112rpx;
  line-height: 76rpx;
}

.search-button::after,
.filter-chip::after,
.state-button::after {
  border: 0;
}

.directory-summary {
  display: flex;
  justify-content: flex-end;
  margin-top: 18rpx;
}

.result-count {
  color: #7a8799;
  font-size: 22rpx;
}

.filter-scroll {
  width: 100%;
  margin-top: 16rpx;
  white-space: nowrap;
}

.filter-chip {
  display: inline-block;
  margin: 0 10rpx 0 0;
  padding: 0 20rpx;
  border: 1rpx solid #d7dee8;
  border-radius: 999rpx;
  background: #ffffff;
  color: #607086;
  font-size: 22rpx;
  line-height: 52rpx;
}

.filter-chip.active {
  border-color: #172033;
  background: #172033;
  color: #ffffff;
}

.merchant-list {
  display: grid;
  gap: 18rpx;
  padding: 20rpx 24rpx calc(40rpx + env(safe-area-inset-bottom));
}

.state-card {
  display: grid;
  justify-items: center;
  gap: 14rpx;
  margin-top: 120rpx;
  padding: 44rpx 28rpx;
  text-align: center;
}

.state-title {
  font-size: 30rpx;
  font-weight: 750;
}

.state-desc {
  color: #7a8799;
  font-size: 24rpx;
  line-height: 1.6;
}

.state-button {
  min-width: 180rpx;
  margin-top: 10rpx;
  line-height: 68rpx;
}

.state-button.secondary {
  border: 1rpx solid #cfd6e1;
  background: #ffffff;
  color: #172033;
}

.load-more-text {
  padding: 14rpx;
  color: #8b97a8;
  font-size: 22rpx;
  text-align: center;
}

</style>
