export const QUOTA_TYPE_PUBLISH = 'publish_quota'
export const QUOTA_TYPE_REFRESH = 'refresh_quota'

export function buildQuotaPurchaseUrl(quotaType) {
  const type = quotaType === QUOTA_TYPE_REFRESH ? QUOTA_TYPE_REFRESH : QUOTA_TYPE_PUBLISH
  return `/pages/vip/index?tab=addons&quotaType=${type}`
}

export function confirmQuotaPurchase(quotaType) {
  const isRefresh = quotaType === QUOTA_TYPE_REFRESH
  return new Promise((resolve) => {
    uni.showModal({
      title: isRefresh ? '刷新次数已用完' : '发布次数已用完',
      content: isRefresh ? '购买刷新次数后，可继续刷新当前发布。' : '购买发布次数后，可继续提交审核。',
      confirmText: isRefresh ? '购买刷新次数' : '购买发布次数',
      cancelText: '暂不购买',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}
