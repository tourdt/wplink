import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)

test('admin launch UI hides manual matching feature', () => {
  const visibleFiles = [
    'src/router/index.js',
    'src/layouts/AdminLayout.vue',
    'src/views/DashboardView.vue',
    'src/views/OperationLogView.vue',
  ]

  const visibleSource = visibleFiles.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('match-cases'), false)
  assert.equal(visibleSource.includes('match_case'), false)
  assert.equal(visibleSource.includes('人工撮合'), false)
  assert.equal(visibleSource.includes('待撮合'), false)
  assert.equal(visibleSource.includes('采购需求'), false)
  assert.equal(visibleSource.includes('pendingDemandCount'), false)
  assert.equal(fs.existsSync(path.join(root, 'src/views/MatchCaseView.vue')), false)
  assert.equal(fs.existsSync(path.join(root, 'src/api/match.js')), false)
})

test('admin routes lazy-load page views to keep initial bundle maintainable', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')

  assert.doesNotMatch(routeSource, /import\s+\w+View\s+from\s+['"]\.\.\/views\//)
  assert.match(routeSource, /const DashboardView = \(\) => import\('\.\.\/views\/DashboardView\.vue'\)/)
  assert.match(routeSource, /const MerchantView = \(\) => import\('\.\.\/views\/MerchantView\.vue'\)/)
})

test('admin ui exposes vip config management entry', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/vipConfig.js'), 'utf8')

  assert.match(routeSource, /const VIPConfigView = \(\) => import\('\.\.\/views\/VIPConfigView\.vue'\)/)
  assert.match(routeSource, /path: 'vip-configs'/)
  assert.match(layoutSource, /index: '\/vip-configs'[\s\S]*label: 'VIP 配置'[\s\S]*moduleCode: 'vip_configs'/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/plans/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/quota-packs/)
  assert.match(apiSource, /\/api\/v1\/admin\/vip\/promotions/)
})

test('admin ui exposes super admin permission management entry', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/adminPermission.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/AdminPermissionView.vue'), 'utf8')
  const authSource = fs.readFileSync(path.join(root, 'src/stores/auth.js'), 'utf8')

  assert.match(routeSource, /const AdminPermissionView = \(\) => import\('\.\.\/views\/AdminPermissionView\.vue'\)/)
  assert.match(routeSource, /path: 'admin-permissions'/)
  assert.match(routeSource, /requiresSuperAdmin: true/)
  assert.match(routeSource, /moduleCode: 'admin_permissions'/)
  assert.match(layoutSource, /moduleCode: 'admin_permissions'/)
  assert.match(layoutSource, /visibleMenuItems/)
  assert.match(layoutSource, /index: '\/admin-permissions'[\s\S]*label: '管理员权限'[\s\S]*moduleCode: 'admin_permissions'/)
  assert.match(apiSource, /\/api\/v1\/admin\/operators/)
  assert.match(apiSource, /\/api\/v1\/admin\/module-permissions/)
  assert.match(viewSource, /<h2>管理员权限<\/h2>/)
  assert.match(viewSource, /updateAdminOperatorStatus/)
  assert.match(viewSource, /平台运营可见模块/)
  assert.match(viewSource, /updateAdminRoleModulePermissions/)
  assert.match(authSource, /isSuperAdmin/)
})

test('admin navigation filters pages by configured module permissions', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const authSource = fs.readFileSync(path.join(root, 'src/stores/auth.js'), 'utf8')
  const moduleSource = fs.readFileSync(path.join(root, 'src/common/adminModulePermissions.js'), 'utf8')

  assert.match(routeSource, /auth\.canAccessModule\(to\.meta\.moduleCode\)/)
  assert.match(routeSource, /firstAccessibleRouteName/)
  assert.match(layoutSource, /auth\.canAccessModule\(item\.moduleCode\)/)
  assert.match(authSource, /modules: data\.modules \|\| \[\]/)
  assert.match(moduleSource, /defaultPlatformOperatorModules = \['resource_review', 'resource_reports'\]/)
  assert.match(moduleSource, /canAccessAdminModule/)
})

test('vip config page provides plan quota pack and promotion tabs', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/VIPConfigView.vue'), 'utf8')

  assert.match(source, /<h2>VIP 配置<\/h2>/)
  assert.match(source, /<el-tab-pane label="VIP 套餐" name="plans">/)
  assert.match(source, /<el-tab-pane label="次数包" name="quotaPacks">/)
  assert.match(source, /<el-tab-pane label="优惠活动" name="promotions">/)
  assert.match(source, /openPlanEditor/)
  assert.match(source, /openQuotaPackEditor/)
  assert.match(source, /openPromotionEditor/)
  assert.match(source, /yuanToCent/)
  assert.match(source, /centToYuan/)
  assert.match(source, /buildPlanPayload/)
  assert.match(source, /buildQuotaPackPayload/)
  assert.match(source, /buildPromotionPayload/)
})

test('admin build splits framework and ui libraries into stable vendor chunks', () => {
  const viteSource = fs.readFileSync(path.join(root, 'vite.config.js'), 'utf8')

  assert.match(viteSource, /manualChunks/)
  assert.match(viteSource, /vue-vendor/)
  assert.match(viteSource, /element-plus/)
  assert.match(viteSource, /vendor/)
})

test('admin app registers only used Element Plus components', () => {
  const mainSource = fs.readFileSync(path.join(root, 'src/main.js'), 'utf8')
  const registrySource = fs.existsSync(path.join(root, 'src/plugins/elementPlus.js'))
    ? fs.readFileSync(path.join(root, 'src/plugins/elementPlus.js'), 'utf8')
    : ''
  const sourceFiles = collectSourceFiles(path.join(root, 'src'))
  const sourceText = sourceFiles.map((file) => fs.readFileSync(file, 'utf8')).join('\n')

  assert.doesNotMatch(mainSource, /import\s+ElementPlus\s+from\s+['"]element-plus['"]/)
  assert.doesNotMatch(mainSource, /app\.use\(ElementPlus\)/)
  assert.doesNotMatch(sourceText, /from\s+['"]element-plus['"]/)
  assert.match(mainSource, /registerElementPlusComponents\(app\)/)
  assert.match(registrySource, /ElTable/)
  assert.match(registrySource, /ElForm/)
  assert.match(registrySource, /ElMessage/)
})

function collectSourceFiles(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) return collectSourceFiles(fullPath)
    return /\.(js|vue)$/.test(entry.name) ? [fullPath] : []
  })
}

test('admin launch UI hides purchase demand entry until the feature is released', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const bannerSource = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.equal(routeSource.includes("path: 'demands'"), false)
  assert.equal(routeSource.includes('DemandView'), false)
  assert.equal(fs.existsSync(path.join(root, 'src/views/DemandView.vue')), false)
  assert.equal(fs.existsSync(path.join(root, 'src/api/demand.js')), false)
  assert.equal(layoutSource.includes('index="/demands"'), false)
  assert.equal(layoutSource.includes('<span>采购需求</span>'), false)
  assert.equal(bannerSource.includes("value: 'demand'"), false)
  assert.equal(bannerSource.includes('/pages/demand/index'), false)
})

test('banner topic form uses image upload with ratio guidance', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /<el-upload/)
  assert.match(source, /建议比例\s*2\.2:1/)
  assert.match(source, /上传封面/)
  assert.equal(source.includes('placeholder="https://..."'), false)
})

test('banner config unifies topic entry and uses selectable non-web targets', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')

  assert.match(source, /<h2>首页运营位<\/h2>/)
  assert.match(layoutSource, /index: '\/banner-topics'[\s\S]*label: '首页运营位'[\s\S]*moduleCode: 'banner_topics'/)
  assert.match(source, /v-if="form\.jumpType === 'webview'"/)
  assert.match(source, /v-else-if="form\.jumpType === 'internal'"/)
  assert.match(source, /internalPageOptions/)
  assert.match(source, /resourceOptions/)
  assert.match(source, /merchantOptions/)
  assert.match(source, /kindOptions/)
  assert.equal(source.includes('<el-option label="专题" value="topic" />'), false)
})

test('banner internal page options retain the stable merchant directory path', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /\{ label: '拿货档口', value: '\/pages\/sourcing-map\/index' \}/)
  assert.doesNotMatch(source, /\{ label: '认证页', value: '\/pages\/verification\/index' \}/)
})

test('home recommend card config hides banner image fields', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /home_recommend_card/)
  assert.match(source, /v-if="isBannerKind"/)
  assert.match(source, /form\.kind === 'home_recommend_card'/)
  assert.match(source, /角标/)
  assert.match(source, /buildSubmitPayload\(\)[\s\S]*payload\.coverUrl = ''/)
})

test('resource type config explains required field meanings', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /供需类型配置/)
  assert.match(source, /必填字段控制商家发布或保存供需信息时必须补全的信息/)
  assert.match(source, /fieldDescriptionMap/)
  assert.match(source, /标题用于搜索、列表卡片和详情页主标题/)
  assert.match(source, /联系电话用于买家联系和平台审核核验/)
  assert.match(source, /required-field-note/)
  assert.doesNotMatch(source, /资源类型配置/)
})

test('resource type config provides visual field schema editor', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /字段配置/)
  assert.match(source, /基础必填字段/)
  assert.match(source, /fieldRows/)
  assert.match(source, /fieldTypeOptions/)
  assert.match(source, /value:\s*'address'[\s\S]{0,40}label:\s*'地址'/)
  assert.match(source, /baseRequiredFieldOptions/)
  assert.match(source, /addField/)
  assert.match(source, /removeField/)
  assert.match(source, /buildConfigPayloadFromVisualEditor/)
  assert.match(source, /allowCustom/)
  assert.match(source, /optionsText/)
  assert.match(source, /displayInList/)
  assert.match(source, /displayInDetail/)
  assert.match(source, /高级 JSON/)
})

test('resource type config visual editor excludes derived summary fields from base required options', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /summaryFieldValueSet/)
  assert.doesNotMatch(source, /value:\s*'category'[\s\S]{0,80}label:\s*'品类'/)
  assert.doesNotMatch(source, /value:\s*'quantityText'[\s\S]{0,80}label:\s*'数量\/产能'/)
  assert.doesNotMatch(source, /value:\s*'priceText'[\s\S]{0,80}label:\s*'价格描述'/)
})

test('resource type config registers interactive Element Plus editor controls', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')
  const registrySource = fs.readFileSync(path.join(root, 'src/plugins/elementPlus.js'), 'utf8')
  const requiredComponents = [
    ['<el-checkbox', 'ElCheckbox'],
    ['<el-radio-group', 'ElRadioGroup'],
    ['<el-radio-button', 'ElRadioButton'],
    ['<el-empty', 'ElEmpty'],
  ]

  for (const [tag, component] of requiredComponents) {
    assert.equal(source.includes(tag), true, `${tag} should remain covered by the registry test`)
    assert.match(
      registrySource,
      new RegExp(`import \\{[^}]*\\b${component}\\b[^}]*\\} from 'element-plus/es/components/`),
      `${component} should be imported for on-demand registration`,
    )
    assert.match(
      registrySource,
      new RegExp(`\\b${component}\\b,`),
      `${component} should be registered with the app`,
    )
  }
})

test('resource type config shows read-only resource direction labels', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /label="类型归属"/)
  assert.match(source, /directionLabel/)
  assert.match(source, /供应类型/)
  assert.match(source, /需求类型/)
  assert.match(source, /directionTagType/)
  assert.match(source, /editing\.direction/)
  assert.doesNotMatch(source, /direction:\s*editing\.value\.direction/)
})

test('resource type config exposes category hierarchy maintenance actions', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/city.js'), 'utf8')

  assert.match(source, /新增一级分类/)
  assert.match(source, /新增二级分类/)
  assert.match(source, /configGroups/)
  assert.match(source, /groupNameFromConfig/)
  assert.match(source, /openCreateGroup/)
  assert.match(source, /openCreateType/)
  assert.match(source, /createResourceTypeConfig/)
  assert.match(source, /配置版本：v\{\{ row\.version \}\}/)
  assert.match(source, /version:\s*editing\.value\.version/)
  assert.match(source, /一级分类/)
  assert.match(source, /二级类型/)
  assert.match(apiSource, /createResourceTypeConfig/)
  assert.match(apiSource, /\/api\/v1\/admin\/resource-type-configs/)
})

test('admin navigation uses supply demand information wording for resource sections', () => {
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const reviewSource = fs.readFileSync(path.join(root, 'src/views/ResourceReviewView.vue'), 'utf8')
  const configSource = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(layoutSource, /供需信息审核/)
  assert.match(layoutSource, /供需类型配置/)
  assert.match(reviewSource, /<h2>供需信息审核<\/h2>/)
  assert.match(configSource, /<h2>供需类型配置<\/h2>/)
  assert.doesNotMatch(layoutSource, />资源审核</)
  assert.doesNotMatch(layoutSource, />资源配置</)
  assert.doesNotMatch(reviewSource, /<h2>资源审核<\/h2>/)
  assert.doesNotMatch(configSource, /<h2>资源类型配置<\/h2>/)
})

test('banner target selectors support searchable remote options', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')

  assert.match(source, /remote-method="searchResourceTargets"/)
  assert.match(source, /remote-method="searchMerchantTargets"/)
  assert.match(source, /loading="resourceTargetLoading"/)
  assert.match(source, /loading="merchantTargetLoading"/)
  assert.match(source, /keyword: query/)
})

test('hot search keywords are configurable from admin web', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/hotSearchKeyword.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/HotSearchKeywordView.vue'), 'utf8')

  assert.match(routeSource, /HotSearchKeywordView/)
  assert.match(routeSource, /hot-search-keywords/)
  assert.match(layoutSource, /index: '\/hot-search-keywords'[\s\S]*label: '热门搜索词'[\s\S]*moduleCode: 'hot_search_keywords'/)
  assert.match(apiSource, /\/api\/v1\/admin\/hot-search-keywords/)
  assert.match(viewSource, /<h2>热门搜索词<\/h2>/)
  assert.match(viewSource, /v-model="form\.keyword"/)
  assert.match(viewSource, /v-model="form\.status"/)
})

test('admin merges location governance into merchants and removes custom map editor', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const merchantSource = fs.readFileSync(path.join(root, 'src/views/MerchantView.vue'), 'utf8')
  const merchantApiSource = fs.readFileSync(path.join(root, 'src/api/merchant.js'), 'utf8')
  const permissionSource = fs.readFileSync(path.join(root, 'src/views/AdminPermissionView.vue'), 'utf8')
  const retiredView = ['Sourcing', 'MapView'].join('')
  const retiredApi = ['sourcing', 'Map'].join('')
  const retiredViewport = ['map', 'Viewport'].join('')
  const retiredRoute = ['sourcing', 'map'].join('-')
  const retiredModule = ['sourcing', 'map'].join('_')
  const retiredLabel = ['拿货', '地图'].join('')
  const retiredIcon = ['Map', 'Location'].join('')

  assert.equal(routeSource.includes(retiredView), false)
  assert.equal(routeSource.includes(retiredRoute), false)
  assert.equal(layoutSource.includes(retiredLabel), false)
  assert.equal(layoutSource.includes(retiredModule), false)
  assert.equal(layoutSource.includes(retiredIcon), false)
  assert.match(permissionSource, /module\.code\s*!==\s*retiredSourcingMapModule/)
  assert.match(merchantSource, /位置状态/)
  assert.match(merchantSource, /位置错误反馈/)
  assert.match(merchantSource, /隐藏位置/)
  assert.match(merchantSource, /恢复位置/)
  assert.match(merchantSource, /pendingLocationReportCount/)
  assert.match(merchantSource, /marketName/)
  assert.match(merchantSource, /boothNo/)
  assert.match(merchantApiSource, /\/api\/v1\/admin\/merchants\/location-reports/)
  assert.match(merchantApiSource, /\/api\/v1\/admin\/merchants\/location-reports\/\$\{reportId\}\/review/)
  assert.match(merchantApiSource, /\/api\/v1\/admin\/merchants\/\$\{merchantId\}\/location-status/)
  assert.equal(fs.existsSync(path.join(root, `src/views/${retiredView}.vue`)), false)
  assert.equal(fs.existsSync(path.join(root, `src/api/${retiredApi}.js`)), false)
  assert.equal(fs.existsSync(path.join(root, `src/common/${retiredViewport}.js`)), false)
  assert.equal(fs.existsSync(path.join(root, `src/common/${retiredViewport}.test.mjs`)), false)
})

test('admin city station filters use dropdown options', () => {
  const citySource = fs.readFileSync(path.join(root, 'src/common/cityStations.js'), 'utf8')
  const filterFiles = [
    'src/views/SearchLogView.vue',
    'src/views/HotSearchKeywordView.vue',
    'src/views/ResourceReviewView.vue',
    'src/views/BannerTopicView.vue',
    'src/views/MerchantView.vue',
    'src/views/ResourceTypeConfigView.vue',
  ]

  assert.match(citySource, /cityStationOptions/)
  assert.match(citySource, /label:\s*'织里'/)
  assert.match(citySource, /value:\s*'zhili'/)

  for (const file of filterFiles) {
    const source = fs.readFileSync(path.join(root, file), 'utf8')
    assert.match(source, /cityStationOptions/)
    assert.match(source, /<el-select[^>]+v-model="filters\.cityCode"/)
    assert.equal(/<el-input[^>]+v-model(?:\.trim)?="filters\.cityCode"/.test(source), false)
  }

  const searchLogSource = fs.readFileSync(path.join(root, 'src/views/SearchLogView.vue'), 'utf8')
  assert.match(searchLogSource, /<el-option v-for="station in cityStationOptions"/)
  assert.equal(searchLogSource.includes('placeholder="zhili"'), false)

})

test('admin merchant identity wording matches mini program copy', () => {
  const merchantSource = fs.readFileSync(path.join(root, 'src/views/MerchantView.vue'), 'utf8')
  const entitlementSource = fs.readFileSync(path.join(root, 'src/views/EntitlementView.vue'), 'utf8')
  const bannerSource = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')
  const identitySource = fs.readFileSync(path.join(root, 'src/common/merchantIdentity.js'), 'utf8')
  const combinedSource = [merchantSource, entitlementSource, bannerSource, identitySource].join('\n')

  for (const token of ['主要身份', '个人', '场地/设备方', '源头工厂', '现货档口', '库存货源', '配套服务', '采购']) {
    assert.match(combinedSource, new RegExp(token))
  }
  assert.doesNotMatch(identitySource, /merchantIdentityOptions = \[[\s\S]*value: 'stall'/)
  assert.doesNotMatch(identitySource, /\{ label: '场地\/设备方', value: 'rental_provider' \}/)
  assert.match(identitySource, /stall: '现货档口'/)
  assert.match(identitySource, /rental_provider: '场地\/设备方'/)

  for (const oldToken of [
    '商家类型',
    "factory: '工厂'",
    "stall: '档口'",
    "stockist: '库存商'",
    "service_provider: '服务商'",
    "factory: '工厂认证'",
    "stall: '档口认证'",
    "stockist: '库存商认证'",
    "service_provider: '服务商认证'",
  ]) {
    assert.equal(combinedSource.includes(oldToken), false)
  }

  for (const oldOption of [
    "label=\"工厂\" value=\"factory\"",
    "label=\"档口\" value=\"stall\"",
    "label=\"库存商\" value=\"stockist\"",
    "label=\"服务商\" value=\"service_provider\"",
  ]) {
    assert.equal(merchantSource.includes(oldOption), false)
  }
})

test('resource type config edits category commercial rules', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/ResourceTypeConfigView.vue'), 'utf8')

  assert.match(source, /commercialRules/)
  assert.match(source, /发布规则/)
  assert.match(source, /联系方式查看/)
  assert.match(source, /付费或 VIP 免费/)
  assert.match(source, /priceCent/)
})
