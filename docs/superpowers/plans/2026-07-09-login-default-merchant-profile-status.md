# 登录默认商家资料状态实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 登录成功后自动为没有商家资料的用户初始化未完善商家，使用户可直接发布供需，并在资料未完善时隐藏供需详情中的商家主页入口。

**Architecture:** 后端在登录流程中补齐默认商家和用户绑定，商家表新增 `profile_status` 标识资料是否完善。资源仍保持 `merchant_id` 必填归属，前端按 `merchant.profileStatus` 控制详情页商家主页入口和发布前置拦截。

**Tech Stack:** Go/go-zero 风格后端、PostgreSQL 迁移、uni-app/Vue 小程序、Node `node:test`、Go `testing`。

---

### Task 1: 后端登录兜底商家

**Files:**
- Modify: `backend/migrations/000016_merchant_profile_status.up.sql`
- Modify: `backend/migrations/000016_merchant_profile_status.down.sql`
- Modify: `backend/app/internal/model/merchant_model.go`
- Modify: `backend/app/internal/logic/auth/auth_logic.go`
- Modify: `backend/app/internal/server/auth_api_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/app/internal/server/auth_api_test.go` 增加测试：登录响应里没有既有商家时，后端会创建默认商家并返回 `profileStatus=incomplete` 的 `managedMerchants[0]`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/server -run TestAuthAPIRouterInitializesDefaultMerchantForNewLogin -count=1`

Expected: FAIL，原因是当前登录不会创建默认商家或返回资料状态。

- [ ] **Step 3: 最小实现**

新增迁移字段 `merchants.profile_status`，默认 `completed` 以兼容历史商家；登录逻辑中当用户没有绑定商家时，创建一个 `profile_status=incomplete` 的默认商家并绑定 owner。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./app/internal/server -run TestAuthAPIRouterInitializesDefaultMerchantForNewLogin -count=1`

Expected: PASS。

### Task 2: 商家详情和资料完善状态

**Files:**
- Modify: `backend/app/api/merchant.api`
- Modify: `backend/app/internal/model/merchant_model.go`
- Modify: `backend/app/internal/logic/merchant/get_merchant_logic.go`
- Modify: `backend/app/internal/logic/merchant/update_merchant_logic.go`
- Modify: `backend/app/internal/logic/merchant/*_test.go`

- [ ] **Step 1: 写失败测试**

补充测试：商家详情返回 `profileStatus`；更新商家资料时将 `profile_status` 从 `incomplete` 更新为 `completed`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/merchant -count=1`

Expected: FAIL，原因是响应和更新逻辑尚未包含 `profileStatus`。

- [ ] **Step 3: 最小实现**

`MerchantDetailResp` 和模型详情补充 `ProfileStatus`，更新资料成功时将状态置为 `completed`。历史缺失值统一兜底为 `completed`。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./app/internal/logic/merchant -count=1`

Expected: PASS。

### Task 3: 小程序发布和详情展示

**Files:**
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/pages/publish/edit.vue`
- Modify: `wxapp/components/ResourcePublishForm.vue`
- Modify: `wxapp/pages/resource/detail.vue`
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/publish-success/index.vue`
- Modify: `wxapp/pages/my/index.vue`

- [ ] **Step 1: 写失败测试**

补充前端源码测试：发布入口和发布编辑页不再调用 `ensureMerchantProfileReady`；资源详情仅在 `merchantInfo.profileStatus !== 'incomplete'` 时渲染商家主页卡片。

- [ ] **Step 2: 运行测试确认失败**

Run: `npm test -- --runInBand` 不适用本项目，改用 `node --test pages/resource/detail.test.mjs pages/my/index.test.mjs pages/publish-success/index.test.mjs`，工作目录 `wxapp`。

Expected: FAIL，原因是发布仍拦截商家资料，详情仍总是渲染商家卡片。

- [ ] **Step 3: 最小实现**

移除发布入口和编辑页的商家资料前置拦截；表单仍使用登录返回并保存的 `merchantId`。资源详情新增 `showMerchantHomeEntry`，未完善资料时不渲染商家卡片。

- [ ] **Step 4: 运行测试确认通过**

Run: `node --test pages/resource/detail.test.mjs pages/my/index.test.mjs pages/publish-success/index.test.mjs`

Expected: PASS。

### Task 4: 全量验证

**Files:**
- Verify touched backend and wxapp tests.

- [ ] **Step 1: 格式化 Go 文件**

Run: `gofmt -w backend/app/internal/model/merchant_model.go backend/app/internal/logic/auth/auth_logic.go backend/app/internal/logic/merchant/get_merchant_logic.go backend/app/internal/logic/merchant/update_merchant_logic.go backend/app/internal/server/auth_api_test.go`

- [ ] **Step 2: 后端测试**

Run: `go test ./app/internal/logic/auth ./app/internal/logic/merchant ./app/internal/server`

- [ ] **Step 3: 小程序测试**

Run: `node --test pages/resource/detail.test.mjs pages/my/index.test.mjs pages/publish-success/index.test.mjs`

- [ ] **Step 4: 验证结果**

所有命令通过，且未引入无关文件改动。
