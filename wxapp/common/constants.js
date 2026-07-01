import { normalizeApiBaseUrl } from './url'

export const API_BASE_URL = normalizeApiBaseUrl(import.meta.env?.VITE_API_BASE_URL || '')

export const DEFAULT_CITY_CODE = 'zhili'

export const STORAGE_KEYS = {
  token: 'wplink_token',
  userId: 'wplink_user_id',
  merchantId: 'wplink_merchant_id',
}
