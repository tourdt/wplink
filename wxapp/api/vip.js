import request from './request'

export function listVIPPlans() {
  return request({
    url: '/api/v1/vip/plans',
    method: 'GET',
  })
}

export function listQuotaPacks() {
  return request({
    url: '/api/v1/vip/quota-packs',
    method: 'GET',
  })
}

export function getMerchantVIP(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/vip`,
    method: 'GET',
    ...options,
  })
}

export function createVIPOrder(merchantId, data) {
  return request({
    url: `/api/v1/merchants/${merchantId}/vip/orders`,
    method: 'POST',
    data,
  })
}

export function createQuotaPackOrder(merchantId, packCode) {
  return createVIPOrder(merchantId, {
    productType: 'quota_pack',
    productCode: packCode,
  })
}

export function createVIPPayment(merchantId, orderId, data = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/vip/orders/${orderId}/payment`,
    method: 'POST',
    data,
  })
}
