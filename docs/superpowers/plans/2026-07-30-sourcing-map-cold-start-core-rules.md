# 拿货地图冷启动核心规则实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 落地拿货地图冷启动第一阶段的最小闭环：招聘入口暂时下线、商家主页不公开联系方式、一商家只绑定一个主档口且常规绑定自动生效、商家每月获得 3 条免费供需发布额度，并让内容安全校验失败进入自动重试而不是常规人工审核。

**架构：** 保留现有 Go/go-zero API、资源状态机和地图绑定申请表，在既有边界内收紧公开响应与数据库唯一性。地图绑定接口继续兼容原路径，但模型层在同一事务内完成点位锁定、唯一性校验、绑定和“系统自动通过”记录；历史待审核记录仍由后台接口处理。供需发布继续使用微信内容安全能力，明确风险自动驳回、依赖失败自动重试、建议复核宽松发布。

**技术栈：** Go 1.23、go-zero 1.7.6、PostgreSQL、Vue 3、uni-app、Node.js test runner。

## 全局约束

- 所有行为修改严格按 RED → GREEN → REFACTOR 执行；先运行目标测试并确认因缺少新行为而失败。
- `.api` 是接口类型源文件；涉及生成类型时使用项目现有 goctl 方式重新生成，不手改 `backend/app/internal/types/types.go`。
- 数据库并发规则必须由唯一索引兜底，Go 事务只负责给出明确业务错误与日志。
- 后端日志记录 merchantId、objectId、resourceId 等稳定标识，不记录电话、微信、OpenID 或完整请求体。
- 前端只展示可行动的中文提示，不暴露 SQL、依赖错误或内部状态名。

### 任务 1：公开商家主页彻底隐藏联系方式

**文件：**

- 修改：`backend/app/internal/logic/merchant/get_merchant_logic_test.go`
- 修改：`backend/app/internal/logic/merchant/get_merchant_logic.go`
- 修改：`backend/app/api/merchant.api`
- 生成：`backend/app/internal/types/types.go`
- 修改：`wxapp/pages/merchant/detail.test.mjs`
- 修改：`wxapp/pages/merchant/detail.vue`

**步骤 1：增加失败测试**

- 将公开商家详情测试改为断言序列化结果中不存在 `contact` 字段。
- 保留并加强商家管理员测试，断言本人管理资料时仍返回原始电话和微信。
- 小程序商家主页测试断言页面不读取 `merchant.contact`，并明确提示联系方式仅随供需信息展示。

**步骤 2：确认 RED**

运行：

```bash
cd backend && go test ./app/internal/logic/merchant -run 'TestGetMerchant(ReturnsProfileTrustAndSummary|ReturnsEditableContactForManager)' -count=1
cd wxapp && node --test pages/merchant/detail.test.mjs
```

预期：公开响应仍含 `contact`，后端测试失败。

**步骤 3：最小实现**

- 将 `MerchantDetailResp.Contact` 改为可选指针，仅在 `UserCanManageMerchant` 返回 true 时构造。
- 删除公开响应中的联系人姓名、脱敏电话和脱敏微信。
- 同步修改 `merchant.api` 中的 `Contact` 为 optional，并按项目方式生成 types。
- 商家主页只保留“联系方式随有效供需信息展示”的说明；资料编辑页继续复用管理员身份下的私有响应。

**步骤 4：确认 GREEN**

重新运行步骤 2 的测试，并运行：

```bash
cd backend && node --test scripts/api_contract.test.mjs
```

### 任务 2：招聘模块暂时下线

**文件：**

- 新增：`backend/migrations/000030_sourcing_map_cold_start_core_rules.up.sql`
- 新增：`backend/migrations/000030_sourcing_map_cold_start_core_rules.down.sql`
- 修改：`backend/app/internal/logic/city/list_resource_types_logic_test.go`
- 修改：`wxapp/pages/home/index.test.mjs`
- 修改：`wxapp/pages/home/index.vue`
- 修改：`wxapp/pages/publish/index.test.mjs`
- 修改：`wxapp/pages/publish/index.vue`
- 修改：`wxapp/scripts/validate-flows.test.mjs`
- 修改：`wxapp/scripts/validate-flows.mjs`

**步骤 1：增加失败测试**

- 城市资源类型测试断言停用类型不会出现在可发布类型响应中，并覆盖 `job_hiring`、`job_seeking`。
- 首页测试断言默认 Banner 和场景入口不再出现招聘/求职。
- 发布页与流程校验测试断言招聘不再属于冷启动主流程。

**步骤 2：确认 RED**

运行：

```bash
cd backend && go test ./app/internal/logic/city -run TestListResourceTypes -count=1
cd wxapp && node --test pages/home/index.test.mjs pages/publish/index.test.mjs scripts/validate-flows.test.mjs
```

预期：首页或流程仍包含招聘入口，前端测试失败。

**步骤 3：最小实现**

- migration 将 `job_hiring`、`job_seeking` 的 `resource_type_configs.status` 更新为 `inactive`，回滚脚本恢复为 `active`。
- 同步从织里站 `city_stations.config.enabledTypeCodes` 中移除这两个类型；回滚时仅补回缺失项。
- 删除首页招聘求职入口与默认 Banner 招聘文案。
- 删除发布页和流程校验中的招聘主流程文案；保留历史资源状态字典，确保旧招聘信息仍能正确显示。

**步骤 4：确认 GREEN**

重新运行步骤 2 的测试。

### 任务 3：一商家一主档口并自动绑定

**文件：**

- 修改：`backend/migrations/000030_sourcing_map_cold_start_core_rules.up.sql`
- 修改：`backend/migrations/000030_sourcing_map_cold_start_core_rules.down.sql`
- 修改：`backend/app/internal/model/map_model_test.go`
- 修改：`backend/app/internal/model/map_model.go`
- 修改：`backend/app/internal/logic/map/binding_logic_test.go`
- 修改：`backend/app/internal/logic/map/binding_logic.go`
- 修改：`backend/app/internal/server/map_api_test.go`
- 修改：`wxapp/pages/merchant/map-binding.test.mjs`
- 修改：`wxapp/pages/merchant/map-binding.vue`

**步骤 1：增加失败测试**

- 模型 SQL/事务测试覆盖：空闲点位自动绑定、申请记录直接为 `approved`、商家已有其他点位时返回 `ErrMapMerchantAlreadyBound`、点位被其他商家占用时返回 `ErrMapObjectAlreadyBound`。
- 逻辑测试期望提交后返回 `approved`，并分别映射“该商家已绑定主档口”和“档口已被其他商家绑定”的状态冲突提示。
- API 路由测试期望普通提交直接得到已绑定状态。
- 小程序测试断言按钮与成功提示使用“确认绑定/档口已绑定”，不再承诺常规人工审核。

**步骤 2：确认 RED**

运行：

```bash
cd backend && go test ./app/internal/model ./app/internal/logic/map ./app/internal/server -run 'MapBind|BindingLogic' -count=1
cd wxapp && node --test pages/merchant/map-binding.test.mjs
```

预期：现有实现返回 `pending`，且没有商家唯一绑定冲突，测试失败。

**步骤 3：最小实现**

- migration 在创建唯一索引前检测历史重复绑定；存在重复数据时主动报错并终止迁移，避免静默解绑。
- 添加 `map_object(merchant_id)` 的非空唯一索引，数据库层保证一个商家只能有一个主档口；一个点位仍由现有单值 `merchant_id` 保证只绑定一个商家。
- `CreateMapBindRequest` 在事务内按稳定顺序锁定商家现有绑定和目标点位，校验冲突后更新 `map_object.merchant_id`。
- 同一事务写入状态为 `approved`、`reviewed_at=now()`、`review_note='系统自动绑定'` 的绑定记录。
- 将 PostgreSQL 唯一冲突映射为稳定业务错误；逻辑层记录安全标识并返回中文可行动提示。
- 保留后台审核历史 pending 记录的能力，并为后台批准路径补充“一商家一档口”冲突检查。
- 小程序移除证明图片的强审核语义，保留可选说明，成功后立即刷新已绑定状态。

**步骤 4：确认 GREEN**

重新运行步骤 2 的测试，并运行：

```bash
cd backend && go test ./app/internal/model ./app/internal/logic/map ./app/internal/server -count=1
```

### 任务 4：每月 3 条免费供需与零常规人工审核

**文件：**

- 修改：`backend/app/internal/model/resource_model_test.go`
- 修改：`backend/app/internal/model/resource_model.go`
- 修改：`backend/app/internal/logic/resource/create_resource_logic_test.go`
- 修改：`backend/app/internal/logic/resource/create_resource_logic.go`
- 修改：`backend/app/internal/logic/resource/submit_resource_logic_test.go`
- 修改：`backend/app/internal/logic/resource/submit_resource_logic.go`
- 修改：`wxapp/pages/publish/index.test.mjs`
- 修改：`wxapp/pages/publish/index.vue`
- 修改：`wxapp/pages/publish-success/index.test.mjs`
- 修改：`wxapp/pages/publish-success/index.vue`

**步骤 1：增加失败测试**

- 额度测试断言不论资料完整度，每个商家当月基础发布额度都是 3、基础刷新额度是 0。
- 创建和提交测试断言缺少内容审核身份时进入 `audit_retry`，而不是 `manual_review`。
- 自动发布额度不足提示断言只引导购买发布次数，不引导开通 VIP。
- 小程序发布页展示“每月 3 条免费发布”，发布成功页把 pending/audit_retry 解释为自动安全检测。

**步骤 2：确认 RED**

运行：

```bash
cd backend && go test ./app/internal/model -run TestProfileMonthlyBenefitsForStatus -count=1
cd backend && go test ./app/internal/logic/resource -run 'Missing(OpenID|User)|Quota|Audit' -count=1
cd wxapp && node --test pages/publish/index.test.mjs pages/publish-success/index.test.mjs
```

预期：基础额度仍为 0，缺少 OpenID 仍转人工，测试失败。

**步骤 3：最小实现**

- `profileMonthlyBenefitsForStatus` 固定返回 `(3, 0)`，沿用现有按月幂等 entitlement 发放机制。
- 将后台代发、缺少用户或 OpenID 的资源标记为 `audit_retry`，记录依赖失败决策并交给现有重试任务；不进入常规人工审核。
- 保留明确违规自动驳回、微信“建议复核”宽松自动发布、连续重试超过上限进入异常人工队列的兜底。
- 修改发布额度不足和草稿提示，移除 VIP 主路径，改为购买发布次数。
- 小程序展示基础免费额度与自动检测状态，不展示“等待人工审核”的常规承诺。

**步骤 4：确认 GREEN**

重新运行步骤 2 的测试，并运行：

```bash
cd backend && go test ./app/internal/logic/resource ./app/internal/model -count=1
cd wxapp && node --test pages/publish/index.test.mjs pages/publish-success/index.test.mjs pages/my-resources/index.test.mjs pages/resource/detail.test.mjs
```

### 任务 5：全量验证与交付

**文件：**

- 检查：本计划涉及的全部文件

**步骤 1：格式化与差异审查**

运行：

```bash
cd backend && gofmt -w app/internal/logic/merchant/get_merchant_logic.go app/internal/logic/merchant/get_merchant_logic_test.go app/internal/logic/map/binding_logic.go app/internal/logic/map/binding_logic_test.go app/internal/model/map_model.go app/internal/model/map_model_test.go app/internal/model/resource_model.go app/internal/model/resource_model_test.go app/internal/logic/resource/create_resource_logic.go app/internal/logic/resource/create_resource_logic_test.go app/internal/logic/resource/submit_resource_logic.go app/internal/logic/resource/submit_resource_logic_test.go
git diff --check
git diff --stat
```

**步骤 2：后端全量测试**

运行：

```bash
cd backend && go test ./...
cd backend && node --test scripts/*.test.mjs
```

**步骤 3：小程序全量验证**

运行：

```bash
cd wxapp && npm test
cd wxapp && npm run validate:pages
cd wxapp && npm run validate:flows
cd wxapp && npm run build:mp-weixin
```

**步骤 4：验收标准**

- 公共商家详情 JSON 不含 `contact`，商家本人仍可编辑联系方式。
- 招聘入口和可发布类型均不可见，旧数据标签未损坏。
- 普通商家选中空闲点位后立即绑定；同商家第二点位和已占用点位均被并发安全地拒绝。
- 每个商家每月获得 3 条基础免费发布额度。
- 常规内容安全流程不产生人工审核任务；风险拦截、依赖重试与异常兜底仍可追踪。
- 后端、小程序测试与小程序构建全部通过。

## 后续阶段

本计划完成后单独实施“地图地址纠错/风险举报”闭环，包括纠错记录表、用户接口、自动阈值、商家确认、后台异常队列，以及第三方导航坐标与场内地图坐标的分离。该阶段不改变本计划的一商家一主档口规则。
