import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const updateScriptPath = path.join(scriptDir, 'update-test-db.sh')
const migrationsDir = path.join(repoRoot, 'backend/migrations')

test('test database update script resets remote database with backup and confirmation', () => {
  const script = fs.readFileSync(updateScriptPath, 'utf8')

  for (const token of [
    'WPLINK_DEPLOY_TARGET',
    'WPLINK_SSH_KEY',
    'SSH_PORT',
    'REMOTE_CONFIG_DIR',
    'wplink.env',
    'DATABASE_URL',
    '--yes',
    'WPLINK_CONFIRM_RESET',
    'pg_dump',
    'wplink-db-backup-',
    'DROP SCHEMA IF EXISTS public CASCADE',
    'CREATE SCHEMA public',
    'schema_migrations',
    'backend/migrations',
    '*.up.sql',
    'seed_demo_data.sql',
    '--seed-demo',
    'resource_type_configs',
    'resources',
    'direction',
    'factory_direct',
    'buy_kids_goods',
    'seek_factory_warehouse',
    'expected 28 category item resource types',
    'expected 8 primary resource groups',
    'expected 7 demand category item resource types',
  ]) {
    assert.match(script, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('test database update script packages current migrations dynamically', () => {
  const script = fs.readFileSync(updateScriptPath, 'utf8')
  const migrationFiles = fs
    .readdirSync(migrationsDir)
    .filter((fileName) => fileName.endsWith('.up.sql'))
    .sort()

  assert(migrationFiles.length > 0)
  assert.match(script, /find "\$ROOT_DIR\/backend\/migrations"[\s\S]*-name '\*\.up\.sql'/)
  assert.match(script, /for migration_file in "\$migration_dir"\/\*\.up\.sql/)
  assert.doesNotMatch(script, /000012_search_trigram_indexes\.up\.sql"[\s\S]*\)/)
  assert.doesNotMatch(script, /\bmapfile\b/)
})
