# 小程序档口位置与周边入驻商家实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将“拿货地图”收敛为列表优先的“拿货档口”，新增以当前商家为强焦点、可浏览 1 公里内周边已入驻商家的独立位置地图页。

**架构：** 后端在现有 `map_object` 与商家绑定关系上增加公开的商家位置上下文接口，一次返回当前商家和按直线距离排序的周边商家；小程序一级页只保留目录职责，独立位置页负责 Marker、导航和半屏周边列表。地图行为通过独立低优先级事件接口记录，不影响浏览、导航或商家跳转主流程。

**技术栈：** Go、go-zero API DSL、PostgreSQL、Vue 3、uni-app、微信小程序原生 `map`、Node.js 原生测试。

## 全局约束

- 所有行为修改严格执行 RED → GREEN → REFACTOR；每个任务先确认新增测试因缺少目标行为而失败。
- 涉及 Go 后端实现时，必须使用 `backend-friendly-errors-logging` 技能统一中文错误、诊断日志与注释。
- `.api` 是接口契约源文件；`backend/app/internal/types/types.go` 只允许通过 goctl 生成，不手写。
- 当前商家与周边商家的位置均以已发布、状态正常的 `map_object` 经纬度为准。
- 周边范围固定为直线距离 1000 米，最多 20 家；只包含已入驻、状态正常、具有合法经纬度的其他商家。
- 当前商家 Marker 使用 `#C23A00`、常驻名称和最高层级；周边 Marker 默认使用 `#94A3B8`，临时选中使用 `#475569`。
- 商家位置页不调用 `uni.getLocation`、不声明 `show-location`，也不新增用户定位授权依赖。
- 首期不提供地图区域搜索、地图/列表同级切换、周边筛选、Marker 聚合、室内地图或路线规划。
- 地图埋点失败必须静默降级，不显示 Toast、不重试阻塞主流程。
- 旧 `legacy-canvas.vue` 和 Canvas 地图辅助文件继续保留，不进入用户访问链路。
- Marker 属性以[微信官方 map 组件文档](https://developers.weixin.qq.com/miniprogram/dev/component/map.html)为准；`callout.display='ALWAYS'`、`alpha` 和 `zIndex` 用于视觉层级。

## 文件职责

- `backend/app/api/map.api`：商家位置上下文公开契约。
- `backend/app/internal/model/map_model.go`：当前商家点位、周边候选、球面距离过滤与排序。
- `backend/app/internal/logic/map/public_logic.go`：位置上下文校验、中文错误和 DTO 组装。
- `wxapp/pages/sourcing-map/index.vue`：只负责拿货档口搜索、筛选和分页列表。
- `wxapp/pages/merchant/location.vue`：当前商家地图、导航、周边 Marker 和半屏列表。
- `wxapp/pages/merchant/locationState.js`：无框架依赖的归一化、Marker 构造和 Marker ID 映射。
- `backend/app/internal/model/merchant_map_events_model*.go`：goctl 生成的事件表基础模型和自定义写入。
- `wxapp/common/merchantMapAnalytics.js`：低优先级地图事件上报。

---

## Task 1：增加商家位置上下文 API 契约

**文件：**

- 修改：`backend/app/api/map.api`
- 生成：`backend/app/internal/types/types.go`
- 修改：`backend/app/internal/types/map_contract_test.go`

**接口：**

- 产出：`GET /api/v1/map/merchants/:merchantId/location-context`
- 产出：`MerchantLocationContextResp { Current MerchantPlaceItem; Nearby []MerchantPlaceItem; RadiusMeters int64; NearbyAvailable bool }`
- 变更：`MerchantPlaceItem.DistanceMeters int64`

- [ ] **步骤 1：先写失败的契约测试**

```go
func TestMerchantLocationContextTypesExposeCurrentNearbyAndDistance(t *testing.T) {
	respType := reflect.TypeOf(MerchantLocationContextResp{})
	for _, name := range []string{"Current", "Nearby", "RadiusMeters", "NearbyAvailable"} {
		if _, ok := respType.FieldByName(name); !ok {
			t.Fatalf("MerchantLocationContextResp missing %s", name)
		}
	}
	itemType := reflect.TypeOf(MerchantPlaceItem{})
	field, ok := itemType.FieldByName("DistanceMeters")
	if !ok || string(field.Tag) != `json:"distanceMeters,optional"` {
		t.Fatal("MerchantPlaceItem.DistanceMeters contract mismatch")
	}
}
```

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd backend && go test ./app/internal/types -run TestMerchantLocationContextTypesExposeCurrentNearbyAndDistance -count=1
```

预期：编译失败，新增响应和字段尚不存在。

- [ ] **步骤 3：修改 `.api` 契约**

```go
type MerchantLocationContextResp {
	Current         MerchantPlaceItem   `json:"current"`
	Nearby          []MerchantPlaceItem `json:"nearby"`
	RadiusMeters    int64               `json:"radiusMeters"`
	NearbyAvailable bool                `json:"nearbyAvailable"`
}

@handler GetMerchantLocationContext
get /map/merchants/:merchantId/location-context returns (MerchantLocationContextResp)
```

同时在 `MerchantPlaceItem` 增加：

```go
DistanceMeters int64 `json:"distanceMeters,optional"`
```

- [ ] **步骤 4：通过 goctl 生成类型**

```bash
cd backend/app
WPLINK_GOCTL_TMP="$(mktemp -d /private/tmp/wplink-location-contract.XXXXXX)"
goctl api go -api ./api/app.api -dir "$WPLINK_GOCTL_TMP"
cp "$WPLINK_GOCTL_TMP/internal/types/types.go" ./internal/types/types.go
goctl api validate --api ./api/app.api
```

预期：只复制生成的 `types.go`，不复制临时 handler 和 logic。

- [ ] **步骤 5：运行测试确认 GREEN**

```bash
cd backend && go test ./app/internal/types -run TestMerchantLocationContextTypesExposeCurrentNearbyAndDistance -count=1
```

- [ ] **步骤 6：提交契约**

```bash
git add backend/app/api/map.api backend/app/internal/types/types.go backend/app/internal/types/map_contract_test.go
git commit -m "feat: 定义商家位置上下文接口"
```

## Task 2：实现当前商家与周边商家模型查询

**文件：**

- 修改：`backend/app/internal/model/map_model.go`
- 修改：`backend/app/internal/model/map_model_test.go`

**接口：**

- 产出：`GetPublishedMerchantPlaceByMerchantID(ctx context.Context, merchantID string) (MerchantPlace, error)`
- 产出：`ListNearbyMerchantPlaces(ctx context.Context, origin MerchantPlace, radiusMeters int64, limit int64) ([]MerchantPlace, error)`
- 产出：`MerchantPlace.DistanceMeters int64`

- [ ] **步骤 1：写失败的距离、过滤和排序测试**

```go
func TestRankNearbyMerchantPlacesFiltersRadiusOriginAndInvalidLocation(t *testing.T) {
	origin := MerchantPlace{Object: MapObject{ID: "object-1", MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}}
	candidates := []MerchantPlace{
		{Object: MapObject{ID: "object-1", MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-near", MerchantID: "merchant-2", Lat: "30.8705000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-far", MerchantID: "merchant-3", Lat: "30.8900000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-invalid", MerchantID: "merchant-4"}},
	}
	items := rankNearbyMerchantPlaces(origin, candidates, 1000, 20)
	if len(items) != 1 || items[0].Object.MerchantID != "merchant-2" {
		t.Fatalf("items = %#v", items)
	}
	if items[0].DistanceMeters < 50 || items[0].DistanceMeters > 60 {
		t.Fatalf("distance = %d, want about 56m", items[0].DistanceMeters)
	}
}
```

另写 SQL 构造测试，断言候选查询包含已发布场景、正常档口、`m.status='active'`、有效经纬度、排除当前 `merchant_id` 和 1 公里外接矩形。

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd backend && go test ./app/internal/model -run 'TestRankNearbyMerchantPlaces|TestBuildNearbyMerchantPlace' -count=1
```

- [ ] **步骤 3：实现球面距离与稳定排序**

```go
func geoDistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeters = 6371000
	lat1Rad, lat2Rad := lat1*math.Pi/180, lat2*math.Pi/180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
```

`rankNearbyMerchantPlaces` 排除当前商家、空商家 ID、非法坐标和半径外结果；按 `DistanceMeters`、`Object.ID` 稳定排序后截取上限。

- [ ] **步骤 4：实现数据库方法**

`GetPublishedMerchantPlaceByMerchantID` 复用 `joinedMapObjectSelectColumns` 和 `scanMerchantPlace`，固定条件：

```sql
o.merchant_id = $1::bigint
AND o.status = 'normal'
AND o.layer = 'booth'
AND s.status = 'published'
AND m.id IS NOT NULL
AND m.status = 'active'
```

`ListNearbyMerchantPlaces` 根据半径计算经纬度外接矩形，只读取矩形内的已入驻候选，再调用纯函数精确过滤和排序。不得查询或返回电话、微信。

- [ ] **步骤 5：运行测试确认 GREEN**

```bash
cd backend
gofmt -w app/internal/model/map_model.go app/internal/model/map_model_test.go
go test ./app/internal/model -run 'TestRankNearbyMerchantPlaces|TestBuildNearbyMerchantPlace|TestBuildMerchantPlaceFilterSQL' -count=1
```

- [ ] **步骤 6：提交模型实现**

```bash
git add backend/app/internal/model/map_model.go backend/app/internal/model/map_model_test.go
git commit -m "feat: 查询商家周边入驻档口"
```

## Task 3：接通位置上下文逻辑与公开路由

**文件：**

- 修改：`backend/app/internal/logic/map/public_logic.go`
- 修改：`backend/app/internal/logic/map/public_logic_test.go`
- 修改：`backend/app/internal/server/map_routes.go`
- 修改：`backend/app/internal/server/map_api_test.go`

**接口：**

- 消费：任务 2 的两个查询方法。
- 产出：`PublicLogic.GetMerchantLocationContext(ctx, merchantID)`。
- 产出：公开路由 `GET /api/v1/map/merchants/{merchantId}/location-context`。

- [ ] **步骤 1：写失败的逻辑测试**

```go
func TestPublicMapLogicGetsMerchantLocationContext(t *testing.T) {
	store := &fakePublicMapStore{
		merchantPlace: model.MerchantPlace{Object: model.MapObject{ID: "object-1", MerchantID: "merchant-1", MerchantName: "小熊星球童装", Lat: "30.8700000", Lng: "120.1200000"}},
		nearbyMerchantPlaces: []model.MerchantPlace{{Object: model.MapObject{ID: "object-2", MerchantID: "merchant-2", MerchantName: "布谷童装", Lat: "30.8705000", Lng: "120.1200000"}, DistanceMeters: 56}},
	}
	resp, err := NewPublicLogic(store).GetMerchantLocationContext(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Current.MerchantId != "merchant-1" || len(resp.Nearby) != 1 || resp.RadiusMeters != 1000 || !resp.NearbyAvailable {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.Nearby[0].DistanceText != "56m" {
		t.Fatalf("nearby = %#v", resp.Nearby[0])
	}
}
```

同时覆盖空 ID、`sql.ErrNoRows`、当前坐标非法和周边查询错误。前三类固定中文错误为“该商家暂时无法查看”“该商家位置待完善”；周边查询错误不得让整页失败，必须返回当前商家、空 `nearby` 和 `nearbyAvailable=false`，并记录带 `merchantId` 的诊断日志。

- [ ] **步骤 2：运行逻辑测试确认 RED**

```bash
cd backend && go test ./app/internal/logic/map -run TestPublicMapLogicGetsMerchantLocationContext -count=1
```

- [ ] **步骤 3：实现逻辑、接口和 DTO 映射**

```go
const (
	merchantLocationRadiusMeters int64 = 1000
	merchantLocationNearbyLimit  int64 = 20
)

type MerchantLocationContextResp struct {
	Current         MerchantPlaceItem   `json:"current"`
	Nearby          []MerchantPlaceItem `json:"nearby"`
	RadiusMeters    int64               `json:"radiusMeters"`
	NearbyAvailable bool                `json:"nearbyAvailable"`
}
```

为 `PublicStore` 增加任务 2 的方法；`mapMerchantPlaceItems` 填充 `DistanceMeters` 和服务端格式化的 `DistanceText`。错误日志记录 `merchantId`、半径和上限，不记录隐私字段。

- [ ] **步骤 4：写失败路由测试并注册路由**

请求：

```go
req := httptest.NewRequest(http.MethodGet, "/api/v1/map/merchants/merchant-1/location-context", nil)
```

断言响应含 `current.merchantId=merchant-1`、`nearby[0].merchantId=merchant-2`、`radiusMeters=1000`、`nearbyAvailable=true`。另测周边存储失败仍返回 HTTP 200、当前商家和 `nearbyAvailable=false`。先确认 404，再在 `map_routes.go` 注册公开 GET 路由。

- [ ] **步骤 5：运行测试确认 GREEN**

```bash
cd backend
gofmt -w app/internal/logic/map/public_logic.go app/internal/logic/map/public_logic_test.go app/internal/server/map_routes.go app/internal/server/map_api_test.go
go test ./app/internal/logic/map ./app/internal/server -run 'MerchantLocationContext|MapAPIRouterGetsMerchantLocationContext' -count=1
```

- [ ] **步骤 6：提交后端位置上下文**

```bash
git add backend/app/internal/logic/map/public_logic.go backend/app/internal/logic/map/public_logic_test.go backend/app/internal/server/map_routes.go backend/app/internal/server/map_api_test.go
git commit -m "feat: 提供商家位置上下文接口"
```

## Task 4：将一级入口收敛为“拿货档口”列表

**文件：**

- 修改：`wxapp/pages.json`
- 修改：`wxapp/pages/home/index.vue`
- 修改：`wxapp/pages/home/index.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
- 修改：`wxapp/components/MerchantPlaceCard.vue`
- 修改：`wxapp/components/MerchantPlaceCard.test.mjs`
- 修改：`wxapp/pages/merchant/detail.vue`
- 修改：`wxapp/pages/merchant/detail.test.mjs`
- 修改：`wxapp/scripts/validate-flows.mjs`
- 修改：`wxapp/scripts/validate-flows.test.mjs`

**接口：**

- 产出：`openMerchantLocation(place)` 跳转 `/pages/merchant/location?merchantId=...`。
- 保留：待认领档口仍可直接 `uni.openLocation`，但不进入商家位置上下文。

- [ ] **步骤 1：先更新页面和流程测试**

```js
assert.equal(tab.text, '拿货档口')
assert.doesNotMatch(source, /viewMode|merchantTencentMap|searchCurrentMapRegion|搜索此区域/)
assert.match(source, /function openMerchantLocation\(place\)/)
assert.match(source, /\/pages\/merchant\/location\?merchantId=/)
```

组件测试断言：已入驻且有坐标显示“查看位置”；待认领且有坐标显示“导航”；缺坐标只显示“位置待完善”。商家详情测试断言移除内嵌 `<map>`，有效坐标时跳转独立位置页。

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd wxapp && node --test pages/home/index.test.mjs pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs pages/merchant/detail.test.mjs scripts/validate-flows.test.mjs
```

- [ ] **步骤 3：修改入口文案和列表卡片动作**

- 页面标题、Tab、首页快捷入口和流程校验文案统一改为“拿货档口”。
- `MerchantPlaceCard` 增加 `location` 事件；已入驻商家按钮为“查看位置”，待认领点位继续发出 `navigate`。

```vue
<button v-if="place.claimed && hasLocation" @click.stop="$emit('location', place)">查看位置</button>
<button v-else-if="hasLocation" @click.stop="$emit('navigate', place)">导航</button>
```

- [ ] **步骤 4：删除一级页地图模式**

删除 `viewMode`、原生 `<map>`、Marker、区域查询、`MAP_PAGE_SIZE`、`uni.getLocation` 和地图选中卡片，只保留搜索、筛选、分页、下拉刷新、认领和商家主页。

```js
function openMerchantLocation(place) {
  if (!place.claimed || !place.merchantId || !hasValidLocation(place)) return
  uni.navigateTo({ url: `/pages/merchant/location?merchantId=${encodeURIComponent(place.merchantId)}` })
}
```

- [ ] **步骤 5：收敛商家主页地址区**

移除内嵌地图，保留文字地址。有效坐标时按钮改为“查看位置”并跳转独立位置页；无坐标时继续复制地址。

```js
function openMerchantLocation() {
  if (!merchant.value.id || !merchantAddressLocation.value?.hasGps) return
  uni.navigateTo({ url: `/pages/merchant/location?merchantId=${encodeURIComponent(merchant.value.id)}` })
}
```

- [ ] **步骤 6：运行测试确认 GREEN**

```bash
cd wxapp && node --test pages/home/index.test.mjs pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs pages/merchant/detail.test.mjs scripts/validate-flows.test.mjs
```

- [ ] **步骤 7：提交一级页收敛**

```bash
git add wxapp/pages.json wxapp/pages/home/index.vue wxapp/pages/home/index.test.mjs wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs wxapp/components/MerchantPlaceCard.vue wxapp/components/MerchantPlaceCard.test.mjs wxapp/pages/merchant/detail.vue wxapp/pages/merchant/detail.test.mjs wxapp/scripts/validate-flows.mjs wxapp/scripts/validate-flows.test.mjs
git commit -m "refactor: 将拿货地图收敛为档口列表"
```

## Task 5：建立商家位置页状态与当前商家地图

**文件：**

- 修改：`wxapp/api/sourcingMap.js`
- 新增：`wxapp/pages/merchant/locationState.js`
- 新增：`wxapp/pages/merchant/locationState.test.mjs`
- 新增：`wxapp/pages/merchant/location.vue`
- 新增：`wxapp/pages/merchant/location.test.mjs`
- 新增：`wxapp/static/map/marker-nearby.png`
- 新增：`wxapp/static/map/marker-nearby-selected.png`
- 修改：`wxapp/pages.json`
- 修改：`wxapp/scripts/validate-pages.mjs`

**接口：**

- 产出：`getMerchantLocationContext(merchantId)`。
- 产出：`normalizeMerchantLocationContext(raw)`、`buildMerchantLocationMarkers(context, selectedNearbyMerchantId)`、`merchantIdFromMarker(markerId, markers)`。

- [ ] **步骤 1：写状态模块失败测试**

测试非法坐标过滤、当前 Marker 固定 ID 1、橙色资源、最大尺寸、`zIndex:10`、`callout.display:'ALWAYS'`，以及周边普通/临时选中资源和 Marker ID 反查。

```js
const markers = buildMerchantLocationMarkers(context, 'merchant-2')
assert.equal(markers[0].merchantId, 'merchant-1')
assert.equal(markers[0].callout.display, 'ALWAYS')
assert.equal(markers[0].zIndex, 10)
assert.equal(markers[1].iconPath, '/static/map/marker-nearby-selected.png')
assert.ok(markers[0].width > markers[1].width)
```

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd wxapp && node --test pages/merchant/locationState.test.mjs
```

- [ ] **步骤 3：实现 API 和纯状态模块**

```js
export function getMerchantLocationContext(merchantId) {
  return request({
    url: `/api/v1/map/merchants/${merchantId}/location-context`,
    method: 'GET',
    suppressErrorToast: true,
  })
}
```

`locationState.js` 复用 `normalizeMerchantPlace`、`hasValidLocation`，不调用 `uni`。

- [ ] **步骤 4：生成两张周边 Marker PNG**

使用 `imagegen` 技能，以现有 `marker-claimed.png` 的轮廓、透明画布和锚点为参考：

- `marker-nearby.png` 填充 `#94A3B8`。
- `marker-nearby-selected.png` 填充 `#475569`。

不增加文字、阴影或 Logo；用 `view_image` 检查透明边缘和颜色差异。

- [ ] **步骤 5：先写位置页失败测试并注册页面**

```js
expectTokens(source, [
  'getMerchantLocationContext',
  'buildMerchantLocationMarkers',
  ':scale="16"',
  '@markertap="handleMarkerTap"',
  '导航到店',
  '该商家暂时无法查看',
  '该商家位置待完善',
])
assert.doesNotMatch(source, /show-location|uni\.getLocation|搜索此区域/)
```

在 `pages.json` 注册 `pages/merchant/location`，标题“店铺位置”；在 `validate-pages.mjs` 加入必需页面。

- [ ] **步骤 6：实现当前商家地图和降级状态**

```js
onLoad(async (options) => {
  merchantId.value = String(options.merchantId || '').trim()
  if (!merchantId.value) {
    errorText.value = '该商家暂时无法查看'
    return
  }
  await loadLocationContext()
})
```

成功后以当前商家为中心、`scale=16`，不设置 `show-location`。底部卡展示名称、地址、主营和“导航到店”；`uni.openLocation` 失败提示“导航打开失败，请稍后重试”。当前商家上下文失败显示“重新加载”和“返回”；只有 `nearbyAvailable=false` 时继续显示地图，并把周边区降级为“周边商家加载失败，请重试”。

- [ ] **步骤 7：运行测试确认 GREEN**

```bash
cd wxapp
node --test pages/merchant/locationState.test.mjs pages/merchant/location.test.mjs
npm run validate:pages
```

- [ ] **步骤 8：提交位置页基础能力**

```bash
git add wxapp/api/sourcingMap.js wxapp/pages/merchant/locationState.js wxapp/pages/merchant/locationState.test.mjs wxapp/pages/merchant/location.vue wxapp/pages/merchant/location.test.mjs wxapp/static/map/marker-nearby.png wxapp/static/map/marker-nearby-selected.png wxapp/pages.json wxapp/scripts/validate-pages.mjs
git commit -m "feat: 新增商家位置地图页"
```

## Task 6：实现周边半屏列表与 Marker 联动

**文件：**

- 修改：`wxapp/pages/merchant/location.vue`
- 修改：`wxapp/pages/merchant/location.test.mjs`
- 修改：`wxapp/pages/merchant/locationState.js`
- 修改：`wxapp/pages/merchant/locationState.test.mjs`

**接口：**

- 产出：`openNearbyDrawer()`、`closeNearbyDrawer()`、`selectNearbyMerchant(merchantId)`、`openNearbyMerchant(place)`。

- [ ] **步骤 1：写失败的抽屉和联动测试**

```js
expectTokens(source, [
  '周边已入驻商家',
  'nearby-drawer',
  'scroll-into-view',
  'selectNearbyMerchant',
  'openNearbyMerchant',
  '/pages/merchant/detail?id=',
  '附近暂无其他入驻商家',
  '周边商家加载失败，请重试',
])
```

状态测试额外断言保持服务端距离顺序、排除当前商家、去重且最多 20 家。

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd wxapp && node --test pages/merchant/locationState.test.mjs pages/merchant/location.test.mjs
```

- [ ] **步骤 3：实现折叠与展开状态**

- 折叠态显示“周边已入驻商家 N 家”。
- `N=0` 时显示“附近暂无其他入驻商家”，不可展开。
- `nearbyAvailable=false` 时显示“周边商家加载失败，请重试”；重试只更新上下文数据，不清空已经渲染的当前商家。
- 展开高度不超过视口 55%，使用 `scroll-view`。
- 每项只展示名称、最多 3 个主营标签、档口地址、`distanceText` 和“查看商家”。
- 不增加搜索、筛选、分页和区域刷新。

- [ ] **步骤 4：实现双向联动**

Marker 点击后设置 `selectedNearbyMerchantId`、展开抽屉、设置 `scrollIntoViewId='nearby-'+merchantId`。列表项点击后更新 Marker，并调用 `uni.createMapContext('merchantLocationMap').includePoints` 同时显示当前商家和周边商家。关闭抽屉时清除临时选择，恢复当前中心和 `scale=16`。

```js
function openNearbyMerchant(place) {
  uni.navigateTo({ url: `/pages/merchant/detail?id=${encodeURIComponent(place.merchantId)}` })
}
```

不得直接把周边商家替换为当前地图上下文。

- [ ] **步骤 5：运行测试确认 GREEN**

```bash
cd wxapp && node --test pages/merchant/locationState.test.mjs pages/merchant/location.test.mjs
```

- [ ] **步骤 6：提交周边交互**

```bash
git add wxapp/pages/merchant/location.vue wxapp/pages/merchant/location.test.mjs wxapp/pages/merchant/locationState.js wxapp/pages/merchant/locationState.test.mjs
git commit -m "feat: 增加周边入驻商家列表"
```

## Task 7：建立地图事件后端记录链路

**文件：**

- 新增：`backend/migrations/000033_merchant_map_events.up.sql`
- 新增：`backend/migrations/000033_merchant_map_events.down.sql`
- 修改：`backend/scripts/validate_migrations.test.mjs`
- 生成：`backend/app/internal/model/merchant_map_events_model_gen.go`
- 新增：`backend/app/internal/model/merchant_map_events_model.go`
- 新增：`backend/app/internal/logic/metrics/record_merchant_map_event_logic.go`
- 新增：`backend/app/internal/logic/metrics/record_merchant_map_event_logic_test.go`
- 修改：`backend/app/api/metrics.api`
- 生成：`backend/app/internal/types/types.go`
- 修改：`backend/app/internal/server/api.go`
- 修改：`backend/app/internal/server/resource_api_test.go`
- 修改：`backend/app/internal/svc/service_context.go`

**接口：**

- 产出：`POST /api/v1/metrics/merchant-map-events`。
- 允许：`location_entry_click`、`location_view`、`navigation_click`、`nearby_drawer_open`、`nearby_marker_click`、`nearby_list_item_click`、`nearby_merchant_click`。
- 允许来源：`directory`、`merchant_detail`、`merchant_location`。

- [ ] **步骤 1：写 migration 失败测试**

要求 up/down 成对存在；up 包含 `merchant_map_events`、`merchant_id`、`target_merchant_id`、`event_type`、`source`、`visitor_key`、`session_id`、允许事件/来源 CHECK 和 `(merchant_id,event_type,created_at)` 索引；down 删除索引和表。

```bash
cd backend && node --test scripts/validate_migrations.test.mjs
```

- [ ] **步骤 2：编写可回滚 migration**

```sql
CREATE TABLE IF NOT EXISTS merchant_map_events (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint REFERENCES users(id) ON DELETE SET NULL,
  merchant_id bigint NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  target_merchant_id bigint REFERENCES merchants(id) ON DELETE SET NULL,
  visitor_key varchar(96) NOT NULL,
  session_id varchar(96) NOT NULL,
  event_type varchar(40) NOT NULL,
  source varchar(30) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_merchant_map_event_type CHECK (event_type IN (
    'location_entry_click', 'location_view', 'navigation_click', 'nearby_drawer_open',
    'nearby_marker_click', 'nearby_list_item_click', 'nearby_merchant_click'
  )),
  CONSTRAINT chk_merchant_map_event_source CHECK (source IN (
    'directory', 'merchant_detail', 'merchant_location'
  ))
);
```

创建商家/事件/时间索引和目标商家/时间索引；down 先删除索引再删除表。

- [ ] **步骤 3：用 goctl 生成表模型并验证 migration**

仅对本地开发数据库执行：

```bash
cd backend
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000033_merchant_map_events.up.sql
WPLINK_MODEL_TMP="$(mktemp -d /private/tmp/wplink-map-events-model.XXXXXX)"
goctl model pg datasource --url "$DATABASE_URL" --table merchant_map_events --dir "$WPLINK_MODEL_TMP" --style go_zero
cp "$WPLINK_MODEL_TMP/merchant_map_events_model_gen.go" app/internal/model/merchant_map_events_model_gen.go
go run ./scripts/verify_migrations.go -config etc/app.yaml
```

生成文件必须带 `Code generated by goctl`；不手写表基础 CRUD。

- [ ] **步骤 4：写事件逻辑失败测试**

覆盖合法事件、未知事件、未知来源、空或超长 visitor/session、周边点击缺失 `targetMerchantId`、数据库错误。中文错误固定为“地图行为类型无效”“地图行为来源无效”“地图行为会话参数无效”“地图行为记录失败，请稍后重试”。

- [ ] **步骤 5：实现模型扩展、逻辑和 API 契约**

```go
type MerchantMapEventInput struct {
	UserID           string
	MerchantID       string
	TargetMerchantID string
	VisitorKey       string
	SessionID        string
	EventType        string
	Source           string
}
```

```go
type MerchantMapEventReq {
	MerchantId       string `json:"merchantId"`
	TargetMerchantId string `json:"targetMerchantId,optional"`
	VisitorKey       string `json:"visitorKey"`
	SessionId        string `json:"sessionId"`
	EventType        string `json:"eventType"`
	Source           string `json:"source"`
}

type MerchantMapEventResp {
	Recorded bool `json:"recorded"`
}
```

用任务 1 的临时目录方式重新生成 `types.go`。在 `service_context.go` 嵌入并初始化 `*model.MerchantMapEventsModel`。

- [ ] **步骤 6：写路由失败测试并注册路由**

仅当 store 实现 `MerchantMapEventStore` 时注册；用 `optionalUserIDFromBearerToken`，匿名可记录，有无效 token 时返回登录过期，不能读取请求体 userId。测试合法匿名、有效 token、未知事件和数据库错误不暴露 SQL。

- [ ] **步骤 7：运行后端测试确认 GREEN**

```bash
cd backend
gofmt -w app/internal/model/merchant_map_events_model.go app/internal/logic/metrics/record_merchant_map_event_logic.go app/internal/logic/metrics/record_merchant_map_event_logic_test.go app/internal/server/api.go app/internal/server/resource_api_test.go app/internal/svc/service_context.go
go test ./app/internal/logic/metrics ./app/internal/server ./app/internal/svc -run 'MerchantMapEvent|ServiceContext' -count=1
node --test scripts/validate_migrations.test.mjs
goctl api validate --api app/api/app.api
```

- [ ] **步骤 8：提交后端事件链路**

```bash
git add backend/migrations/000033_merchant_map_events.up.sql backend/migrations/000033_merchant_map_events.down.sql backend/scripts/validate_migrations.test.mjs backend/app/internal/model/merchant_map_events_model_gen.go backend/app/internal/model/merchant_map_events_model.go backend/app/internal/logic/metrics/record_merchant_map_event_logic.go backend/app/internal/logic/metrics/record_merchant_map_event_logic_test.go backend/app/api/metrics.api backend/app/internal/types/types.go backend/app/internal/server/api.go backend/app/internal/server/resource_api_test.go backend/app/internal/svc/service_context.go
git commit -m "feat: 记录商家地图行为"
```

## Task 8：接入小程序地图行为埋点

**文件：**

- 修改：`wxapp/api/metrics.js`
- 新增：`wxapp/common/merchantMapAnalytics.js`
- 新增：`wxapp/common/merchantMapAnalytics.test.mjs`
- 修改：`wxapp/pages/merchant/location.vue`
- 修改：`wxapp/pages/merchant/location.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
- 修改：`wxapp/pages/merchant/detail.vue`
- 修改：`wxapp/pages/merchant/detail.test.mjs`

**接口：**

- 产出：`recordMerchantMapEvent(data)`。
- 产出：`trackMerchantMapEvent({ merchantId, targetMerchantId, eventType, source })`。

- [ ] **步骤 1：写失败的埋点测试**

断言 visitor key 持久化且不超过 96 字符、会话内复用 session ID、空商家 ID、未知事件或未知来源不请求、请求失败不重试。页面测试断言七类事件分别位于目录/商家主页位置入口、上下文成功、导航、抽屉打开、周边 Marker 点击、周边列表项点击和周边商家跳转处；Marker 内部联动不得触发列表项事件。

- [ ] **步骤 2：运行测试确认 RED**

```bash
cd wxapp && node --test common/merchantMapAnalytics.test.mjs pages/merchant/location.test.mjs pages/sourcing-map/index.test.mjs pages/merchant/detail.test.mjs
```

- [ ] **步骤 3：实现 API 和静默上报模块**

```js
export function recordMerchantMapEvent(data) {
  return request({
    url: '/api/v1/metrics/merchant-map-events',
    method: 'POST',
    data,
    suppressErrorToast: true,
  })
}
```

```js
export function trackMerchantMapEvent({ merchantId, targetMerchantId = '', eventType, source }) {
  if (!merchantId || !ALLOWED_EVENTS.has(eventType) || !ALLOWED_SOURCES.has(source)) return
  recordMerchantMapEvent({
    merchantId,
    targetMerchantId,
    eventType,
    source,
    visitorKey: getVisitorKey(),
    sessionId,
  }).catch(() => {
    // 地图行为属于低优先级埋点，失败不打断位置查看、导航或商家跳转。
  })
}
```

存储键固定为 `wplink_map_visitor_key`。

- [ ] **步骤 4：在目录、商家主页和位置页接入七类事件**

目录和商家主页在真正调用 `uni.navigateTo` 前记录 `location_entry_click`，来源分别为 `directory`、`merchant_detail`；上下文成功且坐标有效后记录 `location_view`；点击导航前记录 `navigation_click`；抽屉确实从关闭变打开后记录 `nearby_drawer_open`；Marker、周边列表项和查看商家分别记录 `nearby_marker_click`、`nearby_list_item_click`、`nearby_merchant_click` 并携带目标商家 ID。Marker 触发的内部列表联动不得混记 `nearby_list_item_click`。位置页内事件来源统一为 `merchant_location`。不得等待埋点 Promise 后再导航或跳转。

- [ ] **步骤 5：运行测试确认 GREEN**

```bash
cd wxapp && node --test common/merchantMapAnalytics.test.mjs pages/merchant/location.test.mjs pages/sourcing-map/index.test.mjs pages/merchant/detail.test.mjs
```

- [ ] **步骤 6：提交小程序埋点**

```bash
git add wxapp/api/metrics.js wxapp/common/merchantMapAnalytics.js wxapp/common/merchantMapAnalytics.test.mjs wxapp/pages/merchant/location.vue wxapp/pages/merchant/location.test.mjs wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs wxapp/pages/merchant/detail.vue wxapp/pages/merchant/detail.test.mjs
git commit -m "feat: 接入商家地图行为埋点"
```

## Task 9：全量验证与真机验收

**文件：**

- 检查：本计划涉及的全部文件。
- 修改：`docs/superpowers/plans/2026-08-01-wxapp-merchant-location-nearby.md`（执行时勾选完成项）。

- [x] **步骤 1：格式、契约和差异检查**

```bash
cd backend
gofmt -w app/internal/model/map_model.go app/internal/model/map_model_test.go app/internal/logic/map/public_logic.go app/internal/logic/map/public_logic_test.go app/internal/server/map_routes.go app/internal/server/map_api_test.go app/internal/model/merchant_map_events_model.go app/internal/logic/metrics/record_merchant_map_event_logic.go app/internal/logic/metrics/record_merchant_map_event_logic_test.go app/internal/server/api.go app/internal/server/resource_api_test.go app/internal/svc/service_context.go
goctl api validate --api app/api/app.api
git diff --check
```

2026-08-01 fresh 验证：`gofmt`、`goctl api validate`、`git diff --check` 均退出 0，`gofmt` 未产生文件差异。

- [x] **步骤 2：后端全量测试**

```bash
cd backend
go test ./...
node --test scripts/*.test.mjs
```

2026-08-01 fresh 验证：原样 `go test ./...` 因 sandbox Go cache、缺少 ignored `etc/app.yaml`、不完整 admin embed 产物和 `httptest` 监听限制退出 1；将 Go 临时目录指向 `/private/tmp`、临时补齐配置与 embed 产物并允许本机随机端口后，35 个 Go 包全部通过（31 个包有测试、4 个包无测试），命令退出 0。`node --test scripts/*.test.mjs` 为 45/45 通过，退出 0。临时文件均已清理。

- [ ] **步骤 3：数据库 migration 集成验证**

```bash
cd backend && go run ./scripts/verify_migrations.go -config etc/app.yaml
```

预期：临时数据库完成全部 up/down 和演示数据重复导入，日志中的 DSN 已脱敏。

2026-08-01 待处理：已在 `--rm`、数据目录 `tmpfs`、仅绑定 `127.0.0.1` 随机端口的临时 PostgreSQL 16 中执行，命令退出 1。旧 `000003_seed_zhili.down.sql` 删除 `city_stations` 时被既有 `banner_topics_city_station_id_fkey` 阻止，尚未执行到本功能新增的 `000033`；容器已删除。该项没有通过，不勾选。

- [x] **步骤 4：小程序全量验证与构建**

```bash
cd wxapp && npm run check
```

2026-08-01 fresh 验证：退出 0；页面与流程校验通过，322/322 测试通过，微信小程序构建完成。存在 1 条 Node VM Modules 实验性 warning、36 条 Sass `legacy-js-api` 和 1 条 Sass `@import` 弃用 warning，并非 0 warning。

- [ ] **步骤 5：微信真机验收（待微信开发者工具/真机验收）**

1. 一级 Tab 和首页入口显示“拿货档口”，默认只出现商家列表。
2. 已入驻商家进入独立位置页，待认领档口仍可直接导航。
3. 当前商家橙色 Marker 和名称常驻，周边 Marker 明显弱化。
4. 位置页不弹出用户定位授权。
5. 周边抽屉只展示 1 公里内最多 20 家已入驻商家，顺序由近到远。
6. Marker 点击定位列表项；列表点击后当前和周边 Marker 同时在视野内。
7. “查看商家”进入周边商家主页；返回后原地图上下文不变。
8. 导航失败、上下文失败、周边失败、无结果和无坐标均有规定的中文反馈。
9. 断网或埋点失败不阻断地图、导航和商家跳转。

当前命令行环境无法证明以上 9 项真机行为，均保持待验收，不声称已完成。

- [x] **步骤 6：最终范围审查**

```bash
git status --short
git diff --check
git diff --stat
```

确认未删除旧 Canvas 文件，未引入地图搜索、周边筛选、聚合、用户定位或无关构建产物。

2026-08-01 fresh 审查：工作树无业务差异，`git diff --check` 退出 0；旧 `legacy-canvas.vue` 和 Canvas/几何辅助文件仍受 Git 跟踪且相对计划基点未修改。位置页与一级目录未出现 `uni.getLocation`、`show-location`、地图区域搜索、周边筛选、Marker 聚合或路线规划；未发现无关构建产物。

- [x] **步骤 7：提交计划验证状态**

```bash
git add docs/superpowers/plans/2026-08-01-wxapp-merchant-location-nearby.md
git commit -m "docs: 更新档口位置实施计划验证状态"
```
