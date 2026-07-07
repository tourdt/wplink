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
		'TopicDemandEntry',
		'DemandEntry',
		'demandEntry',
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

test('api contract does not expose retired manual matching endpoints', () => {
  const retiredSnippets = [
    'match-cases',
    'AdminCreateMatchCase',
    'AdminListMatchCases',
    'AdminUpdateMatchCaseStatus',
    'AdminAddMatchCaseResources',
    'AdminAddMatchCaseParticipants',
    'AdminMatchCase',
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
})

test('dashboard api contract does not expose retired pending demand metric', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of ['PendingDemandCount', 'pendingDemandCount']) {
    assert(!adminApiSource.includes(snippet), `admin.api should not contain ${snippet}`)
    assert(!typesSource.includes(snippet), `types.go should not contain ${snippet}`)
  }
})

test('generated types do not keep retired purchase demand DTOs', () => {
  const source = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of ['CreatePurchaseDemand', 'ListMyPurchaseDemands', 'AdminDemand', 'AdminListDemands', 'AdminUpdateDemandStatus']) {
    assert(!source.includes(snippet), `types.go should not contain ${snippet}`)
  }
})

test('generated types do not keep retired manual matching DTOs', () => {
  const source = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of ['AdminCreateMatchCase', 'AdminListMatchCases', 'AdminUpdateMatchCaseStatus', 'AdminAddMatchCaseResources', 'AdminAddMatchCaseParticipants', 'AdminMatchCase']) {
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

test('retired manual matching business logic is removed', () => {
  for (const retiredPath of [
    'internal/logic/admin/match_case_logic.go',
    'internal/logic/admin/match_case_logic_test.go',
    'internal/model/match_case_model.go',
    'internal/model/match_case_model_test.go',
  ]) {
    assert(!fs.existsSync(path.join(appDir, retiredPath)), `${retiredPath} should be removed`)
  }
})

test('retired purchase demand and manual matching table models are removed', () => {
  for (const retiredPath of [
    'internal/model/purchase_demands_model.go',
    'internal/model/purchase_demands_model_gen.go',
    'internal/model/match_cases_model.go',
    'internal/model/match_cases_model_gen.go',
    'internal/model/match_case_resources_model.go',
    'internal/model/match_case_resources_model_gen.go',
    'internal/model/match_case_participants_model.go',
    'internal/model/match_case_participants_model_gen.go',
  ]) {
    assert(!fs.existsSync(path.join(appDir, retiredPath)), `${retiredPath} should be removed`)
  }

  const searchLogsModel = fs.readFileSync(path.join(appDir, 'internal/model/search_logs_model_gen.go'), 'utf8')
  assert(!searchLogsModel.includes('GeneratedDemandId'), 'search_logs model should not contain GeneratedDemandId')
  assert(!searchLogsModel.includes('generated_demand_id'), 'search_logs model should not contain generated_demand_id')
})
