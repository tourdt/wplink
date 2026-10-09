export const MERCHANT_CATEGORY_OPTIONS = Object.freeze([
  '童装', '女童装', '男童装', '婴幼童装', '连衣裙', '套装',
  '卫衣', '裤装', '外套', '内衣家居服', '针织', '其他',
])

export function mergeMerchantCategories(values = []) {
  return [...new Set((Array.isArray(values) ? values : [])
    .map((value) => String(value || '').trim()).filter(Boolean))]
}

export function toggleMerchantCategory(values, category) {
  const current = mergeMerchantCategories(values)
  const item = String(category || '').trim()
  if (!item) return current
  return current.includes(item) ? current.filter((value) => value !== item) : [...current, item]
}
