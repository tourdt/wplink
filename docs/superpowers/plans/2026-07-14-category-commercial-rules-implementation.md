# 二级分类商业规则 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按二级分类配置发布是否免费、联系方式是否付费或 VIP 免费查看，并完整接入订单、解锁记录、后台配置和小程序详情页。

**Architecture:** 在 `resource_type_configs` 上新增 `commercial_rules` 作为二级分类商业策略源，后端发布与联系方式解锁都读取同一规则。付费查看联系方式使用独立订单表和解锁记录表，避免混入 VIP 套餐订单；小程序只根据后端 `contactAccess` 和解锁接口结果驱动展示。

**Tech Stack:** Go/go-zero 风格后端、PostgreSQL migrations、Vue/uni-app 小程序、Element Plus 管理后台、Node 测试脚本。

---

## File Structure

- Modify: `backend/migrations/000002_core_domain.up.sql`
  - 新装库基础表补齐 `commercial_rules`、联系方式解锁订单表和解锁记录表。
- Create: `backend/migrations/000023_category_commercial_rules.up.sql`
  - 已有库增量迁移，避免和当前主工作区未提交的 `000022` 迁移冲突。
- Create: `backend/migrations/000023_category_commercial_rules.down.sql`
  - 回滚新增表和字段。
- Modify: `backend/scripts/validate_migrations.test.mjs`
  - 静态校验新字段、新表、索引和默认规则。
- Modify: `backend/scripts/api_contract.test.mjs`
  - 校验 `.api` 暴露 `commercialRules`、`contactAccess`、联系方式查看订单和支付接口。
- Modify: `backend/app/api/resource.api`
  - 增加联系方式访问提示、创建联系方式查看订单和支付 DTO/路由。
- Modify: `backend/app/api/admin.api`
  - 增加二级分类商业规则 DTO。
- Modify: `backend/app/internal/types/types.go`
  - 同步 `.api` DTO；若本地可用 goctl，优先从 `.api` 生成。
- Modify: `backend/app/internal/model/resource_type_config_model.go`
  - 读写 `commercial_rules`。
- Modify: `backend/app/internal/model/resource_model.go`
  - 发布扣额度策略读取 `commercial_rules.publish.mode`。
- Modify: `backend/app/internal/model/resource_contact_event_model.go`
  - 查询联系方式解锁所需的分类商业规则、已解锁状态和 VIP 判断。
- Create: `backend/app/internal/model/resource_contact_unlock_model.go`
  - 创建联系方式查看订单、支付单、标记支付成功、写入解锁记录。
- Modify: `backend/app/internal/logic/admin/resource_type_config_logic.go`
  - 校验和保存商业规则。
- Modify: `backend/app/internal/logic/resource/create_resource_logic.go`
  - 免费发布、暂停发布、默认扣额度。
- Modify: `backend/app/internal/logic/resource/submit_resource_logic.go`
  - 草稿提交审核按商业规则处理。
- Modify: `backend/app/internal/logic/resource/get_resource_logic.go`
  - 公开详情返回 `contactAccess`，仍不返回明文联系方式。
- Modify: `backend/app/internal/logic/metrics/record_contact_logic.go`
  - 付费/VIP/已解锁/登录免费判断，成功返回联系方式后计事件和指标。
- Create: `backend/app/internal/logic/payment/contact_unlock_payment_logic.go`
  - 创建联系方式查看订单支付、处理微信回调和开发模拟支付。
- Modify: `backend/app/internal/server/api.go`
  - 绑定资源公开详情、联系事件和新增订单接口。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 注册联系方式查看订单支付路由和后台商业规则字段。
- Modify: `backend/app/internal/server/remaining_api_test.go`
  - 覆盖新增路由。
- Modify: `admin-web/src/views/ResourceTypeConfigView.vue`
  - 增加商业规则可视化表单和高级 JSON 同步。
- Modify: `wxapp/api/resource.js`
  - 增加联系方式查看订单和支付接口封装。
- Modify: `wxapp/pages/resource/detail.vue`
  - 展示 `contactAccess`，按付费/VIP/已解锁流程解锁电话或微信。

## Task 1: 迁移和 API 合同

**Files:**
- Modify: `backend/migrations/000002_core_domain.up.sql`
- Create: `backend/migrations/000023_category_commercial_rules.up.sql`
- Create: `backend/migrations/000023_category_commercial_rules.down.sql`
- Modify: `backend/scripts/validate_migrations.test.mjs`
- Modify: `backend/scripts/api_contract.test.mjs`
- Modify: `backend/app/api/resource.api`
- Modify: `backend/app/api/admin.api`
- Modify: `backend/app/internal/types/types.go`

- [ ] **Step 1: 写失败的 migration 静态测试**

在 `backend/scripts/validate_migrations.test.mjs` 增加断言：

```js
test('category commercial rules migration defines contact unlock commerce schema', () => {
  const source = readMigrationSql(migrationsDir, '000023_category_commercial_rules.up.sql')
  assert.match(source, /ALTER TABLE resource_type_configs\s+ADD COLUMN IF NOT EXISTS commercial_rules jsonb/i)
  assert.match(source, /CREATE TABLE IF NOT EXISTS resource_contact_unlock_orders/i)
  assert.match(source, /CREATE TABLE IF NOT EXISTS resource_contact_unlocks/i)
  assert.match(source, /idx_contact_unlock_orders_out_trade_no/i)
  assert.match(source, /idx_contact_unlocks_resource_user/i)
})
```

- [ ] **Step 2: 运行 migration 测试确认失败**

Run: `node --test backend/scripts/validate_migrations.test.mjs`

Expected: FAIL，提示找不到 `000023_category_commercial_rules.up.sql` 或相关 SQL 片段。

- [ ] **Step 3: 写迁移**

创建 `000023` up/down，up 增加：

```sql
ALTER TABLE resource_type_configs
  ADD COLUMN IF NOT EXISTS commercial_rules jsonb NOT NULL DEFAULT '{"publish":{"mode":"consume_quota"},"contactUnlock":{"mode":"login_free","priceCent":0,"currency":"CNY","vipFree":false,"repeatUnlockDays":30}}'::jsonb;

CREATE TABLE IF NOT EXISTS resource_contact_unlock_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  buyer_user_id bigint NOT NULL REFERENCES users(id),
  buyer_merchant_id bigint REFERENCES merchants(id),
  type_code varchar(64) NOT NULL,
  out_trade_no varchar(64) UNIQUE NOT NULL,
  price_cent integer NOT NULL,
  currency varchar(16) NOT NULL DEFAULT 'CNY',
  commercial_rules_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  paid_at timestamptz,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS resource_contact_unlocks (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  user_id bigint NOT NULL REFERENCES users(id),
  viewer_merchant_id bigint REFERENCES merchants(id),
  source_type varchar(32) NOT NULL,
  order_id bigint REFERENCES resource_contact_unlock_orders(id),
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz
);
```

同步 `000002_core_domain.up.sql`，保证新装库直接具备同样字段和表。

- [ ] **Step 4: 写失败的 API 合同测试**

在 `backend/scripts/api_contract.test.mjs` 增加断言：

```js
test('resource api exposes contact unlock order contracts', () => {
  const resourceApiSource = fs.readFileSync(path.join(apiDir, 'resource.api'), 'utf8')
  assert(resourceApiSource.includes('type ResourceContactAccess'))
  assert(resourceApiSource.includes('type CreateContactUnlockOrderReq'))
  assert(resourceApiSource.includes('post /resources/:resourceId/contact-unlock-orders'))
  assert(resourceApiSource.includes('post /resources/:resourceId/contact-unlock-orders/:orderId/payment'))
})
```

- [ ] **Step 5: 运行 API 合同测试确认失败**

Run: `node --test backend/scripts/api_contract.test.mjs`

Expected: FAIL，提示缺少 `ResourceContactAccess` 或新增路由。

- [ ] **Step 6: 更新 `.api` 和 types**

在 `resource.api` 增加 `ResourceContactAccess`、订单和支付 DTO；在 `admin.api` 增加 `AdminResourceCommercialRules`。优先用 goctl 生成 `types.go`；如果本地 goctl 不可用，则手动同步 `backend/app/internal/types/types.go` 并记录原因。

- [ ] **Step 7: 运行 Task 1 验证**

Run:

```bash
node --test backend/scripts/validate_migrations.test.mjs
node --test backend/scripts/api_contract.test.mjs
```

Expected: PASS。

## Task 2: 商业规则模型和后台校验

**Files:**
- Modify: `backend/app/internal/model/resource_type_config_model.go`
- Modify: `backend/app/internal/logic/admin/resource_type_config_logic_test.go`
- Modify: `backend/app/internal/logic/admin/resource_type_config_logic.go`

- [ ] **Step 1: 写失败的后台规则测试**

增加测试：

```go
func TestUpdateResourceTypeConfigRejectsPaidContactWithoutPrice(t *testing.T) {
  logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})
  _, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
    FieldSchema: map[string]interface{}{},
    DisplayTemplate: map[string]interface{}{},
    ReviewRules: map[string]interface{}{},
    SortWeights: map[string]interface{}{},
    MessageRules: map[string]interface{}{},
    DefaultValidDays: 15,
    Status: "active",
    CommercialRules: map[string]interface{}{
      "publish": map[string]interface{}{"mode": "free"},
      "contactUnlock": map[string]interface{}{"mode": "paid_or_vip", "priceCent": float64(0), "currency": "CNY", "repeatUnlockDays": float64(30)},
    },
  })
  if err == nil || !strings.Contains(err.Error(), "查看价格") {
    t.Fatalf("UpdateResourceTypeConfig() error = %v, want price validation", err)
  }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/admin -run ResourceTypeConfig -count=1`

Workdir: `backend`

Expected: FAIL，提示 `CommercialRules` 字段不存在或校验未触发。

- [ ] **Step 3: 实现商业规则 DTO、默认值和校验**

在 admin logic 中增加：

```go
const (
  PublishPolicyConsumeQuota = "consume_quota"
  PublishPolicyFree = "free"
  PublishPolicyDisabled = "disabled"
)
```

实现 `normalizeCommercialRules`，对缺失配置补默认值，对付费模式校验价格。

- [ ] **Step 4: 更新模型读写**

`ResourceTypeConfig`、`AdminResourceTypeConfig`、`ResourcePublishConfig`、patch/create input 增加 `CommercialRules model.JSONMap`，SQL select/insert/update 都包含 `commercial_rules`。

- [ ] **Step 5: 运行 Task 2 验证**

Run: `go test ./app/internal/logic/admin ./app/internal/model -run 'ResourceTypeConfig|JSON' -count=1`

Workdir: `backend`

Expected: PASS。

## Task 3: 发布免费和暂停发布

**Files:**
- Modify: `backend/app/internal/logic/resource/create_resource_logic_test.go`
- Modify: `backend/app/internal/logic/resource/create_resource_logic.go`
- Modify: `backend/app/internal/logic/resource/submit_resource_logic_test.go`
- Modify: `backend/app/internal/logic/resource/submit_resource_logic.go`
- Modify: `backend/app/internal/model/resource_model.go`

- [ ] **Step 1: 写失败的免费发布测试**

在 create logic 测试 fake config 中设置：

```go
CommercialRules: model.JSONMap{
  "publish": model.JSONMap{"mode": "free"},
  "contactUnlock": model.JSONMap{"mode": "paid_or_vip", "priceCent": 500, "currency": "CNY", "repeatUnlockDays": 30},
},
```

断言 `store.input.ConsumePublishQuota == false`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/resource -run TestCreateResourceFreePublish -count=1`

Workdir: `backend`

Expected: FAIL，当前仍会消耗发布额度。

- [ ] **Step 3: 实现发布策略**

在 `buildResourceInput` 中读取 `config.CommercialRules.publish.mode`：

```go
publishMode := model.PublishModeFromCommercialRules(config.CommercialRules)
if publishMode == model.ResourcePublishModeDisabled {
  return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, "该分类暂不开放发布")
}
consumeQuota := status != model.ResourceStatusDraft && publishMode != model.ResourcePublishModeFree
```

- [ ] **Step 4: 支持草稿提交按策略扣费**

`SubmitResourceForReview` 查询资源关联二级分类的 `commercial_rules`，免费分类跳过 `consumePublishQuotaTx`，暂停发布返回业务错误。

- [ ] **Step 5: 运行 Task 3 验证**

Run: `go test ./app/internal/logic/resource ./app/internal/model -run 'CreateResource|SubmitResource|ResourceModel' -count=1`

Workdir: `backend`

Expected: PASS。

## Task 4: 联系方式订单、支付和解锁模型

**Files:**
- Create: `backend/app/internal/model/resource_contact_unlock_model.go`
- Create: `backend/app/internal/model/resource_contact_unlock_model_test.go`
- Create: `backend/app/internal/logic/payment/contact_unlock_payment_logic.go`
- Create: `backend/app/internal/logic/payment/contact_unlock_payment_logic_test.go`

- [ ] **Step 1: 写失败的支付 logic 测试**

测试开发模拟支付会标记订单支付并创建解锁：

```go
func TestCreateContactUnlockPaymentUsesDevMock(t *testing.T) {
  store := &fakeContactUnlockPaymentStore{
    context: model.ContactUnlockPaymentContext{
      OrderID: "order-1", ResourceID: "resource-1", UserID: "user-1",
      OpenID: "openid-1", Status: model.PaymentOrderStatusPending,
      OutTradeNo: "contact_unlock_1", AmountTotal: 500, Currency: "CNY",
      ResourceTitle: "求职需求",
    },
    order: model.ContactUnlockPaymentOrder{
      ID: "order-1", ResourceID: "resource-1", OutTradeNo: "contact_unlock_1",
      AmountTotal: 500, Currency: "CNY", Status: model.PaymentOrderStatusPending,
      ResourceTitle: "求职需求",
    },
    markResult: model.ContactUnlockPaymentResult{
      OrderID: "order-1", ResourceID: "resource-1", Status: model.PaymentOrderStatusPaid,
    },
  }
  logic := NewCreateContactUnlockPaymentLogic(store, nil, true)
  resp, err := logic.CreateContactUnlockPayment(context.Background(), CreateContactUnlockPaymentReq{
    ResourceID: "resource-1", OrderID: "order-1", UserID: "user-1",
  })
  if err != nil { t.Fatalf("CreateContactUnlockPayment() error = %v", err) }
  if store.markInput.OutTradeNo != "contact_unlock_1" || resp.Status != model.PaymentOrderStatusPaid {
    t.Fatalf("markInput = %#v resp=%#v, want paid contact unlock", store.markInput, resp)
  }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/payment -run ContactUnlock -count=1`

Workdir: `backend`

Expected: FAIL，logic 不存在。

- [ ] **Step 3: 实现模型和支付 logic**

实现 store 方法：

- `CreateContactUnlockOrder`
- `GetContactUnlockPaymentContext`
- `CreateContactUnlockPaymentOrder`
- `MarkContactUnlockOrderPaid`
- `HasActiveContactUnlock`
- `UpsertContactUnlock`

支付 logic 复用 `WechatPayGateway` 和 `nowFunc`，描述使用 `联系方式查看 - {资源标题}`。

- [ ] **Step 4: 运行 Task 4 验证**

Run: `go test ./app/internal/model ./app/internal/logic/payment -run 'ContactUnlock|Payment' -count=1`

Workdir: `backend`

Expected: PASS。

## Task 5: 联系方式解锁策略

**Files:**
- Modify: `backend/app/internal/model/resource_contact_event_model.go`
- Modify: `backend/app/internal/logic/metrics/record_contact_logic_test.go`
- Modify: `backend/app/internal/logic/metrics/record_contact_logic.go`

- [ ] **Step 1: 写失败的付费拦截测试**

增加测试：

```go
func TestRecordContactRequiresPaymentForPaidCategory(t *testing.T) {
  store := &fakeContactStore{
    contact: model.ResourceContactUnlockInfo{
      ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished,
      Phone: "18800000002",
      CommercialRules: model.JSONMap{"contactUnlock": model.JSONMap{"mode": "paid_or_vip", "priceCent": 500, "currency": "CNY", "repeatUnlockDays": 30}},
    },
  }
  _, err := NewRecordContactLogic(store).RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-1", Action: "phone"})
  if err == nil || errx.CodeOf(err) != errx.CodePaymentRequired {
    t.Fatalf("RecordContact() error = %v, want payment required", err)
  }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/metrics -run RecordContact -count=1`

Workdir: `backend`

Expected: FAIL，缺少商业规则字段或错误码。

- [ ] **Step 3: 实现策略判断**

扩展 `ContactStore`：

- `HasActiveContactUnlock(ctx, resourceID, userID string) (model.ContactUnlockState, error)`
- `FindActiveVIPManagedMerchant(ctx, userID string) (model.VIPManagedMerchant, error)`
- `UpsertContactUnlock(ctx, input model.ContactUnlockInput) (model.ContactUnlockResult, error)`

`validateContactUnlock` 按 owner、已解锁、login_free、VIP、需要支付、vip_only、disabled 顺序返回。

- [ ] **Step 4: 运行 Task 5 验证**

Run: `go test ./app/internal/logic/metrics ./app/internal/model -run 'RecordContact|ContactUnlock' -count=1`

Workdir: `backend`

Expected: PASS。

## Task 6: 资源详情 contactAccess 和路由

**Files:**
- Modify: `backend/app/internal/logic/resource/get_resource_logic_test.go`
- Modify: `backend/app/internal/logic/resource/get_resource_logic.go`
- Modify: `backend/app/internal/server/resource_api_test.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/app/internal/server/api.go`
- Modify: `backend/app/internal/server/domain_routes.go`

- [ ] **Step 1: 写失败的详情测试**

断言公开详情返回：

```go
if resp.ContactAccess.Mode != "paid_or_vip" || resp.ContactAccess.PriceCent != 500 || resp.ContactAccess.Unlocked {
  t.Fatalf("contactAccess = %#v, want paid_or_vip locked", resp.ContactAccess)
}
if resp.Contact.Phone != "" {
  t.Fatalf("public detail exposed raw phone")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./app/internal/logic/resource -run GetResource -count=1`

Workdir: `backend`

Expected: FAIL，缺少 `ContactAccess`。

- [ ] **Step 3: 实现详情映射和路由**

从 `ResourceDetail.DisplayTemplate/CommercialRules` 映射 `ContactAccess`。新增订单创建和支付 HTTP 路由，token userId 必须来自 Authorization，不接受前端伪造。

- [ ] **Step 4: 运行 Task 6 验证**

Run: `go test ./app/internal/server ./app/internal/logic/resource -run 'Resource|ContactUnlock' -count=1`

Workdir: `backend`

Expected: PASS。

## Task 7: 后台商业规则 UI

**Files:**
- Modify: `admin-web/src/views/ResourceTypeConfigView.vue`
- Modify: `admin-web/src/api/city.js`
- Add or Modify: `admin-web/scripts/feature-visibility.test.mjs`

- [ ] **Step 1: 写失败的后台前端测试**

在现有脚本测试中断言源代码包含：

```js
assert(source.includes('commercialRules'), 'resource type config should edit commercialRules')
assert(source.includes('付费或 VIP 免费'), 'commercial rules UI should expose paid or vip option')
```

- [ ] **Step 2: 运行测试确认失败**

Run: `node --test admin-web/scripts/feature-visibility.test.mjs`

Expected: FAIL，缺少 `commercialRules`。

- [ ] **Step 3: 实现 UI**

在可视化配置中增加商业规则表单，保存时提交 `commercialRules`；切换高级 JSON 时同步 `commercialRules`。

- [ ] **Step 4: 运行 Task 7 验证**

Run: `node --test admin-web/scripts/feature-visibility.test.mjs`

Expected: PASS。

## Task 8: 小程序详情页付费/VIP 解锁

**Files:**
- Modify: `wxapp/api/resource.js`
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue`

- [ ] **Step 1: 写失败的小程序详情测试**

增加测试断言：

```js
assert(source.includes('contactAccess'), 'resource detail should read contactAccess')
assert(source.includes('createContactUnlockOrder'), 'resource detail should create contact unlock order')
assert(source.includes('PAYMENT_REQUIRED'), 'resource detail should branch on payment required')
```

- [ ] **Step 2: 运行测试确认失败**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: FAIL，缺少新流程。

- [ ] **Step 3: 实现 API 和页面交互**

`wxapp/api/resource.js` 增加 `createContactUnlockOrder`、`createContactUnlockPayment`。详情页点击电话/微信时先走联系解锁接口；遇到 `PAYMENT_REQUIRED` 时创建订单并支付，支付后重试解锁。

- [ ] **Step 4: 运行 Task 8 验证**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: PASS。

## Task 9: 全量验证和提交

**Files:**
- All touched files.

- [ ] **Step 1: 格式化 Go 代码**

Run: `gofmt -w backend/app/internal/model backend/app/internal/logic backend/app/internal/server backend/app/internal/types/types.go`

- [ ] **Step 2: 运行后端测试**

Run:

```bash
go test ./app/internal/logic/resource ./app/internal/logic/metrics ./app/internal/logic/payment ./app/internal/logic/admin ./app/internal/model ./app/internal/server
```

Workdir: `backend`

Expected: PASS。

- [ ] **Step 3: 运行脚本测试**

Run:

```bash
node --test backend/scripts/validate_migrations.test.mjs
node --test backend/scripts/api_contract.test.mjs
node --test wxapp/pages/resource/detail.test.mjs
node --test admin-web/scripts/feature-visibility.test.mjs
```

Expected: PASS。

- [ ] **Step 4: 检查 diff**

Run: `git diff --check`

Expected: PASS，无空白错误。

- [ ] **Step 5: 提交**

Run:

```bash
git add backend admin-web wxapp docs/superpowers/plans/2026-07-14-category-commercial-rules-implementation.md
git commit -m "feat: add category commercial rules"
```
