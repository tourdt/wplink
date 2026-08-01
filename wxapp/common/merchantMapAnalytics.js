import { recordMerchantMapEvent } from '../api/metrics'

const VISITOR_STORAGE_KEY = 'wplink_map_visitor_key'
const MAX_IDENTIFIER_LENGTH = 96
const ALLOWED_EVENTS = new Set([
  'location_entry_click',
  'location_view',
  'navigation_click',
  'nearby_drawer_open',
  'nearby_marker_click',
  'nearby_merchant_click',
])
const ALLOWED_SOURCES = new Set(['directory', 'merchant_detail', 'merchant_location'])
const sessionId = createIdentifier('session')

export function trackMerchantMapEvent({ merchantId, targetMerchantId = '', eventType, source } = {}) {
  const normalizedMerchantId = String(merchantId || '').trim()
  const normalizedTargetMerchantId = String(targetMerchantId || '').trim()
  if (!normalizedMerchantId || !ALLOWED_EVENTS.has(eventType) || !ALLOWED_SOURCES.has(source)) return

  try {
    recordMerchantMapEvent({
      merchantId: normalizedMerchantId,
      targetMerchantId: normalizedTargetMerchantId,
      visitorKey: getVisitorKey(),
      sessionId,
      eventType,
      source,
    }).catch(() => {
      // 地图行为属于低优先级埋点，失败不打断位置查看、导航或商家跳转，也不重试制造重复事件。
    })
  } catch (err) {
    // 本地存储或请求封装同步异常时同样静默降级，业务动作必须继续执行。
  }
}

function getVisitorKey() {
  try {
    const existing = String(uni.getStorageSync(VISITOR_STORAGE_KEY) || '').trim()
    if (existing && existing.length <= MAX_IDENTIFIER_LENGTH) return existing

    const created = createIdentifier('visitor')
    uni.setStorageSync(VISITOR_STORAGE_KEY, created)
    return created
  } catch (err) {
    // 极少数存储不可用场景使用本次生成值，避免低优先级埋点影响页面主流程。
    return createIdentifier('visitor')
  }
}

function createIdentifier(prefix) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
}
