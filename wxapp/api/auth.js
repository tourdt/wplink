import request from './request'

export function wechatLogin(data) {
  return request({
    url: '/api/v1/auth/wechat-login',
    method: 'POST',
    data,
  })
}

export function getMe(options = {}) {
  return request({
    url: '/api/v1/me',
    method: 'GET',
    ...options,
  })
}

export function sendSmsCode(data) {
  return request({
    url: '/api/v1/auth/sms-code',
    method: 'POST',
    data,
  })
}

export function bindPhone(data) {
  return request({
    url: '/api/v1/me/phone',
    method: 'POST',
    data,
  })
}

export function bindWechatPhone(data) {
  return request({
    url: '/api/v1/me/wechat-phone',
    method: 'POST',
    data,
  })
}
