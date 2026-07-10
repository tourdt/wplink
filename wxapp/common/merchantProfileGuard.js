import { STORAGE_KEYS } from './constants.js'

const MERCHANT_PROFILE_PAGE = '/pages/merchant/profile'
let merchantProfilePromptTask = null

function getStoredMerchantId() {
  return uni.getStorageSync(STORAGE_KEYS.merchantId) || ''
}

export function hasMerchantProfile(merchantId = getStoredMerchantId()) {
  return Boolean(String(merchantId || '').trim())
}

export async function ensureMerchantProfileReady(merchantId = getStoredMerchantId()) {
  if (hasMerchantProfile(merchantId)) return true
  await promptCompleteMerchantProfile()
  return false
}

export function promptCompleteMerchantProfile() {
  if (merchantProfilePromptTask) return merchantProfilePromptTask

  // 小程序 onLoad/onShow 可能连续触发，复用同一个弹窗任务避免重复弹窗。
  merchantProfilePromptTask = new Promise((resolve) => {
    uni.showModal({
      title: '完善资料',
      content: '需要先完善发布者资料。',
      confirmText: '去完善',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          uni.navigateTo({ url: MERCHANT_PROFILE_PAGE })
        }
        resolve(false)
      },
      fail: () => resolve(false),
    })
  }).finally(() => {
    merchantProfilePromptTask = null
  })

  return merchantProfilePromptTask
}
