import request from './request'

export function createUploadToken(data) {
  return request({
    url: '/api/v1/uploads/token',
    method: 'POST',
    data,
  })
}
