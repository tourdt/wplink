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

export function updateResourceDraft(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/draft`,
    method: 'PUT',
    data,
  })
}

export function submitResource(resourceId, merchantId = '') {
  return request({
    url: `/api/v1/resources/${resourceId}/submit`,
    method: 'POST',
    data: merchantId ? { merchantId } : {},
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

export function getEditableResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/me/resources/${resourceId}/edit`,
    method: 'GET',
    data: { merchantId },
  })
}

export function getResource(resourceId, options = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}`,
    method: 'GET',
    ...options,
  })
}

export function getOwnResource(resourceId, merchantId, options = {}) {
  const query = merchantId ? `?merchantId=${encodeURIComponent(merchantId)}` : ''
  return request({
    url: `/api/v1/me/resources/${resourceId}/detail${query}`,
    method: 'GET',
    ...options,
  })
}

export function recordResourceDetailView(resourceId) {
  return request({
    url: `/api/v1/resources/${resourceId}/detail-view`,
    method: 'POST',
  })
}

export function refreshResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/resources/${resourceId}/refresh`,
    method: 'POST',
    data: { merchantId },
  })
}

export function markResourceDeal(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/deal-feedback`,
    method: 'POST',
    data,
  })
}

export function takeDownResource(resourceId, merchantId, reason) {
  return request({
    url: `/api/v1/resources/${resourceId}/take-down`,
    method: 'POST',
    data: { merchantId, reason },
  })
}

export function deleteTakenDownResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/resources/${resourceId}`,
    method: 'DELETE',
    data: { merchantId },
  })
}

export function repostSimilarResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/resources/${resourceId}/repost-similar`,
    method: 'POST',
    data: { merchantId },
  })
}

export function recordResourceContact(resourceId, action) {
  return request({
    url: `/api/v1/resources/${resourceId}/contact-events`,
    method: 'POST',
    data: { action },
  })
}
