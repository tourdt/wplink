import http from './http'

export function grantMerchantEntitlement(merchantId, payload) {
  return http.post(`/api/v1/admin/merchants/${merchantId}/entitlements`, payload)
}

export function listMerchantEntitlements(merchantId) {
  return http.get(`/api/v1/merchants/${merchantId}/entitlements`)
}

export function listTopVouchers(merchantId) {
  return http.get(`/api/v1/merchants/${merchantId}/top-vouchers`)
}
