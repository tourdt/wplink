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
    requireAuth: true,
  })
}

export function createVIPOrder(merchantId, data) {
  return request({
    url: `/api/v1/merchants/${merchantId}/vip/orders`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}

export function createQuotaPackOrder(merchantId, packCode, options = {}) {
  const data = {
    productType: 'quota_pack',
    productCode: packCode,
  }
  if (options.resourceId) data.resourceId = options.resourceId
  return createVIPOrder(merchantId, data)
}

export function createVIPPayment(merchantId, orderId, data = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/vip/orders/${orderId}/payment`,
    method: 'POST',
    data,
    requireAuth: true,
  })
}
