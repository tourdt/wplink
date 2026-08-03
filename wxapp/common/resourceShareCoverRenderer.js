export function createResourceShareCoverRenderer({
  getFallbackCover,
  isCurrent,
  onError = () => {},
  renderCover,
  setShareImageUrl,
}) {
  let rendering = false
  let renderingContext = null
  let pendingContext = null

  function request(context) {
    if (!isCurrent(context)) return
    // 同一详情的摘要与完整商家资料会形成不同快照；仅同一快照对象才是重复请求。
    // 单槽 pending 会继续覆盖为最新快照，避免忙碌期间丢更新或积累无界渲染任务。
    if (context === renderingContext || context === pendingContext) return
    pendingContext = context
    void runLatest()
  }

  async function runLatest() {
    if (rendering) return
    const context = pendingContext
    pendingContext = null
    if (!context || !isCurrent(context)) return

    rendering = true
    renderingContext = context
    try {
      const imageUrl = await renderCover(context)
      if (isCurrent(context)) setShareImageUrl(imageUrl || getFallbackCover(context))
    } catch (err) {
      onError(err, context)
      if (isCurrent(context)) setShareImageUrl(getFallbackCover(context))
    } finally {
      rendering = false
      renderingContext = null
      // 画布只能串行导出；锁释放后必须自动消费等待期间记录的最新详情。
      if (pendingContext) void runLatest()
    }
  }

  return { request }
}

export function buildResourceShareRenderContext(context, resource = {}, merchantProfile = {}) {
  return {
    ...context,
    merchant: {
      ...(resource.merchant || {}),
      ...(merchantProfile || {}),
    },
    resource,
  }
}
