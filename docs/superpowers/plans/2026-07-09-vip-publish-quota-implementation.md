# VIP Publish Quota Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现“VIP 赠送发布次数 + 限时优惠”的 v1 闭环，包括套餐查询、订单支付、权益发放、发布额度扣减、VIP 标识展示和小程序入口。

**Architecture:** 后端新增 VIP 领域模型与逻辑，价格和权益由数据库配置驱动，订单保存购买快照；发布额度、刷新次数继续复用 `merchant_entitlements`，置顶券继续复用 `top_vouchers`。认证模块暂不删除，只在小程序主路径隐藏，VIP 不表达平台背书。

**Tech Stack:** Go、go-zero API 定义、PostgreSQL migrations、uni-app 小程序、Node test runner、Go test。

---

## 文件结构

- Create: `backend/migrations/000017_vip_membership.up.sql`
  - 新增 VIP 套餐、套餐版本、优惠、订单、订阅表，并写入冷启动默认套餐与优惠。
- Create: `backend/migrations/000017_vip_membership.down.sql`
  - 删除 VIP 新增表。
- Create: `backend/app/api/vip.api`
  - 定义 VIP 套餐、订阅、订单、支付接口。
- Modify: `backend/app/api/app.api`
  - 导入 `vip.api`。
- Modify: `backend/app/internal/types/types.go`
  - 同步 VIP API DTO。
- Create: `backend/app/internal/model/vip_model.go`
  - 负责 VIP 套餐查询、订单创建、支付成功、订阅续期和权益发放。
- Create: `backend/app/internal/logic/vip/vip_logic.go`
  - 负责套餐列表、商家 VIP 状态、创建订单。
- Create: `backend/app/internal/logic/payment/vip_payment_logic.go`
  - 复用微信支付网关创建 VIP 支付和处理 VIP 支付通知。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 注册 VIP 路由和微信支付 VIP 回调。
- Modify: `backend/app/internal/svc/service_context.go`
  - 将 `VIPModel` 注入 `APIStore`。
- Modify: `backend/app/internal/model/resource_model.go`
  - 发布和草稿提交时扣减发布额度；资源列表和详情返回 VIP 状态。
- Modify: `backend/app/internal/logic/resource/create_resource_logic.go`
  - 将发布额度不足映射为友好中文错误。
- Modify: `backend/app/internal/logic/resource/get_resource_logic.go`
  - 将商家 VIP 状态映射到资源详情。
- Modify: `backend/app/internal/logic/resource/list_resources_logic.go`
  - 将商家 VIP 状态映射到资源列表。
- Modify: `backend/app/internal/server/resource_api_test.go`
  - 补充 API 层发布额度不足和 VIP 字段测试。
- Create: `backend/app/internal/logic/vip/vip_logic_test.go`
  - VIP 逻辑单元测试。
- Create: `backend/app/internal/logic/payment/vip_payment_logic_test.go`
  - VIP 支付逻辑单元测试。
- Create: `backend/app/internal/model/vip_model_test.go`
  - 纯函数和支付单号测试。
- Modify: `wxapp/api/entitlement.js`
  - 如需展示发布额度，保留现有接口。
- Create: `wxapp/api/vip.js`
  - VIP 套餐、订阅、订单和支付 API。
- Modify: `wxapp/pages/my/index.vue`
  - 隐藏认证主入口，增加 VIP 权益入口。
- Create: `wxapp/pages/vip/index.vue`
  - VIP 套餐和权益展示页。
- Modify: `wxapp/pages.json`
  - 注册 VIP 页面。
- Modify: `wxapp/components/ResourceCard.vue`
  - 使用 VIP 标识替代认证标识。
- Modify: `wxapp/components/MerchantBadge.vue`
  - 展示 VIP 标识。
- Modify: `wxapp/pages/resource/detail.vue`
  - 资源详情展示 VIP 标识。
- Create: `wxapp/pages/vip/index.test.mjs`
  - VIP 页面结构和文案测试。
- Modify: `wxapp/scripts/validate-flows.test.mjs`
  - 更新认证入口与 VIP 入口断言。

## Task 1: 数据库迁移与 API 契约

**Files:**
- Create: `backend/migrations/000017_vip_membership.up.sql`
- Create: `backend/migrations/000017_vip_membership.down.sql`
- Create: `backend/app/api/vip.api`
- Modify: `backend/app/api/app.api`
- Modify: `backend/app/internal/types/types.go`
- Test: `backend/scripts/validate_migrations.test.mjs`
- Test: `backend/scripts/api_contract.test.mjs`

- [ ] **Step 1: 写失败测试**

在 `backend/scripts/api_contract.test.mjs` 增加断言：

```js
test('api contract exposes vip membership endpoints', () => {
  const appApi = fs.readFileSync(path.join(root, 'app/api/app.api'), 'utf8')
  const vipApi = fs.readFileSync(path.join(root, 'app/api/vip.api'), 'utf8')
  const typesGo = fs.readFileSync(path.join(root, 'app/internal/types/types.go'), 'utf8')

  assert.match(appApi, /import "vip\.api"/)
  assert.match(vipApi, /get \/vip\/plans returns \(ListVIPPlansResp\)/)
  assert.match(vipApi, /get \/merchants\/:merchantId\/vip returns \(MerchantVIPResp\)/)
  assert.match(vipApi, /post \/merchants\/:merchantId\/vip\/orders \(CreateVIPOrderReq\) returns \(CreateVIPOrderResp\)/)
  assert.match(vipApi, /post \/merchants\/:merchantId\/vip\/orders\/:orderId\/payment \(CreateVIPPaymentReq\) returns \(CreateVIPPaymentResp\)/)
  assert.match(typesGo, /type VIPPlanInfo struct/)
  assert.match(typesGo, /type MerchantVIPResp struct/)
})
```

- [ ] **Step 2: 运行失败测试**

Run: `node --test scripts/api_contract.test.mjs`

Expected: FAIL，原因是 `vip.api` 不存在或 `app.api` 未导入。

- [ ] **Step 3: 实现迁移和 API 定义**

`000017_vip_membership.up.sql` 创建：

- `vip_plans`
- `vip_plan_versions`
- `vip_promotions`
- `vip_orders`
- `merchant_vip_subscriptions`

种子数据：

- `monthly`：49 元，首月优惠 19.9 元。
- `half_year`：299 元，冷启动优惠 199 元。
- `yearly`：499 元，冷启动优惠 299 元。
- 权益版本：`publish_quota=80`，`refresh_quota=30`，`top_voucher_count=3`，`top_duration_hours=24`。

新增 `vip.api`，提供：

- `GET /api/v1/vip/plans`
- `GET /api/v1/merchants/:merchantId/vip`
- `POST /api/v1/merchants/:merchantId/vip/orders`
- `POST /api/v1/merchants/:merchantId/vip/orders/:orderId/payment`

- [ ] **Step 4: 运行契约和迁移测试**

Run:

```bash
node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs
goctl api validate --api app/api/app.api
```

Expected: PASS。

## Task 2: VIP 模型与订单权益发放

**Files:**
- Create: `backend/app/internal/model/vip_model.go`
- Create: `backend/app/internal/model/vip_model_test.go`
- Modify: `backend/app/internal/svc/service_context.go`

- [ ] **Step 1: 写失败测试**

新增模型测试：

```go
func TestBuildVIPOutTradeNoKeepsWechatSafeLength(t *testing.T) {
	got := buildVIPOutTradeNo("order-1234567890abcdefghijklmnopqrstuvwxyz")
	if !strings.HasPrefix(got, "VIP") {
		t.Fatalf("out trade no = %q, want VIP prefix", got)
	}
	if len(got) > 32 {
		t.Fatalf("out trade no length = %d, want <= 32", len(got))
	}
}

func TestVIPBenefitSnapshotFromJSONDefaultsToQuotaPolicy(t *testing.T) {
	snapshot := VIPBenefitSnapshotFromJSON(model.JSONMap{
		"publishQuota":     float64(80),
		"refreshQuota":     float64(30),
		"topVoucherCount":  float64(3),
		"topDurationHours": float64(24),
	})
	if snapshot.PublishPolicy != "quota" || snapshot.PublishQuota != 80 || snapshot.RefreshQuota != 30 || snapshot.TopVoucherCount != 3 {
		t.Fatalf("snapshot = %#v, want quota benefits", snapshot)
	}
}
```

- [ ] **Step 2: 运行失败测试**

Run: `go test ./app/internal/model -run 'TestBuildVIPOutTradeNo|TestVIPBenefitSnapshot' -count=1`

Expected: FAIL，原因是函数未定义。

- [ ] **Step 3: 实现模型**

`vip_model.go` 定义：

- `VIPPlan`
- `VIPBenefitSnapshot`
- `VIPPromotion`
- `MerchantVIPSummary`
- `VIPOrder`
- `CreateVIPOrderInput`
- `CreateVIPPaymentOrderInput`
- `MarkVIPOrderPaidInput`

实现方法：

- `ListVIPPlans(ctx)`
- `GetMerchantVIPSummary(ctx, merchantID)`
- `CreateVIPOrder(ctx, input)`
- `GetVIPPaymentContext(ctx, input)`
- `CreateVIPPaymentOrder(ctx, input)`
- `MarkVIPOrderPaid(ctx, input)`
- `GrantVIPMonthlyBenefits(ctx, tx, merchantID, orderID, snapshot, periodStart, periodEnd)`

支付成功必须在事务内完成：

1. 标记订单 `paid`。
2. 创建或延长有效订阅。
3. 发放当期 `publish_quota`、`refresh_quota`。
4. 生成 `top_vouchers`。
5. 写消息和操作日志。

- [ ] **Step 4: 运行模型测试**

Run: `go test ./app/internal/model -run 'TestBuildVIPOutTradeNo|TestVIPBenefitSnapshot' -count=1`

Expected: PASS。

## Task 3: VIP 逻辑与支付逻辑

**Files:**
- Create: `backend/app/internal/logic/vip/vip_logic.go`
- Create: `backend/app/internal/logic/vip/vip_logic_test.go`
- Create: `backend/app/internal/logic/payment/vip_payment_logic.go`
- Create: `backend/app/internal/logic/payment/vip_payment_logic_test.go`

- [ ] **Step 1: 写失败测试**

VIP 逻辑测试覆盖：

- 套餐列表返回优惠价。
- 创建订单时校验商家和套餐。
- VIP 状态返回 `active`、`expiresAt`、剩余额度。

支付逻辑测试覆盖：

- 网关存在时创建微信预支付。
- dev mock 开启时直接标记支付成功。
- 订单不存在、未登录、支付状态不正确返回友好中文错误。

- [ ] **Step 2: 运行失败测试**

Run:

```bash
go test ./app/internal/logic/vip ./app/internal/logic/payment -run 'VIP|CreateVIP' -count=1
```

Expected: FAIL，原因是 VIP 逻辑包或函数未定义。

- [ ] **Step 3: 实现逻辑**

返回错误必须使用中文友好文案：

- “请选择有效的 VIP 套餐”
- “请先登录后再开通 VIP”
- “VIP 订单已失效，请重新选择套餐”
- “微信支付暂未配置，请联系平台运营”

日志必须记录：

- 创建订单失败：`merchantId`、`planCode`、`userId`。
- 支付创建失败：`merchantId`、`orderId`、`userId`。
- 模拟支付成功：`merchantId`、`orderId`、`outTradeNo`。

- [ ] **Step 4: 运行逻辑测试**

Run:

```bash
go test ./app/internal/logic/vip ./app/internal/logic/payment -run 'VIP|CreateVIP' -count=1
```

Expected: PASS。

## Task 4: 路由注册与权限

**Files:**
- Modify: `backend/app/internal/server/domain_routes.go`
- Test: `backend/app/internal/server/remaining_api_test.go`

- [ ] **Step 1: 写失败测试**

在 `remaining_api_test.go` 增加路由测试：

- `GET /api/v1/vip/plans` 可公开访问。
- `GET /api/v1/merchants/merchant-1/vip` 需要商家权限。
- `POST /api/v1/merchants/merchant-1/vip/orders` 使用登录用户创建订单。
- `POST /api/v1/merchants/merchant-1/vip/orders/order-1/payment` 使用登录用户发起支付。

- [ ] **Step 2: 运行失败测试**

Run: `go test ./app/internal/server -run 'VIP|Remaining' -count=1`

Expected: FAIL，原因是路由未注册。

- [ ] **Step 3: 实现路由**

新增 `VIPAPIStore`：

```go
type VIPAPIStore interface {
    viplogic.Store
    paymentlogic.VIPPaymentStore
}
```

注册：

- `GET /api/v1/vip/plans`
- `GET /api/v1/merchants/{merchantId}/vip`
- `POST /api/v1/merchants/{merchantId}/vip/orders`
- `POST /api/v1/merchants/{merchantId}/vip/orders/{orderId}/payment`
- `POST /api/v1/wechat-pay/vip/notify`

- [ ] **Step 4: 运行路由测试**

Run: `go test ./app/internal/server -run 'VIP|Remaining' -count=1`

Expected: PASS。

## Task 5: 发布额度扣减

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/logic/resource/create_resource_logic.go`
- Modify: `backend/app/internal/logic/resource/submit_resource_logic.go`
- Test: `backend/app/internal/logic/resource/create_resource_logic_test.go`
- Test: `backend/app/internal/server/resource_api_test.go`

- [ ] **Step 1: 写失败测试**

新增测试：

- 创建待审核资源时，store 收到 `ConsumePublishQuota=true`。
- 保存草稿时不消耗发布额度。
- 提交草稿审核时消耗发布额度。
- 发布额度不足返回“本月发布次数已用完，可开通 VIP 或购买发布包”。

- [ ] **Step 2: 运行失败测试**

Run:

```bash
go test ./app/internal/logic/resource ./app/internal/server -run 'PublishQuota|CreateResource|SubmitResource' -count=1
```

Expected: FAIL，原因是扣减字段或错误映射未实现。

- [ ] **Step 3: 实现扣减**

`CreateResourceInput` 增加：

```go
ConsumePublishQuota bool
```

`CreateResource` 在事务内：

- 当目标状态不是 `draft` 时，先扣减可用 `publish_quota`。
- 扣减失败返回 `ErrPublishQuotaInsufficient`。
- 插入资源成功后提交事务。

`SubmitResourceForReview` 在事务内：

- 查询资源所属商家。
- 扣减 `publish_quota`。
- 更新草稿为 `pending`。

- [ ] **Step 4: 运行发布测试**

Run:

```bash
go test ./app/internal/logic/resource ./app/internal/server -run 'PublishQuota|CreateResource|SubmitResource' -count=1
```

Expected: PASS。

## Task 6: VIP 标识返回

**Files:**
- Modify: `backend/app/api/resource.api`
- Modify: `backend/app/api/merchant.api`
- Modify: `backend/app/internal/types/types.go`
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/model/merchant_model.go`
- Modify: `backend/app/internal/logic/resource/list_resources_logic.go`
- Modify: `backend/app/internal/logic/resource/get_resource_logic.go`
- Modify: `backend/app/internal/logic/merchant/get_merchant_logic.go`
- Test: `backend/app/internal/logic/resource/get_resource_logic_test.go`
- Test: `backend/app/internal/logic/merchant/get_merchant_logic_test.go`

- [ ] **Step 1: 写失败测试**

断言资源列表、资源详情、商家详情返回：

```json
{
  "vipStatus": "active"
}
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
go test ./app/internal/logic/resource ./app/internal/logic/merchant -run 'VIP|GetResource|GetMerchant' -count=1
```

Expected: FAIL，原因是字段未返回。

- [ ] **Step 3: 实现字段**

后端模型查询使用 `EXISTS` 判断：

```sql
EXISTS (
  SELECT 1
  FROM merchant_vip_subscriptions mvs
  WHERE mvs.merchant_id = m.id
    AND mvs.status = 'active'
    AND mvs.starts_at <= now()
    AND mvs.expires_at > now()
)
```

返回 `vipStatus = active` 或 `none`。

- [ ] **Step 4: 运行字段测试**

Run:

```bash
go test ./app/internal/logic/resource ./app/internal/logic/merchant -run 'VIP|GetResource|GetMerchant' -count=1
```

Expected: PASS。

## Task 7: 小程序 VIP 入口和展示

**Files:**
- Create: `wxapp/api/vip.js`
- Create: `wxapp/pages/vip/index.vue`
- Create: `wxapp/pages/vip/index.test.mjs`
- Modify: `wxapp/pages.json`
- Modify: `wxapp/pages/my/index.vue`
- Modify: `wxapp/components/ResourceCard.vue`
- Modify: `wxapp/components/MerchantBadge.vue`
- Modify: `wxapp/pages/resource/detail.vue`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: 写失败测试**

新增小程序测试断言：

- 存在 `/pages/vip/index`。
- 我的页面有 VIP 权益入口。
- 我的页面主路径不再展示商家认证入口。
- 资源卡和商家徽章展示 `VIP`，不展示 `已认证`。
- VIP 页面展示“年卡限时 ¥299”“80 条发布额度”“30 次刷新”“3 张置顶券”。

- [ ] **Step 2: 运行失败测试**

Run:

```bash
node --test pages/vip/index.test.mjs pages/my/index.test.mjs components/ResourceCard.test.mjs scripts/validate-flows.test.mjs
```

Expected: FAIL，原因是 VIP 页面和文案未实现。

- [ ] **Step 3: 实现小程序页面**

VIP 页面：

- 读取 `listVIPPlans`。
- 读取当前商家 VIP 状态。
- 展示套餐、优惠价、标准价、权益。
- 点击开通时创建订单并调起支付。
- 支付成功后刷新状态和权益。

我的页面：

- 增加 VIP 权益入口。
- 隐藏认证主入口。

资源组件：

- 使用 `merchant.vipStatus === 'active'` 展示 `VIP`。

- [ ] **Step 4: 运行小程序测试**

Run:

```bash
node --test pages/vip/index.test.mjs pages/my/index.test.mjs components/ResourceCard.test.mjs scripts/validate-flows.test.mjs
```

Expected: PASS。

## Task 8: 全量验证

**Files:**
- All touched files.

- [ ] **Step 1: 格式化**

Run:

```bash
gofmt -w backend/app/internal/model/vip_model.go backend/app/internal/logic/vip/vip_logic.go backend/app/internal/logic/payment/vip_payment_logic.go backend/app/internal/server/domain_routes.go backend/app/internal/svc/service_context.go backend/app/internal/model/resource_model.go backend/app/internal/logic/resource/create_resource_logic.go backend/app/internal/logic/resource/submit_resource_logic.go backend/app/internal/types/types.go
```

- [ ] **Step 2: 后端验证**

Run:

```bash
go test ./...
node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs
goctl api validate --api app/api/app.api
```

Expected: PASS。

- [ ] **Step 3: 小程序验证**

Run:

```bash
node --test $(find pages components common scripts -name '*.test.mjs' -print)
```

Expected: PASS。

- [ ] **Step 4: 最终检查**

Run:

```bash
git status --short
git diff --stat
```

确认只包含 VIP 实施相关文件。

