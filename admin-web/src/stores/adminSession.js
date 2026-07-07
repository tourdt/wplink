export const ADMIN_TOKEN_KEY = 'wplink_admin_token'
export const ADMIN_USER_KEY = 'wplink_admin_user'

export function readAdminToken() {
  return localStorage.getItem(ADMIN_TOKEN_KEY) || ''
}

export function writeAdminSession(token, user) {
  localStorage.setItem(ADMIN_TOKEN_KEY, token)
  localStorage.setItem(ADMIN_USER_KEY, JSON.stringify(user))
}

export function clearAdminSession() {
  localStorage.removeItem(ADMIN_TOKEN_KEY)
  localStorage.removeItem(ADMIN_USER_KEY)
}

export function readAdminUser() {
  try {
    return JSON.parse(localStorage.getItem(ADMIN_USER_KEY) || 'null')
  } catch {
    return null
  }
}
