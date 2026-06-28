import http from './http'

export function listSearchLogs(params = {}) {
  return http.get('/api/v1/admin/search-logs', { params })
}
