import request from './request'

export function getMerchantEntitlements(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/entitlements`,
    method: 'GET',
    ...options,
    requireAuth: true,
  })
}

export function getMerchantEntitlementUsageRecords(merchantId, entitlementId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/entitlements/${entitlementId}/usage-records`,
    method: 'GET',
    ...options,
    requireAuth: true,
  })
}

export function listTopVouchers(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantId}/top-vouchers`,
    method: 'GET',
    requireAuth: true,
  })
}

export function redeemTopVoucher(voucherId, resourceId, merchantId = '') {
  return request({
    url: `/api/v1/top-vouchers/${voucherId}/redeem`,
    method: 'POST',
    data: { merchantId, resourceId },
    requireAuth: true,
  })
}
