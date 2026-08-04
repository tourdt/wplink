<!-- 旧 Canvas 地图保留用于后续研究，不注册到用户访问路径。 -->
<template>
  <view class="sourcing-map-page">
    <view class="search-panel">
      <view class="search-shell">
        <input v-model="keyword" class="search-input" placeholder="搜索档口、配套、路段" confirm-type="search" @confirm="submitSearch" />
        <button class="filter-toggle-button" @click="toggleFiltersExpanded">{{ filterToggleText }}</button>
        <button class="search-button" :disabled="objectLoading" :loading="objectLoading" @click="submitSearch">搜索</button>
      </view>

      <scroll-view class="compact-filter-row" scroll-x>
        <button
          v-for="item in quickFilterItems"
          :key="`${item.groupKey}-${item.value}`"
          :class="['filter-chip', 'compact', { active: isFilterActive(item.groupKey, item.value) }]"
          @click="toggleFilter(item.groupKey, item.value)"
        >
          {{ item.label }}
        </button>
        <button class="filter-chip compact more" @click="toggleFiltersExpanded">
          {{ filtersExpanded ? '收起筛选' : '更多筛选' }}
        </button>
      </scroll-view>

      <view v-if="keyword || hasActiveFilters" class="active-filter-summary">
        <text>{{ activeFilterSummary }}</text>
        <button @click="clearMapConditions">清除</button>
      </view>

      <view v-if="filtersExpanded" class="filter-panel">
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
      <button class="primary-button" :disabled="loading" :loading="loading" @click="loadScenes">重新加载</button>
    </view>

    <view v-else class="map-content">
      <view class="map-card">
        <view class="map-overlay-info">
          <text>{{ selectedSceneName }}</text>
          <text>{{ mapObjectCountText }}</text>
        </view>
        <view class="map-service-controls">
          <button class="map-service-button" :disabled="loading || objectLoading" :loading="loading || objectLoading" @click="refreshCurrentMapData">刷新</button>
          <button class="map-service-button" @click="resetMapViewport">归位</button>
        </view>
        <view
          class="map-canvas-shell"
          :style="mapCanvasStyle"
          @touchstart="handleCanvasTouchStart"
          @touchmove.stop.prevent="handleCanvasTouchMove"
          @touchend="handleCanvasTouchEnd"
          @touchcancel="handleCanvasTouchCancel"
        >
          <view :class="['map-layer', { hidden: mapNativeFallbackHidden }]" :style="mapLayerStyle">
            <view class="map-background-layer" :style="mapLayerSurfaceStyle">
              <image
                class="map-background-image"
                :src="selectedSceneBackground"
                :style="mapBackgroundStyle"
                mode="scaleToFill"
              />
            </view>
          </view>
          <canvas
            id="sourcingMapCanvas"
            canvas-id="sourcingMapCanvas"
            :class="['map-canvas', { ready: mapCanvasOverlayReady }]"
            :width="mapCanvasPixelSize.width"
            :height="mapCanvasPixelSize.height"
            :style="mapCanvasSurfaceStyle"
            @tap="handleCanvasTap"
          />
          <view v-if="boundaryHintEdges.length" class="map-boundary-hints">
            <view v-if="boundaryHintEdges.includes('left')" class="map-edge-hint left" />
            <view v-if="boundaryHintEdges.includes('right')" class="map-edge-hint right" />
            <view v-if="boundaryHintEdges.includes('top')" class="map-edge-hint top" />
            <view v-if="boundaryHintEdges.includes('bottom')" class="map-edge-hint bottom" :style="bottomEdgeHintStyle" />
          </view>
          <view v-if="canvasErrorText" class="map-canvas-error">
            <text>{{ canvasErrorText }}</text>
          </view>
        </view>
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
          :key="`${mapObjectIdentity(object)}-row`"
          :class="objectRowClasses(object)"
          @click="selectMapObject(object)"
        >
          <view>
            <view class="object-title-line">
              <text class="object-name">{{ objectDisplayName(object) }}</text>
              <text v-if="object.isVerifiedMerchant" class="verified-row-badge">商家点位</text>
            </view>
            <text class="object-meta">{{ objectDisplaySourceText(object) }} · {{ objectTypeText(object) }} · {{ object.address || '地址待完善' }}</text>
          </view>
          <text class="object-code">{{ object.code }}</text>
        </button>
      </view>

      <view v-if="selectedObject" class="detail-card">
        <view class="detail-head">
          <view>
            <text class="detail-title">{{ selectedObjectName }}</text>
            <text class="detail-meta">{{ selectedObjectMeta }}</text>
            <text v-if="selectedObject.isVerifiedMerchant" class="verified-detail-badge">商家点位</text>
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
        <view v-if="selectedObjectLocationWarning" class="location-warning">
          <text>多人反馈位置可能有误，请导航前确认</text>
        </view>
        <view class="navigation-actions">
          <button class="primary-button" @click="openSelectedObjectLocation">导航</button>
        </view>
        <text class="contact-policy-tip">联系方式仅随有效供需信息展示</text>
        <view class="report-actions">
          <button class="secondary-button" :disabled="reportSubmitting" :loading="reportAction === 'location'" @click="submitSelectedObjectLocationCorrection">位置纠错</button>
          <button class="secondary-button risk" :disabled="reportSubmitting" :loading="reportAction === 'risk'" @click="submitSelectedObjectRiskReport">举报问题</button>
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
import { computed, getCurrentInstance, nextTick, onUnmounted, ref, watch } from 'vue'
import { onLoad, onReady } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { requireLogin } from '../../common/auth'
import {
  getMapObject,
  listMapCategories,
  listMapObjects,
  listMapScenes,
  listNearbyPois,
  searchMapObjects,
  submitMapLocationCorrection,
  submitMapRiskReport,
} from '../../api/sourcingMap'
import { createSourcingMapRenderer } from './canvasRenderer'
import { createInitialTransform, endGesture, moveGesture, screenToMap, startGesture } from './mapGesture'
import { mapObjectCenter } from './mapGeometry'
import { hitTestMapObjects, isObjectInBounds } from './mapHitTest'
import { isRentableMapObject, isVerifiedMapObject, mapObjectIdentity } from './mapObjectState'

const MAP_MAX_WIDTH_RPX = 750
const MAP_VIEWPORT_WIDTH_RPX = MAP_MAX_WIDTH_RPX
const MAP_VIEWPORT_HEIGHT_RPX = 1000
const MAP_MIN_VIEWPORT_HEIGHT_RPX = 760
const MAP_TOP_CHROME_HEIGHT_RPX = 176
const MAP_MIN_SCALE = 1
const MAP_MAX_SCALE = 3
const MAP_EDGE_FEEDBACK_PADDING_PX = 40
const MAP_OBJECT_SEARCH_LIMIT = 1000
const CANVAS_RENDER_FRAME_DELAY_MS = 16
const DEFAULT_SCENE_NAME = '织里童装拿货地图'
const locationCorrectionReasons = [
  { label: '地址不准确', code: 'address_inaccurate' },
  { label: '导航位置错误', code: 'navigation_inaccurate' },
  { label: '档口已搬迁或不存在', code: 'moved_or_closed' },
  { label: '信息已过期', code: 'information_expired' },
]
const riskReportReasons = [
  { label: '冒用商家', code: 'impersonation' },
  { label: '虚假信息', code: 'false_information' },
  { label: '违规内容', code: 'prohibited_content' },
]
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
  dropship: '一件代发',
  mixed_batch: '支持混批',
  verified: '已入驻',
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
      { label: '一件代发', value: 'dropship' },
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
const rawMapObjects = ref([])
const loadedMapObjectTotal = ref(0)
const loadedObjectKeyword = ref('')
const mapCategories = ref([])
const selectedObject = ref(null)
const selectedObjectId = ref('')
const reportAction = ref('')
const reportSubmitting = computed(() => Boolean(reportAction.value))
const nearbyPois = ref([])
const mapTransform = ref({ scale: 1, offsetX: 0, offsetY: 0 })
const mapViewportSize = ref({ width: 375, height: 500 })
const mapViewportHeightRpx = ref(MAP_VIEWPORT_HEIGHT_RPX)
const mapSafeAreaBottomPx = ref(0)
const mapCanvasOverlayReady = ref(false)
const mapNativeFallbackHidden = ref(false)
const canvasShellRect = ref({ left: 0, top: 0 })
const canvasErrorText = ref('')
const boundaryHintEdges = ref([])
const categoryLabels = ref({ ...defaultLabelDictionary })
const activeFilters = ref(defaultActiveFilters())
const filtersExpanded = ref(false)
const sceneErrorText = ref('地图数据发布后可在这里查看档口和配套点位。')
const componentInstance = getCurrentInstance()
let canvasGestureState = null
let suppressNextCanvasTap = false
let mapRenderer = null
let mapCanvasRenderSeq = 0
let canvasRenderTimer = null
let pendingCanvasRenderOptions = null
// 点位加载和搜索/筛选可能并发，只应用最后一次请求，避免旧响应覆盖新结果。
let objectRequestSeq = 0
let visibleObjectRequestSeq = 0

const selectedSceneName = computed(() => selectedScene.value ? selectedScene.value.name : DEFAULT_SCENE_NAME)
const selectedSceneBackground = computed(() => selectedScene.value ? selectedScene.value.backgroundUrl : '')
const sceneTabsVisible = computed(() => scenes.value.length > 1)
const sceneUnavailable = computed(() => !selectedScene.value || !selectedSceneBackground.value)
const hasActiveFilters = computed(() => activeFilterCount.value > 0)
const activeFilterCount = computed(() => Object.values(activeFilters.value).reduce((total, values) => total + values.length, 0))
const filterGroups = computed(() => buildFilterGroups(mapCategories.value))
const quickFilterItems = computed(() =>
  filterGroups.value.flatMap((group) =>
    group.items.map((item) => ({
      ...item,
      groupKey: group.key,
      groupLabel: group.label,
    })),
  ),
)
const filterToggleText = computed(() => (hasActiveFilters.value ? `已选 ${activeFilterCount.value}` : '筛选'))
const activeFilterSummary = computed(() => {
  const parts = []
  const term = keyword.value.trim()
  if (term) parts.push(`搜索：${term}`)
  if (hasActiveFilters.value) parts.push(`筛选 ${activeFilterCount.value} 项`)
  return parts.join(' · ')
})
const filteredMapObjects = computed(() => applyLocalFilters(rawMapObjects.value))
const mapObjectTotal = computed(() => {
  if (!keyword.value.trim() && hasActiveFilters.value) {
    return filteredMapObjects.value.length
  }
  return loadedMapObjectTotal.value
})
const mapObjectCountText = computed(() => {
  const total = mapObjectTotal.value
  if (keyword.value.trim()) {
    return `搜索到 ${total} 个点位`
  }
  if (hasActiveFilters.value) {
    return `匹配 ${total} 个点位`
  }
  return `全图 ${total} 个点位`
})
const mapZoomLevel = computed(() => getZoomLevelByScale(mapTransform.value.scale))
const visibleMapObjects = computed(() => filteredMapObjects.value.filter((object) => isObjectVisibleAtZoom(object, mapZoomLevel.value)))
const mapObjects = computed(() => visibleMapObjects.value)
const renderMapObjects = computed(() => mapObjects.value.filter((object) => isObjectInBounds(object, getVisibleSceneBounds())))
const mapCanvasStyle = computed(() => `height: ${mapViewportHeightRpx.value}rpx;`)
const mapMinScale = computed(() => calculateMapMinScale())
const mapLayerPixelSize = computed(() => buildMapLayerPixelSize())
const mapCanvasPixelSize = computed(() => buildMapCanvasPixelSize())
const mapLayerStyle = computed(() => buildMapLayerStyle())
const mapLayerSurfaceStyle = computed(() => buildMapLayerSurfaceStyle())
const mapCanvasSurfaceStyle = computed(() => buildMapCanvasSurfaceStyle())
const mapBackgroundStyle = computed(() => mapLayerSurfaceStyle.value)
const mapEdgeFeedbackPadding = computed(() => MAP_EDGE_FEEDBACK_PADDING_PX + mapSafeAreaBottomPx.value)
const bottomEdgeHintStyle = computed(() => buildBottomEdgeHintStyle())
const selectedObjectMerchant = computed(() => selectedObject.value?.merchant || null)
const selectedObjectName = computed(() => selectedObject.value ? objectDisplayName(selectedObject.value) : '点位详情')
const selectedObjectMeta = computed(() => selectedObject.value ? `${objectDisplaySourceText(selectedObject.value)} · ${objectTypeText(selectedObject.value)} · ${selectedObject.value.code || '无编号'}` : '')
const selectedObjectAddress = computed(() => selectedObject.value?.address || '地址待完善')
const selectedObjectLocationWarning = computed(() => Boolean(selectedObject.value?.extra?.locationWarning))
const detailTags = computed(() => {
  if (!selectedObject.value) return []
  const tags = [
    ...(selectedObjectMerchant.value?.mainCategories || []),
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
  ].filter((field) => field.value)
})

onReady(() => {
  initCanvasRenderer()
})

onUnmounted(() => {
  clearScheduledMapCanvasRender()
  mapRenderer?.dispose()
  mapRenderer = null
  mapCanvasRenderSeq += 1
  canvasGestureState = null
})

watch(selectedSceneBackground, () => {
  mapCanvasRenderSeq += 1
  mapCanvasOverlayReady.value = false
  mapNativeFallbackHidden.value = false
})

watch([selectedScene, mapObjects, selectedObject, mapViewportSize, mapTransform], () => {
  renderMapCanvas()
})

onLoad((options = {}) => {
  syncMapViewportSize()
  routeSceneCode.value = decodeRouteValue(options.sceneCode || '')
  keyword.value = decodeRouteValue(options.keyword || options.q || '')
  loadMapCategories()
  loadScenes()
})

async function refreshMapData(options = {}) {
  await loadMapCategories()
  await loadScenes(options)
}

async function refreshCurrentMapData() {
  await refreshMapData({ keepSelection: true })
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
      rawMapObjects.value = []
      loadedMapObjectTotal.value = 0
      loadedObjectKeyword.value = ''
      sceneErrorText.value = '地图暂未开放，请稍后再试。'
      return
    }
    const preferredCode = options.keepSelection ? selectedSceneCode.value : routeSceneCode.value
    const nextScene = scenes.value.find((scene) => scene.code === preferredCode) || scenes.value[0]
    await selectScene(nextScene)
  } catch {
    selectedScene.value = null
    selectedSceneCode.value = ''
    rawMapObjects.value = []
    loadedMapObjectTotal.value = 0
    loadedObjectKeyword.value = ''
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
      items: mergeCategoryOptions(group.key, group.items, categoryOptionsByType(categories, group.type)),
    }))
    .filter((group) => group.items.length)
}

function categoryOptionsByType(categories, type) {
  return (categories || [])
    .filter((item) => item.type === type && isVisibleNormalCategory(item))
    .sort((left, right) => toNumber(left.sort, 0) - toNumber(right.sort, 0))
    .map((item) => ({ label: item.name, value: item.code }))
}

function mergeCategoryOptions(groupKey, defaultOptions, configuredOptions) {
  const seen = new Set()
  return [...configuredOptions, ...defaultOptions].filter((item) => {
    const normalizedValue = normalizeFilterOptionValue(groupKey, item.value)
    if (!normalizedValue || seen.has(normalizedValue)) {
      return false
    }
    item.value = normalizedValue
    seen.add(normalizedValue)
    return true
  })
}

function normalizeFilterOptionValue(groupKey, value) {
  return value
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
  resetCanvasGestureState()
  selectedScene.value = scene
  applySceneDefaultViewport(selectedScene.value)
  await loadSceneObjects()
}

async function loadSceneObjects(options = {}) {
  if (!selectedSceneCode.value) {
    rawMapObjects.value = []
    loadedMapObjectTotal.value = 0
    loadedObjectKeyword.value = ''
    return
  }
  const requestId = ++objectRequestSeq
  const showLoading = !options.silent
  if (showLoading) {
    visibleObjectRequestSeq = requestId
    objectLoading.value = true
  }
  try {
    const term = keyword.value.trim()
    const resp = term
      ? await searchMapObjects({
          ...buildObjectQueryParams(),
          sceneCode: selectedSceneCode.value,
          keyword: term,
          limit: MAP_OBJECT_SEARCH_LIMIT,
        })
      : await listMapObjects(selectedSceneCode.value)
    if (requestId !== objectRequestSeq) return
    const items = resp.items || []
    const total = normalizeResponseTotal(resp.total, items.length)
    loadedMapObjectTotal.value = total
    loadedObjectKeyword.value = term
    rawMapObjects.value = items
    if (options.focusFirst) {
      selectFirstObjectAfterSearch()
    } else {
      syncSelectedObjectAfterLoad()
      if (!options.silent) {
        focusSingleObjectAfterLoad()
      }
    }
  } catch {
    if (requestId !== objectRequestSeq) return
    if (options.silent) return
    rawMapObjects.value = []
    loadedMapObjectTotal.value = 0
    loadedObjectKeyword.value = ''
    clearSelectedObject()
    uni.showToast({ title: '地图点位加载失败，请稍后重试', icon: 'none' })
  } finally {
    if (showLoading && visibleObjectRequestSeq === requestId) {
      objectLoading.value = false
    }
  }
}

function normalizeResponseTotal(value, fallback = 0) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    return Math.max(0, Math.floor(toNumber(fallback, 0)))
  }
  return Math.floor(parsed)
}

async function submitSearch() {
  await loadSceneObjects({ focusFirst: true })
}

async function clearSearch() {
  keyword.value = ''
  if (loadedObjectKeyword.value) {
    await loadSceneObjects({ focusFirst: hasActiveFilters.value })
    return
  }
  applyLocalConditionResults({ focusFirst: hasActiveFilters.value })
}

async function toggleFilter(key, value) {
  const normalizedValue = normalizeFilterOptionValue(key, value)
  const current = activeFilters.value[key] || []
  const exists = current.includes(normalizedValue)
  activeFilters.value = {
    ...activeFilters.value,
    [key]: exists ? current.filter((item) => item !== normalizedValue) : [...current, normalizedValue],
  }
  if (shouldReloadObjectsForConditionChange()) {
    await loadSceneObjects({ focusFirst: true })
    return
  }
  applyLocalConditionResults({ focusFirst: true })
}

async function clearFilters() {
  activeFilters.value = defaultActiveFilters()
  if (shouldReloadObjectsForConditionChange()) {
    await loadSceneObjects({ focusFirst: Boolean(keyword.value.trim()) })
    return
  }
  applyLocalConditionResults()
}

function toggleFiltersExpanded() {
  filtersExpanded.value = !filtersExpanded.value
}

async function clearMapConditions() {
  const wasSearchLoaded = Boolean(loadedObjectKeyword.value)
  keyword.value = ''
  activeFilters.value = defaultActiveFilters()
  if (wasSearchLoaded) {
    await loadSceneObjects()
    return
  }
  applyLocalConditionResults()
}

function shouldReloadObjectsForConditionChange() {
  return Boolean(keyword.value.trim())
}

function applyLocalConditionResults(options = {}) {
  if (options.focusFirst) {
    selectFirstObjectAfterSearch()
    return
  }
  syncSelectedObjectAfterLoad()
  if (!options.silent) {
    focusSingleObjectAfterLoad()
  }
  renderMapCanvas()
}

function isFilterActive(key, value) {
  return (activeFilters.value[key] || []).includes(normalizeFilterOptionValue(key, value))
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
  selectedObjectId.value = mapObjectIdentity(object)
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

function isSelectedObjectInViewport(object = selectedObject.value) {
  return Boolean(object && selectedScene.value && isObjectInBounds(object, getVisibleSceneBounds()))
}

function clearSelectedObjectOutsideViewport() {
  if (!selectedObject.value || isSelectedObjectInViewport(selectedObject.value)) return
  clearSelectedObject()
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
  const latest = mapObjects.value.find((item) => mapObjectIdentity(item) === selectedObjectId.value)
  if (latest && isSelectedObjectInViewport(latest)) {
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
      matchesSelectedValues(item.serviceTags || [], activeFilters.value.serviceTags, 'serviceTags') &&
      matchesSelectedValues(item.poiServiceTags || [], activeFilters.value.poiServiceTags)
    )
  })
}

function isObjectVisibleAtZoom(object, zoomLevel) {
  const minZoom = toNumber(object?.minZoom, 1)
  const maxZoom = toNumber(object?.maxZoom, 5)
  return zoomLevel >= minZoom && zoomLevel <= maxZoom
}

function matchesSelectedValues(values, selected, groupKey = '') {
  if (!selected.length) return true
  return selected.some((value) => values.includes(value))
}

function focusMapObject(object) {
  focusMapCenter(calculateObjectCenter(object))
}

function focusSingleObjectAfterLoad() {
  if (selectedObjectId.value || mapObjects.value.length !== 1) return
  focusMapObject(mapObjects.value[0])
}

function focusMapCenter(center) {
  const metrics = getSceneRenderMetrics()
  const nextTransform = {
    ...mapTransform.value,
    offsetX: Math.round(mapViewportSize.value.width / 2 - center.x * metrics.baseScale * mapTransform.value.scale),
    offsetY: Math.round(mapViewportSize.value.height / 2 - center.y * metrics.baseScale * mapTransform.value.scale),
  }
  setMapTransform(nextTransform)
}

function applySceneDefaultViewport(scene) {
  const metrics = getSceneRenderMetrics(scene)
  setMapTransform(
    createInitialTransform({
      viewportWidth: mapViewportSize.value.width,
      viewportHeight: mapViewportSize.value.height,
      mapWidth: metrics.mapWidth,
      mapHeight: metrics.mapHeight,
      minScale: mapMinScale.value,
      maxScale: MAP_MAX_SCALE,
      scale: normalizeSceneDefaultScale(scene?.defaultScale),
    }),
  )
  const centerX = parseOptionalNumber(scene?.defaultCenterX)
  const centerY = parseOptionalNumber(scene?.defaultCenterY)
  if (centerX == null || centerY == null) {
    return
  }
  focusMapCenter({ x: centerX, y: centerY })
}

function normalizeSceneDefaultScale(value) {
  return normalizeMapScale(value)
}

function normalizeMapScale(value) {
  const scale = toNumber(value, 1)
  const clamped = Math.min(MAP_MAX_SCALE, Math.max(mapMinScale.value, scale))
  return Number(clamped.toFixed(2))
}

function resetMapViewport() {
  if (!selectedScene.value) return
  applySceneDefaultViewport(selectedScene.value)
  renderMapCanvas()
}

function handleCanvasTouchStart(event) {
  if (canvasErrorText.value || !selectedScene.value) return
  syncCanvasShellRect()
  canvasGestureState = startGesture(getCanvasTouches(event), mapTransform.value, getGestureOptions())
  boundaryHintEdges.value = []
}

function handleCanvasTouchMove(event) {
  if (!canvasGestureState) return
  const result = moveGesture(canvasGestureState, getCanvasTouches(event), getGestureOptions({ allowOverflow: true }))
  canvasGestureState = result.state
  suppressNextCanvasTap = result.moved
  setMapTransform(result.transform, { allowOverflow: true })
  clearSelectedObjectOutsideViewport()
  boundaryHintEdges.value = buildBoundaryHintEdges(result.transform)
  scheduleMapCanvasRender({ force: true, interacting: true })
}

function handleCanvasTouchEnd() {
  if (!canvasGestureState) return
  const nextTransform = endGesture(canvasGestureState, getGestureOptions())
  const moved = canvasGestureState.moved
  canvasGestureState = null
  boundaryHintEdges.value = []
  clearScheduledMapCanvasRender()
  setMapTransform(nextTransform)
  clearSelectedObjectOutsideViewport()
  renderMapCanvas()
  if (moved) {
    suppressNextCanvasTap = true
    setTimeout(() => {
      suppressNextCanvasTap = false
    }, 120)
  }
}

function handleCanvasTouchCancel() {
  handleCanvasTouchEnd()
}

function handleCanvasTap(event) {
  if (suppressNextCanvasTap || !selectedScene.value) return
  syncCanvasShellRect()
  const point = getCanvasEventPoint(event)
  const scenePoint = screenToScenePoint(point)
  const hitObject = hitTestMapObjects(mapObjects.value, scenePoint, {
    selectedObjectId: selectedObjectId.value,
    pointRadius: getHitTestPointRadius(),
  })
  if (hitObject) {
    selectMapObject(hitObject, { focus: false })
  }
}

function setMapTransform(transform = {}, options = {}) {
  const nextTransform = endGesture({ transform }, getGestureOptions(options))
  mapTransform.value = {
    scale: normalizeMapScale(nextTransform.scale),
    offsetX: Math.round(toNumber(nextTransform.offsetX, 0)),
    offsetY: Math.round(toNumber(nextTransform.offsetY, 0)),
  }
}

function resetCanvasGestureState() {
  canvasGestureState = null
  suppressNextCanvasTap = false
  boundaryHintEdges.value = []
}

function syncMapViewportSize() {
  const info = getWindowInfo()
  mapViewportHeightRpx.value = calculateMapViewportHeightRpx(info)
  mapSafeAreaBottomPx.value = calculateSafeAreaBottomPx(info)
  mapViewportSize.value = {
    width: toPositiveNumber(info?.windowWidth, rpxToPx(MAP_VIEWPORT_WIDTH_RPX)),
    height: rpxToPx(mapViewportHeightRpx.value),
  }
  syncCanvasShellRect()
  if (selectedScene.value) {
    setMapTransform(mapTransform.value)
  }
}

async function initCanvasRenderer() {
  await nextTick()
  syncCanvasShellRect()
  mapRenderer = createSourcingMapRenderer({
    getObjectLabel: objectDisplayLabel,
    onAssetReady: () => {
      renderMapCanvas({ force: true })
    },
  })
  const initialized = mapRenderer.init({
    canvasId: 'sourcingMapCanvas',
    component: componentInstance?.proxy,
    width: mapCanvasPixelSize.value.width,
    height: mapCanvasPixelSize.value.height,
  })
  canvasErrorText.value = initialized ? '' : '地图渲染失败，请刷新后重试'
  if (initialized) {
    renderMapCanvas()
  }
}

function renderMapCanvas(options = {}) {
  if (!mapRenderer || canvasErrorText.value || !selectedScene.value) return
  if (canvasGestureState && !options.force) return
  const metrics = getSceneRenderMetrics()
  mapRenderer.setScene(selectedScene.value)
  mapRenderer.setObjects(renderMapObjects.value)
  mapRenderer.setSelectedObject(selectedObject.value)
  const renderSeq = ++mapCanvasRenderSeq
  const rendered = mapRenderer.render(mapTransform.value, {
    ...options,
    drawBackground: 'whenReady',
    width: mapCanvasPixelSize.value.width,
    height: mapCanvasPixelSize.value.height,
    baseScale: metrics.baseScale,
    zoomLevel: mapZoomLevel.value,
    onDrawComplete({ backgroundDrawn }) {
      if (renderSeq !== mapCanvasRenderSeq) return
      mapCanvasOverlayReady.value = true
      mapNativeFallbackHidden.value = Boolean(backgroundDrawn)
    },
  })
  if (!rendered) {
    mapCanvasOverlayReady.value = false
    mapNativeFallbackHidden.value = false
    canvasErrorText.value = '地图渲染失败，请刷新后重试'
    return
  }
  mapCanvasOverlayReady.value = true
}

function scheduleMapCanvasRender(options = {}) {
  pendingCanvasRenderOptions = mergeCanvasRenderOptions(pendingCanvasRenderOptions, options)
  if (canvasRenderTimer) return
  canvasRenderTimer = setTimeout(() => {
    const renderOptions = pendingCanvasRenderOptions || {}
    pendingCanvasRenderOptions = null
    canvasRenderTimer = null
    renderMapCanvas(renderOptions)
  }, CANVAS_RENDER_FRAME_DELAY_MS)
}

function clearScheduledMapCanvasRender() {
  clearTimeout(canvasRenderTimer)
  canvasRenderTimer = null
  pendingCanvasRenderOptions = null
}

function mergeCanvasRenderOptions(current, next) {
  if (!current) return { ...next }
  return {
    ...current,
    ...next,
    force: Boolean(current.force || next.force),
    interacting: Boolean(current.interacting || next.interacting),
  }
}

function buildMapLayerStyle() {
  const size = mapLayerPixelSize.value
  return [
    `width: ${size.width}px`,
    `height: ${size.height}px`,
    `transform: translate3d(${mapTransform.value.offsetX}px, ${mapTransform.value.offsetY}px, 0) scale(${mapTransform.value.scale})`,
  ].join('; ')
}

function buildMapLayerSurfaceStyle() {
  const size = mapLayerPixelSize.value
  return [
    `width: ${size.width}px`,
    `height: ${size.height}px`,
  ].join('; ')
}

function buildMapCanvasSurfaceStyle() {
  const size = mapCanvasPixelSize.value
  return [
    `width: ${size.width}px`,
    `height: ${size.height}px`,
  ].join('; ')
}

function buildMapLayerPixelSize() {
  const metrics = getSceneRenderMetrics()
  return {
    width: Math.max(1, Math.round(metrics.mapWidth)),
    height: Math.max(1, Math.round(metrics.mapHeight)),
  }
}

function buildMapCanvasPixelSize() {
  return {
    width: Math.max(1, Math.round(mapViewportSize.value.width)),
    height: Math.max(1, Math.round(mapViewportSize.value.height)),
  }
}

function buildBottomEdgeHintStyle() {
  const height = Math.round(rpxToPx(104) + mapSafeAreaBottomPx.value)
  return `height: ${height}px`
}

function buildBoundaryHintEdges(transform) {
  const options = getGestureOptions()
  const edges = []
  const scaledWidth = options.mapWidth * transform.scale
  const scaledHeight = options.mapHeight * transform.scale
  const minX = options.viewportWidth - scaledWidth
  const minY = options.viewportHeight - scaledHeight
  const threshold = 1

  if (scaledWidth <= options.viewportWidth) {
    const centeredX = Math.round((options.viewportWidth - scaledWidth) / 2)
    if (transform.offsetX > centeredX + threshold) edges.push('left')
    if (transform.offsetX < centeredX - threshold) edges.push('right')
  } else {
    if (transform.offsetX > threshold) edges.push('left')
    if (transform.offsetX < minX - threshold) edges.push('right')
  }
  if (scaledHeight <= options.viewportHeight) {
    const centeredY = Math.round((options.viewportHeight - scaledHeight) / 2)
    if (transform.offsetY > centeredY + threshold) edges.push('top')
    if (transform.offsetY < centeredY - threshold) edges.push('bottom')
  } else {
    if (transform.offsetY > threshold) edges.push('top')
    if (transform.offsetY < minY - threshold) edges.push('bottom')
  }
  return edges
}

function getGestureOptions(extraOptions = {}) {
  const metrics = getSceneRenderMetrics()
  return {
    viewportWidth: mapViewportSize.value.width,
    viewportHeight: mapViewportSize.value.height,
    mapWidth: metrics.mapWidth,
    mapHeight: metrics.mapHeight,
    minScale: mapMinScale.value,
    maxScale: MAP_MAX_SCALE,
    maxOverflow: mapEdgeFeedbackPadding.value,
    ...extraOptions,
  }
}

function getSceneRenderMetrics(scene = selectedScene.value) {
  const sceneWidth = toPositiveNumber(scene?.width, MAP_MAX_WIDTH_RPX)
  const sceneHeight = toPositiveNumber(scene?.height, 420)
  const viewportWidth = toPositiveNumber(mapViewportSize.value.width, rpxToPx(MAP_VIEWPORT_WIDTH_RPX))
  const viewportHeight = toPositiveNumber(mapViewportSize.value.height, rpxToPx(mapViewportHeightRpx.value))
  const baseScale = Math.max(viewportWidth / sceneWidth, viewportHeight / sceneHeight)
  return {
    sceneWidth,
    sceneHeight,
    baseScale,
    mapWidth: sceneWidth * baseScale,
    mapHeight: sceneHeight * baseScale,
  }
}

function calculateMapMinScale() {
  const metrics = getSceneRenderMetrics()
  const viewportWidth = toPositiveNumber(mapViewportSize.value.width, rpxToPx(MAP_VIEWPORT_WIDTH_RPX))
  const viewportHeight = toPositiveNumber(mapViewportSize.value.height, rpxToPx(mapViewportHeightRpx.value))
  const fitScale = Math.min(viewportWidth / metrics.mapWidth, viewportHeight / metrics.mapHeight)
  const normalizedScale = Math.min(MAP_MIN_SCALE, toPositiveNumber(fitScale, MAP_MIN_SCALE))
  return Math.max(0.01, Math.floor(normalizedScale * 100) / 100)
}

function getVisibleSceneBounds() {
  const metrics = getSceneRenderMetrics()
  const topLeft = screenToMap({ x: 0, y: 0 }, mapTransform.value)
  const bottomRight = screenToMap(
    { x: mapViewportSize.value.width, y: mapViewportSize.value.height },
    mapTransform.value,
  )
  const minBaseX = Math.min(topLeft.x, bottomRight.x)
  const maxBaseX = Math.max(topLeft.x, bottomRight.x)
  const minBaseY = Math.min(topLeft.y, bottomRight.y)
  const maxBaseY = Math.max(topLeft.y, bottomRight.y)
  return {
    minX: clampNumber(minBaseX / metrics.baseScale, 0, metrics.sceneWidth),
    minY: clampNumber(minBaseY / metrics.baseScale, 0, metrics.sceneHeight),
    maxX: clampNumber(maxBaseX / metrics.baseScale, 0, metrics.sceneWidth),
    maxY: clampNumber(maxBaseY / metrics.baseScale, 0, metrics.sceneHeight),
  }
}

function screenToScenePoint(point) {
  const metrics = getSceneRenderMetrics()
  const mapPoint = screenToMap(point, mapTransform.value)
  return {
    x: mapPoint.x / metrics.baseScale,
    y: mapPoint.y / metrics.baseScale,
  }
}

function getHitTestPointRadius() {
  const metrics = getSceneRenderMetrics()
  return 28 / Math.max(0.1, metrics.baseScale * mapTransform.value.scale)
}

function getCanvasTouches(event) {
  return Array.from(event?.touches || []).map((touch, index) => ({
    identifier: touch.identifier ?? index,
    ...normalizeCanvasPoint({
      x: toNumber(touch.clientX ?? touch.x, 0),
      y: toNumber(touch.clientY ?? touch.y, 0),
    }),
  }))
}

function getCanvasEventPoint(event) {
  const detail = event?.detail || {}
  const touch = event?.changedTouches?.[0] || event?.touches?.[0] || {}
  if (detail.x != null && detail.y != null) {
    return {
      x: toNumber(detail.x, 0),
      y: toNumber(detail.y, 0),
    }
  }
  const touchPoint = {
    x: toNumber(touch.clientX ?? touch.x, 0),
    y: toNumber(touch.clientY ?? touch.y, 0),
  }
  return normalizeCanvasPoint(touchPoint)
}

function normalizeCanvasPoint(point = {}) {
  return {
    clientX: toNumber(point.x, 0) - canvasShellRect.value.left,
    clientY: toNumber(point.y, 0) - canvasShellRect.value.top,
    x: toNumber(point.x, 0) - canvasShellRect.value.left,
    y: toNumber(point.y, 0) - canvasShellRect.value.top,
  }
}

function syncCanvasShellRect() {
  if (typeof uni === 'undefined' || typeof uni.createSelectorQuery !== 'function') return
  const query = uni.createSelectorQuery()
  const scopedQuery = typeof query.in === 'function' ? query.in(componentInstance?.proxy) : query
  scopedQuery
    .select('.map-canvas-shell')
    .boundingClientRect((rect) => {
      if (!rect) return
      canvasShellRect.value = {
        left: toNumber(rect.left, 0),
        top: toNumber(rect.top, 0),
      }
    })
    .exec()
}

function getWindowInfo() {
  if (typeof uni === 'undefined') return null
  if (typeof uni.getWindowInfo === 'function') {
    return uni.getWindowInfo()
  }
  if (typeof uni.getSystemInfoSync === 'function') {
    return uni.getSystemInfoSync()
  }
  return null
}

function calculateMapViewportHeightRpx(info) {
  const windowHeight = toPositiveNumber(info?.windowHeight, 0)
  const windowWidth = toPositiveNumber(info?.windowWidth, 0)
  if (!windowHeight || !windowWidth) {
    return MAP_VIEWPORT_HEIGHT_RPX
  }
  const windowHeightRpx = (windowHeight / windowWidth) * MAP_MAX_WIDTH_RPX
  return Math.max(MAP_MIN_VIEWPORT_HEIGHT_RPX, Math.round(windowHeightRpx - MAP_TOP_CHROME_HEIGHT_RPX))
}

function calculateSafeAreaBottomPx(info) {
  const insetFromWindow = toNumber(info?.safeAreaInsets?.bottom, null)
  if (insetFromWindow != null) {
    return clampNumber(Math.round(insetFromWindow), 0, 80)
  }
  const screenHeight = toPositiveNumber(info?.screenHeight, 0)
  const safeAreaBottom = toPositiveNumber(info?.safeArea?.bottom, 0)
  if (!screenHeight || !safeAreaBottom) {
    return 0
  }
  return clampNumber(Math.round(screenHeight - safeAreaBottom), 0, 80)
}

function calculateObjectCenter(object) {
  const center = mapObjectCenter(object)
  if (center) return center
  const metrics = getSceneRenderMetrics()
  // 几何数据异常时不要聚焦到左上角，回到当前场景中心，避免用户被误导到不存在的点位。
  return { x: metrics.mapWidth / 2, y: metrics.mapHeight / 2 }
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
  const localObject = mapObjects.value.find((item) => mapObjectIdentity(item) === poi.id)
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

function submitSelectedObjectLocationCorrection() {
  openSelectedObjectReport({
    action: 'location',
    reasons: locationCorrectionReasons,
    submit: submitMapLocationCorrection,
    warningKey: 'locationWarning',
  })
}

function submitSelectedObjectRiskReport() {
  openSelectedObjectReport({
    action: 'risk',
    reasons: riskReportReasons,
    submit: submitMapRiskReport,
    warningKey: 'riskWarning',
  })
}

function openSelectedObjectReport({ action, reasons, submit, warningKey }) {
  const objectId = mapObjectIdentity(selectedObject.value)
  if (!objectId || reportSubmitting.value || !requireLogin()) return

  uni.showActionSheet({
    itemList: reasons.map((item) => item.label),
    async success({ tapIndex }) {
      const reason = reasons[tapIndex]
      if (!reason) return
      reportAction.value = action
      try {
        const resp = await submit(objectId, { reasonCode: reason.code })
        if (resp?.item?.warningTriggered) {
          markSelectedObjectWarning(objectId, warningKey, reason.code)
        }
        uni.showToast({
          title: resp?.message || '反馈已记录',
          icon: 'none',
          duration: 2400,
        })
      } finally {
        reportAction.value = ''
      }
    },
  })
}

function markSelectedObjectWarning(objectId, warningKey, reasonCode) {
  const addWarning = (object) => ({
    ...object,
    extra: {
      ...(object.extra || {}),
      [warningKey]: true,
      [`${warningKey}Reason`]: reasonCode,
    },
  })
  const objectIndex = mapObjects.value.findIndex((item) => mapObjectIdentity(item) === objectId)
  if (objectIndex >= 0) {
    mapObjects.value.splice(objectIndex, 1, addWarning(mapObjects.value[objectIndex]))
  }
  // 请求返回时用户可能已经切换点位，只更新本次提交所对应的详情，避免把风险提示标到其他点位。
  if (mapObjectIdentity(selectedObject.value) === objectId) {
    selectedObject.value = addWarning(selectedObject.value)
  }
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

function objectDisplayLabel(object) {
  if (object.geometryType === 'point') return ''
  const selected = selectedObjectId.value === mapObjectIdentity(object)
  const verified = isVerifiedMapObject(object)
  const rentable = isRentableMapObject(object)
  if (!selected && !verified && !rentable) return ''
  if (!selected && verified && mapZoomLevel.value < 5) return ''
  if (!selected && rentable && mapZoomLevel.value < 4) return ''
  if (rentable && !selected) return object.code || object.name || ''
  return objectDisplayName(object)
}

function objectDisplayName(object) {
  return object?.merchant?.name || object?.name || object?.code || ''
}

function objectDisplaySourceText(object) {
  return object?.displaySource === 'verified_merchant' || object?.isVerifiedMerchant ? '商家点位' : '后台点位'
}

function objectRowClasses(object) {
  return [
    'object-row',
    {
      active: selectedObjectId.value === mapObjectIdentity(object),
      verified: object.displayLevel === 'highlight' || object.isVerifiedMerchant,
      weak: object.displayLevel === 'weak',
    },
  ]
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

function pxToRpx(value) {
  if (typeof uni !== 'undefined' && typeof uni.upx2px === 'function') {
    const oneRpx = uni.upx2px(1)
    return oneRpx > 0 ? value / oneRpx : value
  }
  return value
}

function clampNumber(value, min, max) {
  return Math.min(max, Math.max(min, value))
}
</script>

<style lang="scss" scoped>
.sourcing-map-page {
  box-sizing: border-box;
  width: 100vw;
  max-width: 100vw;
  min-height: 100vh;
  overflow-x: hidden;
  padding: 0 0 28rpx;
  background: $wplink-bg;
}

.filter-toggle-button,
.search-button,
.map-service-button,
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

.filter-toggle-button::after,
.search-button::after,
.map-service-button::after,
.primary-button::after,
.secondary-button::after,
.close-button::after,
.scene-tab::after,
.filter-chip::after,
.object-row::after,
.nearby-row::after {
  border: 0;
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
  gap: 12rpx;
  box-sizing: border-box;
  margin: 16rpx 20rpx 12rpx;
  padding: 16rpx;
}

.search-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 92rpx 92rpx;
  gap: 10rpx;
  align-items: center;
}

.search-input {
  min-height: 64rpx;
  padding: 0 20rpx;
  border-radius: 14rpx;
  background: #f4f7fb;
  color: $wplink-text;
  font-size: 25rpx;
}

.filter-toggle-button {
  padding: 18rpx 10rpx;
  background: $wplink-warning-soft;
  color: #9a5b00;
}

.search-button,
.primary-button {
  padding: 18rpx 14rpx;
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button,
.close-button {
  padding: 18rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.active-filter-summary,
.result-head,
.nearby-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.active-filter-summary {
  color: $wplink-muted;
  font-size: 24rpx;
}

.active-filter-summary button {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 1.2;
}

.active-filter-summary button::after {
  border: 0;
}

.compact-filter-row {
  width: 100%;
  min-width: 0;
  white-space: nowrap;
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
  width: 100%;
  min-width: 0;
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

.filter-chip.compact {
  min-height: 48rpx;
  padding: 0 18rpx;
  font-size: 23rpx;
  line-height: 48rpx;
}

.filter-chip.more {
  background: $wplink-warning-soft;
  color: #9a5b00;
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
  width: 100%;
  min-width: 0;
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
  margin: 0 20rpx;
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
  gap: 18rpx;
  width: 100vw;
  max-width: 100vw;
  min-width: 0;
  overflow-x: hidden;
}

.map-card {
  position: relative;
  box-sizing: border-box;
  width: 100vw;
  max-width: 100vw;
  overflow: hidden;
  border-radius: 0;
  box-shadow: none;
}

.result-head text:last-child {
  color: $wplink-muted;
  font-size: 24rpx;
}

.map-overlay-info {
  position: absolute;
  top: 20rpx;
  left: 20rpx;
  z-index: 8;
  display: grid;
  gap: 4rpx;
  max-width: 420rpx;
  padding: 12rpx 16rpx;
  border-radius: 16rpx;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.12);
}

.map-overlay-info text:first-child {
  color: $wplink-primary;
  font-size: 25rpx;
  font-weight: 900;
  line-height: 1.2;
}

.map-overlay-info text:last-child {
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 800;
  line-height: 1.2;
}

.map-service-controls {
  position: absolute;
  top: 20rpx;
  right: 20rpx;
  z-index: 9;
  display: grid;
  gap: 12rpx;
}

.map-service-button {
  min-width: 96rpx;
  padding: 16rpx 18rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.94);
  color: $wplink-primary;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.14);
}

.map-canvas-shell {
  position: relative;
  width: 100vw;
  max-width: 100vw;
  min-height: 760rpx;
  max-height: none;
  overflow: hidden;
  background: #eef3f8;
  touch-action: none;
}

.map-layer {
  position: absolute;
  top: 0;
  left: 0;
  transform-origin: 0 0;
  will-change: transform;
}

.map-layer.hidden {
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

.map-background-layer {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
}

.map-background-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.map-canvas {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: block;
  width: 100%;
  height: 100%;
  background: transparent;
  opacity: 0;
  pointer-events: none;
}

.map-canvas.ready {
  opacity: 1;
  pointer-events: auto;
}

.map-boundary-hints {
  position: absolute;
  inset: 0;
  z-index: 3;
  pointer-events: none;
}

.map-edge-hint {
  position: absolute;
  opacity: 0.9;
}

.map-edge-hint.left,
.map-edge-hint.right {
  top: 0;
  bottom: 0;
  width: 96rpx;
}

.map-edge-hint.top,
.map-edge-hint.bottom {
  right: 0;
  left: 0;
  height: 104rpx;
}

.map-edge-hint.left {
  left: 0;
  background: linear-gradient(90deg, rgba(31, 92, 154, 0.34), rgba(31, 92, 154, 0));
}

.map-edge-hint.right {
  right: 0;
  background: linear-gradient(270deg, rgba(31, 92, 154, 0.34), rgba(31, 92, 154, 0));
}

.map-edge-hint.top {
  top: 0;
  background: linear-gradient(180deg, rgba(31, 92, 154, 0.34), rgba(31, 92, 154, 0));
}

.map-edge-hint.bottom {
  bottom: 0;
  background: linear-gradient(0deg, rgba(31, 92, 154, 0.34), rgba(31, 92, 154, 0));
}

.map-canvas-error {
  position: absolute;
  inset: 0;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 40rpx;
  background: #eef3f8;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 900;
  line-height: 1.4;
  text-align: center;
}

.result-panel,
.detail-card {
  box-sizing: border-box;
  padding: 24rpx;
}

.result-panel {
  min-width: 0;
  margin: 0 20rpx;
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

.object-row.verified {
  padding-right: 16rpx;
  padding-left: 16rpx;
  border-radius: 16rpx;
  background: rgba($wplink-success, 0.08);
}

.object-row.weak {
  opacity: 0.78;
}

.object-title-line {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
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
  position: fixed;
  right: 28rpx;
  bottom: calc(24rpx + env(safe-area-inset-bottom));
  left: 28rpx;
  z-index: 30;
  display: grid;
  gap: 22rpx;
  max-height: 56vh;
  overflow: auto;
}

.detail-head {
  display: flex;
  justify-content: space-between;
  gap: 18rpx;
}

.verified-row-badge,
.verified-detail-badge {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 4rpx 10rpx;
  border-radius: 999rpx;
  background: $wplink-success;
  color: #fff;
  font-size: 20rpx;
  font-weight: 900;
  line-height: 1.2;
}

.verified-detail-badge {
  margin-top: 8rpx;
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

.navigation-actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 16rpx;
}

.contact-policy-tip {
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 1.4;
  text-align: center;
}

.location-warning {
  padding: 16rpx 18rpx;
  border-radius: 14rpx;
  background: $wplink-warning-soft;
  color: #9a5b00;
  font-size: 23rpx;
  font-weight: 800;
  line-height: 1.45;
}

.report-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.report-actions .risk {
  color: #b42318;
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
