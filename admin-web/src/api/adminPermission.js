import http from './http'

export function listAdminOperators(params = {}) {
  return http.get('/api/v1/admin/operators', { params })
}

export function createAdminOperator(payload) {
  return http.post('/api/v1/admin/operators', payload)
}

export function updateAdminOperator(operatorId, payload) {
  return http.post(`/api/v1/admin/operators/${operatorId}`, payload)
}

export function updateAdminOperatorStatus(operatorId, payload) {
  return http.post(`/api/v1/admin/operators/${operatorId}/status`, payload)
}

export function listAdminModulePermissions() {
  return http.get('/api/v1/admin/module-permissions')
}

export function updateAdminRoleModulePermissions(roleCode, payload) {
  return http.post(`/api/v1/admin/module-permissions/${roleCode}`, payload)
}
