import http from './http'

export function listOperationLogs(params = {}) {
  return http.get('/api/v1/admin/operation-logs', { params })
}
