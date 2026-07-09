import request from './request'

export function getMerchantEntitlements(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/entitlements`,
    method: 'GET',
    ...options,
  })
}

export function listTopVouchers(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantId}/top-vouchers`,
    method: 'GET',
  })
}

export function redeemTopVoucher(voucherId, resourceId, merchantId = '') {
  return request({
    url: `/api/v1/top-vouchers/${voucherId}/redeem`,
    method: 'POST',
    data: { merchantId, resourceId },
  })
}
