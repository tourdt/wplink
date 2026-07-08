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

  for (const displayName of ['库存清仓', '现货货源', '工厂接单', '招工招聘', '出租转让', '配套服务', '找现货', '找库存', '找工厂', '找服务']) {
    assert(seedSql.includes(`'${displayName}'`), `seed should contain resource type display name ${displayName}`)
    assert(displayNameSql.includes(`'${displayName}'`), `display name migration should contain ${displayName}`)
  }
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
  assert(seedSql.includes("'demand'"), 'seed should mark demand resource types with demand direction')
  assert(seedSql.includes("'supply'"), 'seed should mark supply resource types with supply direction')
})

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
