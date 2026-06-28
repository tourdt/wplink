import http from './http'

export function createResource(payload) {
  return http.post('/api/v1/resources', payload)
}

export function createResourceDraft(payload) {
  return http.post('/api/v1/resources/drafts', payload)
}

export function submitResource(resourceId, merchantId = '') {
  return http.post(`/api/v1/resources/${resourceId}/submit`, merchantId ? { merchantId } : {})
}

export function listResources(params = {}) {
  return http.get('/api/v1/resources', { params })
}

export function getResource(resourceId) {
  return http.get(`/api/v1/resources/${resourceId}`)
}

export function listPendingResources(params = {}) {
  return http.get('/api/v1/admin/resources/pending', { params })
}

export function reviewResource(resourceId, payload) {
  return http.post(`/api/v1/admin/resources/${resourceId}/review`, payload)
}
