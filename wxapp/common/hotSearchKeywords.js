import { listHotSearchKeywords } from '../api/discovery'
import { DEFAULT_CITY_CODE } from './constants'

const HOT_SEARCH_KEYWORDS_KEY = 'wplink_hot_search_keywords'
const HOT_SEARCH_KEYWORD_LIMIT = 8

export function normalizeHotSearchKeywords(items = []) {
  const seen = new Set()
  const normalized = []
  for (const item of items || []) {
    const rawKeyword = typeof item === 'string' ? item : item?.keyword
    const keyword = String(rawKeyword || '').trim()
    if (!keyword || seen.has(keyword)) continue
    seen.add(keyword)
    normalized.push(keyword)
    if (normalized.length >= HOT_SEARCH_KEYWORD_LIMIT) break
  }
  return normalized
}

export function getCachedHotSearchKeywords() {
  const items = uni.getStorageSync(HOT_SEARCH_KEYWORDS_KEY)
  return Array.isArray(items) ? normalizeHotSearchKeywords(items) : []
}

export async function loadHotSearchKeywords(cityCode = DEFAULT_CITY_CODE) {
  const cachedItems = getCachedHotSearchKeywords()
  try {
    const resp = await listHotSearchKeywords({ cityCode })
    const items = normalizeHotSearchKeywords(resp.items || [])
    if (items.length) {
      uni.setStorageSync(HOT_SEARCH_KEYWORDS_KEY, items)
    } else {
      uni.removeStorageSync(HOT_SEARCH_KEYWORDS_KEY)
    }
    return items
  } catch {
    return cachedItems
  }
}
