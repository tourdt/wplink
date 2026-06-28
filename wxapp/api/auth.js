import request from './request'

export function wechatLogin(data) {
  return request({
    url: '/api/v1/auth/wechat-login',
    method: 'POST',
    data,
  })
}

export function getMe() {
  return request({
    url: '/api/v1/me',
    method: 'GET',
  })
}

export function bindPhone(data) {
  return request({
    url: '/api/v1/me/phone',
    method: 'POST',
    data,
  })
}
