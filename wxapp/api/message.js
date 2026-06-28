import request from './request'

export function listMessages(params = {}) {
  return request({
    url: '/api/v1/messages',
    method: 'GET',
    data: params,
  })
}

export function readMessage(messageId, userId, roleCode = '') {
  return request({
    url: `/api/v1/messages/${messageId}/read`,
    method: 'POST',
    data: { userId, roleCode },
  })
}
