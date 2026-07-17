import { normalizeApiBaseUrl } from './url.js'

export const API_BASE_URL = normalizeApiBaseUrl(import.meta.env?.VITE_API_BASE_URL || '')

export const DEFAULT_CITY_CODE = 'zhili'

export const DEFAULT_CITY_LOCATION = {
  name: '织里童装城',
  latitude: 30.8732,
  longitude: 120.2255,
}

export const STORAGE_KEYS = {
  token: 'wplink_token',
  userId: 'wplink_user_id',
  merchantId: 'wplink_merchant_id',
}
