import { isRentableMapObject, isVerifiedMapObject } from './mapObjectState.js'

const DEFAULT_POINT_RADIUS = 18

export function hitTestMapObjects(objects = [], mapPoint = {}, options = {}) {
  const hits = objects
    .map((object, index) => ({ object, index, bounds: getObjectBounds(object) }))
    .filter((entry) => isClickableMapObject(entry.object) && isObjectHit(entry.object, mapPoint, options.pointRadius))
    .sort((left, right) => compareHitPriority(left, right, options))
  return hits[0]?.object || null
}

export function isPointInRect(mapPoint = {}, object = {}) {
  const bounds = getObjectBounds(object)
  return (
    mapPoint.x >= bounds.minX &&
    mapPoint.x <= bounds.maxX &&
    mapPoint.y >= bounds.minY &&
    mapPoint.y <= bounds.maxY
  )
}

export function isPointNearPoint(mapPoint = {}, object = {}, radius = DEFAULT_POINT_RADIUS) {
  const geometry = object.geometry || {}
  const x = toNumber(geometry.x, toNumber(object.centerX, 0))
  const y = toNumber(geometry.y, toNumber(object.centerY, 0))
  return Math.hypot(toNumber(mapPoint.x, 0) - x, toNumber(mapPoint.y, 0) - y) <= radius
}

export function isPointInPolygon(mapPoint = {}, object = {}) {
  const points = normalizePolygonPoints(object.geometry || object)
  if (points.length < 3) return false

  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i, i += 1) {
    const current = points[i]
    const previous = points[j]
    const intersects =
      current.y > mapPoint.y !== previous.y > mapPoint.y &&
      mapPoint.x < ((previous.x - current.x) * (mapPoint.y - current.y)) / (previous.y - current.y || 1) + current.x
    if (intersects) inside = !inside
  }
  return inside
}

export function getObjectBounds(object = {}) {
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    return getPolygonBounds(geometry)
  }
  const x = toNumber(geometry.x, toNumber(object.centerX, 0))
  const y = toNumber(geometry.y, toNumber(object.centerY, 0))
  if (object.geometryType === 'point') {
    return { minX: x, minY: y, maxX: x, maxY: y }
  }
  const width = toPositiveNumber(geometry.width, 80)
  const height = toPositiveNumber(geometry.height, 50)
  return { minX: x, minY: y, maxX: x + width, maxY: y + height }
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
  const identity = objectIdentity(entry.object)
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

function normalizePolygonPoints(geometry = {}) {
  return (Array.isArray(geometry.points) ? geometry.points : [])
    .filter(Boolean)
    .map((point) => ({
      x: toNumber(point.x, 0),
      y: toNumber(point.y, 0),
    }))
}

function getPolygonBounds(geometry = {}) {
  const points = normalizePolygonPoints(geometry)
  if (!points.length) {
    return { minX: 0, minY: 0, maxX: 0, maxY: 0 }
  }
  return points.reduce(
    (bounds, point) => ({
      minX: Math.min(bounds.minX, point.x),
      minY: Math.min(bounds.minY, point.y),
      maxX: Math.max(bounds.maxX, point.x),
      maxY: Math.max(bounds.maxY, point.y),
    }),
    {
      minX: points[0].x,
      minY: points[0].y,
      maxX: points[0].x,
      maxY: points[0].y,
    },
  )
}

function objectIdentity(object) {
  return object?.id || object?.code || ''
}

function toNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPositiveNumber(value, fallback) {
  const parsed = toNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}
