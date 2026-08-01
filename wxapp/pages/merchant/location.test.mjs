import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = read('pages/merchant/location.vue')
const apiSource = read('api/sourcingMap.js')
const pagesConfig = JSON.parse(read('pages.json'))

function read(file) {
  const fullPath = path.join(root, file)
  return fs.existsSync(fullPath) ? fs.readFileSync(fullPath, 'utf8') : ''
}

function expectTokens(target, tokens) {
  for (const token of tokens) {
    assert.match(target, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
}

function loadSourcingMapApi(request) {
  const executableSource = apiSource
    .replace("import request from './request'", '')
    .replaceAll('export function ', 'function ')
  const sandbox = { request }
  vm.runInNewContext(`${executableSource}\nglobalThis.sourcingMapApi = { getMerchantLocationContext }`, sandbox)
  return sandbox.sourcingMapApi
}

test('merchant location page is registered with its business title', () => {
  const page = pagesConfig.pages.find((item) => item.path === 'pages/merchant/location')

  assert.deepEqual(page, {
    path: 'pages/merchant/location',
    style: { navigationBarTitleText: '店铺位置' },
  })
})

test('merchant location page loads the context and keeps the current shop as map focus', () => {
  expectTokens(apiSource, [
    'getMerchantLocationContext',
    'suppressErrorToast: true',
  ])
  expectTokens(source, [
    'getMerchantLocationContext',
    'normalizeMerchantLocationContext',
    'buildMerchantLocationMarkers',
    ':scale="mapScale"',
    'const mapScale = ref(16)',
    '@markertap="handleMarkerTap"',
    '导航到店',
  ])
  assert.doesNotMatch(source, /show-location|uni\.getLocation|搜索此区域/)
})

test('merchant location API trims and encodes the merchant id as one path segment', async () => {
  let requestOptions
  const { getMerchantLocationContext } = loadSourcingMapApi((options) => {
    requestOptions = options
    return Promise.resolve({})
  })

  await getMerchantLocationContext(' merchant/a?b#c% ')

  assert.match(
    requestOptions.url,
    /\/api\/v1\/map\/merchants\/merchant%2Fa%3Fb%23c%25\/location-context$/,
  )
  assert.equal(requestOptions.method, 'GET')
  assert.equal(requestOptions.suppressErrorToast, true)
})

test('merchant location page distinguishes full context failure from nearby degradation', () => {
  expectTokens(source, [
    '该商家暂时无法查看',
    '该商家位置待完善',
    '重新加载',
    '返回',
    'nearbyAvailable',
    '周边商家加载失败，请重试',
  ])
})

test('merchant location page exposes current shop details and a safe navigation failure', () => {
  expectTokens(source, [
    '当前档口',
    '店铺地址',
    '主营',
    'uni.openLocation',
    '导航打开失败，请稍后重试',
  ])
})

test('merchant location page exposes the collapsed and half-screen nearby drawer states', () => {
  expectTokens(source, [
    '周边已入驻商家',
    'nearby-drawer',
    'openNearbyDrawer',
    'closeNearbyDrawer',
    'scroll-into-view',
    '附近暂无其他入驻商家',
    '周边商家加载失败，请重试',
  ])
  assert.match(source, /<scroll-view[\s\S]*scroll-y[\s\S]*:scroll-into-view="scrollIntoViewId"/)
  assert.match(source, /max-height:\s*55vh/)
})

test('merchant location page links a nearby marker, list item, and merchant detail without replacing current context', () => {
  expectTokens(source, [
    'id="merchantLocationMap"',
    'selectNearbyMerchant',
    'openNearbyMerchant',
    "uni.createMapContext('merchantLocationMap')",
    'includePoints',
    'scrollIntoViewId',
    'nearbyTags(place)',
    'slice(0, 3)',
    '/pages/merchant/detail?id=',
    'encodeURIComponent(place.merchantId)',
    'mapScale.value = 16',
  ])
  assert.match(source, /:latitude="mapCenter\.latitude"[\s\S]*:longitude="mapCenter\.longitude"/)
})
