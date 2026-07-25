import request from './request'

export function getMerchant(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}`,
    method: 'GET',
    ...options,
  })
}

export function updateMerchant(merchantId, data) {
  return request({
    url: `/api/v1/merchants/${merchantId}`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}
