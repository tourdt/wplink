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

      <view class="map-frame">
        <map
          class="merchant-map"
          :latitude="mapCenter.latitude"
          :longitude="mapCenter.longitude"
          :markers="markers"
          :scale="16"
          @markertap="handleMarkerTap"
        />
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

        <view class="nearby-section">
          <view class="nearby-heading">
            <text class="nearby-title">周边已入驻档口</text>
            <text class="nearby-count">{{ context.nearby.length }} 家</text>
          </view>

          <view v-if="!context.nearbyAvailable" class="nearby-fallback">
            <text class="nearby-fallback-text">周边商家加载失败，请重试</text>
            <button class="nearby-retry" :disabled="loading" @click="retryNearby">
              {{ loading ? '加载中' : '重试周边' }}
            </button>
          </view>
          <scroll-view v-else-if="context.nearby.length" class="nearby-list" scroll-x>
            <view class="nearby-list-inner">
              <button
                v-for="place in context.nearby"
                :key="place.merchantId || place.objectId"
                :class="['nearby-item', { selected: place.merchantId === selectedNearbyMerchantId }]"
                @click="selectNearbyMerchant(place)"
              >
                <text class="nearby-name">{{ place.name }}</text>
                <text class="nearby-meta">{{ place.distanceText || formatDistance(place.distanceMeters) }}</text>
              </button>
            </view>
          </scroll-view>
          <text v-else class="nearby-empty">{{ radiusText }}内暂无其他已入驻档口</text>
        </view>
      </view>
    </template>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantLocationContext } from '../../api/sourcingMap'
import {
  buildMerchantLocationMarkers,
  merchantIdFromMarker,
  normalizeMerchantLocationContext,
} from './locationState'

const merchantId = ref('')
const context = ref(normalizeMerchantLocationContext())
const loading = ref(false)
const errorText = ref('')
const selectedNearbyMerchantId = ref('')

const initialLoading = computed(() => loading.value && !context.value.current)
const markers = computed(() => buildMerchantLocationMarkers(context.value, selectedNearbyMerchantId.value))
const mapCenter = computed(() => ({
  latitude: Number(context.value.current?.lat || 0),
  longitude: Number(context.value.current?.lng || 0),
}))
const currentAddress = computed(() => {
  const current = context.value.current || {}
  return [current.marketName, current.buildingName, current.floorNo, current.address]
    .filter(Boolean)
    .join(' · ') || '店铺地址待完善'
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
  } catch (err) {
    console.error('加载商家位置上下文失败', { merchantId: merchantId.value, err })
    if (preserveCurrent && context.value.current) {
      // 周边信息只是辅助能力；重试失败时保留当前地图和导航，避免次要依赖阻断到店主流程。
      context.value = { ...context.value, nearby: [], nearbyAvailable: false }
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
  selectedNearbyMerchantId.value = tappedMerchantId === context.value.current.merchantId ? '' : tappedMerchantId
}

function selectNearbyMerchant(place) {
  selectedNearbyMerchantId.value = String(place?.merchantId || '').trim()
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
</script>

<style lang="scss" scoped>
.location-page {
  min-height: 100vh;
  padding: 20rpx;
  background: $wplink-bg;
}

.map-heading,
.merchant-heading,
.nearby-heading {
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

.nearby-section {
  display: grid;
  gap: 16rpx;
  padding-top: 24rpx;
  border-top: 1rpx solid $wplink-line;
}

.nearby-title {
  color: $wplink-primary;
  font-size: 27rpx;
  font-weight: 750;
}

.nearby-count,
.nearby-meta,
.nearby-empty,
.nearby-fallback-text {
  color: #64748b;
  font-size: 22rpx;
}

.nearby-list {
  width: 100%;
  white-space: nowrap;
}

.nearby-list-inner {
  display: inline-flex;
  gap: 12rpx;
  padding-right: 20rpx;
}

.nearby-item {
  display: grid;
  justify-items: start;
  gap: 5rpx;
  min-width: 220rpx;
  max-width: 280rpx;
  padding: 16rpx 18rpx;
  border: 1rpx solid #cbd5e1;
  border-radius: 8rpx;
  background: #f8fafc;
  text-align: left;
}

.nearby-item.selected {
  border-color: #475569;
  background: #e2e8f0;
}

.nearby-name {
  display: block;
  width: 100%;
  overflow: hidden;
  color: #334155;
  font-size: 24rpx;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nearby-fallback {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 18rpx;
  border-radius: 8rpx;
  background: #f1f5f9;
}

.nearby-retry {
  flex: none;
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
  padding: 18rpx;
  border-radius: 8rpx;
  background: #f8fafc;
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
