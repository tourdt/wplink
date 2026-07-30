# 完整调试演示数据实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `seed_demo_data.sql` 完善为可被 `reset-test-db.sh` 稳定导入的完整供需调试集，覆盖当前全部类型、方向、状态和关键关联数据。

**Architecture:** 保留单文件 SQL 种子和现有重置流程，用固定 TSID、显式场景行及 `ON CONFLICT` 保证可读性与幂等性。先逐步增强静态校验并观察预期失败，再补齐资源配置快照、28 类型矩阵、生命周期数据和关联数据，最后使用现有迁移验证器在临时 PostgreSQL 数据库完成真实导入验证。

**Tech Stack:** PostgreSQL 14+、SQL、Node.js ESM、Node.js `assert`、Go 1.23 迁移验证器。

## Global Constraints

- 所有实施计划、设计文档和说明使用中文；代码标识、路径和命令保留英文。
- 只修改演示种子及其校验，不修改 API、小程序页面、数据库表结构或迁移。
- 继续使用 `backend/scripts/seed_demo_data.sql` 作为唯一演示种子文件。
- 全部 SQL 保持在单个 `BEGIN`/`COMMIT` 事务中并可重复执行。
- 当前 26 个启用类型必须各有至少一条公开数据；停用的 `job_hiring`、`job_seeking` 不得公开。
- 资源状态必须覆盖 `draft`、`pending`、`manual_review`、`audit_retry`、`published`、`rejected`、`taken_down`、`expired`。
- 资源必须保存完整的类型配置 ID、版本和快照。
- 演示联系信息只使用固定测试号码和测试微信号，不写入真实敏感信息。
- 不执行远程 `reset-test-db.sh`；真实数据库验证只使用验证器创建并自动删除的临时数据库。

---

## 文件职责

- `backend/scripts/validate_demo_seed.mjs`：解析资源场景元组，校验资源必填配置字段、类型/方向/状态矩阵、特殊生命周期和关键关联。
- `backend/scripts/seed_demo_data.sql`：创建用户、商家、完整供需场景及审核、指标、联系、消息、权益和地图数据。
- `backend/scripts/verify_migrations.go`：在临时数据库按顺序执行迁移，并连续导入两次演示种子以验证幂等性。
- `backend/scripts/verify_migrations_test.go`：验证演示种子导入文件序列确实包含连续两次种子执行。
- `docs/superpowers/specs/2026-07-30-complete-demo-seed-design.md`：已确认的需求和验收标准，只作为实现依据，不再修改。

### Task 1: 修复资源配置快照并建立失败校验

**Files:**
- Modify: `backend/scripts/validate_demo_seed.mjs`
- Modify: `backend/scripts/seed_demo_data.sql:155-251`

**Interfaces:**
- Consumes: `resource_type_configs` 的 `id`、`version`、`type_name`、`direction`、schema、模板、规则和有效期字段。
- Produces: 每条 `resources` 数据都具备 `resource_type_config_version` 和 `resource_type_snapshot`，后续列表和详情可直接读取历史快照。

- [ ] **Step 1: 在校验脚本增加资源配置字段断言**

在 `validate_demo_seed.mjs` 读取 `sql` 后加入以下断言：

```js
const resourceInsertStart = sql.indexOf('INSERT INTO resources (')
const resourceInsertEnd = sql.indexOf('INSERT INTO verifications', resourceInsertStart)
assert(resourceInsertStart >= 0 && resourceInsertEnd > resourceInsertStart, '缺少完整的 resources 演示数据区块')

const resourceSql = sql.slice(resourceInsertStart, resourceInsertEnd)
for (const column of [
  'resource_type_config_id',
  'resource_type_config_version',
  'resource_type_snapshot',
  'type_code',
  'direction',
]) {
  assert(resourceSql.includes(column), `resources 演示数据缺少类型配置字段: ${column}`)
}

for (const snapshotKey of [
  "'version'",
  "'typeCode'",
  "'typeName'",
  "'direction'",
  "'fieldSchema'",
  "'requiredFields'",
  "'filterFields'",
  "'displayTemplate'",
  "'reviewRules'",
  "'sortWeights'",
  "'messageRules'",
  "'commercialRules'",
  "'defaultValidDays'",
]) {
  assert(resourceSql.includes(snapshotKey), `资源类型快照缺少字段: ${snapshotKey}`)
}

for (const configMapping of [
  'rtc.id',
  'rtc.version',
  'rtc.type_code',
  'rtc.type_name',
  'rtc.direction',
  'rtc.field_schema',
  'rtc.display_template',
  'rtc.commercial_rules',
  'rtc.default_valid_days',
]) {
  assert(resourceSql.includes(configMapping), `资源数据未从当前类型配置取值: ${configMapping}`)
}

for (const conflictUpdate of [
  'resource_type_config_id = EXCLUDED.resource_type_config_id',
  'resource_type_config_version = EXCLUDED.resource_type_config_version',
  'resource_type_snapshot = EXCLUDED.resource_type_snapshot',
  'direction = EXCLUDED.direction',
]) {
  assert(resourceSql.includes(conflictUpdate), `资源重复导入未更新配置字段: ${conflictUpdate}`)
}
```

同时在文件顶部添加：

```js
import assert from 'node:assert/strict'
```

- [ ] **Step 2: 运行静态校验并确认按预期失败**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
```

Expected: FAIL，首个失败信息为 `resources 演示数据缺少类型配置字段: resource_type_config_version`。

- [ ] **Step 3: 扩展资源插入列并从类型配置构建快照**

在 `INSERT INTO resources` 和对应 `SELECT` 中依次补入：

```sql
resource_type_config_version,
resource_type_snapshot,
```

对应值为：

```sql
rtc.version,
jsonb_build_object(
  'version', rtc.version,
  'typeCode', rtc.type_code,
  'typeName', rtc.type_name,
  'direction', rtc.direction,
  'fieldSchema', rtc.field_schema,
  'requiredFields', rtc.required_fields,
  'filterFields', rtc.filter_fields,
  'displayTemplate', rtc.display_template,
  'reviewRules', rtc.review_rules,
  'sortWeights', rtc.sort_weights,
  'messageRules', rtc.message_rules,
  'commercialRules', rtc.commercial_rules,
  'defaultValidDays', rtc.default_valid_days
),
```

在 `ON CONFLICT (id) DO UPDATE SET` 中加入：

```sql
resource_type_config_version = EXCLUDED.resource_type_config_version,
resource_type_snapshot = EXCLUDED.resource_type_snapshot,
```

- [ ] **Step 4: 运行校验并确认配置快照校验通过**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
```

Expected: PASS，输出 `demo seed static check ok`。

- [ ] **Step 5: 提交配置快照修复**

```bash
git add backend/scripts/validate_demo_seed.mjs backend/scripts/seed_demo_data.sql
git commit -m "fix: 补齐演示资源类型配置快照"
```

### Task 2: 建立完整类型矩阵和演示商家角色

**Files:**
- Modify: `backend/scripts/validate_demo_seed.mjs`
- Modify: `backend/scripts/seed_demo_data.sql:6-251`

**Interfaces:**
- Consumes: 迁移生成的 28 个 `resource_type_configs` 以及固定城市代码 `zhili`。
- Produces: 28 条类型基准场景，其中 26 条启用类型为公开资源，招聘和求职为非公开历史资源。

- [ ] **Step 1: 增加结构化资源场景解析和类型矩阵校验**

在 `validate_demo_seed.mjs` 中定义准确矩阵：

```js
const enabledSupplyTypes = [
  'factory_direct',
  'spot_wholesale',
  'stock_clearance',
  'fabric_supply',
  'accessory_supply',
  'processing_accept',
  'production_support',
  'sample_rental',
  'shop_office_rental',
  'shop_sale',
  'apartment_rental',
  'housing_sale',
  'factory_warehouse_rental',
  'workshop_rental',
  'factory_sale',
  'secondhand_sale',
  'education_training',
  'appliance_repair',
  'moving_cleaning',
  'other_local_service',
]

const enabledDemandTypes = [
  'buy_kids_goods',
  'find_factory',
  'seek_shop_office',
  'seek_housing',
  'seek_factory_warehouse',
  'secondhand_buy',
]

const disabledHistoricalTypes = ['job_hiring', 'job_seeking']
const expectedTypes = [...enabledSupplyTypes, ...enabledDemandTypes, ...disabledHistoricalTypes]

function parseResourceScenarios(resourceSql) {
  return [...resourceSql.matchAll(
    /^\\s*\\((803\\d+),\\s*(802\\d+),\\s*'([^']+)',\\s*'([^']+)',\\s*'([^']+)'/gm,
  )].map((match) => ({
    id: match[1],
    merchantId: match[2],
    typeCode: match[3],
    status: match[4],
    title: match[5],
  }))
}

const scenarios = parseResourceScenarios(resourceSql)
assert(scenarios.length >= 28, `供需演示场景偏少: ${scenarios.length}, want >= 28`)
assert.equal(new Set(scenarios.map((item) => item.id)).size, scenarios.length, '供需演示场景 ID 重复')

for (const typeCode of expectedTypes) {
  assert(scenarios.some((item) => item.typeCode === typeCode), `演示种子缺少供需类型: ${typeCode}`)
}
for (const typeCode of [...enabledSupplyTypes, ...enabledDemandTypes]) {
  assert(
    scenarios.some((item) => item.typeCode === typeCode && item.status === 'published'),
    `启用供需类型缺少公开样例: ${typeCode}`,
  )
}
for (const typeCode of disabledHistoricalTypes) {
  assert(
    !scenarios.some((item) => item.typeCode === typeCode && item.status === 'published'),
    `停用供需类型不应包含公开样例: ${typeCode}`,
  )
}
```

- [ ] **Step 2: 运行校验并确认缺少类型**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
```

Expected: FAIL，提示 `演示种子缺少供需类型: spot_wholesale`。

- [ ] **Step 3: 扩展演示用户、商家和管理员绑定**

使用固定演示 ID，形成以下角色：

| userId | merchantId | 商家 | merchantType | 认证状态 |
| --- | --- | --- | --- | --- |
| `8010000000000000002` | `8020000000000000001` | 湖州织里晨星童装厂 | `factory` | `verified` |
| `8010000000000000003` | `8020000000000000002` | 织里云仓尾货 | `stockist` | `verified` |
| `8010000000000000004` | `8020000000000000003` | 织里快印包装服务商 | `service_provider` | `verified` |
| `8010000000000000005` | `8020000000000000004` | 杭州童装采购商 | `buyer` | `unverified` |
| `8010000000000000006` | `8020000000000000005` | 织里面辅料现货中心 | `material_supplier` | `verified` |
| `8010000000000000007` | `8020000000000000006` | 织里产业物业服务中心 | `property_service` | `verified` |

为新增用户使用手机号 `19900000006`、`19900000007` 和 openid `demo_material_admin_openid`、`demo_property_admin_openid`。为采购商及两个新增商家增加激活的 `owner` 绑定，绑定 ID 依次使用 `8021000000000000004` 至 `8021000000000000006`。

- [ ] **Step 4: 用 28 条显式基准场景替换旧资源 VALUES**

保持每条元组前五列固定为 `id, merchant_id, type_code, status, title`，使校验器可稳定解析。按下表创建基准资源：

| resourceId | merchantId | typeCode | status | title |
| --- | --- | --- | --- | --- |
| `8030000000000000001` | `8020000000000000001` | `factory_direct` | `published` | 童装套装源头工厂一件代发 |
| `8030000000000000002` | `8020000000000000002` | `spot_wholesale` | `published` | 夏款童裙尾货整包批发 |
| `8030000000000000003` | `8020000000000000002` | `stock_clearance` | `published` | 女童春款卫衣库存整包清 |
| `8030000000000000004` | `8020000000000000004` | `buy_kids_goods` | `published` | 求购中大童夏款混批尾货 |
| `8030000000000000005` | `8020000000000000005` | `fabric_supply` | `published` | 320 克纯棉卫衣布现货 |
| `8030000000000000006` | `8020000000000000005` | `accessory_supply` | `published` | 童装吊牌洗标当天打样 |
| `8030000000000000007` | `8020000000000000001` | `processing_accept` | `published` | 童装卫衣工厂本周空档接单 |
| `8030000000000000008` | `8020000000000000004` | `find_factory` | `published` | 寻找女童防晒衣加工厂 |
| `8030000000000000009` | `8020000000000000003` | `production_support` | `published` | 童装裁床整烫配套服务 |
| `8030000000000000010` | `8020000000000000001` | `job_hiring` | `taken_down` | 历史平车工招聘信息 |
| `8030000000000000011` | `8020000000000000004` | `job_seeking` | `expired` | 历史熟练车工求职信息 |
| `8030000000000000012` | `8020000000000000003` | `sample_rental` | `published` | 直播童装样衣按周出租 |
| `8030000000000000013` | `8020000000000000006` | `shop_office_rental` | `published` | 童装城一楼沿街商铺出租 |
| `8030000000000000014` | `8020000000000000006` | `shop_sale` | `published` | 利济路成熟童装商铺出售 |
| `8030000000000000015` | `8020000000000000004` | `seek_shop_office` | `published` | 求租童装城一楼直播档口 |
| `8030000000000000016` | `8020000000000000006` | `apartment_rental` | `published` | 织里工厂附近员工公寓出租 |
| `8030000000000000017` | `8020000000000000006` | `housing_sale` | `published` | 织里两居室住宅诚意出售 |
| `8030000000000000018` | `8020000000000000004` | `seek_housing` | `published` | 求租可住六人的员工宿舍 |
| `8030000000000000019` | `8020000000000000006` | `factory_warehouse_rental` | `published` | 童装城旁一楼仓库出租 |
| `8030000000000000020` | `8020000000000000006` | `workshop_rental` | `published` | 带设备缝制加工间出租 |
| `8030000000000000021` | `8020000000000000006` | `factory_sale` | `published` | 织里园区独栋厂房出售 |
| `8030000000000000022` | `8020000000000000004` | `seek_factory_warehouse` | `published` | 求租可进货车的一楼仓库 |
| `8030000000000000023` | `8020000000000000002` | `secondhand_sale` | `published` | 九成新自动裁床设备转让 |
| `8030000000000000024` | `8020000000000000004` | `secondhand_buy` | `published` | 求购二手童装货架二十组 |
| `8030000000000000025` | `8020000000000000003` | `education_training` | `published` | 童装制版与电商运营培训 |
| `8030000000000000026` | `8020000000000000003` | `appliance_repair` | `published` | 缝纫设备上门检修服务 |
| `8030000000000000027` | `8020000000000000003` | `moving_cleaning` | `published` | 仓库搬运与开荒保洁 |
| `8030000000000000028` | `8020000000000000003` | `other_local_service` | `published` | 童装拍摄与广告制作服务 |

每条资源的 `attributes` 必须使用 `000003_seed_zhili.up.sql` 中该类型的真实 field key；`category`、`price_text` 和 `quantity_text` 与该类型 `display_template.summary` 对应值一致。`processing_accept` 设置有效 `top_started_at/top_expires_at`；`stock_clearance` 设置 1 天后过期；`secondhand_sale` 设置 2 天前成交。

各基准类型使用以下确定属性；SQL 中 `category`、`quantity_text`、`price_text` 分别复制该类型摘要映射对应的值：

```js
const baselineAttributes = {
  factory_direct: {"productCategory":"套装","factoryPriceText":"36-48 元/套","minOrderText":"20 套起批","factoryAdvantage":"小单快反","supportsDropship":true,"deliveryArea":"全国"},
  spot_wholesale: {"stockCategory":"裙装","stockQuantityText":"2600 件","wholesalePriceText":"15-22 元/件","season":"夏季","sizeRange":"90-140","allowSample":true},
  stock_clearance: {"clearanceCategory":"女童","stockQuantityText":"3800 件","packagePriceText":"18-26 元/件","stockCondition":"整包","warehouseLocation":"织里童装城 3 区","allowMixedLot":true},
  buy_kids_goods: {"targetCategory":"混款","demandQuantityText":"3000 件","budgetRange":"单件 10-25 元","expectedDelivery":"7 天内","acceptTailStock":true,"purchaseArea":"杭州"},
  fabric_supply: {"fabricType":"卫衣布","fabricComposition":"纯棉","widthWeight":"185cm / 320g","fabricPriceText":"26 元/公斤","inStockText":"现货 8 吨","colorCount":16},
  accessory_supply: {"accessoryType":"吊牌","materialSpec":"350g 白卡覆膜","stockQuantityText":"日产 10 万套","accessoryPriceText":"0.12 元/套起","minOrderText":"1000 套"},
  processing_accept: {"processCategory":"卫衣","dailyCapacity":"日产 1200 件","processingMode":"来料加工","processingPriceText":"按工艺核价","availableSchedule":"本周可排","acceptSmallOrders":true},
  find_factory: {"targetCategory":"童装","orderQuantity":"5000 件","deliveryDeadline":"20 天","processingMode":"包工包料","budgetRange":"面议","sampleRequired":true},
  production_support: {"supportType":"裁床","serviceArea":"织里及周边","supportPriceText":"按件计费","responseTime":"当天响应","equipmentAvailable":true},
  job_hiring: {"position":"平车工","payText":"计件 0.8-1.2 元","headcount":8,"workLocation":"织里镇利济路 88 号","includeMealsHousing":true,"settlementMode":"计件"},
  job_seeking: {"desiredPosition":"平车工","expectedPayText":"月薪 9000 元以上","experienceYears":"5 年","availableTime":"随时到岗","expectedLocation":"织里"},
  sample_rental: {"sampleRoomType":"直播样衣间","areaText":"80 平","rentText":"1200 元/周","locationText":"织里童装城 2 区","displayCapacity":"可陈列 500 款"},
  shop_office_rental: {"spaceType":"沿街商铺","areaText":"95 平","rentText":"9800 元/月","locationText":"织里童装城一楼","floor":"1 楼","availableTime":"随时可租"},
  shop_sale: {"saleType":"沿街商铺","areaText":"110 平","salePriceText":"总价 260 万元","locationText":"织里利济路中段","certificateStatus":"可过户"},
  seek_shop_office: {"desiredSpaceType":"直播档口","expectedAreaText":"80-120 平","budgetRentText":"1.2 万元/月以内","preferredDistrict":"织里童装城一楼","usagePurpose":"直播与样衣展示"},
  apartment_rental: {"housingType":"员工公寓","roomLayout":"三室一厅","rentText":"3200 元/月","locationText":"织里镇工厂集中区","moveInTime":"本周可入住","furnished":true},
  housing_sale: {"housingType":"普通住宅","areaText":"89 平","salePriceText":"总价 138 万元","locationText":"织里镇中心","certificateStatus":"满两年"},
  seek_housing: {"desiredHousingType":"员工宿舍","expectedAreaText":"90 平以上","budgetRentText":"3500 元/月以内","preferredDistrict":"织里工厂集中区","moveInTime":"一周内"},
  factory_warehouse_rental: {"placeType":"仓库","areaText":"120 平","rentText":"6800 元/月","locationText":"织里童装城旁","powerCapacity":"50kW","truckAccess":true},
  workshop_rental: {"workshopType":"缝制间","equipmentIncluded":true,"areaText":"260 平","rentText":"1.6 万元/月","locationText":"织里镇阿祥路"},
  factory_sale: {"placeType":"独栋厂房","areaText":"3200 平","salePriceText":"总价 1680 万元","locationText":"织里产业园","certificateStatus":"有证","powerCapacity":"630kVA"},
  seek_factory_warehouse: {"desiredPlaceType":"一楼仓","expectedAreaText":"500-800 平","budgetRentText":"3 万元/月以内","preferredDistrict":"织里东部","powerRequirement":"100kW 以上","truckAccessRequired":true},
  secondhand_sale: {"itemType":"设备","conditionLevel":"九成新","secondhandPriceText":"转让价 12 万元","secondhandQuantityText":"1 台","pickupLocation":"织里镇利济路"},
  secondhand_buy: {"wantedItemType":"货架","budgetRange":"每组 300 元以内","demandQuantityText":"20 组","pickupArea":"织里及周边","acceptUsedCondition":true},
  education_training: {"courseType":"童装制版","targetAudience":"工厂版师和创业团队","coursePriceText":"1980 元/期","classTime":"周一至周五晚间","serviceArea":"织里"},
  appliance_repair: {"applianceType":"缝纫设备","serviceArea":"织里全域","repairPriceText":"上门检测 80 元起","responseTime":"2 小时内响应"},
  moving_cleaning: {"serviceType":"仓库搬运","serviceArea":"织里及周边 20 公里","movingPriceText":"按车次和人工报价","appointmentTime":"当天可预约"},
  other_local_service: {"serviceType":"摄影拍摄","serviceArea":"织里童装城","localServicePriceText":"主图套拍 399 元起","responseTime":"次日交片"},
}
```

少量有图资源的 `cover_url` 固定为 `/static/home/factory-hero.jpg`，其余资源使用空字符串；不得新增第三方远程图片地址。

- [ ] **Step 5: 运行类型矩阵校验**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
```

Expected: PASS，输出 `demo seed static check ok`。

- [ ] **Step 6: 提交完整类型矩阵**

```bash
git add backend/scripts/validate_demo_seed.mjs backend/scripts/seed_demo_data.sql
git commit -m "testdata: 覆盖完整供需类型矩阵"
```

### Task 3: 补齐生命周期场景和关联数据

**Files:**
- Modify: `backend/scripts/validate_demo_seed.mjs`
- Modify: `backend/scripts/seed_demo_data.sql:155-360`
- Modify: `backend/scripts/seed_demo_data.sql:857-891`

**Interfaces:**
- Consumes: Task 2 的资源 ID `8030000000000000001` 至 `8030000000000000028` 和商家角色。
- Produces: 至少 34 条资源、八种状态、特殊公开边界，以及与资源状态一致的审核、指标、联系和消息。

- [ ] **Step 1: 增加状态、特殊场景和关联数据校验**

在 `validate_demo_seed.mjs` 中加入：

```js
const expectedStatuses = [
  'draft',
  'pending',
  'manual_review',
  'audit_retry',
  'published',
  'rejected',
  'taken_down',
  'expired',
]

assert(scenarios.length >= 34, `供需演示场景偏少: ${scenarios.length}, want >= 34`)
for (const status of expectedStatuses) {
  assert(scenarios.some((item) => item.status === status), `演示种子缺少资源状态: ${status}`)
}

for (const marker of [
  "now() - interval '1 hours'",
  "now() + interval '24 hours'",
  "now() + interval '1 day'",
  "now() - interval '2 days'",
  "now() - interval '10 days'",
  'top_started_at',
  'top_expires_at',
  'dealt_at',
  'taken_down_at',
  'take_down_reason',
]) {
  assert(resourceSql.includes(marker), `演示种子缺少特殊生命周期场景: ${marker}`)
}

for (const association of [
  'INSERT INTO resource_review_records',
  'INSERT INTO resource_metrics_daily',
  'INSERT INTO resource_contact_events',
  'INSERT INTO messages',
  "'resource_approve'",
  "'resource_reject'",
  "'resource_expiring'",
]) {
  assert(sql.includes(association), `演示种子缺少关联调试数据: ${association}`)
}
```

- [ ] **Step 2: 运行校验并确认状态矩阵失败**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
```

Expected: FAIL，提示 `供需演示场景偏少: 28, want >= 34`。

- [ ] **Step 3: 增加 6 条生命周期扩展资源**

追加以下显式场景：

| resourceId | merchantId | typeCode | status | title | 关键状态字段 |
| --- | --- | --- | --- | --- | --- |
| `8030000000000000029` | `8020000000000000002` | `stock_clearance` | `draft` | 尚未提交的秋款库存草稿 | 发布、刷新时间为空 |
| `8030000000000000030` | `8020000000000000002` | `stock_clearance` | `pending` | 待审核夏款短袖库存 | 发布、刷新时间为空 |
| `8030000000000000031` | `8020000000000000001` | `factory_direct` | `manual_review` | 等待人工复核的工厂货源 | 发布、刷新时间为空 |
| `8030000000000000032` | `8020000000000000005` | `fabric_supply` | `audit_retry` | 内容审核重试中的面料现货 | 发布、刷新时间为空 |
| `8030000000000000033` | `8020000000000000001` | `factory_direct` | `rejected` | 资料不完整的货源演示 | `reject_reason='缺少清晰价格和联系方式确认材料'` |
| `8030000000000000034` | `8020000000000000002` | `secondhand_sale` | `published` | 已成交超过七天的旧设备 | `dealt_at=now() - interval '10 days'` |

同时为基准资源设置：

```sql
-- 8030000000000000007：有效置顶
top_started_at = now() - interval '1 hours'
top_expires_at = now() + interval '24 hours'

-- 8030000000000000003：即将过期
expires_at = now() + interval '1 day'

-- 8030000000000000023：近期成交，仍可公开
dealt_at = now() - interval '2 days'

-- 8030000000000000010：已停用招聘历史下架
taken_down_at = now() - interval '5 days'
take_down_reason = '冷启动阶段暂停招聘入口'
```

- [ ] **Step 4: 同步审核、指标、联系和消息关联**

审核记录使用固定 ID `8044000000000000001` 起，至少包含：

- 所有 26 条公开基准资源的 `approve`。
- `8030000000000000033` 的 `reject`。
- `8030000000000000010` 的 `take_down`。

不要为 `draft`、`pending`、`manual_review`、`audit_retry` 伪造审核通过记录。

指标至少覆盖以下资源：

- 供给：`8030000000000000001`、`8030000000000000003`、`8030000000000000007`。
- 需求：`8030000000000000004`、`8030000000000000008`、`8030000000000000015`。

联系事件由用户 `8010000000000000004` 或 `8010000000000000005` 发起，关联公开资源和对应商家。消息固定包含：

- 库存资源审核通过。
- 工厂货源审核驳回。
- 库存资源即将过期。

消息 `trigger_id` 必须引用现有资源 ID，重复执行时通过固定消息 ID 更新。

- [ ] **Step 5: 运行静态校验和相关脚本测试**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
node --test backend/scripts/validate_migrations.test.mjs deploy/scripts/update-test-db.test.mjs deploy/scripts/reset-test-db.test.mjs
```

Expected: 全部 PASS；演示种子输出 `demo seed static check ok`，Node 测试无失败。

- [ ] **Step 6: 提交生命周期和关联数据**

```bash
git add backend/scripts/validate_demo_seed.mjs backend/scripts/seed_demo_data.sql
git commit -m "testdata: 补齐供需生命周期调试场景"
```

### Task 4: 验证重复导入并完成交付检查

**Files:**
- Modify: `backend/scripts/verify_migrations.go:319-327`
- Modify: `backend/scripts/verify_migrations_test.go`
- Verify only: `backend/scripts/seed_demo_data.sql`
- Verify only: `backend/scripts/validate_demo_seed.mjs`
- Verify only: `deploy/scripts/reset-test-db.sh`

**Interfaces:**
- Consumes: Task 1–3 的完整演示种子。
- Produces: 临时数据库会连续导入两次演示种子，可由用户安全执行 `deploy/scripts/reset-test-db.sh` 的已验证工作区。

- [ ] **Step 1: 编写重复导入文件序列的失败测试**

在 `verify_migrations_test.go` 增加：

```go
func TestCollectDemoSeedImportFilesRunsSeedTwice(t *testing.T) {
	rootDir := t.TempDir()
	migrationsDir := filepath.Join(rootDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	for _, name := range []string{"000001_init.up.sql", "000002_core.up.sql"} {
		if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte("-- "+name), 0o600); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}

	files, err := collectDemoSeedImportFiles(rootDir)
	if err != nil {
		t.Fatalf("collectDemoSeedImportFiles() error = %v", err)
	}
	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, filepath.Base(file))
	}
	want := []string{
		"000001_init.up.sql",
		"000002_core.up.sql",
		"seed_demo_data.sql",
		"seed_demo_data.sql",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("demo seed import files = %#v, want %#v", got, want)
	}
}
```

- [ ] **Step 2: 运行 Go 测试并确认缺少函数**

Run:

```bash
cd backend
go test ./scripts -run TestCollectDemoSeedImportFilesRunsSeedTwice -count=1
```

Expected: FAIL，编译错误包含 `undefined: collectDemoSeedImportFiles`。

- [ ] **Step 3: 实现连续两次导入种子的文件序列**

在 `verify_migrations.go` 中新增：

```go
func collectDemoSeedImportFiles(rootDir string) ([]string, error) {
	files, err := collectMigrationFiles(rootDir, "up")
	if err != nil {
		return nil, err
	}
	seedFile := filepath.Join(rootDir, "scripts", "seed_demo_data.sql")
	return append(files, seedFile, seedFile), nil
}
```

将 `verifyDemoSeedImport` 中原有：

```go
files, err := collectMigrationFiles(rootDir, "up")
if err != nil {
	return err
}
files = append(files, filepath.Join(rootDir, "scripts/seed_demo_data.sql"))
```

替换为：

```go
files, err := collectDemoSeedImportFiles(rootDir)
if err != nil {
	return err
}
```

- [ ] **Step 4: 运行重复导入单测**

Run:

```bash
cd backend
go test ./scripts -run TestCollectDemoSeedImportFilesRunsSeedTwice -count=1
```

Expected: PASS。

- [ ] **Step 5: 运行静态校验、迁移测试和重置脚本测试**

Run:

```bash
node backend/scripts/validate_demo_seed.mjs
node backend/scripts/validate_migrations.mjs
node --test backend/scripts/validate_migrations.test.mjs deploy/scripts/update-test-db.test.mjs deploy/scripts/reset-test-db.test.mjs
```

Expected: 所有命令退出码为 0，无 warning 或 error。

- [ ] **Step 6: 运行 Go 测试**

Run:

```bash
cd backend
go test ./scripts/... ./app/internal/logic/resource/... ./app/internal/logic/discovery/...
```

Expected: 所有包 PASS。

- [ ] **Step 7: 在临时 PostgreSQL 数据库验证迁移和两次种子导入**

Run:

```bash
cd backend
go run ./scripts/verify_migrations.go -config etc/app.yaml
```

Expected: 输出 `migration up/down check ok`；验证器创建临时数据库，执行迁移并连续执行两次演示种子后自动删除临时数据库。若本地 PostgreSQL 或网络不可用，记录原始错误，不执行远程重置替代验证。

- [ ] **Step 8: 提交幂等验证**

```bash
git add backend/scripts/verify_migrations.go backend/scripts/verify_migrations_test.go
git commit -m "test: 验证演示种子可重复导入"
```

- [ ] **Step 9: 检查最终差异和敏感信息**

Run:

```bash
git diff --check
git diff --stat HEAD~3..HEAD
git status --short
rg -n "ghp_|DATABASE_URL=postgres|BEGIN PRIVATE KEY|SECRET_ACCESS_KEY" backend/scripts/seed_demo_data.sql backend/scripts/validate_demo_seed.mjs
```

Expected:

- `git diff --check` 无输出。
- 差异只涉及设计允许的种子、静态校验和临时数据库验证文件。
- 敏感信息扫描无输出。
- 工作区没有未提交的实现文件。

- [ ] **Step 10: 汇总用户执行方式**

交付说明明确给出：

```bash
./deploy/scripts/reset-test-db.sh
```

并说明该命令会先备份、再清空并重建测试库，用户执行后应看到 `importing demo seed data...` 和 `test database updated successfully`。不得由实现代理代替用户执行这个破坏性命令。
