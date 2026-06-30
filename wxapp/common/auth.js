import { getSession } from '../store/session'

const LOGIN_PAGE = '/pages/login/index'
const EMPTY_OPTIONS = {}

export function isLoggedIn() {
  return Boolean(getSession().token)
}

export function getCurrentPageUrl() {
  if (typeof getCurrentPages !== 'function') return ''
  const pages = getCurrentPages()
  const current = pages[pages.length - 1]
  if (!current?.route) return ''
  const query = buildQuery(current.options || EMPTY_OPTIONS)
  return `/${current.route}${query ? `?${query}` : ''}`
}

export function buildLoginUrl(redirect) {
  const redirectUrl = redirect || getCurrentPageUrl()
  if (!redirectUrl) return LOGIN_PAGE
  return `${LOGIN_PAGE}?redirect=${encodeURIComponent(redirectUrl)}`
}

export function redirectToLogin(options = EMPTY_OPTIONS) {
  if (getCurrentPageUrl().startsWith(LOGIN_PAGE)) return
  uni.navigateTo({ url: buildLoginUrl(options.redirect) })
}

export function requireLogin(options = EMPTY_OPTIONS) {
  if (isLoggedIn()) return true
  redirectToLogin(options)
  return false
}

function buildQuery(options) {
  return Object.keys(options)
    .filter((key) => options[key] !== undefined && options[key] !== null && options[key] !== '')
    .map((key) => `${encodeURIComponent(key)}=${encodeURIComponent(options[key])}`)
    .join('&')
}
