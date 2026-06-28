import { STORAGE_KEYS } from '../common/constants'

export function getSession() {
  return {
    token: uni.getStorageSync(STORAGE_KEYS.token) || '',
    userId: uni.getStorageSync(STORAGE_KEYS.userId) || '',
    merchantId: uni.getStorageSync(STORAGE_KEYS.merchantId) || '',
  }
}

export function saveMerchantId(merchantId) {
  uni.setStorageSync(STORAGE_KEYS.merchantId, merchantId)
}

export function saveToken(token) {
  uni.setStorageSync(STORAGE_KEYS.token, token)
}

export function getMerchantId() {
  return uni.getStorageSync(STORAGE_KEYS.merchantId) || ''
}

export function saveUserId(userId) {
  uni.setStorageSync(STORAGE_KEYS.userId, userId)
}

export function getUserId() {
  return uni.getStorageSync(STORAGE_KEYS.userId) || ''
}
