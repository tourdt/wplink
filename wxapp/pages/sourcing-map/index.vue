<template>
  <view class="sourcing-map-page">
    <view class="map-header">
      <view class="header-copy">
        <text class="header-kicker">织里童装</text>
        <text class="header-title">拿货地图</text>
        <text class="header-subtitle">{{ currentSceneName }}</text>
      </view>
      <button class="refresh-button" :disabled="loading" @click="refreshMapData">刷新</button>
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
      <view class="filter-panel">
        <view v-for="group in filterGroups" :key="group.key" class="filter-group">
          <text class="filter-title">{{ group.label }}</text>
          <scroll-view class="filter-options" scroll-x>
            <button
              v-for="item in group.items"
              :key="`${group.key}-${item.value}`"
              :class="['filter-chip', { active: isFilterActive(group.key, item.value) }]"
              @click="toggleFilter(group.key, item.value)"
            >
              {{ item.label }}
            </button>
          </scroll-view>
        </view>
        <view v-if="hasActiveFilters" class="filter-reset-row">
          <text>已选 {{ activeFilterCount }} 项筛选</text>
          <button @click="clearFilters">全部清除</button>
        </view>
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
        <view class="zoom-toolbar">
          <button class="zoom-button" @click="zoomOutMap">缩小</button>
          <text class="zoom-percent">{{ mapZoomPercent }}</text>
          <button class="zoom-button" @click="zoomInMap">放大</button>
          <button class="zoom-button" @click="resetMapZoom">复位</button>
        </view>
        <scroll-view class="map-scroll" scroll-x scroll-y scroll-with-animation :scroll-left="mapScrollLeft" :scroll-top="mapScrollTop">
          <view class="map-stage" :style="stageStyle">
            <image class="map-background" :src="selectedSceneBackground" mode="aspectFill" />
            <button
              v-for="object in mapObjects"
              :key="object.id || object.code"
              :class="['map-object', object.layer === 'booth' ? 'booth' : 'poi', { active: selectedObjectId === objectIdentity(object) }]"
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
          <view v-if="keyword || hasActiveFilters" class="empty-actions">
            <button v-if="keyword" class="secondary-button" @click="clearSearch">清除搜索</button>
            <button v-if="hasActiveFilters" class="secondary-button" @click="clearFilters">清除筛选</button>
          </view>
        </view>
        <button
          v-for="object in mapObjects"
          :key="`${object.id || object.code}-row`"
          :class="['object-row', { active: selectedObjectId === objectIdentity(object) }]"
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
        <view v-if="detailTags.length" class="detail-tag-list">
          <text v-for="tag in detailTags" :key="tag" class="detail-tag">{{ tag }}</text>
        </view>
        <view class="detail-grid">
          <view v-for="field in detailFields" :key="field.label">
            <text class="detail-label">{{ field.label }}</text>
            <text class="detail-value">{{ field.value }}</text>
          </view>
        </view>
        <view class="contact-actions">
          <button class="primary-button" @click="openSelectedObjectLocation">导航</button>
          <button class="secondary-button" @click="callSelectedObject">拨打电话</button>
          <button class="secondary-button" @click="copySelectedWechat">复制微信</button>
        </view>
        <view v-if="nearbyPois.length" class="nearby-section">
          <text class="nearby-title">附近配套</text>
          <button v-for="poi in nearbyPois" :key="poi.id" class="nearby-row" @click="selectNearbyPoi(poi)">
            <text>{{ poi.name }}</text>
            <text>{{ poi.distanceText || '附近' }}</text>
          </button>
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
  getMapObject,
  getMapScene,
  listMapCategories,
  listMapObjects,
  listMapScenes,
  listNearbyPois,
  searchMapObjects,
} from '../../api/sourcingMap'

const MAP_MAX_WIDTH_RPX = 690
const MAP_VIEWPORT_HEIGHT_RPX = 720
const MAP_MIN_SCALE = 1
const MAP_MAX_SCALE = 3
const MAP_SCALE_STEP = 0.35
const DEFAULT_SCENE_NAME = '织里童装拿货地图'
const defaultLabelDictionary = {
  girl: '女童',
  boy: '男童',
  baby: '婴童',
  middle_child: '中大童',
  school_uniform: '校服',
  down_jacket: '羽绒',
  sweater: '毛衫',
  dress: '裙装',
  suit: '套装',
  spot: '现货',
  factory: '源头工厂',
  sample: '支持打样',
  drop_shipping: '一件代发',
  mixed_batch: '支持混批',
  verified: '实地认证',
  recommended: '平台推荐',
  packing: '打包',
  labeling: '贴单',
  carton: '纸箱',
  tape: '胶带',
  storage: '临时寄存',
  national: '全国物流',
  cod: '到付',
  less_than_truckload: '零担',
  full_truckload: '整车',
  zto: '中通',
  yto: '圆通',
  sto: '申通',
  yunda: '韵达',
  jtexpress: '极兔',
  sf: '顺丰',
  bulk_shipping: '批量发货',
}
const defaultFilterGroups = [
  {
    key: 'categories',
    label: '档口分类',
    type: 'booth_category',
    items: [
      { label: '女童', value: 'girl' },
      { label: '男童', value: 'boy' },
      { label: '婴童', value: 'baby' },
      { label: '中大童', value: 'middle_child' },
    ],
  },
  {
    key: 'serviceTags',
    label: '档口服务',
    type: 'booth_service',
    items: [
      { label: '现货', value: 'spot' },
      { label: '源头工厂', value: 'factory' },
      { label: '支持打样', value: 'sample' },
      { label: '一件代发', value: 'drop_shipping' },
    ],
  },
  {
    key: 'types',
    label: '配套类型',
    type: 'poi_type',
    items: [
      { label: '打包站', value: 'packing_station' },
      { label: '物流点', value: 'logistics_point' },
      { label: '快递点', value: 'express_point' },
      { label: '停车场', value: 'parking' },
    ],
  },
  {
    key: 'poiServiceTags',
    label: '配套服务',
    type: 'poi_service',
    items: [
      { label: '打包', value: 'packing' },
      { label: '贴单', value: 'labeling' },
      { label: '纸箱', value: 'carton' },
      { label: '全国物流', value: 'national' },
    ],
  },
]

const loading = ref(false)
const objectLoading = ref(false)
const scenes = ref([])
const selectedScene = ref(null)
const selectedSceneCode = ref('')
const routeSceneCode = ref('')
const keyword = ref('')
const mapObjects = ref([])
const mapCategories = ref([])
const selectedObject = ref(null)
const selectedObjectId = ref('')
const nearbyPois = ref([])
const mapScale = ref(1)
const mapScrollLeft = ref(0)
const mapScrollTop = ref(0)
const categoryLabels = ref({ ...defaultLabelDictionary })
const activeFilters = ref(defaultActiveFilters())
const sceneErrorText = ref('地图数据发布后可在这里查看档口和配套点位。')

const currentSceneName = computed(() => selectedScene.value ? selectedScene.value.name : DEFAULT_SCENE_NAME)
const selectedSceneName = computed(() => selectedScene.value ? selectedScene.value.name : DEFAULT_SCENE_NAME)
const selectedSceneBackground = computed(() => selectedScene.value ? selectedScene.value.backgroundUrl : '')
const sceneTabsVisible = computed(() => scenes.value.length > 1)
const sceneUnavailable = computed(() => !selectedScene.value || !selectedSceneBackground.value)
const hasActiveFilters = computed(() => activeFilterCount.value > 0)
const activeFilterCount = computed(() => Object.values(activeFilters.value).reduce((total, values) => total + values.length, 0))
const filterGroups = computed(() => buildFilterGroups(mapCategories.value))
const stageScale = computed(() => {
  const width = toPositiveNumber(selectedScene.value?.width, MAP_MAX_WIDTH_RPX)
  return Math.min(1, MAP_MAX_WIDTH_RPX / width)
})
const effectiveStageScale = computed(() => stageScale.value * mapScale.value)
const mapZoomLevel = computed(() => getZoomLevelByScale(mapScale.value))
const mapZoomPercent = computed(() => `${Math.round(mapScale.value * 100)}%`)
const stageStyle = computed(() => {
  const width = toPositiveNumber(selectedScene.value?.width, MAP_MAX_WIDTH_RPX)
  const height = toPositiveNumber(selectedScene.value?.height, 420)
  return `width: ${Math.round(width * effectiveStageScale.value)}rpx; height: ${Math.round(height * effectiveStageScale.value)}rpx;`
})
const selectedObjectName = computed(() => selectedObject.value?.name || selectedObject.value?.code || '点位详情')
const selectedObjectMeta = computed(() => selectedObject.value ? `${objectTypeText(selectedObject.value)} · ${selectedObject.value.code || '无编号'}` : '')
const selectedObjectAddress = computed(() => selectedObject.value?.address || '地址待完善')
const detailTags = computed(() => {
  if (!selectedObject.value) return []
  const tags = [
    ...(selectedObject.value.categoryCodes || []),
    ...(selectedObject.value.serviceTags || []),
    ...(selectedObject.value.platformTags || []),
    ...(selectedObject.value.poiServiceTags || []),
  ]
  return formatLabelList(tags)
})
const detailFields = computed(() => {
  if (!selectedObject.value) return []
  const extra = selectedObject.value.extra || {}
  return [
    { label: '地址', value: selectedObjectAddress.value },
    { label: '营业时间', value: formatExtraValue(extra.openHours) },
    { label: '支持服务', value: formatExtraValue(extra.services) },
    { label: '物流线路', value: formatExtraValue(extra.lines) },
    { label: '发货方式', value: formatExtraValue(extra.deliveryTypes) },
    { label: '发车时间', value: formatExtraValue(extra.departureTime) },
    { label: '快递品牌', value: formatExtraValue(extra.brands) },
    { label: '收费说明', value: formatExtraValue(extra.priceNote) },
    { label: '联系电话', value: selectedObject.value.phone || '' },
    { label: '微信', value: selectedObject.value.wechat || '' },
  ].filter((field) => field.value)
})

onLoad((options = {}) => {
  routeSceneCode.value = decodeRouteValue(options.sceneCode || '')
  keyword.value = decodeRouteValue(options.keyword || options.q || '')
  loadMapCategories()
  loadScenes()
})

onPullDownRefresh(async () => {
  // 下拉刷新保留当前场景和关键词，让买手核对档口时不会被重置到默认地图。
  await refreshMapData({ keepSelection: true })
  uni.stopPullDownRefresh()
})

async function refreshMapData(options = {}) {
  await loadMapCategories()
  await loadScenes(options)
}

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

async function loadMapCategories() {
  try {
    const resp = await listMapCategories()
    mapCategories.value = (resp.items || []).filter(isVisibleNormalCategory)
    categoryLabels.value = {
      ...defaultLabelDictionary,
      ...Object.fromEntries(mapCategories.value.map((item) => [item.code, item.name])),
    }
  } catch {
    // 分类接口只影响筛选项和标签文案，失败时保留默认字典，不阻断买手查看地图。
    mapCategories.value = []
    categoryLabels.value = { ...defaultLabelDictionary }
  }
}

function buildFilterGroups(categories) {
  return defaultFilterGroups
    .map((group) => ({
      ...group,
      items: mergeCategoryOptions(group.items, categoryOptionsByType(categories, group.type)),
    }))
    .filter((group) => group.items.length)
}

function categoryOptionsByType(categories, type) {
  return (categories || [])
    .filter((item) => item.type === type && isVisibleNormalCategory(item))
    .sort((left, right) => toNumber(left.sort, 0) - toNumber(right.sort, 0))
    .map((item) => ({ label: item.name, value: item.code }))
}

function mergeCategoryOptions(defaultOptions, configuredOptions) {
  const seen = new Set()
  return [...configuredOptions, ...defaultOptions].filter((item) => {
    if (!item.value || seen.has(item.value)) {
      return false
    }
    seen.add(item.value)
    return true
  })
}

function isVisibleNormalCategory(item) {
  return item?.isVisible !== false && item?.status === 'normal'
}

async function selectScene(scene) {
  if (!scene || !scene.code) return
  selectedSceneCode.value = scene.code
  selectedObject.value = null
  selectedObjectId.value = ''
  nearbyPois.value = []
  mapScale.value = 1
  mapScrollLeft.value = 0
  mapScrollTop.value = 0
  try {
    const resp = await getMapScene(scene.code, { suppressErrorToast: true })
    selectedScene.value = resp.item || scene
    applySceneDefaultViewport(selectedScene.value)
  } catch {
    selectedScene.value = scene
    applySceneDefaultViewport(selectedScene.value)
  }
  await loadSceneObjects()
}

async function loadSceneObjects(options = {}) {
  if (!selectedSceneCode.value) {
    mapObjects.value = []
    return
  }
  objectLoading.value = true
  try {
    const term = keyword.value.trim()
    const resp = term
      ? await searchMapObjects({
          ...buildObjectQueryParams(),
          sceneCode: selectedSceneCode.value,
          keyword: term,
          limit: 50,
        })
      : await listMapObjects(selectedSceneCode.value, buildObjectQueryParams())
    mapObjects.value = applyLocalFilters(resp.items || [])
    if (options.focusFirst) {
      selectFirstObjectAfterSearch()
    } else {
      syncSelectedObjectAfterLoad()
    }
  } catch {
    mapObjects.value = []
    clearSelectedObject()
    uni.showToast({ title: '地图点位加载失败，请稍后重试', icon: 'none' })
  } finally {
    objectLoading.value = false
  }
}

async function submitSearch() {
  await loadSceneObjects({ focusFirst: true })
}

async function clearSearch() {
  keyword.value = ''
  await loadSceneObjects({ focusFirst: hasActiveFilters.value })
}

async function toggleFilter(key, value) {
  const current = activeFilters.value[key] || []
  const exists = current.includes(value)
  activeFilters.value = {
    ...activeFilters.value,
    [key]: exists ? current.filter((item) => item !== value) : [...current, value],
  }
  await loadSceneObjects({ focusFirst: true })
}

async function clearFilters() {
  activeFilters.value = defaultActiveFilters()
  await loadSceneObjects({ focusFirst: Boolean(keyword.value.trim()) })
}

function isFilterActive(key, value) {
  return (activeFilters.value[key] || []).includes(value)
}

function buildObjectQueryParams() {
  const params = {}
  if (activeFilters.value.types.length) params.types = activeFilters.value.types.join(',')
  if (activeFilters.value.categories.length) params.categories = activeFilters.value.categories.join(',')
  if (activeFilters.value.serviceTags.length) params.serviceTags = activeFilters.value.serviceTags.join(',')
  if (activeFilters.value.poiServiceTags.length) params.poiServiceTags = activeFilters.value.poiServiceTags.join(',')
  return params
}

function selectMapObject(object, options = { focus: true }) {
  if (!object) return
  selectedObject.value = object
  selectedObjectId.value = objectIdentity(object)
  if (options.focus) {
    focusMapObject(object)
  }
  loadNearbyPois(object)
}

function clearSelectedObject() {
  selectedObject.value = null
  selectedObjectId.value = ''
  nearbyPois.value = []
}

function selectFirstObjectAfterSearch() {
  if (!mapObjects.value.length) {
    clearSelectedObject()
    return
  }
  selectMapObject(mapObjects.value[0], { focus: true })
}

function syncSelectedObjectAfterLoad() {
  if (!selectedObjectId.value) return
  const latest = mapObjects.value.find((item) => objectIdentity(item) === selectedObjectId.value)
  if (latest) {
    selectedObject.value = latest
    return
  }
  clearSelectedObject()
}

function applyLocalFilters(items) {
  return items.filter((item) => {
    return (
      matchesSelectedValues(item.type ? [item.type] : [], activeFilters.value.types) &&
      matchesSelectedValues(item.categoryCodes || [], activeFilters.value.categories) &&
      matchesSelectedValues(item.serviceTags || [], activeFilters.value.serviceTags) &&
      matchesSelectedValues(item.poiServiceTags || [], activeFilters.value.poiServiceTags)
    )
  })
}

function matchesSelectedValues(values, selected) {
  if (!selected.length) return true
  return selected.some((value) => values.includes(value))
}

function focusMapObject(object) {
  focusMapCenter(calculateObjectCenter(object))
}

function focusMapCenter(center) {
  const scaledX = center.x * effectiveStageScale.value
  const scaledY = center.y * effectiveStageScale.value
  mapScrollLeft.value = Math.max(0, Math.round(rpxToPx(scaledX - MAP_MAX_WIDTH_RPX / 2)))
  mapScrollTop.value = Math.max(0, Math.round(rpxToPx(scaledY - MAP_VIEWPORT_HEIGHT_RPX / 2)))
}

function applySceneDefaultViewport(scene) {
  mapScale.value = normalizeSceneDefaultScale(scene?.defaultScale)
  const centerX = parseOptionalNumber(scene?.defaultCenterX)
  const centerY = parseOptionalNumber(scene?.defaultCenterY)
  if (centerX == null || centerY == null) {
    mapScrollLeft.value = 0
    mapScrollTop.value = 0
    return
  }
  focusMapCenter({ x: centerX, y: centerY })
}

function normalizeSceneDefaultScale(value) {
  const scale = toNumber(value, 1)
  return Math.min(MAP_MAX_SCALE, Math.max(MAP_MIN_SCALE, scale))
}

function zoomInMap() {
  changeMapScale(mapScale.value + MAP_SCALE_STEP)
}

function zoomOutMap() {
  changeMapScale(mapScale.value - MAP_SCALE_STEP)
}

function resetMapZoom() {
  changeMapScale(1)
}

function changeMapScale(nextScale) {
  mapScale.value = Math.min(MAP_MAX_SCALE, Math.max(MAP_MIN_SCALE, Number(nextScale.toFixed(2))))
  if (selectedObject.value) {
    focusMapObject(selectedObject.value)
  }
}

function calculateObjectCenter(object) {
  const geometry = object.geometry || {}
  const x = toNumber(geometry.x, toNumber(object.centerX, 0))
  const y = toNumber(geometry.y, toNumber(object.centerY, 0))
  if (object.geometryType === 'point') {
    return { x, y }
  }
  return {
    x: x + toPositiveNumber(geometry.width, 80) / 2,
    y: y + toPositiveNumber(geometry.height, 50) / 2,
  }
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

async function selectNearbyPoi(poi) {
  if (!poi?.id) return
  const localObject = mapObjects.value.find((item) => objectIdentity(item) === poi.id)
  if (localObject) {
    selectMapObject(localObject, { focus: true })
    return
  }

  try {
    const detail = await getMapObject(poi.id, { suppressErrorToast: true })
    selectMapObject(detail.item || poi, { focus: true })
  } catch {
    selectMapObject(poi, { focus: true })
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

function openSelectedObjectLocation() {
  const payload = buildNavigationPayload(selectedObject.value)
  if (!payload.address && (!payload.latitude || !payload.longitude)) {
    uni.showToast({ title: '该点位暂未提供可导航地址', icon: 'none' })
    return
  }
  if (!payload.latitude || !payload.longitude) {
    uni.setClipboardData({ data: payload.address })
    uni.showToast({ title: '没有精确定位，已复制地址', icon: 'none' })
    return
  }

  uni.openLocation({
    latitude: payload.latitude,
    longitude: payload.longitude,
    name: payload.name,
    address: payload.address,
    scale: 18,
    fail() {
      if (payload.address) {
        uni.setClipboardData({ data: payload.address })
        uni.showToast({ title: '导航打开失败，已复制地址', icon: 'none' })
      }
    },
  })
}

function buildNavigationPayload(object) {
  const latitude = parseCoordinate(object?.lat)
  const longitude = parseCoordinate(object?.lng)
  return {
    latitude,
    longitude,
    name: object?.name || object?.code || selectedSceneName.value,
    address: object?.address || '',
  }
}

function objectStyle(object) {
  const geometry = object.geometry || {}
  const scale = effectiveStageScale.value
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
  if (mapZoomLevel.value < 4) return object.code || ''
  if (mapZoomLevel.value < 5) return object.code || object.name || ''
  return object.name || object.code || ''
}

function objectIdentity(object) {
  return object?.id || object?.code || ''
}

function objectTypeText(object) {
  const typeMap = {
    booth: '档口',
    factory_booth: '源头工厂',
    market_booth: '市场档口',
    warehouse: '仓库',
    sample_room: '样衣间',
    packing_station: '打包站',
    logistics_point: '物流点',
    express_point: '快递点',
    delivery_station: '发货点',
    parking: '停车场',
    restaurant: '餐饮',
    hotel: '酒店',
    toilet: '厕所',
    bank: '银行',
    convenience_store: '便利店',
  }
  return categoryLabels.value[object.type] || typeMap[object.type] || (object.layer === 'poi' ? '配套' : '点位')
}

function formatLabelList(values) {
  return [...new Set((values || []).filter(Boolean).map((value) => categoryLabels.value[value] || value))]
}

function formatExtraValue(value) {
  if (Array.isArray(value)) {
    return formatLabelList(value).join('、')
  }
  if (typeof value === 'boolean') {
    return value ? '支持' : ''
  }
  if (value && typeof value === 'object') {
    return Object.entries(value)
      .filter(([, entryValue]) => entryValue !== '' && entryValue !== false && entryValue != null)
      .map(([key, entryValue]) => `${categoryLabels.value[key] || key}：${formatExtraValue(entryValue)}`)
      .filter(Boolean)
      .join('；')
  }
  return value ? String(value) : ''
}

function getZoomLevelByScale(scale) {
  if (scale < 1.25) return 3
  if (scale < 2) return 4
  return 5
}

function defaultActiveFilters() {
  return {
    types: [],
    categories: [],
    serviceTags: [],
    poiServiceTags: [],
  }
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

function parseCoordinate(value) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

function parseOptionalNumber(value) {
  if (value === '' || value == null) return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function rpxToPx(value) {
  if (typeof uni !== 'undefined' && typeof uni.upx2px === 'function') {
    return uni.upx2px(value)
  }
  return value
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
.zoom-button::after,
.scene-tab::after,
.filter-chip::after,
.map-object::after,
.object-row::after,
.nearby-row::after {
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

.filter-panel {
  display: grid;
  gap: 16rpx;
}

.filter-group {
  display: grid;
  gap: 10rpx;
}

.filter-title {
  color: $wplink-text;
  font-size: 24rpx;
  font-weight: 900;
}

.filter-options {
  white-space: nowrap;
}

.filter-chip {
  display: inline-flex;
  min-height: 56rpx;
  margin: 0 12rpx 0 0;
  padding: 0 20rpx;
  border: 0;
  border-radius: 999rpx;
  background: #f4f7fb;
  color: $wplink-muted;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 56rpx;
}

.filter-chip.active {
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.filter-reset-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  color: $wplink-muted;
  font-size: 24rpx;
}

.filter-reset-row button {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 1.2;
}

.filter-reset-row button::after {
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

.zoom-toolbar {
  display: grid;
  grid-template-columns: 1fr 100rpx 1fr 1fr;
  gap: 12rpx;
  align-items: center;
  padding: 0 24rpx 18rpx;
}

.zoom-button {
  min-width: 0;
  margin: 0;
  padding: 14rpx 10rpx;
  border: 0;
  border-radius: 14rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 23rpx;
  font-weight: 900;
  line-height: 1.2;
}

.zoom-percent {
  color: $wplink-muted;
  font-size: 23rpx;
  font-weight: 900;
  text-align: center;
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

.empty-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14rpx;
  margin-top: 14rpx;
}

.empty-actions .secondary-button {
  margin: 0;
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

.detail-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.detail-tag {
  padding: 8rpx 14rpx;
  border-radius: 999rpx;
  background: $wplink-warning-soft;
  color: #9a5b00;
  font-size: 22rpx;
  font-weight: 800;
  line-height: 1.2;
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
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
  width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.35;
}
</style>
