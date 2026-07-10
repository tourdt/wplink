import http from './http'

export function listGrowthCampaigns(params = {}) {
  return http.get('/api/v1/admin/growth-campaigns', { params })
}

export function createGrowthCampaign(payload) {
  return http.post('/api/v1/admin/growth-campaigns', payload)
}

export function updateGrowthCampaign(campaignCode, payload) {
  return http.post(`/api/v1/admin/growth-campaigns/${campaignCode}`, payload)
}

export function listGrowthRules(campaignCode) {
  return http.get(`/api/v1/admin/growth-campaigns/${campaignCode}/rules`)
}

export function createGrowthRule(campaignCode, payload) {
  return http.post(`/api/v1/admin/growth-campaigns/${campaignCode}/rules`, payload)
}

export function updateGrowthRule(campaignCode, ruleCode, payload) {
  return http.post(`/api/v1/admin/growth-campaigns/${campaignCode}/rules/${ruleCode}`, payload)
}

export function listGrowthRewardGrants(campaignCode, params = {}) {
  return http.get(`/api/v1/admin/growth-campaigns/${campaignCode}/grants`, { params })
}
