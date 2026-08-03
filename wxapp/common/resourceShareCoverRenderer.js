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
    if (rendering && isSameContext(renderingContext, context)) return
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

function isSameContext(left, right) {
  return Boolean(left && right)
    && left.generation === right.generation
    && left.resourceId === right.resourceId
}
