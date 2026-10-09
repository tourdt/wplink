export function parseMerchantCategories(value) {
  const values = Array.isArray(value) ? value : String(value || '').split(/[,，]/)
  return [...new Set(values.map((item) => String(item || '').trim()).filter(Boolean))]
}

export function buildMerchantDirectoryQuery(filters = {}) {
  return {
    cityCode: String(filters.cityCode || '').trim(),
    keyword: String(filters.keyword || '').trim(),
    categories: parseMerchantCategories(filters.categories).join(','),
    page: positiveInteger(filters.page, 1),
    pageSize: boundedPageSize(filters.pageSize),
  }
}

export function normalizeMerchantDirectoryItem(raw = {}) {
  return {
    merchantId: String(raw.merchantId || '').trim(),
    merchantNo: String(raw.merchantNo || '').trim(),
    name: String(raw.name || '').trim() || '未命名商家',
    merchantType: String(raw.merchantType || '').trim(),
    primaryRole: String(raw.primaryRole || '').trim(),
    businessRoles: parseMerchantCategories(raw.businessRoles),
    mainCategories: parseMerchantCategories(raw.mainCategories),
    logoUrl: String(raw.logoUrl || '').trim(),
    addressText: String(raw.addressText || '').trim(),
    marketName: String(raw.marketName || '').trim(),
    buildingName: String(raw.buildingName || '').trim(),
    floorNo: String(raw.floorNo || '').trim(),
    boothNo: String(raw.boothNo || '').trim(),
    latitude: coordinateNumber(raw.latitude),
    longitude: coordinateNumber(raw.longitude),
    locationName: String(raw.locationName || '').trim(),
    lastResourceActiveAt: String(raw.lastResourceActiveAt || '').trim(),
    distanceMeters: nonNegativeNumber(raw.distanceMeters),
  }
}

export function hasValidMerchantLocation(item = {}) {
  const latitude = coordinateNumber(item.latitude)
  const longitude = coordinateNumber(item.longitude)
  return Number.isFinite(latitude) && Number.isFinite(longitude) &&
    latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

export function merchantDirectoryCoordinateText(item = {}) {
  return [item.marketName, item.buildingName, item.floorNo, item.boothNo]
    .map((value) => String(value || '').trim())
    .filter(Boolean)
    .join(' · ') || String(item.addressText || '').trim() || '接待地址待完善'
}

export function merchantDirectoryDetailPath(item = {}) {
  const merchantId = String(item.merchantId || '').trim()
  return merchantId ? `/pages/merchant/detail?id=${encodeURIComponent(merchantId)}` : ''
}

export function mergeMerchantDirectoryItems(current = [], incoming = []) {
  const merged = []
  const seen = new Set()
  for (const item of [...current, ...incoming]) {
    const merchantId = String(item?.merchantId || '').trim()
    if (!merchantId || seen.has(merchantId)) continue
    seen.add(merchantId)
    merged.push(item)
  }
  return merged
}

function coordinateNumber(value) {
  if (value === undefined || value === null || String(value).trim() === '') return Number.NaN
  return Number(value)
}

function nonNegativeNumber(value) {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? number : 0
}

function positiveInteger(value, fallback) {
  const number = Number(value)
  return Number.isInteger(number) && number > 0 ? number : fallback
}

function boundedPageSize(value) {
  const number = positiveInteger(value, 20)
  return number <= 100 ? number : 20
}
