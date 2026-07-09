import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { loadMigrationFiles, validateMigrationFiles } from './validate_migrations.mjs'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const migrationsDir = path.resolve(scriptDir, '../migrations')

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

test('resource type migrations use user-facing display names and rental object type', () => {
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')
  const displayNameSql = fs.readFileSync(path.resolve(migrationsDir, '000014_resource_type_display_names.up.sql'), 'utf8')

  const findRentalSql = fs.readFileSync(path.resolve(migrationsDir, '000015_find_rental_resource_type.up.sql'), 'utf8')

  for (const displayName of ['库存清仓', '现货货源', '工厂接单', '招工招聘', '出租转让', '配套服务', '找现货', '找库存', '找工厂', '找服务']) {
    assert(seedSql.includes(`'${displayName}'`), `seed should contain resource type display name ${displayName}`)
    assert(displayNameSql.includes(`'${displayName}'`), `display name migration should contain ${displayName}`)
  }
  assert(findRentalSql.includes("'找场地'"), 'find rental migration should contain buyer-friendly display name 找场地')
  for (const oldDisplayName of ["'工厂产能'", "'订单需求'", "'出租/转让'"]) {
    assert(!seedSql.includes(oldDisplayName), `seed should not keep old display name ${oldDisplayName}`)
  }

  assert(seedSql.includes('"key":"rentalType"'), 'seed should include rentalType field')
  assert(displayNameSql.includes('"key":"rentalType"'), 'display name migration should include rentalType field')
  assert(displayNameSql.includes('"商品房"'), 'rentalType options should cover normal commodity housing')
})

test('core resource schema supports unified demand direction without retired demand tables', () => {
  const coreSql = fs.readFileSync(path.resolve(migrationsDir, '000002_core_domain.up.sql'), 'utf8')
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')
  const findRentalSql = fs.readFileSync(path.resolve(migrationsDir, '000015_find_rental_resource_type.up.sql'), 'utf8')

  for (const snippet of [
    'direction varchar(32) NOT NULL DEFAULT',
    'chk_resource_type_configs_direction',
    'chk_resources_direction',
    'idx_resources_city_direction_type_status',
  ]) {
    assert(coreSql.includes(snippet), `core migration should include resource direction schema snippet ${snippet}`)
  }

  for (const typeCode of ['buy_goods', 'find_inventory', 'find_factory', 'find_service']) {
    assert(seedSql.includes(`'${typeCode}'`), `seed should contain demand resource type ${typeCode}`)
  }
  assert(findRentalSql.includes("'find_rental'"), 'find rental migration should contain demand resource type find_rental')
  assert(findRentalSql.includes("'demand'"), 'find rental migration should mark find_rental with demand direction')
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

test('resource type seed uses type-specific publish fields and summary mappings', () => {
  const seedSql = fs.readFileSync(path.resolve(migrationsDir, '000003_seed_zhili.up.sql'), 'utf8')
  const rentalDisplaySql = fs.readFileSync(path.resolve(migrationsDir, '000014_resource_type_display_names.up.sql'), 'utf8')
  const findRentalSql = fs.readFileSync(path.resolve(migrationsDir, '000015_find_rental_resource_type.up.sql'), 'utf8')

  const expectedSeedSnippets = [
    '"key":"stockCategory"',
    '"summary":{"category":"stockCategory","quantityText":"stockQuantityText","priceText":"stockPriceText"}',
    '"key":"goodsCategory"',
    '"summary":{"category":"goodsCategory","quantityText":"minOrderQuantity","priceText":"supplyPriceText"}',
    '"key":"factoryCategory"',
    '"summary":{"category":"factoryCategory","quantityText":"dailyCapacity","priceText":"laborPriceText"}',
    '"key":"workLocation"',
    '"summary":{"category":"position","quantityText":"headcount","priceText":"payText"}',
    '"summary":{"category":"rentalType","quantityText":"areaText","priceText":"rentText"}',
    '"key":"servicePriceText"',
    '"summary":{"category":"serviceType","quantityText":"serviceArea","priceText":"servicePriceText"}',
    '"key":"targetCategory"',
    '"summary":{"category":"targetCategory","quantityText":"demandQuantityText","priceText":"budgetRange"}',
    '"summary":{"category":"targetCategory","quantityText":"orderQuantity","priceText":"budgetRange"}',
    '"key":"serviceRequirement"',
    '"summary":{"category":"serviceType","quantityText":"serviceArea","priceText":"budgetRange"}',
  ]
  for (const snippet of expectedSeedSnippets) {
    assert(seedSql.includes(snippet), `seed should contain publish field or summary snippet ${snippet}`)
  }

  assert(rentalDisplaySql.includes('"summary":{"category":"rentalType","quantityText":"areaText","priceText":"rentText"}'), 'rental display migration should preserve rental summary mapping')
  assert(findRentalSql.includes('"key":"supportRequirement"'), 'find rental migration should include supportRequirement field')
  assert(findRentalSql.includes('"summary":{"category":"rentalNeedType","quantityText":"expectedAreaText","priceText":"budgetRentText"}'), 'find rental migration should include summary mapping')

  for (const fixedRequired of [
    '["title","category","quantityText","contactPhone"]',
    '["title","category","priceText","contactPhone"]',
    '["title","category","contactPhone"]',
  ]) {
    assert(!seedSql.includes(fixedRequired), `seed should not require fixed summary fields ${fixedRequired}`)
    assert(!findRentalSql.includes(fixedRequired), `find rental migration should not require fixed summary fields ${fixedRequired}`)
  }
})

test('resource type migrations resolve to a self-contained final publish config', () => {
  const configs = finalResourceTypeConfigsFromMigrations(migrationsDir)
  const expectedTypeCodes = [
    'inventory',
    'goods',
    'factory',
    'job',
    'rental',
    'service',
    'buy_goods',
    'find_inventory',
    'find_factory',
    'find_service',
    'find_rental',
  ]
  const issues = []

  for (const typeCode of expectedTypeCodes) {
    if (!configs.has(typeCode)) {
      issues.push(`缺少资源类型配置 ${typeCode}`)
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

test('find rental migration backfills direction schema for existing databases', () => {
  const findRentalSql = fs.readFileSync(path.resolve(migrationsDir, '000015_find_rental_resource_type.up.sql'), 'utf8')

  for (const snippet of [
    'ALTER TABLE IF EXISTS resource_type_configs',
    'ADD COLUMN IF NOT EXISTS direction',
    'ALTER TABLE IF EXISTS resources',
    "WHERE type_code IN ('buy_goods', 'find_inventory', 'find_factory', 'find_service', 'find_rental')",
    'chk_resource_type_configs_direction',
    'chk_resources_direction',
    'idx_resource_type_configs_direction',
    'idx_resources_city_direction_type_status',
  ]) {
    assert(findRentalSql.includes(snippet), `find rental migration should backfill direction schema snippet ${snippet}`)
  }
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
  applyRentalDisplayOverride(configs, readMigrationSql(migrationsRoot, '000014_resource_type_display_names.up.sql'))
  applyFindRentalConfig(configs, readMigrationSql(migrationsRoot, '000015_find_rental_resource_type.up.sql'))
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

function applyRentalDisplayOverride(configs, sql) {
  const config = configs.get('rental')
  if (!config || !sql.includes("rtc.type_code = 'rental'")) return
  config.fieldSchema = extractJsonAssignment(sql, 'field_schema') || config.fieldSchema
  config.requiredFields = extractJsonAssignment(sql, 'required_fields') || config.requiredFields
  config.filterFields = extractJsonAssignment(sql, 'filter_fields') || config.filterFields
  config.displayTemplate = extractJsonAssignment(sql, 'display_template') || config.displayTemplate
}

function applyFindRentalConfig(configs, sql) {
  const match = sql.match(
    /'find_rental',\s*'找场地',\s*'demand',\s*'((?:''|[^'])*)'::jsonb,\s*'((?:''|[^'])*)'::jsonb,\s*'((?:''|[^'])*)'::jsonb,\s*'((?:''|[^'])*)'::jsonb/s,
  )
  if (!match) return
  upsertResourceTypeConfig(configs, {
    typeCode: 'find_rental',
    typeName: '找场地',
    direction: 'demand',
    fieldSchema: parseJsonLiteral(match[1]),
    requiredFields: parseJsonLiteral(match[2]),
    filterFields: parseJsonLiteral(match[3]),
    displayTemplate: parseJsonLiteral(match[4]),
  })
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

function extractJsonAssignment(sql, fieldName) {
  const match = sql.match(new RegExp(`${fieldName}\\s*=\\s*'((?:''|[^'])*)'::jsonb`, 's'))
  return match ? parseJsonLiteral(match[1]) : null
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
