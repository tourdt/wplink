<template>
  <view class="sourcing-map-page">
    <view class="map-header">
      <view class="header-copy">
        <text class="header-kicker">织里童装</text>
        <text class="header-title">拿货地图</text>
        <text class="header-subtitle">{{ currentSceneName }}</text>
      </view>
      <button class="refresh-button" :disabled="loading" @click="loadScenes">刷新</button>
    </view>

    <view class="search-panel">
      <view class="search-bar">
        <input v-model="keyword" class="search-input" placeholder="搜索档口、配套、路段" confirm-type="search" @confirm="submitSearch" />
        <button class="search-button" :disabled="objectLoading" @click="submitSearch">搜索</button>
      </view>
      <view v-if="keyword" class="search-reset-row">
        <text>当前筛选：{{ keyword }}</text>
        <button @click="clearSearch">清除</button>
      </view>
      <scroll-view v-if="sceneTabsVisible" class="scene-tabs" scroll-x>
        <button
          v-for="scene in scenes"
          :key="scene.code"
          :class="['scene-tab', { active: scene.code === selectedSceneCode }]"
          @click="selectScene(scene)"
        >
          {{ scene.name }}
        </button>
      </scroll-view>
    </view>

    <view v-if="loading" class="state-card">
      <text class="state-title">地图加载中</text>
      <text class="state-desc">正在读取已发布的拿货地图。</text>
    </view>

    <view v-else-if="sceneUnavailable" class="state-card">
      <text class="state-title">地图暂未开放</text>
      <text class="state-desc">{{ sceneErrorText }}</text>
      <button class="primary-button" @click="loadScenes">重新加载</button>
    </view>

    <view v-else class="map-content">
      <view class="map-card">
        <view class="map-card-head">
          <text>{{ selectedSceneName }}</text>
          <text>{{ mapObjects.length }} 个点位</text>
        </view>
        <scroll-view class="map-scroll" scroll-x scroll-y>
          <view class="map-stage" :style="stageStyle">
            <image class="map-background" :src="selectedSceneBackground" mode="aspectFill" />
            <button
              v-for="object in mapObjects"
              :key="object.id || object.code"
              :class="['map-object', object.layer === 'booth' ? 'booth' : 'poi', { active: selectedObjectId === object.id }]"
              :style="objectStyle(object)"
              @click="selectMapObject(object)"
            >
              <text>{{ objectDisplayLabel(object) }}</text>
            </button>
          </view>
        </scroll-view>
      </view>

      <view class="result-panel">
        <view class="result-head">
          <text>点位列表</text>
          <text>{{ objectLoading ? '加载中' : `${mapObjects.length} 个` }}</text>
        </view>
        <view v-if="!mapObjects.length" class="empty-list">
          <text class="empty-title">暂无匹配点位</text>
          <text class="empty-desc">可以换个关键词，或切换其他地图场景。</text>
        </view>
        <button
          v-for="object in mapObjects"
          :key="`${object.id || object.code}-row`"
          :class="['object-row', { active: selectedObjectId === object.id }]"
          @click="selectMapObject(object)"
        >
          <view>
            <text class="object-name">{{ object.name || object.code }}</text>
            <text class="object-meta">{{ objectTypeText(object) }} · {{ object.address || '地址待完善' }}</text>
          </view>
          <text class="object-code">{{ object.code }}</text>
        </button>
      </view>

      <view v-if="selectedObject" class="detail-card">
        <view class="detail-head">
          <view>
            <text class="detail-title">{{ selectedObjectName }}</text>
            <text class="detail-meta">{{ selectedObjectMeta }}</text>
          </view>
          <button class="close-button" @click="clearSelectedObject">收起</button>
        </view>
        <view class="detail-grid">
          <view>
            <text class="detail-label">地址</text>
            <text class="detail-value">{{ selectedObjectAddress }}</text>
          </view>
          <view>
            <text class="detail-label">标签</text>
            <text class="detail-value">{{ selectedObjectTags }}</text>
          </view>
        </view>
        <view class="contact-actions">
          <button class="primary-button" @click="callSelectedObject">拨打电话</button>
          <button class="secondary-button" @click="copySelectedWechat">复制微信</button>
        </view>
        <view v-if="nearbyPois.length" class="nearby-section">
          <text class="nearby-title">附近配套</text>
          <view v-for="poi in nearbyPois" :key="poi.id" class="nearby-row">
            <text>{{ poi.name }}</text>
            <text>{{ poi.distanceText || '附近' }}</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import {
  getMapScene,
  listMapObjects,
  listMapScenes,
  listNearbyPois,
  searchMapObjects,
} from '../../api/sourcingMap'

const MAP_MAX_WIDTH_RPX = 690
const DEFAULT_SCENE_NAME = '织里童装拿货地图'

const loading = ref(false)
const objectLoading = ref(false)
const scenes = ref([])
const selectedScene = ref(null)
const selectedSceneCode = ref('')
const routeSceneCode = ref('')
const keyword = ref('')
const mapObjects = ref([])
const selectedObject = ref(null)
const selectedObjectId = ref('')
const nearbyPois = ref([])
const sceneErrorText = ref('地图数据发布后可在这里查看档口和配套点位。')

const currentSceneName = computed(() => selectedScene.value ? selectedScene.value.name : DEFAULT_SCENE_NAME)
const selectedSceneName = computed(() => selectedScene.value ? selectedScene.value.name : DEFAULT_SCENE_NAME)
const selectedSceneBackground = computed(() => selectedScene.value ? selectedScene.value.backgroundUrl : '')
const sceneTabsVisible = computed(() => scenes.value.length > 1)
const sceneUnavailable = computed(() => !selectedScene.value || !selectedSceneBackground.value)
const stageScale = computed(() => {
  const width = toPositiveNumber(selectedScene.value?.width, MAP_MAX_WIDTH_RPX)
  return Math.min(1, MAP_MAX_WIDTH_RPX / width)
})
const stageStyle = computed(() => {
  const width = toPositiveNumber(selectedScene.value?.width, MAP_MAX_WIDTH_RPX)
  const height = toPositiveNumber(selectedScene.value?.height, 420)
  return `width: ${Math.round(width * stageScale.value)}rpx; height: ${Math.round(height * stageScale.value)}rpx;`
})
const selectedObjectName = computed(() => selectedObject.value?.name || selectedObject.value?.code || '点位详情')
const selectedObjectMeta = computed(() => selectedObject.value ? `${objectTypeText(selectedObject.value)} · ${selectedObject.value.code || '无编号'}` : '')
const selectedObjectAddress = computed(() => selectedObject.value?.address || '地址待完善')
const selectedObjectTags = computed(() => {
  if (!selectedObject.value) return '暂无标签'
  const tags = [
    ...(selectedObject.value.categoryCodes || []),
    ...(selectedObject.value.serviceTags || []),
    ...(selectedObject.value.poiServiceTags || []),
  ]
  return tags.length ? tags.join('、') : '暂无标签'
})

onLoad((options = {}) => {
  routeSceneCode.value = decodeRouteValue(options.sceneCode || '')
  keyword.value = decodeRouteValue(options.keyword || options.q || '')
  loadScenes()
})

onPullDownRefresh(async () => {
  // 下拉刷新保留当前场景和关键词，让买手核对档口时不会被重置到默认地图。
  await loadScenes({ keepSelection: true })
  uni.stopPullDownRefresh()
})

async function loadScenes(options = {}) {
  loading.value = true
  sceneErrorText.value = '地图数据发布后可在这里查看档口和配套点位。'
  try {
    const resp = await listMapScenes({ cityCode: DEFAULT_CITY_CODE })
    scenes.value = resp.items || []
    if (!scenes.value.length) {
      selectedScene.value = null
      selectedSceneCode.value = ''
      mapObjects.value = []
      sceneErrorText.value = '地图暂未开放，请稍后再试。'
      return
    }
    const preferredCode = options.keepSelection ? selectedSceneCode.value : routeSceneCode.value
    const nextScene = scenes.value.find((scene) => scene.code === preferredCode) || scenes.value[0]
    await selectScene(nextScene)
  } catch {
    selectedScene.value = null
    selectedSceneCode.value = ''
    mapObjects.value = []
    sceneErrorText.value = '地图加载失败，请检查网络后重试。'
  } finally {
    loading.value = false
  }
}

async function selectScene(scene) {
  if (!scene || !scene.code) return
  selectedSceneCode.value = scene.code
  selectedObject.value = null
  selectedObjectId.value = ''
  nearbyPois.value = []
  try {
    const resp = await getMapScene(scene.code, { suppressErrorToast: true })
    selectedScene.value = resp.item || scene
  } catch {
    selectedScene.value = scene
  }
  await loadSceneObjects()
}

async function loadSceneObjects() {
  if (!selectedSceneCode.value) {
    mapObjects.value = []
    return
  }
  objectLoading.value = true
  try {
    const term = keyword.value.trim()
    const resp = term
      ? await searchMapObjects({ sceneCode: selectedSceneCode.value, keyword: term, limit: 50 })
      : await listMapObjects(selectedSceneCode.value)
    mapObjects.value = resp.items || []
  } catch {
    mapObjects.value = []
    uni.showToast({ title: '地图点位加载失败，请稍后重试', icon: 'none' })
  } finally {
    objectLoading.value = false
  }
}

async function submitSearch() {
  await loadSceneObjects()
}

async function clearSearch() {
  keyword.value = ''
  await loadSceneObjects()
}

function selectMapObject(object) {
  selectedObject.value = object
  selectedObjectId.value = object.id || ''
  loadNearbyPois(object)
}

function clearSelectedObject() {
  selectedObject.value = null
  selectedObjectId.value = ''
  nearbyPois.value = []
}

async function loadNearbyPois(object) {
  nearbyPois.value = []
  if (!object?.id) return
  try {
    const resp = await listNearbyPois(object.id, { limit: 4 })
    nearbyPois.value = resp.items || []
  } catch {
    nearbyPois.value = []
  }
}

function callSelectedObject() {
  const phone = selectedObject.value?.phone || ''
  if (!phone) {
    uni.showToast({ title: '该点位暂未提供电话', icon: 'none' })
    return
  }
  uni.makePhoneCall({ phoneNumber: phone })
}

function copySelectedWechat() {
  const wechat = selectedObject.value?.wechat || ''
  if (!wechat) {
    uni.showToast({ title: '该点位暂未提供微信', icon: 'none' })
    return
  }
  uni.setClipboardData({ data: wechat })
}

function objectStyle(object) {
  const geometry = object.geometry || {}
  const scale = stageScale.value
  const x = toNumber(geometry.x, toNumber(object.centerX, 0)) * scale
  const y = toNumber(geometry.y, toNumber(object.centerY, 0)) * scale
  if (object.geometryType === 'point') {
    return `left: ${Math.max(0, Math.round(x - 14))}rpx; top: ${Math.max(0, Math.round(y - 14))}rpx; width: 28rpx; height: 28rpx;`
  }
  const width = toPositiveNumber(geometry.width, 80) * scale
  const height = toPositiveNumber(geometry.height, 50) * scale
  return `left: ${Math.round(x)}rpx; top: ${Math.round(y)}rpx; width: ${Math.round(width)}rpx; height: ${Math.round(height)}rpx;`
}

function objectDisplayLabel(object) {
  if (object.geometryType === 'point') return ''
  return object.code || object.name || ''
}

function objectTypeText(object) {
  const typeMap = {
    booth: '档口',
    packing_station: '打包站',
    logistics_point: '物流点',
    express_point: '快递点',
    parking: '停车场',
    restaurant: '餐饮',
  }
  return typeMap[object.type] || (object.layer === 'poi' ? '配套' : '点位')
}

function decodeRouteValue(value) {
  try {
    return decodeURIComponent(value || '')
  } catch {
    return value || ''
  }
}

function toNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPositiveNumber(value, fallback) {
  const parsed = toNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}
</script>

<style lang="scss" scoped>
.sourcing-map-page {
  min-height: 100vh;
  padding: 28rpx;
  background: $wplink-bg;
}

.map-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 24rpx;
}

.header-copy {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.header-kicker {
  color: $wplink-warning;
  font-size: 22rpx;
  font-weight: 800;
}

.header-title {
  color: $wplink-primary;
  font-size: 42rpx;
  font-weight: 900;
  line-height: 1.15;
}

.header-subtitle {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.35;
}

.refresh-button,
.search-button,
.primary-button,
.secondary-button,
.close-button {
  min-width: 0;
  margin: 0;
  border: 0;
  border-radius: 16rpx;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 1.25;
}

.refresh-button::after,
.search-button::after,
.primary-button::after,
.secondary-button::after,
.close-button::after,
.scene-tab::after,
.map-object::after,
.object-row::after {
  border: 0;
}

.refresh-button {
  flex: 0 0 auto;
  padding: 16rpx 22rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.search-panel,
.map-card,
.result-panel,
.detail-card,
.state-card {
  border-radius: 20rpx;
  background: $wplink-card;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.search-panel {
  display: grid;
  gap: 18rpx;
  margin-bottom: 24rpx;
  padding: 22rpx;
}

.search-bar {
  display: grid;
  grid-template-columns: 1fr 112rpx;
  gap: 14rpx;
  align-items: center;
}

.search-input {
  min-height: 72rpx;
  padding: 0 24rpx;
  border-radius: 16rpx;
  background: #f4f7fb;
  color: $wplink-text;
  font-size: 26rpx;
}

.search-button,
.primary-button {
  padding: 20rpx 18rpx;
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button,
.close-button {
  padding: 18rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.search-reset-row,
.map-card-head,
.result-head,
.nearby-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.search-reset-row {
  color: $wplink-muted;
  font-size: 24rpx;
}

.search-reset-row button {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 1.2;
}

.search-reset-row button::after {
  border: 0;
}

.scene-tabs {
  white-space: nowrap;
}

.scene-tab {
  display: inline-flex;
  min-height: 58rpx;
  margin: 0 12rpx 0 0;
  padding: 0 22rpx;
  border: 0;
  border-radius: 999rpx;
  background: #f4f7fb;
  color: $wplink-muted;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 58rpx;
}

.scene-tab.active {
  background: $wplink-warning-soft;
  color: #9a5b00;
}

.state-card {
  display: grid;
  gap: 14rpx;
  padding: 40rpx 32rpx;
}

.state-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 800;
}

.state-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.map-content {
  display: grid;
  gap: 24rpx;
}

.map-card {
  overflow: hidden;
}

.map-card-head {
  padding: 22rpx 24rpx;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 800;
}

.map-card-head text:last-child,
.result-head text:last-child {
  color: $wplink-muted;
  font-size: 24rpx;
}

.map-scroll {
  max-height: 720rpx;
  background: #eef3f8;
}

.map-stage {
  position: relative;
  min-width: 320rpx;
  min-height: 240rpx;
  overflow: hidden;
  background: #e8eef5;
}

.map-background {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.map-object {
  position: absolute;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  margin: 0;
  padding: 0 4rpx;
  border: 3rpx solid $wplink-primary;
  border-radius: 6rpx;
  background: rgba($wplink-primary, 0.18);
  color: $wplink-primary;
  font-size: 18rpx;
  font-weight: 900;
  line-height: 1.1;
}

.map-object.poi {
  padding: 0;
  border-color: $wplink-warning;
  border-radius: 999rpx;
  background: $wplink-warning;
}

.map-object.active {
  border-color: $wplink-success;
  box-shadow: 0 0 0 5rpx rgba($wplink-success, 0.18);
}

.result-panel,
.detail-card {
  padding: 24rpx;
}

.result-head {
  margin-bottom: 16rpx;
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 900;
}

.empty-list {
  display: grid;
  gap: 8rpx;
  padding: 24rpx 0;
}

.empty-title {
  color: $wplink-text;
  font-size: 28rpx;
  font-weight: 800;
}

.empty-desc {
  color: $wplink-muted;
  font-size: 24rpx;
}

.object-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  width: 100%;
  margin: 0;
  padding: 20rpx 0;
  border: 0;
  border-top: 1rpx solid #edf1f6;
  border-radius: 0;
  background: transparent;
  text-align: left;
}

.object-row.active {
  color: $wplink-success;
}

.object-name,
.detail-title {
  display: block;
  color: $wplink-text;
  font-size: 28rpx;
  font-weight: 900;
  line-height: 1.3;
}

.object-meta,
.detail-meta,
.object-code {
  display: block;
  margin-top: 6rpx;
  color: $wplink-muted;
  font-size: 23rpx;
  line-height: 1.35;
}

.object-code {
  flex: 0 0 auto;
  margin-top: 0;
  font-weight: 800;
}

.detail-card {
  display: grid;
  gap: 22rpx;
}

.detail-head {
  display: flex;
  justify-content: space-between;
  gap: 18rpx;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18rpx;
}

.detail-label {
  display: block;
  margin-bottom: 8rpx;
  color: $wplink-muted;
  font-size: 22rpx;
}

.detail-value {
  display: block;
  color: $wplink-text;
  font-size: 25rpx;
  font-weight: 700;
  line-height: 1.45;
  word-break: break-word;
}

.contact-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.nearby-section {
  display: grid;
  gap: 12rpx;
  padding-top: 18rpx;
  border-top: 1rpx solid #edf1f6;
}

.nearby-title {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 900;
}

.nearby-row {
  color: $wplink-muted;
  font-size: 24rpx;
}
</style>
