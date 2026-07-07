import axios from 'axios'
import { ElMessage } from 'element-plus'
import { clearAdminSession, readAdminToken } from '../stores/adminSession'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 12000,
})

let redirectingToLogin = false

http.interceptors.request.use((config) => {
  const token = readAdminToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response.data?.data ?? response.data,
  (error) => {
    if (isUnauthorized(error)) {
      handleUnauthorized()
      return Promise.reject(error)
    }
    const message = error.response?.data?.message || error.response?.data?.msg || '请求失败，请稍后重试'
    ElMessage.error(message)
    return Promise.reject(error)
  },
)

function isUnauthorized(error) {
  return error.response?.status === 401 || error.response?.data?.code === 401 || error.response?.data?.errorCode === 'UNAUTHORIZED'
}

function handleUnauthorized() {
  clearAdminSession()
  if (redirectingToLogin || window.location.pathname.endsWith('/login')) {
    return
  }
  redirectingToLogin = true
  ElMessage.error('登录已失效，请重新登录')

  // 401 统一回到登录页，并带上当前地址，方便重新登录后回到原来的运营页面。
  const base = import.meta.env.BASE_URL || '/'
  const normalizedBase = base.endsWith('/') ? base : `${base}/`
  const loginURL = new URL(`${normalizedBase}login`, window.location.origin)
  loginURL.searchParams.set('redirect', `${window.location.pathname}${window.location.search}${window.location.hash}`)
  window.location.assign(loginURL.toString())
}

export default http
