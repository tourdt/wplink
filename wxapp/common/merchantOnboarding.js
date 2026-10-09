import { buildMerchantLocationPayload } from './merchantLocationForm.js'
import { getBusinessProfileErrors } from './merchantBusinessProfile.js'
import { validateMerchantName } from './merchantName.js'

export function getMerchantOnboardingErrors(form) {
  const errors = {}
  const nameError = validateMerchantName(form?.name)
  if (nameError) errors.name = nameError
  Object.assign(errors, getBusinessProfileErrors(form?.businessProfile, form?.mainCategories))
  if (typeof form?.mapVisible !== 'boolean') errors.mapVisible = '请明确选择是否在地图上公开接待地点'
  // 主体入驻不依赖照片或 GPS；只有主动公开接待地点时校验地址和坐标。
  if (form?.mapVisible === true) {
    if (!String(form?.addressText || '').trim()) errors.addressText = '请填写详细地址'
    if (buildMerchantLocationPayload(form).locationAction !== 'set') errors.location = '请在地图上选择接待地点'
  }
  return errors
}

export function buildMerchantContactVisibility(form) {
  return {
    publicPhone: Boolean(form?.publicPhone && String(form?.contactPhone || '').trim()),
    publicWechat: Boolean(form?.publicWechat && String(form?.contactWechat || '').trim()),
  }
}

export function getMerchantSavedState(detail) {
  if (detail?.ownerVisible === false) return 'paused'
  if (detail?.mapVisible === false) return 'profile_public'
  if (detail?.location?.status === 'visible' && detail?.location?.available === true && detail?.ownerVisible === true) return 'public'
  return 'location_error'
}
