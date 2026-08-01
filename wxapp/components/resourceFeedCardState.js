import { formatListFreshnessDate } from '../common/date.js'
import { resourceTypeLabel as resolveResourceTypeLabel } from '../common/resourceCategories.js'

export function buildResourceFeedCardModel(resource = {}, now) {
  const isDemand = resource.direction === 'demand'
  const images = Array.isArray(resource.images) ? resource.images : []
  const title = String(resource.title || '').trim()
  const quantityText = String(resource.quantityText || '').trim()
  const priceText = String(resource.priceText || '').trim()
  const merchantName = String(resource.merchant?.name || '').trim()
  const refreshedAt = String(resource.refreshedAt || '').trim()

  return {
    isDemand,
    directionLabel: isDemand ? '需求' : '供应',
    coverUrl: String(resource.coverUrl || images[0] || '').trim(),
    resourceTypeLabel: resolveResourceTypeLabel(resource),
    titleText: title || (isDemand ? '需求标题待完善' : '供应标题待完善'),
    quantityText,
    priceText,
    hasTradeInfo: Boolean(quantityText || priceText),
    merchantName: merchantName || (isDemand ? '采购方待确认' : '商家待确认'),
    // 更新时间缺失时不展示“近期”等推测性占位，避免用户误判内容新鲜度。
    freshnessText: refreshedAt ? formatListFreshnessDate(refreshedAt, now) : '',
    isCompleted: Boolean(resource.dealtAt),
  }
}
