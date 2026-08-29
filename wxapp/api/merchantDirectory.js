import request from './request'

export function listMerchantDirectory(params = {}) {
  return request({
    url: '/api/v1/merchant-directory',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}

export function getMerchantLocationContext(merchantId) {
  return request({
    url: `/api/v1/merchants/${merchantPathSegment(merchantId)}/location-context`,
    method: 'GET',
    suppressErrorToast: true,
  })
}

export function submitMerchantLocationReport(merchantId, data = {}) {
  return request({
    url: `/api/v1/merchants/${merchantPathSegment(merchantId)}/location-reports`,
    method: 'POST',
    // 报告用户只能由后端从 Token 识别，前端只提交受控原因和补充说明。
    data: {
      reasonCode: String(data.reasonCode || '').trim(),
      description: String(data.description || '').trim(),
    },
    requireAuth: true,
  })
}

function merchantPathSegment(merchantId) {
  return encodeURIComponent(String(merchantId ?? '').trim())
}
