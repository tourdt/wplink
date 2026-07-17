import request from './request'

export function getResourceFavoriteState(resourceId) {
  return request({
    url: `/api/v1/me/favorite-resources/${resourceId}`,
    method: 'GET',
    requireAuth: true,
  })
}

export function setResourceFavorite(resourceId, favorited) {
  return request({
    url: `/api/v1/me/favorite-resources/${resourceId}`,
    method: 'POST',
    data: { favorited },
    requireAuth: true,
  })
}

export function listFavoriteResources(params = {}) {
  return request({
    url: '/api/v1/me/favorite-resources',
    data: params,
    requireAuth: true,
  })
}

export function getMerchantFollowState(merchantId) {
  return request({
    url: `/api/v1/me/followed-merchants/${merchantId}`,
    method: 'GET',
    requireAuth: true,
  })
}

export function setMerchantFollow(merchantId, followed) {
  return request({
    url: `/api/v1/me/followed-merchants/${merchantId}`,
    method: 'POST',
    data: { followed },
    requireAuth: true,
  })
}

export function listFollowedMerchants(params = {}) {
  return request({
    url: '/api/v1/me/followed-merchants',
    data: params,
    requireAuth: true,
  })
}
