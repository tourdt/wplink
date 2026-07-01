export function buildHotSearchKeywordPayload(payload = {}) {
  return {
    cityCode: payload.cityCode ?? '',
    keyword: payload.keyword ?? '',
    startAt: payload.startAt ?? '',
    endAt: payload.endAt ?? '',
    sortOrder: payload.sortOrder ?? 0,
    status: payload.status ?? '',
  }
}
