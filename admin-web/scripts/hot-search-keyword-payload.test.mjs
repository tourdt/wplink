import assert from 'node:assert/strict'
import test from 'node:test'

import { buildHotSearchKeywordPayload } from '../src/api/hotSearchKeywordPayload.js'

test('hot search keyword save payload omits readonly row fields', () => {
  const payload = buildHotSearchKeywordPayload({
    id: '65620560230482100',
    cityCode: 'zhili',
    keyword: 'summer stock',
    startAt: '',
    endAt: '',
    sortOrder: 20,
    status: 'active',
    updatedAt: '2026-07-01T10:00:00+08:00',
  })

  assert.deepEqual(payload, {
    cityCode: 'zhili',
    keyword: 'summer stock',
    startAt: '',
    endAt: '',
    sortOrder: 20,
    status: 'active',
  })
  assert.equal(Object.hasOwn(payload, 'id'), false)
  assert.equal(Object.hasOwn(payload, 'updatedAt'), false)
})
