import { isRentableMapObject, isVerifiedMapObject, mapObjectIdentity } from './mapObjectState.js'
import {
  mapObjectBounds,
  mapObjectCenter,
  normalizeBounds,
  normalizeMapPoint,
  normalizePolygonPoints,
} from './mapGeometry.js'

const DEFAULT_POINT_RADIUS = 18

export function hitTestMapObjects(objects = [], mapPoint = {}, options = {}) {
  const hits = objects
    .map((object, index) => ({ object, index, bounds: getObjectBounds(object) }))
    .filter((entry) => entry.bounds && isClickableMapObject(entry.object) && isObjectHit(entry.object, mapPoint, options.pointRadius))
    .sort((left, right) => compareHitPriority(left, right, options))
  return hits[0]?.object || null
}

export function isPointInRect(mapPoint = {}, object = {}) {
  const bounds = getObjectBounds(object)
  if (!bounds) return false
  const point = normalizeMapPoint(mapPoint)
  if (!point) return false
  return (
    point.x >= bounds.minX &&
    point.x <= bounds.maxX &&
    point.y >= bounds.minY &&
    point.y <= bounds.maxY
  )
}

export function isPointNearPoint(mapPoint = {}, object = {}, radius = DEFAULT_POINT_RADIUS) {
  const point = normalizeMapPoint(mapPoint)
  const center = mapObjectCenter(object)
  if (!point || !center) return false
  return Math.hypot(point.x - center.x, point.y - center.y) <= radius
}

export function isPointInPolygon(mapPoint = {}, object = {}) {
  const points = normalizePolygonPoints(object.geometry || object)
  const point = normalizeMapPoint(mapPoint)
  if (!point) return false
  if (points.length < 3) return false

  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i, i += 1) {
    const current = points[i]
    const previous = points[j]
    const intersects =
      current.y > point.y !== previous.y > point.y &&
      point.x < ((previous.x - current.x) * (point.y - current.y)) / (previous.y - current.y || 1) + current.x
    if (intersects) inside = !inside
  }
  return inside
}

export function isObjectInBounds(object = {}, bounds = {}) {
  if (!object) return false
  const viewportBounds = normalizeBounds(bounds)
  if (!viewportBounds) return false
  const objectBounds = getObjectBounds(object)
  if (!objectBounds) return false
  return (
    objectBounds.maxX >= viewportBounds.minX &&
    objectBounds.minX <= viewportBounds.maxX &&
    objectBounds.maxY >= viewportBounds.minY &&
    objectBounds.minY <= viewportBounds.maxY
  )
}

export function getObjectBounds(object = {}) {
  return mapObjectBounds(object)
}

function isObjectHit(object, mapPoint, pointRadius = DEFAULT_POINT_RADIUS) {
  if (!object) return false
  if (object.geometryType === 'polygon') return isPointInPolygon(mapPoint, object)
  if (object.geometryType === 'point') return isPointNearPoint(mapPoint, object, pointRadius)
  return isPointInRect(mapPoint, object)
}

function compareHitPriority(left, right, options = {}) {
  const leftScore = hitPriorityScore(left, options)
  const rightScore = hitPriorityScore(right, options)
  if (leftScore !== rightScore) return rightScore - leftScore

  const leftArea = boundsArea(left.bounds)
  const rightArea = boundsArea(right.bounds)
  if (leftArea !== rightArea) return leftArea - rightArea

  return left.index - right.index
}

function hitPriorityScore(entry, options = {}) {
  let score = 0
  const identity = mapObjectIdentity(entry.object)
  if (identity && identity === options.selectedObjectId) score += 100
  if (isVerifiedMapObject(entry.object)) score += 50
  if (isRentableMapObject(entry.object)) score += 40
  if (entry.object?.geometryType === 'point') score += 8
  return score
}

function isClickableMapObject(object) {
  if (!object) return false
  if (object.geometryType === 'point') return true
  return Boolean(isVerifiedMapObject(object) || isRentableMapObject(object))
}

function boundsArea(bounds) {
  return Math.max(0, bounds.maxX - bounds.minX) * Math.max(0, bounds.maxY - bounds.minY)
}
