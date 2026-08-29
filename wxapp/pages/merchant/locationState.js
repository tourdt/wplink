import {
  hasValidMerchantLocation,
  normalizeMerchantDirectoryItem,
} from '../sourcing-map/merchantDirectoryState.js'

const CURRENT_MARKER_ID = 1
const CURRENT_MARKER = {
  iconPath: '/static/map/marker-selected.png',
  width: 46,
  height: 56,
  zIndex: 10,
}
const NEARBY_MARKER = {
  iconPath: '/static/map/marker-nearby.png',
  width: 34,
  height: 41,
  zIndex: 4,
}
const MAX_NEARBY_MERCHANTS = 20
const MAX_RADIUS_METERS = 1000

export function normalizeMerchantLocationContext(raw = {}) {
  const current = normalizeLocationMerchant(raw.current)
  const currentMerchantId = String(current?.merchantId || '').trim()
  const seenMerchantIds = new Set()
  const nearby = []

  // 服务端负责 1000 米距离计算和排序；客户端只做身份去重、坐标防御和最多 20 家的展示边界。
  for (const rawMerchant of (Array.isArray(raw.nearby) ? raw.nearby : [])) {
    const merchant = normalizeLocationMerchant(rawMerchant)
    const nearbyMerchantId = String(merchant?.merchantId || '').trim()
    if (!merchant || !nearbyMerchantId || nearbyMerchantId === currentMerchantId || seenMerchantIds.has(nearbyMerchantId)) continue
    seenMerchantIds.add(nearbyMerchantId)
    nearby.push(merchant)
    if (nearby.length === MAX_NEARBY_MERCHANTS) break
  }

  const rawRadiusMeters = Number(raw.radiusMeters)
  const radiusMeters = Number.isFinite(rawRadiusMeters) && rawRadiusMeters > 0
    ? Math.min(rawRadiusMeters, MAX_RADIUS_METERS)
    : 0
  return {
    current,
    nearby,
    radiusMeters,
    nearbyAvailable: Boolean(raw.nearbyAvailable),
  }
}

export function buildMerchantLocationMarkers(context = {}, selectedNearbyMerchantId = '') {
  if (!context.current || !hasValidMerchantLocation(context.current)) return []
  const current = context.current
  const markers = [{
    id: CURRENT_MARKER_ID,
    merchantId: current.merchantId,
    latitude: Number(current.latitude),
    longitude: Number(current.longitude),
    ...CURRENT_MARKER,
    anchor: { x: 0.5, y: 1 },
    callout: {
      content: current.name,
      color: '#061625',
      fontSize: 13,
      borderRadius: 6,
      bgColor: '#ffffff',
      padding: 8,
      display: 'ALWAYS',
      textAlign: 'center',
    },
  }]

  for (const [index, merchant] of (context.nearby || []).entries()) {
    if (!hasValidMerchantLocation(merchant)) continue
    const selected = merchant.merchantId === String(selectedNearbyMerchantId || '').trim()
    markers.push({
      id: index + 2,
      merchantId: merchant.merchantId,
      latitude: Number(merchant.latitude),
      longitude: Number(merchant.longitude),
      ...NEARBY_MARKER,
      iconPath: selected ? '/static/map/marker-nearby-selected.png' : NEARBY_MARKER.iconPath,
      zIndex: selected ? 6 : NEARBY_MARKER.zIndex,
      anchor: { x: 0.5, y: 1 },
    })
  }
  return markers
}

export function merchantIdFromMarker(markerId, markers = []) {
  const marker = markers.find((item) => String(item.id) === String(markerId))
  return String(marker?.merchantId || '').trim()
}

export function nearbyMerchantDomId(merchantId) {
  const normalizedMerchantId = String(merchantId || '').trim()
  if (!normalizedMerchantId) return ''
  const encodedMerchantId = Array.from(normalizedMerchantId, (character) => character.codePointAt(0).toString(16)).join('-')
  return `nearby-${encodedMerchantId}`
}

export function merchantMainTags(merchant = {}) {
  const tags = (merchant.mainCategories || []).map((tag) => String(tag || '').trim()).filter(Boolean)
  return [...new Set(tags)].slice(0, 3)
}

function normalizeLocationMerchant(raw) {
  if (!raw || typeof raw !== 'object') return null
  const merchant = normalizeMerchantDirectoryItem(raw)
  if (!merchant.merchantId || !hasValidMerchantLocation(merchant)) return null
  return merchant
}
