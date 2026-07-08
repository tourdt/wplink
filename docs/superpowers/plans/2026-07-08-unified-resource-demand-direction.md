# 统一资源模型需求方向 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有 `resources` 统一资源模型中加入需求方向，让小程序支持“找资源/看需求”、发布资源和发布需求，并让后台审核、搜索和我的发布复用同一套资源生命周期。

**Architecture:** 后端以 `resources.direction` 和 `resource_type_configs.direction` 区分供给与需求，继续复用资源创建、审核、搜索、详情和指标链路。小程序保持 5 个 tab，将“资源”tab 升级为“供需”，发布 tab 先进入发布类型选择页，再复用发布表单按方向加载字段。

**Tech Stack:** Go/go-zero、PostgreSQL migration、goctl API 类型生成、uni-app/Vue 3、小程序 Node 静态流程测试。

---

### Task 1: 数据模型与后端资源方向

**Files:**
- Modify: `backend/migrations/000002_core_domain.up.sql`
- Modify: `backend/migrations/000003_seed_zhili.up.sql`
- Modify: `backend/migrations/000014_resource_type_display_names.up.sql`
- Modify: `backend/app/api/resource.api`
- Modify: `backend/app/api/city.api`
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/model/resource_type_config_model.go`
- Modify: `backend/app/internal/model/resources_model_gen.go`
- Modify: `backend/app/internal/model/resource_type_configs_model_gen.go`
- Test: `backend/app/internal/logic/resource/create_resource_logic_test.go`
- Test: `backend/app/internal/logic/resource/list_resources_logic_test.go`
- Test: `backend/app/internal/logic/city/list_resource_types_logic_test.go`
- Test: `backend/scripts/validate_migrations.test.mjs`

- [x] **Step 1: 写失败测试**
  - 增加创建需求资源测试，断言 `typeCode=buy_goods` 时响应和入库方向为 `demand`。
  - 增加列表筛选测试，断言 `direction=supply` 只返回供给、`direction=demand` 只返回需求。
  - 增加城市资源类型测试，断言接口能按方向返回类型配置。
  - 修改 migration 静态测试，允许需求方向资源类型，但继续禁止旧 `purchase_demands` 独立表。

- [x] **Step 2: 运行失败测试**
  - `go test ./app/internal/logic/resource ./app/internal/logic/city`
  - `node backend/scripts/validate_migrations.test.mjs`
  - 预期失败原因是缺少 `direction` 字段、类型配置或请求字段。

- [x] **Step 3: 实现后端最小改动**
  - 在 migration 中给 `resource_type_configs` 和 `resources` 增加 `direction varchar(32) NOT NULL DEFAULT 'supply'`。
  - 在种子中加入需求类型：`buy_goods`、`find_inventory`、`find_factory`、`find_service`。
  - 在 `.api` 中增加 `direction` 查询和响应字段，用 `goctl api go` 重新生成 types/handler。
  - 在模型查询中读取、写入、筛选方向，校验方向只能是 `supply` 或 `demand`。
  - 按项目规则添加中文友好错误和关键失败日志。

- [x] **Step 4: 运行后端测试通过**
  - `go test ./app/internal/logic/resource ./app/internal/logic/city ./app/internal/model`
  - `node backend/scripts/validate_migrations.test.mjs`

### Task 2: 小程序供需列表与卡片

**Files:**
- Modify: `wxapp/pages.json`
- Modify: `wxapp/pages/search/index.vue`
- Modify: `wxapp/pages/search/result.vue`
- Modify: `wxapp/components/ResourceCard.vue`
- Create: `wxapp/components/DemandCard.vue`
- Modify: `wxapp/api/resource.js`
- Modify: `wxapp/common/enums.js`
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `wxapp/pages/search/index.test.mjs`
- Test: `wxapp/pages/search/result.test.mjs`

- [x] **Step 1: 写失败测试**
  - 断言 tab 文案从“资源”调整为“供需”。
  - 断言供需页存在“找资源/看需求”分段，搜索占位跟随方向。
  - 断言需求列表使用 `DemandCard`，不和资源卡混排。

- [x] **Step 2: 运行失败测试**
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node wxapp/pages/search/index.test.mjs`
  - `node wxapp/pages/search/result.test.mjs`

- [x] **Step 3: 实现小程序列表**
  - `pages/search/index.vue` 改成供需市场页，按 `direction` 请求列表。
  - `pages/search/result.vue` 增加“资源/需求”分段，首页进入默认资源。
  - 新增 `DemandCard.vue`，用单据式布局展示类型、标题、品类、数量、预算、交期、城市和发布时间。
  - 保持资源和需求分开显示，不做默认混排。

- [x] **Step 4: 运行小程序测试通过**
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node wxapp/pages/search/index.test.mjs`
  - `node wxapp/pages/search/result.test.mjs`

### Task 3: 小程序发布入口与需求表单文案

**Files:**
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/components/ResourcePublishForm.vue`
- Modify: `wxapp/pages/my-resources/index.vue`
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `wxapp/pages/my-resources/index.test.mjs`

- [x] **Step 1: 写失败测试**
  - 断言发布 tab 首屏有“发布资源”和“发布需求”。
  - 断言表单按 `direction=demand` 展示“需求信息/采购要求/参考图片”。
  - 断言我的发布支持“资源发布/需求发布”分段。

- [x] **Step 2: 运行失败测试**
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node wxapp/pages/my-resources/index.test.mjs`

- [x] **Step 3: 实现发布体验**
  - 发布 tab 先展示选择中心，点击后携带 `direction` 和可选 `typeCode` 进入表单。
  - 发布表单复用现有动态字段，但按方向切换文案、占位和图片说明。
  - 我的发布增加方向筛选，需求项隐藏置顶动作，保留编辑、关闭、再发类似、详情。

- [x] **Step 4: 运行相关测试通过**
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node wxapp/pages/my-resources/index.test.mjs`

### Task 4: 首页入口和最终验证

**Files:**
- Modify: `wxapp/pages/home/index.vue`
- Modify: `backend/scripts/api_contract.test.mjs`
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `backend/scripts/api_contract.test.mjs`

- [x] **Step 1: 写失败测试**
  - 断言首页搜索文案为全局供需搜索。
  - 断言首页快捷入口包含“看需求/发需求”。
  - 断言后端 API 不恢复旧 `purchase-demands` 独立接口，但允许资源方向字段。

- [x] **Step 2: 运行失败测试**
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node backend/scripts/api_contract.test.mjs`

- [x] **Step 3: 实现首页和契约收口**
  - 首页搜索作为全局入口，进入结果页默认资源，可切需求。
  - 快捷入口补充需求入口，跳转供需页需求分段或发布需求选择。
  - 更新静态契约测试，明确“需求方向资源”是发布方案，旧独立需求系统仍禁止。

- [x] **Step 4: 全量验证**
  - `go test ./...`
  - `node backend/scripts/api_contract.test.mjs`
  - `node backend/scripts/validate_migrations.test.mjs`
  - `node wxapp/scripts/validate-flows.test.mjs`
  - `node wxapp/scripts/validate-pages.mjs`
