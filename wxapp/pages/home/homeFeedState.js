export function getHomeFeedState({
  merchantCount = 0,
  resourceCount = 0,
  hasRecommendCard = false,
} = {}) {
  const hasMerchants = Number(merchantCount) > 0
  const hasResources = Number(resourceCount) > 0 || Boolean(hasRecommendCard)

  return {
    hasMerchants,
    hasResources,
    hasAnyContent: hasMerchants || hasResources,
    showSwitcher: hasMerchants && hasResources,
    defaultTab: hasMerchants ? 'merchants' : hasResources ? 'resources' : '',
  }
}
