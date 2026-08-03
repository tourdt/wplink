# 小程序公开供需列表统一 ResourceFeedCard 设计

## 目标

让小程序所有面向浏览的供需列表复用 `ResourceFeedCard`，以统一供应与需求信息的图片、类型、交易信息和商家信息展示结构。

## 范围

- 调整搜索页、收藏页与专题页的供需列表卡片。
- 已使用 `ResourceFeedCard` 的首页、供需市场、商家公开供需和详情同类推荐不重复修改。
- “我的发布”保留独立管理卡片，因为它承担审核状态、续期、编辑和下架等操作，不属于公开浏览列表。

## 方案

搜索页移除按 `direction` 分别渲染 `DemandCard` 与 `ResourceCard` 的分支，统一在既有 `ResourceExposure` 内渲染 `ResourceFeedCard`。

收藏页与专题页保留原有循环、曝光埋点、分页/空状态和点击跳转，仅替换卡片组件为 `ResourceFeedCard`。`ResourceFeedCard` 通过 `resourceFeedCardState.js` 已同时兼容供应与需求的展示口径，因此不需要为页面新增数据转换层。

```text
搜索 / 收藏 / 专题资源数据
          ↓
现有筛选、曝光与分页逻辑
          ↓
ResourceFeedCard
          ↓
打开对应供需详情
```

## 验收标准

1. 搜索、收藏和专题中的供应与需求都使用 `ResourceFeedCard`。
2. 上述页面不再导入或渲染 `DemandCard`、`ResourceCard`。
3. 搜索筛选、收藏切换、专题曝光、分页、空状态和详情跳转保持原行为。
4. “我的发布”继续使用其管理型卡片。

## 验证

为搜索、收藏和专题页补充或更新页面测试，确认其渲染 `ResourceFeedCard` 且保留对应的点击与曝光入口；同步更新全量流程校验中相关旧卡片断言，并执行完整小程序测试套件与页面校验。
