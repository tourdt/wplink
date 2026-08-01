<template>
  <view class="location-page">
    <view v-if="initialLoading" class="status-card">
      <view class="status-indicator" />
      <text class="status-title">正在加载店铺位置</text>
      <text class="status-copy">正在确认档口坐标与周边信息</text>
    </view>

    <view v-else-if="errorText || !context.current" class="status-card error-card">
      <text class="status-kicker">LOCATION UNAVAILABLE</text>
      <text class="status-title">{{ errorText || '该商家暂时无法查看' }}</text>
      <text class="status-copy">你可以重新加载，或先返回上一页查看商家资料。</text>
      <view class="status-actions">
        <button class="primary-action" @click="loadLocationContext()">重新加载</button>
        <button class="secondary-action" @click="goBack">返回</button>
      </view>
    </view>

    <template v-else>
      <view class="map-heading">
        <view>
          <text class="map-eyebrow">织里档口位置</text>
          <text class="map-title">确认到店路线</text>
        </view>
        <text class="map-radius">{{ radiusText }}</text>
      </view>

      <view class="map-workspace">
        <view class="map-frame">
          <map
            id="merchantLocationMap"
            class="merchant-map"
            :latitude="mapCenter.latitude"
            :longitude="mapCenter.longitude"
            :markers="markers"
            :scale="mapScale"
            @markertap="handleMarkerTap"
          />
        </view>

        <view :class="['nearby-drawer', { expanded: nearbyDrawerOpen }]">
          <view v-if="!context.nearbyAvailable" class="nearby-fallback">
            <text class="nearby-fallback-text">周边商家加载失败，请重试</text>
            <button class="nearby-retry" :disabled="loading" @click="retryNearby">
              {{ loading ? '加载中' : '重试周边' }}
            </button>
          </view>

          <button
            v-else-if="context.nearby.length"
            class="nearby-toggle"
            @click="nearbyDrawerOpen ? closeNearbyDrawer() : openNearbyDrawer()"
          >
            <text class="nearby-title">周边已入驻商家</text>
            <view class="nearby-toggle-meta">
              <text class="nearby-count">{{ context.nearby.length }} 家</text>
              <text class="nearby-chevron">{{ nearbyDrawerOpen ? '收起' : '展开' }}</text>
            </view>
          </button>

          <text v-else class="nearby-empty">附近暂无其他入驻商家</text>

          <scroll-view
            v-if="nearbyDrawerOpen && context.nearbyAvailable && context.nearby.length"
            class="nearby-list"
            scroll-y
            scroll-with-animation
            :scroll-into-view="scrollIntoViewId"
          >
            <view
              v-for="place in context.nearby"
              :id="nearbyMerchantDomId(place.merchantId)"
              :key="place.merchantId"
              :class="['nearby-item', { selected: place.merchantId === selectedNearbyMerchantId }]"
              @click="selectNearbyMerchant(place.merchantId)"
            >
              <view class="nearby-item-heading">
                <text class="nearby-name">{{ place.name }}</text>
                <text class="nearby-distance">{{ place.distanceText || formatDistance(place.distanceMeters) }}</text>
              </view>
              <view v-if="merchantMainTags(place).length" class="nearby-tags">
                <text v-for="tag in merchantMainTags(place)" :key="tag" class="nearby-tag">{{ tag }}</text>
              </view>
              <text class="nearby-address">{{ merchantAddress(place) }}</text>
              <button class="nearby-detail" @click.stop="openNearbyMerchant(place)">查看商家</button>
            </view>
          </scroll-view>
        </view>
      </view>

      <view class="merchant-sheet">
        <view class="merchant-heading">
          <view class="merchant-heading-copy">
            <text class="current-label">当前档口</text>
            <text class="merchant-name">{{ context.current.name }}</text>
          </view>
          <text v-if="context.current.code" class="booth-code">{{ context.current.code }}</text>
        </view>

        <view class="merchant-facts">
          <view class="merchant-fact">
            <text class="fact-label">店铺地址</text>
            <text class="fact-value">{{ currentAddress }}</text>
          </view>
          <view class="merchant-fact">
            <text class="fact-label">主营</text>
            <text class="fact-value">{{ currentCategories }}</text>
          </view>
        </view>

        <button class="navigate-button" @click="openCurrentLocation">
          <text class="navigate-label">导航到店</text>
          <text class="navigate-arrow">→</text>
        </button>
      </view>
    </template>
  </view>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantLocationContext } from '../../api/sourcingMap'
import {
  buildMerchantLocationMarkers,
  merchantMainTags,
  merchantIdFromMarker,
  nearbyMerchantDomId,
  normalizeMerchantLocationContext,
} from './locationState'

const merchantId = ref('')
const context = ref(normalizeMerchantLocationContext())
const loading = ref(false)
const errorText = ref('')
const selectedNearbyMerchantId = ref('')
const nearbyDrawerOpen = ref(false)
const scrollIntoViewId = ref('')
const mapScale = ref(16)

const initialLoading = computed(() => loading.value && !context.value.current)
const markers = computed(() => buildMerchantLocationMarkers(context.value, selectedNearbyMerchantId.value))
const mapCenter = computed(() => ({
  latitude: Number(context.value.current?.lat || 0),
  longitude: Number(context.value.current?.lng || 0),
}))
const currentAddress = computed(() => {
  return merchantAddress(context.value.current, '店铺地址待完善')
})
const currentCategories = computed(() => {
  const current = context.value.current || {}
  return [...(current.categoryCodes || []), ...(current.serviceTags || [])].slice(0, 4).join(' · ') || '主营待完善'
})
const radiusText = computed(() => {
  const meters = context.value.radiusMeters
  if (!meters) return '周边信息'
  return meters >= 1000 ? `${(meters / 1000).toFixed(meters % 1000 === 0 ? 0 : 1)}km` : `${meters}m`
})

onLoad(async (options) => {
  // 路由只接受明确的商家 ID，避免分享链接缺参时误用其他页面残留状态。
  merchantId.value = String(options.merchantId || '').trim()
  if (!merchantId.value) {
    errorText.value = '该商家暂时无法查看'
    return
  }
  await loadLocationContext()
})

async function loadLocationContext({ preserveCurrent = false } = {}) {
  if (!merchantId.value || loading.value) return
  loading.value = true
  if (!preserveCurrent) errorText.value = ''

  try {
    const raw = await getMerchantLocationContext(merchantId.value)
    const nextContext = normalizeMerchantLocationContext(raw)
    if (!nextContext.current) {
      if (!preserveCurrent) {
        context.value = normalizeMerchantLocationContext()
        errorText.value = '该商家位置待完善'
      }
      return
    }
    context.value = nextContext
    errorText.value = ''
    selectedNearbyMerchantId.value = ''
    scrollIntoViewId.value = ''
    nearbyDrawerOpen.value = false
    mapScale.value = 16
  } catch (err) {
    console.error('加载商家位置上下文失败', { merchantId: merchantId.value, err })
    if (preserveCurrent && context.value.current) {
      // 周边信息只是辅助能力；重试失败时保留当前地图和导航，避免次要依赖阻断到店主流程。
      context.value = { ...context.value, nearbyAvailable: false }
      nearbyDrawerOpen.value = false
      selectedNearbyMerchantId.value = ''
      scrollIntoViewId.value = ''
      uni.showToast({ title: '周边商家加载失败，请重试', icon: 'none' })
      return
    }
    const message = String(err?.message || '')
    errorText.value = message.includes('位置待完善') ? '该商家位置待完善' : '该商家暂时无法查看'
  } finally {
    loading.value = false
  }
}

function retryNearby() {
  loadLocationContext({ preserveCurrent: true })
}

function handleMarkerTap(event) {
  const markerId = event?.detail?.markerId ?? event?.markerId
  const tappedMerchantId = merchantIdFromMarker(markerId, markers.value)
  if (!tappedMerchantId) return
  if (tappedMerchantId === context.value.current.merchantId) {
    closeNearbyDrawer()
    return
  }
  selectNearbyMerchant(tappedMerchantId)
}

function openNearbyDrawer() {
  if (!context.value.nearbyAvailable || !context.value.nearby.length) return
  nearbyDrawerOpen.value = true
}

function closeNearbyDrawer() {
  nearbyDrawerOpen.value = false
  selectedNearbyMerchantId.value = ''
  scrollIntoViewId.value = ''
  mapScale.value = 16

  const current = context.value.current
  if (!current) return
  // 关闭周边浏览后主动恢复当前档口中心，确保临时选择不会改变位置页的主商家上下文。
  nextTick(() => {
    uni.createMapContext('merchantLocationMap').moveToLocation({
      latitude: Number(current.lat),
      longitude: Number(current.lng),
      fail(err) {
        console.warn('恢复当前商家地图中心失败', { merchantId: current.merchantId, err })
      },
    })
  })
}

function selectNearbyMerchant(merchantId) {
  const normalizedMerchantId = String(merchantId || '').trim()
  const place = context.value.nearby.find((item) => item.merchantId === normalizedMerchantId)
  if (!place || !context.value.current) return

  selectedNearbyMerchantId.value = normalizedMerchantId
  nearbyDrawerOpen.value = true
  scrollIntoViewId.value = nearbyMerchantDomId(normalizedMerchantId)
  mapScale.value = 14

  // 列表选择只调整临时视野，同时纳入当前档口和周边档口，不能把周边商家替换成页面主上下文。
  nextTick(() => {
    uni.createMapContext('merchantLocationMap').includePoints({
      points: [context.value.current, place].map((item) => ({
        latitude: Number(item.lat),
        longitude: Number(item.lng),
      })),
      padding: [72, 48, 72, 48],
      fail(err) {
        console.warn('调整周边商家地图视野失败', { merchantId: normalizedMerchantId, err })
      },
    })
  })
}

function openNearbyMerchant(place) {
  const normalizedMerchantId = String(place?.merchantId || '').trim()
  if (!normalizedMerchantId) return
  uni.navigateTo({
    url: `/pages/merchant/detail?id=${encodeURIComponent(normalizedMerchantId)}`,
    fail(err) {
      console.warn('打开周边商家详情失败', { merchantId: normalizedMerchantId, err })
      uni.showToast({ title: '商家详情打开失败，请稍后重试', icon: 'none' })
    },
  })
}

function openCurrentLocation() {
  const current = context.value.current
  if (!current) return
  uni.openLocation({
    latitude: Number(current.lat),
    longitude: Number(current.lng),
    name: current.name,
    address: currentAddress.value,
    scale: 18,
    fail(err) {
      console.warn('打开商家导航失败', { merchantId: current.merchantId, err })
      uni.showToast({ title: '导航打开失败，请稍后重试', icon: 'none' })
    },
  })
}

function goBack() {
  // 独立分享链接可能没有可返回页面，此时降级到稳定的拿货档口入口。
  uni.navigateBack({
    delta: 1,
    fail() {
      uni.switchTab({ url: '/pages/sourcing-map/index' })
    },
  })
}

function formatDistance(distanceMeters) {
  const meters = Number(distanceMeters)
  if (!Number.isFinite(meters) || meters <= 0) return '附近'
  return meters < 1000 ? `${Math.round(meters)}m` : `${(meters / 1000).toFixed(1)}km`
}

function merchantAddress(place = {}, fallback = '档口地址待完善') {
  return [place.marketName, place.buildingName, place.floorNo, place.address]
    .filter(Boolean)
    .join(' · ') || fallback
}
</script>

<style lang="scss" scoped>
.location-page {
  min-height: 100vh;
  padding: 20rpx;
  background: $wplink-bg;
}

.map-heading,
.merchant-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
}

.map-heading {
  padding: 8rpx 4rpx 20rpx;
}

.map-eyebrow,
.status-kicker {
  display: block;
  color: $wplink-warning;
  font-size: 20rpx;
  font-weight: 800;
  letter-spacing: 3rpx;
}

.map-title {
  display: block;
  margin-top: 4rpx;
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 800;
  letter-spacing: -1rpx;
}

.map-radius {
  flex: none;
  padding: 8rpx 14rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 8rpx;
  background: $wplink-card;
  color: #64748b;
  font-size: 22rpx;
  font-weight: 700;
}

.map-frame {
  overflow: hidden;
  border: 1rpx solid #cbd5e1;
  border-radius: 12rpx 12rpx 0 0;
  background: #dce5ef;
}

.map-workspace {
  position: relative;
}

.merchant-map {
  display: block;
  width: 100%;
  height: 680rpx;
}

.merchant-sheet {
  position: relative;
  display: grid;
  gap: 24rpx;
  padding: 28rpx;
  border-top: 8rpx solid $wplink-warning;
  border-radius: 0 0 12rpx 12rpx;
  background: $wplink-card;
  box-shadow: 0 18rpx 52rpx rgba(6, 22, 37, 0.12);
}

.merchant-heading-copy {
  min-width: 0;
}

.current-label {
  display: inline-flex;
  padding: 5rpx 12rpx;
  border-radius: 6rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 20rpx;
  font-weight: 700;
  letter-spacing: 2rpx;
}

.merchant-name {
  display: block;
  overflow: hidden;
  margin-top: 12rpx;
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.booth-code {
  flex: none;
  min-width: 104rpx;
  padding: 16rpx 12rpx;
  border: 2rpx solid $wplink-primary;
  border-radius: 8rpx;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 800;
  text-align: center;
}

.merchant-facts {
  display: grid;
  gap: 16rpx;
  padding: 20rpx;
  border-left: 4rpx solid #94a3b8;
  background: #f8fafc;
}

.merchant-fact {
  display: grid;
  grid-template-columns: 112rpx minmax(0, 1fr);
  gap: 12rpx;
}

.fact-label {
  color: #64748b;
  font-size: 24rpx;
}

.fact-value {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-word;
}

.navigate-button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 92rpx;
  padding: 0 28rpx;
  border-radius: 10rpx;
  background: $wplink-warning;
  color: $wplink-card;
}

.navigate-label {
  font-size: 30rpx;
  font-weight: 800;
}

.navigate-arrow {
  font-size: 38rpx;
  font-weight: 400;
}

.nearby-drawer {
  position: absolute;
  right: 16rpx;
  bottom: 16rpx;
  left: 16rpx;
  z-index: 5;
  overflow: hidden;
  border: 1rpx solid #cbd5e1;
  border-radius: 10rpx;
  background: $wplink-card;
  box-shadow: 0 16rpx 38rpx rgba(6, 22, 37, 0.18);
}

.nearby-drawer.expanded {
  display: flex;
  flex-direction: column;
  height: 55vh;
  max-height: 55vh;
}

.nearby-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 88rpx;
  padding: 0 22rpx;
  border-radius: 0;
  background: $wplink-card;
  text-align: left;
}

.nearby-toggle::after,
.nearby-detail::after,
.nearby-retry::after {
  border: 0;
}

.nearby-title {
  color: $wplink-primary;
  font-size: 27rpx;
  font-weight: 750;
}

.nearby-toggle-meta,
.nearby-item-heading,
.nearby-fallback {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.nearby-count,
.nearby-distance,
.nearby-empty,
.nearby-fallback-text {
  color: #64748b;
  font-size: 22rpx;
}

.nearby-chevron {
  color: $wplink-warning;
  font-size: 21rpx;
  font-weight: 800;
}

.nearby-list {
  flex: 1;
  min-height: 0;
  width: 100%;
  height: calc(55vh - 88rpx);
  max-height: calc(55vh - 88rpx);
  border-top: 1rpx solid $wplink-line;
}

.nearby-item {
  display: grid;
  gap: 12rpx;
  margin: 0 18rpx;
  padding: 22rpx 4rpx;
  border-bottom: 1rpx solid $wplink-line;
  text-align: left;
}

.nearby-item.selected {
  margin: 0;
  padding-right: 22rpx;
  padding-left: 22rpx;
  border-left: 5rpx solid $wplink-warning;
  background: #f1f5f9;
}

.nearby-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: $wplink-primary;
  font-size: 27rpx;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nearby-distance {
  flex: none;
  font-weight: 700;
}

.nearby-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
}

.nearby-tag {
  padding: 5rpx 10rpx;
  border: 1rpx solid #cbd5e1;
  border-radius: 5rpx;
  color: #475569;
  font-size: 20rpx;
  line-height: 1.2;
}

.nearby-address {
  overflow: hidden;
  color: #64748b;
  font-size: 22rpx;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nearby-detail {
  min-height: 64rpx;
  margin: 0;
  padding: 0 18rpx;
  border: 1rpx solid #94a3b8;
  border-radius: 7rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 22rpx;
  font-weight: 800;
  line-height: 62rpx;
}

.nearby-fallback {
  min-height: 88rpx;
  padding: 0 22rpx;
}

.nearby-retry {
  flex: none;
  margin: 0;
  padding: 10rpx 16rpx;
  border: 1rpx solid #94a3b8;
  border-radius: 7rpx;
  background: $wplink-card;
  color: #475569;
  font-size: 22rpx;
  font-weight: 700;
}

.nearby-empty {
  display: block;
  min-height: 88rpx;
  padding: 0 22rpx;
  line-height: 88rpx;
}

.status-card {
  display: grid;
  gap: 16rpx;
  align-content: center;
  min-height: calc(100vh - 40rpx);
  padding: 48rpx 36rpx;
  border-top: 8rpx solid $wplink-primary;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.error-card {
  border-top-color: $wplink-warning;
}

.status-indicator {
  width: 44rpx;
  height: 44rpx;
  margin: 0 auto 8rpx;
  border: 6rpx solid #d8e0ec;
  border-top-color: $wplink-warning;
  border-radius: 50%;
}

.status-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 800;
}

.status-copy {
  color: $wplink-muted;
  font-size: 25rpx;
  line-height: 1.6;
}

.status-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
  margin-top: 18rpx;
}

.primary-action,
.secondary-action {
  min-height: 84rpx;
  border-radius: 9rpx;
  font-size: 27rpx;
  font-weight: 800;
}

.primary-action {
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-action {
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
  color: $wplink-primary;
}
</style>
