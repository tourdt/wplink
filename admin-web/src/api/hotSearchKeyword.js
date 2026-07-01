import http from './http'
import { buildHotSearchKeywordPayload } from './hotSearchKeywordPayload'

export { buildHotSearchKeywordPayload }

export function listHotSearchKeywords(params = {}) {
  return http.get('/api/v1/admin/hot-search-keywords', { params })
}

export function createHotSearchKeyword(payload) {
  return http.post('/api/v1/admin/hot-search-keywords', buildHotSearchKeywordPayload(payload))
}

export function updateHotSearchKeyword(configId, payload) {
  return http.post(`/api/v1/admin/hot-search-keywords/${configId}`, buildHotSearchKeywordPayload(payload))
}
