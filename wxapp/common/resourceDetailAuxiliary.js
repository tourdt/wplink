export function createResourceDetailAuxiliaryLoader({
  getFavoriteState,
  hasAuthToken,
  listRelatedResources,
  setFavorited,
  setRelatedResources,
  setShareImageUrl,
}) {
  let currentGeneration = 0
  let currentResourceId = ''

  function begin(resourceId) {
    currentGeneration += 1
    currentResourceId = String(resourceId || '')
    setRelatedResources([])
    setFavorited(false)
    setShareImageUrl('')
    return { generation: currentGeneration, resourceId: currentResourceId }
  }

  function isCurrent(context) {
    return Boolean(context)
      && context.generation === currentGeneration
      && context.resourceId === currentResourceId
  }

  function run(context, {
    initializeSharing,
    isOwnResource,
    loadMerchantProfile,
    recordResourceDetailView,
  }) {
    // 旧生命周期即使在详情主体请求完成后才恢复，也不能清空或覆盖当前详情的推荐状态。
    if (!isCurrent(context)) return Promise.resolve([])

    // 分享入口不等待商家资料、推荐、浏览或收藏等低优先级请求。
    initializeSharing(context)
    const auxiliaryTasks = [
      Promise.resolve().then(() => loadMerchantProfile(context)),
      loadRelatedResourcesForContext(context, isOwnResource),
    ]
    if (!isOwnResource) {
      auxiliaryTasks.push(
        Promise.resolve().then(() => recordResourceDetailView(context.resourceId)),
        loadFavoriteStateForContext(context),
      )
    }
    return Promise.allSettled(auxiliaryTasks)
  }

  async function loadFavoriteStateForContext(context) {
    // 收藏按钮会基于当前布尔值执行相反操作，因此无 token 或旧响应都不能沿用上一详情状态。
    if (!isCurrent(context) || !hasAuthToken()) return
    try {
      const response = await getFavoriteState(context.resourceId)
      if (isCurrent(context)) setFavorited(Boolean(response?.favorited))
    } catch (err) {
      if (isCurrent(context)) setFavorited(false)
    }
  }

  async function loadRelatedResourcesForContext(context, isOwnResource) {
    // 清空也属于状态写入：必须由仍处于当前 generation 的详情发起。
    if (!isCurrent(context)) return
    setRelatedResources([])
    try {
      const response = await listRelatedResources(
        context.resourceId,
        { pageSize: 3 },
        {
          suppressErrorToast: true,
          requireAuth: Boolean(isOwnResource),
        },
      )
      if (!isCurrent(context)) return
      setRelatedResources(Array.isArray(response?.items) ? response.items : [])
    } catch (err) {
      // 静默降级为空数组，且不能由迟到请求擦除当前详情已经加载出的推荐。
      if (isCurrent(context)) setRelatedResources([])
    }
  }

  return { begin, isCurrent, run }
}
