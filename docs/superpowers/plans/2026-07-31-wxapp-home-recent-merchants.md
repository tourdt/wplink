# 小程序首页新入驻商家实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在小程序首页展示最多 6 家最近完成正式入驻的自主商家，为新商家提供曝光并增强平台活跃感。

**Architecture:** 以 `merchants.onboarded_at` 固化首次完成资料时间，MerchantModel 提供严格过滤后的首页查询，discovery 领域暴露固定上限的公开接口。小程序并行加载该接口，在快捷入口与近期供需之间渲染两列卡片；空结果或请求失败时静默隐藏。

**Tech Stack:** PostgreSQL migrations、Go 1.25、go-zero/logx、Go `net/http` 路由、uni-app/Vue 3、Node.js `node:test`

## Global Constraints

- 所有实施计划、设计文档、产品文档使用中文撰写。
- 首页最多展示 6 家，数量由服务端固定，客户端不能扩大。
- 只展示 `active + completed + onboarded_at 非空 + active owner 绑定` 的自主入驻商家。
- 预收录档口、资料未完成、停用和删除商家不得进入首页。
- 首次完成资料后固定 `onboarded_at`，后续编辑不得刷新排序。
- 公开接口不得返回联系人、手机号、微信或其他敏感资料。
- 首页接口失败或空结果时隐藏区块，不阻断现有首页内容。
- `.api` 是接口契约来源；生产路由继续遵循仓库现有的薄路由与 discovery logic 模式。
- 后端错误记录可诊断日志，前端只接收安全、清晰的中文提示。

---

### Task 1: 固化正式入驻时间并提供商家查询

**Files:**
- Create: `backend/migrations/000032_merchant_onboarded_at.up.sql`
- Create: `backend/migrations/000032_merchant_onboarded_at.down.sql`
- Modify: `backend/scripts/validate_migrations.test.mjs`
- Modify: `backend/app/internal/model/merchant_model.go`
- Create: `backend/app/internal/model/merchant_model_test.go`

**Interfaces:**
- Produces: `model.HomeRecentMerchant`
- Produces: `MerchantModel.ListHomeRecentMerchants(ctx context.Context, cityCode string, limit int64) ([]HomeRecentMerchant, error)`
- Preserves: `MerchantModel.UpdateMerchant(ctx, merchantID, patch)` existing signature

- [ ] **Step 1: 为迁移和 SQL 业务规则编写失败测试**

在迁移测试中断言 `000032` 同时包含字段、回填、部分索引和回滚；在 `merchant_model_test.go` 中断言首页查询 SQL 包含完成状态、有效 owner、城市、正式入驻时间倒序与固定 LIMIT，并断言更新 SQL 使用 `COALESCE(onboarded_at, $9)` 固定首次时间。

```go
func TestListHomeRecentMerchantsSQLScopesEligibleSelfOnboardedMerchants(t *testing.T) {
    for _, token := range []string{
        "m.status = 'active'",
        "m.profile_status = 'completed'",
        "m.onboarded_at IS NOT NULL",
        "mab.status = 'active'",
        "mab.role = 'owner'",
        "cs.code = $1",
        "ORDER BY m.onboarded_at DESC, m.id DESC",
        "LIMIT $2",
    } {
        if !strings.Contains(listHomeRecentMerchantsSQL, token) {
            t.Fatalf("query missing %q", token)
        }
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
go test ./app/internal/model
```

Expected: FAIL，提示 `000032` 不存在以及 `listHomeRecentMerchantsSQL` 未定义。

- [ ] **Step 3: 新增迁移并更新商家模型**

迁移新增：

```sql
ALTER TABLE merchants
  ADD COLUMN IF NOT EXISTS onboarded_at timestamptz;

UPDATE merchants
SET onboarded_at = created_at
WHERE profile_status = 'completed'
  AND onboarded_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_merchants_home_recent
  ON merchants(city_station_id, onboarded_at DESC, id DESC)
  WHERE deleted_at IS NULL
    AND status = 'active'
    AND profile_status = 'completed'
    AND onboarded_at IS NOT NULL;
```

更新商家资料 SQL 将首次完成时间与资料完成状态在同一事务内写入：

```sql
onboarded_at = COALESCE(onboarded_at, $9),
profile_status = 'completed'
```

首页查询返回 `id/name/merchant_type/main_categories/logo_url/address_text/onboarded_at`，通过 `EXISTS` 或 owner 绑定 JOIN 排除平台预录和无自主归属商家。

- [ ] **Step 4: 格式化并运行模型与迁移测试**

Run:

```bash
gofmt -w app/internal/model/merchant_model.go app/internal/model/merchant_model_test.go
node --test scripts/validate_migrations.test.mjs
go test ./app/internal/model
```

Expected: PASS。

- [ ] **Step 5: 提交数据层改动**

```bash
git add backend/migrations/000032_merchant_onboarded_at.up.sql backend/migrations/000032_merchant_onboarded_at.down.sql backend/scripts/validate_migrations.test.mjs backend/app/internal/model/merchant_model.go backend/app/internal/model/merchant_model_test.go
git commit -m "feat: 记录商家正式入驻时间"
```

### Task 2: 新增首页新入驻商家公开接口

**Files:**
- Modify: `backend/app/api/discovery.api`
- Modify: `backend/app/internal/logic/discovery/banner_topic_logic.go`
- Modify: `backend/app/internal/logic/discovery/banner_topic_logic_test.go`
- Modify: `backend/app/internal/server/domain_routes.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/scripts/api_contract.test.mjs`

**Interfaces:**
- Consumes: `ListHomeRecentMerchants(ctx, cityCode, limit)`
- Produces: `GET /api/v1/home/recent-merchants?cityCode=<code>`
- Produces: `ListHomeRecentMerchantsResp{Items []HomeRecentMerchantItem}`

- [ ] **Step 1: 编写逻辑、路由和契约失败测试**

逻辑测试验证城市去空格、固定 `limit=6`、字段映射及友好错误：

```go
func TestListHomeRecentMerchantsUsesFixedHomepageLimit(t *testing.T) {
    resp, err := logic.ListHomeRecentMerchants(context.Background(), ListHomeRecentMerchantsReq{CityCode: " zhili "})
    // 断言 store 收到 zhili 和 6，响应只含公开字段。
}

func TestListHomeRecentMerchantsReturnsFriendlyError(t *testing.T) {
    // store 返回 db timeout，断言公开文案为“新入驻商家加载失败，请稍后重试”。
}
```

路由测试将 `/api/v1/home/recent-merchants?cityCode=zhili` 加入现有全路由用例，并验证响应成功。

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend
node --test scripts/api_contract.test.mjs
go test ./app/internal/logic/discovery ./app/internal/server
```

Expected: FAIL，提示接口契约、逻辑方法和路由尚不存在。

- [ ] **Step 3: 先修改 `.api` 契约，再实现 logic 与薄路由**

`.api` 新增：

```go
type HomeRecentMerchantItem {
    Id             string   `json:"id"`
    Name           string   `json:"name"`
    MerchantType   string   `json:"merchantType"`
    MainCategories []string `json:"mainCategories"`
    LogoUrl        string   `json:"logoUrl,optional"`
    AddressText    string   `json:"addressText,optional"`
    OnboardedAt    string   `json:"onboardedAt"`
}

type HomeRecentMerchantsResp {
    Items []HomeRecentMerchantItem `json:"items"`
}
```

logic 固定使用 `homeRecentMerchantsLimit = 6`。数据库失败时使用 `logx.Errorf` 记录 `cityCode/limit/root error`，返回安全中文错误；route 只读取 `cityCode` 并调用 logic。

- [ ] **Step 4: 运行接口测试**

Run:

```bash
gofmt -w app/internal/logic/discovery/banner_topic_logic.go app/internal/logic/discovery/banner_topic_logic_test.go app/internal/server/domain_routes.go app/internal/server/remaining_api_test.go
node --test scripts/api_contract.test.mjs
go test ./app/internal/logic/discovery ./app/internal/server
```

Expected: PASS。

- [ ] **Step 5: 提交公开接口**

```bash
git add backend/app/api/discovery.api backend/app/internal/logic/discovery/banner_topic_logic.go backend/app/internal/logic/discovery/banner_topic_logic_test.go backend/app/internal/server/domain_routes.go backend/app/internal/server/remaining_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "feat: 新增首页新入驻商家接口"
```

### Task 3: 在小程序首页展示最多 6 家商家

**Files:**
- Modify: `wxapp/api/discovery.js`
- Create: `wxapp/components/HomeRecentMerchantCard.vue`
- Create: `wxapp/components/HomeRecentMerchantCard.test.mjs`
- Modify: `wxapp/pages/home/index.vue`
- Modify: `wxapp/pages/home/index.test.mjs`

**Interfaces:**
- Consumes: `listHomeRecentMerchants({cityCode})`
- Consumes: `HomeRecentMerchantItem`
- Produces: 首页“新入驻商家”两列卡片区块

- [ ] **Step 1: 编写组件和首页失败测试**

测试断言：

- 首页并行调用 `listHomeRecentMerchants`；
- 只对 `items.slice(0, 6)` 赋值；
- 区块使用 `v-if="recentMerchants.length"`；
- “更多”进入 `/pages/sourcing-map/index`；
- 卡片点击进入 `/pages/merchant/detail?id=<id>`；
- 卡片展示“新入驻”、主营品类和入驻日期，不含电话、微信、VIP。

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd wxapp
node --test pages/home/index.test.mjs components/HomeRecentMerchantCard.test.mjs
```

Expected: FAIL，提示组件和 API 调用不存在。

- [ ] **Step 3: 实现 API、卡片和首页区块**

`wxapp/api/discovery.js` 新增静默请求：

```js
export function listHomeRecentMerchants(params = {}) {
  return request({
    url: '/api/v1/home/recent-merchants',
    method: 'GET',
    data: params,
    suppressErrorToast: true,
  })
}
```

首页在 `loadHomeData()` 的 `Promise.all` 中并行加载；成功后最多保留 6 家，失败时清空数组。模板将新商家区块放在快捷入口后、近期供需前，使用两列网格展示卡片。

- [ ] **Step 4: 运行小程序测试**

Run:

```bash
npm --prefix wxapp run check
```

Expected: PASS。

- [ ] **Step 5: 提交小程序展示**

```bash
git add wxapp/api/discovery.js wxapp/components/HomeRecentMerchantCard.vue wxapp/components/HomeRecentMerchantCard.test.mjs wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs
git commit -m "feat: 首页展示新入驻商家"
```

### Task 4: 全量验证与范围检查

**Files:**
- Verify only

**Interfaces:**
- Verifies: 迁移、API、Go 后端和小程序完整链路

- [ ] **Step 1: 运行全量检查**

Run:

```bash
make check-backend
npm --prefix wxapp run check
```

Expected: 所有命令 PASS。

- [ ] **Step 2: 检查差异与敏感字段**

Run:

```bash
git diff --check HEAD~3
git diff --stat HEAD~3
rg -n "phone|wechat|contact" wxapp/components/HomeRecentMerchantCard.vue backend/app/internal/logic/discovery/banner_topic_logic.go
git status --short
```

Expected:

- 无空白错误；
- 新接口与卡片不暴露联系方式；
- 只包含设计、迁移、商家模型、discovery 接口和首页展示相关文件；
- 用户原有的后台打包产物改动保持未暂存、未覆盖。
