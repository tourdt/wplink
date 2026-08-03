# 供需详情标签与同类推荐恢复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 恢复需求/供应详情中有意义的展示标签，并让公开详情与本人详情都能稳定展示按相似度排序的公开同类供需。

**架构：** 后端继续拥有展示标签规则，并新增独立的相关推荐只读接口；推荐查询在数据库中按同类型、同业务分组、同供需方向分层排序。小程序只消费 `presentation.tags` 与相关推荐接口，详情辅助请求使用独立容错，任何一个失败都不阻断其他区域。

**技术栈：** Go 1.25、go-zero API 定义、PostgreSQL、Vue 3 / uni-app、Node.js `node:test`。

## 全局约束

- 原始详情 `tags` 必须保持原值与顺序，只允许 `presentation.tags` 使用规范化展示值。
- 展示标签只过滤与页面可见 `typeName` 完全相同的项；不得再按 `category` 或包含关系删除标签。
- 没有特征标签时，以非空且不等于 `typeName` 的 `category` 作为唯一兜底标签。
- 推荐结果只能包含公开发布、未删除、未过期且商家状态有效的资源，并排除当前资源。
- 已发布源资源允许匿名请求；未发布源资源必须携带有效用户凭证，且用户能够管理源资源所属商家。
- 未发布源资源鉴权失败时统一返回“资源不存在或暂不可查看”，不得暴露源资源是否存在、状态、类型或所属商家。
- 推荐顺序固定为：相同 `typeCode`、相同 `displayTemplate.group.code`、相同 `direction`，每层沿用置顶及刷新时间排序。
- 推荐默认 3 条，最大 6 条；结果必须稳定去重。
- 公开详情与本人详情都加载推荐；浏览统计、收藏、商家资料及推荐彼此失败隔离。
- API 生成类型以 `.api` 为真源；只同步目标 goctl 生成类型，不覆盖仓库内无关类型。
- 新增业务注释使用中文；不得向用户暴露数据库或内部错误详情。

---

### Task 1：恢复展示标签语义并补齐发布标签配置

**文件：**
- 修改：`backend/app/internal/logic/resource/get_resource_logic.go`
- 修改：`backend/app/internal/logic/resource/get_resource_logic_test.go`
- 修改：`backend/migrations/000003_seed_zhili.up.sql`
- 修改：`backend/scripts/validate_migrations.test.mjs`

**接口：**
- 消费：`filterResourcePresentationTags(tags []string, typeName string, category string) []string`
- 产出：稳定、非空且不过度过滤的 `ResourcePresentation.Tags`
- 产出：28 个预置活跃供需类型均具有 `fieldSchema.tagOptions` 与 `maxTags: 8`

- [ ] **Step 1：编写展示标签回归测试**

在 `get_resource_logic_test.go` 将标签测试改为表驱动，并至少包含以下实际断言：

```go
tests := []struct {
    name     string
    tags     []string
    typeName string
    category string
    want     []string
}{
    {name: "only exact visible type is removed", tags: []string{"求购尾货", "急采"}, typeName: "求购尾货", category: "童装", want: []string{"急采"}},
    {name: "category remains when it is not the visible type", tags: []string{"童装"}, typeName: "求购尾货", category: "童装", want: []string{"童装"}},
    {name: "contained feature remains", tags: []string{"尾货", "可接受断码"}, typeName: "求购尾货", category: "童装尾货", want: []string{"尾货", "可接受断码"}},
    {name: "category is fallback for legacy empty tags", tags: nil, typeName: "求购尾货", category: "童装", want: []string{"童装"}},
    {name: "no duplicate fallback", tags: nil, typeName: "童装", category: "童装", want: []string{}},
}
```

保留 mapper 级断言，证明 `resp.Tags` 仍为原始标签，`resp.Presentation.Tags` 为展示标签。

- [ ] **Step 2：运行 Go 测试并确认 RED**

运行：

```bash
cd backend
mkdir -p /private/tmp/wplink-detail-recovery-go-cache /private/tmp/wplink-detail-recovery-go-tmp
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/logic/resource
```

预期：现有包含关系/分类过滤导致至少三个新用例失败。

- [ ] **Step 3：实现最小标签规则**

在 `filterResourcePresentationTags` 中按以下顺序处理：

```go
for _, rawTag := range tags {
    tag := strings.TrimSpace(rawTag)
    if tag == "" || tag == strings.TrimSpace(typeName) {
        continue
    }
    if _, exists := seen[tag]; exists {
        continue
    }
    seen[tag] = struct{}{}
    filtered = append(filtered, tag)
}
fallback := strings.TrimSpace(category)
if len(filtered) == 0 && fallback != "" && fallback != strings.TrimSpace(typeName) {
    filtered = append(filtered, fallback)
}
```

删除按 `category` 或 `strings.Contains` 过滤的分支，并保留“不改写原始标签”的中文注释。

- [ ] **Step 4：运行 Go 测试并确认 GREEN**

运行 Step 2 的命令，预期资源逻辑包全部通过。

- [ ] **Step 5：编写预置类型标签配置测试**

在 `validate_migrations.test.mjs` 增加测试，解析 `000003_seed_zhili.up.sql` 并断言以下 28 个类型最终都出现在标签配置 `VALUES` 中，且配置包含非空 `tagOptions` 与 `"maxTags":8`：

```js
const expectedTypeCodes = [
  'factory_direct', 'spot_wholesale', 'stock_clearance', 'buy_kids_goods',
  'fabric_supply', 'accessory_supply', 'processing_accept', 'find_factory',
  'production_support', 'job_hiring', 'job_seeking', 'sample_rental',
  'shop_office_rental', 'shop_sale', 'seek_shop_office', 'apartment_rental',
  'housing_sale', 'seek_housing', 'factory_warehouse_rental', 'workshop_rental',
  'factory_sale', 'seek_factory_warehouse', 'secondhand_sale', 'secondhand_buy',
  'education_training', 'appliance_repair', 'moving_cleaning', 'other_local_service',
]
```

- [ ] **Step 6：运行迁移测试并确认 RED**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
```

预期：当前仅 10 个类型配置标签，缺失类型断言失败。

- [ ] **Step 7：补齐剩余类型的标签选项**

扩展 `000003_seed_zhili.up.sql` 中现有标签配置更新，使用以下精确配置；现有 10 个租售类型保留原配置：

```text
factory_direct: 源头工厂、小单快反、来样定制、长期供货、现货、支持拿样
spot_wholesale: 现货、可混批、支持拿样、整包、断码、尾季
stock_clearance: 现货、可混批、支持看货、整包、断码、轻瑕
buy_kids_goods: 急采、可接受断码、可接受尾货、长期采购、预算明确、可自提
fabric_supply: 现货、可寄样、支持定制、小单起订、颜色齐全、长期供货
accessory_supply: 现货、可寄样、支持定制、小单起订、款式齐全、长期供货
processing_accept: 小单快反、来样加工、包工包料、可打样、交期稳定、长期合作
find_factory: 急找工厂、小单优先、包工包料、来样加工、长期合作、交期明确
production_support: 上门服务、快速响应、长期合作、可开票、支持定制
job_hiring: 急招、包吃住、计件工资、长期招聘、熟练工优先
job_seeking: 随时到岗、熟练工、接受计件、可长期、可夜班
shop_sale: 一楼临街、交通便利、带车位、靠近商圈、产权清晰、可议价
secondhand_sale: 现货、成色良好、可现场看、可议价、支持自提
secondhand_buy: 急购、可接受二手、预算明确、可自提、长期求购
education_training: 小班教学、上门辅导、可试听、可预约、长期招生
appliance_repair: 上门维修、快速响应、可预约、配件齐全、可开票
moving_cleaning: 上门服务、快速响应、可预约、可开票、周末可约
other_local_service: 上门服务、快速响应、可预约、可开票、长期服务
```

所有新增配置使用 `maxTags: 8`。不回填现有 `resources.tags`，旧数据由分类兜底处理。

- [ ] **Step 8：运行迁移与资源逻辑测试并确认 GREEN**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/logic/resource
```

预期：全部通过。

- [ ] **Step 9：格式检查并提交**

```bash
cd backend
gofmt -w app/internal/logic/resource/get_resource_logic.go app/internal/logic/resource/get_resource_logic_test.go
cd ..
git diff --check
git add backend/app/internal/logic/resource/get_resource_logic.go backend/app/internal/logic/resource/get_resource_logic_test.go backend/migrations/000003_seed_zhili.up.sql backend/scripts/validate_migrations.test.mjs
git commit -m "fix: restore resource detail presentation tags"
```

---

### Task 2：新增后端同类供需推荐接口

**文件：**
- 新增：`backend/app/internal/logic/resource/list_related_resources_logic.go`
- 新增：`backend/app/internal/logic/resource/list_related_resources_logic_test.go`
- 修改：`backend/app/internal/model/resource_model.go`
- 修改：`backend/app/internal/model/resource_model_test.go`
- 修改：`backend/app/api/resource.api`
- 修改：`backend/app/internal/types/types.go`（通过 goctl 临时生成结果同步目标类型）
- 修改：`backend/app/internal/server/api.go`
- 修改：`backend/app/internal/server/resource_api_test.go`
- 修改：`backend/scripts/api_contract.test.mjs`

**接口：**
- 产出源上下文接口：`GetRelatedResourceSource(ctx context.Context, resourceID string) (model.RelatedResourceSource, error)`，仅包含鉴权及推荐所需的 `Status`、`MerchantID`、`TypeCode`、`Direction`、`GroupCode`。
- 产出模型接口：`ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]model.ResourceListItem, error)`
- 产出逻辑接口：`ListRelatedResources(ctx context.Context, resourceID string, req RelatedResourcesReq) (RelatedResourcesResp, error)`
- 产出 HTTP 接口：`GET /api/v1/resources/:resourceId/related?pageSize=3`
- 响应：`RelatedResourcesResp{Items []ResourceListItem}`

- [ ] **Step 1：编写模型 SQL 结构测试**

在 `resource_model_test.go` 增加 `TestListRelatedResourcesSQLRanksAndFiltersCandidates`，断言新 SQL 包含：

```go
required := []string{
    "candidate.id <> source.id",
    "candidate.status = 'published'",
    "merchant.status = 'active'",
    "candidate.type_code = source.type_code THEN 1",
    "candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code THEN 2",
    "candidate.direction = source.direction THEN 3",
    "candidate.expires_at IS NULL OR candidate.expires_at > now()",
    "LIMIT $2",
}
```

同时断言 SQL 不要求源资源为 `published`，使通过鉴权的本人私有详情能够以其类型快照寻找公开候选；接口响应不得返回源资源字段。增加源上下文 SQL 测试，确认只读取资源状态、所属商家和推荐计算所需字段，不返回描述、联系方式等私有内容。

- [ ] **Step 2：运行模型测试并确认 RED**

```bash
cd backend
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/model
```

预期：`listRelatedResourcesSQL` 尚不存在导致编译失败。

- [ ] **Step 3：实现单查询推荐模型**

在 `resource_model.go` 定义 `RelatedResourceSource`、`getRelatedResourceSourceSQL`、`listRelatedResourcesSQL` 与对应方法。先由最小源上下文查询支持路由鉴权；推荐 SQL 使用 `source` CTE 获取当前资源的 `type_code`、`direction`、`displayTemplate.group.code`，候选只选择公开有效资源；用 `CASE` 生成优先级并按以下顺序排序：

```sql
ORDER BY
  similarity_priority ASC,
  CASE WHEN candidate.top_expires_at IS NOT NULL AND candidate.top_expires_at > now() THEN 1 ELSE 0 END DESC,
  COALESCE(candidate.refreshed_at, candidate.published_at, candidate.created_at) DESC,
  candidate.id DESC
LIMIT $2
```

扫描结果复用 `model.ResourceListItem` 字段结构；没有源资源时返回空切片，不返回内部 SQL 错误给前端。

- [ ] **Step 4：运行模型测试并确认 GREEN**

运行 Step 2 命令，预期模型包全部通过。

- [ ] **Step 5：编写推荐逻辑 RED 测试**

新增 `list_related_resources_logic_test.go`，使用 fake store 验证：

```go
func TestListRelatedResourcesUsesDefaultAndMaximumPageSize(t *testing.T)
func TestListRelatedResourcesMapsRankedItemsWithoutChangingOrder(t *testing.T)
func TestListRelatedResourcesReturnsEmptyItemsInsteadOfNil(t *testing.T)
```

默认 `pageSize` 传给 store 为 3；请求 20 时传 6；响应保持 store 排序，`Items` 必须为非 nil 空数组。

- [ ] **Step 6：运行逻辑测试并确认 RED**

```bash
cd backend
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/logic/resource
```

预期：相关逻辑类型和构造函数未定义导致编译失败。

- [ ] **Step 7：实现推荐逻辑与共享列表映射**

新增：

```go
type RelatedResourcesStore interface {
    ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]model.ResourceListItem, error)
}

type RelatedResourcesReq struct {
    PageSize int64
}

type RelatedResourcesResp struct {
    Items []ResourceListItem `json:"items"`
}
```

将 `list_resources_logic.go` 中单项映射提取为同包私有函数 `resourceListItemFromModel(item model.ResourceListItem) ResourceListItem`，列表与推荐共同使用。推荐查询失败时记录包含 `resourceId` 与根错误的中文诊断日志，并返回安全错误“相关推荐加载失败，请稍后重试”。

- [ ] **Step 8：运行资源逻辑测试并确认 GREEN**

运行 Step 6 命令，预期全部通过。

- [ ] **Step 9：先编写 API 契约与路由 RED 测试**

在 `resource_api_test.go` 增加：

```go
func TestResourceAPIRouterListsRelatedResources(t *testing.T)
func TestResourceAPIRouterListsRelatedResourcesForPrivateSourceOwner(t *testing.T)
func TestResourceAPIRouterHidesPrivateRelatedSourceFromAnonymousAndNonOwner(t *testing.T)
```

公开源测试匿名请求 `/api/v1/resources/resource-1/related?pageSize=3`，断言 fake store 收到 `resource-1` 和限制 3，响应 `items` 保持推荐顺序。私有源测试分别覆盖未登录、非所属商家管理者与所属商家管理者；前两者统一返回“资源不存在或暂不可查看”，且不得调用候选查询，管理者请求成功。扩展 fake store 实现 `GetRelatedResourceSource`、`ListRelatedResources`，并复用现有 `UserCanManageMerchant` fake 能力。

在 `backend/scripts/api_contract.test.mjs` 的资源 API 契约测试中断言 `.api` 包含 `RelatedResourcesReq`、`RelatedResourcesResp` 与路由。

- [ ] **Step 10：运行 API 测试并确认 RED**

```bash
cd backend
node --test scripts/api_contract.test.mjs
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/server
```

预期：路由与 API 类型尚未定义导致失败。

- [ ] **Step 11：增加 `.api` 契约并通过 goctl 同步目标类型**

在 `resource.api` 增加：

```go
type RelatedResourcesReq {
    PageSize int64 `form:"pageSize,optional"`
}

type RelatedResourcesResp {
    Items []ResourceListItem `json:"items"`
}
```

以及：

```go
@handler ListRelatedResources
get /resources/:resourceId/related (RelatedResourcesReq) returns (RelatedResourcesResp)
```

先运行 `goctl api validate --api app/api/app.api`。生成时复制 `resource.api` 与其共享 `vip.api` 到临时目录，在临时 `resource.api` 加入 `import "vip.api"` 后执行 `goctl api go`；只将生成结果中的两个目标类型同步到 `backend/app/internal/types/types.go`，禁止覆盖整个文件。

- [ ] **Step 12：注册自定义服务器路由**

在 `ResourceAPIStore` 嵌入 `resourcelogic.RelatedResourcesStore`，在公开资源详情路由附近增加 GET handler：

1. 读取 `resourceId`、`pageSize`，通过 `GetRelatedResourceSource` 获取最小上下文；不存在时返回统一不可查看错误。
2. 源资源为 `published` 时直接继续，允许匿名请求。
3. 源资源非 `published` 时解析有效用户凭证并调用 `UserCanManageMerchant`；未登录、凭证无效或无管理权限均返回“资源不存在或暂不可查看”，不执行候选查询。
4. 鉴权通过后调用 `NewListRelatedResourcesLogic(store).ListRelatedResources`，通过现有 `response.JSON` 返回。

读取源上下文或权限查询出现内部错误时记录包含 `resourceId` 与根错误的中文诊断日志，对前端只返回安全错误。禁止复用会在依赖为空时放行的管理接口辅助函数。

- [ ] **Step 13：运行 API 与后端目标测试并确认 GREEN**

```bash
cd backend
goctl api validate --api app/api/app.api
node --test scripts/api_contract.test.mjs
GOCACHE=/private/tmp/wplink-detail-recovery-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-go-tmp go test ./app/internal/model ./app/internal/logic/resource ./app/internal/server
```

预期：全部通过。

- [ ] **Step 14：格式检查并提交**

```bash
cd backend
gofmt -w app/internal/model/resource_model.go app/internal/model/resource_model_test.go app/internal/logic/resource/list_resources_logic.go app/internal/logic/resource/list_related_resources_logic.go app/internal/logic/resource/list_related_resources_logic_test.go app/internal/server/api.go app/internal/server/resource_api_test.go
cd ..
git diff --check
git add backend/app/api/resource.api backend/app/internal/types/types.go backend/app/internal/model/resource_model.go backend/app/internal/model/resource_model_test.go backend/app/internal/logic/resource/list_resources_logic.go backend/app/internal/logic/resource/list_related_resources_logic.go backend/app/internal/logic/resource/list_related_resources_logic_test.go backend/app/internal/server/api.go backend/app/internal/server/resource_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "feat: add related resource recommendations"
```

---

### Task 3：接入详情页推荐并隔离辅助请求失败

**文件：**
- 修改：`wxapp/api/resource.js`
- 修改：`wxapp/api/resource.test.mjs`
- 修改：`wxapp/pages/resource/detail.vue`
- 修改：`wxapp/pages/resource/detail.test.mjs`
- 修改：`wxapp/scripts/validate-flows.test.mjs`

**接口：**
- 消费：`GET /api/v1/resources/:resourceId/related?pageSize=3`
- 产出前端函数：`listRelatedResources(resourceId, params = {}, options = {})`
- 页面状态：`relatedResources` 始终为数组，公开详情与本人详情均加载

- [ ] **Step 1：编写 API 行为 RED 测试**

在 `resource.test.mjs` 增加真实模块测试：

```js
test('related resource API encodes resource id and standardizes missing items', async () => {
  const calls = []
  const api = await loadResourceApi(async (options) => {
    calls.push(options)
    return {}
  })

  const result = await api.listRelatedResources(
    'resource/1',
    { pageSize: 3 },
    { suppressErrorToast: true },
  )

  assert.deepEqual(result, { items: [] })
  assert.deepEqual(calls, [{
    url: '/api/v1/resources/resource%2F1/related',
    method: 'GET',
    data: { pageSize: 3 },
    suppressErrorToast: true,
  }])
})
```

- [ ] **Step 2：运行 API 测试并确认 RED**

```bash
cd wxapp
node --experimental-vm-modules --test api/resource.test.mjs
```

预期：`listRelatedResources` 尚未导出导致失败。

- [ ] **Step 3：实现推荐 API 封装**

在 `resource.js` 增加：

```js
export function listRelatedResources(resourceId, params = {}, options = {}) {
  const encodedId = encodeURIComponent(String(resourceId || '').trim())
  return request({
    url: `/api/v1/resources/${encodedId}/related`,
    method: 'GET',
    data: params,
    ...options,
  }).then((response) => ({
    ...response,
    items: Array.isArray(response?.items) ? response.items : [],
  }))
}
```

- [ ] **Step 4：运行 API 测试并确认 GREEN**

运行 Step 2 命令，预期通过。

- [ ] **Step 5：编写详情生命周期与渲染 RED 测试**

在 `detail.test.mjs` 与 `validate-flows.test.mjs` 增加约束：

```js
assert.match(source, /listRelatedResources/)
assert.doesNotMatch(source, /listResources\(\{ typeCode: resource\.value\.typeCode/)
assert.match(source, /relatedResources\.value = \[\]/)
assert.match(source, /Promise\.allSettled/)
assert.match(source, /loadRelatedResources\(options\.id\)/)
```

同时截取 `if (!isOwnResource.value)` 代码块，断言其中只包含公开详情专属的浏览统计与收藏，不包含 `loadRelatedResources`；现有 `v-if="relatedResources.length"` 与 `ResourceList variant="feed"` 断言保留。

- [ ] **Step 6：运行页面测试并确认 RED**

```bash
cd wxapp
node --experimental-vm-modules --test pages/resource/detail.test.mjs scripts/validate-flows.test.mjs
```

预期：页面仍调用 `listResources`，推荐仍位于本人详情排除分支，断言失败。

- [ ] **Step 7：改造详情辅助加载流程**

将 `listResources` import 替换为 `listRelatedResources`。详情主体成功后：

```js
const auxiliaryTasks = [
  loadMerchantProfile(),
  loadRelatedResources(options.id),
]
if (!isOwnResource.value) {
  auxiliaryTasks.push(recordResourceDetailView(options.id), loadFavoriteState(options.id))
}
await Promise.allSettled(auxiliaryTasks)
```

分享菜单和封面调度不依赖这些辅助请求。`loadRelatedResources(resourceId)` 必须先清空旧状态，再使用 `{ pageSize: 3 }` 与 `{ suppressErrorToast: true, requireAuth: isOwnResource.value }` 调用 API：本人详情明确携带登录凭证，公开详情保持可匿名；捕获失败后保持空数组且不弹全局错误。删除按 `typeCode` 调用公开列表的旧实现。

- [ ] **Step 8：运行页面与 API 测试并确认 GREEN**

```bash
cd wxapp
node --experimental-vm-modules --test api/resource.test.mjs pages/resource/detail.test.mjs scripts/validate-flows.test.mjs
```

预期：全部通过。

- [ ] **Step 9：运行小程序完整检查并提交**

```bash
cd wxapp
npm run check
cd ..
git diff --check
git add wxapp/api/resource.js wxapp/api/resource.test.mjs wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs wxapp/scripts/validate-flows.test.mjs
git commit -m "fix: restore resource detail related recommendations"
```

---

## 最终验证

在全部任务提交后运行：

```bash
cd backend
mkdir -p /private/tmp/wplink-detail-recovery-all-go-cache /private/tmp/wplink-detail-recovery-all-go-tmp
GOCACHE=/private/tmp/wplink-detail-recovery-all-go-cache GOTMPDIR=/private/tmp/wplink-detail-recovery-all-go-tmp go test ./app/...
node --test scripts/*.test.mjs
cd ../wxapp
npm run check
cd ..
git diff --check
git status --short
```

验收时人工核对四条数据路径：

1. 旧需求无原始标签时显示分类兜底标签。
2. 新需求存在真实特征标签时保留特征标签，仅去掉与可见类型完全相同的标签。
3. 从市场与“我的发布”进入详情都发起同类推荐请求。
4. 同类型不足时，接口按同分组、同方向补足至最多 3 条；辅助请求失败不影响详情主体。
