import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
const apiSource = readOptionalSource('api/sourcingMap.js')
const source = readOptionalSource('pages/sourcing-map/index.vue')
const rendererSource = readOptionalSource('pages/sourcing-map/canvasRenderer.js')
const gestureSource = readOptionalSource('pages/sourcing-map/mapGesture.js')
const hitTestSource = readOptionalSource('pages/sourcing-map/mapHitTest.js')

function readOptionalSource(file) {
  const fullPath = path.join(root, file)
  return fs.existsSync(fullPath) ? fs.readFileSync(fullPath, 'utf8') : ''
}

function expectTokens(target, tokens) {
  for (const token of tokens) {
    assert.match(target, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
}

function extractFunction(name) {
  const start = source.indexOf(`function ${name}`)
  assert.notEqual(start, -1, `${name} should exist`)
  const next = source.indexOf('\nfunction ', start + 1)
  return source.slice(start, next === -1 ? source.length : next)
}

test('sourcing map page is reachable from wxapp home', () => {
  assert.ok(pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/index'))
  assert.match(homeSource, /拿货地图/)
  assert.match(homeSource, /openSourcingMap/)
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
})

test('sourcing map api uses public map endpoints', () => {
  expectTokens(apiSource, [
    'listMapScenes',
    'getMapScene',
    'listMapObjects',
    'searchMapObjects',
    'getMapObject',
    'listNearbyPois',
    'listMapCategories',
    '/api/v1/map/scenes',
    '/api/v1/map/objects/search',
    '/api/v1/map/categories',
  ])
})

test('sourcing map page loads scenes, renders canvas and shows contact actions', () => {
  expectTokens(source, [
    'onLoad',
    'loadScenes',
    'loadSceneObjects',
    'selectScene',
    'selectMapObject',
    'submitSearch',
    'clearSearch',
    'createSourcingMapRenderer',
    'renderMapCanvas',
    'mapCanvasStyle',
    'selectedObject',
    'callSelectedObject',
    'copySelectedWechat',
    'nearbyPois',
    'loadNearbyPois',
    '地图暂未开放',
    '暂无匹配点位',
  ])
})

test('sourcing map overlay shows total object count text instead of loaded viewport count', () => {
  expectTokens(source, [
    'mapObjectTotal',
    'mapObjectCountText',
    'normalizeResponseTotal',
    '全图',
    '匹配',
    '搜索到',
    '{{ mapObjectCountText }}',
    'resp.total',
  ])
  assert.doesNotMatch(source, /<text>\{\{\s*mapObjects\.length\s*\}\} 个点位<\/text>/)
})

test('sourcing map page disables pull down refresh', () => {
  const page = pagesConfig.pages.find((entry) => entry.path === 'pages/sourcing-map/index')
  assert.ok(page)
  assert.notEqual(page.style?.enablePullDownRefresh, true)
  assert.doesNotMatch(source, /onPullDownRefresh/)
  assert.doesNotMatch(source, /stopPullDownRefresh/)
})

test('sourcing map page supports quick category and poi filters', () => {
  expectTokens(source, [
    'filterGroups',
    'mapCategories',
    'loadMapCategories',
    'buildFilterGroups',
    'mergeCategoryOptions',
    'categoryLabels',
    'activeFilters',
    'toggleFilter',
    'clearFilters',
    'buildObjectQueryParams',
    '档口分类',
    '档口服务',
    '配套服务',
    '女童',
    '现货',
    '打包站',
    '物流点',
    '停车场',
    'booth_category',
    'booth_service',
    'poi_type',
    'poi_service',
  ])
  assert.match(source, /listMapObjects\(selectedSceneCode\.value,\s*buildObjectQueryParams\(\{ includeViewport: true \}\)\)/)
  assert.match(source, /searchMapObjects\(\{\s*\.\.\.buildObjectQueryParams\(\{ includeViewport: false \}\),\s*sceneCode:/)
})

test('sourcing map empty results can clear search and filters', () => {
  expectTokens(source, [
    'empty-actions',
    '清除搜索',
    '清除筛选',
    'keyword',
    'hasActiveFilters',
    '@click="clearSearch"',
    '@click="clearFilters"',
  ])
  assert.match(source, /v-if="keyword"/)
  assert.match(source, /v-if="hasActiveFilters"/)
})

test('sourcing map page keeps search filters compact by default', () => {
  expectTokens(source, [
    'search-shell',
    'compact-filter-row',
    'filter-toggle-button',
    'filtersExpanded',
    'toggleFiltersExpanded',
    'quickFilterItems',
    'activeFilterSummary',
    'clearMapConditions',
    '更多筛选',
    '收起筛选',
  ])
  assert.match(source, /<view v-if="filtersExpanded" class="filter-panel">/)
})

test('sourcing map page uses canvas for map gestures instead of movable dom points', () => {
  expectTokens(source, [
    'map-canvas-shell',
    'map-canvas',
    'canvas-id="sourcingMapCanvas"',
    '@touchstart="handleCanvasTouchStart"',
    '@touchmove.stop.prevent="handleCanvasTouchMove"',
    '@touchend="handleCanvasTouchEnd"',
    '@touchcancel="handleCanvasTouchCancel"',
    '@tap="handleCanvasTap"',
    'createSourcingMapRenderer',
    'startGesture',
    'moveGesture',
    'endGesture',
    'hitTestMapObjects',
  ])
  assert.doesNotMatch(source, /<movable-area/)
  assert.doesNotMatch(source, /<movable-view/)
  assert.doesNotMatch(source, /v-for="entry in polygonObjects"/)
  assert.doesNotMatch(source, /v-for="entry in rectAndPointObjects"/)
})

test('sourcing map renders base map and objects through the same canvas with native image fallback', () => {
  expectTokens(source, [
    'map-background-layer',
    'map-background-image',
    'mapRenderer.setObjects(mapObjects.value)',
    "drawBackground: 'whenReady'",
    'onDrawComplete',
  ])
  assert.match(source, /<view :class="\['map-layer', \{ hidden: mapNativeFallbackHidden \}\]" :style="mapLayerStyle">[\s\S]*class="map-background-image"[\s\S]*<\/view>\s*<canvas/)
  assert.doesNotMatch(source, /v-for="object in mapObjects"[\s\S]*-marker/)
  assert.doesNotMatch(source, /map-dom-object/)
  assert.doesNotMatch(source, /mapDomObjectStyle/)
  assert.doesNotMatch(source, /mapDomObjectClasses/)
  assert.doesNotMatch(source, /polygonDomPoints/)
})

test('sourcing map hides the native fallback after the canvas has drawn the base image and objects', () => {
  expectTokens(source, [
    'map-background-layer',
    'map-background-image',
    ':src="selectedSceneBackground"',
    ':style="mapBackgroundStyle"',
    "drawBackground: 'whenReady'",
    'backgroundDrawn',
    'mapCanvasRenderSeq',
    'mapNativeFallbackHidden',
  ])
  assert.match(source, /:class="\['map-layer', \{ hidden: mapNativeFallbackHidden \}\]"/)
  assert.match(source, /:class="\['map-canvas', \{ ready: mapCanvasOverlayReady \}\]"/)
  assert.doesNotMatch(source, /mapCanvasBackgroundReady/)
  assert.match(source, /\.map-background-layer\s*\{[\s\S]*position: absolute[\s\S]*inset: 0/)
  assert.match(source, /\.map-background-image\s*\{[\s\S]*position: absolute[\s\S]*width: 100%[\s\S]*height: 100%/)
  assert.match(source, /\.map-canvas\s*\{[\s\S]*position: absolute[\s\S]*inset: 0/)
  assert.match(source, /\.map-canvas\.ready\s*\{[\s\S]*opacity: 1/)
  assert.match(source, /\.map-layer\.hidden\s*\{[\s\S]*opacity: 0/)
})

test('sourcing map shows the canvas overlay after render is queued and hides native fallback only after the canvas base image draw completes', () => {
  expectTokens(source, [
    'mapCanvasOverlayReady',
    'mapNativeFallbackHidden',
    ':class="[\'map-canvas\', { ready: mapCanvasOverlayReady }]"',
    "drawBackground: 'whenReady'",
    'onDrawComplete',
    'backgroundDrawn',
    'mapCanvasRenderSeq',
  ])
  const renderMapCanvas = extractFunction('renderMapCanvas')
  assert.match(renderMapCanvas, /const renderSeq = \+\+mapCanvasRenderSeq/)
  assert.match(renderMapCanvas, /if \(!rendered\) \{[\s\S]*return\s*\}\s*mapCanvasOverlayReady\.value = true/)
  assert.match(renderMapCanvas, /onDrawComplete\(\{ backgroundDrawn \}\) \{[\s\S]*mapCanvasOverlayReady\.value = true[\s\S]*mapNativeFallbackHidden\.value = Boolean\(backgroundDrawn\)/)
  assert.doesNotMatch(source, /mapCanvasBackgroundReady/)
})

test('sourcing map keeps fallback map and viewport canvas sizes explicit', () => {
  expectTokens(source, [
    'mapLayerPixelSize',
    'buildMapLayerPixelSize',
    'mapCanvasPixelSize',
    'buildMapCanvasPixelSize',
    ':width="mapCanvasPixelSize.width"',
    ':height="mapCanvasPixelSize.height"',
    "drawBackground: 'whenReady'",
  ])
  assert.match(source, /const mapLayerPixelSize = computed\(\(\) => buildMapLayerPixelSize\(\)\)/)
  assert.match(source, /const mapCanvasPixelSize = computed\(\(\) => buildMapCanvasPixelSize\(\)\)/)
  assert.match(source, /<canvas[\s\S]*:width="mapCanvasPixelSize\.width"[\s\S]*:height="mapCanvasPixelSize\.height"/)
  assert.match(source, /function renderMapCanvas\(options = \{\}\)[\s\S]*width: mapCanvasPixelSize\.value\.width,[\s\S]*height: mapCanvasPixelSize\.value\.height/)
})

test('sourcing map renders the active transform inside a viewport-sized canvas', () => {
  expectTokens(source, [
    'mapCanvasPixelSize',
    'buildMapCanvasPixelSize',
    ':width="mapCanvasPixelSize.width"',
    ':height="mapCanvasPixelSize.height"',
    "drawBackground: 'whenReady'",
    'renderMapCanvas({ force: true, interacting: true })',
  ])
  assert.match(source, /<view :class="\['map-layer', \{ hidden: mapNativeFallbackHidden \}\]" :style="mapLayerStyle">[\s\S]*class="map-background-image"[\s\S]*<\/view>\s*<canvas/)
  assert.match(source, /mapRenderer\.render\(mapTransform\.value,[\s\S]*width: mapCanvasPixelSize\.value\.width,[\s\S]*height: mapCanvasPixelSize\.value\.height/)
})

test('sourcing map page focuses and highlights selected map objects', () => {
  expectTokens(source, [
    'mapTransform',
    'focusMapObject',
    'focusMapCenter',
    'selectFirstObjectAfterSearch',
    'calculateObjectCenter',
    'selectedObjectId',
  ])
  assert.match(source, /function selectMapObject\(object,\s*options = \{ focus: true \}\)/)
  assert.match(source, /if \(options\.focus\) \{\s*focusMapObject\(object\)\s*\}/)
})

test('sourcing map page removes title header and zoom toolbar chrome', () => {
  expectTokens(source, [
    'map-overlay-info',
    'map-service-controls',
    'map-service-button',
    'refreshCurrentMapData',
    'resetMapViewport',
    '归位',
    '刷新',
  ])
  for (const token of [
    'map-header',
    'header-kicker',
    'header-title',
    'header-subtitle',
    'refresh-button',
    'map-card-head',
    'zoom-toolbar',
    'zoom-button',
    'zoom-percent',
    'mapZoomPercent',
    'zoomInMap',
    'zoomOutMap',
    'resetMapZoom',
    'MAP_SCALE_STEP',
    '放大',
    '缩小',
  ]) {
    assert.doesNotMatch(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('sourcing map page applies configured default scene viewport', () => {
  expectTokens(source, [
    'applySceneDefaultViewport',
    'normalizeSceneDefaultScale',
    'focusMapCenter',
    'defaultScale',
    'defaultCenterX',
    'defaultCenterY',
    'mapMinScale',
    'MAP_MIN_SCALE',
    'MAP_MAX_SCALE',
  ])
  assert.match(source, /selectedScene\.value = resp\.item \|\| scene[\s\S]*applySceneDefaultViewport\(selectedScene\.value\)/)
  assert.match(source, /function applySceneDefaultViewport\(scene\)[\s\S]*setMapTransform\(/)
  assert.match(source, /focusMapCenter\(\{ x: centerX, y: centerY \},\s*\{ reloadViewport: false \}\)/)
  assert.match(source, /function normalizeSceneDefaultScale\(value\)[\s\S]*function normalizeMapScale\(value\)[\s\S]*Math\.min\(MAP_MAX_SCALE,\s*Math\.max\(mapMinScale\.value,\s*scale\)/)
})

test('sourcing map page provides navigation with address fallback', () => {
  expectTokens(source, [
    '导航',
    'openSelectedObjectLocation',
    'uni.openLocation',
    'buildNavigationPayload',
    '没有精确定位，已复制地址',
    '该点位暂未提供可导航地址',
  ])
  assert.match(source, /openLocation\(\{[\s\S]*latitude:\s*payload\.latitude,[\s\S]*longitude:\s*payload\.longitude,[\s\S]*name:\s*payload\.name,[\s\S]*address:\s*payload\.address/)
  assert.match(source, /if \(!payload\.latitude \|\| !payload\.longitude\) \{[\s\S]*uni\.setClipboardData\(\{ data: payload\.address \}\)/)
})

test('sourcing map page renders readable object and poi details', () => {
  expectTokens(source, [
    'detailFields',
    'detailTags',
    'defaultLabelDictionary',
    'categoryLabels',
    'formatLabelList',
    'formatExtraValue',
    '营业时间',
    '支持服务',
    '物流线路',
    '发车时间',
    '收费说明',
    'selectNearbyPoi',
    'getMapObject',
  ])
  assert.match(source, /<view v-if="detailTags\.length" class="detail-tag-list">/)
  assert.match(source, /v-for="field in detailFields"/)
  assert.match(source, /@click="selectNearbyPoi\(poi\)"/)
  assert.match(source, /const detail = await getMapObject\(poi\.id/)
})

test('sourcing map highlights verified merchants and weak admin objects in canvas and list', () => {
  expectTokens(source, [
    'displaySource',
    'displayLevel',
    'isVerifiedMerchant',
    'verified_merchant',
    'highlight',
    'weak',
    'objectRowClasses',
    'selectedObjectMerchant',
    '认证商户',
    '后台点位',
  ])
  expectTokens(rendererSource, [
    'drawVerifiedBadge',
    'isVerifiedObject',
    'drawSelectedOutline',
    'displayLevel',
    'isVerifiedMapObject',
  ])
  assert.match(source, /objectDisplayName\(object\)/)
  assert.match(source, /v-if="selectedObject\.isVerifiedMerchant"/)
})

test('sourcing map page supports canvas gesture zoom and level based labels', () => {
  expectTokens(source, [
    'mapTransform',
    'mapZoomLevel',
    'handleCanvasTouchMove',
    'resetMapViewport',
    'screenToMap',
    'moveGesture',
    'mapLayerStyle',
  ])
  assert.match(source, /const mapZoomLevel = computed\(\(\) => getZoomLevelByScale\(mapTransform\.value\.scale\)\)/)
  assert.match(source, /function getZoomLevelByScale\(scale\)/)
  assert.match(source, /function handleCanvasTouchMove\(event\)/)
  assert.match(source, /if \(!selected && !verified && !rentable\) return ''/)
  assert.match(source, /if \(!selected && verified && mapZoomLevel\.value < 5\) return ''/)
  assert.match(source, /if \(!selected && rentable && mapZoomLevel\.value < 4\) return ''/)
  assert.match(source, /return objectDisplayName\(object\)/)
})

test('sourcing map redraws a viewport canvas during gestures to avoid native canvas transform drift', () => {
  expectTokens(source, [
    'map-layer',
    'mapLayerStyle',
    'mapLayerSurfaceStyle',
    'buildMapLayerStyle',
    ':style="mapLayerStyle"',
    'mapCanvasSurfaceStyle',
    'renderMapCanvas({ force: true, interacting: true })',
  ])

  const moveHandler = extractFunction('handleCanvasTouchMove')
  const endHandler = extractFunction('handleCanvasTouchEnd')
  assert.match(moveHandler, /renderMapCanvas\(\{ force: true,\s*interacting: true \}\)/)
  assert.match(endHandler, /renderMapCanvas\(\)/)
  assert.match(source, /<view :class="\['map-layer', \{ hidden: mapNativeFallbackHidden \}\]" :style="mapLayerStyle">[\s\S]*class="map-background-image"[\s\S]*<\/view>\s*<canvas/)
})

test('sourcing map gives rubber band feedback when users drag past map bounds', () => {
  expectTokens(source, [
    'boundaryHintEdges',
    'buildBoundaryHintEdges',
    'map-boundary-hints',
    'map-edge-hint',
    'allowOverflow: true',
    'centeredY',
  ])
  assert.match(source, /v-if="boundaryHintEdges\.length" class="map-boundary-hints"/)
  assert.match(source, /setMapTransform\(result\.transform,\s*\{ allowOverflow: true \}\)/)
  assert.match(source, /if \(scaledHeight <= options\.viewportHeight\) \{[\s\S]*centeredY[\s\S]*edges\.push\('top'\)[\s\S]*edges\.push\('bottom'\)/)
  assert.match(source, /\.map-edge-hint\.left/)
  assert.match(source, /\.map-edge-hint\.right/)
  assert.match(source, /\.map-edge-hint\.top/)
  assert.match(source, /\.map-edge-hint\.bottom/)
})

test('sourcing map exposes edge feedback padding and accounts for bottom safe area', () => {
  expectTokens(source, [
    'MAP_EDGE_FEEDBACK_PADDING_PX',
    'mapSafeAreaBottomPx',
    'mapEdgeFeedbackPadding',
    'bottomEdgeHintStyle',
    'calculateSafeAreaBottomPx',
    'safeAreaInsets',
    'safeArea',
  ])
  assert.match(source, /const mapSafeAreaBottomPx = ref\(0\)/)
  assert.match(source, /const mapEdgeFeedbackPadding = computed\(\(\) => MAP_EDGE_FEEDBACK_PADDING_PX \+ mapSafeAreaBottomPx\.value\)/)
  assert.match(source, /maxOverflow:\s*mapEdgeFeedbackPadding\.value/)
  assert.match(source, /mapSafeAreaBottomPx\.value = calculateSafeAreaBottomPx\(info\)/)
  assert.match(source, /class="map-edge-hint bottom" :style="bottomEdgeHintStyle"/)
  assert.match(source, /function buildBottomEdgeHintStyle\(\)[\s\S]*mapSafeAreaBottomPx\.value/)
})

test('sourcing map scales the base map by the shortest edge to cover the viewport', () => {
  expectTokens(source, ['getSceneRenderMetrics', 'viewportWidth / sceneWidth', 'viewportHeight / sceneHeight'])
  assert.match(source, /const baseScale = Math\.max\(viewportWidth \/ sceneWidth,\s*viewportHeight \/ sceneHeight\)/)
})

test('sourcing map can zoom out until the whole base map is visible', () => {
  expectTokens(source, [
    'mapMinScale',
    'calculateMapMinScale',
    'viewportWidth / metrics.mapWidth',
    'viewportHeight / metrics.mapHeight',
  ])
  assert.match(source, /const mapMinScale = computed\(\(\) => calculateMapMinScale\(\)\)/)
  assert.match(source, /minScale: mapMinScale\.value/)
  assert.doesNotMatch(source, /minScale: MAP_MIN_SCALE/)
  assert.match(source, /function normalizeMapScale\(value\)[\s\S]*Math\.max\(mapMinScale\.value,\s*scale\)/)
})

test('sourcing map page filters visible objects by configured zoom range', () => {
  expectTokens(source, [
    'rawMapObjects',
    'visibleMapObjects',
    'isObjectVisibleAtZoom',
    'minZoom',
    'maxZoom',
  ])
  assert.match(source, /const mapObjects = computed\(\(\) => visibleMapObjects\.value\)/)
  assert.match(source, /const visibleMapObjects = computed\(\(\) => rawMapObjects\.value\.filter\(\(object\) => isObjectVisibleAtZoom\(object,\s*mapZoomLevel\.value\)\)\)/)
  assert.match(source, /function isObjectVisibleAtZoom\(object,\s*zoomLevel\)[\s\S]*const minZoom = toNumber\(object\?\.minZoom,\s*1\)[\s\S]*const maxZoom = toNumber\(object\?\.maxZoom,\s*5\)[\s\S]*return zoomLevel >= minZoom && zoomLevel <= maxZoom/)
})

test('sourcing map page requests objects by current canvas viewport', () => {
  expectTokens(source, [
    '@touchmove.stop.prevent="handleCanvasTouchMove"',
    'handleCanvasTouchMove',
    'handleCanvasTouchEnd',
    'scheduleViewportObjectReload',
    'buildViewportQueryParams',
    'VIEWPORT_PADDING_RATIO',
    'viewportReloadTimer',
    'screenToMap',
    'minX',
    'minY',
    'maxX',
    'maxY',
    'zoom: mapZoomLevel.value',
  ])

  const moveHandler = extractFunction('handleCanvasTouchMove')
  const endHandler = extractFunction('handleCanvasTouchEnd')
  assert.doesNotMatch(moveHandler, /loadSceneObjects/)
  assert.match(endHandler, /scheduleViewportObjectReload\(\)/)
  assert.match(source, /listMapObjects\(selectedSceneCode\.value,\s*buildObjectQueryParams\(\{ includeViewport: true \}\)\)/)
  assert.match(source, /setTimeout\(async \(\) => \{[\s\S]*await loadSceneObjects\(\{ keepSelection: true,\s*silent: true \}\)/)
})

test('sourcing map clears selected merchant details when the selected object leaves the visible viewport', () => {
  expectTokens(source, [
    'isObjectInBounds',
    'isSelectedObjectInViewport',
    'clearSelectedObjectOutsideViewport',
    'getVisibleSceneBounds',
  ])

  const moveHandler = extractFunction('handleCanvasTouchMove')
  const endHandler = extractFunction('handleCanvasTouchEnd')
  const syncSelection = extractFunction('syncSelectedObjectAfterLoad')
  const visibilityChecker = extractFunction('isSelectedObjectInViewport')

  assert.match(source, /import \{ hitTestMapObjects,\s*isObjectInBounds \} from '\.\/mapHitTest'/)
  assert.match(moveHandler, /setMapTransform\(result\.transform,\s*\{ allowOverflow: true \}\)[\s\S]*clearSelectedObjectOutsideViewport\(\)/)
  assert.match(endHandler, /setMapTransform\(nextTransform\)[\s\S]*clearSelectedObjectOutsideViewport\(\)/)
  assert.match(syncSelection, /latest && isSelectedObjectInViewport\(latest\)/)
  assert.match(visibilityChecker, /isObjectInBounds\(object,\s*getVisibleSceneBounds\(\)\)/)
})

test('sourcing map converts touch and tap coordinates into canvas-local coordinates', () => {
  expectTokens(source, [
    'canvasShellRect',
    'syncCanvasShellRect',
    'normalizeCanvasPoint',
    'uni.createSelectorQuery',
    '.map-canvas-shell',
  ])
  assert.match(source, /function handleCanvasTouchStart\(event\)[\s\S]*syncCanvasShellRect\(\)/)
  assert.match(source, /function getCanvasTouches\(event\)[\s\S]*normalizeCanvasPoint\(/)
  assert.match(source, /function getCanvasEventPoint\(event\)[\s\S]*normalizeCanvasPoint\(/)
})

test('sourcing map reloads viewport objects after programmatic focus changes', () => {
  const focusHandler = extractFunction('focusMapCenter')
  assert.match(focusHandler, /setMapTransform\(nextTransform\)/)
  assert.match(focusHandler, /scheduleViewportObjectReload\(\)/)
})

test('sourcing map centers the only loaded object so a single result is visible', () => {
  expectTokens(source, [
    'focusSingleObjectAfterLoad',
    'mapObjects.value.length !== 1',
    'selectedObjectId.value',
    'focusMapObject(mapObjects.value[0], { reloadViewport: false })',
  ])
  assert.match(source, /function loadSceneObjects\(options = \{\}\)[\s\S]*syncSelectedObjectAfterLoad\(\)[\s\S]*focusSingleObjectAfterLoad\(\)/)
  assert.match(source, /function focusMapObject\(object,\s*options = \{\}\)[\s\S]*focusMapCenter\(calculateObjectCenter\(object\),\s*options\)/)
})

test('sourcing map uses dropship as the canonical one-piece shipping tag with legacy alias support', () => {
  expectTokens(source, [
    'dropship: \'一件代发\'',
    'drop_shipping: \'一件代发\'',
    'tagAliases',
    'expandFilterValues',
    'normalizeFilterOptionValue',
  ])
  assert.match(source, /\{ label: '一件代发', value: 'dropship' \}/)
  assert.doesNotMatch(source, /\{ label: '一件代发', value: 'drop_shipping' \}/)
})

test('sourcing map renders and hits polygon map objects through canvas modules', () => {
  expectTokens(source, [
    'polygon',
    'calculatePolygonCenter',
    'hitTestMapObjects',
    'canvasRenderer',
    'mapHitTest',
  ])
  expectTokens(rendererSource, ['drawPolygonObject', 'normalizePolygonPoints', 'fillPolygonPath'])
  expectTokens(hitTestSource, ['isPointInPolygon', 'getObjectBounds', 'hitTestMapObjects'])
  assert.match(source, /function calculateObjectCenter\(object\)[\s\S]*if \(object\.geometryType === 'polygon'\) \{[\s\S]*return calculatePolygonCenter\(geometry\)/)
})

test('sourcing map uses a full-screen width canvas viewport without horizontal page overflow', () => {
  expectTokens(source, [
    'const MAP_MAX_WIDTH_RPX = 750',
    'mapViewportHeightRpx',
    'mapViewportSize',
    'mapCanvasStyle',
    'syncMapViewportSize',
    'calculateMapViewportHeightRpx',
    ':style="mapCanvasStyle"',
    'width: 100vw',
    'max-width: 100vw',
    'overflow-x: hidden',
    'box-sizing: border-box',
  ])

  assert.match(source, /\.map-card\s*\{[\s\S]*width: 100vw[\s\S]*max-width: 100vw/)
  assert.match(source, /\.map-canvas-shell\s*\{[\s\S]*width: 100vw[\s\S]*max-width: 100vw/)
  assert.match(source, /mapViewportSize\.value\.height/)
  assert.match(source, /rpxToPx\(mapViewportHeightRpx\.value\)/)
})

test('sourcing map viewport reload is silent and ignores stale object responses', () => {
  expectTokens(source, [
    'objectRequestSeq',
    'visibleObjectRequestSeq',
    'const requestId = ++objectRequestSeq',
    'const showLoading = !options.silent',
    'requestId !== objectRequestSeq',
    'loadSceneObjects({ keepSelection: true, silent: true })',
  ])

  assert.match(source, /if \(showLoading\) \{[\s\S]*objectLoading\.value = true/)
  assert.match(source, /if \(requestId !== objectRequestSeq\) return/)
})

test('sourcing map canvas tap reuses existing selection flow', () => {
  expectTokens(source, [
    'handleCanvasTap',
    'getCanvasEventPoint',
    'screenToScenePoint',
    'hitTestMapObjects',
    'selectMapObject',
    'selectedObjectId',
  ])

  const tapHandler = extractFunction('handleCanvasTap')
  assert.match(tapHandler, /hitTestMapObjects\(/)
  assert.match(tapHandler, /selectMapObject\(hitObject/)
  assert.doesNotMatch(tapHandler, /loadNearbyPois\(/)
})
