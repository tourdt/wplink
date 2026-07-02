# Sourcing Map Admin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `admin-web` 增加拿货地图管理页面，让运营可以创建场景、上传底图、标注矩形档口和点位 POI、保存对象并发布场景。

**Architecture:** 后台通过 `src/api/sourcingMap.js` 调用已完成的 `/api/v1/admin/map/...` 接口；`SourcingMapView.vue` 使用三栏布局，左侧场景列表，中间原生绝对定位标注画布，右侧对象和场景编辑面板。第一期不引入 Konva，先用 Vue 状态 + pointer/mouse 事件完成矩形和点位标注，后续再升级 polygon 和复杂图层工具。

**Tech Stack:** Vue 3、Element Plus、Vue Router、Axios、Node test runner、Vite。

---

## 文件结构

- Create: `admin-web/src/api/sourcingMap.js`
  - 封装地图场景、对象、分类、发布和批量生成 API。
- Modify: `admin-web/src/api/upload.js`
  - 增加 `uploadMapBackgroundImage(file)`，使用 `purpose: 'map_background'`。
- Modify: `admin-web/src/api/uploadUtils.js`
  - 让七牛错误消息支持自定义资源名，默认仍是“封面”。
- Modify: `admin-web/src/api/uploadUtils.test.mjs`
  - 保留封面默认文案，并补底图上传错误文案测试。
- Create: `admin-web/src/views/SourcingMapView.vue`
  - 地图管理主页面，包含场景列表、场景表单、标注画布、对象表单、批量生成抽屉。
- Modify: `admin-web/src/router/index.js`
  - 注册 `/sourcing-map` 路由。
- Modify: `admin-web/src/layouts/AdminLayout.vue`
  - 侧边栏新增“拿货地图”入口。
- Modify: `admin-web/scripts/feature-visibility.test.mjs`
  - 补充后台拿货地图入口、API 封装和页面关键控件测试。

## Task 1: API 封装和上传工具

**Files:**
- Create: `admin-web/src/api/sourcingMap.js`
- Modify: `admin-web/src/api/upload.js`
- Modify: `admin-web/src/api/uploadUtils.js`
- Modify: `admin-web/src/api/uploadUtils.test.mjs`

- [ ] **Step 1: 写失败测试**

在 `admin-web/src/api/uploadUtils.test.mjs` 追加：

```js
test('builds map background upload error message', () => {
  assert.equal(
    buildQiniuUploadErrorMessage(631, '{"error":"bucket not found"}', '底图'),
    '底图上传失败（七牛 631：bucket not found）',
  )
})
```

Run:

```bash
cd admin-web && node src/api/uploadUtils.test.mjs
```

Expected: FAIL，因为 `buildQiniuUploadErrorMessage` 还不支持第三个参数。

- [ ] **Step 2: 实现上传工具和地图 API**

`uploadUtils.js` 将函数签名调整为：

```js
export function buildQiniuUploadErrorMessage(status, bodyText = '', resourceName = '封面') {
  const detail = parseQiniuErrorDetail(bodyText)
  if (!detail) return `${resourceName}上传失败（七牛 ${status}）`
  return `${resourceName}上传失败（七牛 ${status}：${detail}）`
}
```

`upload.js` 增加：

```js
export async function uploadMapBackgroundImage(file) {
  const token = await createUploadToken({
    purpose: 'map_background',
    fileName: file.name || 'map-background.png',
    contentType: inferContentType(file),
    fileSize: file.size || 1,
  })
  const formData = new FormData()
  formData.append('token', token.uploadToken)
  formData.append('key', token.objectKey)
  formData.append('file', file)
  const resp = await fetch(token.uploadUrl, { method: 'POST', body: formData })
  if (!resp.ok) {
    const bodyText = await resp.text().catch(() => '')
    throw new Error(buildQiniuUploadErrorMessage(resp.status, bodyText, '底图'))
  }
  return buildUploadedFileUrl(token)
}
```

`sourcingMap.js` 提供：

```js
import http from './http'

export function listMapScenes(params = {}) {
  return http.get('/api/v1/admin/map/scenes', { params })
}

export function saveMapScene(payload) {
  if (payload.code) return http.post(`/api/v1/admin/map/scenes/${payload.code}`, payload)
  return http.post('/api/v1/admin/map/scenes', payload)
}

export function publishMapScene(sceneCode) {
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/publish`)
}

export function listMapObjects(sceneCode, params = {}) {
  return http.get(`/api/v1/admin/map/scenes/${sceneCode}/objects`, { params })
}

export function saveMapObject(sceneCode, payload) {
  if (payload.id) return http.post(`/api/v1/admin/map/objects/${payload.id}`, payload)
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/objects`, payload)
}

export function updateMapObjectStatus(objectId, status) {
  return http.post(`/api/v1/admin/map/objects/${objectId}/status`, { status })
}

export function batchGenerateMapObjects(sceneCode, payload) {
  return http.post(`/api/v1/admin/map/scenes/${sceneCode}/objects/batch-generate`, payload)
}

export function listMapCategories(params = {}) {
  return http.get('/api/v1/admin/map/categories', { params })
}
```

- [ ] **Step 3: 验证 API 工具测试**

Run:

```bash
cd admin-web && node src/api/uploadUtils.test.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add admin-web/src/api/sourcingMap.js admin-web/src/api/upload.js admin-web/src/api/uploadUtils.js admin-web/src/api/uploadUtils.test.mjs
git commit -m "feat: add admin sourcing map api client"
```

## Task 2: 路由、侧边栏和可见性测试

**Files:**
- Modify: `admin-web/scripts/feature-visibility.test.mjs`
- Modify: `admin-web/src/router/index.js`
- Modify: `admin-web/src/layouts/AdminLayout.vue`
- Create: `admin-web/src/views/SourcingMapView.vue`

- [ ] **Step 1: 写失败测试**

在 `feature-visibility.test.mjs` 追加：

```js
test('sourcing map admin is configurable from admin web', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/sourcingMap.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  assert.match(routeSource, /SourcingMapView/)
  assert.match(routeSource, /sourcing-map/)
  assert.match(layoutSource, /<span>拿货地图<\/span>/)
  assert.match(apiSource, /\/api\/v1\/admin\/map\/scenes/)
  assert.match(viewSource, /<h2>拿货地图<\/h2>/)
  assert.match(viewSource, /添加档口/)
  assert.match(viewSource, /添加配套/)
  assert.match(viewSource, /批量生成/)
  assert.match(viewSource, /map-canvas/)
})
```

Run:

```bash
cd admin-web && node scripts/feature-visibility.test.mjs
```

Expected: FAIL，因为路由、侧边栏和页面尚未实现。

- [ ] **Step 2: 创建最小页面并注册路由**

`SourcingMapView.vue` 先包含页面标题、三个按钮和 `.map-canvas` 容器。

`router/index.js` import `SourcingMapView`，children 增加：

```js
{ path: 'sourcing-map', name: 'sourcingMap', component: SourcingMapView },
```

`AdminLayout.vue` 引入 `MapLocation` 图标并新增菜单项：

```vue
<el-menu-item index="/sourcing-map">
  <el-icon><MapLocation /></el-icon>
  <span>拿货地图</span>
</el-menu-item>
```

- [ ] **Step 3: 验证可见性测试**

Run:

```bash
cd admin-web && node scripts/feature-visibility.test.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add admin-web/scripts/feature-visibility.test.mjs admin-web/src/router/index.js admin-web/src/layouts/AdminLayout.vue admin-web/src/views/SourcingMapView.vue
git commit -m "feat: add admin sourcing map entry"
```

## Task 3: 场景列表和场景表单

**Files:**
- Modify: `admin-web/src/views/SourcingMapView.vue`

- [ ] **Step 1: 写页面源码测试**

在 `feature-visibility.test.mjs` 的拿货地图测试中补充断言：

```js
assert.match(viewSource, /listMapScenes/)
assert.match(viewSource, /saveMapScene/)
assert.match(viewSource, /publishMapScene/)
assert.match(viewSource, /uploadMapBackgroundImage/)
assert.match(viewSource, /v-model="sceneForm\.backgroundUrl"/)
```

Run:

```bash
cd admin-web && node scripts/feature-visibility.test.mjs
```

Expected: FAIL。

- [ ] **Step 2: 实现场景列表和表单**

`SourcingMapView.vue` 实现：

- `onMounted(loadScenes)`。
- 左侧 `el-table` 显示场景 code/name/status。
- 场景表单字段：`code/name/type/cityCode/backgroundUrl/width/height/defaultScale/defaultCenterX/defaultCenterY/status`。
- 底图上传使用 `el-upload` 的 `http-request` 自定义上传，调用 `uploadMapBackgroundImage`。
- 保存调用 `saveMapScene`。
- 发布调用 `publishMapScene`。

- [ ] **Step 3: 验证测试**

Run:

```bash
cd admin-web && node scripts/feature-visibility.test.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add admin-web/scripts/feature-visibility.test.mjs admin-web/src/views/SourcingMapView.vue
git commit -m "feat: add sourcing map scene management"
```

## Task 4: 原生标注画布和对象表单

**Files:**
- Modify: `admin-web/src/views/SourcingMapView.vue`

- [ ] **Step 1: 写页面源码测试**

在拿货地图测试中补充：

```js
assert.match(viewSource, /listMapObjects/)
assert.match(viewSource, /saveMapObject/)
assert.match(viewSource, /startDragObject/)
assert.match(viewSource, /handleCanvasClick/)
assert.match(viewSource, /geometryType/)
assert.match(viewSource, /objectForm\.geometry/)
```

Expected: FAIL。

- [ ] **Step 2: 实现对象加载、添加、选择和拖动**

实现：

- 选择场景后 `loadObjects(scene.code)`。
- 画布背景使用 `selectedScene.backgroundUrl`。
- `.map-object.booth` 渲染矩形。
- `.map-object.poi` 渲染点位。
- `添加档口`：创建 rect 默认 `x:100,y:100,width:80,height:50`。
- `添加配套`：创建 point 默认 `x:160,y:160`，type 默认 `packing_station`。
- 点击对象选中并填充右侧对象表单。
- 对象表单保存调用 `saveMapObject(selectedScene.code, objectForm)`。
- 对象拖动更新 `geometry.x/y`，释放后不自动保存，运营需要点保存。

- [ ] **Step 3: 验证测试**

Run:

```bash
cd admin-web && node scripts/feature-visibility.test.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add admin-web/scripts/feature-visibility.test.mjs admin-web/src/views/SourcingMapView.vue
git commit -m "feat: add sourcing map annotation canvas"
```

## Task 5: 批量生成和构建验证

**Files:**
- Modify: `admin-web/src/views/SourcingMapView.vue`

- [ ] **Step 1: 写页面源码测试**

在拿货地图测试中补充：

```js
assert.match(viewSource, /batchGenerateMapObjects/)
assert.match(viewSource, /batchForm\.startCode/)
assert.match(viewSource, /direction/)
```

Expected: FAIL。

- [ ] **Step 2: 实现批量生成抽屉**

实现：

- 批量表单字段：`startCode/count/direction/startX/startY/width/height/gap/type/layer/categoryCodes/serviceTags`。
- 点击“批量生成”打开抽屉。
- 提交调用 `batchGenerateMapObjects(selectedScene.code, batchForm)`。
- 成功后刷新对象列表。

- [ ] **Step 3: 跑测试和构建**

Run:

```bash
cd admin-web && node src/api/uploadUtils.test.mjs
cd admin-web && node scripts/feature-visibility.test.mjs
cd admin-web && npm run build
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add admin-web/src/views/SourcingMapView.vue admin-web/scripts/feature-visibility.test.mjs
git commit -m "feat: add sourcing map batch generation"
```

## Self-Review

- Spec coverage: 本计划覆盖方案 C 第一期后台场景管理、底图上传、矩形和点位标注、对象保存、发布、批量生成；不覆盖 Konva、polygon、多人协作和小程序页面。
- Placeholder scan: 本计划没有未定项；每个任务包含文件、测试命令、实现内容和提交点。
- Type consistency: API 函数统一使用 `sourcingMap`，页面统一使用 `sceneForm`、`objectForm`、`batchForm`，路由统一为 `/sourcing-map`。
