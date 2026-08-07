import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repositoryRoot = path.resolve(scriptDir, '../..')
const productDocsDir = path.join(repositoryRoot, 'docs/product')

const archivedProductDocs = [
  'apparel-industry-platform-prd.md',
  'domain-model-ddd.md',
  'database-er-design.md',
  'resource-management-rules.md',
]
const archivedStatus = '文档状态：历史归档，不作为当前实现、验收、数据库迁移或 API 生成依据'
const currentSources = [
  'backend/app/api/app.api',
  'backend/migrations/',
  'docs/product/technical-architecture.md',
  'docs/product/mvp-acceptance-checklist.md',
]

test('历史产品文档声明归档状态并指向当前事实来源', () => {
  for (const documentName of archivedProductDocs) {
    const source = fs.readFileSync(path.join(productDocsDir, documentName), 'utf8')
    assert(source.includes(archivedStatus), `${documentName} must declare its archived status`)
    for (const currentSource of currentSources) {
      assert(source.includes(currentSource), `${documentName} must link to current source ${currentSource}`)
    }
  }
})

test('当前架构文档声明历史归档文档不得作为当前依据', () => {
  const architectureSource = fs.readFileSync(path.join(productDocsDir, 'technical-architecture.md'), 'utf8')
  assert(
    architectureSource.includes('历史归档文档不得作为当前依据'),
    'technical-architecture.md must define the priority of archived product documents',
  )
})

test('本地检查与 CI 都执行文档来源治理校验', () => {
  const makefileSource = fs.readFileSync(path.join(repositoryRoot, 'Makefile'), 'utf8')
  const workflowSource = fs.readFileSync(path.join(repositoryRoot, '.github/workflows/ci.yml'), 'utf8')

  assert(
    makefileSource.includes('scripts/product_doc_governance.test.mjs'),
    'Makefile must execute product_doc_governance.test.mjs',
  )
  assert(
    workflowSource.includes('scripts/product_doc_governance.test.mjs'),
    'CI must execute product_doc_governance.test.mjs',
  )
})
