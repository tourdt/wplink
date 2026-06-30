import { API_BASE_URL, STORAGE_KEYS } from '../common/constants'
import { buildApiUrl } from '../common/url'

export default function request(options) {
  return new Promise((resolve, reject) => {
    const token = uni.getStorageSync(STORAGE_KEYS.token)
    uni.request({
      url: buildApiUrl(API_BASE_URL, options.url),
      method: options.method || 'GET',
      data: options.data || {},
      header: {
        Authorization: token ? `Bearer ${token}` : '',
        ...(options.header || {}),
      },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data?.data ?? res.data)
          return
        }
        const message = res.data?.message || res.data?.msg || '请求失败，请稍后重试'
        if (!options.suppressErrorToast) {
          uni.showToast({ title: message, icon: 'none' })
        }
        reject(new Error(message))
      },
      fail: (err) => {
        uni.showToast({ title: '网络异常，请稍后重试', icon: 'none' })
        reject(err)
      },
    })
  })
}
