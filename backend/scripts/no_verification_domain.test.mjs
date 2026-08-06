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
  ].join('\n')
  const mapAPI = source('app/api/map.api')

  // 地图公开对象保留唯一的兼容展示字段 verificationStatus；该字段由 merchants.profile_status
  // 推导，并不恢复已下线的商家认证领域、路由、模型或数据来源。先锁定字段的精确声明与数量，
  // 再删除该唯一白名单行继续执行原有宽口径守卫，避免误把其他 verification 符号放进来。
  const allowedMapDisplayField = /^\s*VerificationStatus\s+string\s+`json:"verificationStatus"`\s*$/gm
  assert.equal(mapAPI.match(allowedMapDisplayField)?.length, 1)

  assert.doesNotMatch(apiSources, /verification|Verification|认证/)
  assert.doesNotMatch(mapAPI.replace(allowedMapDisplayField, ''), /verification|Verification|认证/)
})

test('初始库结构不再创建认证相关字段或数据表', () => {
  const schema = source('migrations/000002_core_domain.up.sql')

  assert.doesNotMatch(schema, /verification|Verification|认证/)
})
