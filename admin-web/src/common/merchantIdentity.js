export const merchantIdentityOptions = [
  { label: '个人', value: 'individual' },
  { label: '源头工厂', value: 'factory' },
  { label: '库存货源', value: 'stockist' },
  { label: '配套服务', value: 'service_provider' },
  { label: '采购', value: 'buyer' },
]

export const merchantTypeText = {
  individual: '个人',
  rental_provider: '场地/设备方',
  factory: '源头工厂',
  stall: '现货档口',
  stockist: '库存货源',
  service_provider: '配套服务',
  buyer: '采购',
}

export const verificationTypeText = {
  factory: '源头工厂认证',
  stall: '现货档口认证',
  stockist: '库存货源认证',
  service_provider: '配套服务认证',
}

export function merchantTypeLabel(type) {
  return merchantTypeText[type] || type
}
