import assert from 'node:assert/strict'
import test from 'node:test'

import { buildViewportBounds, mapCenterFromSize, mapPointFromClientPoint, normalizeMapZoom, scaledMapSize } from './mapViewport.js'

test('normalizes map zoom to supported range', () => {
  assert.equal(normalizeMapZoom(0.1), 0.25)
  assert.equal(normalizeMapZoom(1.234), 1.23)
  assert.equal(normalizeMapZoom(4), 3)
})

test('converts client point to original map pixel coordinates under zoom', () => {
  const point = mapPointFromClientPoint({
    clientX: 260,
    clientY: 180,
    stageRect: { left: 20, top: 30 },
    scale: 2,
  })

  assert.deepEqual(point, { x: 120, y: 75 })
})

test('builds viewport bounds in original map pixels under zoom', () => {
  const bounds = buildViewportBounds({
    scrollLeft: 400,
    scrollTop: 160,
    clientWidth: 800,
    clientHeight: 500,
    mapWidth: 3000,
    mapHeight: 1800,
    scale: 2,
    paddingRatio: 0.25,
  })

  assert.deepEqual(bounds, {
    minX: 100,
    minY: 18,
    maxX: 700,
    maxY: 393,
  })
})

test('calculates scaled map stage size from original dimensions', () => {
  assert.deepEqual(scaledMapSize({ width: 3000, height: 1800, scale: 0.5 }), {
    width: 1500,
    height: 900,
  })
})

test('calculates default scene center from background dimensions', () => {
  assert.deepEqual(mapCenterFromSize({ width: 3000, height: 1800 }), {
    x: 1500,
    y: 900,
  })
})
