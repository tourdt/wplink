import test from 'node:test'
import assert from 'node:assert/strict'
import { buildResourceDetailPresentation } from './resourceDetailState.js'

test('builds demand direction and type presentation without summary facts', () => {
  const presentation = buildResourceDetailPresentation({
    direction: 'demand',
    typeName: '招加工厂',
    quantityText: '5000 件',
    priceText: '面议',
  })

  assert.deepEqual(presentation, {
    isDemand: true,
    noun: '需求',
    typeName: '招加工厂',
  })
})

test('recognizes legacy demand type codes', () => {
  const presentation = buildResourceDetailPresentation({
    typeCode: 'seek_factory_warehouse',
    typeName: '我要求租',
    category: '一楼仓',
    quantityText: '500-800 平',
    priceText: '3 万元/月以内',
  })

  assert.equal(presentation.isDemand, true)
  assert.equal(presentation.noun, '需求')
  assert.equal(presentation.typeName, '我要求租')
})
