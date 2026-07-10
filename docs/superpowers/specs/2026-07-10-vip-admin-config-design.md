# VIP 管理后台运营配置设计

## 背景

当前项目已经具备 VIP 商业化的基础能力：

- 小程序通过 `/api/v1/vip/plans` 读取 VIP 套餐。
- 小程序通过 `/api/v1/vip/quota-packs` 读取次数包。
- 购买时会将套餐权益写入 `vip_orders.benefits_snapshot`，历史订单按购买时快照执行。
- 支付成功后会按快照发放 `merchant_entitlements`。
- 管理后台已有“权益发放”，但它用于给单个商家补发额度，不适合维护可售 VIP 商品和优惠。

本次目标是在管理后台新增完整 VIP 运营配置能力，让运营人员能维护 VIP 套餐、次数包和优惠活动，并让现有小程序 VIP 页面读取到最新可售配置。

## 目标

1. 管理后台新增“VIP 配置”菜单和页面。
2. 支持维护 VIP 套餐的名称、周期、价格、状态、排序和权益数量。
3. 支持维护次数包的名称、描述、价格、优惠价、状态、排序和权益数量。
4. 支持维护 VIP 优惠活动，包括适用套餐、优惠类型、优惠价、生效时间、结束时间、名额和状态。
5. 保存套餐权益时生成新的 `vip_plan_versions`，不覆盖历史订单快照。
6. 配置变更写入操作日志，便于排查谁在何时调整了价格或权益。
7. 后端返回中文友好错误，内部日志保留足够诊断上下文。

## 非目标

1. 不实现审批流、草稿发布、回滚版本管理。
2. 不改造微信支付流程。
3. 不重构小程序 VIP 页面交互，只保证它继续读取最新可售配置。
4. 不改变 `vip_orders.benefits_snapshot` 的历史订单语义。
5. 不删除或合并现有“权益发放”页面。
6. 不实现多等级会员，例如黄金 VIP、钻石 VIP。

## 方案选择

采用“新增后台专用 API + 新增 `VIPConfigView` + 复用现有 VIP 表”的方案。

理由：

- 现有 `vip_plans`、`vip_plan_versions`、`vip_promotions`、`vip_quota_packs` 已经覆盖核心数据结构，不需要新增表。
- 小程序公开接口已经读取这些表，后台保存后可自然影响新购买和新展示。
- 套餐权益通过版本表承载，能保证历史订单不受后续调价和权益调整影响。
- 独立后台模块比混入“权益发放”更清晰，避免把“可售商品配置”和“单商家补发额度”混在一起。

暂不采用完整审批流。该能力更适合后续多人运营、强审计或线上高风险配置阶段，本阶段先用管理员登录、字段校验、操作日志和停用能力控制风险。

## 后台页面设计

新增后台菜单：

- 路由：`/vip-configs`
- 菜单名称：`VIP 配置`
- 页面文件：`admin-web/src/views/VIPConfigView.vue`
- API 文件：`admin-web/src/api/vipConfig.js`

页面使用 `el-tabs` 分为三类：

1. `VIP 套餐`
2. `次数包`
3. `优惠活动`

### VIP 套餐

列表字段：

- 套餐名称
- 套餐编码
- 周期
- 标准价
- 当前可售优惠价
- 权益摘要
- 状态
- 排序
- 更新时间

编辑字段：

- `code`：套餐编码，新增时填写，编辑后不允许修改。
- `name`：套餐名称。
- `durationMonths`：套餐周期，单位月，必须大于 0。
- `standardPriceCent`：标准价，单位分，必须大于等于 0。
- `status`：`active` 或 `inactive`。
- `displayOrder`：排序值。
- `benefits.publishPolicy`：默认 `quota`。
- `benefits.publishQuota`：发布额度，必须大于等于 0。
- `benefits.refreshQuota`：刷新次数，必须大于等于 0。
- `benefits.topVoucherCount`：置顶券数量，必须大于等于 0。
- `benefits.topDurationHours`：置顶时长，配置置顶券时必须大于 0。
- `benefits.homepageImageLimit`：主页图片上限，必须大于等于 0。

保存规则：

- 新增或更新 `vip_plans` 的当前可售字段。
- 每次保存套餐权益都生成新的 `vip_plan_versions.version`。
- 旧版本保留，不影响已创建订单。
- 若权益字段完全未变化，也可以生成新版本，保证运营操作可追溯；实现阶段可按简洁性选择固定生成新版本。

### 次数包

列表字段：

- 名称
- 编码
- 描述
- 标准价
- 优惠价
- 权益摘要
- 状态
- 排序
- 更新时间

编辑字段：

- `code`：次数包编码，新增时填写，编辑后不允许修改。
- `name`：名称。
- `description`：描述。
- `standardPriceCent`：标准价，单位分，必须大于等于 0。
- `salePriceCent`：优惠价，允许为空；填写时必须大于等于 0 且不高于标准价。
- `saleLabel`：优惠标签。
- `status`：`active` 或 `inactive`。
- `displayOrder`：排序值。
- `benefits.publishQuota`
- `benefits.refreshQuota`
- `benefits.topVoucherCount`
- `benefits.topDurationHours`

保存规则：

- 直接新增或更新 `vip_quota_packs`。
- 已创建订单继续按订单内 `benefits_snapshot` 执行，不受次数包配置变更影响。

### 优惠活动

列表字段：

- 优惠编码
- 适用套餐
- 优惠类型
- 优惠价
- 时间范围
- 名额
- 已使用
- 状态
- 更新时间

编辑字段：

- `code`：优惠编码，新增时填写，编辑后不允许修改。
- `planCode`：适用 VIP 套餐。
- `promotionType`：`first_purchase` 或 `launch`。
- `salePriceCent`：优惠价，必须大于等于 0。
- `startsAt`：开始时间。
- `endsAt`：结束时间，允许为空；填写时必须晚于开始时间。
- `quotaLimit`：名额上限，允许为空；填写时必须大于等于已使用数量。
- `status`：`active` 或 `inactive`。

保存规则：

- 新增或更新 `vip_promotions`。
- 优惠生效判断继续沿用现有查询：状态启用、开始时间已到、结束时间未过、名额未用完。
- 首购优惠仍由创建订单逻辑判断商家是否已有已支付 VIP 订单。

## 后端接口设计

新增 `backend/app/api/admin.api` 后台合约类型和接口。

接口前缀统一使用 `/api/v1/admin/vip`：

- `GET /plans`
- `POST /plans`
- `POST /plans/:planCode`
- `GET /quota-packs`
- `POST /quota-packs`
- `POST /quota-packs/:packCode`
- `GET /promotions`
- `POST /promotions`
- `POST /promotions/:promotionCode`

请求和响应类型放在 `admin.api`，实际手写路由注册遵循当前项目模式，添加到 `backend/app/internal/server/domain_routes.go`。

新增后端逻辑文件：

- `backend/app/internal/logic/admin/vip_config_logic.go`
- `backend/app/internal/logic/admin/vip_config_logic_test.go`

新增或扩展模型方法：

- `ListAdminVIPPlans`
- `SaveAdminVIPPlan`
- `ListAdminQuotaPacks`
- `SaveAdminQuotaPack`
- `ListAdminVIPPromotions`
- `SaveAdminVIPPromotion`

这些方法可以直接放在 `backend/app/internal/model/vip_model.go`，因为现有 VIP 读写和支付快照逻辑已经集中在该文件中。实现时需保持函数边界清晰，避免把后台校验散落到 SQL 层。

## 数据流

### 套餐保存

1. 管理员在后台提交套餐表单。
2. 前端把金额统一转换为分提交。
3. 后端校验套餐编码、名称、周期、价格、状态、权益字段。
4. 后端在事务内 upsert `vip_plans`。
5. 后端查询当前最大版本号，插入新的 `vip_plan_versions`。
6. 后端写入 `operation_logs`，记录调整前后的关键字段。
7. 小程序下一次读取 `/api/v1/vip/plans` 时拿到最新 active 版本。

### 次数包保存

1. 管理员提交次数包表单。
2. 后端校验编码、名称、价格、状态和权益字段。
3. 后端 upsert `vip_quota_packs`。
4. 后端写操作日志。
5. 小程序下一次读取 `/api/v1/vip/quota-packs` 时拿到最新 active 次数包。

### 优惠保存

1. 管理员提交优惠表单。
2. 后端校验优惠编码、套餐存在性、优惠类型、价格、时间范围、名额。
3. 后端 upsert `vip_promotions`。
4. 后端写操作日志。
5. 新订单创建时继续按现有逻辑选择当前可用优惠。

## 权限与审计

- 所有 `/api/v1/admin/vip/*` 接口沿用管理后台管理员鉴权。
- 操作人从管理员 token 解析，前端不可信任传入操作人。
- 保存成功时写 `operation_logs`：
  - 套餐：`vip_plan_save`
  - 次数包：`vip_quota_pack_save`
  - 优惠：`vip_promotion_save`
- `object_type` 使用 `vip_config`。
- `object_id` 使用配置编码。
- `before_snapshot` 和 `after_snapshot` 记录关键字段，避免记录敏感信息。

## 错误处理与日志

后端用户可见错误使用中文：

- 编码为空：`请填写配置编码`
- 名称为空：`请填写配置名称`
- 周期无效：`套餐周期必须大于 0`
- 价格无效：`价格不能小于 0`
- 优惠价高于原价：`优惠价不能高于标准价`
- 结束时间无效：`结束时间必须晚于开始时间`
- 套餐不存在：`请选择有效的 VIP 套餐`
- 状态无效：`配置状态不正确`

内部日志使用 `logx`，至少包含：

- 操作类型。
- 配置编码。
- 操作人 ID。
- 关键状态。
- 原始错误。

示例：

```go
logx.Errorf("保存 VIP 套餐配置失败: operatorId=%s planCode=%s err=%+v", operatorID, input.Code, err)
```

## 前端实现约束

- 使用现有 Vue3、Vue Router、Element Plus 模式。
- 路由保持懒加载。
- 新增 Element Plus 组件时同步更新 `admin-web/src/plugins/elementPlus.js`。
- 金额输入在页面显示为元，提交时转换为分。
- 表格和抽屉沿用现有后台配置页风格，保持运营工具的密度和可扫描性。
- 不在页面内添加说明型大段文案，只保留必要字段标签、占位和确认提示。

## 测试计划

### 后端逻辑测试

覆盖：

- 保存套餐时校验价格、周期、状态、权益数量。
- 保存套餐时把输入传给 Store，并保留权益快照字段。
- 保存次数包时拒绝优惠价高于标准价。
- 保存优惠时拒绝结束时间早于开始时间。
- 保存优惠时拒绝不存在的套餐。

### 后端路由测试

覆盖：

- 管理员 token 可访问 `/api/v1/admin/vip/plans`。
- 保存套餐时操作人来自管理员 token，而不是请求体。
- 非管理员访问后台 VIP 配置接口返回未登录或无权限。

### 模型测试

覆盖：

- 保存套餐权益会插入新的 `vip_plan_versions`。
- 小程序公开套餐列表读取最新 active 版本。
- 次数包保存后公开接口只返回 active 配置。
- 优惠名额和时间窗口仍按现有查询规则生效。

### API 合约测试

更新 `backend/scripts/api_contract.test.mjs`，确认 `admin.api` 暴露后台 VIP 配置接口和 DTO。

### 前端测试

更新 `admin-web/scripts/feature-visibility.test.mjs` 或新增脚本测试，确认：

- 后台路由包含 `VIPConfigView` 懒加载。
- 侧边栏包含 `VIP 配置` 菜单。
- 页面包含套餐、次数包、优惠活动三个 tab。
- API 文件调用 `/api/v1/admin/vip` 前缀。

## 验收标准

1. 管理后台出现“VIP 配置”菜单。
2. 管理员可以新增、编辑、停用 VIP 套餐。
3. 管理员可以新增、编辑、停用次数包。
4. 管理员可以新增、编辑、停用优惠活动。
5. 小程序 VIP 页面读取到后台保存后的 active 套餐和次数包。
6. 新建 VIP 订单使用保存后的最新权益版本。
7. 已有订单的 `benefits_snapshot` 不被后台配置修改影响。
8. 非法配置提交会收到明确中文错误。
9. 后端保存失败会记录包含配置编码和操作人 ID 的诊断日志。
10. 相关 Go 测试、Node 脚本测试和后台构建通过。
