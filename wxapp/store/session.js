import { STORAGE_KEYS } from '../common/constants.js'

export function getSession() {
  return {
    token: normalizeSessionValue(uni.getStorageSync(STORAGE_KEYS.token)),
    userId: normalizeSessionValue(uni.getStorageSync(STORAGE_KEYS.userId)),
    merchantId: normalizeSessionValue(uni.getStorageSync(STORAGE_KEYS.merchantId)),
  }
}

export function saveMerchantId(merchantId) {
  const value = normalizeSessionValue(merchantId)
  if (!value) {
    uni.removeStorageSync(STORAGE_KEYS.merchantId)
    return
  }
  uni.setStorageSync(STORAGE_KEYS.merchantId, value)
}

export function saveToken(token) {
  uni.setStorageSync(STORAGE_KEYS.token, normalizeSessionValue(token))
}

export function clearSession() {
  uni.removeStorageSync(STORAGE_KEYS.token)
  uni.removeStorageSync(STORAGE_KEYS.userId)
  uni.removeStorageSync(STORAGE_KEYS.merchantId)
}

export function getMerchantId() {
  return normalizeSessionValue(uni.getStorageSync(STORAGE_KEYS.merchantId))
}

export function saveUserId(userId) {
  uni.setStorageSync(STORAGE_KEYS.userId, normalizeSessionValue(userId))
}

export function getUserId() {
  return normalizeSessionValue(uni.getStorageSync(STORAGE_KEYS.userId))
}

function normalizeSessionValue(value) {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}
