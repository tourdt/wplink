import http from './http'

export function getDashboardOverview(params = {}) {
  return http.get('/api/v1/admin/dashboard/overview', { params })
}
