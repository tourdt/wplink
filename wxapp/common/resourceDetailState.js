const DEMAND_TYPE_PATTERN = /^(buy_|find_|seek_)/
const FULL_WIDTH_LABEL_PATTERN = /(地址|位置|区域|地点|交期|时效|范围|要求|工艺|备注|说明|时间|条件|方式)/

// 详情页统一使用服务端摘要字段，避免按二级类型在页面中分支，新增类型可直接复用。
export function buildResourceDetailPresentation(resource = {}) {
  const isDemand = isDemandResource(resource)
  const typeName = normalizeText(resource.typeName) || normalizeText(resource.category) || '供需信息'

  return {
    isDemand,
    noun: isDemand ? '需求' : '供应',
    typeName,
  }
}

// 核心数量和报价与后台配置属性统一展示，空值和长文本规则复用详情参数处理逻辑。
export function buildResourceDetailSpecItems(resource = {}, attributeItems = []) {
  const isDemand = isDemandResource(resource)
  return buildDetailSpecItems([
    { label: '数量/面积', value: resource.quantityText },
    { label: isDemand ? '预算/报价' : '价格/报价', value: resource.priceText },
    ...attributeItems,
  ])
}

// 长文本和语义上需要完整阅读的参数使用整行，短参数仍保持双列信息密度。
export function buildDetailSpecItems(attributeItems = []) {
  return attributeItems
    .filter((item) => normalizeText(item?.label) && normalizeText(item?.value))
    .map((item) => ({
      label: normalizeText(item.label),
      value: normalizeText(item.value),
      fullWidth: shouldUseFullWidthSpec(item.label, item.value),
    }))
}

function isDemandResource(resource) {
  const direction = normalizeText(resource.direction)
  if (direction === 'demand') return true
  const typeCode = normalizeText(resource.typeCode)
  return DEMAND_TYPE_PATTERN.test(typeCode) || typeCode.endsWith('_buy') || typeCode === 'job_seeking'
}

function shouldUseFullWidthSpec(label, value) {
  return FULL_WIDTH_LABEL_PATTERN.test(normalizeText(label)) || /\r?\n/.test(String(value)) || Array.from(normalizeText(value)).length > 12
}

function normalizeText(value) {
  return String(value || '').trim()
}
