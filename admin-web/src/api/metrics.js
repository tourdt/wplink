import http from './http'

export function getResourceMetrics(resourceId, params = {}) {
  return http.get(`/api/v1/resources/${resourceId}/metrics`, { params })
}

export function getMerchantMetricsSummary(merchantId) {
  return http.get(`/api/v1/merchants/${merchantId}/metrics/summary`)
}
