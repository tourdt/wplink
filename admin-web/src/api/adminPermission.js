import http from './http'

export function listAdminOperators(params = {}) {
  return http.get('/api/v1/admin/operators', { params })
}

export function createAdminOperator(payload) {
  return http.post('/api/v1/admin/operators', payload)
}

export function updateAdminOperator(userId, payload) {
  return http.post(`/api/v1/admin/operators/${userId}`, payload)
}

export function updateAdminOperatorStatus(userId, payload) {
  return http.post(`/api/v1/admin/operators/${userId}/status`, payload)
}

export function listAdminModulePermissions() {
  return http.get('/api/v1/admin/module-permissions')
}

export function updateAdminRoleModulePermissions(roleCode, payload) {
  return http.post(`/api/v1/admin/module-permissions/${roleCode}`, payload)
}
