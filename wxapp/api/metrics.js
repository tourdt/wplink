import request from './request'

export function getResourceMetrics(resourceId, params = {}) {
  return request({
    url: `/api/v1/resources/${resourceId}/metrics`,
    method: 'GET',
    data: params,
  })
}

export function getMerchantMetricsSummary(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/metrics/summary`,
    method: 'GET',
    suppressErrorToast: options.suppressErrorToast,
  })
}
