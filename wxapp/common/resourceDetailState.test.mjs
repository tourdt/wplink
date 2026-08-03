import test from 'node:test'
import assert from 'node:assert/strict'
import {
  buildDetailSpecItems,
  buildResourceDetailPresentation,
  buildResourceDetailSpecItems,
} from './resourceDetailState.js'

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

test('places demand quantity, budget, and attributes in unified spec order', () => {
  const items = buildResourceDetailSpecItems({
    direction: 'demand',
    quantityText: '5000 件',
    priceText: '面议',
  }, [
    { label: '交期要求', value: '20 天内完成' },
  ])

  assert.deepEqual(items, [
    { label: '数量/面积', value: '5000 件', fullWidth: false },
    { label: '预算/报价', value: '面议', fullWidth: false },
    { label: '交期要求', value: '20 天内完成', fullWidth: true },
  ])
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
