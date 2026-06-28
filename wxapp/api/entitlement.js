import request from './request'

export function getMerchantEntitlements(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantId}/entitlements`,
    method: 'GET',
  })
}

export function listTopVouchers(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantId}/top-vouchers`,
    method: 'GET',
  })
}

export function redeemTopVoucher(voucherId, resourceId) {
  return request({
    url: `/api/v1/top-vouchers/${voucherId}/redeem`,
    method: 'POST',
    data: { resourceId },
  })
}
