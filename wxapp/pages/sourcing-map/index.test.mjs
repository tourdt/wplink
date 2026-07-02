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
    '/api/v1/map/scenes',
    '/api/v1/map/objects/search',
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
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
  assert.match(source, /listMapObjects\(selectedSceneCode\.value,\s*buildObjectQueryParams\(\)\)/)
  assert.match(source, /searchMapObjects\(\{\s*\.\.\.buildObjectQueryParams\(\),\s*sceneCode:/)
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
    'labelDictionary',
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
