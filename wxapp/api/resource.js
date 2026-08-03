import request from './request'

export function createResource(data) {
  return request({
    url: '/api/v1/resources',
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function createResourceDraft(data) {
  return request({
    url: '/api/v1/resources/drafts',
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function updateResourceDraft(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/draft`,
    method: 'PUT',
    data,
    requireAuth: true,
  })
}

export function submitResource(resourceId, merchantId = '') {
  return request({
    url: `/api/v1/resources/${resourceId}/submit`,
    method: 'POST',
    data: merchantId ? { merchantId } : {},
    requireAuth: true,
  })
}

export function listResources(params = {}, options = {}) {
  return request({
    url: '/api/v1/resources',
    method: 'GET',
    data: params,
    ...options,
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
    requireAuth: true,
  })
}

export function getEditableResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/me/resources/${resourceId}/edit`,
    method: 'GET',
    data: { merchantId },
    requireAuth: true,
  })
}

export function getResource(resourceId, options = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}`,
    method: 'GET',
    ...options,
  }).then(normalizeResourceDetail)
}

export function getOwnResource(resourceId, merchantId, options = {}) {
  const query = merchantId ? `?merchantId=${encodeURIComponent(merchantId)}` : ''
  return request({
    url: `/api/v1/me/resources/${resourceId}/detail${query}`,
    method: 'GET',
    ...options,
    requireAuth: true,
  }).then(normalizeResourceDetail)
}

// 详情页只消费后端 presentation；空列表在 API 边界收敛，避免页面生命周期内重复兜底。
function normalizeResourceDetail(detail) {
  const resourceDetail = detail && typeof detail === 'object' ? detail : {}
  const presentation = resourceDetail.presentation && typeof resourceDetail.presentation === 'object'
    ? resourceDetail.presentation
    : {}
  return {
    ...resourceDetail,
    presentation: {
      ...presentation,
      fields: Array.isArray(presentation.fields) ? presentation.fields : [],
      tags: Array.isArray(presentation.tags) ? presentation.tags : [],
    },
  }
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
    requireAuth: true,
  })
}

export function markResourceDeal(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/deal-feedback`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function takeDownResource(resourceId, merchantId, reason) {
  return request({
    url: `/api/v1/resources/${resourceId}/take-down`,
    method: 'POST',
    data: { merchantId, reason },
    requireAuth: true,
  })
}

export function deleteTakenDownResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/resources/${resourceId}`,
    method: 'DELETE',
    data: { merchantId },
    requireAuth: true,
  })
}

export function repostSimilarResource(resourceId, merchantId) {
  return request({
    url: `/api/v1/resources/${resourceId}/repost-similar`,
    method: 'POST',
    data: { merchantId },
    requireAuth: true,
  })
}

export function recordResourceContact(resourceId, action) {
  return request({
    url: `/api/v1/resources/${resourceId}/contact-events`,
    method: 'POST',
    data: { action },
  })
}

export function reportResource(resourceId, data) {
  return request({
    url: `/api/v1/resources/${resourceId}/reports`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function createContactUnlockOrder(resourceId, data = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}/contact-unlock-orders`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function createContactUnlockPayment(resourceId, orderId, data = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}/contact-unlock-orders/${orderId}/payment`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}
