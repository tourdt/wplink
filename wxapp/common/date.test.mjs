import assert from 'node:assert/strict'
import test from 'node:test'

import { formatDateToDay } from './date.js'

test('formats timestamp to day without time or timezone', () => {
  assert.equal(formatDateToDay('2026-06-30T10:24:00+08:00'), '2026-06-30')
})

test('keeps day value unchanged', () => {
  assert.equal(formatDateToDay('2026-06-30'), '2026-06-30')
})

test('returns placeholder for empty date', () => {
  assert.equal(formatDateToDay(''), '-')
})
