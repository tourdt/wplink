import request from './request'

export function recordResourceExposures(data) {
  return request({
    url: '/api/v1/metrics/exposures/batch',
    method: 'POST',
    data,
    suppressErrorToast: true,
  })
}

export function recordMerchantMapEvent(data) {
  return request({
    url: '/api/v1/metrics/merchant-map-events',
    method: 'POST',
    data,
    suppressErrorToast: true,
  })
}

export function getResourceMetrics(resourceId, params = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}/metrics`,
    method: 'GET',
    data: params,
    requireAuth: true,
  })
}

export function getMerchantMetricsSummary(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/metrics/summary`,
    method: 'GET',
    suppressErrorToast: options.suppressErrorToast,
    requireAuth: true,
  })
}
