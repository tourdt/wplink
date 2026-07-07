import { isRentableMapObject, isVerifiedMapObject } from './mapObjectState.js'

const COLORS = {
  background: '#eef3f8',
  mapFallback: '#e7edf5',
  boothFill: 'rgba(31, 92, 154, 0.20)',
  boothStroke: '#1f5c9a',
  boothOutlineFill: 'rgba(255, 255, 255, 0.12)',
  boothOutlineStroke: 'rgba(100, 116, 139, 0.42)',
  poiFill: '#f59e0b',
  poiStroke: '#b45309',
  verifiedFill: 'rgba(22, 163, 74, 0.24)',
  verifiedStroke: '#15803d',
  verifiedMarkerFill: '#16a34a',
  verifiedMarkerSoft: 'rgba(22, 163, 74, 0.16)',
  verifiedMarkerBorder: 'rgba(22, 163, 74, 0.62)',
  rentableFill: 'rgba(254, 243, 199, 0.82)',
  rentableStroke: '#b45309',
  rentableText: '#92400e',
  weakFill: 'rgba(100, 116, 139, 0.18)',
  weakStroke: '#64748b',
  selectedStroke: '#16a34a',
  text: '#0f172a',
  mutedText: '#64748b',
  badgeFill: '#16a34a',
}

const VERIFIED_LABEL_ZOOM_LEVEL = 5
const VERIFIED_PIN_ZOOM_LEVEL = 4
const VERIFIED_OVERVIEW_ZOOM_LEVEL = 3

export function createSourcingMapRenderer(options = {}) {
  let ctx = null
  let scene = null
  let objects = []
  let sortedObjects = []
  let selectedObject = null
  let width = 0
  let height = 0
  let lastBackgroundDrawn = false
  const backgroundPathCache = new Map()
  const pendingBackgroundUrls = new Set()
  const failedBackgroundUrls = new Set()

  function init(initOptions = {}) {
    width = toPositiveNumber(initOptions.width, width || 375)
    height = toPositiveNumber(initOptions.height, height || 500)
    if (typeof uni === 'undefined' || typeof uni.createCanvasContext !== 'function') {
      return false
    }

    try {
      ctx = uni.createCanvasContext(initOptions.canvasId, initOptions.component)
      return Boolean(ctx)
    } catch {
      ctx = null
      return false
    }
  }

  function setScene(nextScene) {
    scene = nextScene || null
    ensureBackgroundImage(scene?.backgroundUrl)
  }

  function setObjects(nextObjects = []) {
    const normalizedObjects = Array.isArray(nextObjects) ? nextObjects : []
    if (normalizedObjects === objects) return
    objects = normalizedObjects
    sortedObjects = [...objects].sort(compareObjectPaintOrder)
  }

  function setSelectedObject(nextObject) {
    selectedObject = nextObject || null
  }

  function render(transform = {}, renderOptions = {}) {
    if (!ctx) return false
    width = toPositiveNumber(renderOptions.width, width || 375)
    height = toPositiveNumber(renderOptions.height, height || 500)
    const metrics = buildRenderMetrics(scene, transform, { ...renderOptions, selectedObject, backgroundPathCache }, width, height)
    let backgroundDrawn = false

    clearCanvas(ctx, width, height)
    if (renderOptions.drawBackground === 'whenReady') {
      backgroundDrawn = drawMapBackground(ctx, scene, metrics, { onlyWhenReady: true })
    } else if (renderOptions.drawBackground !== false) {
      backgroundDrawn = drawMapBackground(ctx, scene, metrics)
    }
    lastBackgroundDrawn = backgroundDrawn

    for (const object of sortedObjects) {
      drawMapObject(ctx, object, metrics, renderOptions)
    }

    for (const object of sortedObjects) {
      if (shouldDrawObjectLabel(object, renderOptions, selectedObject)) {
        drawObjectLabel(ctx, object, metrics, renderOptions, options)
      }
    }

    if (selectedObject) {
      drawSelectedOutline(ctx, selectedObject, metrics)
    }

    drawCanvas(ctx, () => {
      if (typeof renderOptions.onDrawComplete === 'function') {
        renderOptions.onDrawComplete({ backgroundDrawn })
      }
    })
    return true
  }

  function hasBackgroundImage() {
    return Boolean(scene?.backgroundUrl && backgroundPathCache.has(scene.backgroundUrl))
  }

  function hasRenderedBackground() {
    return lastBackgroundDrawn
  }

  function dispose() {
    ctx = null
    scene = null
    objects = []
    sortedObjects = []
    selectedObject = null
    lastBackgroundDrawn = false
    pendingBackgroundUrls.clear()
    failedBackgroundUrls.clear()
  }

  function ensureBackgroundImage(url) {
    if (!url || backgroundPathCache.has(url) || pendingBackgroundUrls.has(url) || failedBackgroundUrls.has(url)) return
    if (typeof uni === 'undefined' || typeof uni.getImageInfo !== 'function') {
      backgroundPathCache.set(url, url)
      return
    }
    pendingBackgroundUrls.add(url)
    uni.getImageInfo({
      src: url,
      success(result) {
        backgroundPathCache.set(url, result.path || url)
        if (typeof options.onAssetReady === 'function') {
          options.onAssetReady()
        }
      },
      fail() {
        failedBackgroundUrls.add(url)
        if (typeof options.onAssetReady === 'function') {
          options.onAssetReady()
        }
      },
      complete() {
        pendingBackgroundUrls.delete(url)
      },
    })
  }

  return {
    init,
    setScene,
    setObjects,
    setSelectedObject,
    render,
    hasBackgroundImage,
    hasRenderedBackground,
    dispose,
  }
}

function drawMapObject(ctx, object, metrics, options) {
  if (!objectBounds(object)) return
  if (object?.geometryType === 'polygon') {
    drawPolygonObject(ctx, object, metrics)
    return
  }
  if (object?.geometryType === 'point') {
    drawPointObject(ctx, object, metrics)
    return
  }
  drawRectObject(ctx, object, metrics, options)
}

function drawRectObject(ctx, object, metrics) {
  const bounds = objectBounds(object)
  if (!bounds) return
  const x = projectX(bounds.minX, metrics)
  const y = projectY(bounds.minY, metrics)
  const width = projectSize(bounds.maxX - bounds.minX, metrics)
  const height = projectSize(bounds.maxY - bounds.minY, metrics)

  if (shouldDrawVerifiedOverviewPoint(object, metrics)) {
    drawVerifiedOverviewPoint(ctx, object, metrics)
    return
  }

  drawBoothOutline(ctx, x, y, width, height, object)

  if (isRentableObject(object)) {
    drawRentableBooth(ctx, object, metrics, { x, y, width, height })
    return
  }

  if (isVerifiedObject(object)) {
    drawVerifiedBoothMarker(ctx, object, metrics, { x, y, width, height })
  }
}

function drawBoothOutline(ctx, x, y, width, height, object) {
  const outlineFill = object?.displayLevel === 'weak' ? COLORS.weakFill : COLORS.boothOutlineFill
  const outlineStroke = object?.displayLevel === 'weak' ? COLORS.weakStroke : COLORS.boothOutlineStroke

  setFillStyle(ctx, outlineFill)
  setStrokeStyle(ctx, outlineStroke)
  setLineWidth(ctx, 1)
  fillRect(ctx, x, y, width, height)
  strokeRect(ctx, x, y, width, height)
}

function drawRentableBooth(ctx, object, metrics, rect) {
  setFillStyle(ctx, COLORS.rentableFill)
  setStrokeStyle(ctx, COLORS.rentableStroke)
  setLineWidth(ctx, 2)
  setLineDash(ctx, [5, 4])
  fillRect(ctx, rect.x, rect.y, rect.width, rect.height)
  strokeRect(ctx, rect.x, rect.y, rect.width, rect.height)
  setLineDash(ctx, [])

  if (toPositiveNumber(metrics.zoomLevel, VERIFIED_LABEL_ZOOM_LEVEL) < VERIFIED_PIN_ZOOM_LEVEL && !isSameObject(object, metrics.selectedObject)) return

  const label = '出租'
  const fontSize = Math.max(10, Math.min(12, 11 * metrics.scale))
  setFillStyle(ctx, COLORS.rentableText)
  setFontSize(ctx, fontSize)
  setTextAlign(ctx, 'center')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, label, rect.x + rect.width / 2, rect.y + rect.height / 2, rect.width)
}

function drawVerifiedBoothMarker(ctx, object, metrics, rect) {
  const zoomLevel = toPositiveNumber(metrics.zoomLevel, VERIFIED_LABEL_ZOOM_LEVEL)
  const center = objectCenter(object)
  if (!center) return
  const markerX = projectX(center.x, metrics)
  const markerY = projectY(center.y, metrics)
  const selected = isSameObject(object, metrics.selectedObject)

  if (zoomLevel < VERIFIED_PIN_ZOOM_LEVEL && !selected) return
  if (zoomLevel < VERIFIED_LABEL_ZOOM_LEVEL && !selected) {
    drawVerifiedPin(ctx, markerX, markerY)
    return
  }

  drawVerifiedLabelMarker(ctx, object, metrics, rect)
}

function drawPointObject(ctx, object, metrics) {
  if (shouldDrawVerifiedOverviewPoint(object, metrics)) {
    drawVerifiedOverviewPoint(ctx, object, metrics)
    return
  }

  const center = objectCenter(object)
  if (!center) return
  const x = projectX(center.x, metrics)
  const y = projectY(center.y, metrics)
  const radius = Math.max(5, Math.min(12, 7 * metrics.scale))
  const palette = objectPalette(object)

  beginPath(ctx)
  arc(ctx, x, y, radius, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, object.geometryType === 'point' && !isVerifiedObject(object) ? COLORS.poiFill : palette.fill)
  fill(ctx)
  setStrokeStyle(ctx, palette.stroke)
  setLineWidth(ctx, 2)
  stroke(ctx)

  if (isVerifiedObject(object)) {
    drawVerifiedBadge(ctx, x + radius + 3, y - radius - 3)
  }
}

function drawPolygonObject(ctx, object, metrics) {
  const points = normalizePolygonPoints(object.geometry)
  if (points.length < 3) return

  if (shouldDrawVerifiedOverviewPoint(object, metrics)) {
    drawVerifiedOverviewPoint(ctx, object, metrics)
    return
  }

  beginPath(ctx)
  fillPolygonPath(ctx, points, metrics)
  closePath(ctx)
  setFillStyle(ctx, object?.displayLevel === 'weak' ? COLORS.weakFill : COLORS.boothOutlineFill)
  fill(ctx)
  setStrokeStyle(ctx, object?.displayLevel === 'weak' ? COLORS.weakStroke : COLORS.boothOutlineStroke)
  setLineWidth(ctx, 1)
  stroke(ctx)

  if (isRentableObject(object)) {
    drawRentablePolygonBadge(ctx, object, metrics)
    return
  }

  if (isVerifiedObject(object)) {
    drawVerifiedBoothMarker(ctx, object, metrics, projectedObjectBounds(object, metrics))
  }
}

function fillPolygonPath(ctx, points, metrics) {
  points.forEach((point, index) => {
    const x = projectX(point.x, metrics)
    const y = projectY(point.y, metrics)
    if (index === 0) {
      moveTo(ctx, x, y)
    } else {
      lineTo(ctx, x, y)
    }
  })
}

function drawObjectLabel(ctx, object, metrics, renderOptions, options) {
  if (metrics.scale < 0.85) return
  const label = options.getObjectLabel ? options.getObjectLabel(object, renderOptions.zoomLevel) : object.name || object.code || ''
  if (!label) return

  const center = objectCenter(object)
  if (!center) return
  const x = projectX(center.x, metrics)
  const y = projectY(center.y, metrics)
  const maxWidth = Math.max(36, objectWidth(object) * metrics.baseScale * metrics.scale - 8)
  const fontSize = Math.max(10, Math.min(13, 11 * metrics.scale))

  setFillStyle(ctx, COLORS.text)
  setFontSize(ctx, fontSize)
  setTextAlign(ctx, 'center')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, truncateLabel(label, maxWidth, fontSize), x, y, maxWidth)
}

function shouldDrawObjectLabel(object, renderOptions = {}, selectedObject = null) {
  if (isVerifiedObject(object) || isRentableObject(object)) return false
  if (!isSameObject(object, selectedObject)) return false
  if (!renderOptions.interacting) return true
  return isSameObject(object, selectedObject)
}

function drawSelectedOutline(ctx, object, metrics) {
  if (!object) return
  setStrokeStyle(ctx, COLORS.selectedStroke)
  setLineWidth(ctx, 3)

  if (object.geometryType === 'polygon') {
    const points = normalizePolygonPoints(object.geometry)
    if (points.length < 3) return
    beginPath(ctx)
    fillPolygonPath(ctx, points, metrics)
    closePath(ctx)
    stroke(ctx)
    return
  }

  const bounds = objectBounds(object)
  if (!bounds) return
  const x = projectX(bounds.minX, metrics)
  const y = projectY(bounds.minY, metrics)
  const width = projectSize(bounds.maxX - bounds.minX, metrics)
  const height = projectSize(bounds.maxY - bounds.minY, metrics)
  if (object.geometryType === 'point') {
    beginPath(ctx)
    arc(ctx, x, y, Math.max(11, 12 * metrics.scale), 0, Math.PI * 2)
    closePath(ctx)
    stroke(ctx)
    return
  }
  strokeRect(ctx, x - 2, y - 2, width + 4, height + 4)
}

function drawRentablePolygonBadge(ctx, object, metrics) {
  if (toPositiveNumber(metrics.zoomLevel, VERIFIED_LABEL_ZOOM_LEVEL) < VERIFIED_PIN_ZOOM_LEVEL && !isSameObject(object, metrics.selectedObject)) return

  const center = objectCenter(object)
  if (!center) return
  const x = projectX(center.x, metrics)
  const y = projectY(center.y, metrics)
  setFillStyle(ctx, COLORS.rentableFill)
  setStrokeStyle(ctx, COLORS.rentableStroke)
  setLineWidth(ctx, 2)
  fillRect(ctx, x - 18, y - 10, 36, 20)
  strokeRect(ctx, x - 18, y - 10, 36, 20)
  setFillStyle(ctx, COLORS.rentableText)
  setFontSize(ctx, 11)
  setTextAlign(ctx, 'center')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, '出租', x, y, 34)
}

function drawVerifiedBadge(ctx, x, y) {
  const width = 22
  const height = 12
  setFillStyle(ctx, COLORS.badgeFill)
  fillRect(ctx, x, y, width, height)
  setFillStyle(ctx, '#ffffff')
  setFontSize(ctx, 8)
  setTextAlign(ctx, 'center')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, '认', x + width / 2, y + height / 2 + 0.5, width)
}

function drawVerifiedPin(ctx, x, y) {
  const radius = 8
  beginPath(ctx)
  arc(ctx, x, y, radius + 4, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, COLORS.verifiedMarkerSoft)
  fill(ctx)

  beginPath(ctx)
  arc(ctx, x, y, radius, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, COLORS.verifiedMarkerFill)
  fill(ctx)

  beginPath(ctx)
  arc(ctx, x, y, 3, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, '#ffffff')
  fill(ctx)
}

function drawVerifiedOverviewPoint(ctx, object, metrics) {
  const center = objectCenter(object)
  if (!center) return
  const x = projectX(center.x, metrics)
  const y = projectY(center.y, metrics)
  const radius = Math.max(4, Math.min(7, 5 * metrics.scale))

  beginPath(ctx)
  arc(ctx, x, y, radius, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, COLORS.verifiedMarkerFill)
  fill(ctx)
  setStrokeStyle(ctx, COLORS.verifiedStroke)
  setLineWidth(ctx, 1.5)
  stroke(ctx)
}

function drawVerifiedLabelMarker(ctx, object, metrics, rect) {
  const label = objectLabelText(object)
  if (!label) {
    drawVerifiedPin(ctx, rect.x + rect.width / 2, rect.y + rect.height / 2)
    return
  }

  const fontSize = 11
  const markerHeight = 28
  const markerWidth = Math.max(84, Math.min(136, 44 + label.length * fontSize))
  const markerX = Math.max(6, rect.x + rect.width / 2 - markerWidth / 2)
  const markerY = Math.max(6, rect.y - markerHeight - 8)
  const logoRadius = 10
  const logoX = markerX + 17
  const logoY = markerY + markerHeight / 2

  setFillStyle(ctx, 'rgba(255, 255, 255, 0.96)')
  setStrokeStyle(ctx, COLORS.verifiedMarkerBorder)
  setLineWidth(ctx, 1)
  fillRect(ctx, markerX, markerY, markerWidth, markerHeight)
  strokeRect(ctx, markerX, markerY, markerWidth, markerHeight)

  beginPath(ctx)
  arc(ctx, logoX, logoY, logoRadius, 0, Math.PI * 2)
  closePath(ctx)
  setFillStyle(ctx, COLORS.verifiedMarkerFill)
  fill(ctx)

  setFillStyle(ctx, '#ffffff')
  setFontSize(ctx, 10)
  setTextAlign(ctx, 'center')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, logoText(object), logoX, logoY + 0.5, logoRadius * 2)

  setFillStyle(ctx, COLORS.text)
  setFontSize(ctx, fontSize)
  setTextAlign(ctx, 'left')
  setTextBaseline(ctx, 'middle')
  fillText(ctx, truncateLabel(label, markerWidth - 40, fontSize), markerX + 34, logoY, markerWidth - 40)
}

function drawMapBackground(ctx, scene, metrics, options = {}) {
  const mapWidth = toPositiveNumber(scene?.width, metrics.viewportWidth / metrics.baseScale) * metrics.baseScale * metrics.scale
  const mapHeight = toPositiveNumber(scene?.height, metrics.viewportHeight / metrics.baseScale) * metrics.baseScale * metrics.scale
  const imagePath = scene?.backgroundUrl ? metrics.backgroundPathCache.get(scene.backgroundUrl) : ''

  if (imagePath && typeof ctx.drawImage === 'function') {
    try {
      ctx.drawImage(imagePath, metrics.offsetX, metrics.offsetY, mapWidth, mapHeight)
      return true
    } catch {
      if (options.onlyWhenReady) return false
    }
  }

  if (options.onlyWhenReady) return false

  setFillStyle(ctx, COLORS.background)
  fillRect(ctx, 0, 0, metrics.viewportWidth, metrics.viewportHeight)
  setFillStyle(ctx, COLORS.mapFallback)
  fillRect(ctx, metrics.offsetX, metrics.offsetY, mapWidth, mapHeight)
  return true
}

function clearCanvas(ctx, width, height) {
  if (typeof ctx.clearRect === 'function') {
    ctx.clearRect(0, 0, width, height)
  }
}

function buildRenderMetrics(scene, transform, options, width, height) {
  const sceneWidth = toPositiveNumber(scene?.width, width)
  const baseScale = toPositiveNumber(options.baseScale, width / sceneWidth)
  return {
    viewportWidth: width,
    viewportHeight: height,
    baseScale,
    scale: toPositiveNumber(transform.scale, 1),
    offsetX: toNumber(transform.offsetX, 0),
    offsetY: toNumber(transform.offsetY, 0),
    zoomLevel: toPositiveNumber(options.zoomLevel, 5),
    selectedObject: options.selectedObject || null,
    backgroundPathCache: options.backgroundPathCache || new Map(),
  }
}

function compareObjectPaintOrder(left, right) {
  const leftScore = objectPaintScore(left)
  const rightScore = objectPaintScore(right)
  if (leftScore !== rightScore) return leftScore - rightScore
  return toNumber(left.sort, 0) - toNumber(right.sort, 0)
}

function objectPaintScore(object) {
  if (object?.displayLevel === 'weak') return 0
  if (isVerifiedObject(object)) return 3
  if (object?.geometryType === 'point') return 2
  return 1
}

function objectPalette(object) {
  if (isVerifiedMapObject(object)) {
    return { fill: COLORS.verifiedFill, stroke: COLORS.verifiedStroke }
  }
  if (object?.displayLevel === 'weak') {
    return { fill: COLORS.weakFill, stroke: COLORS.weakStroke }
  }
  if (object?.geometryType === 'point') {
    return { fill: COLORS.poiFill, stroke: COLORS.poiStroke }
  }
  return { fill: COLORS.boothFill, stroke: COLORS.boothStroke }
}

function isVerifiedObject(object) {
  return isVerifiedMapObject(object)
}

function shouldDrawVerifiedOverviewPoint(object, metrics) {
  return isVerifiedObject(object) && toPositiveNumber(metrics.zoomLevel, VERIFIED_LABEL_ZOOM_LEVEL) <= VERIFIED_OVERVIEW_ZOOM_LEVEL
}

function isRentableObject(object) {
  return isRentableMapObject(object)
}

function isSameObject(left, right) {
  const leftID = objectIdentity(left)
  const rightID = objectIdentity(right)
  return Boolean(leftID && rightID && leftID === rightID)
}

function objectIdentity(object) {
  return object?.id || object?.code || ''
}

function objectCenter(object = {}) {
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    const points = normalizePolygonPoints(geometry)
    if (!points.length) return null
    const sums = points.reduce((acc, point) => ({ x: acc.x + point.x, y: acc.y + point.y }), { x: 0, y: 0 })
    return { x: sums.x / points.length, y: sums.y / points.length }
  }
  const point = objectPoint(object)
  if (!point) return null
  const { x, y } = point
  if (object.geometryType === 'point') return { x, y }
  const width = objectWidth(object)
  const height = objectHeight(object)
  if (width == null || height == null) return null
  return {
    x: x + width / 2,
    y: y + height / 2,
  }
}

function projectedObjectBounds(object = {}, metrics) {
  const bounds = objectBounds(object)
  if (!bounds) return null
  return {
    x: projectX(bounds.minX, metrics),
    y: projectY(bounds.minY, metrics),
    width: projectSize(bounds.maxX - bounds.minX, metrics),
    height: projectSize(bounds.maxY - bounds.minY, metrics),
  }
}

function objectLabelText(object = {}) {
  return object?.merchant?.name || object.name || object.code || ''
}

function logoText(object = {}) {
  const text = object?.merchant?.name || object.name || object.code || '认'
  return String(text).trim().slice(0, 1) || '认'
}

function objectBounds(object = {}) {
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    const points = normalizePolygonPoints(geometry)
    if (!points.length) return null
    return points.reduce(
      (bounds, point) => ({
        minX: Math.min(bounds.minX, point.x),
        minY: Math.min(bounds.minY, point.y),
        maxX: Math.max(bounds.maxX, point.x),
        maxY: Math.max(bounds.maxY, point.y),
      }),
      { minX: points[0].x, minY: points[0].y, maxX: points[0].x, maxY: points[0].y },
    )
  }
  const point = objectPoint(object)
  if (!point) return null
  const { x, y } = point
  if (object.geometryType === 'point') {
    return { minX: x, minY: y, maxX: x, maxY: y }
  }
  const width = objectWidth(object)
  const height = objectHeight(object)
  if (width == null || height == null) return null
  return { minX: x, minY: y, maxX: x + width, maxY: y + height }
}

function objectWidth(object = {}) {
  return geometryPositiveNumber(object.geometry?.width, 80)
}

function objectHeight(object = {}) {
  return geometryPositiveNumber(object.geometry?.height, 50)
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

function objectPoint(object = {}) {
  const geometry = object.geometry || {}
  const x = geometryNumber(geometry.x, object.centerX)
  const y = geometryNumber(geometry.y, object.centerY)
  if (x == null || y == null) return null
  return { x, y }
}

function geometryNumber(value, fallback) {
  // 渲染层不能把异常坐标兜底到原点，否则坏数据会在地图左上角显示成可见点位。
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

function projectX(value, metrics) {
  return metrics.offsetX + value * metrics.baseScale * metrics.scale
}

function projectY(value, metrics) {
  return metrics.offsetY + value * metrics.baseScale * metrics.scale
}

function projectSize(value, metrics) {
  return value * metrics.baseScale * metrics.scale
}

function truncateLabel(label, maxWidth, fontSize) {
  const text = String(label)
  const maxChars = Math.max(1, Math.floor(maxWidth / Math.max(1, fontSize)))
  return text.length > maxChars ? `${text.slice(0, maxChars)}...` : text
}

function drawCanvas(ctx, callback) {
  if (typeof ctx.draw === 'function') {
    ctx.draw(false, callback)
    return
  }
  if (typeof callback === 'function') {
    callback()
  }
}

function beginPath(ctx) {
  if (typeof ctx.beginPath === 'function') ctx.beginPath()
}

function closePath(ctx) {
  if (typeof ctx.closePath === 'function') ctx.closePath()
}

function moveTo(ctx, x, y) {
  if (typeof ctx.moveTo === 'function') ctx.moveTo(x, y)
}

function lineTo(ctx, x, y) {
  if (typeof ctx.lineTo === 'function') ctx.lineTo(x, y)
}

function arc(ctx, x, y, radius, startAngle, endAngle) {
  if (typeof ctx.arc === 'function') ctx.arc(x, y, radius, startAngle, endAngle)
}

function fill(ctx) {
  if (typeof ctx.fill === 'function') ctx.fill()
}

function stroke(ctx) {
  if (typeof ctx.stroke === 'function') ctx.stroke()
}

function fillRect(ctx, x, y, width, height) {
  if (typeof ctx.fillRect === 'function') ctx.fillRect(x, y, width, height)
}

function strokeRect(ctx, x, y, width, height) {
  if (typeof ctx.strokeRect === 'function') ctx.strokeRect(x, y, width, height)
}

function fillText(ctx, text, x, y, maxWidth) {
  if (typeof ctx.fillText === 'function') ctx.fillText(text, x, y, maxWidth)
}

function setFillStyle(ctx, value) {
  if (typeof ctx.setFillStyle === 'function') {
    ctx.setFillStyle(value)
  } else {
    ctx.fillStyle = value
  }
}

function setStrokeStyle(ctx, value) {
  if (typeof ctx.setStrokeStyle === 'function') {
    ctx.setStrokeStyle(value)
  } else {
    ctx.strokeStyle = value
  }
}

function setLineWidth(ctx, value) {
  if (typeof ctx.setLineWidth === 'function') {
    ctx.setLineWidth(value)
  } else {
    ctx.lineWidth = value
  }
}

function setLineDash(ctx, value) {
  if (typeof ctx.setLineDash === 'function') ctx.setLineDash(value)
}

function setFontSize(ctx, value) {
  if (typeof ctx.setFontSize === 'function') {
    ctx.setFontSize(value)
  } else {
    ctx.font = `${value}px sans-serif`
  }
}

function setTextAlign(ctx, value) {
  if (typeof ctx.setTextAlign === 'function') {
    ctx.setTextAlign(value)
  } else {
    ctx.textAlign = value
  }
}

function setTextBaseline(ctx, value) {
  if (typeof ctx.setTextBaseline === 'function') {
    ctx.setTextBaseline(value)
  } else {
    ctx.textBaseline = value
  }
}

function toNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPositiveNumber(value, fallback) {
  const parsed = toNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}
