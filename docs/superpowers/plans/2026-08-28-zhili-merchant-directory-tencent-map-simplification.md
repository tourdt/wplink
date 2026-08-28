# 织里拿货档口腾讯地图收缩实施计划

> **实施要求：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务执行；使用复选框（`- [ ]`）记录进度。

**Goal:** 删除自定义地图、预录档口和绑定审核链路，让织里真实入驻商家通过自身腾讯地图位置自动进入拿货档口目录。

**Architecture:** `merchants` 成为唯一商家位置事实来源，商家资料写入结构化档口字段和经纬度；商家领域直接提供目录、附近商家和位置报告 API。小程序保持列表优先和单商家腾讯地图位置页，后台只处理位置异常，不维护场景或点位。

**Tech Stack:** Go 1.25.12、go-zero 1.7.6、goctl 1.7.5、PostgreSQL 17、uni-app/Vue 3 微信小程序、Vue 3/Element Plus 管理后台、Node.js 22。

**Spec:** `docs/superpowers/specs/2026-08-28-zhili-merchant-directory-tencent-map-simplification-design.md`

## 全局约束

- 项目尚未上线，不保留旧数据库、旧 API、旧页面、旧事件名或旧迁移兼容路径。
- 只运营织里站；接口仍校验 `cityCode=zhili`，不增加多城市交互。
- 一商家只维护一个主档口位置；不引入 `merchant_locations` 多门店模型。
- 地图位置可选；未定位商家仍可发布供需和展示主页。
- 拿货档口列表优先，不增加全局地图或列表/地图切换。
- 不引入 PostGIS、Redis、Elasticsearch、消息队列或腾讯距离矩阵。
- `.api` 是 API 契约唯一来源；必须使用 goctl 1.7.5 和 `backend/app/goctl/` 模板生成代码。
- 后端日志记录安全业务上下文和根错误；小程序只显示友好中文错误，禁止泄露手机号、地址、Token、密钥、SQL 或第三方原始响应。
- 所有生产行为先写失败测试并确认失败原因，再写最小实现；每个任务完成后提交一次。

---

### 任务 1：让固定 goctl 门禁在 CI 和本地可复现

**文件：**

- 修改：`.github/workflows/ci.yml`
- 修改：`backend/scripts/api_codegen.test.mjs`

**接口：**

- 输入：仓库要求的 goctl 版本 `1.7.5`。
- 产出：CI 安装固定版本并执行 API 生成、路由盘点和代码生成测试；`make check-backend` 与 CI 使用同一门禁。

- [ ] **步骤 1：增加失败的 CI 策略测试**

在 `backend/scripts/api_codegen.test.mjs` 增加读取 `.github/workflows/ci.yml` 的测试，要求存在固定安装命令和完整代码生成测试：

```js
test('ci installs pinned goctl and runs the generated api gate', () => {
  const workflow = fs.readFileSync(path.resolve(backendDir, '../.github/workflows/ci.yml'), 'utf8')
  assert.match(workflow, /go install github\.com\/zeromicro\/go-zero\/tools\/goctl@v1\.7\.5/)
  assert.match(workflow, /scripts\/api_route_inventory\.test\.mjs/)
  assert.match(workflow, /scripts\/api_codegen\.test\.mjs/)
  assert.match(workflow, /make check-api-generated/)
})
```

- [ ] **步骤 2：确认策略测试因 CI 未安装 goctl 而失败**

运行：

```bash
cd backend
node --test scripts/api_codegen.test.mjs
```

预期：新增用例 FAIL，提示 CI 缺少 goctl 1.7.5 安装或生成门禁。

- [ ] **步骤 3：在 CI 安装固定 goctl 并统一后端门禁**

在后端 Job 的 Go/Node 安装步骤后增加：

```yaml
      - name: 安装固定 goctl
        run: go install github.com/zeromicro/go-zero/tools/goctl@v1.7.5
      - name: 校验迁移、文档与 API 生成链路
        working-directory: backend
        run: node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs scripts/product_doc_governance.test.mjs
      - name: 校验生成结果
        run: make check-api-generated
```

删除 CI 中覆盖范围更小的旧“校验迁移与 API 契约”步骤，避免两套清单漂移。现有 `Makefile` 的 `check-backend` 已包含相同的五类 Node 门禁和 `check-api-generated`，保持不变。

- [ ] **步骤 4：使用 goctl 1.7.5 运行生成门禁**

运行：

```bash
goctl --version
cd backend
node --test scripts/api_codegen.test.mjs scripts/api_route_inventory.test.mjs
cd ..
make check-api-generated
```

预期：版本输出 `goctl version 1.7.5`，全部测试 PASS。

- [ ] **步骤 5：提交工具链门禁**

```bash
git add .github/workflows/ci.yml backend/scripts/api_codegen.test.mjs
git commit -m "build: 固定地图收缩所需 API 生成门禁"
```

---

### 任务 2：重写冷启动数据库为最终商家位置结构

**文件：**

- 修改：`backend/migrations/000002_core_domain.up.sql`
- 修改：`backend/migrations/000002_core_domain.down.sql`
- 修改：`backend/migrations/000003_seed_zhili.up.sql`
- 修改：`backend/migrations/000003_seed_zhili.down.sql`
- 删除：`backend/migrations/000010_sourcing_map.up.sql`
- 删除：`backend/migrations/000010_sourcing_map.down.sql`
- 删除：`backend/migrations/000011_map_object_bind_requests.up.sql`
- 删除：`backend/migrations/000011_map_object_bind_requests.down.sql`
- 删除：`backend/migrations/000030_sourcing_map_cold_start_core_rules.up.sql`
- 删除：`backend/migrations/000030_sourcing_map_cold_start_core_rules.down.sql`
- 删除：`backend/migrations/000031_map_object_reports.up.sql`
- 删除：`backend/migrations/000031_map_object_reports.down.sql`
- 删除：`backend/migrations/000033_merchant_map_events.up.sql`
- 删除：`backend/migrations/000033_merchant_map_events.down.sql`
- 修改：`backend/scripts/validate_migrations.test.mjs`
- 修改：`backend/scripts/seed_demo_data.sql`
- 修改：`backend/scripts/validate_demo_seed.mjs`
- 生成：`backend/app/internal/model/merchants_model_gen.go`
- 创建：`backend/app/internal/model/merchant_location_reports_model.go`
- 生成：`backend/app/internal/model/merchant_location_reports_model_gen.go`
- 创建：`backend/app/internal/model/merchant_directory_events_model.go`
- 生成：`backend/app/internal/model/merchant_directory_events_model_gen.go`

**接口：**

- 输入：设计文档第 5 节最终结构。
- 产出：`merchants` 结构化位置字段、`merchant_location_reports`、`merchant_directory_events`；干净数据库不包含任何旧地图表。

- [ ] **步骤 1：写数据库结构失败测试**

在 `validate_migrations.test.mjs` 增加断言：

```js
test('cold-start schema stores one public location on merchants without custom map tables', () => {
  const migrations = loadMigrationFiles(migrationsDir)
  const core = migrations.find((item) => item.fileName === '000002_core_domain.up.sql').sql
  const allUpSQL = migrations.filter((item) => item.direction === 'up').map((item) => item.sql).join('\n')
  for (const field of ['market_name', 'building_name', 'floor_no', 'booth_no', 'latitude', 'longitude', 'location_name', 'location_status', 'location_updated_at']) {
    assert.match(core, new RegExp(`\\b${field}\\b`))
  }
  assert.match(core, /CREATE TABLE IF NOT EXISTS merchant_location_reports/)
  assert.match(core, /CREATE TABLE IF NOT EXISTS merchant_directory_events/)
  assert.doesNotMatch(allUpSQL, /CREATE TABLE IF NOT EXISTS map_(scene|object|category|object_bind_request|object_reports)/)
  assert.doesNotMatch(allUpSQL, /CREATE TABLE IF NOT EXISTS merchant_map_events/)
})
```

同时把原来要求 `000030`、`000031`、`000033` 存在的用例改为要求旧结构不存在。`000030` 中与地图无关的“织里暂停招聘发布”规则不能随文件一起丢失：测试改为断言 `000003_seed_zhili.up.sql` 直接把 `job_hiring`、`job_seeking` 写为 `inactive`，且织里站 `enabledTypeCodes`、`primaryCategories` 从初始值起就不包含招聘项。

- [ ] **步骤 2：确认数据库测试因新字段和表缺失而失败**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
```

预期：FAIL，指出 `market_name`、报告表或目录事件表缺失。

- [ ] **步骤 3：修改核心迁移并加入严格约束**

在 `merchants` 定义中用结构化字段替换 `location jsonb`，加入：

```sql
market_name varchar(128),
building_name varchar(64),
floor_no varchar(32),
booth_no varchar(64),
latitude double precision,
longitude double precision,
location_name varchar(128),
location_status varchar(32) NOT NULL DEFAULT 'hidden',
location_updated_at timestamptz,
CONSTRAINT chk_merchants_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
CONSTRAINT chk_merchants_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
CONSTRAINT chk_merchants_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
CONSTRAINT chk_merchants_location_status CHECK (location_status IN ('visible', 'hidden', 'risk_hidden')),
CONSTRAINT chk_merchants_visible_location CHECK (location_status <> 'visible' OR (latitude IS NOT NULL AND longitude IS NOT NULL))
```

创建报告表、目录事件表、部分唯一索引、位置组合索引、`main_categories` GIN 索引和市场/档口 trigram 索引。down 迁移按依赖逆序删除事件表、报告表和核心表。

- [ ] **步骤 4：删除旧地图迁移并改写织里演示数据**

删除列出的五对旧迁移。把 `000030` 中与地图无关的招聘停发最终状态折叠进 `000003_seed_zhili.up.sql`，避免删除地图迁移时意外恢复招聘入口。`000003_seed_zhili.up.sql` 和 `seed_demo_data.sql` 直接为演示商家写入市场、楼栋、楼层、档口号和有效织里坐标，不再插入 scene/object/category/bind 数据。

演示商家示例字段必须使用：

```sql
market_name = '织里童装城',
building_name = '一期',
floor_no = '1F',
booth_no = 'A108',
latitude = 30.8732,
longitude = 120.2255,
location_name = '织里童装城',
location_status = 'visible'
```

- [ ] **步骤 5：从最终表结构重新生成 goctl Model**

先把最终迁移应用到可丢弃开发库，并明确确认 DSN 非空：

```bash
cd backend
test -n "$WPLINK_MODEL_DATABASE_URL"
WPLINK_MODEL_TMP="$(mktemp -d /private/tmp/wplink-directory-model.XXXXXX)"
goctl model pg datasource --url "$WPLINK_MODEL_DATABASE_URL" --table merchants --dir "$WPLINK_MODEL_TMP" --style go_zero
goctl model pg datasource --url "$WPLINK_MODEL_DATABASE_URL" --table merchant_location_reports --dir "$WPLINK_MODEL_TMP" --style go_zero
goctl model pg datasource --url "$WPLINK_MODEL_DATABASE_URL" --table merchant_directory_events --dir "$WPLINK_MODEL_TMP" --style go_zero
```

把三个 `*_model_gen.go` 生成文件同步到 `app/internal/model/`；`merchants_model.go` 保留现有 `withSession` 扩展，两个新表的非 `_gen.go` 包装文件采用 goctl 生成内容。确认文件头均为 `Code generated by goctl`，且 `merchants_model_gen.go` 不再包含 `Location` JSON 字段。

- [ ] **步骤 6：验证静态迁移和演示数据**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
node scripts/validate_demo_seed.mjs
```

预期：全部 PASS，且输出中无旧地图表要求。

- [ ] **步骤 7：提交最终冷启动结构**

```bash
git add backend/migrations backend/scripts/validate_migrations.test.mjs backend/scripts/seed_demo_data.sql backend/scripts/validate_demo_seed.mjs backend/app/internal/model/merchants_model_gen.go backend/app/internal/model/merchant_location_reports_model* backend/app/internal/model/merchant_directory_events_model*
git commit -m "refactor: 以商家位置替代自定义地图表"
```

---

### 任务 3：实现商家位置设置和清除写入链路

**文件：**

- 修改：`backend/app/api/merchant.api`
- 修改：`backend/app/internal/model/merchant_model.go`
- 修改：`backend/app/internal/model/merchant_model_test.go`
- 修改：`backend/app/internal/logic/merchant/update_merchant_logic.go`
- 修改：`backend/app/internal/logic/merchant/update_merchant_logic_test.go`
- 修改：`backend/app/internal/logic/merchant/get_merchant_logic.go`
- 修改：`backend/app/internal/logic/merchant/get_merchant_logic_test.go`
- 修改：`backend/app/internal/handler/merchant/get_merchant_handler.go`
- 修改：`backend/app/internal/handler/merchant/update_merchant_handler.go`
- 生成：`backend/app/internal/types/types.go`
- 生成：`backend/app/internal/handler/routes.go`

**接口：**

- 输入：`UpdateMerchantReq.LocationAction` 为 `set` 或 `clear`。
- 产出：`model.MerchantLocationPatch`、结构化 `MerchantLocationInfo` 和后端强校验。

- [ ] **步骤 1：写位置参数失败测试**

在 `update_merchant_logic_test.go` 增加：

```go
func TestUpdateMerchantSetsVisibleStructuredLocation(t *testing.T) {
    store := &fakeMerchantUpdateStore{updatedAt: "2026-08-28T10:00:00+08:00"}
    logic := NewUpdateMerchantLogic(store)
    _, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
        Name: "织里晨星童装", LocationAction: "set", MarketName: "织里童装城",
        BuildingName: "一期", FloorNo: "1F", BoothNo: "A108",
        Latitude: float64Ptr(30.8732), Longitude: float64Ptr(120.2255), LocationName: "织里童装城",
    })
    if err != nil { t.Fatalf("UpdateMerchant() error = %v", err) }
    if store.patch.Location.Status != model.MerchantLocationStatusVisible || store.patch.Location.BoothNo != "A108" {
        t.Fatalf("location patch = %#v", store.patch.Location)
    }
}

func TestUpdateMerchantRejectsHalfLocation(t *testing.T) {
    logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})
    _, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
        Name: "织里晨星童装", LocationAction: "set", Latitude: float64Ptr(30.8732),
    })
    if errx.CodeOf(err) != errx.CodeValidationFailed { t.Fatalf("error = %v", err) }
}

func float64Ptr(value float64) *float64 { return &value }
```

另加 clear 用例，要求经纬度和档口字段清空、状态为 `hidden`。为 `GetMerchant` 增加公开访问和商家本人访问两组用例：公开访问 `hidden/risk_hidden` 位置时必须返回 `Available=false` 且经纬度为零；通过 `UserCanManageMerchant` 确认是商家本人时可返回保存在数据库中的风险位置，供资料页核对和重新选点。

- [ ] **步骤 2：确认新增 Logic 测试失败**

运行：

```bash
cd backend
go test ./app/internal/logic/merchant -run 'TestUpdateMerchant(SetsVisibleStructuredLocation|RejectsHalfLocation|ClearsLocation)' -count=1
```

预期：编译或断言 FAIL，因为结构化位置类型尚不存在。

- [ ] **步骤 3：定义商家位置类型和校验**

在 Model 定义：

```go
const (
    MerchantLocationStatusVisible    = "visible"
    MerchantLocationStatusHidden     = "hidden"
    MerchantLocationStatusRiskHidden = "risk_hidden"
)

type MerchantLocationPatch struct {
    Action, MarketName, BuildingName, FloorNo, BoothNo, LocationName, Status string
    Latitude, Longitude float64
}
```

Logic 对 `set` 校验有限数值和经纬度范围，对 `clear` 忽略请求中的坐标并构造空位置，对其他动作返回“地图位置操作不正确，请重新选择”。数据库错误记录 `merchantId`、`locationAction` 和根错误，对外返回“商家资料保存失败，请稍后重试”。

`UpdateMerchantReq` 的 Logic 层经纬度使用 `*float64`，区分字段缺失和合法数值；`update_merchant_handler.go` 使用仅服务该路由的指针型请求 Payload 解析 JSON，再映射到 Logic。`.api` 仍声明 optional 数值字段作为外部契约，不能因为 goctl 生成的 `float64` 零值而把缺失字段误判为 `(0,0)`。

- [ ] **步骤 4：更新 SQL 和读取 DTO**

`updateMerchantSQL` 使用 `locationAction` 分支设置或清空全部位置列。`GetMerchant` 返回结构化位置：

```go
type MerchantLocationInfo struct {
    MarketName, BuildingName, FloorNo, BoothNo, LocationName, Status, UpdatedAt string
    Latitude, Longitude *float64
    Available bool
}
```

位置不存在时 `Available=false`。Go 数值字段无法表达 `null`，因此响应中的经纬度使用 `*float64` 并在 `.api` 中标记 optional；Handler 外包一层带 `omitempty` 的位置 Payload，确保不可见位置完全省略经纬度，而不是把 `(0,0)` 伪装成位置。

- [ ] **步骤 5：修改 `.api` 并重新生成**

把 `Location map[string]interface{}` 替换为结构化 `MerchantLocationInfo`，为更新请求增加精确字段，然后运行：

```bash
make generate-api
make check-api-generated
```

检查生成 diff 只包含预期 types/routes/handler 骨架变化，完成 `get_merchant_handler.go`、`update_merchant_handler.go` 的字段映射并清除所有 `WPLINK_API_HANDLER_STUB`。

- [ ] **步骤 6：验证商家位置写入链路**

运行：

```bash
cd backend
go test ./app/internal/logic/merchant ./app/internal/model ./app/internal/handler/merchant -count=1
cd ..
make check-api-generated
```

预期：全部 PASS。

- [ ] **步骤 7：提交商家位置写入**

```bash
git add backend/app/api/merchant.api backend/app/internal/model/merchant_model.go backend/app/internal/model/merchant_model_test.go backend/app/internal/logic/merchant backend/app/internal/handler/merchant backend/app/internal/types/types.go backend/app/internal/handler/routes.go
git commit -m "feat: 支持商家自助维护腾讯地图位置"
```

---

### 任务 4：实现拿货档口目录和附近商家读取链路

**文件：**

- 创建：`backend/app/internal/model/merchant_directory_model.go`
- 创建：`backend/app/internal/model/merchant_directory_model_test.go`
- 创建：`backend/app/internal/logic/merchant/directory_logic.go`
- 创建：`backend/app/internal/logic/merchant/directory_logic_test.go`
- 修改：`backend/app/api/merchant.api`
- 修改：`backend/app/internal/svc/service_context.go`
- 生成：`backend/app/internal/handler/merchant/list_merchant_directory_handler.go`
- 生成：`backend/app/internal/handler/merchant/get_merchant_location_context_handler.go`
- 生成：`backend/app/internal/types/types.go`
- 生成：`backend/app/internal/handler/routes.go`

**接口：**

- 产出：`ListMerchantDirectory(ctx, filter)`、`CountMerchantDirectory(ctx, filter)`、`GetVisibleMerchantLocation(ctx, merchantID)`、`ListNearbyVisibleMerchants(ctx, origin, 1000, 20)`。
- API：`GET /api/v1/merchant-directory`、`GET /api/v1/merchants/:merchantId/location-context`。

- [ ] **步骤 1：写目录 SQL 和距离失败测试**

在新 Model 测试中断言目录只读取真实商家：

```go
func TestBuildMerchantDirectoryWhereOnlyReturnsVisibleCompletedMerchants(t *testing.T) {
    where, args := buildMerchantDirectoryWhere(MerchantDirectoryFilter{CityCode: "zhili", Keyword: "A108", Categories: []string{"童装"}})
    for _, token := range []string{"m.status = 'active'", "m.profile_status = 'completed'", "m.location_status = 'visible'", "m.latitude IS NOT NULL", "m.longitude IS NOT NULL", "m.booth_no", "m.main_categories ?|"} {
        if !strings.Contains(where, token) { t.Fatalf("where = %q, missing %q", where, token) }
    }
    if len(args) != 3 { t.Fatalf("args = %#v", args) }
}
```

增加距离排序用例，要求 1000 米外商家被过滤、结果最多 20 家且按距离升序。

- [ ] **步骤 2：确认 Model 测试失败**

运行：

```bash
cd backend
go test ./app/internal/model -run 'Test(BuildMerchantDirectoryWhere|RankNearbyVisibleMerchants)' -count=1
```

预期：FAIL，因为新 Model 函数尚不存在。

- [ ] **步骤 3：实现聚焦的目录 Model**

在 `merchant_directory_model.go` 定义：

```go
type MerchantDirectoryFilter struct { CityCode, Keyword string; Categories []string; Page, PageSize int64 }
type MerchantDirectoryItem struct { MerchantID, MerchantNo, Name, MerchantType, LogoURL, AddressText, MarketName, BuildingName, FloorNo, BoothNo, LocationName string; MainCategories []string; Latitude, Longitude float64; LastResourceActiveAt time.Time; DistanceMeters float64 }
```

目录 SQL 直接查询 `merchants`，LEFT JOIN 每个商家最近已发布资源时间；关键词排序使用 CASE，分页使用相同 ORDER BY。附近查询先使用经纬度外接矩形，再用纯函数 Haversine 精排。

- [ ] **步骤 4：写 Logic 失败测试并实现友好错误**

测试默认分页、100 上限、空数组标准化、当前位置不存在、附近查询失败保留当前商家：

```go
func TestGetMerchantLocationContextKeepsCurrentWhenNearbyFails(t *testing.T) {
    store := &fakeDirectoryStore{current: visibleMerchant("merchant-1"), nearbyErr: errors.New("db unavailable")}
    resp, err := NewMerchantDirectoryLogic(store).GetLocationContext(context.Background(), "merchant-1")
    if err != nil || resp.Current.MerchantID != "merchant-1" || resp.NearbyAvailable { t.Fatalf("resp=%#v err=%v", resp, err) }
}
```

数据库错误日志包含操作、分页或商家 ID，响应只返回“商家列表加载失败，请稍后重试”或“该商家位置待完善”。

- [ ] **步骤 5：增加 API、生成 Handler 并接线**

在 `merchant.api` 定义目录、位置响应和两个路由，运行：

```bash
make generate-api
```

实现两个新 Handler 的参数解析和字段映射；公开接口不得返回联系方式。更新 `ServiceContext` 使 `APIStore` 满足新 Logic 接口。

- [ ] **步骤 6：验证目录后端**

运行：

```bash
cd backend
go test ./app/internal/model ./app/internal/logic/merchant ./app/internal/handler/merchant ./app/internal/server -count=1
cd ..
make check-api-generated
```

预期：全部 PASS，契约和运行时出现两个新路由。

- [ ] **步骤 7：提交目录读取链路**

```bash
git add backend/app/api/merchant.api backend/app/internal/model/merchant_directory_model* backend/app/internal/logic/merchant/directory_logic* backend/app/internal/handler/merchant backend/app/internal/svc/service_context.go backend/app/internal/types/types.go backend/app/internal/handler/routes.go
git commit -m "feat: 从商家位置提供拿货档口目录"
```

---

### 任务 5：实现位置报告、后台治理和目录行为统计

**文件：**

- 创建：`backend/app/internal/model/merchant_location_report_model.go`
- 创建：`backend/app/internal/model/merchant_location_report_model_test.go`
- 创建：`backend/app/internal/logic/merchant/location_report_logic.go`
- 创建：`backend/app/internal/logic/merchant/location_report_logic_test.go`
- 修改：`backend/app/internal/logic/admin/merchant_admin_logic.go`
- 修改：`backend/app/internal/logic/admin/merchant_admin_logic_test.go`
- 修改：`backend/app/internal/logic/merchant/update_merchant_logic.go`
- 修改：`backend/app/internal/logic/merchant/update_merchant_logic_test.go`
- 修改：`backend/app/api/merchant.api`
- 修改：`backend/app/api/admin.api`
- 修改：`backend/app/api/metrics.api`
- 创建：`backend/app/internal/model/merchant_directory_event_model.go`
- 创建：`backend/app/internal/model/merchant_directory_event_model_test.go`
- 修改：`backend/app/internal/model/user_model.go`
- 修改：`backend/app/internal/model/user_model_test.go`
- 重命名：`backend/app/internal/logic/metrics/record_merchant_map_event_logic.go` → `backend/app/internal/logic/metrics/record_merchant_directory_event_logic.go`
- 重命名：`backend/app/internal/logic/metrics/record_merchant_map_event_logic_test.go` → `backend/app/internal/logic/metrics/record_merchant_directory_event_logic_test.go`
- 重命名：`backend/app/internal/server/merchant_map_event_limiter.go` → `backend/app/internal/server/merchant_directory_event_limiter.go`
- 重命名：`backend/app/internal/server/merchant_map_event_limiter_test.go` → `backend/app/internal/server/merchant_directory_event_limiter_test.go`
- 重命名：`backend/app/internal/task/merchant_map_event_cleanup_task.go` → `backend/app/internal/task/merchant_directory_event_cleanup_task.go`
- 重命名：`backend/app/internal/task/merchant_map_event_cleanup_scheduler.go` → `backend/app/internal/task/merchant_directory_event_cleanup_scheduler.go`
- 重命名：`backend/app/internal/task/merchant_map_event_cleanup_test.go` → `backend/app/internal/task/merchant_directory_event_cleanup_test.go`
- 修改：`backend/app/internal/task/coordinator.go`
- 修改：`backend/app/internal/task/coordinator_test.go`
- 修改：`backend/app/internal/task/scheduler_runtime_test.go`
- 修改：`backend/app/internal/config/config.go`
- 修改：`backend/app/internal/config/load.go`
- 修改：`backend/app/internal/config/load_test.go`
- 修改：`backend/app/internal/config/production_validation.go`
- 修改：`backend/app/internal/config/production_validation_test.go`
- 修改：`backend/app/internal/svc/service_context.go`
- 修改：`backend/app/internal/svc/service_context_test.go`
- 修改：`backend/app/app.go`
- 修改：`backend/app/internal/server/auth_api_test.go`
- 修改：`backend/app/internal/server/favorite_api_test.go`
- 修改：`backend/app/internal/server/resource_api_test.go`
- 修改：`backend/etc/app.yaml.example`
- 修改：`backend/etc/app.production.yaml.example`
- 修改：`backend/app/internal/permission/admin.go`
- 修改：`backend/app/internal/handler/adminmerchant/handlers.go`
- 删除：`backend/app/internal/handler/metrics/record_merchant_map_event_handler.go`
- 重命名：`backend/app/internal/handler/metrics/record_merchant_map_event_handler_test.go` → `backend/app/internal/handler/metrics/record_merchant_directory_event_handler_test.go`
- 生成：`backend/app/internal/handler/merchant/create_merchant_location_report_handler.go`
- 生成：`backend/app/internal/handler/adminmerchant/admin_list_merchant_location_reports_handler.go`
- 生成：`backend/app/internal/handler/adminmerchant/admin_review_merchant_location_report_handler.go`
- 生成：`backend/app/internal/handler/adminmerchant/admin_update_merchant_location_status_handler.go`
- 生成：`backend/app/internal/handler/metrics/record_merchant_directory_event_handler.go`
- 生成：`backend/app/internal/types/types.go`、`backend/app/internal/handler/routes.go`

**接口：**

- API：`POST /api/v1/merchants/:merchantId/location-reports` 提交位置报告；后台报告列表/处理、后台位置状态操作、目录事件上报。
- 事务：第三个不同用户的 pending 报告把位置改为 `risk_hidden`；重新选点结案 pending 报告。

- [ ] **步骤 1：写三人报告自动隐藏失败测试**

使用 sqlmock 写 `TestCreateMerchantLocationReportHidesLocationAtThirdReporter`，严格验证 `ExpectBegin` → `SELECT merchants ... FOR UPDATE` 返回 visible 商家 → 查询同用户 pending 报告返回 `sql.ErrNoRows` → `INSERT merchant_location_reports` → `COUNT(DISTINCT reporter_user_id)` 返回 3 → `UPDATE merchants SET location_status='risk_hidden'` → `ExpectCommit`。断言结果 `AutoHidden=true`、`ActiveReporterCount=3`；数据库错误分支必须 `ExpectRollback`。

增加同一用户重复 pending 报告的幂等测试：第二次请求返回原报告 ID 和当前状态，不插入新行、不增加有效报告人数，也不返回冲突。再增加商家重新选点时把全部 pending 报告更新为 `resolved` 的事务测试。

- [ ] **步骤 2：确认报告 Model 测试失败**

运行：

```bash
cd backend
go test ./app/internal/model -run 'Test(CreateMerchantLocationReport|ResolvePendingLocationReports)' -count=1
```

预期：FAIL，因为报告 Model 尚不存在。

- [ ] **步骤 3：实现报告事务和安全日志**

Model 使用 `SELECT ... FOR UPDATE` 保证报告计数和状态更新一致；插入使用部分唯一索引配合事务内已存在报告查询实现幂等。Logic 只接受受控原因 `navigation_inaccurate`、`wrong_market`、`wrong_booth`、`other`，说明最多 200 字。

第三人触发时输出：

```go
logx.WithContext(ctx).Infow("商家位置因多人反馈已自动隐藏",
    logx.Field("merchantId", merchantID),
    logx.Field("activeReporterCount", result.ActiveReporterCount),
    logx.Field("oldStatus", model.MerchantLocationStatusVisible),
    logx.Field("newStatus", model.MerchantLocationStatusRiskHidden))
```

扩展 `MerchantModel.UpdateMerchant` 的既有事务：当 `LocationAction=set` 成功写入新位置时，在同一事务把该商家的 pending 报告改为 `resolved`，保留原因、快照和审计字段；`clear` 不自动结案。`update_merchant_logic_test.go` 必须覆盖重新选点后恢复 `visible`，且 Model sqlmock 测试确认更新位置与结案报告同一提交。

- [ ] **步骤 4：实现后台位置治理**

在 admin merchant API 增加：

```text
GET  /api/v1/admin/merchants/location-reports
POST /api/v1/admin/merchants/location-reports/:reportId/review
POST /api/v1/admin/merchants/:merchantId/location-status
```

所有路由归属 `merchants` 权限模块。人工恢复只允许坐标有效的商家；人工隐藏/恢复和报告审核与 `operation_logs` 同事务写入。

- [ ] **步骤 5：把地图事件改成目录事件**

将契约改为：

```text
POST /api/v1/metrics/merchant-directory-events
```

类型改名为 `MerchantDirectoryEventReq/Resp`，Model 使用 `merchant_directory_events`。允许事件严格限定为 `directory_exposure`、`directory_merchant_click`、`location_view`、`nearby_marker_click`、`nearby_list_item_click`、`nearby_merchant_click`、`nearby_drawer_open`、`navigation_click`、`location_report_submit`。同步重命名 Handler、限流器、配置键、清理任务、调度器和测试，不保留旧事件路由、旧 Go 符号或旧 YAML 配置键。清理任务保留固定 Advisory Lock 号 `1004`，只重命名领域标识，不改变多实例协调协议。

- [ ] **步骤 6：生成 API 并验证权限、幂等和事件**

运行：

```bash
make generate-api
cd backend
go test ./app/internal/model ./app/internal/logic/merchant ./app/internal/logic/admin ./app/internal/logic/metrics ./app/internal/handler/merchant ./app/internal/handler/adminmerchant ./app/internal/handler/metrics ./app/internal/server ./app/internal/task ./app/internal/config ./app/internal/svc -count=1
cd ..
make check-api-generated
```

预期：全部 PASS，无 Handler 骨架标记。

- [ ] **步骤 7：提交位置治理和统计**

```bash
git add -A backend/app backend/etc
git commit -m "feat: 增加商家位置异常自动治理"
```

---

### 任务 6：删除旧自定义地图后端和权限模块

**文件：**

- 删除：`backend/app/api/map.api`
- 修改：`backend/app/api/app.api`
- 删除：`backend/app/internal/logic/map/`
- 删除：`backend/app/internal/model/map_model.go`
- 删除：`backend/app/internal/model/map_model_test.go`
- 删除：`backend/app/internal/model/merchant_map_events_model.go`
- 删除：`backend/app/internal/model/merchant_map_events_model_gen.go`
- 删除：`backend/app/internal/model/merchant_map_events_model_test.go`
- 删除：`backend/app/internal/handler/map/`
- 修改：`backend/app/internal/svc/service_context.go`
- 修改：`backend/app/internal/svc/service_context_test.go`
- 修改：`backend/app/internal/svc/service_context_contract_test.go`
- 修改：`backend/app/internal/permission/admin.go`
- 修改：`backend/app/internal/permission/admin_test.go`
- 修改：`backend/scripts/api_contract.test.mjs`
- 修改：`backend/scripts/api_route_inventory.test.mjs`
- 生成：`backend/app/internal/handler/routes.go`
- 生成：`backend/app/internal/types/types.go`

**接口：**

- 输入：任务 3～5 已提供全部保留能力。
- 产出：运行时不再存在 scene/object/category/POI/prelisted/bind 路由或 `sourcing_map` 后台权限模块。

- [ ] **步骤 1：先写旧路由必须消失的失败测试**

在契约测试增加：

```js
test('production contract contains only merchant-owned location APIs', () => {
  const paths = parseAPIContracts(apiDir).map((route) => route.path)
  for (const fragment of ['/map/scenes', '/map/objects', '/map/categories', '/map/bind-candidates', '/map-binding']) {
    assert.equal(paths.some((path) => path.includes(fragment)), false, `retired route remains: ${fragment}`)
  }
  assert.equal(paths.includes('/api/v1/merchant-directory'), true)
  assert.equal(paths.includes('/api/v1/merchants/:merchantId/location-context'), true)
})
```

- [ ] **步骤 2：确认测试因旧 map.api 仍被导入而失败**

运行：

```bash
cd backend
node --test scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs
```

预期：FAIL 并列出旧地图路由。

- [ ] **步骤 3：删除旧地图契约、实现和依赖装配**

从 `app.api` 移除 map import，删除列出的旧目录和文件。移除 ServiceContext 中只服务 `map_scene/map_object` 的依赖；保留任务 3～5 的商家目录 Model、腾讯逆地理编码和目录事件清理。

从后台权限常量、默认模块集合和路径映射删除 `sourcing_map`；新的位置治理接口全部映射到 `merchants`。

- [ ] **步骤 4：重新生成并做残留扫描**

运行：

```bash
make generate-api
rg -n 'map_scene|map_object|map-binding|bind-candidates|sourcing_map|MerchantMapEvent|merchant_map_event' backend/app backend/common backend/scripts backend/etc --glob '!**/*historical*'
```

预期：除明确的历史文档测试字符串外无生产引用；若扫描命中生产文件，逐个删除对应旧依赖后重跑。

- [ ] **步骤 5：验证后端契约和编译**

运行：

```bash
cd backend
node --test scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs
go test ./app/internal/... ./common/... -count=1
go vet ./...
cd ..
make check-api-generated
```

预期：全部 PASS。

- [ ] **步骤 6：提交旧地图后端删除**

```bash
git add -A backend/app backend/common backend/scripts
git commit -m "refactor: 删除自定义地图和档口绑定后端"
```

---

### 任务 7：改造小程序商家资料位置表单

**文件：**

- 创建：`wxapp/common/merchantLocationForm.js`
- 创建：`wxapp/common/merchantLocationForm.test.mjs`
- 修改：`wxapp/pages/merchant/profile.vue`
- 修改：`wxapp/api/merchant.js`
- 修改：`wxapp/pages/merchant/detail.vue`
- 修改：`wxapp/pages/merchant/detail.test.mjs`

**接口：**

- 产出：`normalizeMerchantLocationForm(detail)`、`buildMerchantLocationPayload(form)`、`clearMerchantLocationForm(form)`。
- 请求：资料保存始终发送 `locationAction=set|clear` 和结构化字段。

- [ ] **步骤 1：写表单纯逻辑失败测试**

```js
test('buildMerchantLocationPayload sets a complete selected location', () => {
  assert.deepEqual(buildMerchantLocationPayload({
    marketName: '织里童装城', buildingName: '一期', floorNo: '1F', boothNo: 'A108',
    latitude: 30.8732, longitude: 120.2255, locationName: '织里童装城',
  }), {
    locationAction: 'set', marketName: '织里童装城', buildingName: '一期', floorNo: '1F', boothNo: 'A108',
    latitude: 30.8732, longitude: 120.2255, locationName: '织里童装城',
  })
})

test('buildMerchantLocationPayload clears incomplete coordinates', () => {
  assert.deepEqual(buildMerchantLocationPayload({ latitude: 30.8732 }), {
    locationAction: 'clear', marketName: '', buildingName: '', floorNo: '', boothNo: '', latitude: 0, longitude: 0, locationName: '',
  })
})
```

- [ ] **步骤 2：确认纯逻辑测试失败**

运行：

```bash
cd wxapp
node --test common/merchantLocationForm.test.mjs
```

预期：FAIL，因为模块不存在。

- [ ] **步骤 3：实现纯逻辑并接入资料页**

资料页增加市场、楼栋、楼层、档口号输入；`chooseLocation` 写经纬度和 `locationName`。地址文本输入或“清除地图位置”调用统一 clear helper，保存时合并 `buildMerchantLocationPayload(form)`。

保留现有腾讯逆地理编码失败提示、请求互斥、图片上传和联系方式逻辑。新增中文注释说明地址修改为什么必须清除旧坐标。

- [ ] **步骤 4：更新商家详情位置入口**

详情页只在 `detail.location.status === 'visible' && detail.location.available` 时显示“查看位置”，不再读取任意 JSON 键。

- [ ] **步骤 5：验证商家资料和详情**

运行：

```bash
cd wxapp
node --test common/merchantLocationForm.test.mjs pages/merchant/detail.test.mjs
npm run validate:flows
```

预期：全部 PASS。

- [ ] **步骤 6：提交小程序商家位置表单**

```bash
git add wxapp/common/merchantLocationForm* wxapp/pages/merchant/profile.vue wxapp/pages/merchant/detail.vue wxapp/pages/merchant/detail.test.mjs wxapp/api/merchant.js
git commit -m "feat(wxapp): 商家自助维护结构化档口位置"
```

---

### 任务 8：改造拿货档口和位置页并删除画布、认领代码

**文件：**

- 创建：`wxapp/api/merchantDirectory.js`
- 创建：`wxapp/common/merchantDirectoryAnalytics.js`
- 创建：`wxapp/common/merchantDirectoryAnalytics.test.mjs`
- 修改：`wxapp/api/metrics.js`
- 修改：`wxapp/api/request.test.mjs`
- 修改：`wxapp/request-loading-contract.test.mjs`
- 创建：`wxapp/pages/sourcing-map/merchantDirectoryState.js`
- 创建：`wxapp/pages/sourcing-map/merchantDirectoryState.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
- 修改：`wxapp/components/MerchantPlaceCard.vue`
- 修改：`wxapp/pages/merchant/location.vue`
- 修改：`wxapp/pages/merchant/locationState.js`
- 修改：`wxapp/pages/merchant/location.test.mjs`
- 修改：`wxapp/pages/merchant/locationState.test.mjs`
- 修改：`wxapp/pages.json`
- 修改：`wxapp/scripts/validate-pages.mjs`
- 修改：`wxapp/scripts/validate-flows.mjs`
- 删除：`wxapp/api/sourcingMap.js`
- 删除：`wxapp/common/merchantMapAnalytics.js`
- 删除：`wxapp/common/merchantMapAnalytics.test.mjs`
- 删除：`wxapp/pages/merchant/map-binding.vue`
- 删除：`wxapp/pages/merchant/map-binding.test.mjs`
- 删除：`wxapp/pages/sourcing-map/legacy-canvas.vue`
- 删除：`wxapp/pages/sourcing-map/canvasRenderer.js`
- 删除：`wxapp/pages/sourcing-map/canvasRenderer.test.mjs`
- 删除：`wxapp/pages/sourcing-map/mapGeometry.js`
- 删除：`wxapp/pages/sourcing-map/mapGeometry.test.mjs`
- 删除：`wxapp/pages/sourcing-map/mapGesture.js`
- 删除：`wxapp/pages/sourcing-map/mapGesture.test.mjs`
- 删除：`wxapp/pages/sourcing-map/mapHitTest.js`
- 删除：`wxapp/pages/sourcing-map/mapHitTest.test.mjs`
- 删除：`wxapp/pages/sourcing-map/mapObjectState.js`
- 删除：`wxapp/pages/sourcing-map/mapObjectState.test.mjs`
- 删除：`wxapp/pages/sourcing-map/merchantPlaceState.js`
- 删除：`wxapp/pages/sourcing-map/merchantPlaceState.test.mjs`
- 删除：`wxapp/pages/sourcing-map/tencentMapState.js`
- 删除：`wxapp/pages/sourcing-map/tencentMapState.test.mjs`
- 删除：`wxapp/static/map/marker-claimed.png`
- 删除：`wxapp/static/map/marker-prelisted.png`

**接口：**

- 目录 API：`listMerchantDirectory`、`getMerchantLocationContext`、`submitMerchantLocationReport`。
- 统计 API：`trackMerchantDirectoryEvent` 上报 `/metrics/merchant-directory-events`。

- [ ] **步骤 1：写目录状态失败测试**

```js
test('merchant directory query contains only city keyword categories and pagination', () => {
  assert.deepEqual(buildMerchantDirectoryQuery({ cityCode: 'zhili', keyword: ' A108 ', categories: ['童装'], page: 2, pageSize: 20 }), {
    cityCode: 'zhili', keyword: 'A108', categories: '童装', page: 2, pageSize: 20,
  })
})

test('directory items never expose prelisted or claim state', () => {
  const item = normalizeMerchantDirectoryItem({ merchantId: '1', name: '晨星童装', boothNo: 'A108', latitude: 30.8732, longitude: 120.2255 })
  assert.equal('claimed' in item, false)
  assert.equal('objectId' in item, false)
})
```

- [ ] **步骤 2：确认目录状态测试失败**

运行：

```bash
cd wxapp
node --test pages/sourcing-map/merchantDirectoryState.test.mjs
```

预期：FAIL，因为新模块不存在。

- [ ] **步骤 3：实现新 API、状态和统计模块**

API 只调用三个新商家路由；`api/metrics.js` 将方法和 URL 改为 `recordMerchantDirectoryEvent` 与 `/api/v1/metrics/merchant-directory-events`。目录统计保留有界 `visitorKey/sessionId`、静默失败和不清理可选登录态的既有语义，但只允许设计文档中的九个新事件名。

- [ ] **步骤 4：收缩拿货档口列表**

删除 source filter、待认领、认领弹窗和绑定跳转。原地图分类接口随旧后端删除；为避免两人团队维护第二套分类配置，主营品类筛选改为可清空的文字输入，按逗号拆分后直接匹配商家 `mainCategories`。列表卡片只使用 `merchantId` 作为身份，展示市场/楼栋/楼层/档口号、主营品类，并提供“进入主页”“查看位置”。目录结果首次落地时对本页未记录的商家上报 `directory_exposure`，进入主页或位置统一上报 `directory_merchant_click`。

导航反馈本地存储改为 `{ merchantId, name }`。目录页和位置页从 `onShow` 共用同一份一次性反馈状态，返回页面后由登录用户调用商家位置报告接口；成功后上报 `location_report_submit`，重复 pending 报告按后端幂等结果显示“已收到你的反馈”。

- [ ] **步骤 5：调整位置页和 Marker**

Marker 只区分当前商家、附近商家和选中商家，不再区分 claimed/prelisted。位置报告使用当前 `merchantId`；附近商家仍保持 1000 米、最多 20 家、失败不阻断当前导航。`locationState.js` 直接规范化新目录 DTO，不再导入待删除的 `merchantPlaceState.js`。

- [ ] **步骤 6：删除旧页面和画布模块**

删除列出的文件，从 `pages.json` 和页面校验脚本移除 map-binding。删除 `marker-claimed.png`、`marker-prelisted.png`，保留位置页使用的 `marker-selected.png`、`marker-nearby.png`、`marker-nearby-selected.png`。

- [ ] **步骤 7：验证小程序目录与位置流程**

运行：

```bash
cd wxapp
npm run validate:pages
npm run validate:flows
node --test common/merchantDirectoryAnalytics.test.mjs pages/sourcing-map/index.test.mjs pages/sourcing-map/merchantDirectoryState.test.mjs pages/merchant/location.test.mjs pages/merchant/locationState.test.mjs
rg -n '待认领|这是我的档口|map-binding|legacy-canvas|objectId|map_scene|map_object' pages components api common scripts
```

预期：测试全部 PASS，残留扫描不命中生产代码。

- [ ] **步骤 8：提交小程序目录收缩**

```bash
git add -A wxapp
git commit -m "refactor(wxapp): 收缩拿货档口为真实商家目录"
```

---

### 任务 9：把位置治理合并到后台商家管理

**文件：**

- 修改：`admin-web/src/views/MerchantView.vue`
- 修改：`admin-web/src/api/merchant.js`
- 修改：`admin-web/src/router/index.js`
- 修改：`admin-web/src/layouts/AdminLayout.vue`
- 修改：`admin-web/src/views/BannerTopicView.vue`
- 修改：`admin-web/scripts/feature-visibility.test.mjs`
- 删除：`admin-web/src/views/SourcingMapView.vue`
- 删除：`admin-web/src/api/sourcingMap.js`
- 删除：`admin-web/src/common/mapViewport.js`
- 删除：`admin-web/src/common/mapViewport.test.mjs`

**接口：**

- 输入：任务 5 的后台商家位置接口。
- 产出：商家页位置状态筛选、位置摘要、报告审核、隐藏和恢复操作；无独立地图菜单。

- [ ] **步骤 1：写后台 UI 失败测试**

在现有后台源代码契约测试中增加：

```js
test('admin merges location governance into merchants and removes custom map editor', () => {
  const router = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layout = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const merchant = fs.readFileSync(path.join(root, 'src/views/MerchantView.vue'), 'utf8')
  assert.doesNotMatch(router, /SourcingMapView|sourcing-map/)
  assert.doesNotMatch(layout, /拿货地图|sourcing_map/)
  assert.match(merchant, /位置状态/)
  assert.match(merchant, /位置错误反馈/)
  assert.match(merchant, /隐藏位置/)
  assert.match(merchant, /恢复位置/)
})
```

删除该测试文件中所有读取 `SourcingMapView.vue` 或断言旧地图编辑功能存在的用例；保留小程序内部跳转路径 `/pages/sourcing-map/index`，但把运营位标签文案从“拿货地图”改为“拿货档口”。

- [ ] **步骤 2：确认后台测试失败**

运行：

```bash
cd admin-web
npm test
```

预期：新增用例 FAIL，因为旧地图路由仍存在且商家页缺少治理操作。

- [ ] **步骤 3：扩展后台 API 和商家页**

`merchant.js` 增加：

```js
export const listMerchantLocationReports = (params = {}) => http.get('/api/v1/admin/merchants/location-reports', { params })
export const reviewMerchantLocationReport = (reportId, payload) => http.post(`/api/v1/admin/merchants/location-reports/${reportId}/review`, payload)
export const updateMerchantLocationStatus = (merchantId, payload) => http.post(`/api/v1/admin/merchants/${merchantId}/location-status`, payload)
```

商家页增加位置状态筛选、市场/档口/状态列、详情中的位置摘要、报告 Drawer 和确认弹窗。按钮期间禁用重复请求；失败显示后端友好消息或“位置状态更新失败，请重试”。

- [ ] **步骤 4：删除独立地图后台**

删除 SourcingMapView、API、画布公共逻辑和路由菜单，移除不再使用的 `MapLocation` 图标导入。后台权限页面不再显示 `sourcing_map` 模块。`BannerTopicView.vue` 只保留小程序拿货档口的稳定页面路径，不再把它描述成后台地图功能。

- [ ] **步骤 5：验证后台**

运行：

```bash
cd admin-web
npm test
npm run build
rg -n 'SourcingMapView|sourcing_map|地图场景|标注画布|绑定申请|拿货地图' src scripts
```

预期：测试和构建 PASS，残留扫描不命中生产代码。

- [ ] **步骤 6：提交后台位置治理**

```bash
git add -A admin-web
git commit -m "refactor(admin): 合并商家位置异常治理"
```

---

### 任务 10：更新当前文档并完成全量验收

**文件：**

- 修改：`docs/product/technical-architecture.md`
- 修改：`docs/product/mvp-acceptance-checklist.md`
- 修改：`docs/product/api-implementation-checklist.md`
- 修改：`docs/product/wxapp-manual-acceptance.md`
- 修改：`docs/product/deployment-config.md`
- 修改：`docs/architecture.md`
- 修改：`docs/deployment.md`
- 修改：`backend/scripts/product_doc_governance.test.mjs`

**接口：**

- 输入：任务 1～9 的最终代码和契约。
- 产出：当前事实文档只描述商家自助位置、真实商家目录、腾讯地图位置页和异常治理。

- [ ] **步骤 1：写文档来源失败测试**

增加当前架构文档禁止恢复旧能力的断言：

```js
test('current docs describe merchant-owned locations without custom map operations', () => {
  const architecture = fs.readFileSync(path.join(productDocsDir, 'technical-architecture.md'), 'utf8')
  for (const retired of ['地图场景', '画布', '预录档口', '绑定申请']) {
    assert.equal(architecture.includes(retired), false, `current architecture still describes ${retired}`)
  }
  assert.match(architecture, /商家自主选择腾讯地图位置/)
  assert.match(architecture, /merchant_location_reports/)
  assert.match(architecture, /merchant_directory_events/)
})
```

- [ ] **步骤 2：确认文档测试失败**

运行：

```bash
cd backend
node --test scripts/product_doc_governance.test.mjs
```

预期：FAIL，因为当前架构仍描述场景、画布和绑定。

- [ ] **步骤 3：更新当前事实文档和验收口径**

架构文档更新数据流、表分组、核心关系、API、自动任务、故障排查和技术债；验收清单覆盖自助定位、目录搜索、附近商家、导航、三人报告隐藏、后台异常治理。删除演示场景、点位、绑定审核和旧地图 API 描述。

`deployment-config.md` 保留腾讯地图逆地理编码配置和调用基线，删除自定义地图运维说明。历史归档 PRD 不重写产品正文。

- [ ] **步骤 4：执行静态和生成门禁**

运行：

```bash
node backend/scripts/validate_demo_seed.mjs
make check-api-generated
cd backend
node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs scripts/product_doc_governance.test.mjs
```

预期：全部 PASS。

- [ ] **步骤 5：执行 PostgreSQL 真实迁移验证**

使用可丢弃 PostgreSQL：

```bash
cd backend
go run ./scripts/verify_migrations.go -config etc/app.yaml
cd ..
WPLINK_TEST_POSTGRES_DSN="$WPLINK_TEST_POSTGRES_DSN" make check-postgres
```

预期：全部 up/down、演示数据、任务协调和 SQL 限流测试 PASS；数据库中不存在旧地图表。

- [ ] **步骤 6：执行后端、后台和小程序全量验证**

运行：

```bash
make check
make check-dependencies
bash deploy/scripts/build-release.sh
git diff --check
git status --short
```

预期：全部命令退出码为 0；工作区只包含计划内文档改动或为空；发布包构建成功。

- [ ] **步骤 7：真机验收腾讯地图能力**

在微信开发者工具和真机依次验证：

1. 商家不选位置仍能保存资料和发布供需；
2. 商家选择织里腾讯地图位置并填写市场、楼栋、楼层、档口号；
3. 保存后无需后台审核即可出现在拿货档口；
4. 搜索商家名、市场和档口号均能命中；
5. 位置页显示当前商家和附近商家 Marker；
6. 点击导航打开腾讯地图位置；
7. 三个不同测试账号报告后位置自动隐藏；
8. 商家重新选点后恢复，后台能查询已结案报告。

- [ ] **步骤 8：提交文档和最终治理**

```bash
git add docs backend/scripts/product_doc_governance.test.mjs
git commit -m "docs: 更新织里商家目录和位置治理说明"
```
