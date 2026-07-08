import request from './request'

export function listCityStations() {
  return request({
    url: '/api/v1/city-stations',
    method: 'GET',
  })
}

export function listCityResourceTypes(cityCode, params = {}) {
  return request({
    url: `/api/v1/city-stations/${cityCode}/resource-types`,
    method: 'GET',
    data: params,
  })
}
