import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
const apiSource = readOptionalSource('api/sourcingMap.js')
const source = readOptionalSource('pages/sourcing-map/index.vue')

function readOptionalSource(file) {
  const fullPath = path.join(root, file)
  return fs.existsSync(fullPath) ? fs.readFileSync(fullPath, 'utf8') : ''
}

test('sourcing map page is reachable from wxapp home', () => {
  assert.ok(pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/index'))
  assert.match(homeSource, /拿货地图/)
  assert.match(homeSource, /openSourcingMap/)
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
})

test('sourcing map api uses public map endpoints', () => {
  for (const token of [
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
  ]) {
    assert.match(apiSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('sourcing map page loads scenes, renders objects and shows contact actions', () => {
  for (const token of [
    'onLoad',
    'onPullDownRefresh',
    'loadScenes',
    'loadSceneObjects',
    'selectScene',
    'selectMapObject',
    'submitSearch',
    'clearSearch',
    'objectStyle',
    'stageStyle',
    'selectedObject',
    'callSelectedObject',
    'copySelectedWechat',
    'nearbyPois',
    'loadNearbyPois',
    '地图暂未开放',
    '暂无匹配点位',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('sourcing map page supports quick category and poi filters', () => {
  for (const token of [
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
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /listMapObjects\(selectedSceneCode\.value,\s*buildObjectQueryParams\(\{ includeViewport: true \}\)\)/)
  assert.match(source, /searchMapObjects\(\{\s*\.\.\.buildObjectQueryParams\(\{ includeViewport: false \}\),\s*sceneCode:/)
})

test('sourcing map empty results can clear search and filters', () => {
  for (const token of [
    'empty-actions',
    '清除搜索',
    '清除筛选',
    'keyword',
    'hasActiveFilters',
    '@click="clearSearch"',
    '@click="clearFilters"',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /v-if="keyword"/)
  assert.match(source, /v-if="hasActiveFilters"/)
})

test('sourcing map page focuses and highlights selected map objects', () => {
  for (const token of [
    ':scroll-left="mapScrollLeft"',
    ':scroll-top="mapScrollTop"',
    'scroll-with-animation',
    'mapScrollLeft',
    'mapScrollTop',
    'focusMapObject',
    'selectFirstObjectAfterSearch',
    'calculateObjectCenter',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /function selectMapObject\(object,\s*options = \{ focus: true \}\)/)
  assert.match(source, /if \(options\.focus\) \{\s*focusMapObject\(object\)\s*\}/)
})

test('sourcing map page applies configured default scene viewport', () => {
  for (const token of [
    'applySceneDefaultViewport',
    'normalizeSceneDefaultScale',
    'focusMapCenter',
    'defaultScale',
    'defaultCenterX',
    'defaultCenterY',
    'MAP_MIN_SCALE',
    'MAP_MAX_SCALE',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /selectedScene\.value = resp\.item \|\| scene[\s\S]*applySceneDefaultViewport\(selectedScene\.value\)/)
  assert.match(source, /function applySceneDefaultViewport\(scene\)[\s\S]*mapScale\.value = normalizeSceneDefaultScale\(scene\?\.defaultScale\)/)
  assert.match(source, /focusMapCenter\(\{ x: centerX, y: centerY \}\)/)
  assert.match(source, /function normalizeSceneDefaultScale\(value\)[\s\S]*Math\.min\(MAP_MAX_SCALE,\s*Math\.max\(MAP_MIN_SCALE/)
})

test('sourcing map page provides navigation with address fallback', () => {
  for (const token of [
    '导航',
    'openSelectedObjectLocation',
    'uni.openLocation',
    'buildNavigationPayload',
    '没有精确定位，已复制地址',
    '该点位暂未提供可导航地址',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /openLocation\(\{[\s\S]*latitude:\s*payload\.latitude,[\s\S]*longitude:\s*payload\.longitude,[\s\S]*name:\s*payload\.name,[\s\S]*address:\s*payload\.address/)
  assert.match(source, /if \(!payload\.latitude \|\| !payload\.longitude\) \{[\s\S]*uni\.setClipboardData\(\{ data: payload\.address \}\)/)
})

test('sourcing map page renders readable object and poi details', () => {
  for (const token of [
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
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /<view v-if="detailTags\.length" class="detail-tag-list">/)
  assert.match(source, /v-for="field in detailFields"/)
  assert.match(source, /@click="selectNearbyPoi\(poi\)"/)
  assert.match(source, /const detail = await getMapObject\(poi\.id/)
})

test('sourcing map page supports zoom controls and level based labels', () => {
  for (const token of [
    'zoom-toolbar',
    'zoomInMap',
    'zoomOutMap',
    'resetMapZoom',
    'mapScale',
    'mapZoomPercent',
    'mapZoomLevel',
    'effectiveStageScale',
    'objectDisplayLabel',
    '放大',
    '缩小',
    '复位',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /const effectiveStageScale = computed\(\(\) => stageScale\.value \* mapScale\.value\)/)
  assert.match(source, /const mapZoomLevel = computed\(\(\) => getZoomLevelByScale\(mapScale\.value\)\)/)
  assert.match(source, /function getZoomLevelByScale\(scale\)/)
  assert.match(source, /function changeMapScale\(nextScale\)/)
  assert.match(source, /if \(mapZoomLevel\.value < 4\) return object\.code \|\| ''/)
  assert.match(source, /return object\.name \|\| object\.code \|\| ''/)
})

test('sourcing map page filters visible objects by configured zoom range', () => {
  for (const token of [
    'rawMapObjects',
    'visibleMapObjects',
    'isObjectVisibleAtZoom',
    'minZoom',
    'maxZoom',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /const mapObjects = computed\(\(\) => visibleMapObjects\.value\)/)
  assert.match(source, /const visibleMapObjects = computed\(\(\) => rawMapObjects\.value\.filter\(\(object\) => isObjectVisibleAtZoom\(object,\s*mapZoomLevel\.value\)\)\)/)
  assert.match(source, /function isObjectVisibleAtZoom\(object,\s*zoomLevel\)[\s\S]*const minZoom = toNumber\(object\?\.minZoom,\s*1\)[\s\S]*const maxZoom = toNumber\(object\?\.maxZoom,\s*5\)[\s\S]*return zoomLevel >= minZoom && zoomLevel <= maxZoom/)
})

test('sourcing map page requests objects by current viewport', () => {
  for (const token of [
    '@scroll="handleMapScroll"',
    'handleMapScroll',
    'scheduleViewportObjectReload',
    'buildViewportQueryParams',
    'VIEWPORT_PADDING_RATIO',
    'viewportReloadTimer',
    'pxToRpx',
    'minX',
    'minY',
    'maxX',
    'maxY',
    'zoom: mapZoomLevel.value',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /listMapObjects\(selectedSceneCode\.value,\s*buildObjectQueryParams\(\{ includeViewport: true \}\)\)/)
  assert.match(source, /function changeMapScale\(nextScale\)[\s\S]*scheduleViewportObjectReload\(\)/)
  assert.match(source, /setTimeout\(async \(\) => \{[\s\S]*await loadSceneObjects\(\{ keepSelection: true \}\)/)
})

test('sourcing map page renders polygon map objects', () => {
  for (const token of [
    'polygon',
    'polygonObjects',
    'rectAndPointObjects',
    'polygonObjectStyle',
    'polygonStagePoints',
    'map-polygon',
    'calculatePolygonCenter',
    'clip-path: polygon',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /<view[\s\S]*v-for="object in polygonObjects"[\s\S]*:style="polygonObjectStyle\(object\)"/)
  assert.match(source, /const polygonObjects = computed\(\(\) => mapObjects\.value\.filter/)
  assert.match(source, /function calculateObjectCenter\(object\)[\s\S]*if \(object\.geometryType === 'polygon'\) \{[\s\S]*return calculatePolygonCenter\(geometry\)/)
})
