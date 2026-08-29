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
const currentFactDocuments = [
  'docs/product/technical-architecture.md',
  'docs/product/mvp-acceptance-checklist.md',
  'docs/product/api-implementation-checklist.md',
  'docs/product/wxapp-manual-acceptance.md',
  'docs/product/deployment-config.md',
  'docs/product/local-dev-runbook.md',
  'docs/architecture.md',
  'docs/deployment.md',
]
const retiredLocationTerms = [
  '自定义地图',
  '地图场景',
  '标注画布',
  '预录档口',
  '预录点位',
  '绑定申请',
  '绑定审核',
  'map_scene',
  'map_object',
  'merchant_map_events',
  'sourcing_map',
]

function findRetiredLocationReferences(documentSources) {
  const issues = []
  for (const documentPath of currentFactDocuments) {
    const source = documentSources.get(documentPath) || ''
    for (const retiredTerm of retiredLocationTerms) {
      if (source.includes(retiredTerm)) {
        issues.push(`${documentPath}: ${retiredTerm}`)
      }
    }
  }
  return issues
}

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

for (const documentPath of currentFactDocuments) {
  test(`文档治理会拒绝 ${documentPath} 中明确退役的位置领域语义`, () => {
    for (const retiredTerm of retiredLocationTerms) {
      const fixtureSources = new Map(currentFactDocuments.map((currentPath) => [currentPath, '当前事实']))
      fixtureSources.set(documentPath, `当前事实\n${retiredTerm}`)
      assert.deepEqual(findRetiredLocationReferences(fixtureSources), [`${documentPath}: ${retiredTerm}`])
    }

    const actualSources = new Map(currentFactDocuments.map((currentPath) => [
      currentPath,
      fs.readFileSync(path.join(repositoryRoot, currentPath), 'utf8'),
    ]))
    assert.deepEqual(findRetiredLocationReferences(actualSources), [])
  })
}

test('当前架构文档保留商家自主位置与目录治理事实', () => {
  const architectureSource = fs.readFileSync(path.join(productDocsDir, 'technical-architecture.md'), 'utf8')

  assert.match(architectureSource, /商家自主选择腾讯地图位置/)
  assert.match(architectureSource, /merchant_location_reports/)
  assert.match(architectureSource, /merchant_directory_events/)
  assert.match(architectureSource, /pages\/sourcing-map\/index\.vue/)
})

test('当前 API、部署与真机文档保持腾讯位置最终口径', () => {
  const apiChecklist = fs.readFileSync(path.join(productDocsDir, 'api-implementation-checklist.md'), 'utf8')
  const deploymentConfig = fs.readFileSync(path.join(productDocsDir, 'deployment-config.md'), 'utf8')
  const manualAcceptance = fs.readFileSync(path.join(productDocsDir, 'wxapp-manual-acceptance.md'), 'utf8')

  assert.match(apiChecklist, /GET \/api\/v1\/locations\/reverse-geocode/)
  assert.match(deploymentConfig, /RequestTimeout[^\n]*小于等于 0[^\n]*默认 5 秒/)
  for (const required of [
    'MerchantDirectoryEventCleanupInterval',
    'MerchantDirectoryEventCleanupTimeout',
    'MerchantDirectoryEventRetentionDays',
  ]) {
    assert.match(deploymentConfig, new RegExp(`生产[^\\n]*${required}|${required}[^\\n]*生产`))
  }
  assert.match(manualAcceptance, /无有效坐标[^\n]*仍可显示和复制文字地址/)
  assert.match(manualAcceptance, /不显示查看位置或导航/)
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
