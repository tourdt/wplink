const DEMAND_TYPE_PATTERN = /^(buy_|find_|seek_)/

// 详情页仅在此统一供需方向和类型文案；详细参数完全由后端 presentation.fields 决定。
export function buildResourceDetailPresentation(resource = {}) {
  const isDemand = isDemandResource(resource)
  const typeName = normalizeText(resource.typeName) || normalizeText(resource.category) || '供需信息'

  return {
    isDemand,
    noun: isDemand ? '需求' : '供应',
    typeName,
  }
}

function isDemandResource(resource) {
  const direction = normalizeText(resource.direction)
  if (direction === 'demand') return true
  const typeCode = normalizeText(resource.typeCode)
  return DEMAND_TYPE_PATTERN.test(typeCode) || typeCode.endsWith('_buy') || typeCode === 'job_seeking'
}

function normalizeText(value) {
  return String(value || '').trim()
}
