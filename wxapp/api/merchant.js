import request from './request'

const MERCHANT_UPDATE_FIELDS = [
  'name',
  'mainCategories',
  'description',
  'logoUrl',
  'images',
  'contactName',
  'contactPhone',
  'contactWechat',
  'publicPhone',
  'publicWechat',
  'businessProfile',
  'mapVisible',
  'addressText',
  'locationAction',
  'marketName',
  'buildingName',
  'floorNo',
  'boothNo',
  'latitude',
  'longitude',
  'locationName',
]

export function getMerchant(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}`,
    method: 'GET',
    ...options,
  })
}

export function setMerchantVisibility(merchantId, visible) {
  return request({
    url: `/api/v1/merchants/${merchantId}/visibility`,
    method: 'POST',
    data: { visible },
    requireAuth: true,
  })
}

export function updateMerchant(merchantId, data) {
  // 资料更新只发送当前 API 契约字段，避免旧 location JSON 或页面临时状态重新进入请求体。
  const payload = {}
  for (const field of MERCHANT_UPDATE_FIELDS) {
    if (Object.prototype.hasOwnProperty.call(data || {}, field)) {
      payload[field] = data[field]
    }
  }
  return request({
    url: `/api/v1/merchants/${merchantId}`,
    method: 'POST',
    data: payload,
    requireAuth: true,
  })
}
