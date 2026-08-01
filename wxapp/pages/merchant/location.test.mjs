import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

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
    '/api/v1/map/merchants/${merchantId}/location-context',
    'suppressErrorToast: true',
  ])
  expectTokens(source, [
    'getMerchantLocationContext',
    'normalizeMerchantLocationContext',
    'buildMerchantLocationMarkers',
    ':scale="16"',
    '@markertap="handleMarkerTap"',
    '导航到店',
  ])
  assert.doesNotMatch(source, /show-location|uni\.getLocation|搜索此区域/)
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
