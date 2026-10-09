export const merchantIdentityOptions = [
  { label: '现货档口', value: 'stall' },
  { label: '加工工厂', value: 'factory' },
  { label: '房东 / 出租方', value: 'landlord' },
  { label: '配套服务', value: 'service_provider' },
  { label: '货源供应', value: 'supplier' },
]

export const merchantTypeText = {
  stall: '现货档口',
  factory: '加工工厂',
  landlord: '房东 / 出租方',
  service_provider: '配套服务',
  supplier: '货源供应',
}

export function merchantTypeLabel(type) {
  return merchantTypeText[type] || type
}
