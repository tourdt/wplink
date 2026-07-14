import assert from 'node:assert/strict'
import test from 'node:test'

import {
  DEFAULT_RESOURCE_SHARE_IMAGE,
  buildResourceSharePath,
  buildResourceSharePayload,
  buildResourceSharePosterModel,
  buildResourceTimelinePayload,
  getResourceShareCoverSource,
} from './resourceShare.js'

const resource = {
  id: 'res_1001',
  title: '织里童装库存 3000 件急清',
  typeCode: 'stock_clearance',
  cityName: '织里',
  category: '童装',
  quantityText: '3000 件',
  priceText: '打包 9.9 元',
  coverUrl: 'https://img.example.com/cover.jpg',
  merchant: {
    name: '陈厂长童装工厂',
    vipStatus: 'active',
  },
}

test('resource share payload uses a concrete resource path and generated image', () => {
  const payload = buildResourceSharePayload(resource, '/tmp/share-cover.png')

  assert.equal(payload.title, '库存出售｜织里童装库存 3000 件急清')
  assert.equal(payload.path, '/pages/resource/detail?id=res_1001')
  assert.equal(payload.imageUrl, '/tmp/share-cover.png')
})

test('resource timeline payload carries resource id in query because timeline cannot customize page path', () => {
  const payload = buildResourceTimelinePayload(resource, '/tmp/share-cover.png')

  assert.equal(payload.title, '库存出售｜织里童装库存 3000 件急清')
  assert.equal(payload.query, 'id=res_1001')
  assert.equal(payload.imageUrl, '/tmp/share-cover.png')
})

test('resource share helpers fall back safely when resource data is incomplete', () => {
  assert.equal(buildResourceSharePath({}), '/pages/home/index')
  assert.equal(getResourceShareCoverSource({ images: ['https://img.example.com/first.jpg'] }), 'https://img.example.com/first.jpg')
  assert.equal(getResourceShareCoverSource({}), DEFAULT_RESOURCE_SHARE_IMAGE)
})

test('resource share poster model keeps the most useful sales details for canvas rendering', () => {
  const model = buildResourceSharePosterModel(resource, { name: '陈厂长童装工厂' })

  assert.equal(model.typeLabel, '库存出售')
  assert.equal(model.title, '织里童装库存 3000 件急清')
  assert.equal(model.coverSource, 'https://img.example.com/cover.jpg')
  assert.deepEqual(model.badges, ['VIP', '织里', '童装'])
  assert.deepEqual(model.summaryLines, ['3000 件', '打包 9.9 元'])
  assert.equal(model.merchantName, '陈厂长童装工厂')
})
