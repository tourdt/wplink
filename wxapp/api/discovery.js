import request from './request'

export function listHomeBanners(params = {}) {
  return request({
    url: '/api/v1/home/banners',
    method: 'GET',
    data: params,
  })
}

export function listHomeRecommendCards(params = {}) {
  return request({
    url: '/api/v1/home/recommend-cards',
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
