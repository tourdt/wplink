import http from './http'

export function listMapScenes(params = {}) {
  return http.get('/api/v1/admin/map/scenes', { params })
}

export function saveMapScene(payload) {
  if (payload.code) {
    return http.post(`/api/v1/admin/map/scenes/${payload.code}`, payload)
  }
  return http.post('/api/v1/admin/map/scenes', payload)
}

export function publishMapScene(sceneCode) {
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/publish`)
}

export function listMapObjects(sceneCode, params = {}) {
  return http.get(`/api/v1/admin/map/scenes/${sceneCode}/objects`, { params })
}

export function saveMapObject(sceneCode, payload) {
  if (payload.id) {
    return http.post(`/api/v1/admin/map/objects/${payload.id}`, payload)
  }
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/objects`, payload)
}

export function updateMapObjectStatus(objectId, status) {
  return http.post(`/api/v1/admin/map/objects/${objectId}/status`, { status })
}

export function batchGenerateMapObjects(sceneCode, payload) {
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/objects/batch-generate`, payload)
}

export function listMapCategories(params = {}) {
  return http.get('/api/v1/admin/map/categories', { params })
}
