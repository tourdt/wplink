const DEFAULT_MIN_SCALE = 1
const DEFAULT_MAX_SCALE = 3

export function createInitialTransform(options = {}) {
  const scale = normalizeScale(options.scale ?? DEFAULT_MIN_SCALE, options)
  return clampTransform(
    {
      scale,
      offsetX: centerOffset(options.viewportWidth, options.mapWidth, scale),
      offsetY: centerOffset(options.viewportHeight, options.mapHeight, scale),
    },
    options,
  )
}

export function startGesture(touches = [], transform = {}, options = {}) {
  const normalizedTouches = normalizeTouches(touches)
  const startTransform = normalizeTransform(transform, options)
  const mode = normalizedTouches.length >= 2 ? 'pinch' : 'pan'
  const state = {
    mode,
    startTouches: normalizedTouches,
    startTransform,
    transform: startTransform,
    moved: false,
  }

  if (mode === 'pinch') {
    const center = touchCenter(normalizedTouches[0], normalizedTouches[1])
    state.startDistance = touchDistance(normalizedTouches[0], normalizedTouches[1])
    state.startCenter = center
    state.startMapAnchor = screenToMap(center, startTransform)
  }

  return state
}

export function moveGesture(state, touches = [], options = {}) {
  if (!state) {
    return { state: null, transform: createInitialTransform(options), moved: false }
  }

  const normalizedTouches = normalizeTouches(touches)
  if (!normalizedTouches.length) {
    return { state, transform: state.transform, moved: state.moved }
  }

  const nextTransform = state.mode === 'pinch' && normalizedTouches.length >= 2
    ? movePinchGesture(state, normalizedTouches, options)
    : movePanGesture(state, normalizedTouches, options)
  const clampedTransform = clampTransform(nextTransform, options)
  const moved = state.moved || hasTransformMoved(state.startTransform, clampedTransform)
  const nextState = {
    ...state,
    transform: clampedTransform,
    moved,
  }
  return { state: nextState, transform: clampedTransform, moved }
}

export function endGesture(state, options = {}) {
  if (!state) {
    return createInitialTransform(options)
  }
  return clampTransform(state.transform, options)
}

export function screenToMap(point = {}, transform = {}) {
  const normalizedTransform = normalizeTransform(transform)
  return {
    x: (toNumber(point.x, 0) - normalizedTransform.offsetX) / normalizedTransform.scale,
    y: (toNumber(point.y, 0) - normalizedTransform.offsetY) / normalizedTransform.scale,
  }
}

export function mapToScreen(point = {}, transform = {}) {
  const normalizedTransform = normalizeTransform(transform)
  return {
    x: toNumber(point.x, 0) * normalizedTransform.scale + normalizedTransform.offsetX,
    y: toNumber(point.y, 0) * normalizedTransform.scale + normalizedTransform.offsetY,
  }
}

export function clampTransform(transform = {}, options = {}) {
  const scale = normalizeScale(transform.scale, options)
  const viewportWidth = toPositiveNumber(options.viewportWidth, 0)
  const viewportHeight = toPositiveNumber(options.viewportHeight, 0)
  const mapWidth = toPositiveNumber(options.mapWidth, viewportWidth)
  const mapHeight = toPositiveNumber(options.mapHeight, viewportHeight)
  const scaledWidth = mapWidth * scale
  const scaledHeight = mapHeight * scale

  return {
    scale,
    offsetX: clampAxisOffset(transform.offsetX, viewportWidth, scaledWidth, options),
    offsetY: clampAxisOffset(transform.offsetY, viewportHeight, scaledHeight, options),
  }
}

function movePanGesture(state, touches, options) {
  const current = touches[0]
  const start = state.startTouches[0] || current
  return {
    scale: state.startTransform.scale,
    offsetX: state.startTransform.offsetX + current.x - start.x,
    offsetY: state.startTransform.offsetY + current.y - start.y,
  }
}

function movePinchGesture(state, touches, options) {
  const currentDistance = touchDistance(touches[0], touches[1])
  const startDistance = state.startDistance || currentDistance || 1
  const nextScale = normalizeScale(state.startTransform.scale * (currentDistance / startDistance), options)
  const currentCenter = touchCenter(touches[0], touches[1])
  const anchor = state.startMapAnchor || screenToMap(state.startCenter || currentCenter, state.startTransform)
  return {
    scale: nextScale,
    offsetX: currentCenter.x - anchor.x * nextScale,
    offsetY: currentCenter.y - anchor.y * nextScale,
  }
}

function normalizeTouches(touches = []) {
  return Array.from(touches)
    .filter(Boolean)
    .map((touch, index) => ({
      id: touch.identifier ?? index,
      x: toNumber(touch.clientX ?? touch.x, 0),
      y: toNumber(touch.clientY ?? touch.y, 0),
    }))
}

function touchDistance(left, right) {
  return Math.hypot(right.x - left.x, right.y - left.y)
}

function touchCenter(left, right) {
  return {
    x: (left.x + right.x) / 2,
    y: (left.y + right.y) / 2,
  }
}

function normalizeTransform(transform = {}, options = {}) {
  return {
    scale: normalizeScale(transform.scale, options),
    offsetX: toNumber(transform.offsetX, 0),
    offsetY: toNumber(transform.offsetY, 0),
  }
}

function normalizeScale(value, options = {}) {
  const minScale = toPositiveNumber(options.minScale, DEFAULT_MIN_SCALE)
  const maxScale = Math.max(minScale, toPositiveNumber(options.maxScale, DEFAULT_MAX_SCALE))
  return clampNumber(toPositiveNumber(value, minScale), minScale, maxScale)
}

function clampAxisOffset(offset, viewportSize, scaledMapSize, options = {}) {
  const nextOffset = toNumber(offset, 0)
  if (!viewportSize || !scaledMapSize) return nextOffset
  if (scaledMapSize <= viewportSize) {
    const centeredOffset = Math.round((viewportSize - scaledMapSize) / 2)
    if (options.allowOverflow) {
      return clampAxisOffsetWithOverflow(nextOffset, centeredOffset, centeredOffset, options)
    }
    return centeredOffset
  }
  const minOffset = viewportSize - scaledMapSize
  const maxOffset = 0
  if (options.allowOverflow) {
    return clampAxisOffsetWithOverflow(nextOffset, minOffset, maxOffset, options)
  }
  return clampNumber(nextOffset, minOffset, maxOffset)
}

function clampAxisOffsetWithOverflow(offset, minOffset, maxOffset, options = {}) {
  const maxOverflow = toPositiveNumber(options.maxOverflow, 64)
  if (offset > maxOffset) {
    return maxOffset + dampOverflow(offset - maxOffset, maxOverflow)
  }
  if (offset < minOffset) {
    return minOffset - dampOverflow(minOffset - offset, maxOverflow)
  }
  return offset
}

function dampOverflow(distance, maxOverflow) {
  const damped = Math.sqrt(Math.max(0, distance)) * 5
  return Math.min(maxOverflow, damped)
}

function centerOffset(viewportSize, mapSize, scale) {
  const viewport = toPositiveNumber(viewportSize, 0)
  const map = toPositiveNumber(mapSize, viewport)
  return Math.round((viewport - map * scale) / 2)
}

function hasTransformMoved(startTransform, nextTransform) {
  return (
    Math.abs(startTransform.offsetX - nextTransform.offsetX) > 2 ||
    Math.abs(startTransform.offsetY - nextTransform.offsetY) > 2 ||
    Math.abs(startTransform.scale - nextTransform.scale) > 0.01
  )
}

function toNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPositiveNumber(value, fallback) {
  const parsed = toNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}

function clampNumber(value, min, max) {
  return Math.min(max, Math.max(min, value))
}
