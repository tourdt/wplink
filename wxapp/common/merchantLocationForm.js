const EMPTY_LOCATION_FORM = Object.freeze({
  marketName: '',
  buildingName: '',
  floorNo: '',
  boothNo: '',
  latitude: null,
  longitude: null,
  locationName: '',
})

const LOCATION_INTENT_SET = 'set'
const LOCATION_INTENT_CLEAR = 'clear'
const LOCATION_INTENT_DECISION_REQUIRED = 'decision_required'

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

export function createMerchantLocationSession({ status = 'hidden', form = {} } = {}) {
  const hasValidCoordinates = buildMerchantLocationPayload(form).locationAction === 'set'
  return {
    intent: status === 'risk_hidden' && hasValidCoordinates
      ? LOCATION_INTENT_DECISION_REQUIRED
      : hasValidCoordinates ? LOCATION_INTENT_SET : LOCATION_INTENT_CLEAR,
    resolving: false,
    requestToken: 0,
  }
}

export function invalidateMerchantLocationSession(session) {
  const current = normalizeLocationSession(session)
  return finishLocationRequest(current, current.intent)
}

export function applyMerchantLocationBaseline(session, baseline = {}) {
  const current = normalizeLocationSession(session)
  const next = createMerchantLocationSession(baseline)
  return {
    ...next,
    requestToken: current.requestToken + 1,
  }
}

export function reduceMerchantLocationSession(session, event = {}) {
  const current = normalizeLocationSession(session)
  switch (event.type) {
    case 'selection_started':
      return {
        ...current,
        resolving: true,
        requestToken: current.requestToken + 1,
      }
    case 'selection_succeeded':
      if (!isMerchantLocationRequestCurrent(current, event.token)) return current
      return finishLocationRequest(current, LOCATION_INTENT_SET)
    case 'selection_failed':
      if (!isMerchantLocationRequestCurrent(current, event.token)) return current
      return finishLocationRequest(current, current.intent)
    case 'selection_cancelled':
      if (!isMerchantLocationRequestCurrent(current, event.token)) return current
      return finishLocationRequest(current, current.intent)
    case 'location_cleared':
    case 'address_changed':
      return finishLocationRequest(current, LOCATION_INTENT_CLEAR)
    default:
      return current
  }
}

export function isMerchantLocationRequestCurrent(session, token) {
  return Boolean(session?.resolving) && Number.isInteger(token) && session.requestToken === token
}

export function buildMerchantLocationSubmission(session, form = {}) {
  if (session?.resolving) {
    return { ok: false, message: '地图位置解析中，请稍候再保存' }
  }
  if (session?.intent === LOCATION_INTENT_DECISION_REQUIRED) {
    return { ok: false, message: '该位置暂未公开，请重新选择地图位置或清除地图位置后再保存' }
  }
  const payload = session?.intent === LOCATION_INTENT_CLEAR
    ? buildMerchantLocationPayload(clearMerchantLocationForm(form))
    : buildMerchantLocationPayload(form)
  return { ok: true, payload }
}

export async function resolveLatestMerchantLocationRequest({ token, task, getSession }) {
  try {
    const value = await task()
    if (!isMerchantLocationRequestCurrent(getSession(), token)) return { current: false }
    return { current: true, value }
  } catch (error) {
    if (!isMerchantLocationRequestCurrent(getSession(), token)) return { current: false }
    return { current: true, error }
  }
}

function normalizeCoordinatePair(latitude, longitude) {
  const normalizedLatitude = normalizeCoordinate(latitude)
  const normalizedLongitude = normalizeCoordinate(longitude)
  if (normalizedLatitude === null || normalizedLongitude === null) return null
  if (normalizedLatitude < -90 || normalizedLatitude > 90) return null
  if (normalizedLongitude < -180 || normalizedLongitude > 180) return null
  return { latitude: normalizedLatitude, longitude: normalizedLongitude }
}

function normalizeLocationSession(session = {}) {
  const intent = [LOCATION_INTENT_SET, LOCATION_INTENT_CLEAR, LOCATION_INTENT_DECISION_REQUIRED].includes(session.intent)
    ? session.intent
    : LOCATION_INTENT_CLEAR
  return {
    intent,
    resolving: Boolean(session.resolving),
    requestToken: Number.isInteger(session.requestToken) && session.requestToken >= 0 ? session.requestToken : 0,
  }
}

function finishLocationRequest(session, intent) {
  return {
    intent,
    resolving: false,
    requestToken: session.requestToken + 1,
  }
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
