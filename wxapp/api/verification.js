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

export function getVerificationBillingConfig(cityCode = 'zhili') {
  return request({
    url: `/api/v1/verification-billing?cityCode=${encodeURIComponent(cityCode)}`,
    method: 'GET',
  })
}

export function createVerificationPayment(merchantId, verificationId, data = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/verifications/${verificationId}/payment`,
    method: 'POST',
    data,
  })
}
