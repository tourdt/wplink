export const MIN_MAP_ZOOM = 0.25
export const MAX_MAP_ZOOM = 3

export function normalizeMapZoom(value, min = MIN_MAP_ZOOM, max = MAX_MAP_ZOOM) {
  const parsed = Number(value)
  const fallback = 1
  const safeValue = Number.isFinite(parsed) ? parsed : fallback
  const clamped = Math.min(max, Math.max(min, safeValue))
  return Math.round(clamped * 100) / 100
}

export function scaledMapSize({ width, height, scale }) {
  const safeScale = normalizeMapZoom(scale)
  return {
    width: Math.max(1, Math.round(Number(width) * safeScale)),
    height: Math.max(1, Math.round(Number(height) * safeScale)),
  }
}

export function mapCenterFromSize({ width, height }) {
  return {
    x: Math.max(0, Math.round(Number(width) / 2)),
    y: Math.max(0, Math.round(Number(height) / 2)),
  }
}

export function mapPointFromClientPoint({ clientX, clientY, stageRect, scale }) {
  const safeScale = normalizeMapZoom(scale)
  return {
    x: Math.max(0, Math.round((Number(clientX) - Number(stageRect?.left || 0)) / safeScale)),
    y: Math.max(0, Math.round((Number(clientY) - Number(stageRect?.top || 0)) / safeScale)),
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
  const paddingX = Number(clientWidth) * Number(paddingRatio || 0)
  const paddingY = Number(clientHeight) * Number(paddingRatio || 0)
  const width = Number(mapWidth)
  const height = Number(mapHeight)

  return {
    minX: Math.round(clamp((Number(scrollLeft) - paddingX) / safeScale, 0, width)),
    minY: Math.round(clamp((Number(scrollTop) - paddingY) / safeScale, 0, height)),
    maxX: Math.round(clamp((Number(scrollLeft) + Number(clientWidth) + paddingX) / safeScale, 0, width)),
    maxY: Math.round(clamp((Number(scrollTop) + Number(clientHeight) + paddingY) / safeScale, 0, height)),
  }
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value))
}
