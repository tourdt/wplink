import test from 'node:test'
import assert from 'node:assert/strict'
import {
  buildDetailSpecItems,
  buildResourceDetailPresentation,
} from './resourceDetailState.js'

test('builds demand headline and only includes populated key facts', () => {
  const presentation = buildResourceDetailPresentation({
    direction: 'demand',
    typeName: '招加工厂',
    title: '招加工厂｜童装 5000 件 面议',
    quantityText: '5000 件',
    priceText: '面议',
  })

  assert.deepEqual(presentation, {
    isDemand: true,
    noun: '需求',
    typeName: '招加工厂',
    headline: '招加工厂｜童装 5000 件 面议',
    facts: [
      { key: 'quantity', label: '数量/面积', value: '5000 件' },
      { key: 'price', label: '预算/报价', value: '面议' },
    ],
  })
})

test('falls back to configured summary fields and legacy demand type codes', () => {
  const presentation = buildResourceDetailPresentation({
    typeCode: 'seek_factory_warehouse',
    typeName: '我要求租',
    category: '一楼仓',
    quantityText: '500-800 平',
    priceText: '3 万元/月以内',
  })

  assert.equal(presentation.isDemand, true)
  assert.equal(presentation.noun, '需求')
  assert.equal(presentation.headline, '一楼仓｜500-800 平｜3 万元/月以内')
})

test('marks long and semantic detail values as full width', () => {
  const items = buildDetailSpecItems([
    { label: '数量', value: '3200 件' },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认' },
    { label: '服务范围', value: '织里及周边' },
  ])

  assert.deepEqual(items, [
    { label: '数量', value: '3200 件', fullWidth: false },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认', fullWidth: true },
    { label: '服务范围', value: '织里及周边', fullWidth: true },
  ])
})
