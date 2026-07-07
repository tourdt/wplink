/**
 * 统一计算地图对象的边界盒。
 *
 * 渲染层、命中测试、可视区域裁剪都依赖同一套坐标规则，避免同一个坏坐标在不同模块中
 * 表现不一致。返回 null 表示对象几何数据不可用，调用方应直接跳过该对象。
 */
export function mapObjectBounds(object = {}) {
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    return polygonBounds(geometry)
  }
  const origin = mapObjectOrigin(object)
  if (!origin) return null
  if (object.geometryType === 'point') {
    return { minX: origin.x, minY: origin.y, maxX: origin.x, maxY: origin.y }
  }
  const width = geometryPositiveNumber(geometry.width)
  const height = geometryPositiveNumber(geometry.height)
  if (width == null || height == null) return null
  return { minX: origin.x, minY: origin.y, maxX: origin.x + width, maxY: origin.y + height }
}

/**
 * 统一计算地图对象的视觉中心点。
 *
 * 多边形取所有顶点的平均点，矩形取边界中心，点位直接返回点坐标。该中心用于标签、
 * 认证商户标记和点位命中半径，必须和边界计算共用相同的非法数据处理策略。
 */
export function mapObjectCenter(object = {}) {
  const geometry = object.geometry || {}
  if (object.geometryType === 'polygon') {
    const points = normalizePolygonPoints(geometry)
    if (!points.length) return null
    const sums = points.reduce((acc, point) => ({ x: acc.x + point.x, y: acc.y + point.y }), { x: 0, y: 0 })
    return { x: sums.x / points.length, y: sums.y / points.length }
  }
  const origin = mapObjectOrigin(object)
  if (!origin) return null
  if (object.geometryType === 'point') return origin
  const width = geometryPositiveNumber(geometry.width)
  const height = geometryPositiveNumber(geometry.height)
  if (width == null || height == null) return null
  return {
    x: origin.x + width / 2,
    y: origin.y + height / 2,
  }
}

/**
 * 标准化多边形顶点。
 *
 * 只要任一顶点缺失或不是有限数字，就认为整个多边形不可用。这样可以防止部分坏顶点
 * 拉伸边界盒，导致对象被错误绘制、误选或错误保留在可视区域内。
 */
export function normalizePolygonPoints(geometry = {}) {
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

/**
 * 标准化用户触点或转换后的地图坐标。
 */
export function normalizeMapPoint(mapPoint = {}) {
  const x = finiteNumber(mapPoint.x)
  const y = finiteNumber(mapPoint.y)
  if (x == null || y == null) return null
  return { x, y }
}

/**
 * 标准化视口边界。
 *
 * 调用方可以传入任意顺序的 min/max，本函数会自动归正；非有限数字会返回 null，
 * 防止异常视口把所有对象都判断为可见或不可见。
 */
export function normalizeBounds(bounds = {}) {
  const minX = finiteNumber(bounds.minX)
  const minY = finiteNumber(bounds.minY)
  const maxX = finiteNumber(bounds.maxX)
  const maxY = finiteNumber(bounds.maxY)
  if ([minX, minY, maxX, maxY].some((value) => value == null)) {
    return null
  }
  return {
    minX: Math.min(minX, maxX),
    minY: Math.min(minY, maxY),
    maxX: Math.max(minX, maxX),
    maxY: Math.max(minY, maxY),
  }
}

function polygonBounds(geometry = {}) {
  const points = normalizePolygonPoints(geometry)
  if (!points.length) return null
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

function mapObjectOrigin(object = {}) {
  const geometry = object.geometry || {}
  const x = geometryNumber(geometry.x)
  const y = geometryNumber(geometry.y)
  if (x == null || y == null) return null
  return { x, y }
}

function geometryNumber(value) {
  // 系统未上线，地图对象必须写入完整 canonical geometry；缺字段直接拒绝，避免后续再背历史兜底。
  if (value === undefined || value === null) return null
  return finiteNumber(value)
}

function geometryPositiveNumber(value) {
  const parsed = geometryNumber(value)
  return parsed != null && parsed > 0 ? parsed : null
}

function finiteNumber(value) {
  if (typeof value === 'string' && value.trim() === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}
