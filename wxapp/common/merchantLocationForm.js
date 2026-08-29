const EMPTY_LOCATION_FORM = Object.freeze({
  marketName: '',
  buildingName: '',
  floorNo: '',
  boothNo: '',
  latitude: null,
  longitude: null,
  locationName: '',
})

export function normalizeMerchantLocationForm(detail = {}) {
  const location = detail?.location || {}
  const coordinates = normalizeCoordinatePair(location.latitude, location.longitude)
  if (!coordinates) return clearMerchantLocationForm(location)

  return {
    marketName: normalizeText(location.marketName),
    buildingName: normalizeText(location.buildingName),
    floorNo: normalizeText(location.floorNo),
    boothNo: normalizeText(location.boothNo),
    latitude: coordinates.latitude,
    longitude: coordinates.longitude,
    locationName: normalizeText(location.locationName),
  }
}

export function buildMerchantLocationPayload(form = {}) {
  const coordinates = normalizeCoordinatePair(form.latitude, form.longitude)
  if (!coordinates) {
    return {
      locationAction: 'clear',
      marketName: '',
      buildingName: '',
      floorNo: '',
      boothNo: '',
      latitude: 0,
      longitude: 0,
      locationName: '',
    }
  }

  return {
    locationAction: 'set',
    marketName: normalizeText(form.marketName),
    buildingName: normalizeText(form.buildingName),
    floorNo: normalizeText(form.floorNo),
    boothNo: normalizeText(form.boothNo),
    latitude: coordinates.latitude,
    longitude: coordinates.longitude,
    locationName: normalizeText(form.locationName),
  }
}

export function clearMerchantLocationForm(_form = {}) {
  return { ...EMPTY_LOCATION_FORM }
}

function normalizeCoordinatePair(latitude, longitude) {
  const normalizedLatitude = normalizeCoordinate(latitude)
  const normalizedLongitude = normalizeCoordinate(longitude)
  if (normalizedLatitude === null || normalizedLongitude === null) return null
  if (normalizedLatitude < -90 || normalizedLatitude > 90) return null
  if (normalizedLongitude < -180 || normalizedLongitude > 180) return null
  return { latitude: normalizedLatitude, longitude: normalizedLongitude }
}

function normalizeCoordinate(value) {
  if (typeof value !== 'number' && typeof value !== 'string') return null
  if (typeof value === 'string' && value.trim() === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function normalizeText(value) {
  return String(value ?? '').trim()
}
