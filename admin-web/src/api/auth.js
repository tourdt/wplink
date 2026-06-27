import http from './http'

export function loginAdmin(payload) {
  return http.post('/api/v1/admin/auth/login', payload)
}
