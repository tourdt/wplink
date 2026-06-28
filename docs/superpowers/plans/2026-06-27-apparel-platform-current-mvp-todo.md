# 服装产业资源撮合平台当前 MVP TODO 计划

> **给 agentic workers：** 必须使用子技能：用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务逐项实施。本计划使用复选框（`- [ ]`）跟踪进度。

**目标：** 基于当前产品文档和原型，完成首个 MVP 垂直闭环：商家发布资源、运营审核、买家搜索/查看详情/联系、商家查看发布效果、运营进行人工撮合。

**架构：** 以 `docs/product/technical-architecture.md` 当前确认方向为准：`backend/` 使用 Go + go-zero API，数据库使用 PostgreSQL；`admin-web/` 使用 Vue 3 + Vite + Element Plus；新增 `wxapp/` uni-app 微信小程序。所有库存、货源、工厂、订单、招聘、出租、服务统一落到 `resources` 资源模型，不为每类业务单独做一套系统。

**技术栈：** Go、go-zero API 契约、PostgreSQL JSONB、Vue 3、Vite、Element Plus、Pinia、uni-app、微信小程序、现有静态高保真原型 `prototypes/wxapp-hifi/`。

---

## 规划依据

产品文档：

- `docs/product/apparel-industry-platform-prd.md`
- `docs/product/domain-model-ddd.md`
- `docs/product/database-er-design.md`
- `docs/product/api-contract-design.md`
- `docs/product/technical-architecture.md`

原型：

- `prototypes/wxapp-hifi/README.md`
- `prototypes/wxapp-hifi/index.html`

当前实现基线：

- `backend/app/api/app.api` 目前只引入了 `admin.api`。
- `backend/app/api/admin.api` 目前只定义了后台登录接口。
- `backend/migrations/000001_admin_auth.up.sql` 已覆盖用户、角色、后台登录凭证、运营人员资料和操作日志。
- `backend/app/internal/logic/adminauth/` 已有后台登录领域逻辑和测试。
- `admin-web/` 已有登录、数据概览、资源审核、商家、需求、认证、权益和操作日志路由，但多数页面仍是静态 mock 数据。
- `wxapp/` 还不存在；小程序流程目前只存在于静态高保真原型中。

旧计划说明：

- `docs/superpowers/plans/2026-06-26-apparel-platform-mvp.md` 使用 TypeScript/Fastify/Taro 方案，已经不符合当前确认架构。后续只作为历史上下文，不作为实施依据。

## MVP 边界

必须交付：

- 织里城市站种子数据。
- `inventory`、`goods`、`factory`、`order`、`job`、`rental`、`service` 七类统一资源类型配置。
- 商家主页基础创建、编辑和公开详情。
- 资源发布、提交审核、审核通过、驳回、发布、刷新、标记成交、下架、过期、列表、搜索和详情。
- 资源详情联系动作：拨打电话、复制微信、进入商家主页、分享。
- 详情浏览和联系行为统计。
- 基础认证流程和公开信用标签。
- 商家“我的发布”管理：状态、效果数据、刷新、使用置顶券、标记成交、下架、再发类似。
- 搜索无结果后的采购需求提交，以及运营侧需求列表。
- 运营人工撮合记录。
- 审核、生命周期、认证、撮合和效果反馈基础消息。
- 首页 Banner、专题和 web-view 配置能力，覆盖原型验证流程。
- 后台数据概览、审核、商家、需求、认证、权益和操作日志页面接入真实 API。
- 一个按原型核心流程落地的 uni-app 微信小程序。

MVP 不做：

- 担保支付、下单交易、佣金结算、合同和履约。
- 复杂即时通讯。
- 复杂信用分。
- 完整推荐引擎。
- 营销群发。
- 超出字段和简单筛选之外的复杂多城市运营隔离。
- V1.1 才需要的收藏、关注商家、保存搜索、新货提醒等能力；产品文档要求保留入口的地方可以先保留 UI 入口。

## 实施 TODO

### 阶段 0：锁定当前接口契约

**文件：**

- 修改：`backend/app/api/app.api`
- 修改：`backend/app/api/admin.api`
- 新建：`backend/app/api/auth.api`
- 新建：`backend/app/api/city.api`
- 新建：`backend/app/api/merchant.api`
- 新建：`backend/app/api/resource.api`
- 新建：`backend/app/api/demand.api`
- 新建：`backend/app/api/verification.api`
- 新建：`backend/app/api/entitlement.api`
- 新建：`backend/app/api/message.api`
- 新建：`backend/app/api/metrics.api`
- 新建：`docs/product/api-implementation-checklist.md`

- [x] 在 `backend/app/api/app.api` 中引入所有 API 文件，让 `app.api` 成为接口单一入口。
- [x] 保持现有 `/api/v1/admin/auth/login` 契约不变。
- [x] 将产品 API 契约翻译为 go-zero `type` 定义和路由，并保持 `docs/product/api-contract-design.md` 中的 JSON 字段名。
- [x] 使用产品文档中的稳定状态和类型枚举：资源状态 `draft`、`pending`、`published`、`rejected`、`expired`、`dealt`、`taken_down`、`archived`；资源类型 `inventory`、`goods`、`factory`、`order`、`job`、`rental`、`service`。
- [x] 新增 API 实施清单，映射每个产品接口到后端 logic 包、后台页面和小程序页面。
- [x] 在 `backend/` 运行 `go test ./...`。

验收标准：

- 所有 MVP 路由在实现前已经被 API 文件描述清楚。
- 不新增独立的库存、工厂、招聘或出租业务系统路由。
- 后端现有测试通过。

### 阶段 1：数据库基础

**文件：**

- 新建：`backend/migrations/000002_core_domain.up.sql`
- 新建：`backend/migrations/000002_core_domain.down.sql`
- 新建：`backend/migrations/000003_seed_zhili.up.sql`
- 新建：`backend/migrations/000003_seed_zhili.down.sql`

- [x] 新增 `city_stations`、`merchants`、`merchant_admin_bindings`、`resource_type_configs`、`resources`、`resource_review_records`、`verifications`、`credit_records`、`purchase_demands`、`search_logs`、`match_cases`、`match_case_resources`、`match_case_participants`、`merchant_entitlements`、`top_vouchers`、`resource_contact_events`、`resource_metrics_daily` 和 `messages`。
- [x] 按 `docs/product/database-er-design.md` 保持稳定通用字段关系化，业务变化字段使用 JSONB。
- [x] 添加资源按城市/类型/状态列表、商家资源管理、attributes GIN、搜索日志、联系事件和日指标相关索引。
- [x] 将织里城市站设为 active 种子数据。
- [x] 种子写入七类资源类型配置，包含默认有效期、必填字段、筛选字段和展示模板。
- [x] 如本地环境需要后台登录夹具，种子写入一个运营账号；凭证必须标明仅本地使用，不能用于生产。
- [x] 在本地 PostgreSQL 数据库验证 migration up/down。

验收标准：

- 干净 PostgreSQL 数据库可以完整迁移 up/down。
- 种子数据足够支撑首页、搜索筛选和发布类型选择，无需手动改库。

### 阶段 2：后端公共基础设施

**文件：**

- 新建：`backend/common/response/response.go`
- 新建：`backend/common/errx/errors.go`
- 新建：`backend/app/internal/config/config.go`
- 新建：`backend/app/internal/svc/service_context.go`
- 新建：`backend/app/internal/model/db.go`
- 新建：`backend/app/internal/model/json.go`
- 新建：`backend/app/internal/session/admin_token.go`
- 新建：`backend/app/internal/permission/admin.go`
- 修改：`backend/app/internal/logic/adminauth/login_service.go`

- [x] 新增统一 JSON 响应封装，兼容当前后台登录接口和后续小程序/后台 API。
- [x] 新增中文友好的公开错误信息和内部错误包装，接口不暴露 SQL、表名、堆栈和密钥。
- [x] 新增 PostgreSQL 数据库连接。
- [x] 新增多表写入事务辅助方法。
- [x] 新增 JSONB 扫描/序列化辅助方法，服务 attributes、配置、标签和快照字段。
- [x] 将后台 token 签发接入现有 `adminauth.LoginService`。
- [x] 新增 `platform_operator` 和 `super_admin` 权限判断辅助方法。
- [x] 运行登录和错误映射相关单元测试。

验收标准：

- 当前后台登录测试继续通过。
- 后端只有一种公开错误响应结构。
- 返回给前端的业务错误中文、明确、可操作。

### 阶段 3：城市站和资源类型配置

**文件：**

- 新建：`backend/app/internal/model/city_station_model.go`
- 新建：`backend/app/internal/model/resource_type_config_model.go`
- 新建：`backend/app/internal/logic/city/list_city_stations_logic.go`
- 新建：`backend/app/internal/logic/city/list_resource_types_logic.go`
- 新建：`backend/app/internal/logic/admin/resource_type_config_logic.go`
- 修改：`admin-web/src/router/index.js`
- 新建：`admin-web/src/api/city.js`
- 新建：`admin-web/src/views/ResourceTypeConfigView.vue`
- 新建：`wxapp/api/city.js`

- [x] 实现公开城市站列表 API。
- [x] 实现指定城市站资源类型列表 API。
- [x] 实现后台资源类型配置列表/更新 API，支持字段模板、必填字段、筛选字段、展示模板、审核规则、有效期和状态。
- [x] 新增后台资源类型配置页，包含 JSON 编辑区域和清晰校验提示。
- [x] 新增小程序城市/资源类型 API 封装。
- [x] 验证首页和发布筛选可以从种子配置加载资源类型。

验收标准：

- 后台可以查看和调整资源类型配置。
- 小程序发布流程由配置驱动，不按业务类型硬编码表单。

### 阶段 4：商家主页

**文件：**

- 新建：`backend/app/internal/model/merchant_model.go`
- 新建：`backend/app/internal/model/merchant_admin_binding_model.go`
- 新建：`backend/app/internal/logic/merchant/create_merchant_logic.go`
- 新建：`backend/app/internal/logic/merchant/get_merchant_logic.go`
- 新建：`backend/app/internal/logic/merchant/update_merchant_logic.go`
- 新建：`backend/app/internal/logic/admin/merchant_admin_logic.go`
- 新建：`admin-web/src/api/merchant.js`
- 修改：`admin-web/src/views/MerchantView.vue`
- 新建：`wxapp/api/merchant.js`
- 新建：`wxapp/pages/merchant/detail.vue`

- [x] 实现商家创建、编辑、详情和列表 API。
- [x] 校验商家必填字段：城市、名称、商家类型、主营品类、联系人、联系电话。
- [x] 商家详情包含认证状态、公开信用标签、当前资源、历史资源和最近活跃时间。
- [x] 后台商家页面接入真实列表、详情和编辑 API。
- [x] 实现小程序商家主页，覆盖原型中的认证、信用标签、简介、图片、当前资源、历史资源和联系入口。
- [x] 等资源发布完成后，补充“封禁商家不能发布资源”的测试。

验收标准：

- 买家可以从资源详情进入商家主页。
- 运营可以在审核资源或认证前查看商家信息。

### 阶段 5：资源发布和审核

**文件：**

- 新建：`backend/app/internal/model/resource_model.go`
- 新建：`backend/app/internal/model/resource_review_record_model.go`
- 新建：`backend/app/internal/logic/resource/create_resource_logic.go`
- 新建：`backend/app/internal/logic/resource/submit_resource_logic.go`
- 新建：`backend/app/internal/logic/resource/update_resource_logic.go`
- 新建：`backend/app/internal/logic/resource/list_resources_logic.go`
- 新建：`backend/app/internal/logic/resource/get_resource_logic.go`
- 新建：`backend/app/internal/logic/admin/review_resource_logic.go`
- 新建：`backend/app/internal/logic/admin/take_down_resource_logic.go`
- 新建：`admin-web/src/api/resource.js`
- 修改：`admin-web/src/views/ResourceReviewView.vue`
- 新建：`wxapp/api/resource.js`
- 新建：`wxapp/pages/publish/index.vue`
- 新建：`wxapp/pages/resource/detail.vue`

- [x] 实现草稿创建和提交审核 API。
- [x] 根据 `resource_type_configs.required_fields` 校验必填字段。
- [x] 将类型差异字段写入 `resources.attributes`。
- [x] 提交后状态设为 `pending`；如为运营代发，写入操作日志。
- [x] 实现后台待审核资源列表、通过、驳回并填写原因、下架并填写原因。
- [x] 审核通过时设置 `published_at`、`refreshed_at`、`expires_at`，状态设为 `published`，并写入审核记录。
- [x] 驳回时设置 `reject_reason`，状态设为 `rejected`，并写入审核记录。
- [x] 实现公开资源列表/详情 API，默认只返回已发布且未过期资源。
- [x] 实现小程序发布页和提交成功页，提交成功页可进入消息中心。
- [x] 实现小程序资源详情，包含核心字段、标签、商家入口、同类推荐和联系动作。
- [x] 添加必填校验、状态流转、驳回原因和“仅已发布资源可公开展示”的后端测试。

验收标准：

- 商家或运营可以发布资源、提交审核、在后台通过审核，并在前台搜索和详情页看到资源。
- 被驳回、待审核、已过期、已下架和已归档资源不会进入公开列表。

### 阶段 6：搜索、专题、Banner 和需求兜底

**文件：**

- 新建：`backend/app/internal/model/search_log_model.go`
- 新建：`backend/app/internal/model/banner_topic_model.go`
- 新建：`backend/app/internal/logic/resource/search_resources_logic.go`
- 新建：`backend/app/internal/logic/demand/create_demand_logic.go`
- 新建：`backend/app/internal/logic/demand/list_my_demands_logic.go`
- 新建：`backend/app/internal/logic/admin/banner_topic_logic.go`
- 新建：`backend/app/internal/logic/admin/search_log_logic.go`
- 新建：`admin-web/src/api/demand.js`
- 修改：`admin-web/src/views/DemandView.vue`
- 新建：`admin-web/src/views/BannerTopicView.vue`
- 新建：`wxapp/pages/home/index.vue`
- 新建：`wxapp/pages/search/index.vue`
- 新建：`wxapp/pages/topic/index.vue`
- 新建：`wxapp/pages/webview/index.vue`
- 新建：`wxapp/pages/demand/index.vue`

- [x] 实现关键词搜索，覆盖标题、描述、品类、商家名，以及 MVP 阶段可支持的相关 attributes 字段。
- [x] 支持城市、资源类型、品类、认证状态和最近刷新排序筛选。
- [x] 记录搜索日志，包含筛选条件和结果数。
- [x] 实现搜索无结果和需求入口场景下的采购需求提交。
- [x] 新增后台需求列表、详情和状态更新 API。
- [x] 新增 Banner/专题配置表和后台管理，支持标题、副标题、封面、城市、资源类型范围、跳转类型、跳转目标、标签、上下线时间、排序和状态。
- [x] 实现首页 Banner 列表 API，只返回当前城市、启用中、时间有效的 Banner。
- [x] 实现专题列表 API，只返回审核通过、已发布、未过期资源；如专题为空，返回提交需求入口信息。
- [x] 对配置的活动网页 URL 做 web-view 域名白名单校验。
- [x] 实现小程序首页、搜索、专题、web-view 和需求页，匹配原型流程。

验收标准：

- 用户能从首屏直接搜索资源。
- 搜索无结果可以自然转为提交采购需求。
- Banner 可以跳转到专题、资源详情、商家主页、需求入口、内部页面或允许的网页 URL。

### 阶段 7：联系事件和发布效果

**文件：**

- 新建：`backend/app/internal/model/resource_contact_event_model.go`
- 新建：`backend/app/internal/model/resource_metric_daily_model.go`
- 新建：`backend/app/internal/logic/metrics/record_detail_view_logic.go`
- 新建：`backend/app/internal/logic/metrics/record_contact_logic.go`
- 新建：`backend/app/internal/logic/metrics/get_resource_metrics_logic.go`
- 新建：`backend/app/internal/logic/metrics/get_merchant_metrics_logic.go`
- 新建：`admin-web/src/api/metrics.js`
- 修改：`admin-web/src/views/DashboardView.vue`
- 新建：`wxapp/api/metrics.js`

- [x] 记录资源详情浏览次数。
- [x] 记录联系动作：电话、微信、商家主页、分享。
- [x] 按需幂等增加每日资源指标。
- [x] 在商家“我的发布”资源卡片中展示指标。
- [x] 后台数据概览展示待审核资源、待认证、采购需求、联系次数。
- [x] 商家侧指标不暴露具体访问用户身份。
- [x] 添加每日指标 upsert 和不同联系动作计数测试。

验收标准：

- 商家可以看到每条资源的曝光、详情浏览和联系基础数据。
- 后台数据概览不再使用静态 mock 指标。

### 阶段 8：认证、信用标签和权益

**文件：**

- 新建：`backend/app/internal/model/verification_model.go`
- 新建：`backend/app/internal/model/credit_record_model.go`
- 新建：`backend/app/internal/model/merchant_entitlement_model.go`
- 新建：`backend/app/internal/model/top_voucher_model.go`
- 新建：`backend/app/internal/logic/verification/submit_verification_logic.go`
- 新建：`backend/app/internal/logic/admin/review_verification_logic.go`
- 新建：`backend/app/internal/logic/entitlement/list_entitlements_logic.go`
- 新建：`backend/app/internal/logic/entitlement/use_top_voucher_logic.go`
- 新建：`admin-web/src/api/verification.js`
- 新建：`admin-web/src/api/entitlement.js`
- 修改：`admin-web/src/views/VerificationView.vue`
- 修改：`admin-web/src/views/EntitlementView.vue`
- 新建：`wxapp/pages/verification/index.vue`

- [x] 实现工厂、档口、库存和服务商认证提交。
- [x] 实现后台认证通过、驳回和撤销。
- [x] 认证通过后更新商家或资源认证状态，并创建公开信用记录。
- [x] 认证通过后发放基础权益：发布额度、刷新额度和置顶券。
- [x] 实现商家权益列表。
- [x] 置顶券只能用于当前商家的已发布资源。
- [x] 确保置顶券不能让待审核、已驳回或已下架资源公开展示。
- [x] 后台认证和权益页面接入真实 API。
- [x] 小程序新增认证入口和认证状态展示。

验收标准：

- 已认证商家/资源标签都有认证和信用记录作为事实依据。
- 权益可以使用，但不能绕过审核。

### 阶段 9：我的发布管理

**文件：**

- 新建：`backend/app/internal/logic/resource/list_my_resources_logic.go`
- 新建：`backend/app/internal/logic/resource/refresh_resource_logic.go`
- 新建：`backend/app/internal/logic/resource/mark_dealt_logic.go`
- 新建：`backend/app/internal/logic/resource/take_down_own_resource_logic.go`
- 新建：`backend/app/internal/logic/resource/repost_similar_logic.go`
- 新建：`wxapp/pages/my/index.vue`
- 新建：`wxapp/pages/my-resources/index.vue`
- 修改：`wxapp/pages/publish/index.vue`

- [x] 实现商家资源管理列表，支持全部、待审核、已发布、即将过期、已过期、已成交、已下架筛选。
- [x] 每条资源展示标题、类型、状态、发布时间、过期时间、曝光、详情浏览、电话点击和微信复制。
- [x] 已发布资源可刷新、使用置顶券、编辑、标记成交或下架。
- [x] 待审核资源禁止刷新和置顶。
- [x] 已过期或已成交资源可以复制字段再发类似资源，新记录进入草稿。
- [x] 实现小程序“我的”和“我的发布”页面，匹配原型预期。
- [x] 添加刷新额度消耗、状态保护和再发草稿创建测试。

验收标准：

- 商家能在一个页面看到资源状态、效果数据和下一步动作，有明确复访理由。

### 阶段 10：消息和生命周期任务

**文件：**

- 新建：`backend/app/internal/model/message_model.go`
- 新建：`backend/app/internal/logic/message/list_messages_logic.go`
- 新建：`backend/app/internal/logic/message/read_message_logic.go`
- 新建：`backend/app/internal/task/resource_lifecycle_task.go`
- 新建：`wxapp/pages/messages/index.vue`

- [x] 为审核通过/驳回、资源即将过期、资源已过期、资源下架、认证结果、撮合进度和效果反馈创建消息。
- [x] 实现用户消息列表和已读 API。
- [x] 新增生命周期任务，将到期资源标记为过期，并在过期前发送提醒。
- [x] MVP 只做站内消息，不做营销群发。
- [x] 实现小程序消息中心标签页，匹配原型。
- [x] 添加过期状态流转和消息创建测试。

验收标准：

- 审核、生命周期、撮合状态变化可以通过消息触达用户或商家，不需要用户反复手动查找。

### 阶段 11：人工撮合和操作日志

**文件：**

- 新建：`backend/app/internal/model/match_case_model.go`
- 新建：`backend/app/internal/logic/admin/match_case_logic.go`
- 新建：`backend/app/internal/model/operation_log_model.go`
- 新建：`admin-web/src/api/match.js`
- 新建：`admin-web/src/views/MatchCaseView.vue`
- 修改：`admin-web/src/views/OperationLogView.vue`

- [x] 实现后台从采购需求创建撮合记录。
- [x] 允许运营添加候选资源和参与方。
- [x] 跟踪撮合状态：open、contacted、succeeded、failed、closed。
- [x] 成功或失败关闭撮合时必须填写结果说明。
- [x] 记录敏感操作：审核通过、驳回、下架、发放、撤销、撮合状态更新。
- [x] 操作日志页面接入真实查询 API。

验收标准：

- 运营可以在后台完成早期人工撮合流程，不依赖表格。
- 敏感后台操作可审计。

### 阶段 12：管理后台接入

**文件：**

- 修改：`admin-web/src/api/http.js`
- 修改：`admin-web/src/layouts/AdminLayout.vue`
- 修改：`admin-web/src/views/DashboardView.vue`
- 修改：`admin-web/src/views/ResourceReviewView.vue`
- 修改：`admin-web/src/views/MerchantView.vue`
- 修改：`admin-web/src/views/DemandView.vue`
- 修改：`admin-web/src/views/VerificationView.vue`
- 修改：`admin-web/src/views/EntitlementView.vue`
- 修改：`admin-web/src/views/OperationLogView.vue`

- [x] 将所有静态 mock 表格数据替换为 API 调用。
- [x] 为每个后台页面补齐 loading、空状态、错误和重试状态。
- [x] 所有破坏性或状态变更操作必须有确认；产品要求填写原因的地方必须提供原因输入。
- [x] 后台 UI 保持运营工具风格：筛选在前、表格在后、详情使用抽屉或弹窗。
- [x] 新增资源类型配置、Banner/专题配置、撮合记录路由。
- [x] 运行 `npm --prefix admin-web run build`。

验收标准：

- 后台可以基于真实后端数据处理 MVP 审核、撮合和认证闭环。

### 阶段 13：小程序脚手架和原型转实现

**文件：**

- 新建：`wxapp/package.json`
- 新建：`wxapp/pages.json`
- 新建：`wxapp/manifest.json`
- 新建：`wxapp/main.js`
- 新建：`wxapp/App.vue`
- 新建：`wxapp/api/request.js`
- 新建：`wxapp/common/constants.js`
- 新建：`wxapp/common/enums.js`
- 新建：`wxapp/store/session.js`
- 新建：`wxapp/uni.scss`
- 新建：`wxapp/components/ResourceCard.vue`
- 新建：`wxapp/components/MerchantBadge.vue`
- 新建：`wxapp/components/MetricStrip.vue`

- [x] 按架构文档的 `wxapp/` 结构搭建 uni-app 工程。
- [x] 将原型页面转为真实页面：首页、搜索、专题、web-view、资源详情、商家详情、发布、发布成功、需求、需求成功、消息、我的、我的发布。
- [x] 实现底部 Tab：首页、搜索、发布、消息、我的。
- [x] 使用真实 API 封装；本地假数据只能放在显式 mock 开关后面。
- [x] 保留原型验证目标：首屏搜索、买家联系路径、搜索无结果提交需求、商家发布路径、消息/效果路径、Banner/专题/web-view 路径。
- [x] 定义脚本后运行一次小程序构建命令。

验收标准：

- 静态原型不再是唯一可执行的小程序流程。
- 买家和商家的 MVP 核心路径可以连接本地 API 测试。

### 阶段 14：端到端演示数据和验收

**文件：**

- 新建：`backend/scripts/seed_demo_data.sql`
- 新建：`docs/product/mvp-acceptance-checklist.md`
- 如存在仓库 README 则修改：`README.md`
- 如不存在 README 则新建：`docs/product/local-dev-runbook.md`

- [x] 种子写入演示商家：认证工厂、认证库存商、服务商和采购商。
- [x] 种子写入七类已发布资源：库存、货源、工厂、订单、招聘、出租和服务。
- [x] 种子写入一条待审核资源、一条已驳回资源、一条即将过期资源和一条已过期资源。
- [x] 种子写入一条采购需求和一条 open 状态撮合记录。
- [x] 编写本地运行手册，覆盖 PostgreSQL migration、后端服务、后台前端和小程序启动。
- [x] 编写 MVP 验收清单，覆盖发布-审核-搜索-联系-指标、搜索无结果-需求、认证-权益、我的发布、消息、Banner-专题-web-view 和人工撮合流程。
- [x] 运行后端单元测试。
- [x] 运行后台构建。
- [ ] 按原型验收目标手动验证小程序流程。

验收标准：

- 评审者无需了解代码库，也能启动本地栈并完成核心 MVP 演示。

## 推荐执行顺序

1. 阶段 0 到阶段 2：接口契约、数据库和后端基础设施。
2. 阶段 3 到阶段 5：城市配置、商家和资源审核垂直切片。
3. 阶段 6 到阶段 7：买家发现、需求兜底和效果数据。
4. 阶段 8 到阶段 11：认证、权益、资源管理、消息、撮合和日志。
5. 阶段 12 到阶段 14：后台真实数据接入、uni-app 转实现、演示数据和验收验证。

## 第一个开发切片

先做最小可验证闭环：

- [x] 添加城市、商家、资源类型配置、资源、审核记录、指标和消息的核心 migration。
- [x] 添加城市、资源类型、商家和资源 API。
- [x] 将 `admin-web/src/views/ResourceReviewView.vue` 接入待审核资源 API。
- [x] 构建最小 `wxapp/` 页面：首页、搜索、资源详情、发布、我的发布。
- [x] 验证：发布资源 -> 待审核 -> 后台通过 -> 出现在搜索结果 -> 进入详情 -> 触发联系动作 -> 我的发布看到指标变化。

该切片先验证产品核心承诺，再扩展认证、Banner 专题、需求撮合和权益。
