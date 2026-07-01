import http from './http'

export function listPendingVerifications(params = {}) {
  return http.get('/api/v1/admin/verifications/pending', { params })
}

export function reviewVerification(verificationId, payload) {
  return http.post(`/api/v1/admin/verifications/${verificationId}/review`, payload)
}

export function getVerificationBillingConfig(params = {}) {
  return http.get('/api/v1/admin/verification-billing', { params })
}

export function updateVerificationBillingConfig(payload) {
  return http.post('/api/v1/admin/verification-billing', payload)
}
