# API 实施清单

版本：v0.1  
日期：2026-06-28
来源：

- `docs/product/api-contract-design.md`
- `docs/superpowers/plans/2026-06-27-apparel-platform-current-mvp-todo.md`
- `backend/app/api/app.api`

## 约定

- API 契约源文件统一放在 `backend/app/api/*.api`。
- `backend/app/api/app.api` 是 go-zero API 单一入口，其他 `.api` 文件只按领域拆分。
- 小程序和后台共用 `/api/v1` 前缀。
- 管理后台接口统一使用 `/api/v1/admin` 前缀。
- 运行时实现必须保持 `resources` 统一资源模型，不能为库存、工厂、招聘、出租等类型拆独立业务系统。
- 前端可见错误必须中文、明确、可操作；后端日志记录内部原因，接口不返回 SQL、堆栈、表名、token 或敏感原始字段。

## 账号与权限

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/auth/wechat-login` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | 登录/启动流程 | 已接 handler，测试通过 |
| `POST /api/v1/auth/sms-code` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | `wxapp/pages/my/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | `wxapp/pages/my/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/me/phone` | `backend/app/api/auth.api` | `backend/app/internal/logic/auth/auth_logic.go` | 不适用 | 绑定手机号流程 | 已接 handler，测试通过 |
| `POST /api/v1/admin/auth/login` | `backend/app/api/admin.api` | `backend/app/internal/logic/adminauth/login_service.go` | `admin-web/src/views/LoginView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/uploads/token` | `backend/app/api/upload.api` | `backend/app/internal/logic/upload/upload_token_logic.go` | Banner/认证资料 URL 上传前置 | 发布/认证图片上传前置 | 已接 handler，测试通过 |

## 城市站与配置

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/city-stations` | `backend/app/api/city.api` | `backend/app/internal/logic/city/list_city_stations_logic.go` | 可用于全局筛选 | 首页/发布/搜索 | 已接 handler，测试通过 |
| `GET /api/v1/city-stations/:cityCode/resource-types` | `backend/app/api/city.api` | `backend/app/internal/logic/city/list_resource_types_logic.go` | 资源类型配置页 | 发布/搜索筛选 | 已接 handler，测试通过 |

## 商家

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/merchants` | `backend/app/api/merchant.api` | `backend/app/internal/logic/merchant/create_merchant_logic.go` | `admin-web/src/views/MerchantView.vue` | 商家入驻/我的 | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId` | `backend/app/api/merchant.api` | `backend/app/internal/logic/merchant/get_merchant_logic.go` | 商家详情抽屉 | `wxapp/pages/merchant/detail.vue` | 已接 handler，测试通过 |
| `PATCH /api/v1/merchants/:merchantId` | `backend/app/api/merchant.api` | `backend/app/internal/logic/merchant/update_merchant_logic.go` | 商家编辑 | 商家资料编辑 | 已接 handler，测试通过 |

## 资源

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/create_resource_logic.go` | 代发资源 | `wxapp/pages/publish/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/list_resources_logic.go` | 可用于资源检索 | `wxapp/pages/search/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/resource-search` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/search_resources_logic.go` | 可用于资源检索 | `wxapp/pages/search/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/resources` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 不适用 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/resources/:resourceId` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/get_resource_logic.go` | 资源详情抽屉 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/detail-view` | `backend/app/api/resource.api` | `backend/app/internal/logic/metrics/record_detail_view_logic.go` | 效果统计 | `wxapp/pages/resource/detail.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/refresh` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 资源运营操作 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/deal-feedback` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 成交标记 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/take-down` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 资源运营操作 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/repost-similar` | `backend/app/api/resource.api` | `backend/app/internal/logic/resource/my_resource_logic.go` | 不适用 | `wxapp/pages/my-resources/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/resources/:resourceId/contact-events` | `backend/app/api/resource.api` | `backend/app/internal/logic/metrics/record_contact_logic.go` | 效果统计 | 资源详情联系按钮 | 已接 handler，测试通过 |

## 采购需求

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/purchase-demands` | `backend/app/api/demand.api` | `backend/app/internal/logic/demand/create_demand_logic.go` | 需求线索池 | `wxapp/pages/demand/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/me/purchase-demands` | `backend/app/api/demand.api` | `backend/app/internal/logic/demand/list_my_demands_logic.go` | 不适用 | `wxapp/pages/my-demands/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/admin/purchase-demands` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/demand_admin_logic.go` | `admin-web/src/views/DemandView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/purchase-demands/:demandId` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/demand_admin_logic.go` | `admin-web/src/views/DemandView.vue` | 不适用 | 已接 handler，测试通过 |
| `PATCH /api/v1/admin/purchase-demands/:demandId/status` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/demand_admin_logic.go` | `admin-web/src/views/DemandView.vue` | 不适用 | 已接 handler，测试通过 |

## 发现与运营位

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/home/banners` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/home/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/topics/:topicId/resources` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/topic/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/webview/validate` | `backend/app/api/discovery.api` | `backend/app/internal/logic/discovery/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | `wxapp/pages/webview/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/admin/banner-topics` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/banner-topics` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |
| `PATCH /api/v1/admin/banner-topics/:configId` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/banner_topic_logic.go` | `admin-web/src/views/BannerTopicView.vue` | 不适用 | 已接 handler，测试通过 |

## 认证

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `POST /api/v1/merchants/:merchantId/verifications` | `backend/app/api/verification.api` | `backend/app/internal/logic/verification/submit_verification_logic.go` | 认证审核列表 | `wxapp/pages/verification/index.vue` | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId/verifications/latest` | `backend/app/api/verification.api` | `backend/app/internal/logic/verification/submit_verification_logic.go` | 商家详情 | 认证入口/商家主页 | 已接 handler，测试通过 |

## 权益与置顶

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/merchants/:merchantId/entitlements` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | `admin-web/src/views/EntitlementView.vue` | 我的/我的发布 | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId/top-vouchers` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | 权益详情 | 我的发布置顶操作 | 已接 handler，测试通过 |
| `POST /api/v1/top-vouchers/:voucherId/redeem` | `backend/app/api/entitlement.api` | `backend/app/internal/logic/entitlement/entitlement_logic.go` | 置顶核销记录 | 我的发布置顶操作 | 已接 handler，测试通过 |

## 发布效果

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/resources/:resourceId/metrics` | `backend/app/api/metrics.api` | `backend/app/internal/logic/metrics/get_resource_metrics_logic.go` | 资源效果详情 | 我的发布资源卡片/效果页 | 已接 handler，测试通过 |
| `GET /api/v1/merchants/:merchantId/metrics/summary` | `backend/app/api/metrics.api` | `backend/app/internal/logic/metrics/get_merchant_metrics_logic.go` | 数据概览/商家详情 | 我的页/商家后台入口 | 已接 handler，测试通过 |

## 消息

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/messages` | `backend/app/api/message.api` | `backend/app/internal/logic/message/message_logic.go` | 消息发送记录 | `wxapp/pages/messages/index.vue` | 已接 handler，测试通过 |
| `POST /api/v1/messages/:messageId/read` | `backend/app/api/message.api` | `backend/app/internal/logic/message/message_logic.go` | 不适用 | `wxapp/pages/messages/index.vue` | 已接 handler，测试通过 |

## 管理后台

| 接口 | API 文件 | 后端 Logic | 后台页面 | 小程序页面 | 状态 |
|---|---|---|---|---|---|
| `GET /api/v1/admin/dashboard/overview` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/dashboard_logic.go` | `admin-web/src/views/DashboardView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/resources/pending` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/list_pending_resources_logic.go` | `admin-web/src/views/ResourceReviewView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/resources/:resourceId/review` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/review_resource_logic.go` | `admin-web/src/views/ResourceReviewView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/verifications/pending` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/verification_admin_logic.go` | `admin-web/src/views/VerificationView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/verifications/:verificationId/review` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/verification_admin_logic.go` | `admin-web/src/views/VerificationView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/merchants/:merchantId/entitlements` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/entitlement_admin_logic.go` | `admin-web/src/views/EntitlementView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/match-cases` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/match_case_logic.go` | `admin-web/src/views/MatchCaseView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/match-cases` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/match_case_logic.go` | `admin-web/src/views/MatchCaseView.vue` | 不适用 | 已接 handler，测试通过 |
| `PATCH /api/v1/admin/match-cases/:matchCaseId/status` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/match_case_logic.go` | `admin-web/src/views/MatchCaseView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/match-cases/:matchCaseId/resources` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/match_case_logic.go` | `admin-web/src/views/MatchCaseView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/match-cases/:matchCaseId/participants` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/match_case_logic.go` | `admin-web/src/views/MatchCaseView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/operation-logs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/operation_log_logic.go` | `admin-web/src/views/OperationLogView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/search-logs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/search_log_logic.go` | `admin-web/src/views/SearchLogView.vue` | 不适用 | 已接 handler，测试通过 |
| `POST /api/v1/admin/tasks/resource-lifecycle/run` | `backend/app/api/admin.api` | `backend/app/internal/task/resource_lifecycle_task.go` | 运维/运营手动触发 | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/merchants` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/merchant_admin_logic.go` | `admin-web/src/views/MerchantView.vue` | 不适用 | 已接 handler，测试通过 |
| `GET /api/v1/admin/resource-type-configs` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/resource_type_config_logic.go` | `admin-web/src/views/ResourceTypeConfigView.vue` | 不适用 | 已接 handler，测试通过 |
| `PATCH /api/v1/admin/resource-type-configs/:configId` | `backend/app/api/admin.api` | `backend/app/internal/logic/admin/resource_type_config_logic.go` | `admin-web/src/views/ResourceTypeConfigView.vue` | 不适用 | 已接 handler，测试通过 |

## 后续计划内接口

以下能力在当前 MVP TODO 中已规划，但产品 API 契约文档还没有完整展开。进入对应阶段前需要补充 `.api` 契约：

- 直接接入具体短信厂商 SDK（当前已支持通用 HTTP 验证码服务接入）
