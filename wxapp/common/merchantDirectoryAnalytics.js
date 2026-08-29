import { recordMerchantDirectoryEvent } from '../api/metrics'

const VISITOR_STORAGE_KEY = 'wplink_directory_visitor_key'
const NAVIGATION_FEEDBACK_KEY = 'wplink_directory_navigation_feedback'
const MAX_IDENTIFIER_LENGTH = 96
const ALLOWED_EVENTS = new Set([
  'directory_exposure',
  'directory_merchant_click',
  'location_view',
  'nearby_marker_click',
  'nearby_list_item_click',
  'nearby_merchant_click',
  'nearby_drawer_open',
  'navigation_click',
  'location_report_submit',
])
const ALLOWED_SOURCES = new Set(['directory', 'merchant_detail', 'merchant_location'])
const sessionId = createIdentifier('session')
let navigationFeedbackRevision = 0
let activeNavigationFeedbackClaim = null

export function trackMerchantDirectoryEvent({ merchantId, targetMerchantId = '', eventType, source } = {}) {
  const normalizedMerchantId = String(merchantId || '').trim()
  const normalizedTargetMerchantId = String(targetMerchantId || '').trim()
  if (!normalizedMerchantId || !ALLOWED_EVENTS.has(eventType) || !ALLOWED_SOURCES.has(source)) return

  try {
    Promise.resolve(recordMerchantDirectoryEvent({
      merchantId: normalizedMerchantId,
      targetMerchantId: normalizedTargetMerchantId,
      visitorKey: getVisitorKey(),
      sessionId,
      eventType,
      source,
    })).catch(() => {
      // 目录行为统计失败不能阻断商家浏览、位置查看、导航或反馈提交，也不重试制造重复事件。
    })
  } catch (err) {
    // 存储或请求封装同步异常时同样静默降级，保证买家的主流程继续执行。
  }
}

export function saveNavigationFeedback(target = {}) {
  const feedback = normalizeNavigationFeedback(target)
  if (!feedback) return false
  try {
    // 跨页缓存只保留公开商家身份；不保存坐标、旧点位身份或其他页面状态。
    uni.setStorageSync(NAVIGATION_FEEDBACK_KEY, feedback)
    navigationFeedbackRevision += 1
    return true
  } catch (err) {
    return false
  }
}

export function claimNavigationFeedback() {
  if (activeNavigationFeedbackClaim) return null
  try {
    const feedback = normalizeNavigationFeedback(uni.getStorageSync(NAVIGATION_FEEDBACK_KEY))
    if (!feedback) return null
    // 同一 JS 进程共享一个同步认领；目录页和位置页连续 onShow 时只有先到者能发起报告请求。
    activeNavigationFeedbackClaim = {
      feedback,
      revision: navigationFeedbackRevision,
    }
    return feedback
  } catch (err) {
    return null
  }
}

export function completeNavigationFeedback(feedback) {
  const claim = activeNavigationFeedbackClaim
  if (!claim || claim.feedback !== feedback) return false
  activeNavigationFeedbackClaim = null
  if (claim.revision !== navigationFeedbackRevision) return false

  try {
    const stored = normalizeNavigationFeedback(uni.getStorageSync(NAVIGATION_FEEDBACK_KEY))
    if (!stored || stored.merchantId !== feedback.merchantId || stored.name !== feedback.name) return false
    uni.removeStorageSync(NAVIGATION_FEEDBACK_KEY)
    navigationFeedbackRevision += 1
    return true
  } catch (err) {
    // 完成清理失败时保留缓存，允许下一次 onShow 重新认领并完成反馈。
    return false
  }
}

export function releaseNavigationFeedback(feedback) {
  if (!activeNavigationFeedbackClaim || activeNavigationFeedbackClaim.feedback !== feedback) return false
  activeNavigationFeedbackClaim = null
  return true
}

export function clearNavigationFeedback() {
  try {
    uni.removeStorageSync(NAVIGATION_FEEDBACK_KEY)
  } catch (err) {
    // 清理失败只会跳过本次反馈询问，不影响导航失败提示。
  } finally {
    navigationFeedbackRevision += 1
    activeNavigationFeedbackClaim = null
  }
}

function normalizeNavigationFeedback(target) {
  const merchantId = String(target?.merchantId || '').trim()
  if (!merchantId) return null
  return { merchantId, name: String(target?.name || '').trim() }
}

function getVisitorKey() {
  try {
    const existing = String(uni.getStorageSync(VISITOR_STORAGE_KEY) || '').trim()
    if (existing && existing.length <= MAX_IDENTIFIER_LENGTH) return existing
    const created = createIdentifier('visitor')
    uni.setStorageSync(VISITOR_STORAGE_KEY, created)
    return created
  } catch (err) {
    return createIdentifier('visitor')
  }
}

function createIdentifier(prefix) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
}
