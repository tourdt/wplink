import http from './http'

export function listCityStations() {
  return http.get('/api/v1/city-stations')
}

export function listCityResourceTypes(cityCode) {
  return http.get(`/api/v1/city-stations/${cityCode}/resource-types`)
}

export function listResourceTypeConfigs(params = {}) {
  return http.get('/api/v1/admin/resource-type-configs', { params })
}

export function updateResourceTypeConfig(configId, payload) {
  return http.patch(`/api/v1/admin/resource-type-configs/${configId}`, payload)
}
