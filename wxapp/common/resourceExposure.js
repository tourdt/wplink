import { recordResourceExposures } from '../api/metrics'

const VISITOR_STORAGE_KEY = 'wplink_exposure_visitor_key'
const FLUSH_DELAY_MS = 600
const MAX_BATCH_SIZE = 20

let pendingItems = []
let flushTimer = null
const sessionId = createIdentifier('session')

export function enqueueResourceExposure({ resourceId, source, visibleDurationMs }) {
  const normalizedId = String(resourceId || '').trim()
  const normalizedSource = String(source || '').trim()
  if (!normalizedId || !normalizedSource || Number(visibleDurationMs) < 800) return

  pendingItems.push({
    resourceId: normalizedId,
    source: normalizedSource,
    visibleDurationMs: Math.round(Number(visibleDurationMs)),
  })
  if (pendingItems.length >= MAX_BATCH_SIZE) {
    flushResourceExposures()
    return
  }
  if (!flushTimer) {
    flushTimer = setTimeout(flushResourceExposures, FLUSH_DELAY_MS)
  }
}

export function flushResourceExposures() {
  if (flushTimer) {
    clearTimeout(flushTimer)
    flushTimer = null
  }
  if (!pendingItems.length) return

  const queued = pendingItems
  pendingItems = []
  const groups = new Map()
  queued.forEach((item) => {
    if (!groups.has(item.source)) groups.set(item.source, [])
    groups.get(item.source).push({
      resourceId: item.resourceId,
      visibleDurationMs: item.visibleDurationMs,
    })
  })
  groups.forEach((items, source) => {
    recordResourceExposures({
      visitorKey: getVisitorKey(),
      sessionId,
      source,
      items: deduplicateItems(items),
    }).catch(() => {
      // 曝光属于低优先级埋点，失败不打断用户浏览，也不无限重试造成重复请求。
    })
  })
}

function deduplicateItems(items) {
  const bestByResource = new Map()
  items.forEach((item) => {
    const current = bestByResource.get(item.resourceId)
    if (!current || item.visibleDurationMs > current.visibleDurationMs) {
      bestByResource.set(item.resourceId, item)
    }
  })
  return Array.from(bestByResource.values()).slice(0, 50)
}

function getVisitorKey() {
  const existing = String(uni.getStorageSync(VISITOR_STORAGE_KEY) || '').trim()
  if (existing) return existing
  const created = createIdentifier('visitor')
  uni.setStorageSync(VISITOR_STORAGE_KEY, created)
  return created
}

function createIdentifier(prefix) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
}
