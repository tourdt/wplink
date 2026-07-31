import { hasValidLocation } from './merchantPlaceState.js'

const MARKER_ICON_PATHS = {
  claimed: '/static/map/marker-claimed.png',
  prelisted: '/static/map/marker-prelisted.png',
  selected: '/static/map/marker-selected.png',
}

export function buildTencentMapMarkers(places = [], selectedObjectId = '') {
  return places.filter(hasValidLocation).map((place, index) => {
    const selected = place.objectId === selectedObjectId
    return {
      id: index + 1,
      objectId: place.objectId,
      latitude: Number(place.lat),
      longitude: Number(place.lng),
      title: place.name,
      iconPath: selected
        ? MARKER_ICON_PATHS.selected
        : place.claimed ? MARKER_ICON_PATHS.claimed : MARKER_ICON_PATHS.prelisted,
      width: selected ? 38 : 32,
      height: selected ? 46 : 39,
      anchor: { x: 0.5, y: 1 },
    }
  })
}

export function placeIdFromMarker(markerId, markers = []) {
  return markers.find((marker) => Number(marker.id) === Number(markerId))?.objectId || ''
}

export function fallbackMapCenter(places = [], defaultCenter = {}) {
  const first = places.find(hasValidLocation)
  if (first) {
    return {
      latitude: Number(first.lat),
      longitude: Number(first.lng),
    }
  }
  return {
    latitude: Number(defaultCenter.latitude || 30.87),
    longitude: Number(defaultCenter.longitude || 120.12),
  }
}
