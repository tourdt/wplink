import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const statePath = path.resolve(new URL('.', import.meta.url).pathname, 'resourceFeedCardState.js')
const stateModule = fs.existsSync(statePath) ? await import('./resourceFeedCardState.js') : {}
const { buildResourceFeedCardModel } = stateModule

test('buildResourceFeedCardModel normalizes demand content for the shared compact card', () => {
  assert.equal(typeof buildResourceFeedCardModel, 'function')
  const model = buildResourceFeedCardModel({
    direction: 'demand',
    images: ['/images/demand.png'],
    typeName: '找加工',
    title: '  需要小单快返加工  ',
    quantityText: '  500 件  ',
    priceText: '  价格面议  ',
    merchant: { name: '  星河采购  ' },
    refreshedAt: '2026-07-31T09:30:00+08:00',
    dealtAt: '2026-08-01T10:00:00+08:00',
  }, new Date(2026, 7, 1, 12, 0, 0))

  assert.deepEqual(model, {
    isDemand: true,
    directionLabel: '需求',
    coverUrl: '/images/demand.png',
    resourceTypeLabel: '找加工',
    titleText: '需要小单快返加工',
    quantityText: '500 件',
    priceText: '价格面议',
    hasTradeInfo: true,
    merchantName: '星河采购',
    freshnessText: '昨天',
    isCompleted: true,
  })
})

test('buildResourceFeedCardModel uses supply fallbacks without rendering a fake refresh time', () => {
  assert.equal(typeof buildResourceFeedCardModel, 'function')
  const model = buildResourceFeedCardModel({
    direction: 'unexpected',
    images: [],
    merchant: {},
  }, new Date(2026, 7, 1, 12, 0, 0))

  assert.deepEqual(model, {
    isDemand: false,
    directionLabel: '供应',
    coverUrl: '',
    resourceTypeLabel: '',
    titleText: '供应标题待完善',
    quantityText: '',
    priceText: '',
    hasTradeInfo: false,
    merchantName: '商家待确认',
    freshnessText: '',
    isCompleted: false,
  })
})

test('buildResourceFeedCardModel uses demand-specific fallbacks', () => {
  assert.equal(typeof buildResourceFeedCardModel, 'function')
  const model = buildResourceFeedCardModel({ direction: 'demand' })

  assert.equal(model.titleText, '需求标题待完善')
  assert.equal(model.merchantName, '采购方待确认')
})
