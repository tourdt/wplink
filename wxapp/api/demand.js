import request from './request'

export function createDemand(data) {
  return request({
    url: '/api/v1/purchase-demands',
    method: 'POST',
    data,
  })
}

export function listMyDemands(params = {}) {
  return request({
    url: '/api/v1/me/purchase-demands',
    method: 'GET',
    data: params,
  })
}
