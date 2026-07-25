import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const appDir = path.resolve(scriptDir, '../app')
const apiDir = path.join(appDir, 'api')
const typesFile = path.join(appDir, 'internal/types/types.go')
const productDocsDir = path.resolve(scriptDir, '../../docs/product')

test('product docs do not describe retired purchase demand or manual matching features', () => {
  const retiredSnippets = [
    '采购需求',
    '人工撮合',
    'purchase_demands',
    'match_cases',
    'match_case',
    'purchase-demands',
    'match-cases',
    '/demands',
    '撮合记录',
    '撮合工作台',
    '撮合进度',
    '撮合权益',
    '撮合指标',
  ]
  const docFiles = fs.readdirSync(productDocsDir)
    .filter((fileName) => fileName.endsWith('.md'))
    .map((fileName) => ({
      name: fileName,
      source: fs.readFileSync(path.join(productDocsDir, fileName), 'utf8'),
    }))

  for (const file of docFiles) {
    for (const snippet of retiredSnippets) {
      assert(!file.source.includes(snippet), `${file.name} should not contain retired feature copy ${snippet}`)
    }
  }
})

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

test('api contract exposes demand direction through unified resource APIs', () => {
  const resourceApiSource = fs.readFileSync(path.join(apiDir, 'resource.api'), 'utf8')
  const cityApiSource = fs.readFileSync(path.join(apiDir, 'city.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const source of [resourceApiSource, cityApiSource, typesSource]) {
    assert(source.includes('Direction'), 'unified resource APIs should expose Direction')
    assert(source.includes('direction'), 'unified resource APIs should expose direction json/form tags')
  }
  assert(resourceApiSource.includes('CreateResourceReq'), 'resource creation stays on resources API')
  assert(resourceApiSource.includes('ListResourcesReq'), 'resource list stays on resources API')
  assert(cityApiSource.includes('ListResourceTypesReq'), 'resource type lookup accepts direction')
})

test('resource type api exposes optimistic versions', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const cityApiSource = fs.readFileSync(path.join(apiDir, 'city.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const source of [adminApiSource, cityApiSource, typesSource]) {
    assert(source.includes('Version'), 'resource type contracts should expose a config version')
    assert(source.includes('json:"version"'), 'resource type contracts should serialize the config version')
  }
  assert.match(adminApiSource, /type AdminUpdateResourceTypeConfigReq \{[\s\S]*Version\s+int64/)
})

test('api contract exposes list cover image through public resource APIs', () => {
  const resourceApiSource = fs.readFileSync(path.join(apiDir, 'resource.api'), 'utf8')
  const discoveryApiSource = fs.readFileSync(path.join(apiDir, 'discovery.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const source of [resourceApiSource, discoveryApiSource, typesSource]) {
    assert(source.includes('CoverUrl'), 'public resource APIs should expose CoverUrl')
    assert(source.includes('coverUrl'), 'public resource APIs should expose coverUrl json tag')
  }
  assert(resourceApiSource.includes('type ResourceListItem'), 'resource list item keeps a cover image')
  assert(discoveryApiSource.includes('type HomeResourceItem'), 'home resource item keeps a cover image')
  assert(discoveryApiSource.includes('type TopicResourceItem'), 'topic resource item keeps a cover image')
})

test('api contract exposes vip membership endpoints', () => {
  const appApiSource = fs.readFileSync(path.join(apiDir, 'app.api'), 'utf8')
  const vipApiSource = fs.readFileSync(path.join(apiDir, 'vip.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  assert.match(appApiSource, /import "vip\.api"/)
  assert.match(vipApiSource, /get \/vip\/plans returns \(ListVIPPlansResp\)/)
  assert.match(vipApiSource, /get \/vip\/quota-packs returns \(ListQuotaPacksResp\)/)
  assert.match(vipApiSource, /get \/merchants\/:merchantId\/vip returns \(MerchantVIPResp\)/)
  assert.match(vipApiSource, /post \/merchants\/:merchantId\/vip\/orders \(CreateVIPOrderReq\) returns \(CreateVIPOrderResp\)/)
  assert.match(vipApiSource, /post \/merchants\/:merchantId\/vip\/orders\/:orderId\/payment \(CreateVIPPaymentReq\) returns \(CreateVIPPaymentResp\)/)
  assert.match(typesSource, /type VIPPlanInfo struct/)
  assert.match(typesSource, /type QuotaPackInfo struct/)
  assert.match(typesSource, /ProductType string `json:"productType,optional"`/)
  assert.match(typesSource, /type MerchantVIPResp struct/)
})

test('admin api contract exposes vip config endpoints', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type AdminVIPBenefitConfig',
    'type AdminVIPPlanConfigItem',
    'type AdminSaveVIPPlanConfigReq',
    'type AdminQuotaPackConfigItem',
    'type AdminSaveQuotaPackConfigReq',
    'type AdminVIPPromotionConfigItem',
    'type AdminSaveVIPPromotionConfigReq',
    'get /vip/plans returns (AdminListVIPPlansResp)',
    'post /vip/plans (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/plans/:planCode (AdminSaveVIPPlanConfigReq) returns (AdminSaveVIPConfigResp)',
    'get /vip/quota-packs returns (AdminListQuotaPacksResp)',
    'post /vip/quota-packs (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/quota-packs/:packCode (AdminSaveQuotaPackConfigReq) returns (AdminSaveVIPConfigResp)',
    'get /vip/promotions returns (AdminListVIPPromotionsResp)',
    'post /vip/promotions (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)',
    'post /vip/promotions/:promotionCode (AdminSaveVIPPromotionConfigReq) returns (AdminSaveVIPConfigResp)',
  ]) {
    assert(adminApiSource.includes(snippet), `admin.api should contain ${snippet}`)
  }

  for (const snippet of [
    'type AdminVIPPlanConfigItem struct',
    'type AdminSaveVIPPlanConfigReq struct',
    'type AdminQuotaPackConfigItem struct',
    'type AdminVIPPromotionConfigItem struct',
  ]) {
    assert(typesSource.includes(snippet), `types.go should contain ${snippet}`)
  }
})

test('admin api contract exposes admin permission endpoints', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type AdminOperatorItem',
    'type AdminListOperatorsReq',
    'type AdminSaveOperatorReq',
    'type AdminModuleItem',
    'type AdminModulePermissionsResp',
    'type AdminUpdateRoleModulePermissionsReq',
    'get /operators (AdminListOperatorsReq) returns (AdminListOperatorsResp)',
    'post /operators (AdminSaveOperatorReq) returns (AdminSaveOperatorResp)',
    'post /operators/:operatorId (AdminSaveOperatorReq) returns (AdminSaveOperatorResp)',
    'post /operators/:operatorId/status (AdminUpdateOperatorStatusReq) returns (AdminSaveOperatorResp)',
    'get /module-permissions returns (AdminModulePermissionsResp)',
    'post /module-permissions/:roleCode (AdminUpdateRoleModulePermissionsReq) returns (AdminUpdateRoleModulePermissionsResp)',
  ]) {
    assert(adminApiSource.includes(snippet), `admin.api should contain ${snippet}`)
  }

  for (const snippet of [
    'type AdminOperatorItem struct',
    'type AdminListOperatorsReq struct',
    'type AdminSaveOperatorReq struct',
    'type AdminUpdateOperatorStatusReq struct',
    'type AdminModuleItem struct',
    'type AdminModulePermissionsResp struct',
    'type AdminUpdateRoleModulePermissionsReq struct',
  ]) {
    assert(typesSource.includes(snippet), `types.go should contain ${snippet}`)
  }
  assert.match(adminApiSource, /Modules \[\]string `json:"modules"`/)
  assert.match(typesSource, /Modules \[\]string `json:"modules"`/)
})

test('resource api exposes category commercial contact unlock contracts', () => {
  const resourceApiSource = fs.readFileSync(path.join(apiDir, 'resource.api'), 'utf8')
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type ResourceContactAccess',
    'type CreateContactUnlockOrderReq',
    'type CreateContactUnlockPaymentReq',
    'post /resources/:resourceId/contact-unlock-orders (CreateContactUnlockOrderReq) returns (CreateContactUnlockOrderResp)',
    'post /resources/:resourceId/contact-unlock-orders/:orderId/payment (CreateContactUnlockPaymentReq) returns (CreateContactUnlockPaymentResp)',
  ]) {
    assert(resourceApiSource.includes(snippet), `resource.api should contain ${snippet}`)
  }
  assert.match(resourceApiSource, /ContactAccess\s+ResourceContactAccess\s+`json:"contactAccess"`/)

  for (const snippet of [
    'type ResourceContactAccess struct',
    'type CreateContactUnlockOrderReq struct',
    'type CreateContactUnlockPaymentReq struct',
  ]) {
    assert(typesSource.includes(snippet), `types.go should contain ${snippet}`)
  }
  assert.match(typesSource, /CommercialRules\s+map\[string\]interface\{\}\s+`json:"commercialRules"`/)

  assert.match(adminApiSource, /CommercialRules\s+map\[string\]interface\{\}\s+`json:"commercialRules"`/, 'admin.api should expose commercialRules')
})

test('merchant detail contract exposes editable contact only as optional fields', () => {
  const merchantApiSource = fs.readFileSync(path.join(apiDir, 'merchant.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const source of [merchantApiSource, typesSource]) {
    assert.match(source, /Phone\s+string `json:"phone,optional"`/, 'merchant contact should expose optional editable phone')
    assert.match(source, /Wechat\s+string `json:"wechat,optional"`/, 'merchant contact should expose optional editable wechat')
    assert(source.includes('PhoneMasked'), 'merchant contact should keep masked phone for public detail')
    assert(source.includes('WechatMasked'), 'merchant contact should keep masked wechat for public detail')
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
