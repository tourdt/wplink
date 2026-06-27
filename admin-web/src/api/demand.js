import http from './http'

export function listDemands(params = {}) {
  return http.get('/api/v1/admin/purchase-demands', { params })
}

export function getDemand(demandId) {
  return http.get(`/api/v1/admin/purchase-demands/${demandId}`)
}

export function updateDemandStatus(demandId, payload) {
  return http.patch(`/api/v1/admin/purchase-demands/${demandId}/status`, payload)
}

export function createDemand(payload) {
  return http.post('/api/v1/purchase-demands', payload)
}

export function listMyDemands(params = {}) {
  return http.get('/api/v1/me/purchase-demands', { params })
}
