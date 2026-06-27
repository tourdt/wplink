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

export function getMerchantId() {
  return uni.getStorageSync(STORAGE_KEYS.merchantId) || ''
}
