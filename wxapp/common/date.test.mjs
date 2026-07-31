import assert from 'node:assert/strict'
import test from 'node:test'

import { formatDateToDay, formatListFreshnessDate } from './date.js'

process.env.TZ = 'Asia/Shanghai'

test('formats timestamp to day without time or timezone', () => {
  assert.equal(formatDateToDay('2026-06-30T10:24:00+08:00'), '2026-06-30')
})

test('keeps day value unchanged', () => {
  assert.equal(formatDateToDay('2026-06-30'), '2026-06-30')
})

test('returns placeholder for empty date', () => {
  assert.equal(formatDateToDay(''), '-')
})

test('formats list freshness date as today or yesterday', () => {
  const now = new Date('2026-06-30T12:00:00+08:00')

  assert.equal(formatListFreshnessDate('2026-06-30T08:24:00+08:00', now), '今天')
  assert.equal(formatListFreshnessDate('2026-06-29T22:30:00+08:00', now), '昨天')
})

test('formats RFC3339 freshness in local timezone across a UTC day boundary', () => {
  const now = new Date('2026-08-01T01:00:00+08:00')

  assert.equal(formatListFreshnessDate('2026-07-31T16:30:00Z', now), '今天')
})

test('formats older list freshness date as month and day', () => {
  const now = new Date('2026-06-30T12:00:00+08:00')

  assert.equal(formatListFreshnessDate('2026-06-28T10:24:00+08:00', now), '06-28')
  assert.equal(formatListFreshnessDate('2025-12-31', now), '12-31')
})

test('uses placeholder for empty list freshness date', () => {
  assert.equal(formatListFreshnessDate(''), '近期')
})
