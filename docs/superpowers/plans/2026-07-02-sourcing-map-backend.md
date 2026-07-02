# Sourcing Map Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建设拿货地图第一期后端数据结构和 API，让后台可以维护单街区地图数据，小程序可以读取已发布场景、对象详情和附近配套。

**Architecture:** 以 `map_scene`、`map_object`、`map_category` 三张表作为数据核心，公开接口只读取 `published/normal` 数据，后台接口维护 `draft/published/archived` 场景和对象。`.api` 与 SQL schema 作为源头，生成边界遵循 goctl，业务逻辑放在 `backend/app/internal/logic/map`，SQL 扩展放在 `backend/app/internal/model/map_model.go`，路由按现有 `domain_routes.go` 模式接入。

**Tech Stack:** Go 1.25、go-zero/goctl、PostgreSQL、JSONB、`net/http`、Node migration static check。

---

## 文件结构

- Create: `backend/migrations/000010_sourcing_map.up.sql`
  - 创建 `map_scene`、`map_object`、`map_category` 和索引，插入基础分类标签种子数据。
- Create: `backend/migrations/000010_sourcing_map.down.sql`
  - 按依赖顺序删除地图表。
- Create: `backend/app/api/map.api`
  - 定义公开地图 API 和后台地图 API 的请求/响应类型。
- Modify: `backend/app/api/app.api`
  - import `map.api`。
- Create: `backend/app/internal/model/map_model.go`
  - 地图业务查询和写入的自定义 model，封装 SQL、JSONB 编解码、bounds/search 字段计算。
- Create: `backend/app/internal/model/map_model_test.go`
  - 测试 bounds/search 计算和附近配套排序。
- Create: `backend/app/internal/logic/map/public_logic.go`
  - 小程序公开查询逻辑：场景列表、场景详情、对象列表、搜索、对象详情、附近配套。
- Create: `backend/app/internal/logic/map/public_logic_test.go`
  - 测试公开逻辑只查询发布数据、裁剪参数、映射响应。
- Create: `backend/app/internal/logic/map/admin_logic.go`
  - 后台管理逻辑：场景列表/保存/发布、对象保存/状态、批量生成、分类列表/保存。
- Create: `backend/app/internal/logic/map/admin_logic_test.go`
  - 测试后台校验、发布前置条件、批量生成。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 新增 `MapAPIStore` 并注册地图公开和后台路由。
- Create: `backend/app/internal/server/map_routes.go`
  - 手写地图路由接线，handler 只做解析和 response 映射。
- Create: `backend/app/internal/server/map_api_test.go`
  - 测试公开和后台地图 API 路由。
- Modify: `backend/app/internal/svc/service_context.go`
  - 将 `MapModel` 加入 `APIStore`。

## Task 1: 迁移和 API 源文件

**Files:**
- Create: `backend/migrations/000010_sourcing_map.up.sql`
- Create: `backend/migrations/000010_sourcing_map.down.sql`
- Create: `backend/app/api/map.api`
- Modify: `backend/app/api/app.api`
- Test: `backend/scripts/validate_migrations.mjs`

- [ ] **Step 1: 写迁移静态检查的失败测试条件**

确认当前 migration 静态检查会识别新增迁移缺失 down。先只创建 `000010_sourcing_map.up.sql`，运行：

```bash
cd backend && node scripts/validate_migrations.mjs
```

Expected: FAIL，提示 `000010_sourcing_map 缺少 down migration`。

- [ ] **Step 2: 写完整 up/down migration**

`000010_sourcing_map.up.sql` 需要包含：

```sql
CREATE TABLE IF NOT EXISTS map_scene (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint REFERENCES city_stations(id),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  parent_code varchar(64),
  background_url text NOT NULL,
  width int NOT NULL,
  height int NOT NULL,
  min_scale numeric(5,2) NOT NULL DEFAULT 0.5,
  max_scale numeric(5,2) NOT NULL DEFAULT 5,
  default_scale numeric(5,2) NOT NULL DEFAULT 1,
  default_center_x numeric(10,2),
  default_center_y numeric(10,2),
  floor_no varchar(20),
  sort int NOT NULL DEFAULT 0,
  revision int NOT NULL DEFAULT 1,
  status varchar(20) NOT NULL DEFAULT 'draft',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS map_object (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  scene_code varchar(64) NOT NULL REFERENCES map_scene(code),
  merchant_id bigint REFERENCES merchants(id),
  code varchar(64) NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  layer varchar(30) NOT NULL,
  geometry_type varchar(30) NOT NULL,
  geometry jsonb NOT NULL,
  center_x numeric(10,2),
  center_y numeric(10,2),
  min_x numeric(10,2),
  min_y numeric(10,2),
  max_x numeric(10,2),
  max_y numeric(10,2),
  min_zoom int NOT NULL DEFAULT 1,
  max_zoom int NOT NULL DEFAULT 5,
  category_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  service_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  platform_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  poi_service_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  address text,
  phone varchar(30),
  wechat varchar(50),
  lat numeric(10,7),
  lng numeric(10,7),
  search_text text NOT NULL DEFAULT '',
  extra jsonb NOT NULL DEFAULT '{}'::jsonb,
  sort int NOT NULL DEFAULT 0,
  status varchar(20) NOT NULL DEFAULT 'normal',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(scene_code, code)
);

CREATE TABLE IF NOT EXISTS map_category (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  icon_url text,
  sort int NOT NULL DEFAULT 0,
  is_visible boolean NOT NULL DEFAULT true,
  status varchar(20) NOT NULL DEFAULT 'normal',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

补充索引：

```sql
CREATE INDEX IF NOT EXISTS idx_map_scene_city_status ON map_scene(city_station_id, status, sort);
CREATE INDEX IF NOT EXISTS idx_map_scene_parent ON map_scene(parent_code);
CREATE INDEX IF NOT EXISTS idx_map_object_scene ON map_object(scene_code);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_type ON map_object(scene_code, type);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_bounds ON map_object(scene_code, min_x, max_x, min_y, max_y);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_status_sort ON map_object(scene_code, status, sort);
```

`000010_sourcing_map.down.sql` 使用：

```sql
DROP TABLE IF EXISTS map_object;
DROP TABLE IF EXISTS map_category;
DROP TABLE IF EXISTS map_scene;
```

- [ ] **Step 3: 写 `map.api` 并接入 `app.api`**

`map.api` 使用 `/api/v1` 前缀，至少定义：

```go
@server(
  prefix: /api/v1
  group: map
)
service wplink-api {
  @handler ListMapScenes
  get /map/scenes (ListMapScenesReq) returns (ListMapScenesResp)

  @handler GetMapScene
  get /map/scenes/:sceneCode returns (MapSceneResp)

  @handler ListMapObjects
  get /map/scenes/:sceneCode/objects (ListMapObjectsReq) returns (ListMapObjectsResp)

  @handler SearchMapObjects
  get /map/objects/search (SearchMapObjectsReq) returns (SearchMapObjectsResp)

  @handler GetMapObject
  get /map/objects/:objectId returns (MapObjectDetailResp)

  @handler ListNearbyPois
  get /map/objects/:objectId/nearby-pois (ListNearbyPoisReq) returns (ListNearbyPoisResp)
}
```

后台接口同文件使用 `/api/v1/admin` 前缀，定义 scenes、objects、categories 的管理接口。

- [ ] **Step 4: 验证 migration 和 api**

Run:

```bash
cd backend && node scripts/validate_migrations.mjs
cd backend && goctl api validate --api app/api/app.api
```

Expected: both PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/000010_sourcing_map.up.sql backend/migrations/000010_sourcing_map.down.sql backend/app/api/map.api backend/app/api/app.api
git commit -m "feat: add sourcing map backend schema"
```

## Task 2: 地图 model 和计算规则

**Files:**
- Create: `backend/app/internal/model/map_model.go`
- Create: `backend/app/internal/model/map_model_test.go`

- [ ] **Step 1: 写 bounds/search 失败测试**

在 `map_model_test.go` 写：

```go
func TestBuildMapObjectDerivedFieldsForRect(t *testing.T) {
	input := MapObjectInput{
		Code: "A001",
		Name: "A001 小鹿童装",
		Type: "booth",
		GeometryType: "rect",
		Geometry: JSONMap{"x": float64(520), "y": float64(260), "width": float64(80), "height": float64(50)},
		CategoryCodes: []string{"girl"},
		ServiceTags: []string{"spot"},
		Address: "利济路中段 A001",
	}
	fields, err := BuildMapObjectDerivedFields(input)
	if err != nil {
		t.Fatalf("BuildMapObjectDerivedFields() error = %v", err)
	}
	if fields.CenterX != 560 || fields.CenterY != 285 || fields.MinX != 520 || fields.MaxX != 600 {
		t.Fatalf("fields = %#v, want rect bounds", fields)
	}
	if !strings.Contains(fields.SearchText, "A001") || !strings.Contains(fields.SearchText, "小鹿童装") || !strings.Contains(fields.SearchText, "girl") {
		t.Fatalf("searchText = %q, want searchable code name category", fields.SearchText)
	}
}
```

Run:

```bash
cd backend && go test ./app/internal/model -run TestBuildMapObjectDerivedFieldsForRect -v
```

Expected: FAIL，提示 `BuildMapObjectDerivedFields` 未定义。

- [ ] **Step 2: 实现 model 类型和计算函数**

在 `map_model.go` 定义：

- 状态常量：`MapSceneStatusDraft`、`MapSceneStatusPublished`、`MapObjectStatusNormal`。
- 类型：`MapScene`、`MapObject`、`MapCategory`、`MapObjectInput`、`MapObjectDerivedFields`、`ListMapScenesFilter`、`ListMapObjectsFilter`。
- 函数：`BuildMapObjectDerivedFields(input MapObjectInput) (MapObjectDerivedFields, error)`。

规则：

- `rect`：`center = x + width/2, y + height/2`，bounds 为矩形范围。
- `point`：`center = x, y`，bounds 为同一点。
- 不支持的 geometry 返回友好错误：`地图标注形状不支持`。
- `search_text` 拼接 code、name、type、分类、标签、地址，去掉空值。

- [ ] **Step 3: 写附近配套排序测试**

测试 `SortNearbyMapObjects(origin, candidates, limit)`：

```go
func TestSortNearbyMapObjectsOrdersByDistance(t *testing.T) {
	origin := MapObject{ID: "booth-1", CenterX: 100, CenterY: 100}
	candidates := []MapObject{
		{ID: "poi-far", CenterX: 300, CenterY: 100},
		{ID: "poi-near", CenterX: 130, CenterY: 100},
	}
	items := SortNearbyMapObjects(origin, candidates, 1)
	if len(items) != 1 || items[0].ID != "poi-near" {
		t.Fatalf("items = %#v, want nearest poi", items)
	}
	if items[0].DistanceText != "30m" {
		t.Fatalf("distanceText = %q, want 30m", items[0].DistanceText)
	}
}
```

Expected: FAIL until function exists.

- [ ] **Step 4: 实现 SQL model 方法**

`MapModel` 包装 `*sql.DB` 和 `sqlx.SqlConn`，提供：

- `ListPublishedScenes(ctx, filter ListMapScenesFilter) ([]MapScene, error)`
- `GetPublishedScene(ctx, sceneCode string) (MapScene, error)`
- `ListPublishedObjects(ctx, filter ListMapObjectsFilter) ([]MapObject, error)`
- `SearchPublishedObjects(ctx, filter ListMapObjectsFilter) ([]MapObject, error)`
- `GetPublishedObject(ctx, objectID string) (MapObject, error)`
- `ListObjectsBySceneAndTypes(ctx, sceneCode string, types []string) ([]MapObject, error)`
- `ListAdminScenes(ctx, filter ListMapScenesFilter) ([]MapScene, error)`
- `SaveScene(ctx, input MapSceneInput) (MapScene, error)`
- `PublishScene(ctx, sceneCode string) (MapScene, error)`
- `SaveObject(ctx, input MapObjectInput) (MapObject, error)`
- `UpdateObjectStatus(ctx, objectID string, status string) (MapObject, error)`
- `BatchCreateObjects(ctx, inputs []MapObjectInput) ([]MapObject, error)`
- `ListCategories(ctx, categoryType string) ([]MapCategory, error)`
- `SaveCategory(ctx, input MapCategoryInput) (MapCategory, error)`

SQL 错误不直接返回给前端逻辑；model 返回原始错误，logic 层映射为友好中文。

- [ ] **Step 5: Run model tests**

```bash
cd backend && go test ./app/internal/model -run 'TestBuildMapObjectDerivedFields|TestSortNearbyMapObjects' -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/app/internal/model/map_model.go backend/app/internal/model/map_model_test.go
git commit -m "feat: add sourcing map model"
```

## Task 3: 公开查询 logic

**Files:**
- Create: `backend/app/internal/logic/map/public_logic.go`
- Create: `backend/app/internal/logic/map/public_logic_test.go`

- [ ] **Step 1: 写 published-only 失败测试**

测试 `ListScenes` 传给 store 的状态只允许 published：

```go
func TestPublicMapLogicListsPublishedScenes(t *testing.T) {
	store := &fakePublicMapStore{scenes: []model.MapScene{{Code: "zhili_lijilu_middle", Name: "利济路中段", Status: model.MapSceneStatusPublished}}}
	logic := NewPublicLogic(store)
	resp, err := logic.ListScenes(context.Background(), ListScenesReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListScenes() error = %v", err)
	}
	if store.sceneFilter.CityCode != "zhili" || store.sceneFilter.Status != model.MapSceneStatusPublished {
		t.Fatalf("filter = %#v, want zhili published", store.sceneFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Code != "zhili_lijilu_middle" {
		t.Fatalf("items = %#v, want scene", resp.Items)
	}
}
```

Run:

```bash
cd backend && go test ./app/internal/logic/map -run TestPublicMapLogicListsPublishedScenes -v
```

Expected: FAIL，logic 包不存在。

- [ ] **Step 2: 实现公开 logic**

`public_logic.go` 定义：

- `PublicStore` interface，依赖 Task 2 的 model 方法。
- `PublicLogic` 和构造函数 `NewPublicLogic(store PublicStore)`.
- 请求/响应类型：`ListScenesReq/Resp`、`MapSceneItem`、`ListObjectsReq/Resp`、`MapObjectItem`、`MapObjectDetailResp`、`NearbyPoiItem`。
- 方法：`ListScenes`、`GetScene`、`ListObjects`、`SearchObjects`、`GetObject`、`ListNearbyPois`。

错误映射：

- 场景不存在：`errx.New(errx.CodeNotFound, "地图场景不存在或未发布")`
- 对象不存在：`errx.New(errx.CodeNotFound, "地图点位不存在或未发布")`
- 查询失败：记录 log，返回 `地图数据加载失败，请稍后重试`

- [ ] **Step 3: 写对象搜索和附近配套测试**

覆盖：

- keyword trim。
- `limit <= 0` 默认 10。
- nearby 只传 POI types。
- 响应不暴露 hidden 对象。

- [ ] **Step 4: Run logic tests**

```bash
cd backend && go test ./app/internal/logic/map -run Public -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/app/internal/logic/map/public_logic.go backend/app/internal/logic/map/public_logic_test.go
git commit -m "feat: add public sourcing map logic"
```

## Task 4: 后台管理 logic

**Files:**
- Create: `backend/app/internal/logic/map/admin_logic.go`
- Create: `backend/app/internal/logic/map/admin_logic_test.go`

- [ ] **Step 1: 写场景保存校验失败测试**

```go
func TestAdminMapLogicRejectsSceneWithoutBackground(t *testing.T) {
	logic := NewAdminLogic(&fakeAdminMapStore{})
	_, err := logic.SaveScene(context.Background(), SaveSceneReq{
		Code: "zhili_lijilu_middle",
		Name: "利济路中段",
		Type: "street_segment",
		Width: 3000,
		Height: 1800,
	})
	if err == nil {
		t.Fatal("SaveScene() error = nil, want validation error")
	}
}
```

Expected: FAIL until admin logic exists.

- [ ] **Step 2: 实现后台 logic**

`admin_logic.go` 定义：

- `AdminStore` interface。
- `AdminLogic` 和构造函数 `NewAdminLogic(store AdminStore)`.
- 方法：`ListScenes`、`SaveScene`、`PublishScene`、`ListObjects`、`SaveObject`、`UpdateObjectStatus`、`BatchGenerateObjects`、`ListCategories`、`SaveCategory`。

关键校验：

- 场景 code/name/type/backgroundUrl 必填。
- width/height 必须大于 0。
- `status` 只允许 `draft/published/archived`。
- 对象 code/name/type/layer/geometryType 必填。
- 第一版对象 geometryType 只允许 `rect/point`。
- 发布前 store 返回对象数量必须大于 0。
- 批量生成数量范围 1 到 200。

- [ ] **Step 3: 写批量生成测试**

输入：

- 起始编号 `A001`
- 数量 `3`
- 横向
- 起点 `x=100,y=200`
- 宽高 `80x50`
- 间距 `5`

期望生成 `A001/A002/A003`，x 分别为 `100/185/270`。

- [ ] **Step 4: Run admin logic tests**

```bash
cd backend && go test ./app/internal/logic/map -run Admin -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/app/internal/logic/map/admin_logic.go backend/app/internal/logic/map/admin_logic_test.go
git commit -m "feat: add admin sourcing map logic"
```

## Task 5: 路由接线和服务上下文

**Files:**
- Modify: `backend/app/internal/server/domain_routes.go`
- Create: `backend/app/internal/server/map_routes.go`
- Create: `backend/app/internal/server/map_api_test.go`
- Modify: `backend/app/internal/svc/service_context.go`

- [ ] **Step 1: 写公开路由失败测试**

`map_api_test.go` 测试：

```go
func TestMapAPIRouterListsPublishedScenes(t *testing.T) {
	store := &fakeMapAPIStore{
		scenes: []model.MapScene{{Code: "zhili_lijilu_middle", Name: "利济路中段", Status: model.MapSceneStatusPublished}},
	}
	router := NewAPIRouter(store)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/scenes?cityCode=zhili", nil)
	router.ServeHTTP(rec, req)
	data := decodeEnvelopeData(t, rec, http.StatusOK)
	items := data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items = %#v, want one scene", items)
	}
}
```

Expected: FAIL，路由未注册。

- [ ] **Step 2: 实现路由接线**

`domain_routes.go` 新增：

```go
type MapAPIStore interface {
	maplogic.PublicStore
	maplogic.AdminStore
}
```

`registerOptionalDomainRoutes` 中：

```go
if mapStore, ok := store.(MapAPIStore); ok {
	registerMapRoutes(mux, mapStore)
}
```

`map_routes.go` 注册公开和后台路由，handler 只负责：

- 读 path/query/body。
- 调用 `maplogic.NewPublicLogic(store)` 或 `maplogic.NewAdminLogic(store)`。
- `response.JSON(w, resp, err)`。

- [ ] **Step 3: 接入 service context**

`APIStore` 增加：

```go
*model.MapModel
```

`newAPIStore` 中：

```go
MapModel: model.NewMapModel(db),
```

- [ ] **Step 4: Run server tests**

```bash
cd backend && go test ./app/internal/server -run Map -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/app/internal/server/domain_routes.go backend/app/internal/server/map_routes.go backend/app/internal/server/map_api_test.go backend/app/internal/svc/service_context.go
git commit -m "feat: wire sourcing map api routes"
```

## Task 6: 后端总体验证

**Files:**
- All backend files from Task 1-5.

- [ ] **Step 1: gofmt**

```bash
gofmt -w backend/app/internal/model/map_model.go backend/app/internal/model/map_model_test.go backend/app/internal/logic/map/public_logic.go backend/app/internal/logic/map/public_logic_test.go backend/app/internal/logic/map/admin_logic.go backend/app/internal/logic/map/admin_logic_test.go backend/app/internal/server/map_routes.go backend/app/internal/server/map_api_test.go backend/app/internal/server/domain_routes.go backend/app/internal/svc/service_context.go
```

- [ ] **Step 2: API and migration validation**

```bash
cd backend && node scripts/validate_migrations.mjs
cd backend && goctl api validate --api app/api/app.api
```

Expected: PASS.

- [ ] **Step 3: Backend unit tests**

```bash
cd backend && go test ./app/internal/model ./app/internal/logic/map ./app/internal/server ./app/internal/svc
```

Expected: PASS.

- [ ] **Step 4: Full backend tests**

```bash
cd backend && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit verification fixes if needed**

If verification fixes are required:

```bash
git add backend
git commit -m "test: verify sourcing map backend"
```

## Self-Review

- Spec coverage: 本计划覆盖方案 C 第一期的后端数据结构、公开 API、后台 API、附近配套和服务接线；不覆盖后台 Vue 标注工具和小程序页面，它们应分别进入独立计划。
- Placeholder scan: 本计划没有 `TBD`、`TODO` 或“后续补充”式占位；每个任务都有具体文件、测试、命令和期望结果。
- Type consistency: 统一使用 `MapScene`、`MapObject`、`MapCategory`、`MapModel`、`PublicLogic`、`AdminLogic`，接口前缀统一为 `/api/v1/map` 和 `/api/v1/admin/map`。
