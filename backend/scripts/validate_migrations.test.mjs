import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { loadMigrationFiles, validateMigrationFiles } from './validate_migrations.mjs'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const migrationsDir = path.resolve(scriptDir, '../migrations')
const expectedResourceTypeCodes = [
  'factory_direct',
  'spot_wholesale',
  'stock_clearance',
  'buy_kids_goods',
  'fabric_supply',
  'accessory_supply',
  'processing_accept',
  'find_factory',
  'production_support',
  'job_hiring',
  'job_seeking',
  'sample_rental',
  'shop_office_rental',
  'shop_sale',
  'seek_shop_office',
  'apartment_rental',
  'housing_sale',
  'seek_housing',
  'factory_warehouse_rental',
  'workshop_rental',
  'factory_sale',
  'seek_factory_warehouse',
  'secondhand_sale',
  'secondhand_buy',
  'education_training',
  'appliance_repair',
  'moving_cleaning',
  'other_local_service',
]
const expectedResourceGroups = new Map([
  ['kids_wholesale', '童装批发'],
  ['materials', '面料辅料'],
  ['production', '加工生产'],
  ['jobs', '招聘求职'],
  ['shop_office', '商铺办公'],
  ['housing', '住宅公寓'],
  ['factory_warehouse', '厂房仓库'],
  ['local_services', '本地服务'],
])

test('current migrations pass static validation', () => {
  const issues = validateMigrationFiles(loadMigrationFiles(migrationsDir))

  assert.deepEqual(issues, [])
})

test('migrations and demo seed do not point to retired demand pages', () => {
  const retiredSnippets = ['/pages/demand/index', '/pages/my-demands/index', '/pages/demand-success/index']
  const sqlFiles = [
    ...loadMigrationFiles(migrationsDir).map((file) => ({ name: file.fileName, sql: file.sql })),
    {
      name: 'seed_demo_data.sql',
      sql: fs.readFileSync(path.resolve(scriptDir, 'seed_demo_data.sql'), 'utf8'),
    },
  ]

  for (const file of sqlFiles) {
    for (const snippet of retiredSnippets) {
      assert(
        !file.sql.includes(snippet),
        `${file.name} should not contain retired miniapp path ${snippet}`,
      )
    }
  }
})

test('migrations do not create retired purchase demand or manual matching schema', () => {
  const retiredSchemaSnippets = [
    'CREATE TABLE IF NOT EXISTS purchase_demands',
    'CREATE TABLE IF NOT EXISTS match_cases',
    'CREATE TABLE IF NOT EXISTS match_case_resources',
    'CREATE TABLE IF NOT EXISTS match_case_participants',
    'generated_demand_id',
    'REFERENCES purchase_demands',
    'REFERENCES match_cases',
    'idx_purchase_demands',
    'idx_match_cases',
    'idx_match_case_participants',
  ]

  for (const file of loadMigrationFiles(migrationsDir)) {
    for (const snippet of retiredSchemaSnippets) {
      assert(!file.sql.includes(snippet), `${file.fileName} should not contain retired schema ${snippet}`)
    }
  }
})

test('resource type seed uses category-item display names and primary groups', () => {
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')

  for (const displayName of ['工厂直批', '尾货批发', '库存出售', '求购尾货', '最新面料', '辅料配饰', '承接加工', '招加工厂', '生产配套', '招聘员工', '我要求职', '厂房/仓库出租', '搬家保洁']) {
    assert(seedSql.includes(`'${displayName}'`), `seed should contain category item display name ${displayName}`)
  }
  for (const [groupCode, groupName] of expectedResourceGroups) {
    assert(seedSql.includes(`"code":"${groupCode}"`), `seed should contain primary group code ${groupCode}`)
    assert(seedSql.includes(`"name":"${groupName}"`), `seed should contain primary group name ${groupName}`)
  }
  for (const oldDisplayName of ["'库存清仓'", "'现货货源'", "'工厂接单'", "'招工招聘'", "'出租转让'", "'配套服务'", "'找现货'", "'找库存'", "'找服务'", "'找场地'"]) {
    assert(!seedSql.includes(oldDisplayName), `seed should not keep old display name ${oldDisplayName}`)
  }
})

test('core resource schema supports unified demand direction without retired demand tables', () => {
  const coreSql = fs.readFileSync(path.resolve(migrationsDir, '000002_core_domain.up.sql'), 'utf8')
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')
  for (const snippet of [
    'direction varchar(32) NOT NULL DEFAULT',
    'chk_resource_type_configs_direction',
    'chk_resources_direction',
    'idx_resources_city_direction_type_status',
  ]) {
    assert(coreSql.includes(snippet), `core migration should include resource direction schema snippet ${snippet}`)
  }

  for (const typeCode of ['buy_kids_goods', 'find_factory', 'job_seeking', 'seek_shop_office', 'seek_housing', 'seek_factory_warehouse', 'secondhand_buy']) {
    assert(seedSql.includes(`'${typeCode}'`), `seed should contain demand resource type ${typeCode}`)
  }
  assert(seedSql.includes("'demand'"), 'seed should mark demand resource types with demand direction')
  assert(seedSql.includes("'supply'"), 'seed should mark supply resource types with supply direction')
})

test('vip membership migration supports online quota pack purchase products', () => {
  const vipSql = fs.readFileSync(path.resolve(migrationsDir, '000017_vip_membership.up.sql'), 'utf8')

  for (const snippet of [
    'CREATE TABLE IF NOT EXISTS vip_quota_packs',
    "product_type varchar(32) NOT NULL DEFAULT 'vip_plan'",
    'product_snapshot jsonb NOT NULL DEFAULT',
    "chk_vip_orders_product_type CHECK (product_type IN ('vip_plan', 'quota_pack'))",
    "('publish_5', '发布次数包'",
    "('refresh_10', '刷新次数包'",
    "('top_3', '置顶券包'",
    '"publishQuota":5',
    '"refreshQuota":10',
    '"topVoucherCount":3',
  ]) {
    assert(vipSql.includes(snippet), `vip migration should include quota pack snippet ${snippet}`)
  }
})

test('vip migration backfills product columns for databases that already applied old 000017', () => {
  const repairSql = fs.readFileSync(path.resolve(migrationsDir, '000018_vip_order_product_backfill.up.sql'), 'utf8')

  for (const snippet of [
    'ALTER TABLE IF EXISTS vip_orders',
    'ADD COLUMN IF NOT EXISTS product_type',
    'ADD COLUMN IF NOT EXISTS product_code',
    'ADD COLUMN IF NOT EXISTS product_name',
    'ADD COLUMN IF NOT EXISTS product_snapshot',
    'ALTER COLUMN plan_id DROP NOT NULL',
    'ALTER COLUMN plan_version_id DROP NOT NULL',
    'UPDATE vip_orders o',
    "product_type = 'vip_plan'",
    "CREATE TABLE IF NOT EXISTS vip_quota_packs",
  ]) {
    assert(repairSql.includes(snippet), `repair migration should include snippet ${snippet}`)
  }
})

test('entitlement migrations store top vouchers as merchant entitlement batches with usage records', () => {
  const coreSql = fs.readFileSync(path.resolve(migrationsDir, '000002_core_domain.up.sql'), 'utf8')
  const upgradeSql = fs.readFileSync(path.resolve(migrationsDir, '000019_entitlement_usage_records.up.sql'), 'utf8')
  const demoSeedSql = fs.readFileSync(path.resolve(scriptDir, 'seed_demo_data.sql'), 'utf8')

  for (const sql of [coreSql, upgradeSql]) {
    for (const snippet of [
      'top_started_at timestamptz',
      'top_expires_at timestamptz',
      'allowed_type_codes jsonb NOT NULL DEFAULT',
      'top_duration_hours integer NOT NULL DEFAULT 0',
      'CREATE TABLE IF NOT EXISTS merchant_entitlement_usage_records',
      'before_remaining_amount integer NOT NULL',
      'after_remaining_amount integer NOT NULL',
      'resource_id bigint REFERENCES resources(id)',
    ]) {
      assert(sql.includes(snippet), `entitlement migration should include snippet ${snippet}`)
    }
    assert(!sql.includes('CREATE TABLE IF NOT EXISTS top_vouchers'), 'top_vouchers should not be created after entitlement unification')
  }

  assert(upgradeSql.includes('DROP TABLE IF EXISTS top_vouchers'), 'upgrade migration should drop retired top_vouchers table')
  assert(!demoSeedSql.includes('INSERT INTO top_vouchers'), 'demo seed should grant top vouchers through merchant_entitlements')
  assert(demoSeedSql.includes("'top_voucher'"), 'demo seed should include a top voucher entitlement')
  assert(demoSeedSql.includes('allowed_type_codes'), 'demo seed should preserve top voucher type limits on merchant_entitlements')
})

test('growth campaign migration supports configurable stoppable rewards', () => {
  const upSql = fs.readFileSync(path.resolve(migrationsDir, '000020_growth_campaigns.up.sql'), 'utf8')
  const downSql = fs.readFileSync(path.resolve(migrationsDir, '000020_growth_campaigns.down.sql'), 'utf8')

  for (const snippet of [
    'CREATE TABLE IF NOT EXISTS growth_campaigns',
    'CREATE TABLE IF NOT EXISTS growth_campaign_rules',
    'CREATE TABLE IF NOT EXISTS growth_reward_grants',
    'rule_name varchar(128) NOT NULL DEFAULT',
    "status varchar(32) NOT NULL DEFAULT 'draft'",
    'idempotency_key varchar(255) NOT NULL',
    'UNIQUE (idempotency_key)',
    'idx_growth_campaign_rules_campaign_status',
    'idx_growth_reward_grants_merchant',
    "'growth_campaign'",
    "'first_login_publish_quota', '首次登录赠送发布次数'",
    "'share_view_refresh_quota', '分享有效浏览奖励', 'resource_share_effective_view', 'inactive'",
    "'invitee_first_resource_approved', '邀请成功奖励', 'invitee_first_resource_approved', 'inactive'",
  ]) {
    assert(upSql.includes(snippet), `growth campaign migration should include snippet ${snippet}`)
  }
  assert(!upSql.includes("'top_voucher'"), 'growth campaign rewards should not expose top vouchers before top parameters are configurable')
  assert(
    upSql.indexOf('ADD COLUMN IF NOT EXISTS rule_name') > 0 &&
      upSql.indexOf('ADD COLUMN IF NOT EXISTS rule_name') < upSql.indexOf('INSERT INTO growth_campaign_rules'),
    'growth campaign migration should backfill rule_name before inserting seeded rules for databases with an existing growth_campaign_rules table',
  )

  for (const snippet of [
    'DROP TABLE IF EXISTS growth_reward_grants',
    'DROP TABLE IF EXISTS growth_campaign_rules',
    'DROP TABLE IF EXISTS growth_campaigns',
  ]) {
    assert(downSql.includes(snippet), `growth campaign rollback should include snippet ${snippet}`)
  }
})

test('resource type seed uses type-specific publish fields and summary mappings', () => {
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')

  const expectedSeedSnippets = [
    '"key":"productCategory"',
    '"summary":{"category":"productCategory","quantityText":"minOrderText","priceText":"factoryPriceText"}',
    '"key":"stockCategory"',
    '"summary":{"category":"stockCategory","quantityText":"stockQuantityText","priceText":"wholesalePriceText"}',
    '"key":"clearanceCategory"',
    '"summary":{"category":"clearanceCategory","quantityText":"stockQuantityText","priceText":"packagePriceText"}',
    '"key":"fabricType"',
    '"key":"accessoryType"',
    '"key":"processCategory"',
    '"key":"workLocation"',
    '"summary":{"category":"position","quantityText":"headcount","priceText":"payText"}',
    '"key":"desiredPosition"',
    '"key":"desiredSpaceType"',
    '"key":"desiredPlaceType"',
    '"key":"targetCategory"',
    '"summary":{"category":"targetCategory","quantityText":"orderQuantity","priceText":"budgetRange"}',
    '"key":"wantedItemType"',
    '"key":"applianceType"',
  ]
  for (const snippet of expectedSeedSnippets) {
    assert(seedSql.includes(snippet), `seed should contain publish field or summary snippet ${snippet}`)
  }

  for (const fixedRequired of [
    '["title","category","quantityText","contactPhone"]',
    '["title","category","priceText","contactPhone"]',
    '["title","category","contactPhone"]',
  ]) {
    assert(!seedSql.includes(fixedRequired), `seed should not require fixed summary fields ${fixedRequired}`)
  }
})

test('resource type migrations resolve to a self-contained final publish config', () => {
  const configs = finalResourceTypeConfigsFromMigrations(migrationsDir)
  const issues = []

  for (const typeCode of expectedResourceTypeCodes) {
    if (!configs.has(typeCode)) {
      issues.push(`缺少供需类型配置 ${typeCode}`)
    }
  }

  for (const config of configs.values()) {
    const dynamicFields = new Set()
    for (const field of config.fieldSchema.fields || []) {
      if (!field.key) {
        issues.push(`${config.typeCode} 存在缺少 key 的字段配置`)
        continue
      }
      if (resourceConfigBaseFields.has(field.key)) {
        issues.push(`${config.typeCode} 字段 ${field.key} 与基础字段冲突`)
      }
      if (dynamicFields.has(field.key)) {
        issues.push(`${config.typeCode} 字段 ${field.key} 重复定义`)
      }
      dynamicFields.add(field.key)
    }

    const allowedFields = new Set([...resourceConfigBaseFields, ...dynamicFields])
    const checkFieldList = (label, fields = []) => {
      for (const field of fields) {
        if (!allowedFields.has(field)) {
          issues.push(`${config.typeCode} ${config.typeName}: ${label} 引用未定义字段 ${field}`)
        }
      }
    }

    if (!config.displayTemplate.summary || typeof config.displayTemplate.summary !== 'object') {
      issues.push(`${config.typeCode} ${config.typeName}: 缺少 display_template.summary`)
    } else {
      for (const [target, source] of Object.entries(config.displayTemplate.summary)) {
        if (!resourceSummaryTargetFields.has(target)) {
          issues.push(`${config.typeCode} ${config.typeName}: summary 目标字段 ${target} 不支持`)
        }
        if (!allowedFields.has(source)) {
          issues.push(`${config.typeCode} ${config.typeName}: summary.${target} 引用未定义字段 ${source}`)
        }
      }
    }
    if (!config.displayTemplate.group || typeof config.displayTemplate.group !== 'object') {
      issues.push(`${config.typeCode} ${config.typeName}: 缺少 display_template.group`)
    } else if (expectedResourceGroups.get(config.displayTemplate.group.code) !== config.displayTemplate.group.name) {
      issues.push(`${config.typeCode} ${config.typeName}: display_template.group 无效 ${JSON.stringify(config.displayTemplate.group)}`)
    }

    for (const summaryField of resourceSummaryTargetFields) {
      if (config.requiredFields.includes(summaryField)) {
        issues.push(`${config.typeCode} ${config.typeName}: required_fields 不应直接要求摘要字段 ${summaryField}`)
      }
    }
    checkFieldList('required_fields', config.requiredFields)
    checkFieldList('filter_fields', config.filterFields)
    checkFieldList('display_template.list', config.displayTemplate.list || [])
    checkFieldList('display_template.detail', config.displayTemplate.detail || [])
  }

  assert.deepEqual(issues, [])
})

test('legacy follow-up resource type migrations are no-op after category item seed rewrite', () => {
  const displayNameSql = fs.readFileSync(path.resolve(migrationsDir, '000014_resource_type_display_names.up.sql'), 'utf8')
  const findRentalSql = fs.readFileSync(path.resolve(migrationsDir, '000015_find_rental_resource_type.up.sql'), 'utf8')

  assert(displayNameSql.includes('SELECT 1;'), 'display name migration should be a no-op after seed rewrite')
  assert(findRentalSql.includes('SELECT 1;'), 'find rental migration should be a no-op after seed rewrite')
  assert(!displayNameSql.includes("'inventory'"), 'display name migration should not reintroduce old type codes')
  assert(!findRentalSql.includes("'find_rental'"), 'find rental migration should not reintroduce old demand type code')
})

const resourceConfigBaseFields = new Set([
  'merchantId',
  'cityCode',
  'typeCode',
  'title',
  'category',
  'district',
  'quantityText',
  'priceText',
  'description',
  'contactName',
  'contactPhone',
  'contactWechat',
  'images',
  'tags',
])

const resourceSummaryTargetFields = new Set(['category', 'quantityText', 'priceText'])

function finalResourceTypeConfigsFromMigrations(migrationsRoot) {
  const configs = new Map()
  applySeedResourceTypeConfigs(configs, readMigrationSql(migrationsRoot, '000003_seed_zhili.up.sql'))
  applyFieldSchemaOverride(configs, readMigrationSql(migrationsRoot, '000013_resource_field_schema_options.up.sql'))
  return configs
}

function readMigrationSql(migrationsRoot, fileName) {
  return fs.readFileSync(path.resolve(migrationsRoot, fileName), 'utf8')
}

function applySeedResourceTypeConfigs(configs, sql) {
  const valuesBlock = extractValuesBlock(sql, 'INSERT INTO resource_type_configs')
  for (const row of extractSqlTuples(valuesBlock)) {
    const values = extractSqlStringLiterals(row)
    if (values.length < 7) continue
    upsertResourceTypeConfig(configs, {
      typeCode: values[0],
      typeName: values[1],
      direction: values[2],
      fieldSchema: parseJsonLiteral(values[3]),
      requiredFields: parseJsonLiteral(values[4]),
      filterFields: parseJsonLiteral(values[5]),
      displayTemplate: parseJsonLiteral(values[6]),
    })
  }
}

function applyFieldSchemaOverride(configs, sql) {
  const valuesBlock = extractValuesBlock(sql, 'CROSS JOIN')
  for (const row of extractSqlTuples(valuesBlock)) {
    const values = extractSqlStringLiterals(row)
    const config = configs.get(values[0])
    if (config && values[1]) {
      config.fieldSchema = parseJsonLiteral(values[1])
    }
  }
}

function upsertResourceTypeConfig(configs, config) {
  configs.set(config.typeCode, {
    fieldSchema: { fields: [] },
    requiredFields: [],
    filterFields: [],
    displayTemplate: {},
    ...configs.get(config.typeCode),
    ...config,
  })
}

function extractValuesBlock(sql, marker) {
  const markerIndex = sql.indexOf(marker)
  if (markerIndex < 0) return ''
  const valuesIndex = sql.indexOf('VALUES', markerIndex)
  if (valuesIndex < 0) return ''
  const endIndex = sql.indexOf(') AS cfg', valuesIndex)
  if (endIndex < 0) return ''
  return sql.slice(valuesIndex + 'VALUES'.length, endIndex)
}

function extractSqlTuples(valuesBlock) {
  const rows = []
  let inQuote = false
  let depth = 0
  let start = -1
  for (let index = 0; index < valuesBlock.length; index += 1) {
    const char = valuesBlock[index]
    if (char === "'" && valuesBlock[index + 1] === "'") {
      index += 1
      continue
    }
    if (char === "'") {
      inQuote = !inQuote
      continue
    }
    if (inQuote) continue
    if (char === '(') {
      if (depth === 0) start = index
      depth += 1
    } else if (char === ')') {
      depth -= 1
      if (depth === 0 && start >= 0) {
        rows.push(valuesBlock.slice(start, index + 1))
      }
    }
  }
  return rows
}

function extractSqlStringLiterals(sql) {
  return [...sql.matchAll(/'((?:''|[^'])*)'/g)].map((match) => match[1].replace(/''/g, "'"))
}

function parseJsonLiteral(value) {
  return JSON.parse(value.replace(/''/g, "'"))
}

test('reports a migration without matching down file', () => {
  const issues = validateMigrationFiles([
    {
      version: '000001',
      name: 'demo',
      direction: 'up',
      fileName: '000001_demo.up.sql',
      sql: 'CREATE TABLE IF NOT EXISTS demo_items (id bigint PRIMARY KEY);',
    },
  ])

  assert(issues.some((issue) => issue.includes('000001_demo') && issue.includes('缺少 down')))
})

test('reports tables created by up but not removed by down', () => {
  const issues = validateMigrationFiles([
    {
      version: '000001',
      name: 'demo',
      direction: 'up',
      fileName: '000001_demo.up.sql',
      sql: 'CREATE TABLE IF NOT EXISTS demo_items (id bigint PRIMARY KEY);',
    },
    {
      version: '000001',
      name: 'demo',
      direction: 'down',
      fileName: '000001_demo.down.sql',
      sql: 'SELECT 1;',
    },
  ])

  assert(issues.some((issue) => issue.includes('demo_items') && issue.includes('down 未删除')))
})

test('loadMigrationFiles ignores unrelated files and keeps migration metadata', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-migrations-'))
  fs.writeFileSync(path.join(tempDir, '000002_core.up.sql'), 'CREATE TABLE IF NOT EXISTS core_items (id bigint);')
  fs.writeFileSync(path.join(tempDir, '000002_core.down.sql'), 'DROP TABLE IF EXISTS core_items;')
  fs.writeFileSync(path.join(tempDir, 'README.md'), 'ignore me')

  const files = loadMigrationFiles(tempDir)

  assert.equal(files.length, 2)
  assert.deepEqual(
    files.map((file) => `${file.version}_${file.name}.${file.direction}`),
    ['000002_core.down', '000002_core.up'],
  )
})
