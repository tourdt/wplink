import http from './http.js'

export function listMerchants(params = {}, config = {}) {
  return http.get('/api/v1/admin/merchants', { ...config, params })
}

export function createMerchant(payload) {
  return http.post('/api/v1/merchants', payload)
}

export function updateMerchant(merchantId, payload) {
  return http.post(`/api/v1/merchants/${merchantId}`, payload)
}

export function listMerchantLocationReports(params = {}, config = {}) {
  return http.get('/api/v1/admin/merchants/location-reports', { ...config, params })
}

export function reviewMerchantLocationReport(reportId, payload, config = {}) {
  return http.post(`/api/v1/admin/merchants/location-reports/${reportId}/review`, payload, config)
}

export function updateMerchantLocationStatus(merchantId, payload, config = {}) {
  return http.post(`/api/v1/admin/merchants/${merchantId}/location-status`, payload, config)
}

const merchantLocationGovernanceRequestConfig = { skipGlobalErrorMessage: true }

export const merchantLocationGovernanceApi = {
  listMerchants: (params = {}) => listMerchants(params, merchantLocationGovernanceRequestConfig),
  listMerchantLocationReports: (params = {}) => listMerchantLocationReports(params, merchantLocationGovernanceRequestConfig),
  reviewMerchantLocationReport: (reportId, payload) => reviewMerchantLocationReport(reportId, payload, merchantLocationGovernanceRequestConfig),
  updateMerchantLocationStatus: (merchantId, payload) => updateMerchantLocationStatus(merchantId, payload, merchantLocationGovernanceRequestConfig),
}
