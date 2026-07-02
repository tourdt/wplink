# Sourcing Map Wxapp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `wxapp` 增加拿货地图入口和独立浏览页，让买手能从首页进入地图、查看已发布场景、搜索/点选档口和配套点位。

**Architecture:** 小程序通过 `wxapp/api/sourcingMap.js` 调用公开地图接口；首页快捷入口跳转到 `pages/sourcing-map/index`；地图页使用后台标注的底图尺寸和对象几何数据渲染缩放后的静态地图，第一期支持场景选择、关键词筛选、点位详情、电话拨打和微信复制。

**Tech Stack:** Uni App、Vue 3、微信小程序、Node 源码测试、Vite/uni build。

---

## 文件结构

- Create: `wxapp/api/sourcingMap.js`
  - 封装公开地图场景、对象、搜索、详情和附近配套接口。
- Create: `wxapp/pages/sourcing-map/index.vue`
  - 独立拿货地图页，包含场景切换、搜索、地图对象渲染和点位详情。
- Create: `wxapp/pages/sourcing-map/index.test.mjs`
  - 用源码断言锁定路由、入口、API 和地图页核心行为。
- Modify: `wxapp/pages.json`
  - 注册 `pages/sourcing-map/index`。
- Modify: `wxapp/pages/home/index.vue`
  - 在首页快捷入口增加“拿货地图”并跳转到独立页面。
- Modify: `wxapp/scripts/validate-pages.mjs`
  - 将地图页纳入页面存在性校验。
- Modify: `wxapp/scripts/validate-flows.mjs`
  - 将地图入口和地图浏览流纳入业务流校验。

## Task 1: API、路由和首页入口

**Files:**
- Create: `wxapp/pages/sourcing-map/index.test.mjs`
- Create: `wxapp/api/sourcingMap.js`
- Modify: `wxapp/pages.json`
- Modify: `wxapp/pages/home/index.vue`
- Modify: `wxapp/scripts/validate-pages.mjs`
- Modify: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 写失败测试**

新增 `wxapp/pages/sourcing-map/index.test.mjs`，断言：

```js
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
const apiSource = fs.readFileSync(path.join(root, 'api/sourcingMap.js'), 'utf8')
const source = fs.readFileSync(path.join(root, 'pages/sourcing-map/index.vue'), 'utf8')

test('sourcing map page is reachable from wxapp home', () => {
  assert.ok(pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/index'))
  assert.match(homeSource, /拿货地图/)
  assert.match(homeSource, /openSourcingMap/)
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
})

test('sourcing map api uses public map endpoints', () => {
  for (const token of [
    'listMapScenes',
    'getMapScene',
    'listMapObjects',
    'searchMapObjects',
    'getMapObject',
    'listNearbyPois',
    '/api/v1/map/scenes',
    '/api/v1/map/objects/search',
  ]) {
    assert.match(apiSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})
```

Run:

```bash
cd wxapp && node pages/sourcing-map/index.test.mjs
```

Expected: FAIL，因为 API、页面、路由和首页入口尚未实现。

- [ ] **Step 2: 实现 API、路由和入口**

`wxapp/api/sourcingMap.js` 提供：

```js
import request from './request'

export function listMapScenes(params = {}) {
  return request({ url: '/api/v1/map/scenes', method: 'GET', data: params, suppressErrorToast: true })
}

export function getMapScene(sceneCode, options = {}) {
  return request({ url: `/api/v1/map/scenes/${sceneCode}`, method: 'GET', ...options })
}

export function listMapObjects(sceneCode, params = {}) {
  return request({ url: `/api/v1/map/scenes/${sceneCode}/objects`, method: 'GET', data: params, suppressErrorToast: true })
}

export function searchMapObjects(params = {}) {
  return request({ url: '/api/v1/map/objects/search', method: 'GET', data: params, suppressErrorToast: true })
}

export function getMapObject(objectId, options = {}) {
  return request({ url: `/api/v1/map/objects/${objectId}`, method: 'GET', ...options })
}

export function listNearbyPois(objectId, params = {}) {
  return request({ url: `/api/v1/map/objects/${objectId}/nearby-pois`, method: 'GET', data: params, suppressErrorToast: true })
}
```

`pages.json` 注册 `pages/sourcing-map/index`，`home/index.vue` 增加快捷入口并在 `openScene` 中分支调用 `openSourcingMap()`。

- [ ] **Step 3: 验证和提交**

Run:

```bash
cd wxapp && node pages/sourcing-map/index.test.mjs
cd wxapp && npm run validate:pages
cd wxapp && npm run validate:flows
```

Expected: PASS.

Commit:

```bash
git add docs/superpowers/plans/2026-07-02-sourcing-map-wxapp.md wxapp/api/sourcingMap.js wxapp/pages.json wxapp/pages/home/index.vue wxapp/pages/sourcing-map/index.test.mjs wxapp/scripts/validate-pages.mjs wxapp/scripts/validate-flows.mjs
git commit -m "feat: add wxapp sourcing map entry"
```

## Task 2: 地图浏览页

**Files:**
- Modify: `wxapp/pages/sourcing-map/index.test.mjs`
- Create/Modify: `wxapp/pages/sourcing-map/index.vue`

- [ ] **Step 1: 写页面行为失败测试**

在 `index.test.mjs` 追加断言：

```js
test('sourcing map page loads scenes, renders objects and shows contact actions', () => {
  for (const token of [
    'onLoad',
    'onPullDownRefresh',
    'loadScenes',
    'loadSceneObjects',
    'selectScene',
    'selectMapObject',
    'submitSearch',
    'clearSearch',
    'objectStyle',
    'stageStyle',
    'selectedObject',
    'callSelectedObject',
    'copySelectedWechat',
    'nearbyPois',
    'loadNearbyPois',
    '地图暂未开放',
    '暂无匹配点位',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})
```

Run:

```bash
cd wxapp && node pages/sourcing-map/index.test.mjs
```

Expected: FAIL。

- [ ] **Step 2: 实现地图页**

地图页实现：

- `onLoad` 读取 `sceneCode` 和 `keyword` 路由参数。
- `loadScenes()` 请求已发布场景，优先选择路由 `sceneCode`，否则选择第一个场景。
- `loadSceneObjects()` 按当前场景和关键词读取对象。
- `stageStyle` 根据场景底图宽高和 `MAP_MAX_WIDTH_RPX = 690` 缩放。
- `objectStyle(object)` 根据 `geometryType` 和 `geometry` 计算矩形或点位位置。
- `selectMapObject(object)` 展示详情卡，并调用 `loadNearbyPois(object.id)`。
- `callSelectedObject()` 调用 `uni.makePhoneCall`，缺少电话时提示“该点位暂未提供电话”。
- `copySelectedWechat()` 调用 `uni.setClipboardData`，缺少微信时提示“该点位暂未提供微信”。

- [ ] **Step 3: 全量验证和提交**

Run:

```bash
cd wxapp && node pages/sourcing-map/index.test.mjs
cd wxapp && npm run validate:pages
cd wxapp && npm run validate:flows
cd wxapp && npm run build:mp-weixin
```

Expected: PASS；构建若只有体积或平台提示警告，可以提交。

Commit:

```bash
git add wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs wxapp/scripts/validate-flows.mjs
git commit -m "feat: add wxapp sourcing map page"
```

## Self-Review

- Spec coverage: 覆盖小程序第一期入口、公开接口封装、地图浏览、搜索、点位详情和基础联系动作；不覆盖实时导航、复杂缩放拖拽、路线规划和多楼层切换。
- Placeholder scan: 无 TBD/TODO/待补充。
- Type consistency: 页面统一使用 `sceneCode`、`mapObjects`、`selectedObject`、`nearbyPois`；API 函数名和后端公开接口保持一致。
