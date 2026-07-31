import request from './request'

export function listHomeOperationConfig(params = {}) {
  return request({
    url: '/api/v1/home/operation-config',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function listHomeResources(params = {}) {
  return request({
    url: '/api/v1/home/resources',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function listHomeRecentMerchants(params = {}) {
  return request({
    url: '/api/v1/home/recent-merchants',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function listHotSearchKeywords(params = {}) {
  return request({
    url: '/api/v1/search/hot-keywords',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function getTopicResources(topicId, params = {}) {
  return request({
    url: `/api/v1/topics/${topicId}/resources`,
    method: 'GET',
    data: params,
  })
}

export function validateWebview(url) {
  return request({
    url: '/api/v1/webview/validate',
    method: 'POST',
    data: { url },
  })
}
