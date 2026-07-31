import request from './request'

export function listMerchantPlaces(params = {}) {
  return request({
    url: '/api/v1/map/merchant-places',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function listMapScenes(params = {}) {
  return request({
    url: '/api/v1/map/scenes',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function getMapScene(sceneCode, options = {}) {
  return request({
    url: `/api/v1/map/scenes/${sceneCode}`,
    method: 'GET',
    ...options,
  })
}

export function listMapObjects(sceneCode, params = {}) {
  return request({
    url: `/api/v1/map/scenes/${sceneCode}/objects`,
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function searchMapObjects(params = {}) {
  return request({
    url: '/api/v1/map/objects/search',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function listMapCategories(params = {}) {
  return request({
    url: '/api/v1/map/categories',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function getMapObject(objectId, options = {}) {
  return request({
    url: `/api/v1/map/objects/${objectId}`,
    method: 'GET',
    ...options,
  })
}

export function listNearbyPois(objectId, params = {}) {
  return request({
    url: `/api/v1/map/objects/${objectId}/nearby-pois`,
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function submitMapLocationCorrection(objectId, data = {}) {
  return request({
    url: `/api/v1/map/objects/${objectId}/location-corrections`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function submitMapRiskReport(objectId, data = {}) {
  return request({
    url: `/api/v1/map/objects/${objectId}/risk-reports`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function getMerchantMapBinding(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/map-binding`,
    method: 'GET',
    suppressErrorToast: true,
    ...options,
    requireAuth: true,
  })
}

export function listMapBindCandidates(params = {}) {
  return request({
    url: '/api/v1/map/bind-candidates',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
    requireAuth: true,
  })
}

export function submitMapBindRequest(merchantId, data = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/map-binding-requests`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}
