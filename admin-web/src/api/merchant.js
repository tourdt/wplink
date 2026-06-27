import http from './http'

export function listMerchants(params = {}) {
  return http.get('/api/v1/admin/merchants', { params })
}

export function createMerchant(payload) {
  return http.post('/api/v1/merchants', payload)
}

export function getMerchant(merchantId) {
  return http.get(`/api/v1/merchants/${merchantId}`)
}

export function updateMerchant(merchantId, payload) {
  return http.patch(`/api/v1/merchants/${merchantId}`, payload)
}
