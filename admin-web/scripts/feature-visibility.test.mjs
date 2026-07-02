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
    'src/views/DemandView.vue',
  ]

  const visibleSource = visibleFiles.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('match-cases'), false)
  assert.equal(visibleSource.includes('人工撮合'), false)
  assert.equal(visibleSource.includes('待撮合'), false)
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
  assert.match(layoutSource, /<span>首页运营位<\/span>/)
  assert.match(source, /v-if="form\.jumpType === 'webview'"/)
  assert.match(source, /v-else-if="form\.jumpType === 'internal'"/)
  assert.match(source, /internalPageOptions/)
  assert.match(source, /resourceOptions/)
  assert.match(source, /merchantOptions/)
  assert.match(source, /kindOptions/)
  assert.equal(source.includes('<el-option label="专题" value="topic" />'), false)
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

  assert.match(source, /必填字段控制商家发布或保存资源时必须补全的信息/)
  assert.match(source, /fieldDescriptionMap/)
  assert.match(source, /标题用于搜索、列表卡片和详情页主标题/)
  assert.match(source, /联系电话用于买家联系和平台审核核验/)
  assert.match(source, /required-field-note/)
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
  assert.match(layoutSource, /<span>热门搜索词<\/span>/)
  assert.match(apiSource, /\/api\/v1\/admin\/hot-search-keywords/)
  assert.match(viewSource, /<h2>热门搜索词<\/h2>/)
  assert.match(viewSource, /v-model="form\.keyword"/)
  assert.match(viewSource, /v-model="form\.status"/)
})

test('sourcing map admin is configurable from admin web', () => {
  const routeSource = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layoutSource = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'src/api/sourcingMap.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  assert.match(routeSource, /SourcingMapView/)
  assert.match(routeSource, /sourcing-map/)
  assert.match(layoutSource, /<span>拿货地图<\/span>/)
  assert.match(apiSource, /\/api\/v1\/admin\/map\/scenes/)
  assert.match(viewSource, /<h2>拿货地图<\/h2>/)
  assert.match(viewSource, /添加档口/)
  assert.match(viewSource, /添加配套/)
  assert.match(viewSource, /批量生成/)
  assert.match(viewSource, /map-canvas/)
  assert.match(viewSource, /listMapScenes/)
  assert.match(viewSource, /saveMapScene/)
  assert.match(viewSource, /publishMapScene/)
  assert.match(viewSource, /uploadMapBackgroundImage/)
  assert.match(viewSource, /v-model="sceneForm\.backgroundUrl"/)
})

test('admin city station filters use dropdown options', () => {
  const citySource = fs.readFileSync(path.join(root, 'src/common/cityStations.js'), 'utf8')
  const filterFiles = [
    'src/views/SearchLogView.vue',
    'src/views/DemandView.vue',
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

  const verificationSource = fs.readFileSync(path.join(root, 'src/views/VerificationView.vue'), 'utf8')
  assert.match(verificationSource, /<el-select[^>]+v-model="billingForm\.cityCode"/)
  assert.equal(/<el-input[^>]+v-model(?:\.trim)?="billingForm\.cityCode"/.test(verificationSource), false)
})

test('admin merchant identity wording matches mini program copy', () => {
  const merchantSource = fs.readFileSync(path.join(root, 'src/views/MerchantView.vue'), 'utf8')
  const verificationSource = fs.readFileSync(path.join(root, 'src/views/VerificationView.vue'), 'utf8')
  const entitlementSource = fs.readFileSync(path.join(root, 'src/views/EntitlementView.vue'), 'utf8')
  const bannerSource = fs.readFileSync(path.join(root, 'src/views/BannerTopicView.vue'), 'utf8')
  const identitySource = fs.readFileSync(path.join(root, 'src/common/merchantIdentity.js'), 'utf8')
  const combinedSource = [merchantSource, verificationSource, entitlementSource, bannerSource, identitySource].join('\n')

  for (const token of ['主要身份', '源头工厂', '现货档口', '库存货源', '配套服务']) {
    assert.match(combinedSource, new RegExp(token))
  }

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

test('verification review drawer shows submitted certification materials', () => {
  const source = fs.readFileSync(path.join(root, 'src/views/VerificationView.vue'), 'utf8')

  for (const token of [
    '营业主体',
    '统一社会信用代码',
    '联系人姓名',
    '联系电话',
    '联系微信',
    '经营地址',
    '营业执照',
    '门头/场地',
    '经营实拍',
    '授权证明',
    '其他证明',
    '未提交',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.match(source, /materialInfoItems/)
  assert.match(source, /materialImageItems/)
  assert.match(source, /socialCreditCode/)
  assert.match(source, /businessName/)
  assert.match(source, /licenseUrl/)
  assert.match(source, /storefrontUrl/)
  assert.match(source, /sceneUrl/)
  assert.match(source, /<el-image/)
})
