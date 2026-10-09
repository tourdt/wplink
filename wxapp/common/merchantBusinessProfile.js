export const BUSINESS_ROLE_OPTIONS = [
  { value: 'stall', label: '现货档口' },
  { value: 'factory', label: '加工工厂' },
  { value: 'landlord', label: '房东 / 出租方' },
  { value: 'service_provider', label: '配套服务' },
  { value: 'supplier', label: '货源供应' },
]
export const ENTITY_KIND_OPTIONS = [{ value: 'individual', label: '个人' }, { value: 'organization', label: '企业 / 机构' }]
export const BUSINESS_ARRAY_FIELDS = [
  { role: 'factory', key: 'processingMethods', label: '加工工艺', placeholder: '例如：裁剪、缝制、整烫' },
  { role: 'landlord', key: 'propertyTypes', label: '出租类型', placeholder: '例如：厂房、仓库、商铺' },
  { role: 'service_provider', key: 'serviceItems', label: '服务项目', placeholder: '例如：物流、维修、设计' },
]
const validRole = (role) => BUSINESS_ROLE_OPTIONS.some((option) => option.value === role)
export function normalizeBusinessItems(value) {
  return [...new Set((Array.isArray(value) ? value : []).map((item) => String(item || '').trim()).filter(Boolean))]
}
export function normalizeBusinessProfile(profile = { schemaVersion: 1 }) {
  return {
    schemaVersion: profile.schemaVersion,
    entityKind: profile.entityKind || '',
    primaryRole: profile.primaryRole || '',
    roles: normalizeBusinessItems(profile.roles),
    serviceArea: String(profile.serviceArea || '').trim(),
    processingMethods: normalizeBusinessItems(profile.processingMethods),
    propertyTypes: normalizeBusinessItems(profile.propertyTypes),
    serviceItems: normalizeBusinessItems(profile.serviceItems),
  }
}
export function hasBusinessProfileStructure(profile) {
  // 结构完整与业务完成分开校验：新账号和草稿可保留空身份，但必须明确是当前版本。
  return profile?.schemaVersion === 1
    && ['entityKind', 'primaryRole', 'serviceArea'].every((key) => typeof profile[key] === 'string')
    && ['roles', 'processingMethods', 'propertyTypes', 'serviceItems'].every((key) =>
      Array.isArray(profile[key]) && profile[key].every((item) => typeof item === 'string'))
}
export function getBusinessProfile(detail) {
  const profile = detail?.businessProfile
  if (hasBusinessProfileStructure(profile)) return normalizeBusinessProfile(profile)
  // 展示页初次计算可能尚未加载资料，返回不可提交的空状态，不能把缺失/未知数据升级为 v1。
  return normalizeBusinessProfile({ schemaVersion: profile?.schemaVersion !== 1 ? profile?.schemaVersion ?? 0 : 0 })
}
export function getBusinessRoleLabel(role) {
  return BUSINESS_ROLE_OPTIONS.find((option) => option.value === role)?.label || ''
}
export function getBusinessRoleLabels(profile) {
  return normalizeBusinessItems([profile?.primaryRole, ...(profile?.roles || [])]).map(getBusinessRoleLabel).filter(Boolean)
}
export function needsBusinessCategories(profile) {
  return profile?.roles?.some((role) => ['stall', 'factory', 'supplier'].includes(role)) === true
}
export function getBusinessProfileSummary(detail) {
  const profile = getBusinessProfile(detail)
  const parts = []
  if (needsBusinessCategories(profile) && detail?.mainCategories?.length) parts.push(`主营：${detail.mainCategories.join('、')}`)
  for (const [role, key, label] of [['factory', 'processingMethods', '加工'], ['landlord', 'propertyTypes', '出租'], ['service_provider', 'serviceItems', '服务']]) {
    if (profile.roles.includes(role) && profile[key].length) parts.push(`${label}：${profile[key].join('、')}`)
  }
  return parts.join('；')
}
export function getBusinessProfileErrors(value, mainCategories = []) {
  const profile = normalizeBusinessProfile(value || {})
  const errors = {}
  if (!hasBusinessProfileStructure(value)) errors.businessProfile = '业务资料格式不完整，请刷新后重试'
  if (profile.schemaVersion !== 1) errors.businessProfile = '业务资料版本暂不支持，请刷新后重试'
  if (!ENTITY_KIND_OPTIONS.some((option) => option.value === profile.entityKind)) errors.entityKind = '请选择主体性质'
  if (!validRole(profile.primaryRole)) errors.primaryRole = '请选择主要业务'
  if (!profile.roles.length || profile.roles.some((role) => !validRole(role)) || !profile.roles.includes(profile.primaryRole)) errors.roles = '请选择有效的业务身份，并包含主要业务'
  if (needsBusinessCategories(profile) && !normalizeBusinessItems(mainCategories).length) errors.mainCategories = '请选择经营品类'
  for (const field of BUSINESS_ARRAY_FIELDS) {
    if (profile.roles.includes(field.role) && !profile[field.key].length) errors[field.key] = `请填写${field.label}`
    if (profile[field.key].length > 20 || profile[field.key].some((item) => [...item].length > 50)) errors[field.key] = `${field.label}最多20项，每项不超过50字`
  }
  if ([...profile.serviceArea].length > 200) errors.serviceArea = '服务区域不能超过200字'
  return errors
}
