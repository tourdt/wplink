export const MIN_MAP_ZOOM = 0.25
export const MAX_MAP_ZOOM = 3

export function normalizeMapZoom(value, min = MIN_MAP_ZOOM, max = MAX_MAP_ZOOM) {
  const safeMin = positiveNumber(min, MIN_MAP_ZOOM)
  const safeMax = Math.max(safeMin, positiveNumber(max, MAX_MAP_ZOOM))
  const safeValue = finiteNumber(value, 1)
  const clamped = Math.min(safeMax, Math.max(safeMin, safeValue))
  return Math.round(clamped * 100) / 100
}

export function scaledMapSize({ width, height, scale }) {
  const safeScale = normalizeMapZoom(scale)
  return {
    width: Math.max(1, Math.round(dimensionNumber(width, 1) * safeScale)),
    height: Math.max(1, Math.round(dimensionNumber(height, 1) * safeScale)),
  }
}

export function mapCenterFromSize({ width, height }) {
  return {
    x: Math.max(0, Math.round(nonNegativeNumber(width, 0) / 2)),
    y: Math.max(0, Math.round(nonNegativeNumber(height, 0) / 2)),
  }
}

export function mapPointFromClientPoint({ clientX, clientY, stageRect, scale }) {
  const safeScale = normalizeMapZoom(scale)
  const stageLeft = finiteNumber(stageRect?.left, 0)
  const stageTop = finiteNumber(stageRect?.top, 0)
  return {
    x: Math.max(0, Math.round((finiteNumber(clientX, 0) - stageLeft) / safeScale)),
    y: Math.max(0, Math.round((finiteNumber(clientY, 0) - stageTop) / safeScale)),
  }
}

export function buildViewportBounds({
  scrollLeft,
  scrollTop,
  clientWidth,
  clientHeight,
  mapWidth,
  mapHeight,
  scale,
  paddingRatio,
}) {
  const safeScale = normalizeMapZoom(scale)
  const viewportWidth = nonNegativeNumber(clientWidth, 0)
  const viewportHeight = nonNegativeNumber(clientHeight, 0)
  const width = dimensionNumber(mapWidth, Math.max(1, viewportWidth))
  const height = dimensionNumber(mapHeight, Math.max(1, viewportHeight))
  const safePaddingRatio = nonNegativeNumber(paddingRatio, 0)
  const paddingX = viewportWidth * safePaddingRatio
  const paddingY = viewportHeight * safePaddingRatio
  const safeScrollLeft = nonNegativeNumber(scrollLeft, 0)
  const safeScrollTop = nonNegativeNumber(scrollTop, 0)

  // 视口边界会作为后台查询参数，必须保证始终是有限数字，避免异常 UI 状态污染请求。
  return {
    minX: Math.round(clamp((safeScrollLeft - paddingX) / safeScale, 0, width)),
    minY: Math.round(clamp((safeScrollTop - paddingY) / safeScale, 0, height)),
    maxX: Math.round(clamp((safeScrollLeft + viewportWidth + paddingX) / safeScale, 0, width)),
    maxY: Math.round(clamp((safeScrollTop + viewportHeight + paddingY) / safeScale, 0, height)),
  }
}

function finiteNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function nonNegativeNumber(value, fallback) {
  return Math.max(0, finiteNumber(value, fallback))
}

function positiveNumber(value, fallback) {
  const parsed = finiteNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}

function dimensionNumber(value, fallback) {
  return Math.max(1, positiveNumber(value, fallback))
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value))
}
