# 小程序拿货地图手势交互优化设计

## 目标

优化 `wxapp` 拿货地图页面的首屏空间和地图交互体验。默认状态下搜索过滤区应明显收敛，把主要首屏空间让给地图；地图区域应支持接近地图 App 的单指拖动和双指缩放。

## 现状判断

当前页面位于 `wxapp/pages/sourcing-map/index.vue`。搜索区包含搜索框、当前筛选提示、四组筛选项和场景 tabs，默认全部展开，导致地图首屏高度被压缩。

地图当前用 `scroll-view` 承载业务底图，通过按钮改变 `mapScale` 后重新计算底图和点位尺寸。这个方式可以横纵滚动，但不具备小程序原生双指缩放能力，拖动惯性和触摸体验也不接近地图类应用。

## 推荐方案

采用“手势优先”方案：

- 搜索区默认压缩为一行搜索框、一枚筛选按钮和一条横向快捷筛选 chip 行。
- 全量筛选组默认收起，用户点击“筛选 / 已选 N”后再展开。
- 地图容器从 `scroll-view` 改为 `movable-area + movable-view`。
- `movable-view` 开启 `direction="all"`、`scale`、`inertia`，让小程序原生承接拖动和双指缩放。
- 移除顶部标题区、地图标题行、放大/缩小/比例工具条，把刷新和归位作为地图内浮动服务按钮。
- 视口加载参数继续使用当前场景坐标系，不改变后端接口和地图对象坐标。
- 点位详情改为底部浮层样式，选中点位后不把地图整体向下挤压。

## 交互细节

搜索过滤区：

- 搜索框继续支持确认搜索和按钮搜索。
- 快捷筛选 chip 横向滚动，展示当前分类、服务和配套筛选项。
- 筛选展开后显示原有分组标题和完整 chip，保留“全部清除”。
- 有关键词或筛选时显示一行简短状态和“清除”入口。

地图区：

- 单指拖动直接移动 `movable-view`。
- 双指缩放更新 `mapScale`，并继续驱动 `mapZoomLevel`、点位标签显示和视口请求 zoom 参数。
- 不展示放大、缩小和缩放比例；用户通过手势完成缩放。
- 刷新和归位按钮浮动在地图右上角，归位回到场景配置的默认视图。
- 场景默认中心点继续通过 `defaultCenterX/defaultCenterY` 聚焦。

## 数据与接口

本次不改 API。`listMapObjects`、`searchMapObjects`、`getMapScene`、`listMapCategories` 等接口保持不变。

视口查询从原来的 `scrollLeft/scrollTop` 改为 `movable-view` 的 `x/y` 位移换算：

- `minX/minY/maxX/maxY` 仍使用场景原始坐标。
- `zoom` 仍由 `mapScale` 映射为 3、4、5。
- 关键词搜索时继续不带视口参数，避免搜索结果被当前地图视野限制。

## 验收标准

- 页面源码包含 `movable-area` 和 `movable-view`，地图不再依赖 `scroll-view` 做主要拖动容器。
- 默认筛选区不再展示四组纵向筛选，只有搜索行、快捷筛选行和必要状态提示。
- 页面顶部标题区和地图缩放工具条不再占用地图空间。
- 地图内显示场景信息浮层和刷新、归位服务按钮。
- 双指缩放事件会更新 `mapScale`，拖动事件会更新地图位移并触发防抖视口加载。
- 现有搜索、筛选、场景切换、点位选择、附近配套和导航能力保持可用。
- `wxapp/pages/sourcing-map/index.test.mjs`、`wxapp/scripts/validate-pages.mjs`、`wxapp/scripts/validate-flows.mjs` 通过。
