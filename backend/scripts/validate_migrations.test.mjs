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

test('reports a migration without matching down file', () => {
  const issues = validateMigrationFiles([
    {
      version: '000001',
      name: 'demo',
      direction: 'up',
      fileName: '000001_demo.up.sql',
      sql: 'CREATE TABLE IF NOT EXISTS demo_items (id uuid PRIMARY KEY);',
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
      sql: 'CREATE TABLE IF NOT EXISTS demo_items (id uuid PRIMARY KEY);',
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
  fs.writeFileSync(path.join(tempDir, '000002_core.up.sql'), 'CREATE TABLE IF NOT EXISTS core_items (id uuid);')
  fs.writeFileSync(path.join(tempDir, '000002_core.down.sql'), 'DROP TABLE IF EXISTS core_items;')
  fs.writeFileSync(path.join(tempDir, 'README.md'), 'ignore me')

  const files = loadMigrationFiles(tempDir)

  assert.equal(files.length, 2)
  assert.deepEqual(
    files.map((file) => `${file.version}_${file.name}.${file.direction}`),
    ['000002_core.down', '000002_core.up'],
  )
})
