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
