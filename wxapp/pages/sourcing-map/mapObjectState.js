export function isVerifiedMapObject(object) {
  return Boolean(object?.isVerifiedMerchant || object?.displayLevel === 'highlight')
}

// 选中态、命中优先级和列表高亮必须使用同一身份规则；系统未上线，不再为历史静态 code 做身份兜底。
export function mapObjectIdentity(object) {
  return object?.id || ''
}

export function isRentableMapObject(object) {
  if (!object) return false
  if (isRentableMapValue(object.displayLevel) || isRentableMapValue(object.status) || isRentableMapValue(object.type)) return true
  const tagValues = [
    ...(object.platformTags || []),
    ...(object.serviceTags || []),
    ...(object.categoryCodes || []),
  ]
  if (tagValues.some(isRentableMapValue)) return true

  const extra = object.extra || {}
  return Boolean(
    extra.isRentable === true ||
      extra.forRent === true ||
      isRentableMapValue(extra.rentalStatus) ||
      isRentableMapValue(extra.rentStatus) ||
      isRentableMapValue(extra.boothStatus),
  )
}

function isRentableMapValue(value) {
  const normalized = String(value || '').trim().toLowerCase()
  return [
    'rent',
    'rentable',
    'rental',
    'for_rent',
    'available_for_rent',
    'renting',
    'leasing',
    'vacant',
    '出租',
    '待出租',
    '招租',
  ].includes(normalized)
}
