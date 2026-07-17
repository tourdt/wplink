import request from './request'

export function reverseGeocodeLocation(params = {}) {
  return request({
    url: '/api/v1/locations/reverse-geocode',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}
