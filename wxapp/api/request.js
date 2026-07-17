import { API_BASE_URL, STORAGE_KEYS } from '../common/constants'
import { redirectToLogin } from '../common/auth'
import { buildApiUrl } from '../common/url'
import { clearSession } from '../store/session'

const UNAUTHORIZED_ERROR_CODE = 'UNAUTHORIZED'
const REQUIRE_AUTH_MESSAGE = '请先登录'

export default function request(options = {}) {
  return new Promise((resolve, reject) => {
    const token = normalizeToken(uni.getStorageSync(STORAGE_KEYS.token))
    if (options.requireAuth && !token) {
      const message = options.authErrorMessage || REQUIRE_AUTH_MESSAGE
      redirectToLogin({ redirect: options.redirect })
      if (!options.suppressErrorToast) {
        uni.showToast({ title: message, icon: 'none' })
      }
      const error = new Error(message)
      error.code = UNAUTHORIZED_ERROR_CODE
      error.statusCode = 401
      reject(error)
      return
    }
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
        const unauthorizedSession = isUnauthorizedSession(res)
        if (unauthorizedSession) {
          clearSession()
          redirectToLogin()
        }
        if (unauthorizedSession || !options.suppressErrorToast) {
          uni.showToast({ title: message, icon: 'none' })
        }
        const error = new Error(message)
        error.code = res.data?.errorCode || ''
        error.statusCode = res.statusCode
        reject(error)
      },
      fail: (err) => {
        if (!options.suppressErrorToast) {
          uni.showToast({ title: '网络异常，请稍后重试', icon: 'none' })
        }
        reject(err)
      },
    })
  })
}

function isUnauthorizedSession(res) {
  return res.statusCode === 401 || res.data?.code === 401 || res.data?.errorCode === UNAUTHORIZED_ERROR_CODE
}

function normalizeToken(token) {
  if (token === undefined || token === null) return ''
  return String(token).trim()
}
