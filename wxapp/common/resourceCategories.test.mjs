import assert from 'node:assert/strict'
import test from 'node:test'

import {
  flattenGroupedResourceTypes,
  groupResourceTypes,
  resourceTypeLabel,
} from './resourceCategories.js'

test('groupResourceTypes groups category items by configured primary category', () => {
  const groups = groupResourceTypes([
    {
      typeCode: 'stock_clearance',
      typeName: '尾货/库存出售',
      direction: 'supply',
      displayTemplate: { group: { code: 'kids_wholesale', name: '童装批发', sort: 10 } },
    },
    {
      typeCode: 'seek_factory_warehouse',
      typeName: '求租求购厂房仓库',
      direction: 'demand',
      displayTemplate: { group: { code: 'factory_warehouse', name: '厂房仓库', sort: 70 } },
    },
    {
      typeCode: 'legacy_misc',
      typeName: '临时类目',
      direction: 'supply',
      displayTemplate: {},
    },
  ])

  assert.deepEqual(groups.map((group) => group.code), ['kids_wholesale', 'factory_warehouse', 'other'])
  assert.equal(groups[0].items[0].label, '尾货/库存出售')
  assert.equal(groups[1].items[0].direction, 'demand')
  assert.equal(groups[2].name, '其他类目')
})

test('flattenGroupedResourceTypes keeps group names in picker labels', () => {
  const groups = groupResourceTypes([
    {
      typeCode: 'job_hiring',
      typeName: '招聘员工',
      direction: 'demand',
      displayTemplate: { group: { code: 'jobs', name: '招聘求职', sort: 40 } },
    },
  ])

  assert.deepEqual(flattenGroupedResourceTypes(groups).map((item) => item.label), ['招聘求职 / 招聘员工'])
})

test('resourceTypeLabel prefers dynamic type name over fallback map', () => {
  assert.equal(resourceTypeLabel({ typeCode: 'stock_clearance', typeName: '尾货/库存出售' }), '尾货/库存出售')
  assert.equal(resourceTypeLabel({ typeCode: 'unknown', typeName: '' }), '')
})
