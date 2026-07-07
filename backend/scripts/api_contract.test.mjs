import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const appDir = path.resolve(scriptDir, '../app')
const apiDir = path.join(appDir, 'api')
const typesFile = path.join(appDir, 'internal/types/types.go')

test('api contract does not expose retired purchase demand endpoints', () => {
  const retiredSnippets = [
    'demand.api',
    'purchase-demands',
    'CreatePurchaseDemand',
    'ListMyPurchaseDemands',
    'AdminListDemands',
    'AdminDemand',
    'AdminUpdateDemandStatus',
  ]
  const contractFiles = fs.readdirSync(apiDir)
    .filter((fileName) => fileName.endsWith('.api'))
    .map((fileName) => ({
      name: fileName,
      source: fs.readFileSync(path.join(apiDir, fileName), 'utf8'),
    }))

  for (const file of contractFiles) {
    for (const snippet of retiredSnippets) {
      assert(!file.source.includes(snippet), `${file.name} should not contain ${snippet}`)
    }
  }
  assert(!fs.existsSync(path.join(apiDir, 'demand.api')), 'retired demand.api should be removed')
})

test('generated types do not keep retired purchase demand DTOs', () => {
  const source = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of ['CreatePurchaseDemand', 'ListMyPurchaseDemands', 'AdminDemand', 'AdminListDemands', 'AdminUpdateDemandStatus']) {
    assert(!source.includes(snippet), `types.go should not contain ${snippet}`)
  }
})

test('retired purchase demand business logic is removed', () => {
  const demandLogicDir = path.join(appDir, 'internal/logic/demand')
  if (fs.existsSync(demandLogicDir)) {
    assert.equal(fs.readdirSync(demandLogicDir).length, 0, 'internal/logic/demand should not contain files')
  }

  for (const retiredPath of ['internal/logic/admin/demand_admin_logic.go', 'internal/logic/admin/demand_admin_logic_test.go', 'internal/model/demand_model.go']) {
    assert(!fs.existsSync(path.join(appDir, retiredPath)), `${retiredPath} should be removed`)
  }
})
