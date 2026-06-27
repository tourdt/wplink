import request from './request'

export function submitVerification(merchantId, data) {
  return request({
    url: `/api/v1/merchants/${merchantId}/verifications`,
    method: 'POST',
    data,
  })
}

export function getLatestVerification(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantId}/verifications/latest`,
    method: 'GET',
  })
}
