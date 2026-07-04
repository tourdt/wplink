# 拿货地图档口状态分层渲染 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按方案 2 优化拿货地图档口展示，让基础档口弱化为轮廓层，认证和出租点位作为可点击状态层，并按缩放层级控制名称展示。

**Architecture:** 保持现有小程序 Canvas 架构，不改后端接口和地图对象数据结构。`canvasRenderer.js` 负责状态分层绘制，`mapHitTest.js` 负责只命中认证和出租等可点击对象，`index.vue` 复用现有详情卡和列表逻辑。

**Tech Stack:** uni-app、Vue3、Canvas 2D API、Node.js `node:test`。

---

### Task 1: Canvas 渲染状态分层

**Files:**
- Modify: `wxapp/pages/sourcing-map/canvasRenderer.test.mjs`
- Modify: `wxapp/pages/sourcing-map/canvasRenderer.js`

- [ ] **Step 1: Write the failing test**

在 `canvasRenderer.test.mjs` 增加测试：弱档口只绘制基础轮廓，不显示文字；认证商户近景显示名称，中景退化为 marker；出租档口显示“出租”。

- [ ] **Step 2: Run test to verify it fails**

Run: `node wxapp/pages/sourcing-map/canvasRenderer.test.mjs`
Expected: FAIL，因为当前 renderer 仍给认证档口直接画矩形和文字，且没有出租状态绘制。

- [ ] **Step 3: Write minimal implementation**

在 `canvasRenderer.js` 内增加轻量状态判断函数：

```js
function isRentableObject(object) {
  return object?.displayLevel === 'rentable' || object?.status === 'renting' || object?.extra?.rentalStatus === 'renting'
}
```

然后调整绘制顺序：基础轮廓、出租状态、认证 marker/label、选中态。

- [ ] **Step 4: Run test to verify it passes**

Run: `node wxapp/pages/sourcing-map/canvasRenderer.test.mjs`
Expected: PASS。

### Task 2: 点击规则收敛到可点击状态

**Files:**
- Modify: `wxapp/pages/sourcing-map/mapHitTest.test.mjs`
- Modify: `wxapp/pages/sourcing-map/mapHitTest.js`

- [ ] **Step 1: Write the failing test**

在 `mapHitTest.test.mjs` 增加测试：普通未认证档口不命中；认证档口和出租档口可命中。

- [ ] **Step 2: Run test to verify it fails**

Run: `node wxapp/pages/sourcing-map/mapHitTest.test.mjs`
Expected: FAIL，因为当前命中检测会命中所有 rect。

- [ ] **Step 3: Write minimal implementation**

在 `mapHitTest.js` 中对 rect/polygon 档口增加 `isClickableMapObject` 判断，只允许认证、highlight、出租、POI 点位命中。

- [ ] **Step 4: Run test to verify it passes**

Run: `node wxapp/pages/sourcing-map/mapHitTest.test.mjs`
Expected: PASS。

### Task 3: 页面标签和最终验证

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.vue`

- [ ] **Step 1: Keep label policy aligned**

让页面传给 renderer 的 `objectDisplayLabel` 与新缩放规则一致：远景不主动显示认证名称，搜索/选中由 renderer 兜底。

- [ ] **Step 2: Run focused verification**

Run:

```bash
node wxapp/pages/sourcing-map/canvasRenderer.test.mjs
node wxapp/pages/sourcing-map/mapHitTest.test.mjs
node wxapp/pages/sourcing-map/index.test.mjs
```

Expected: PASS。

### Self-Review

- 需求覆盖：覆盖基础档口弱轮廓、认证 marker/name、出租状态、未认证不可点、缩放隐藏认证名称。
- 无后端改动：出租识别使用现有字段兼容，不新增 API。
- 风险控制：只改拿货地图页面内部渲染和命中检测，列表、详情、导航、电话、微信逻辑保持不变。
