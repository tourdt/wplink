import { hasValidLocation, normalizeMerchantPlace } from '../sourcing-map/merchantPlaceState.js'

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

export function normalizeMerchantLocationContext(raw = {}) {
  const current = normalizeLocationPlace(raw.current)
  const nearby = (Array.isArray(raw.nearby) ? raw.nearby : [])
    .map(normalizeLocationPlace)
    .filter(Boolean)
  const radiusMeters = Number(raw.radiusMeters)

  return {
    current,
    nearby,
    radiusMeters: Number.isFinite(radiusMeters) && radiusMeters > 0 ? radiusMeters : 0,
    nearbyAvailable: Boolean(raw.nearbyAvailable),
  }
}

export function buildMerchantLocationMarkers(context = {}, selectedNearbyMerchantId = '') {
  if (!context.current || !hasValidLocation(context.current)) return []

  const current = context.current
  const markers = [{
    id: CURRENT_MARKER_ID,
    merchantId: current.merchantId,
    latitude: Number(current.lat),
    longitude: Number(current.lng),
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

  for (const [index, place] of (context.nearby || []).entries()) {
    if (!hasValidLocation(place)) continue
    const selected = String(place.merchantId || '') === String(selectedNearbyMerchantId || '')
    markers.push({
      id: index + 2,
      merchantId: place.merchantId,
      latitude: Number(place.lat),
      longitude: Number(place.lng),
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

function normalizeLocationPlace(raw) {
  if (!raw || typeof raw !== 'object') return null
  const place = normalizeMerchantPlace(raw)
  if (!hasValidLocation(place)) return null
  const distanceMeters = Number(raw.distanceMeters)
  return {
    ...place,
    distanceMeters: Number.isFinite(distanceMeters) && distanceMeters >= 0 ? distanceMeters : 0,
  }
}
