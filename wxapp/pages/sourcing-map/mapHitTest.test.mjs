import assert from 'node:assert/strict'
import test from 'node:test'
import {
  hitTestMapObjects,
  isPointInPolygon,
  isPointInRect,
  isPointNearPoint,
} from './mapHitTest.js'

const rectObject = {
  id: 'rect-1',
  geometryType: 'rect',
  geometry: { x: 10, y: 20, width: 120, height: 80 },
}

const pointObject = {
  id: 'point-1',
  geometryType: 'point',
  geometry: { x: 240, y: 160 },
}

const polygonObject = {
  id: 'polygon-1',
  geometryType: 'polygon',
  geometry: {
    points: [
      { x: 300, y: 100 },
      { x: 380, y: 120 },
      { x: 350, y: 220 },
      { x: 280, y: 180 },
    ],
  },
}

test('rect hit testing uses object bounds', () => {
  assert.equal(isPointInRect({ x: 40, y: 60 }, rectObject), true)
  assert.equal(isPointInRect({ x: 4, y: 60 }, rectObject), false)
})

test('point hit testing uses a configurable radius', () => {
  assert.equal(isPointNearPoint({ x: 250, y: 168 }, pointObject, 16), true)
  assert.equal(isPointNearPoint({ x: 270, y: 190 }, pointObject, 16), false)
})

test('polygon hit testing supports irregular areas', () => {
  assert.equal(isPointInPolygon({ x: 330, y: 155 }, polygonObject), true)
  assert.equal(isPointInPolygon({ x: 260, y: 120 }, polygonObject), false)
})

test('hitTestMapObjects returns the top priority hit object', () => {
  const objects = [
    {
      id: 'large',
      platformTags: ['rentable'],
      geometryType: 'rect',
      geometry: { x: 0, y: 0, width: 200, height: 200 },
    },
    {
      id: 'verified',
      isVerifiedMerchant: true,
      geometryType: 'rect',
      geometry: { x: 20, y: 20, width: 160, height: 160 },
    },
    {
      id: 'selected',
      platformTags: ['rentable'],
      geometryType: 'rect',
      geometry: { x: 40, y: 40, width: 120, height: 120 },
    },
  ]

  const hitObject = hitTestMapObjects(objects, { x: 80, y: 80 }, { selectedObjectId: 'selected' })
  assert.equal(hitObject.id, 'selected')
})

test('hitTestMapObjects falls back to smaller geometry when priorities tie', () => {
  const objects = [
    {
      id: 'large',
      platformTags: ['rentable'],
      geometryType: 'rect',
      geometry: { x: 0, y: 0, width: 200, height: 200 },
    },
    {
      id: 'small',
      platformTags: ['rentable'],
      geometryType: 'rect',
      geometry: { x: 60, y: 60, width: 40, height: 40 },
    },
  ]

  const hitObject = hitTestMapObjects(objects, { x: 80, y: 80 })
  assert.equal(hitObject.id, 'small')
})

test('hitTestMapObjects ignores ordinary booth outlines but keeps verified and rentable booths clickable', () => {
  const ordinaryBooth = {
    id: 'ordinary-booth',
    geometryType: 'rect',
    geometry: { x: 0, y: 0, width: 80, height: 60 },
  }
  const verifiedBooth = {
    id: 'verified-booth',
    isVerifiedMerchant: true,
    geometryType: 'rect',
    geometry: { x: 100, y: 0, width: 80, height: 60 },
  }
  const rentableBooth = {
    id: 'rentable-booth',
    platformTags: ['rentable'],
    geometryType: 'rect',
    geometry: { x: 200, y: 0, width: 80, height: 60 },
  }

  assert.equal(hitTestMapObjects([ordinaryBooth], { x: 30, y: 30 }), null)
  assert.equal(hitTestMapObjects([verifiedBooth], { x: 130, y: 30 })?.id, 'verified-booth')
  assert.equal(hitTestMapObjects([rentableBooth], { x: 230, y: 30 })?.id, 'rentable-booth')
})

test('hitTestMapObjects keeps point POIs clickable after booth click filtering', () => {
  const poiObject = {
    id: 'poi-1',
    geometryType: 'point',
    geometry: { x: 50, y: 50 },
  }

  assert.equal(hitTestMapObjects([poiObject], { x: 56, y: 56 })?.id, 'poi-1')
})
