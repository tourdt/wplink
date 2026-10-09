export function getMyStorefrontState({ token = '', merchantId = '', loading = false, error = false, merchant = {} } = {}) {
  if (!token) return 'guest'
  if (loading) return 'loading'
  if (error) return 'error'
  if (!merchantId) return 'unregistered'
  if (merchant.profileStatus === 'incomplete') return 'incomplete'
  if (merchant.ownerVisible === false) return 'paused'
  // 地图公开与主页公开是独立能力，无固定接待地点的主体仍可正常展示业务主页。
  if (merchant.mapVisible !== true) return 'profile_only'
  if (merchant.location?.status !== 'visible' || merchant.location?.available !== true) return 'location_error'
  return 'active'
}

export function getStorefrontPlace(detail = {}) {
  // 管理者接口包含私有地点，主页预览也按公开口径渲染，避免误导用户认为地址已公开。
  if (detail.mapVisible !== true) return { address: '', position: '', canNavigate: false, latitude: null, longitude: null }
  const location = detail.location || {}
  const position = [location.marketName, location.buildingName, location.floorNo, location.boothNo]
    .map((value) => String(value ?? '').trim())
    .filter(Boolean).join(' · ')
  const address = String(detail.addressText || '').trim() || position
  const latitude = coordinate(location.latitude, -90, 90)
  const longitude = coordinate(location.longitude, -180, 180)
  return {
    address,
    position,
    canNavigate: location.status === 'visible' && location.available === true && latitude !== null && longitude !== null,
    latitude,
    longitude,
  }
}

export function getPublicContact(detail = {}, isOwner = false) {
  const contact = detail.contact || {}
  // 管理者详情可含未公开资料；仅在对应公开开关开启时展示。访客详情已由后端筛掉未授权字段。
  return {
    phone: !isOwner || detail.publicPhone === true ? String(contact.phone || '').trim() : '',
    wechat: !isOwner || detail.publicWechat === true ? String(contact.wechat || '').trim() : '',
  }
}

function coordinate(value, minimum, maximum) {
  if (value === null || value === undefined || String(value).trim() === '') return null
  const number = Number(value)
  return Number.isFinite(number) && number >= minimum && number <= maximum ? number : null
}
