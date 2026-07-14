# 二级分类商业规则设计

## 背景

衣货通已经具备统一供需信息发布、二级分类配置、发布额度、VIP 套餐、次数包和受控联系方式解锁能力。现有规则整体偏通用：发布资源时默认消耗 `publish_quota`，资源详情默认只展示脱敏联系方式，登录用户点击电话或微信后可以解锁完整联系方式。

新的业务诉求是：部分二级分类需要降低发布门槛，例如“我要求职”这类需求可以免费发布；但查看这些需求联系方式的人，通常是招聘方或老板，需要付费查看，或者在拥有有效 VIP 权益时免费查看。这个规则不能写死在某个类型编码上，否则后续新增“找工人”“找档口”“求合伙”等分类时会反复改业务代码。

## 目标

1. 支持按二级分类配置发布是否免费。
2. 支持按二级分类配置查看联系方式是否登录免费、单次付费、VIP 免费或完全禁止。
3. 公开详情继续只返回脱敏联系方式，完整电话和微信必须通过后端解锁接口返回。
4. 付费查看联系方式必须有订单和解锁记录，便于重复查看、退款、审计和指标统计。
5. VIP 免费查看以当前系统的商家 VIP 订阅为准，后端验证查看人是否能管理有效 VIP 商家。
6. 后台运营可以配置二级分类商业规则，不需要代码发布才能调整策略。

## 非目标

1. 不引入即时通讯、在线简历、在线招聘管理或担保交易。
2. 不把 VIP 解释为平台认证或真实性背书。
3. 不改变资源主体模型，仍使用统一 `resources` 表承载供应和需求。
4. 不把联系方式明文放到公开详情、列表、搜索或分享接口。
5. 不做复杂风控评分，只保留后续扩展边界。

## 核心概念

### 发布策略

发布策略决定创建待审核资源或提交草稿审核时是否扣减发布额度。

- `consume_quota`：默认策略，发布消耗 1 次 `publish_quota`。
- `free`：发布免费，不扣减 `publish_quota`。
- `disabled`：暂不允许用户自主发布，用于运营临时关闭某个分类。

草稿保存不消耗发布额度。草稿提交审核时按提交时二级分类的发布策略判断。

### 联系方式解锁策略

联系方式解锁策略决定登录用户点击电话或微信时，是否能拿到完整联系方式。

- `login_free`：默认策略，登录后可免费解锁。
- `paid`：需要单次付费解锁。
- `paid_or_vip`：有效 VIP 商家管理员可免费解锁，其他登录用户需要单次付费。
- `vip_only`：只有有效 VIP 商家管理员可解锁，不提供单次付费。
- `disabled`：禁止解锁，前端展示“该分类暂不开放联系方式”。

“VIP 用户”在本系统中按商家 VIP 判断：查看人必须是某个有效 VIP 商家的管理员。若查看人管理多个商家，后端可以自动匹配其名下任一有效 VIP 商家；前端后续也可以传 `viewerMerchantId` 指定使用哪个商家的 VIP 权益。

## 二级分类配置

新增 `resource_type_configs.commercial_rules jsonb`，独立承载商业化规则，避免和 `display_template`、`review_rules` 混用。

后端接口和前端页面使用 `commercialRules` 作为 JSON 字段名，数据库使用 `commercial_rules`。迁移时需要为所有历史二级分类写入默认规则，保证未配置的分类继续保持现有行为。

默认配置：

```json
{
  "publish": {
    "mode": "consume_quota"
  },
  "contactUnlock": {
    "mode": "login_free",
    "priceCent": 0,
    "currency": "CNY",
    "vipFree": false,
    "repeatUnlockDays": 30
  }
}
```

“我要求职”一类配置示例：

```json
{
  "publish": {
    "mode": "free"
  },
  "contactUnlock": {
    "mode": "paid_or_vip",
    "priceCent": 500,
    "currency": "CNY",
    "vipFree": true,
    "repeatUnlockDays": 30
  }
}
```

配置校验规则：

- `publish.mode` 必须是 `consume_quota`、`free` 或 `disabled`。
- `contactUnlock.mode` 必须是 `login_free`、`paid`、`paid_or_vip`、`vip_only` 或 `disabled`。
- `paid` 和 `paid_or_vip` 的 `priceCent` 必须大于 0。
- `login_free`、`vip_only`、`disabled` 的 `priceCent` 可以为 0。
- `repeatUnlockDays` 必须在 1 到 365 之间，默认 30。
- `currency` 默认 `CNY`，首期只允许 `CNY`。

## 发布流程

后端创建待审核资源或提交草稿审核时读取二级分类商业规则。

判断顺序：

1. 校验资源类型、城市站、商家状态和必填字段。
2. 读取 `commercial_rules.publish.mode`。
3. `disabled` 时返回“该分类暂不开放发布”。
4. `free` 时创建资源或提交审核，不扣发布额度，记录日志 `publish_policy=free`。
5. `consume_quota` 时沿用现有 `merchant_entitlements.publish_quota` 扣减逻辑。

免费发布不写入权益核销流水，因为没有消耗权益。为便于审计，资源创建成功日志需要包含 `merchantId`、`resourceId`、`typeCode`、`publishPolicy` 和 `createdByRole`。

## 联系方式查看流程

公开详情接口仍只返回：

- 联系人。
- 脱敏电话。
- 脱敏微信。
- 联系方式访问提示 `contactAccess`，不包含明文电话和微信。

建议 `contactAccess` 示例：

```json
{
  "mode": "paid_or_vip",
  "priceCent": 500,
  "currency": "CNY",
  "vipFree": true,
  "unlocked": false,
  "actionText": "支付 5 元查看联系方式"
}
```

完整联系方式仍通过 `POST /api/v1/resources/{resourceId}/contact-events` 解锁，后端作为最终判断方。

判断顺序：

1. 校验 action 是 `phone` 或 `wechat`。
2. 校验用户已登录。
3. 校验资源存在、已发布、未过期、未下架，所属商家正常。
4. 校验对应联系方式存在，缺失时不创建订单、不写事件、不增加指标。
5. 如果查看人能管理资源所属商家，则免费返回联系方式，不写联系事件和指标。
6. 如果存在有效解锁记录，则返回联系方式，并正常写联系事件和指标。
7. 读取分类 `contactUnlock.mode`。
8. `login_free` 直接返回联系方式，并写联系事件和指标。
9. `paid_or_vip` 或 `vip_only` 时，如果查看人管理有效 VIP 商家，则创建或刷新 VIP 来源解锁记录，返回联系方式，并写联系事件和指标。
10. `paid` 或 `paid_or_vip` 且未解锁时，返回需要支付的业务错误，前端引导创建订单。
11. `vip_only` 且无有效 VIP 时，返回“开通 VIP 后可查看联系方式”。
12. `disabled` 返回“该分类暂不开放联系方式”。

需要支付时返回稳定错误码，例如 `PAYMENT_REQUIRED`，并附带价格信息：

```json
{
  "code": "PAYMENT_REQUIRED",
  "message": "支付后可查看联系方式",
  "data": {
    "priceCent": 500,
    "currency": "CNY"
  }
}
```

## 付费订单与解锁记录

新增联系方式解锁订单，避免复用 `vip_orders` 造成 VIP 套餐、次数包和单次联系方式商品混杂。

### `resource_contact_unlock_orders`

字段建议：

- `id`
- `resource_id`
- `buyer_user_id`
- `buyer_merchant_id`，可为空，用于记录使用哪个商家身份购买或享受 VIP。
- `type_code`
- `out_trade_no`
- `price_cent`
- `currency`
- `commercial_rules_snapshot`
- `status`：`pending`、`paid`、`closed`、`refunded`
- `paid_at`
- `expires_at`：待支付订单过期时间。
- `created_at`
- `updated_at`

创建订单时锁定价格快照。即使运营后续调整二级分类价格，已创建订单仍按订单快照执行。

### `resource_contact_unlocks`

字段建议：

- `id`
- `resource_id`
- `user_id`
- `viewer_merchant_id`，可为空。
- `source_type`：`paid`、`vip`、`login_free`、`owner`、`manual`
- `order_id`，付费解锁时关联订单。
- `starts_at`
- `expires_at`
- `created_at`
- `last_used_at`

唯一性建议：

- 同一 `resource_id`、`user_id`、`viewer_merchant_id`、未过期解锁只保留一条有效记录。
- 个人付费查看时 `viewer_merchant_id` 为空，按用户维度重复查看免付费。
- VIP 免费查看时 `viewer_merchant_id` 使用实际有效 VIP 商家 ID，解锁有效期不超过 `repeatUnlockDays`，也不超过 VIP 订阅到期时间。

支付成功后创建 `paid` 解锁记录。用户再次点击电话或微信时，后端命中有效解锁记录后返回完整联系方式，不重复收费。

## 支付流程

新增接口：

- `POST /api/v1/resources/{resourceId}/contact-unlock-orders`
- `POST /api/v1/resources/{resourceId}/contact-unlock-orders/{orderId}/payment`

创建订单时后端必须校验：

- 用户已登录。
- 资源仍可公开联系。
- 分类策略确实需要付费。
- 对应资源至少有一个可解锁联系方式。
- 当前用户不存在有效付费或 VIP 解锁记录。

如果用户已经解锁，创建订单接口直接返回 `alreadyUnlocked=true`，不创建新订单。

支付成功回调中：

1. 校验订单状态是 `pending`。
2. 校验支付金额等于订单快照金额。
3. 标记订单 `paid`。
4. 写入 `resource_contact_unlocks`。
5. 记录日志，包含 `orderId`、`resourceId`、`buyerUserId`、`priceCent` 和 `unlockExpiresAt`。

如果支付成功后资源已经下架，订单仍标记为 `paid`，但解锁接口返回“资源已下架，请联系平台处理退款”。后台后续可以基于订单状态执行人工退款。

## 小程序交互

资源详情页默认展示脱敏联系方式。

按钮文案：

- 未登录：`登录后查看`
- 登录免费：`查看联系方式`
- 已解锁：`拨打电话`、`复制微信`
- 付费：`支付 5 元查看`
- VIP 免费：`VIP 免费查看`
- VIP 限定：`开通 VIP 查看`
- 禁止解锁：`暂不开放联系方式`

点击流程：

1. 本地无 token 时跳登录页，登录后回到当前详情。
2. 有 token 时先调用联系方式解锁接口。
3. 如果后端直接返回电话或微信，执行拨号或复制。
4. 如果返回 `PAYMENT_REQUIRED`，创建联系方式解锁订单并发起微信支付。
5. 支付成功后再次调用联系方式解锁接口，拿到完整联系方式。
6. 所有失败提示展示后端中文错误，不展示内部错误。

## 后台配置

供需类型配置页增加“商业规则”区域：

- 发布策略：消耗发布额度、免费发布、暂停发布。
- 联系方式策略：登录免费、付费查看、付费或 VIP 免费、仅 VIP、暂停开放。
- 单次查看价格，单位元，提交后端为分。
- 重复查看有效期，单位天。
- VIP 免费开关。

高级 JSON 保留完整 `commercialRules` 编辑能力。后端校验必须阻止非法模式、非法价格和非法有效期。

## 指标与事件

联系方式成功返回后才写入联系事件和效果指标。

- 已付费、VIP 免费、登录免费解锁后点击电话，增加 `phone_click_count` 和 `contact_click_count`。
- 已付费、VIP 免费、登录免费解锁后复制微信，增加 `wechat_copy_count` 和 `contact_click_count`。
- 资源所属商家管理员查看自己的联系方式，不写联系事件，不增加指标。
- 支付订单创建不等于联系成功，不能计入电话或微信指标。

`resource_contact_events` 后续可增加 `unlock_source` 和 `unlock_id`，用于区分 `paid`、`vip`、`login_free` 等来源。

## 错误处理

面向前端的错误文案：

- 未登录：`请先登录后联系商家`
- 资源不可联系：`资源不存在或已下架`
- 电话缺失：`商家暂未填写电话`
- 微信缺失：`商家暂未填写微信，可电话联系`
- 需要支付：`支付后可查看联系方式`
- VIP 限定：`开通 VIP 后可查看联系方式`
- 暂停开放：`该分类暂不开放联系方式`
- 订单失效：`订单已失效，请重新下单`
- 支付金额异常：`支付信息异常，请联系平台处理`
- 保存失败：`操作失败，请稍后重试`

后端日志需要包含资源 ID、用户 ID、商家 ID、二级分类编码、策略模式、订单 ID 和根因错误。不能在日志中输出完整电话、微信号、支付通知原文中的敏感字段。

## 测试策略

后端测试：

- 免费发布分类创建待审核资源不扣发布额度。
- 默认发布分类仍扣 `publish_quota`。
- 暂停发布分类返回“该分类暂不开放发布”。
- 付费联系方式分类未付费时返回 `PAYMENT_REQUIRED`。
- 有效 VIP 商家管理员可免费解锁 `paid_or_vip` 分类联系方式。
- 非 VIP 用户支付成功后可重复查看同一资源联系方式。
- 发布方查看自己的资源联系方式不写事件和指标。
- 已下架、已过期资源不能创建联系方式查看订单。
- 缺少电话或微信时不创建订单、不计指标。

前端测试：

- 详情页按 `contactAccess` 展示正确按钮文案。
- `PAYMENT_REQUIRED` 后进入支付流程。
- 支付成功后再次解锁并复制微信或拨号。
- VIP 免费查看不展示支付按钮。
- 未登录点击联系方式会登录并回跳。

迁移和接口测试：

- migration 静态测试覆盖新表、索引和约束。
- API 合约测试覆盖新增订单和支付接口。
- go-zero 可生成的 DTO 必须从 `.api` 文件生成，不手写生成代码。

推荐验证命令：

```bash
node --test backend/scripts/validate_migrations.test.mjs
go test ./backend/app/internal/logic/resource ./backend/app/internal/logic/metrics ./backend/app/internal/logic/payment ./backend/app/internal/logic/admin ./backend/app/internal/server
node --test wxapp/pages/resource/detail.test.mjs
```

## 实施边界

优先修改：

- `backend/migrations`
- `backend/app/api/resource.api`
- `backend/app/api/admin.api`
- `backend/app/internal/model`
- `backend/app/internal/logic/resource`
- `backend/app/internal/logic/metrics`
- `backend/app/internal/logic/payment`
- `backend/app/internal/logic/admin/resource_type_config_logic.go`
- `backend/app/internal/server/domain_routes.go`
- `wxapp/pages/resource/detail.vue`
- `admin-web/src/views/ResourceTypeConfigView.vue`
- 对应测试文件

不应顺手重构地图、搜索、收藏、消息、认证和无关页面布局。

## 验收标准

1. 普通二级分类保持现有行为：发布消耗额度，登录后可解锁联系方式。
2. 配置为免费发布的二级分类发布时不扣 `publish_quota`。
3. 配置为付费查看的二级分类，未付费用户无法拿到完整联系方式。
4. 配置为付费或 VIP 免费的二级分类，有效 VIP 商家管理员可以免费查看完整联系方式。
5. 非 VIP 用户支付成功后，在有效期内重复查看同一资源联系方式不重复收费。
6. 公开详情、列表、搜索和分享接口都不暴露完整电话或微信。
7. 支付订单、解锁记录、联系事件和效果指标之间的含义清晰且可审计。
8. 后台可以配置二级分类商业规则，并拒绝非法价格、模式和有效期。
9. 推荐验证命令通过，或明确记录无法运行的环境原因。
