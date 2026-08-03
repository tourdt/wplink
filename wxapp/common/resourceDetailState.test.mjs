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

test('omits only attributes whose keys are the core summary sources', () => {
  const items = buildResourceDetailSpecItems({
    direction: 'demand',
    quantityText: '5000 件',
    priceText: '面议',
    summarySourceKeys: {
      quantityText: 'demandQuantityText',
      priceText: 'budgetRange',
    },
  }, [
    { key: 'demandQuantityText', label: '需求数量', value: '5000 件' },
    { key: 'budgetRange', label: '预算范围', value: '面议' },
    { key: 'deliveryBudgetNote', label: '预算范围', value: '含运费预算另议' },
    { key: 'deliveryTime', label: '交期要求', value: '20 天内完成' },
  ])

  assert.deepEqual(items, [
    { label: '数量/面积', value: '5000 件', fullWidth: false },
    { label: '预算/报价', value: '面议', fullWidth: false },
    { label: '预算范围', value: '含运费预算另议', fullWidth: false },
    { label: '交期要求', value: '20 天内完成', fullWidth: true },
  ])
})

test('keeps short service scope in two columns and expands only allowed labels or long values', () => {
  const items = buildDetailSpecItems([
    { label: '数量', value: '3200 件' },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认' },
    { label: '服务范围', value: '织里及周边' },
    { label: '工艺', value: '锁边' },
    { label: '生产能力', value: '可承接秋冬童装羽绒服针织衫梭织裤装订单并支持来样打版' },
  ])

  assert.deepEqual(items, [
    { label: '数量', value: '3200 件', fullWidth: false },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认', fullWidth: true },
    { label: '服务范围', value: '织里及周边', fullWidth: false },
    { label: '工艺', value: '锁边', fullWidth: false },
    { label: '生产能力', value: '可承接秋冬童装羽绒服针织衫梭织裤装订单并支持来样打版', fullWidth: true },
  ])
})
