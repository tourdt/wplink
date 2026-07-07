import assert from 'node:assert/strict'
import test from 'node:test'

import {
  mapObjectBounds,
  mapObjectCenter,
  normalizeBounds,
  normalizeMapPoint,
  normalizePolygonPoints,
} from './mapGeometry.js'

test('normalizes finite map object geometry for rect, point and polygon', () => {
  assert.deepEqual(
    mapObjectBounds({
      geometryType: 'rect',
      geometry: { x: '10', y: '20', width: '80', height: '50' },
    }),
    { minX: 10, minY: 20, maxX: 90, maxY: 70 },
  )
  assert.deepEqual(
    mapObjectCenter({
      geometryType: 'rect',
      geometry: { x: 10, y: 20, width: 80, height: 50 },
    }),
    { x: 50, y: 45 },
  )
  assert.deepEqual(
    mapObjectBounds({
      geometryType: 'point',
      geometry: { x: 30, y: 40 },
    }),
    { minX: 30, minY: 40, maxX: 30, maxY: 40 },
  )
  assert.deepEqual(
    normalizePolygonPoints({
      points: [
        { x: 0, y: 0 },
        { x: 100, y: 0 },
        { x: 80, y: 40 },
      ],
    }),
    [
      { x: 0, y: 0 },
      { x: 100, y: 0 },
      { x: 80, y: 40 },
    ],
  )
})

test('rejects non-finite geometry numbers instead of falling back to the origin', () => {
  assert.equal(
    mapObjectBounds({
      geometryType: 'rect',
      geometry: { x: Number.NaN, y: 20, width: 80, height: 50 },
    }),
    null,
  )
  assert.equal(
    mapObjectCenter({
      geometryType: 'point',
      geometry: { x: Number.POSITIVE_INFINITY, y: 40 },
    }),
    null,
  )
  assert.deepEqual(normalizePolygonPoints({ points: [{ x: 0, y: 0 }, { x: Number.NaN, y: 1 }, { x: 2, y: 2 }] }), [])
})

test('rejects incomplete canonical geometry instead of historical fallbacks', () => {
  assert.equal(
    mapObjectCenter({
      geometryType: 'point',
      centerX: 30,
      centerY: 40,
      geometry: {},
    }),
    null,
  )
  assert.equal(
    mapObjectBounds({
      geometryType: 'rect',
      geometry: { x: 10, y: 20 },
    }),
    null,
  )
  assert.equal(
    mapObjectCenter({
      geometryType: 'rect',
      geometry: { x: 10, y: 20, width: 80 },
    }),
    null,
  )
})

test('normalizes viewport points and bounds with finite numbers only', () => {
  assert.deepEqual(normalizeMapPoint({ x: '12', y: '34' }), { x: 12, y: 34 })
  assert.equal(normalizeMapPoint({ x: '', y: 34 }), null)
  assert.deepEqual(normalizeBounds({ minX: 100, minY: 20, maxX: 10, maxY: 80 }), {
    minX: 10,
    minY: 20,
    maxX: 100,
    maxY: 80,
  })
  assert.equal(normalizeBounds({ minX: 0, minY: Number.NaN, maxX: 100, maxY: 100 }), null)
})
