import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const root = resolve(import.meta.dirname, '..')

function source(path) {
  return readFileSync(resolve(root, path), 'utf8')
}

test('运行时 API 不再暴露商家认证领域', () => {
  const apiSources = [
    source('app/api/app.api'),
    source('app/api/admin.api'),
    source('app/api/merchant.api'),
    source('app/api/resource.api'),
    source('app/api/discovery.api'),
    source('app/api/favorite.api'),
    source('app/api/map.api'),
  ].join('\n')

  assert.doesNotMatch(apiSources, /verification|Verification|认证/)
})

test('初始库结构不再创建认证相关字段或数据表', () => {
  const schema = source('migrations/000002_core_domain.up.sql')

  assert.doesNotMatch(schema, /verification|Verification|认证/)
})
