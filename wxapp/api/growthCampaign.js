import request from './request'

export function getActiveGrowthCampaigns(options = {}) {
  return request({
    url: '/api/v1/growth-campaigns/active',
    method: 'GET',
    ...options,
  })
}

export function getGrowthTasks(merchantId, options = {}) {
  return request({
    url: `/api/v1/merchants/${merchantId}/growth-tasks`,
    method: 'GET',
    ...options,
    requireAuth: true,
  })
}
