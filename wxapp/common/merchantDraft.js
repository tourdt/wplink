import { hasBusinessProfileStructure } from './merchantBusinessProfile.js'

const DRAFT_PREFIX = 'wplink_merchant_draft_v2'

export function isMerchantDraftSupported(draft) {
  return Boolean(draft && hasBusinessProfileStructure(draft.businessProfile) && typeof draft.mapVisible === 'boolean')
}

function draftKey(userId, merchantId) {
  const user = String(userId || '').trim()
  const merchant = String(merchantId || '').trim()
  return user && merchant ? `${DRAFT_PREFIX}:${encodeURIComponent(user)}:${encodeURIComponent(merchant)}` : ''
}

function hasTemporaryImage(draft) {
  const urls = [draft?.logoUrl, ...(Array.isArray(draft?.images) ? draft.images : [])].filter(Boolean)
  return urls.some((url) => !/^https?:\/\//i.test(String(url)))
}

export function getMerchantDraft(userId, merchantId) {
  const key = draftKey(userId, merchantId)
  if (!key) return null
  try {
    const value = uni.getStorageSync(key)
    if (!isMerchantDraftSupported(value) || hasTemporaryImage(value)) return null
    return value
  } catch {
    return null
  }
}

export function saveMerchantDraft(userId, merchantId, draft) {
  const key = draftKey(userId, merchantId)
  if (!key || !isMerchantDraftSupported(draft) || hasTemporaryImage(draft)) return false
  try {
    uni.setStorageSync(key, { ...draft, savedAt: Date.now() })
    return true
  } catch {
    return false
  }
}

export function clearMerchantDraft(userId, merchantId) {
  const key = draftKey(userId, merchantId)
  if (!key) return false
  try {
    uni.removeStorageSync(key)
    return true
  } catch {
    return false
  }
}
