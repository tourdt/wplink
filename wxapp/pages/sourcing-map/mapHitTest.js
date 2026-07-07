import { isRentableMapObject, isVerifiedMapObject } from './mapObjectState.js'

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
  const center = objectPoint(object)
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
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    return getPolygonBounds(geometry)
  }
  const center = objectPoint(object)
  if (!center) return null
  if (object.geometryType === 'point') {
    return { minX: center.x, minY: center.y, maxX: center.x, maxY: center.y }
  }
  const width = geometryPositiveNumber(geometry.width, 80)
  const height = geometryPositiveNumber(geometry.height, 50)
  if (width == null || height == null) return null
  return { minX: center.x, minY: center.y, maxX: center.x + width, maxY: center.y + height }
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
  const points = Array.isArray(geometry.points) ? geometry.points : []
  const normalized = []
  for (const point of points) {
    if (!point) return []
    const x = geometryNumber(point.x)
    const y = geometryNumber(point.y)
    if (x == null || y == null) return []
    normalized.push({ x, y })
  }
  return normalized
}

function getPolygonBounds(geometry = {}) {
  const points = normalizePolygonPoints(geometry)
  if (!points.length) {
    return null
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

function normalizeBounds(bounds = {}) {
  const minX = Number(bounds.minX)
  const minY = Number(bounds.minY)
  const maxX = Number(bounds.maxX)
  const maxY = Number(bounds.maxY)
  if (![minX, minY, maxX, maxY].every(Number.isFinite)) {
    return null
  }
  return {
    minX: Math.min(minX, maxX),
    minY: Math.min(minY, maxY),
    maxX: Math.max(minX, maxX),
    maxY: Math.max(minY, maxY),
  }
}

function objectIdentity(object) {
  return object?.id || object?.code || ''
}

function objectPoint(object = {}) {
  const geometry = object.geometry || {}
  const x = geometryNumber(geometry.x, object.centerX)
  const y = geometryNumber(geometry.y, object.centerY)
  if (x == null || y == null) return null
  return { x, y }
}

function normalizeMapPoint(mapPoint = {}) {
  const x = finiteNumber(mapPoint.x)
  const y = finiteNumber(mapPoint.y)
  if (x == null || y == null) return null
  return { x, y }
}

function geometryNumber(value, fallback) {
  // 命中测试不能把异常坐标兜底到原点，否则坏数据会在地图左上角被误选中。
  if (value !== undefined && value !== null) {
    return finiteNumber(value)
  }
  return finiteNumber(fallback)
}

function geometryPositiveNumber(value, fallback) {
  const parsed = geometryNumber(value, fallback)
  return parsed != null && parsed > 0 ? parsed : null
}

function finiteNumber(value) {
  if (typeof value === 'string' && value.trim() === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}
