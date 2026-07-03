# 小程序拿货地图 Canvas 交互重构实施计划

> **给执行代理：** 实施本计划时必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务逐项执行。步骤使用 checkbox（`- [ ]`）跟踪进度。

**目标：** 将 `wxapp/pages/sourcing-map/index.vue` 的地图主交互层从 `movable-view + DOM 点位` 重构为 `canvas + 自定义手势 + 命中检测`，让单指拖动、双指缩放接近地图 App 的稳定跟手体验。

**架构：** 页面层继续负责场景、搜索、筛选、详情、附近配套、电话、微信、导航等业务逻辑；新增纯函数手势模块和命中检测模块；新增 Canvas 渲染模块承载底图和点位绘制。手势过程中只更新内存态 transform 和重绘 canvas，不更新点位 DOM、不请求接口、不替换 `rawMapObjects`。

**技术栈：** Uni App、Vue 3 `<script setup>`、微信小程序 Canvas、Node.js `node:test` 静态和纯函数验证、现有 `wxapp` 构建校验。

---

## 文件结构

- 新增：`wxapp/pages/sourcing-map/mapGesture.js`
  - 纯函数手势引擎，负责拖动、双指缩放、边界约束、屏幕坐标转地图坐标。
- 新增：`wxapp/pages/sourcing-map/mapGesture.test.mjs`
  - 验证单指拖动、双指缩放中心稳定、边界约束和坐标换算。
- 新增：`wxapp/pages/sourcing-map/mapHitTest.js`
  - 纯函数命中检测，负责 rect、point、polygon 和重叠对象优先级。
- 新增：`wxapp/pages/sourcing-map/mapHitTest.test.mjs`
  - 验证各类几何命中和重叠优先级。
- 新增：`wxapp/pages/sourcing-map/canvasRenderer.js`
  - Canvas 渲染器，负责底图、档口、POI、polygon、标签、认证徽标和选中态绘制。
- 修改：`wxapp/pages/sourcing-map/index.vue`
  - 替换主地图模板、接入 canvas、接入手势和命中检测，保留现有业务动作。
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
  - 增加 Canvas 架构、手势期间不请求接口、业务动作保留的源码断言。

---

## Task 1: 先写失败测试

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.test.mjs`
- Create: `wxapp/pages/sourcing-map/mapGesture.test.mjs`
- Create: `wxapp/pages/sourcing-map/mapHitTest.test.mjs`

- [ ] **Step 1: 增加页面架构断言**

在 `index.test.mjs` 中新增测试，锁定以下要求：

- 地图主体存在 `canvas-id="sourcingMapCanvas"`、`map-canvas-shell`、`map-canvas`。
- 页面导入或使用 `canvasRenderer.js`、`mapGesture.js`、`mapHitTest.js`。
- 主地图不再使用 `movable-area`、`movable-view` 承载点位。
- 模板中不再通过 `v-for="entry in polygonObjects"`、`v-for="entry in rectAndPointObjects"` 渲染地图点位 DOM。
- 存在 `handleCanvasTouchStart`、`handleCanvasTouchMove`、`handleCanvasTouchEnd`、`handleCanvasTouchCancel`、`handleCanvasTap`。

- [ ] **Step 2: 增加交互约束断言**

在 `index.test.mjs` 中新增测试，锁定以下要求：

- 手势处理中不直接调用 `loadSceneObjects`。
- 手势结束后通过防抖或延迟函数触发静默视口加载。
- 视口请求仍使用请求序号，旧请求不能覆盖新视口数据。
- Canvas 点击命中后复用 `selectMapObject`，详情、附近配套和联系动作不另起业务分支。

- [ ] **Step 3: 增加手势纯函数测试**

`mapGesture.test.mjs` 覆盖：

- 单指从 `(100, 100)` 拖到 `(130, 150)` 后，`offsetX/offsetY` 按位移变化。
- 双指缩放从距离 `100` 到 `150` 后，`scale` 按比例放大且不超过最大值。
- 双指缩放时，缩放中心对应的地图坐标保持稳定。
- 拖动超出边界时，地图不会被完全拖出可视区域。
- `screenToMap` 能按当前 `scale/offsetX/offsetY` 反算地图坐标。

- [ ] **Step 4: 增加命中检测纯函数测试**

`mapHitTest.test.mjs` 覆盖：

- rect 对象命中 bounds 内部，不命中外部。
- point 对象按点击半径命中。
- polygon 对象使用 point-in-polygon 命中。
- 多对象重叠时优先命中认证商户、当前选中对象、面积更小的对象。

- [ ] **Step 5: 运行测试确认失败**

Run:

```bash
node wxapp/pages/sourcing-map/index.test.mjs
node wxapp/pages/sourcing-map/mapGesture.test.mjs
node wxapp/pages/sourcing-map/mapHitTest.test.mjs
```

Expected: FAIL。失败原因是 Canvas 模块和页面接入尚未实现。

## Task 2: 实现纯函数手势层

**Files:**
- Create: `wxapp/pages/sourcing-map/mapGesture.js`
- Test: `wxapp/pages/sourcing-map/mapGesture.test.mjs`

- [ ] **Step 1: 定义 transform 和配置**

导出默认配置和工具函数：

- `createInitialTransform({ viewportWidth, viewportHeight, mapWidth, mapHeight })`
- `clampTransform(transform, bounds)`
- `screenToMap(point, transform)`
- `mapToScreen(point, transform)`

transform 统一使用 `{ scale, offsetX, offsetY }`，单位使用 canvas 像素坐标，避免 rpx、px 和业务地图坐标在手势层混用。

- [ ] **Step 2: 实现单指拖动**

导出：

- `startGesture(touches, transform)`
- `moveGesture(state, touches, options)`
- `endGesture(state)`

单指拖动只基于 `touch.clientX/clientY` 的差值更新 `offsetX/offsetY`，移动过程中不产生任何副作用。

- [ ] **Step 3: 实现双指缩放**

双指缩放使用两指中心作为锚点：

- 记录开始时两指中心的地图坐标。
- 根据两指距离比例计算新 `scale`。
- 反推新的 `offsetX/offsetY`，保证锚点在屏幕上的位置稳定。
- 通过 `minScale/maxScale` 限制缩放范围。

- [ ] **Step 4: 实现边界约束**

边界规则：

- 地图大于视口时，至少保留可视区域覆盖地图内容，不允许出现整屏空白。
- 地图小于视口时，地图居中。
- 可预留少量阻尼空间，但 `endGesture` 后要回到合法边界。

- [ ] **Step 5: 跑手势测试**

Run:

```bash
node wxapp/pages/sourcing-map/mapGesture.test.mjs
```

Expected: PASS。

## Task 3: 实现命中检测层

**Files:**
- Create: `wxapp/pages/sourcing-map/mapHitTest.js`
- Test: `wxapp/pages/sourcing-map/mapHitTest.test.mjs`

- [ ] **Step 1: 统一几何读取**

兼容现有对象结构：

- `geometryType`
- `geometry`
- rect bounds
- point x/y
- polygon points

缺失或异常几何直接跳过，不抛出阻断页面的异常。

- [ ] **Step 2: 实现几何命中**

导出：

- `hitTestMapObjects(objects, mapPoint, options)`
- `isPointInRect(mapPoint, object)`
- `isPointNearPoint(mapPoint, object, radius)`
- `isPointInPolygon(mapPoint, object)`

- [ ] **Step 3: 实现重叠优先级**

命中候选排序规则：

1. 当前选中对象。
2. 认证商户。
3. 面积更小的 rect 或 polygon。
4. 点位对象。
5. 原始列表顺序。

- [ ] **Step 4: 跑命中检测测试**

Run:

```bash
node wxapp/pages/sourcing-map/mapHitTest.test.mjs
```

Expected: PASS。

## Task 4: 实现 Canvas 渲染层

**Files:**
- Create: `wxapp/pages/sourcing-map/canvasRenderer.js`

- [ ] **Step 1: 定义渲染器接口**

导出 `createSourcingMapRenderer(options)`，对页面暴露：

- `init({ canvasId, component, width, height, pixelRatio })`
- `setScene(scene)`
- `setObjects(objects)`
- `setSelectedObject(object)`
- `render(transform, options)`
- `dispose()`

页面只调用这些方法，不在 `index.vue` 中散落 canvas 绘制细节。

- [ ] **Step 2: 实现基础绘制**

绘制顺序：

1. 背景色或底图。
2. polygon 区域。
3. rect 档口。
4. point POI。
5. 选中态。

底图加载失败时绘制浅色背景，不阻断点位绘制。

- [ ] **Step 3: 实现精细绘制**

非手势状态下补充：

- 档口名称。
- POI 名称。
- 认证徽标。
- 选中对象高亮边框。

文字绘制需要按当前 scale 控制密度，避免小比例下大量文字造成卡顿和重叠。

- [ ] **Step 4: 实现手势轻量模式**

`render(transform, { interacting: true })` 只绘制底图、简化图形和选中态，不绘制全部标签和认证徽标。

## Task 5: 页面接入 Canvas

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.vue`
- Test: `wxapp/pages/sourcing-map/index.test.mjs`

- [ ] **Step 1: 替换地图主体模板**

将 `map-card` 内的 `movable-area/movable-view/map-stage` 点位 DOM 替换为：

```vue
<view
  class="map-canvas-shell"
  @touchstart="handleCanvasTouchStart"
  @touchmove.stop.prevent="handleCanvasTouchMove"
  @touchend="handleCanvasTouchEnd"
  @touchcancel="handleCanvasTouchCancel"
>
  <canvas
    id="sourcingMapCanvas"
    canvas-id="sourcingMapCanvas"
    class="map-canvas"
    :style="mapCanvasStyle"
    @tap="handleCanvasTap"
  />
</view>
```

保留 `map-overlay-info`、刷新、归位、结果列表和详情卡。

- [ ] **Step 2: 初始化尺寸和 renderer**

在页面生命周期中：

- 获取系统窗口宽高。
- 让地图默认铺满剩余屏幕高度。
- 使用 `100vw` 和 `overflow-x: hidden` 防止页面宽度超过屏幕。
- 初始化 renderer，并在场景、对象、选中态变化后重绘。

- [ ] **Step 3: 接入手势事件**

事件处理规则：

- `touchstart` 只初始化手势状态。
- `touchmove` 只更新 transform 并调用轻量绘制。
- `touchend/touchcancel` 提交最终 transform、精细重绘，并延迟触发视口加载。
- 手势中不更新 Vue 点位渲染状态，不触发 `mapObjects` 重算。

- [ ] **Step 4: 接入点击命中**

`handleCanvasTap` 流程：

1. 忽略刚结束的拖动或缩放误触。
2. 将 tap 坐标转为地图坐标。
3. 调用 `hitTestMapObjects`。
4. 命中后调用现有 `selectMapObject(object)`。
5. 未命中时只清理选中态或保持现状，不弹 toast。

- [ ] **Step 5: 迁移视口换算**

`buildViewportQueryParams` 使用当前 transform 反算可视 bounds：

- 左上角屏幕坐标转地图坐标。
- 右下角屏幕坐标转地图坐标。
- 按当前业务 zoom 和筛选条件请求对象。

继续保留搜索关键词模式：有关键词时以搜索结果为准，不被当前视口裁剪。

- [ ] **Step 6: 清理旧 DOM 地图状态**

删除或停止使用旧的：

- `mapGestureScale`
- `mapMoveX`
- `mapMoveY`
- `movableStageStyle`
- `mapGestureAreaStyle`
- `polygonObjects`
- `rectAndPointObjects`
- `mapObjectRenderEntries`
- `objectStyle`
- `polygonObjectStyle`

只保留结果列表、详情卡和业务所需的对象格式化方法。

## Task 6: 视口加载和体验细化

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.vue`
- Test: `wxapp/pages/sourcing-map/index.test.mjs`

- [ ] **Step 1: 手势结束后静默加载**

实现 `scheduleViewportObjectsLoad()`：

- 延迟 300 到 500ms。
- 新手势开始时取消旧定时器。
- 调用 `loadSceneObjects({ silent: true })`。
- 保持现有请求序号保护，旧请求不能覆盖新结果。

- [ ] **Step 2: 增加简单视口缓存**

缓存 key 使用：

- `sceneCode`
- zoom bucket
- bounds bucket
- 分类、特色、楼层、价格等筛选条件

缓存命中时先绘制缓存对象，再静默刷新，避免拖动后空白。

- [ ] **Step 3: 优化归位和刷新**

- `resetMapViewport` 重置 transform 到场景默认视野并精细重绘。
- `refreshCurrentMapData` 使用当前 transform 对应 bounds 刷新。
- 刷新失败保留旧点位和旧画面，只提示轻量错误。

- [ ] **Step 4: 处理 Canvas 失败兜底**

Canvas 初始化失败时：

- 展示“地图渲染失败，请刷新后重试”。
- 保留点位列表、详情和联系动作可用。
- 不恢复旧 DOM 地图，避免同时维护两套地图主渲染。

## Task 7: 全量验证

**Files:**
- Test: `wxapp/pages/sourcing-map/index.test.mjs`
- Test: `wxapp/pages/sourcing-map/mapGesture.test.mjs`
- Test: `wxapp/pages/sourcing-map/mapHitTest.test.mjs`
- Test: `wxapp/scripts/validate-pages.mjs`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 跑新增和既有页面测试**

Run:

```bash
node wxapp/pages/sourcing-map/index.test.mjs
node wxapp/pages/sourcing-map/mapGesture.test.mjs
node wxapp/pages/sourcing-map/mapHitTest.test.mjs
```

Expected: PASS。

- [ ] **Step 2: 跑小程序页面和流程校验**

Run:

```bash
npm --prefix wxapp run validate:pages
npm --prefix wxapp run validate:flows
```

Expected: PASS。

- [ ] **Step 3: 跑小程序构建**

Run:

```bash
npm --prefix wxapp run build:mp-weixin
```

Expected: PASS。若仅出现 Sass deprecation warnings，可记录但不阻断本次交付。

- [ ] **Step 4: 真机或开发者工具人工验收**

验收项：

- 默认地图铺满屏幕剩余区域，页面横向不超屏。
- 单指拖动稳定，不自动回原位。
- 双指缩放中心稳定，松手不闪烁、不跳变。
- 快速连续拖动时不触发接口抖动。
- 100 到 300 个可见点位仍能连续拖动。
- 点击 rect、point、polygon 能打开详情卡。
- 搜索、筛选、场景切换、刷新、归位、附近配套、导航、电话、复制微信保持可用。

## 自检

- 本计划对应设计文档 `docs/superpowers/specs/2026-07-03-sourcing-map-canvas-interaction-design.md` 的 Canvas 主渲染方案。
- 改动范围只覆盖拿货地图小程序页面和同目录交互模块，不改后端、不改后台标注、不改接口协议。
- 先用纯函数测试锁定手势和命中检测，再接入页面，降低重构风险。
- 验收标准围绕用户反馈的拖动回弹、缩放不跟手、默认全屏、横向超屏和地图 App 级交互体验。
