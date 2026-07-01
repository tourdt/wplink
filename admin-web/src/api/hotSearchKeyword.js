import http from './http'

export function listHotSearchKeywords(params = {}) {
  return http.get('/api/v1/admin/hot-search-keywords', { params })
}

export function createHotSearchKeyword(payload) {
  return http.post('/api/v1/admin/hot-search-keywords', payload)
}

export function updateHotSearchKeyword(configId, payload) {
  return http.post(`/api/v1/admin/hot-search-keywords/${configId}`, payload)
}
