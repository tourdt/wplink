# 供需详情同类推荐复用供需列表卡片设计

## 目标

让供需详情页的“同类推荐”与供需市场列表使用同一套 `ResourceFeedCard` 视觉与信息层级，避免详情页出现另一种精简卡片样式。

## 范围

- 仅调整 `wxapp/pages/resource/detail.vue` 中同类推荐区域传给 `ResourceList` 的展示变体。
- 保持现有推荐数据来源、最多展示数量、点击跳转、资源曝光来源和空状态文案不变。
- 不修改 `ResourceFeedCard`、`ResourceList` 的公共实现，也不调整联系解锁流程。

## 方案

详情页已通过 `ResourceList` 渲染 `relatedResources`。将其 `variant` 从 `compact` 改为 `feed` 后，`ResourceList` 会走既有分支使用 `ResourceFeedCard`，与供需市场列表的卡片样式一致。

```text
详情页加载同类型资源
        ↓
relatedResources
        ↓
ResourceList（variant="feed"）
        ↓
ResourceFeedCard（供需市场同款）
        ↓
点击进入对应供需详情
```

## 验收标准

1. 有同类资源时，详情页“同类推荐”使用 `ResourceList` 的 `feed` 变体。
2. 该变体由已有 `ResourceFeedCard` 渲染，视觉结构与供需市场列表保持一致。
3. 推荐项仍可打开对应详情，且推荐数据查询与数量限制不变。

## 验证

在 `wxapp/pages/resource/detail.test.mjs` 增加源码级回归测试，断言“同类推荐”区域传入 `variant="feed"`；先观察测试在现状下失败，再改动页面使其通过，并运行详情页测试。
