# API 实施清单

版本：v0.2
日期：2026-08-06
来源：

- `docs/superpowers/plans/2026-06-27-apparel-platform-current-mvp-todo.md`
- `backend/app/api/app.api`

## 约定

- API 契约源文件统一放在 `backend/app/api/*.api`。
- `backend/app/api/app.api` 是 go-zero API 单一入口，其他 `.api` 文件只按领域拆分。
- 生产请求固定经过 `.api -> goctl 1.7.5 generated routes/types -> Handler -> Logic -> Model/外部依赖`，运行时不按依赖是否存在裁剪路由。
- 小程序和后台共用 `/api/v1` 前缀。
- 管理后台接口统一使用 `/api/v1/admin` 前缀。
- 运行时实现必须保持 `resources` 统一供需信息模型，不能为库存、工厂、招聘、出租等类型拆独立业务系统。
- 前端可见错误必须中文、明确、可操作；后端日志记录内部原因，接口不返回 SQL、堆栈、表名、token 或敏感原始字段。

## 生成与维护门禁

- API 生成只使用根目录 `make generate-api`。生成脚本校验 goctl 必须为 1.7.5，并显式读取仓库内 `backend/app/goctl/` 模板；不得直接用开发机任意版本的全局 goctl、用户模板或 `GOCTL_HOME` 生成项目文件。
- 生成结果使用 `make check-api-generated` 检查；该门禁校验 goctl 版本、契约与 generated routes/types 一致、路由无重复，并拒绝缺失 Handler 或残留 `WPLINK_API_HANDLER_STUB`。
- 生产 Server 的运行时路由多重集由 Go 测试 `TestGeneratedRouteParity` 校验，该测试随根目录 `make check` 执行；单独运行 `make check-api-generated` 不替代运行时路由装配测试。
- 新增或修改 API 时，先修改对应领域 `.api`；若新增领域文件，再加入 `backend/app/api/app.api` import。随后运行 `make generate-api`，实现 Handler/Logic 和所需 Model/外部依赖，补齐身份与权限测试、成功和错误行为测试。
- 生成的 `WPLINK_API_HANDLER_STUB` 仅用于提示缺失实现，可以在开发过程中短暂存在，提交前必须实现对应 Handler 并清除。
- 提交前至少执行 `make check-api-generated`、`cd backend && node --test scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs`、相关 Go 行为测试；最终执行根目录 `make check`。
- `server.NewGoZeroServer` 通过 `handler.RegisterHandlers` 注册全部契约路由。额外路由只有 `/healthz`、`/readyz`；`/admin` 由嵌入式静态 NotFound 回退承接。未声明 API 返回 404，错误 HTTP method 返回 405。

## 账号与权限

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/auth/wechat-login` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | 登录/启动流程 | 已接 handler，测试通过 |
| `POST /api/v1/auth/sms-code` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | 后续手机号绑定预留 | 已接 handler，首发不验收 |
| `GET /api/v1/me` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | `wxapp/pages/my/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/me/phone` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | 后续手机号绑定预留 | 已接 handler，首发不验收 |
| `POST /api/v1/admin/auth/login` | `backend/app/api/admin.api` | `backend/app/internal/logic/adminauth/login_service.go` | `admin-web/src/views/LoginView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/uploads/token` | `backend/app/api/upload.api` | `backend/app/internal/logic/upload/upload_token_logic.go` | Banner 等运营图片上传前置 | 发布与商家资料图片上传前置 | 已接 handler，测试通过 |

## 城市站与配置

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/city-stations` | `backend/app/api/city.api` | `backend/app/internal/logic/city/list_city_stations_logic.go` | 可用于全局筛选 | 首页/发布/搜索 | 已接 handler，测试通过 |
| `GET /api/v1/city-stations/:cityCode/resource-types` | `backend/app/api/city.api` | `backend/app/internal/logic/city/list_resource_types_logic.go` | 供需类型配置页 | 发布/搜索筛选 | 已接 handler，测试通过 |

## 商家

普通用户登录时由后端幂等初始化唯一商家，资料页只更新该商家，不提供再次创建商家的接口。

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/merchants/:merchantId` | `backend/app/api/merchant.api` | `backend/app/internal/logic/merchant/get_merchant_logic.go` | 商家详情抽屉 | `wxapp/pages/merchant/detail.vue`，展示商家资料和发布记录 | 已接 handler，测试通过 |
| `POST /api/v1/merchants/:merchantId` | `backend/app/api/merchant.api` | `backend/app/internal/logic/merchant/update_merchant_logic.go` | 商家编辑 | `wxapp/pages/merchant/profile.vue` 商家资料编辑 | 已接 handler，测试通过 |

## 供需信息

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/create_resource_logic.go` | 代发供需信息 | `wxapp/pages/publish/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/drafts` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/create_resource_logic.go` | 可用于代发草稿 | `wxapp/pages/publish/index.vue` 保存草稿 | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/submit` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/submit_resource_logic.go` | 可用于代发草稿提交 | `wxapp/pages/my-resources/index.vue` 草稿提交审核 | 已接 handler，测试通过 |
| `GET /api/v1/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/list_resources_logic.go` | 可用于供需信息检索 | `wxapp/pages/search/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/resource-search` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/search_resources_logic.go` | 可用于供需信息检索 | `wxapp/pages/search/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 不适用 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/resources/:resourceId` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/get_resource_logic.go` | 供需信息详情抽屉 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/detail-view` | `backend/app/api/resource.api` | `backend/app/internal/logic/metrics/record_detail_view_logic.go` | 效果统计 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/refresh` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 供需信息运营操作 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/deal-feedback` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 成交标记 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/take-down` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 供需信息运营操作 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/repost-similar` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 不适用 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/contact-events` | `backend/app/api/resource.api` | `backend/app/internal/logic/metrics/record_contact_logic.go` | 效果统计 | 供需信息详情联系按钮 | 已接 handler，测试通过 |

## 需求方向供需信息

说明：需求方向供需信息统一使用供需信息发布、供需信息列表、我的发布和后台供需信息审核接口，不再保留独立 API 文件、独立小程序页面或独立后台专用处理页面。

| 场景 | 统一接口 | API 文件 | 前端入口 | 状态 |
|---|---|---|---|---|
| 发布找货、找厂、找服务等需求方向供需信息 | `POST /api/v1/resources` | `backend/app/api/resource.api` | `wxapp/pages/publish/index.vue` | 已接 handler，测试通过 |
| 查看我的需求方向供需信息 | `GET /api/v1/me/resources?direction=demand` | `backend/app/api/resource.api` | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| 后台审核需求方向供需信息 | `GET /api/v1/admin/resources/pending`、`POST /api/v1/admin/resources/:resourceId/review` | `backend/app/api/admin.api` | `admin-web/src/views/ResourceReviewView.vue` | 已接 handler，测试通过 |

## 发现与运营位

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/home/operation-config` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/home/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/topics/:topicId/resources` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/topic/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/webview/validate` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/webview/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/admin/banner-topics` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/banner-topics` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/banner-topics/:configId` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |

## 权益与置顶

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/merchants/:merchantId/entitlements` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | `admin-web/src/views/EntitlementView.vue` | 我的页权益余量 | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId/top-vouchers` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | 权益详情 | 我的页券余量、我的发布置顶操作 | 已接 handler，测试通过 |
| `POST /api/v1/top-vouchers/:voucherId/redeem` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | 置顶核销记录 | 我的发布置顶操作 | 已接 handler，测试通过 |

## 发布效果

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/resources/:resourceId/metrics` | `backend/app/api/metrics.api` | `backend/app/internal/logic/metrics/get_resource_metrics_logic.go` | 供需信息效果详情 | 我的发布供需信息卡片/效果页 | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId/metrics/summary` | `backend/app/api/metrics.api` | `backend/app/internal/logic/metrics/get_merchant_metrics_logic.go` | 数据概览/商家详情 | 我的页/商家后台入口 | 已接 handler，测试通过 |

## 消息

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/messages` | `backend/app/api/message.api` | `backend/app/internal/logic/message/message_logic.go` | 消息发送记录 | `wxapp/pages/messages/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/messages/:messageId/read` | `backend/app/api/message.api` | `backend/app/internal/logic/message/message_logic.go` | 不适用 | `wxapp/pages/messages/index.vue` | 已接 handler，测试通过 |

## 收藏、关注和保存搜索

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/me/favorite-resources` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/favorites/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/favorite-resources/:resourceId` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/me/favorite-resources/:resourceId` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/followed-merchants` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/favorites/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/followed-merchants/:merchantId` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/merchant/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/me/followed-merchants/:merchantId` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/merchant/detail.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/saved-searches` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/search/index.vue`、`wxapp/pages/favorites/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/me/saved-searches` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/search/index.vue` | 已接 handler，测试通过 |
| `DELETE /api/v1/me/saved-searches/:savedSearchId` | `backend/app/api/favorite.api` | `backend/app/internal/logic/favorite/favorite_logic.go` | 不适用 | `wxapp/pages/favorites/index.vue` | 已接 handler，测试通过 |

## 管理后台

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/admin/dashboard/overview` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/dashboard_logic.go` | `admin-web/src/views/DashboardView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/resources/pending` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/list_pending_resources_logic.go` | `admin-web/src/views/ResourceReviewView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/resources/:resourceId/review` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/review_resource_logic.go` | `admin-web/src/views/ResourceReviewView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/merchants/:merchantId/entitlements` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/entitlement_admin_logic.go` | `admin-web/src/views/EntitlementView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/operation-logs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/operation_log_logic.go` | `admin-web/src/views/OperationLogView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/search-logs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/search_log_logic.go` | `admin-web/src/views/SearchLogView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/tasks/resource-lifecycle/run` | `backend/app/api/admin.api` | `backend/app/internal/task/resource_lifecycle_task.go` | 运维/运营手动触发 | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/merchants` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/merchant_admin_logic.go` | `admin-web/src/views/MerchantView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/resource-type-configs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/resource_type_config_logic.go` | `admin-web/src/views/ResourceTypeConfigView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/resource-type-configs/:configId` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/resource_type_config_logic.go` | `admin-web/src/views/ResourceTypeConfigView.vue` | 不适用 | 已接 handler，测试通过 |

## 后续计划内接口

以下能力在当前 MVP TODO 中已规划，但产品 API 契约文档还没有完整展开。进入对应阶段前需要补充 `.api` 契约：

- 直接接入具体短信厂商 SDK（当前已支持通用 HTTP 验证码服务接入）
