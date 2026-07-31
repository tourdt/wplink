const SOURCE_CLAIMED = 'merchant_claimed'
const SOURCE_PRELISTED = 'platform_prelisted'

export function buildMerchantPlaceQuery(filters = {}) {
  const query = {
    cityCode: String(filters.cityCode || '').trim(),
    keyword: String(filters.keyword || '').trim(),
    categories: cleanList(filters.categories).join(','),
    merchantTypes: cleanList(filters.merchantTypes).join(','),
    claimed: ['claimed', 'prelisted'].includes(filters.claimed) ? filters.claimed : 'all',
    page: positiveInteger(filters.page, 1),
    pageSize: positiveInteger(filters.pageSize, 20),
  }
  if (filters.bounds) {
    for (const key of ['minLat', 'maxLat', 'minLng', 'maxLng']) {
      if (filters.bounds[key] !== undefined && filters.bounds[key] !== '') {
        query[key] = String(filters.bounds[key])
      }
    }
  }
  return Object.fromEntries(Object.entries(query).filter(([, value]) => value !== ''))
}

export function normalizeMerchantPlace(raw = {}) {
  const claimed = raw.sourceType === SOURCE_CLAIMED || Boolean(raw.claimed && raw.merchantId)
  return {
    objectId: String(raw.objectId || ''),
    merchantId: claimed ? String(raw.merchantId || '') : '',
    name: String(raw.name || raw.code || '未命名档口'),
    code: String(raw.code || ''),
    merchantType: String(raw.merchantType || ''),
    categoryCodes: cleanList(raw.categoryCodes),
    serviceTags: cleanList(raw.serviceTags),
    platformTags: cleanList(raw.platformTags),
    cityCode: String(raw.cityCode || ''),
    marketName: String(raw.marketName || ''),
    buildingName: String(raw.buildingName || ''),
    floorNo: String(raw.floorNo || ''),
    address: String(raw.address || ''),
    coverUrl: String(raw.coverUrl || ''),
    claimed,
    sourceType: claimed ? SOURCE_CLAIMED : SOURCE_PRELISTED,
    lat: String(raw.lat || ''),
    lng: String(raw.lng || ''),
    distanceText: String(raw.distanceText || ''),
    riskWarning: Boolean(raw.riskWarning),
  }
}

export function hasValidLocation(place = {}) {
  const lat = Number(place.lat)
  const lng = Number(place.lng)
  return place.lat !== '' && place.lng !== '' &&
    Number.isFinite(lat) && Number.isFinite(lng) &&
    lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

export function merchantPlaceSourceLabel(place = {}) {
  return place.sourceType === SOURCE_CLAIMED || place.claimed ? '已入驻' : '待认领'
}

function cleanList(values) {
  const list = Array.isArray(values) ? values : String(values || '').split(',')
  return [...new Set(list.map((value) => String(value || '').trim()).filter(Boolean))]
}

function positiveInteger(value, fallback) {
  const number = Number(value)
  return Number.isInteger(number) && number > 0 ? number : fallback
}
