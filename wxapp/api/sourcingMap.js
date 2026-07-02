import request from './request'

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
