import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 12000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('wplink_admin_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response.data?.data ?? response.data,
  (error) => {
    const message = error.response?.data?.message || error.response?.data?.msg || '请求失败，请稍后重试'
    ElMessage.error(message)
    return Promise.reject(error)
  },
)

export default http
