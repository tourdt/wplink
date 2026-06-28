import request from './request'

export function getResourceFavoriteState(resourceId) {
  return request({
    url: `/api/v1/me/favorite-resources/${resourceId}`,
    method: 'GET',
  })
}

export function setResourceFavorite(resourceId, favorited) {
  return request({
    url: `/api/v1/me/favorite-resources/${resourceId}`,
    method: 'POST',
    data: { favorited },
  })
}

export function listFavoriteResources(params = {}) {
  return request({
    url: '/api/v1/me/favorite-resources',
    data: params,
  })
}

export function getMerchantFollowState(merchantId) {
  return request({
    url: `/api/v1/me/followed-merchants/${merchantId}`,
    method: 'GET',
  })
}

export function setMerchantFollow(merchantId, followed) {
  return request({
    url: `/api/v1/me/followed-merchants/${merchantId}`,
    method: 'POST',
    data: { followed },
  })
}

export function listFollowedMerchants(params = {}) {
  return request({
    url: '/api/v1/me/followed-merchants',
    data: params,
  })
}

export function listSavedSearches(params = {}) {
  return request({
    url: '/api/v1/me/saved-searches',
    data: params,
  })
}

export function createSavedSearch(data) {
  return request({
    url: '/api/v1/me/saved-searches',
    method: 'POST',
    data,
  })
}

export function deleteSavedSearch(savedSearchId) {
  return request({
    url: `/api/v1/me/saved-searches/${savedSearchId}`,
    method: 'DELETE',
  })
}
