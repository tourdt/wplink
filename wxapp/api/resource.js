import request from './request'

export function createResource(data) {
  return request({
    url: '/api/v1/resources',
    method: 'POST',
    data,
  })
}

export function createResourceDraft(data) {
  return request({
    url: '/api/v1/resources/drafts',
    method: 'POST',
    data,
  })
}

export function submitResource(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}/submit`,
    method: 'POST',
  })
}

export function listResources(params = {}) {
  return request({
    url: '/api/v1/resources',
    method: 'GET',
    data: params,
  })
}

export function searchResources(params = {}) {
  return request({
    url: '/api/v1/resource-search',
    method: 'GET',
    data: params,
  })
}

export function listMyResources(params = {}) {
  return request({
    url: '/api/v1/me/resources',
    method: 'GET',
    data: params,
  })
}

export function getResource(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}`,
    method: 'GET',
  })
}

export function recordResourceDetailView(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}/detail-view`,
    method: 'POST',
  })
}

export function refreshResource(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}/refresh`,
    method: 'POST',
  })
}

export function markResourceDeal(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/deal-feedback`,
    method: 'POST',
    data,
  })
}

export function takeDownResource(resourceId, reason) {
  return request({
    url: `/api/v1/resources/${resourceId}/take-down`,
    method: 'POST',
    data: { reason },
  })
}

export function repostSimilarResource(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}/repost-similar`,
    method: 'POST',
  })
}

export function recordResourceContact(resourceId, action) {
  return request({
    url: `/api/v1/resources/${resourceId}/contact-events`,
    method: 'POST',
    data: { action },
  })
}
