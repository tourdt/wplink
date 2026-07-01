export const merchantIdentityOptions = [
  { label: '源头工厂', value: 'factory' },
  { label: '现货档口', value: 'stall' },
  { label: '库存货源', value: 'stockist' },
  { label: '配套服务', value: 'service_provider' },
  { label: '采购商', value: 'buyer' },
]

export const merchantTypeText = merchantIdentityOptions.reduce((result, item) => {
  result[item.value] = item.label
  return result
}, {})

export const verificationTypeText = {
  factory: '源头工厂认证',
  stall: '现货档口认证',
  stockist: '库存货源认证',
  service_provider: '配套服务认证',
}

export function merchantTypeLabel(type) {
  return merchantTypeText[type] || type
}
