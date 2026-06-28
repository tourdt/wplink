export const LOCAL_API_BASE_URL = 'http://127.0.0.1:4000'

export function normalizeApiBaseUrl(baseUrl) {
  return String(baseUrl || LOCAL_API_BASE_URL).trim().replace(/\/+$/, '')
}

export function buildApiUrl(baseUrl, path) {
  const normalizedBaseUrl = normalizeApiBaseUrl(baseUrl)
  const normalizedPath = String(path || '').startsWith('/') ? path : `/${path || ''}`
  return `${normalizedBaseUrl}${normalizedPath}`
}
