import http from './http'

export function listPendingVerifications(params = {}) {
  return http.get('/api/v1/admin/verifications/pending', { params })
}

export function reviewVerification(verificationId, payload) {
  return http.post(`/api/v1/admin/verifications/${verificationId}/review`, payload)
}
