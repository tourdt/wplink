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

      <view class="mode-row">
        <view class="view-switch">
          <button :class="{ active: viewMode === 'list' }" @click="switchView('list')">商家列表</button>
          <button :class="{ active: viewMode === 'map' }" @click="switchView('map')">地图找货</button>
        </view>
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

    <view v-if="viewMode === 'list'" class="merchant-list">
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
          @navigate="openPlaceLocation"
          @claim="openPlaceClaim"
        />
        <text class="load-more-text">{{ loadMoreText }}</text>
      </template>
    </view>

    <view v-else class="native-map-shell">
      <map
        id="merchantTencentMap"
        class="tencent-map"
        :latitude="mapCenter.latitude"
        :longitude="mapCenter.longitude"
        :markers="markers"
        :scale="mapScale"
        show-location
        @markertap="handleMarkerTap"
        @regionchange="handleRegionChange"
      />
      <button v-if="searchAreaVisible" class="search-area-button" @click="searchCurrentMapRegion">搜索此区域</button>
      <view v-if="loading" class="map-loading-pill">正在更新商家</view>
      <view v-if="!markers.length && !loading" class="map-empty-pill">当前结果暂无可用位置，可切回商家列表查看</view>

      <view v-if="selectedPlace" class="marker-card">
        <MerchantPlaceCard
          :place="selectedPlace"
          selected
          @select="handlePlaceSelect"
          @detail="openMerchantDetail"
          @navigate="openPlaceLocation"
          @claim="openPlaceClaim"
        />
        <view v-if="!selectedPlace.claimed" class="prelisted-note">
          <text>平台预录档口，商家可通过卡片入口提交资料认领。</text>
        </view>
        <view class="feedback-row">
          <button @click="submitPlaceLocationCorrection(selectedPlace)">位置纠错</button>
          <button @click="submitPlaceRiskReport(selectedPlace)">举报问题</button>
        </view>
      </view>
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
  submitMapRiskReport,
} from '../../api/sourcingMap'
import {
  buildMerchantPlaceQuery,
  hasValidLocation,
  merchantDetailPath,
  normalizeMerchantPlace,
} from './merchantPlaceState'
import {
  buildTencentMapMarkers,
  fallbackMapCenter,
  placeIdFromMarker,
} from './tencentMapState'

const DEFAULT_MAP_CENTER = { latitude: 30.87, longitude: 120.12 }
const NAVIGATION_FEEDBACK_KEY = 'sourcing-map-navigation-feedback'
const LIST_PAGE_SIZE = 20
const MAP_PAGE_SIZE = 100

const viewMode = ref('list')
const keyword = ref('')
const claimedFilter = ref('all')
const categoryCodes = ref([])
const categories = ref([])
const places = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const errorText = ref('')
const bounds = ref(null)
const selectedObjectId = ref('')
const searchAreaVisible = ref(false)
const mapCenter = ref({ ...DEFAULT_MAP_CENTER })
const mapScale = ref(14)
const loadedOnce = ref(false)
let requestVersion = 0
let shouldAskNavigationFeedback = false

const sourceFilters = [
  { label: '全部商家', value: 'all' },
  { label: '已入驻', value: 'claimed' },
  { label: '待认领', value: 'prelisted' },
]
const hasMore = computed(() => places.value.length < total.value)
const hasConditions = computed(() => Boolean(keyword.value.trim() || claimedFilter.value !== 'all' || categoryCodes.value.length || bounds.value))
const loadMoreText = computed(() => {
  if (loading.value) return '正在加载'
  if (hasMore.value) return '继续上拉查看更多'
  return places.value.length ? '已展示全部商家' : ''
})
const selectedPlace = computed(() => places.value.find((place) => place.objectId === selectedObjectId.value) || null)
const markers = computed(() => buildTencentMapMarkers(places.value, selectedObjectId.value))

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
  if (viewMode.value === 'list' && hasMore.value && !loading.value) {
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
      pageSize: viewMode.value === 'map' ? MAP_PAGE_SIZE : LIST_PAGE_SIZE,
      bounds: bounds.value,
    }))
    if (version !== requestVersion) return
    const nextItems = (resp.items || []).map(normalizeMerchantPlace)
    places.value = reset ? nextItems : [...places.value, ...nextItems]
    total.value = Number(resp.total || 0)
    page.value = targetPage
    if (!selectedPlace.value) selectedObjectId.value = ''
    if (viewMode.value === 'map') {
      mapCenter.value = fallbackMapCenter(places.value, mapCenter.value)
    }
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
  bounds.value = null
  searchAreaVisible.value = false
  loadPlaces({ reset: true })
}

function selectSourceFilter(value) {
  claimedFilter.value = value
  bounds.value = null
  loadPlaces({ reset: true })
}

function toggleCategory(code) {
  categoryCodes.value = categoryCodes.value.includes(code)
    ? categoryCodes.value.filter((item) => item !== code)
    : [...categoryCodes.value, code]
  bounds.value = null
  loadPlaces({ reset: true })
}

function clearConditions() {
  keyword.value = ''
  claimedFilter.value = 'all'
  categoryCodes.value = []
  bounds.value = null
  searchAreaVisible.value = false
  loadPlaces({ reset: true })
}

function switchView(mode) {
  if (viewMode.value === mode) return
  viewMode.value = mode
  if (mode === 'map') {
    selectedObjectId.value = ''
    mapCenter.value = fallbackMapCenter(places.value, mapCenter.value)
    locateUser()
  }
  // 地图一次读取较完整的点位集合，列表恢复轻量分页；筛选条件在两种视图间保持一致。
  loadPlaces({ reset: true })
}

function locateUser() {
  // 微信原生地图使用 GCJ-02；拒绝定位时保留商家点位中心，不阻断地图浏览。
  uni.getLocation({
    type: 'gcj02',
    success: ({ latitude, longitude }) => {
      mapCenter.value = { latitude: Number(latitude), longitude: Number(longitude) }
    },
  })
}

function handleRegionChange(event) {
  if (event.type === 'end' && ['drag', 'scale'].includes(event.causedBy)) {
    searchAreaVisible.value = true
  }
}

function searchCurrentMapRegion() {
  const mapContext = uni.createMapContext('merchantTencentMap')
  mapContext.getRegion({
    success: ({ southwest, northeast }) => {
      bounds.value = {
        minLat: southwest.latitude,
        maxLat: northeast.latitude,
        minLng: southwest.longitude,
        maxLng: northeast.longitude,
      }
      searchAreaVisible.value = false
      selectedObjectId.value = ''
      loadPlaces({ reset: true })
    },
    fail: () => {
      uni.showToast({ title: '地图区域读取失败，请重试', icon: 'none' })
    },
  })
}

function handleMarkerTap(event) {
  selectedObjectId.value = placeIdFromMarker(event.detail.markerId, markers.value)
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
  // 列表卡片和地图摘要卡共用同一入口，避免地图模式只选中商家却无法继续查看主页。
  uni.navigateTo({ url })
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

function submitPlaceLocationCorrection(place) {
  submitLocationCorrection(place.objectId)
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

async function submitPlaceRiskReport(place) {
  if (!requireLogin()) return
  try {
    await submitMapRiskReport(place.objectId, {
      reasonCode: 'false_information',
      description: '用户从商家目录反馈档口资料可能有误',
    })
    uni.showToast({ title: '举报已提交', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '举报提交失败，请重试', icon: 'none' })
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

.search-row,
.mode-row {
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
.view-switch button::after,
.filter-chip::after,
.state-button::after,
.search-area-button::after,
.prelisted-note button::after,
.feedback-row button::after {
  border: 0;
}

.mode-row {
  justify-content: space-between;
  margin-top: 18rpx;
}

.view-switch {
  display: flex;
  padding: 5rpx;
  border-radius: 10rpx;
  background: #eef2f7;
}

.view-switch button {
  margin: 0;
  padding: 0 24rpx;
  border-radius: 7rpx;
  background: transparent;
  color: #65748a;
  font-size: 23rpx;
  line-height: 54rpx;
}

.view-switch button.active {
  background: #ffffff;
  color: #172033;
  font-weight: 700;
  box-shadow: 0 3rpx 10rpx rgba(15, 23, 42, 0.1);
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

.native-map-shell {
  position: relative;
  height: calc(100vh - 238rpx - env(safe-area-inset-bottom));
  min-height: 720rpx;
}

.tencent-map {
  width: 100%;
  height: 100%;
}

.search-area-button,
.map-loading-pill,
.map-empty-pill {
  position: absolute;
  left: 50%;
  z-index: 3;
  transform: translateX(-50%);
}

.search-area-button {
  top: 24rpx;
  margin: 0;
  padding: 0 28rpx;
  border-radius: 999rpx;
  background: #172033;
  color: #ffffff;
  font-size: 23rpx;
  line-height: 64rpx;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.2);
}

.map-loading-pill,
.map-empty-pill {
  top: 104rpx;
  max-width: 620rpx;
  padding: 14rpx 22rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.94);
  color: #607086;
  font-size: 22rpx;
  text-align: center;
  box-shadow: 0 5rpx 16rpx rgba(15, 23, 42, 0.12);
}

.marker-card {
  position: absolute;
  right: 20rpx;
  bottom: calc(20rpx + env(safe-area-inset-bottom));
  left: 20rpx;
  z-index: 4;
}

.prelisted-note,
.feedback-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14rpx;
  padding: 14rpx 20rpx;
  background: #fff7f2;
}

.prelisted-note text {
  color: #8a3b1c;
  font-size: 21rpx;
}

.prelisted-note button,
.feedback-row button {
  margin: 0;
  background: transparent;
  color: #a83200;
  font-size: 21rpx;
  line-height: 48rpx;
}

.feedback-row {
  justify-content: flex-end;
  border-top: 1rpx solid rgba(194, 58, 0, 0.12);
  border-radius: 0 0 14rpx 14rpx;
}
</style>
