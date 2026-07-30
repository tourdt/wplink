import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

const scriptPath = path.resolve(new URL('.', import.meta.url).pathname, 'seed_demo_data.sql')

if (!fs.existsSync(scriptPath)) {
  throw new Error('缺少 backend/scripts/seed_demo_data.sql')
}

const sql = fs.readFileSync(scriptPath, 'utf8')
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

function parseResourceScenarios(resourceSQL) {
  return [...resourceSQL.matchAll(
    /^\s*\((803\d+),\s*(802\d+),\s*'([^']+)',\s*'([^']+)',\s*'([^']+)'/gm,
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

const requiredSnippets = [
  '认证工厂',
  '认证库存商',
  '服务商',
  '采购商',
  'type_code',
  'stock_clearance',
  'factory_direct',
  'processing_accept',
  'find_factory',
  'job_hiring',
  'factory_warehouse_rental',
  'production_support',
  'top_voucher',
  'allowed_type_codes',
  'top_duration_hours',
  'pending',
  'rejected',
  'expired',
  'messages',
  'resource_metrics_daily',
  'map_scene',
  'map_object',
  'zhili_lijilu_demo',
  '利济路童装拿货示范图',
  '晨星童装 A 区',
  '云仓尾货 B 区',
  '利济路打包点',
  'A006',
  'C306',
  '童装城快递集包点',
  '利济路样衣中心',
  '早餐简餐补给点',
  'published',
]

for (const snippet of requiredSnippets) {
  if (!sql.includes(snippet)) {
    throw new Error(`演示种子缺少关键内容: ${snippet}`)
  }
}

const demoSceneRefs = sql.match(/'zhili_lijilu_demo'/g) || []
if (demoSceneRefs.length < 13) {
  throw new Error(`拿货地图演示数据对象偏少: scene refs=${demoSceneRefs.length}, want >= 13`)
}

if (sql.includes('8020000000000002,')) {
  throw new Error('演示种子包含疑似截断的商家 ID: 8020000000000002')
}

const retiredSnippets = [
  'INSERT INTO purchase_demands',
  'INSERT INTO match_cases',
  'INSERT INTO match_case_resources',
  'INSERT INTO match_case_participants',
  'match_progress',
  "'match_create'",
  "'match_case'",
  'INSERT INTO top_vouchers',
  'UPDATE top_vouchers',
  '采购需求已进入撮合',
  '演示撮合单',
]

for (const snippet of retiredSnippets) {
  if (sql.includes(snippet)) {
    throw new Error(`演示种子不应包含已下线采购需求/撮合内容: ${snippet}`)
  }
}

console.log('demo seed static check ok')
