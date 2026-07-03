# 小程序拿货地图 Canvas 交互优化设计

## 目标

将 `wxapp` 拿货地图的拖动和双指缩放体验优化到接近地图 App 的交互感受：单指拖动稳定跟手，双指缩放中心稳定，手势过程中不因为 DOM 重排、接口返回或点位列表刷新产生明显卡顿、回弹或闪烁。

本设计只重构小程序地图交互层，不改变后端地图 API、后台标注工具、地图对象数据结构、搜索筛选和点位详情业务逻辑。

## 当前问题

当前页面位于 `wxapp/pages/sourcing-map/index.vue`。地图主体使用 `movable-area + movable-view`，点位使用大量 `view` 节点渲染。这个方案已经做过几轮优化，但仍存在架构瓶颈：

- `movable-view` 的 `x/y/scale-value` 是受控属性，手势期间仍需要经过 JS 状态同步，重渲染时容易产生不跟手或回弹。
- 每个档口、POI、polygon 都是 DOM 节点，且包含文字、徽标、样式和 clip-path，点位数量上来后视图层合成压力大。
- 缩放层级变化会触发 `mapObjects`、`mapObjectRenderEntries` 和标签重新计算，容易在缩放结束时卡顿。
- 视口懒加载返回后会替换点位数据，即使静默加载也会触发地图点位重渲染。
- 全屏地图增加了渲染面积，继续使用 DOM 点位会进一步放大性能压力。

## 推荐方案

采用 Canvas 地图引擎替换 DOM 点位层：

- 底图、档口矩形、POI、polygon、标签、认证标识、选中态全部绘制到 canvas。
- 搜索区、筛选区、刷新/归位按钮、点位列表、详情卡继续使用现有 Vue/DOM 实现。
- 手势中只更新 canvas 的内存态 transform，不更新 Vue 点位 DOM，不请求接口，不替换点位数组。
- 手势结束后再提交业务缩放层级、静默加载视口点位并重绘 canvas。
- 点击 canvas 后通过命中检测找到点位对象，复用现有 `selectMapObject`、附近配套、导航、电话和微信逻辑。

## 架构

### 页面层

`wxapp/pages/sourcing-map/index.vue` 继续承担页面生命周期、接口调用、筛选搜索、场景切换、详情卡展示和用户操作入口。

页面模板中的地图主体从：

```vue
<movable-area>
  <movable-view>
    <view class="map-stage">大量点位 DOM</view>
  </movable-view>
</movable-area>
```

改为：

```vue
<view class="map-canvas-shell" @touchstart="handleCanvasTouchStart" @touchmove="handleCanvasTouchMove" @touchend="handleCanvasTouchEnd" @touchcancel="handleCanvasTouchCancel">
  <canvas canvas-id="sourcingMapCanvas" id="sourcingMapCanvas" class="map-canvas" :style="mapCanvasStyle" @tap="handleCanvasTap" />
</view>
```

### Canvas 渲染层

新增 `wxapp/pages/sourcing-map/canvasRenderer.js`，负责纯绘制逻辑。

职责：

- 创建和缓存 canvas context。
- 根据场景底图、点位列表、选中点位和 transform 绘制地图。
- 绘制底图、矩形档口、点状 POI、polygon、标签、认证徽标和选中态。
- 缩放中可以使用轻量绘制模式，只画底图和简单图形；手势结束后再绘制标签和复杂装饰。

### 手势控制层

新增 `wxapp/pages/sourcing-map/mapGesture.js`，负责纯手势计算。

职责：

- 记录单指拖动的起点、位移和当前 transform。
- 记录双指缩放的距离、中心点和缩放比例。
- 将屏幕坐标转换为地图坐标。
- 限制平移边界，避免地图完全拖出视口。
- 输出新的 `{ scale, offsetX, offsetY, interacting }`。

### 命中检测层

新增 `wxapp/pages/sourcing-map/mapHitTest.js`，负责从地图坐标命中点位。

职责：

- rect：判断地图坐标是否落在矩形 bounds 内。
- point：按固定点击半径判断。
- polygon：使用 point-in-polygon 判断。
- 多个对象重叠时，优先命中认证商户、选中对象、面积更小的对象，再按排序字段兜底。

## 数据流

### 首次加载

1. `onLoad` 加载分类和场景。
2. `selectScene` 获取场景详情和默认视野。
3. `loadSceneObjects` 获取首屏视口点位。
4. 页面初始化 canvas 尺寸和 renderer。
5. renderer 使用场景底图、点位和默认 transform 绘制首帧。

### 单指拖动

1. `touchstart` 记录起点和当前 transform。
2. `touchmove` 只更新手势内存态 transform。
3. 每次移动通过 renderer 重绘 canvas。
4. 不更新 `rawMapObjects`，不触发 `mapObjects` DOM 重排，不请求接口。
5. `touchend` 后 300 到 500ms 静默加载新视口点位并重绘。

### 双指缩放

1. `touchstart` 记录双指距离、缩放中心和当前 transform。
2. `touchmove` 计算新 scale，并保持缩放中心对应的地图坐标不漂移。
3. 缩放中使用轻量绘制模式，减少文字绘制和复杂徽标。
4. `touchend` 后提交业务 zoom，按 zoom 规则绘制标签和筛选可见对象。

### 点击点位

1. `tap` 获取屏幕坐标。
2. 手势控制层反算地图坐标。
3. 命中检测层找到对象。
4. 调用现有 `selectMapObject(object)`，展示详情卡并加载附近配套。

## 视口加载策略

- 手势期间不请求接口。
- 手势结束后延迟 300 到 500ms 再请求，避免连续拖动造成接口抖动。
- 请求使用递增序号，旧请求不能覆盖新视图。
- 维护简单内存缓存，缓存 key 使用 `sceneCode + zoom + bounds bucket + filters`。
- 搜索关键词模式继续走搜索接口，不受当前视口限制。

## 绘制策略

### 正常模式

绘制顺序：

1. 底图。
2. 弱展示对象。
3. 普通档口和 POI。
4. 认证和高亮对象。
5. 标签。
6. 选中态边框。

### 手势模式

绘制顺序：

1. 底图。
2. 简化档口块和 POI 点。
3. 选中态。

手势模式不绘制全部文字标签和认证徽标，降低每帧绘制成本。

## 错误处理

- canvas 初始化失败时展示“地图渲染失败，请刷新后重试”，并保留点位列表可用。
- 底图加载失败时展示浅色背景和点位，不阻断用户查看列表。
- 视口加载失败时保留旧点位和旧绘制结果，只在用户主动刷新时显示 toast。
- 点位命中失败不提示，避免用户误触时被打扰。

## 测试策略

静态测试继续使用 `wxapp/pages/sourcing-map/index.test.mjs`，新增断言：

- 页面使用 canvas，而不是 `movable-area/movable-view` 承载主地图。
- 存在 `canvasRenderer.js`、`mapGesture.js`、`mapHitTest.js`。
- 手势中不调用 `loadSceneObjects`。
- canvas 点击复用 `selectMapObject`。
- 视口加载仍为静默、带请求序号。

新增纯函数测试：

- `mapGesture.test.mjs`：拖动、双指缩放、边界限制、屏幕坐标转地图坐标。
- `mapHitTest.test.mjs`：rect、point、polygon 命中和重叠优先级。

人工验收：

- 真机单指拖动不回弹、不横向滚页面。
- 双指缩放中心稳定，松手后无明显闪烁。
- 100 到 300 个可见点位仍能连续拖动。
- 点击点位打开详情，电话、微信、导航能力保持可用。

## 分阶段范围

### 第一阶段：Canvas 主渲染闭环

目标：用 canvas 替换地图 DOM 点位层，保留现有业务能力。

覆盖：

- canvas 渲染底图、rect、point、polygon。
- 单指拖动和双指缩放。
- 点击命中后打开详情卡。

不覆盖：

- 复杂惯性动画。
- 瓦片切图。
- 路径规划。

### 第二阶段：体验细化

目标：补齐地图 App 级细节。

覆盖：

- 缩放中心稳定。
- 边界阻尼。
- 手势中简化绘制，停手后精细绘制。
- 视口缓存。

### 第三阶段：规模化

目标：支撑更大场景和更多点位。

覆盖：

- 底图分级或瓦片化。
- 点位按网格索引命中。
- 按 zoom 聚合或隐藏低价值标签。

## 验收标准

- 地图主体不再使用大量点位 DOM 节点渲染。
- 单指拖动和双指缩放手势中不触发接口请求。
- 手势中不替换 `rawMapObjects`。
- canvas 点击能命中 rect、point、polygon。
- 搜索、筛选、场景切换、刷新、归位、详情、附近配套、导航、电话、复制微信保持可用。
- `node wxapp/pages/sourcing-map/index.test.mjs` 通过。
- `npm --prefix wxapp run validate:pages` 通过。
- `npm --prefix wxapp run validate:flows` 通过。
- `npm --prefix wxapp run build:mp-weixin` 通过。
