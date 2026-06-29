import http from './http'

export function listBannerTopics(params = {}) {
  return http.get('/api/v1/admin/banner-topics', { params })
}

export function createBannerTopic(payload) {
  return http.post('/api/v1/admin/banner-topics', payload)
}

export function updateBannerTopic(configId, payload) {
  return http.post(`/api/v1/admin/banner-topics/${configId}`, payload)
}
