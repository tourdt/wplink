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
  assert.match(viewSource, /listMapObjects/)
  assert.match(viewSource, /saveMapObject/)
  assert.match(viewSource, /startDragObject/)
  assert.match(viewSource, /handleCanvasClick/)
  assert.match(viewSource, /geometryType/)
  assert.match(viewSource, /objectForm\.geometry/)
  assert.match(viewSource, /batchGenerateMapObjects/)
  assert.match(viewSource, /batchForm\.startCode/)
  assert.match(viewSource, /direction/)
})

test('sourcing map admin can maintain object tags and poi detail fields', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    '主营分类',
    '档口服务',
    '平台标签',
    '配套服务',
    '营业时间',
    '支持服务',
    '物流线路',
    '发货方式',
    '发车时间',
    '快递品牌',
    '收费说明',
    'categoryOptions',
    'serviceTagOptions',
    'platformTagOptions',
    'poiServiceTagOptions',
    'extraServiceOptions',
    'logisticsLineOptions',
    'deliveryTypeOptions',
    'expressBrandOptions',
    'objectForm.extra.openHours',
    'objectForm.extra.services',
    'objectForm.extra.lines',
    'objectForm.extra.deliveryTypes',
    'objectForm.extra.departureTime',
    'objectForm.extra.priceNote',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /categoryCodes:\s*objectForm\.categoryCodes \|\| \[\]/)
  assert.match(viewSource, /serviceTags:\s*objectForm\.serviceTags \|\| \[\]/)
  assert.match(viewSource, /platformTags:\s*objectForm\.platformTags \|\| \[\]/)
  assert.match(viewSource, /poiServiceTags:\s*objectForm\.poiServiceTags \|\| \[\]/)
  assert.match(viewSource, /extra:\s*normalizedExtra\(\)/)
})

test('sourcing map admin can maintain standard map categories', () => {
  const apiSource = fs.readFileSync(path.join(root, 'src/api/sourcingMap.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'saveMapCategory',
    '/api/v1/admin/map/categories',
    '标准标签',
    '新增标签',
    '保存标签',
    'categoryForm.code',
    'categoryForm.name',
    'categoryForm.type',
    'categoryFilters.type',
    'categoryFilters.status',
    'categoryTypeOptions',
    'categoryStatusOptions',
    'categoryOptionItems',
    'loadCategoryOptions',
    '全部类型',
    '全部状态',
    '筛选标签',
    'booth_category',
    'booth_service',
    'poi_service',
    'normal',
    'loadCategories',
    'submitCategory',
    'selectCategory',
    'mergedCategoryOptions',
    'mergedServiceTagOptions',
    'mergedPlatformTagOptions',
    'mergedPoiServiceTagOptions',
  ]) {
    const source = token === '/api/v1/admin/map/categories' || token === 'saveMapCategory' ? apiSource + viewSource : viewSource
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /listMapCategories\(\{\s*type:\s*categoryFilters\.type,\s*status:\s*categoryFilters\.status,\s*\}\)/)
  assert.match(viewSource, /listMapCategories\(\{\s*status:\s*'normal'\s*\}\)/)
  assert.match(viewSource, /function mapCategoryOptions\(type\)[\s\S]*categoryOptionItems\.value/)
})

test('sourcing map admin can filter map objects in a selected scene', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'objectFilters.keyword',
    'objectFilters.type',
    'objectFilters.status',
    'defaultObjectFilters',
    'clearObjectFilters',
    'objectTypeOptions',
    'objectStatusOptions',
    '筛选点位',
    '全部类型',
    '全部状态',
    '搜索编码/名称',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(
    viewSource,
    /listMapObjects\(sceneCode,\s*\{\s*types:\s*objectFilters\.type,\s*status:\s*objectFilters\.status,\s*keyword:\s*objectFilters\.keyword,\s*\}\)/,
  )
})

test('sourcing map admin can configure object display order', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    '点位排序',
    'objectForm.sort',
    'sort: objectForm.sort',
    'controls-position="right"',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /<el-input-number v-model="objectForm\.sort" :min="0" controls-position="right" \/>/)
})

test('sourcing map admin can configure object zoom visibility', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    '最小显示级别',
    '最大显示级别',
    'objectForm.minZoom',
    'objectForm.maxZoom',
    'minZoom: objectForm.minZoom',
    'maxZoom: objectForm.maxZoom',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /<el-input-number v-model="objectForm\.minZoom" :min="1" :max="5" controls-position="right" \/>/)
  assert.match(viewSource, /<el-input-number v-model="objectForm\.maxZoom" :min="1" :max="5" controls-position="right" \/>/)
})

test('sourcing map admin validates object zoom range before save', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'validateObjectZoomRange',
    '最小显示级别不能大于最大显示级别',
    '显示级别必须在 1 到 5 之间',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /function submitObject\(\)[\s\S]*if \(!validateObjectZoomRange\(\)\) \{[\s\S]*return[\s\S]*\}[\s\S]*saveMapObject/)
  assert.match(viewSource, /function validateObjectZoomRange\(\)[\s\S]*const minZoom = toNumber\(objectForm\.minZoom,\s*1\)[\s\S]*const maxZoom = toNumber\(objectForm\.maxZoom,\s*5\)/)
})

test('sourcing map admin syncs object layer from selected type', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'syncObjectLayerByType',
    'poiTypeValues',
    'packing_station',
    'logistics_point',
    'parking',
    'objectForm.layer',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /<el-select v-model="objectForm\.type" @change="syncObjectLayerByType">/)
  assert.match(viewSource, /objectForm\.layer = poiTypeValues\.has\(objectForm\.type\) \? 'poi' : 'booth'/)
})

test('sourcing map admin batch generation supports all object types', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    '批量生成点位',
    '点位宽',
    '点位高',
    'objectTypeOptions',
    'batchForm.type',
    'syncBatchLayerByType',
    'poiTypeValues',
    'factory_booth',
    'logistics_point',
    'restaurant',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /<el-option v-for="item in objectTypeOptions" :key="item\.value" :label="item\.label" :value="item\.value" \/>/)
  assert.match(viewSource, /<el-select v-model="batchForm\.type" @change="syncBatchLayerByType">/)
  assert.match(viewSource, /batchForm\.layer = poiTypeValues\.has\(batchForm\.type\) \? 'poi' : 'booth'/)
})

test('sourcing map admin can filter map scenes', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'sceneFilters.type',
    'sceneFilters.status',
    'defaultSceneFilters',
    'clearSceneFilters',
    'sceneTypeOptions',
    'sceneStatusOptions',
    '筛选场景',
    '全部场景类型',
    '全部场景状态',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(
    viewSource,
    /listMapScenes\(\{\s*cityCode:\s*defaultCityCode,\s*type:\s*sceneFilters\.type,\s*status:\s*sceneFilters\.status,\s*\}\)/,
  )
})

test('sourcing map admin can configure default scene viewport', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    '默认视野',
    '默认缩放',
    '默认中心 X',
    '默认中心 Y',
    '设为当前画布中心',
    'sceneForm.defaultScale',
    'sceneForm.defaultCenterX',
    'sceneForm.defaultCenterY',
    'setSceneDefaultCenterFromCanvas',
    'mapCanvasRef.value',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /v-model="sceneForm\.defaultCenterX"/)
  assert.match(viewSource, /v-model="sceneForm\.defaultCenterY"/)
  assert.match(viewSource, /sceneForm\.defaultCenterX = String\(centerX\)/)
  assert.match(viewSource, /sceneForm\.defaultCenterY = String\(centerY\)/)
})

test('sourcing map admin can quickly update object status', () => {
  const apiSource = fs.readFileSync(path.join(root, 'src/api/sourcingMap.js'), 'utf8')
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  assert.match(apiSource, /updateMapObjectStatus/)
  for (const token of [
    'updateMapObjectStatus',
    'changeObjectStatus',
    '状态操作',
    '设为正常',
    '设为隐藏',
    '设为歇业',
    '点位状态已更新',
    '点位缺少 ID',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /await updateMapObjectStatus\(row\.id,\s*status\)/)
})

test('sourcing map admin distinguishes inactive object status on canvas', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'objectStatusTagType',
    'objectStatusClass',
    'objectTitle',
    'object-status-badge',
    'status-hidden',
    'status-closed',
    ':title="objectTitle(object)"',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /objectStatusClass\(object\.status\)/)
  assert.match(viewSource, /<el-tag size="small" :type="objectStatusTagType\[row\.status\] \|\| 'info'">/)
})

test('sourcing map admin can locate a table object on the canvas', () => {
  const viewSource = fs.readFileSync(path.join(root, 'src/views/SourcingMapView.vue'), 'utf8')

  for (const token of [
    'mapCanvasRef',
    'locateObject',
    'scrollCanvasToObject',
    'object-row-actions',
    '定位',
    '@click.stop="locateObject(row)"',
    'scrollLeft',
    'scrollTop',
  ]) {
    assert.match(viewSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(viewSource, /const mapCanvasRef = ref\(null\)/)
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
