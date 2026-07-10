import http from './http'

export function listVIPPlanConfigs() {
  return http.get('/api/v1/admin/vip/plans')
}

export function createVIPPlanConfig(payload) {
  return http.post('/api/v1/admin/vip/plans', payload)
}

export function updateVIPPlanConfig(planCode, payload) {
  return http.post(`/api/v1/admin/vip/plans/${planCode}`, payload)
}

export function listQuotaPackConfigs() {
  return http.get('/api/v1/admin/vip/quota-packs')
}

export function createQuotaPackConfig(payload) {
  return http.post('/api/v1/admin/vip/quota-packs', payload)
}

export function updateQuotaPackConfig(packCode, payload) {
  return http.post(`/api/v1/admin/vip/quota-packs/${packCode}`, payload)
}

export function listVIPPromotionConfigs() {
  return http.get('/api/v1/admin/vip/promotions')
}

export function createVIPPromotionConfig(payload) {
  return http.post('/api/v1/admin/vip/promotions', payload)
}

export function updateVIPPromotionConfig(promotionCode, payload) {
  return http.post(`/api/v1/admin/vip/promotions/${promotionCode}`, payload)
}
