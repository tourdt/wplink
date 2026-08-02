# 商家详情公开供应卡片统一设计

日期：2026-08-02  
适用范围：衣货通小程序商家详情页

## 背景

商家详情页的“公开供应”当前通过 `ResourceList` 渲染 `ResourceCard`，而供需市场列表通过 `ResourceFeedCard` 渲染。两种卡片在封面尺寸、供需标签位置、交易信息和商家信息的层级上不一致，使同一条供需在不同入口呈现出不同的视觉语言。

## 目标与验收标准

- 商家详情页的“公开供应”与供需市场列表使用同一套 `ResourceFeedCard` 结构和样式。
- 仅改变商家详情页公开供应的卡片呈现，不影响首页、资源详情关联列表和其他 `ResourceList` 调用方。
- 保持商家详情页既有的空态、分页加载、曝光埋点和跳转行为。
- 自动化测试覆盖新增变体的组件选择与商家详情页的变体传递。

## 方案比较

1. 在商家详情页复制 `ResourceFeedCard` 的结构和样式：改动直观，但会形成两套重复实现，后续供需列表调整容易再次偏离。
2. 为 `ResourceList` 增加专用 `feed` 变体：在该变体下复用 `ResourceFeedCard`，默认仍渲染 `ResourceCard`。推荐，改动小且能保持统一。
3. 将所有 `ResourceCard` 改造成供需列表样式：会影响首页和详情页等不在本次范围内的入口，风险较高。

## 设计

### 组件边界

- `ResourceList` 保留现有的列表容器、空态、加载更多按钮与 `ResourceExposure` 埋点职责。
- 当 `variant` 为 `feed` 时，`ResourceList` 使用 `ResourceFeedCard`；其他取值继续使用 `ResourceCard`，并保留原有 `variant` 传递。
- 商家详情页在“公开供应”的 `ResourceList` 上显式传入 `variant="feed"`。

### 数据与交互

卡片继续接收原有 `resource` 数据并通过原有 `open` 事件跳转资源详情。加载更多、空态提示、分页和曝光来源 `merchant` 均不变，因此无需调整接口或页面状态。

### 测试

- 先在 `ResourceList.test.mjs` 添加断言，要求 `feed` 变体使用 `ResourceFeedCard`、默认分支仍使用 `ResourceCard`；确认测试在实现前失败。
- 在 `merchant/detail.test.mjs` 添加断言，要求公开供应列表传入 `variant="feed"`；确认测试在实现前失败。
- 实现后运行两份定向测试与小程序现有页面校验脚本。

## 范围控制

不修改资源接口、卡片数据模型、供需市场页面、`ResourceCard` 的默认视觉样式，也不增加新的交互或业务字段。
