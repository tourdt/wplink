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

test('product docs describe the generated API route as the only production route', () => {
  const retiredRouteDescriptions = [
    'NewAPIRouter',
    'NewProductionAPIRouter',
    'API_NOT_CONNECTED',
    '迁移期双轨制',
    '兼容 API Router',
    '兼容 Router',
    '未配置 API handler 的兜底路由',
    '未迁移端点由 fallback',
    'NotMigrated',
    'registerOptionalDomainRoutes',
    'backend/app/internal/server/goctl_routes.go',
    'backend/app/internal/server/api.go',
    'domain_routes.go',
  ]
  const docFiles = fs.readdirSync(productDocsDir)
    .filter((fileName) => fileName.endsWith('.md'))
    .map((fileName) => ({
      name: fileName,
      source: fs.readFileSync(path.join(productDocsDir, fileName), 'utf8'),
    }))

  for (const file of docFiles) {
    for (const snippet of retiredRouteDescriptions) {
      assert(!file.source.includes(snippet), `${file.name} should not contain retired route description ${snippet}`)
    }
  }

  const architectureSource = fs.readFileSync(path.join(productDocsDir, 'technical-architecture.md'), 'utf8')
  assert(
    architectureSource.includes('make check-api-generated'),
    'technical-architecture.md should document make check-api-generated',
  )
})

test('current product docs do not restore the retired merchant verification domain', () => {
  const currentDocNames = [
    'wxapp-manual-acceptance.md',
    'deployment-config.md',
    'api-implementation-checklist.md',
    'mvp-acceptance-checklist.md',
  ]
  const retiredContractReferences = [
    '/merchants/{merchantId}/verifications',
    '/api/v1/merchants/:merchantId/verifications',
    '/admin/verifications',
    'wxapp/pages/verification',
    'pages/verification/index',
    'VerificationView',
  ]
  const retiredCapabilityCopy = [
    '商家认证',
    '认证状态',
    '认证资料',
    '认证提交',
    '认证审核',
    '认证结果',
    '待认证',
  ]

  for (const fileName of currentDocNames) {
    const source = fs.readFileSync(path.join(productDocsDir, fileName), 'utf8')
    for (const snippet of [...retiredContractReferences, ...retiredCapabilityCopy]) {
      assert(!source.includes(snippet), `${fileName} should not describe retired merchant verification capability ${snippet}`)
    }
  }

  const historicalDesign = fs.readFileSync(path.join(productDocsDir, 'api-contract-design.md'), 'utf8')
  assert(historicalDesign.includes('文档状态：历史设计，已废弃'), 'api-contract-design.md should be marked as retired historical design')
  assert(historicalDesign.includes('不作为当前实现、验收或生成依据'), 'api-contract-design.md should reject current implementation usage')
  assert(historicalDesign.includes('backend/app/api/app.api'), 'api-contract-design.md should name the current API contract entry')
  assert(historicalDesign.includes('make check-api-generated'), 'api-contract-design.md should name the generated API gate')

  const implementationChecklist = fs.readFileSync(path.join(productDocsDir, 'api-implementation-checklist.md'), 'utf8')
  assert(
    !implementationChecklist.includes('docs/product/api-contract-design.md'),
    'api-implementation-checklist.md should not use the retired design as a current source',
  )
})

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

test('resource api contract exposes related resource recommendations', () => {
  const resourceApiSource = fs.readFileSync(path.join(apiDir, 'resource.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type RelatedResourcesReq',
    'type RelatedResourcesResp',
    'PageSize int64 `form:"pageSize,optional"`',
    'Items []ResourceListItem `json:"items"`',
    'get /resources/:resourceId/related (RelatedResourcesReq) returns (RelatedResourcesResp)',
  ]) {
    assert(resourceApiSource.includes(snippet), `resource.api should contain ${snippet}`)
  }

  const apiMerchant = resourceApiSource.match(/type ResourceMerchantBrief \{([\s\S]*?)\n\}/)?.[1] || ''
  const generatedMerchant = typesSource.match(/type ResourceMerchantBrief struct \{([\s\S]*?)\n\}/)?.[1] || ''
  for (const [name, contract] of [['resource.api', apiMerchant], ['generated types', generatedMerchant]]) {
    for (const field of ['json:"id"', 'json:"name"', 'json:"vipStatus"']) {
      assert(contract.includes(field), `${name} ResourceMerchantBrief should contain ${field}`)
    }
    assert(!contract.includes('verificationStatus'), `${name} ResourceMerchantBrief should not contain verificationStatus`)
  }
})

test('discovery api exposes public recent merchants without contact fields', () => {
  const discoveryApiSource = fs.readFileSync(path.join(apiDir, 'discovery.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  for (const snippet of [
    'type HomeRecentMerchantItem',
    'type HomeRecentMerchantsResp',
    'OnboardedAt',
    'onboardedAt',
  ]) {
    assert(discoveryApiSource.includes(snippet), `discovery.api should contain ${snippet}`)
    assert(typesSource.includes(snippet), `types.go should contain generated ${snippet}`)
  }
  assert(
    discoveryApiSource.includes('get /home/recent-merchants (HomeRecentMerchantsReq) returns (HomeRecentMerchantsResp)'),
    'discovery.api should expose the recent merchant route',
  )
  const itemContract = discoveryApiSource.match(/type HomeRecentMerchantItem \{([\s\S]*?)\n\}/)?.[1] || ''
  for (const sensitiveField of ['Phone', 'Wechat', 'Contact']) {
    assert(!itemContract.includes(sensitiveField), `recent merchant item should not expose ${sensitiveField}`)
  }
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

test('message role scope is declared by the public API contract and generated DTOs', () => {
  const messageApiSource = fs.readFileSync(path.join(apiDir, 'message.api'), 'utf8')
  const typesSource = fs.readFileSync(typesFile, 'utf8')

  const listContract = messageApiSource.match(/type ListMessagesReq \{([\s\S]*?)\n\}/)?.[1] || ''
  const readContract = messageApiSource.match(/type ReadMessageReq \{([\s\S]*?)\n\}/)?.[1] || ''
  const generatedList = typesSource.match(/type ListMessagesReq struct \{([\s\S]*?)\n\}/)?.[1] || ''
  const generatedRead = typesSource.match(/type ReadMessageReq struct \{([\s\S]*?)\n\}/)?.[1] || ''

  assert.match(listContract, /RoleCode\s+string `form:"roleCode,optional"`/)
  assert.match(readContract, /RoleCode\s+string `json:"roleCode,optional"`/)
  assert.match(generatedList, /RoleCode\s+string `form:"roleCode,optional"`/)
  assert.match(generatedRead, /RoleCode\s+string `json:"roleCode,optional"`/)
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

test('content audit callback contract uses query signatures without auth middleware', () => {
  const appApiSource = fs.readFileSync(path.join(apiDir, 'app.api'), 'utf8')
  const callbackPath = path.join(apiDir, 'callback.api')

  assert(fs.existsSync(callbackPath), 'callback.api should define provider callback routes')
  const callbackApiSource = fs.readFileSync(callbackPath, 'utf8')
  assert.match(appApiSource, /import "callback\.api"/)
  for (const snippet of [
    '@handler VerifyContentAuditCallback',
    'get /wechat/content-audit/media-callback (VerifyContentAuditCallbackReq)',
    '@handler HandleContentAuditCallback',
    'post /wechat/content-audit/media-callback (HandleContentAuditCallbackReq)',
  ]) {
    assert(callbackApiSource.includes(snippet), `callback.api should contain ${snippet}`)
  }
  assert.match(callbackApiSource, /Signature\s+string `form:"signature"`/)
  assert.match(callbackApiSource, /Timestamp\s+string `form:"timestamp"`/)
  assert.match(callbackApiSource, /Nonce\s+string `form:"nonce"`/)
  assert.match(callbackApiSource, /Echostr\s+string `form:"echostr"`/)
  assert(!callbackApiSource.includes('middleware:'), 'provider callbacks must not use user or admin middleware')
  assert(!callbackApiSource.includes('json:"'), 'POST callback body must stay raw and bounded in the Handler')
})

test('growth api contract exposes all runtime routes with exact DTO fields', () => {
  const appApiSource = fs.readFileSync(path.join(apiDir, 'app.api'), 'utf8')
  const growthPath = path.join(apiDir, 'growth.api')

  assert(fs.existsSync(growthPath), 'growth.api should define growth campaign routes')
  const growthApiSource = fs.readFileSync(growthPath, 'utf8')
  assert.match(appApiSource, /import "growth\.api"/)

  for (const route of [
    'get /growth-campaigns/active returns (ListActiveGrowthCampaignsResp)',
    'get /merchants/:merchantId/growth-tasks returns (GetGrowthTasksResp)',
    'get /growth-campaigns (ListGrowthCampaignsReq) returns (ListGrowthCampaignsResp)',
    'post /growth-campaigns (SaveGrowthCampaignReq) returns (SaveGrowthConfigResp)',
    'post /growth-campaigns/:campaignCode (SaveGrowthCampaignReq) returns (SaveGrowthConfigResp)',
    'get /growth-campaigns/:campaignCode/rules returns (ListGrowthRulesResp)',
    'post /growth-campaigns/:campaignCode/rules (SaveGrowthRuleReq) returns (SaveGrowthConfigResp)',
    'post /growth-campaigns/:campaignCode/rules/:ruleCode (SaveGrowthRuleReq) returns (SaveGrowthConfigResp)',
    'get /growth-campaigns/:campaignCode/grants (ListGrowthRewardGrantsReq) returns (ListGrowthRewardGrantsResp)',
  ]) {
    assert(growthApiSource.includes(route), `growth.api should contain ${route}`)
  }

  assert.match(
    growthApiSource,
    /@server \(\s*prefix: \/api\/v1\/admin\s*group:\s+admingrowth\s*middleware: AdminAuth\s*\)/,
    'admin growth routes must use the AdminAuth middleware',
  )

  const jsonFields = (typeName) => {
    const body = growthApiSource.match(new RegExp(`type ${typeName} \\{([\\s\\S]*?)\\n\\}`))?.[1]
    assert(body, `growth.api should define ${typeName}`)
    return [...body.matchAll(/`json:"([^",]+)(?:,optional)?"`/g)].map((match) => match[1])
  }

  const exactFields = {
    PublicGrowthCampaignItem: ['code', 'name', 'title', 'hint', 'rules'],
    PublicGrowthRuleItem: [
      'ruleCode', 'ruleName', 'triggerEvent', 'rewardType', 'rewardAmount', 'rewardText', 'validDays',
      'perUserLimit', 'perUserDailyLimit', 'perResourceDailyLimit', 'description', 'conditions',
    ],
    GrowthTaskCampaignInfo: ['code', 'title', 'hint'],
    GrowthTaskSummary: [
      'publishQuotaRemaining', 'refreshQuotaRemaining', 'starterCompletedCount', 'starterTotalCount',
    ],
    GrowthTaskItem: [
      'taskCode', 'group', 'title', 'description', 'progressCurrent', 'progressTarget', 'status',
      'rewardType', 'rewardAmount', 'rewardText', 'validDays', 'actionType', 'actionText', 'hint',
    ],
    GrowthCampaignItem: ['code', 'name', 'status', 'startsAt', 'endsAt', 'config', 'updatedAt'],
    GrowthRuleItem: [
      'campaignCode', 'ruleCode', 'ruleName', 'triggerEvent', 'status', 'priority', 'conditions',
      'rewardType', 'rewardAmount', 'validDays', 'perUserLimit', 'perUserDailyLimit',
      'perResourceDailyLimit', 'description', 'updatedAt',
    ],
    GrowthGrantItem: [
      'id', 'campaignCode', 'ruleCode', 'ruleName', 'merchantId', 'resourceId', 'rewardType',
      'rewardAmount', 'status', 'reason', 'createdAt',
    ],
  }
  for (const [typeName, fields] of Object.entries(exactFields)) {
    assert.deepEqual(jsonFields(typeName), fields, `${typeName} JSON fields should match the existing logic DTO exactly`)
  }
})

test('map api separates anonymous public routes from AdminAuth protected routes', () => {
  const mapApiSource = fs.readFileSync(path.join(apiDir, 'map.api'), 'utf8')
  const serverBlocks = [...mapApiSource.matchAll(/@server \(([\s\S]*?)\)\s*service wplink-api \{([\s\S]*?)\n\}/g)]
  assert.equal(serverBlocks.length, 2, 'map.api should define exactly one public and one admin server block')

  const publicBlock = serverBlocks.find((match) => match[1].includes('prefix: /api/v1\n'))
  const adminBlock = serverBlocks.find((match) => match[1].includes('prefix: /api/v1/admin'))
  assert(publicBlock, 'map.api should define public /api/v1 map routes')
  assert(adminBlock, 'map.api should define admin /api/v1/admin map routes')
  assert(!publicBlock[1].includes('middleware:'), 'public map routes must remain anonymous')
  assert.match(adminBlock[1], /middleware:\s*AdminAuth/, 'admin map routes must use AdminAuth')

  const publicHandlers = [...publicBlock[2].matchAll(/@handler\s+(\w+)/g)].map((match) => match[1])
  const adminHandlers = [...adminBlock[2].matchAll(/@handler\s+(\w+)/g)].map((match) => match[1])
  assert.equal(publicHandlers.length, 14, 'public map contract should keep 14 handlers')
  assert.equal(adminHandlers.length, 14, 'admin map contract should keep 14 handlers')
  assert.equal(new Set([...publicHandlers, ...adminHandlers]).size, 28, 'map contract should expose 28 unique handlers')

  const mapObjectItem = mapApiSource.match(/type MapObjectItem \{([\s\S]*?)\n\}/)?.[1] || ''
  const mapObjectMerchantItem = mapApiSource.match(/type MapObjectMerchantItem \{([\s\S]*?)\n\}/)?.[1] || ''
  assert.match(mapObjectItem, /IsVerifiedMerchant\s+bool\s+`json:"isVerifiedMerchant"`/,
    'map object contract should expose the existing verified display flag')
  assert.match(mapObjectMerchantItem, /VerificationStatus\s+string\s+`json:"verificationStatus"`/,
    'map merchant summary should match the existing logic DTO')
})

test('admin api protects every non-login group and keeps resource list request independent', () => {
  const adminApiSource = fs.readFileSync(path.join(apiDir, 'admin.api'), 'utf8')
  const serverBlocks = [...adminApiSource.matchAll(/@server \(([\s\S]*?)\)\s*service wplink-api \{([\s\S]*?)\n\}/g)]

  assert(serverBlocks.length > 1, 'admin.api should define login and protected admin groups')
  for (const [, metadata] of serverBlocks) {
    const group = metadata.match(/group:\s+(\w+)/)?.[1]
    assert(group, 'every admin @server block should declare a group')
    if (group === 'adminauth') {
      assert(!/middleware:\s*AdminAuth/.test(metadata), 'admin login must remain outside AdminAuth')
      continue
    }
    assert.match(metadata, /middleware:\s*AdminAuth/, `${group} must use AdminAuth`)
  }

  const resourcesReq = adminApiSource.match(/type AdminResourcesReq \{([\s\S]*?)\n\}/)?.[1]
  assert(resourcesReq, 'admin.api should define an independent AdminResourcesReq')
  for (const field of [
    'CityCode string `form:"cityCode,optional"`',
    'TypeCode string `form:"typeCode,optional"`',
    'Status   string `form:"status,optional"`',
    'Page     int64  `form:"page,optional"`',
    'PageSize int64  `form:"pageSize,optional"`',
  ]) {
    assert(resourcesReq.includes(field), `AdminResourcesReq should contain ${field}`)
  }
  const resourceBlock = serverBlocks.find(([, metadata]) => /group:\s+adminresource/.test(metadata))
  assert(resourceBlock, 'admin.api should define adminresource routes')
  assert(
    resourceBlock[2].includes('get /resources (AdminResourcesReq) returns (AdminPendingResourcesResp)'),
    'admin resources list must use AdminResourcesReq',
  )
  assert(
    resourceBlock[2].includes('get /resources/pending (AdminPendingResourcesReq) returns (AdminPendingResourcesResp)'),
    'pending resources route must keep its dedicated request',
  )
  assert.equal(
    [...resourceBlock[2].matchAll(/@handler\s+AdminListResources\b/g)].length,
    1,
    'admin resources list handler must be declared exactly once',
  )

  const resourceItem = adminApiSource.match(/type AdminPendingResourceItem \{([\s\S]*?)\n\}/)?.[1] || ''
  assert.match(
    resourceItem,
    /Status\s+string\s+`json:"status"`/,
    'the shared admin resource item must declare the runtime status field',
  )
})
