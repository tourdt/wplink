export const DEFAULT_RESOURCE_SHARE_IMAGE = '/static/resource/default-resource-cover.png'

export const RESOURCE_SHARE_COVER_CANVAS_ID = 'resourceShareCoverCanvas'
export const RESOURCE_SHARE_COVER_SIZE = {
  width: 600,
  height: 480,
}

const resourceTypeText = {
  inventory: '库存清仓',
  goods: '现货货源',
  factory: '工厂接单',
  job: '招工招聘',
  rental: '出租转让',
  service: '配套服务',
  buy_goods: '找现货',
  find_inventory: '找库存',
  find_factory: '找工厂',
  find_service: '找服务',
  find_rental: '找场地',
}

export function buildResourceSharePath(resource = {}) {
  const id = normalizeText(resource.id)
  return id ? `/pages/resource/detail?id=${encodeURIComponent(id)}` : '/pages/home/index'
}

export function getResourceShareCoverSource(resource = {}) {
  const images = Array.isArray(resource.images) ? resource.images : []
  return normalizeText(resource.coverUrl) || normalizeText(images[0]) || DEFAULT_RESOURCE_SHARE_IMAGE
}

export function buildResourceSharePayload(resource = {}, imageUrl = '') {
  const payload = {
    title: buildResourceShareTitle(resource),
    path: buildResourceSharePath(resource),
    imageUrl: normalizeText(imageUrl) || getResourceShareCoverSource(resource),
  }
  return payload
}

export function buildResourceTimelinePayload(resource = {}, imageUrl = '') {
  const id = normalizeText(resource.id)
  return {
    title: buildResourceShareTitle(resource),
    query: id ? `id=${encodeURIComponent(id)}` : '',
    imageUrl: normalizeText(imageUrl) || getResourceShareCoverSource(resource),
  }
}

export function buildResourceSharePosterModel(resource = {}, merchant = {}) {
  const typeLabel = getResourceTypeLabel(resource)
  const cityName = normalizeText(resource.cityName || resource.city || resource.cityCode)
  const category = normalizeText(resource.category)
  const badges = [
    isVIPResource(resource, merchant) ? 'VIP' : '',
    cityName,
    category,
  ].filter(Boolean)

  return {
    typeLabel,
    title: normalizeText(resource.title) || '服装产业供需信息',
    coverSource: getResourceShareCoverSource(resource),
    badges,
    summaryLines: buildSummaryLines(resource),
    merchantName: normalizeText(merchant.name || resource.merchant?.name) || '衣货通商家',
    footerText: '衣货通 · 服装产业供需信息',
  }
}

export function buildResourceShareTitle(resource = {}) {
  const typeLabel = getResourceTypeLabel(resource)
  const title = normalizeText(resource.title || resource.category) || '服装产业供需信息'
  return `${typeLabel}｜${title}`
}

export function getResourceTypeLabel(resource = {}) {
  return resourceTypeText[resource.typeCode] || resourceTypeText[resource.resourceType] || '衣货通供应'
}

function buildSummaryLines(resource = {}) {
  return [
    normalizeText(resource.quantityText),
    normalizeText(resource.priceText),
  ].filter(Boolean)
}

function isVIPResource(resource = {}, merchant = {}) {
  return resource.merchant?.vipStatus === 'active' || merchant.vipStatus === 'active'
}

function normalizeText(value) {
  return String(value || '').trim()
}
