import http from './http'

export function listMatchCases(params = {}) {
  return http.get('/api/v1/admin/match-cases', { params })
}

export function createMatchCase(payload) {
  return http.post('/api/v1/admin/match-cases', payload)
}

export function updateMatchCaseStatus(matchCaseId, payload) {
  return http.post(`/api/v1/admin/match-cases/${matchCaseId}/status`, payload)
}

export function addMatchCaseResources(matchCaseId, payload) {
  return http.post(`/api/v1/admin/match-cases/${matchCaseId}/resources`, payload)
}

export function addMatchCaseParticipants(matchCaseId, payload) {
  return http.post(`/api/v1/admin/match-cases/${matchCaseId}/participants`, payload)
}
