import request from './request'

export function listMessages(params = {}) {
  return request({
    url: '/api/v1/messages',
    method: 'GET',
    data: params,
    requireAuth: true,
  })
}

export function readMessage(messageId) {
  return request({
    url: `/api/v1/messages/${messageId}/read`,
    method: 'POST',
    data: {},
    requireAuth: true,
  })
}
