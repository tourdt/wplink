# 商户地图点位绑定 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现商户从小程序提交地图档口绑定申请、后台审核后绑定 `map_object.merchant_id` 的闭环。

**Architecture:** 后端新增 `map_object_bind_request` 持久化申请，商户侧接口负责权限校验和申请提交，后台接口负责审核并在事务内写入点位绑定。小程序在商户资料页增加状态入口和独立绑定页，后台在拿货地图页面增加绑定审核标签页。

**Tech Stack:** Go 1.25、go-zero 风格手写地图路由、PostgreSQL、uni-app Vue3、Element Plus。

---

## 文件结构

- Create: `backend/migrations/000011_map_object_bind_requests.up.sql`
- Create: `backend/migrations/000011_map_object_bind_requests.down.sql`
- Modify: `backend/app/api/map.api`
- Modify: `backend/app/internal/model/map_model.go`
- Modify: `backend/app/internal/model/map_model_test.go`
- Create: `backend/app/internal/logic/map/binding_logic.go`
- Create: `backend/app/internal/logic/map/binding_logic_test.go`
- Modify: `backend/app/internal/server/domain_routes.go`
- Modify: `backend/app/internal/server/map_routes.go`
- Modify: `backend/app/internal/server/map_api_test.go`
- Modify: `wxapp/api/sourcingMap.js`
- Modify: `wxapp/pages.json`
- Modify: `wxapp/scripts/validate-pages.mjs`
- Modify: `wxapp/pages/merchant/profile.vue`
- Create: `wxapp/pages/merchant/map-binding.vue`
- Modify: `admin-web/src/api/sourcingMap.js`
- Modify: `admin-web/src/views/SourcingMapView.vue`

## Task 1: 数据表和契约

- [ ] 写迁移测试期望：`go test ./scripts ./app/internal/model -run 'TestVerifyMigrations|TestBuildMapBinding' -v` 在新增模型测试前失败。
- [ ] 新增 `map_object_bind_request` up/down migration，包含状态、申请材料、审核字段和 pending 唯一索引。
- [ ] 更新 `map.api`，补充商户侧绑定状态、候选点位、提交申请和后台审核接口契约。
- [ ] 运行 `go test ./scripts -run TestVerifyMigrations -v` 通过迁移检查。

## Task 2: 后端模型

- [ ] 在 `map_model_test.go` 写失败测试：创建待审核申请时记录 `merchantId/objectId/sceneCode`，审核通过时写入 `map_object.merchant_id`。
- [ ] 在 `map_model.go` 增加申请类型、状态常量、候选查询、状态查询、创建申请、列表申请、审核申请方法。
- [ ] 通过 `go test ./app/internal/model -run 'TestMapBinding|TestBuildMapObjectDerivedFields' -v`。

## Task 3: 后端逻辑和路由

- [ ] 在 `binding_logic_test.go` 写失败测试：商户重复提交待审核申请返回冲突；审核已绑定点位返回冲突；驳回缺少原因返回校验错误。
- [ ] 新增 `binding_logic.go`，实现商户绑定状态、候选查询、提交申请、后台申请列表和审核。
- [ ] 修改 `MapAPIStore`、`registerMapRoutes`，让商户侧接口使用 `requireMerchantPermission`，后台接口走现有后台 token 包装。
- [ ] 更新 `map_api_test.go` 覆盖提交申请和后台审核路由。
- [ ] 通过 `go test ./app/internal/logic/map ./app/internal/server -run 'Map|Binding' -v`。

## Task 4: 小程序商户侧

- [ ] 在 `wxapp/scripts/validate-pages.mjs` 增加新页面必需检查，先运行失败。
- [ ] 在 `wxapp/api/sourcingMap.js` 增加绑定状态、候选点位、提交申请 API。
- [ ] 在 `pages.json` 注册 `pages/merchant/map-binding`。
- [ ] 在 `merchant/profile.vue` 增加“拿货地图档口”模块、状态加载和跳转入口。
- [ ] 新建 `merchant/map-binding.vue`，支持搜索场景档口、选择候选、填写备注和提交申请。
- [ ] 通过 `node wxapp/scripts/validate-pages.mjs`。

## Task 5: 后台审核入口

- [ ] 在 `admin-web/src/api/sourcingMap.js` 增加绑定申请列表和审核 API。
- [ ] 在 `SourcingMapView.vue` 增加“绑定审核”标签页，展示申请列表、状态筛选、通过和驳回动作。
- [ ] 审核通过后刷新申请列表和当前场景点位列表，避免后台看到旧绑定状态。
- [ ] 通过 `npm --prefix admin-web run build`。

## Task 6: 最终验证

- [ ] 运行 `go test ./app/internal/model ./app/internal/logic/map ./app/internal/server ./scripts -v`。
- [ ] 运行 `node wxapp/scripts/validate-pages.mjs`。
- [ ] 运行 `npm --prefix admin-web run build`。
- [ ] 检查 `git diff --stat`，确认只包含本功能相关文件和当前已有地图渲染脏文件之外的必要改动。

## 自检

- 需求覆盖：商户发起申请、后台审核、写入点位绑定、小程序状态入口、后台审核入口均有任务覆盖。
- 协议一致：后端、wxapp、admin-web 均使用 `mapBinding`、`bindRequest`、`objectId`、`merchantId` 命名。
- 范围控制：不做自动坐标换算、不做商户直接编辑地图坐标、不覆盖已绑定点位。
