import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { validateFlows } from './validate-flows.mjs'

test('current wxapp pages satisfy MVP flow checks', () => {
  assert.deepEqual(validateFlows(path.resolve(new URL('..', import.meta.url).pathname)), [])
})

test('launch UI hides matching feature copy', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const files = [
    'pages/home/index.vue',
    'pages/login/index.vue',
    'pages/messages/index.vue',
    'pages/my/index.vue',
    'pages/market/index.vue',
    'pages/search/index.vue',
  ]

  const visibleSource = files.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('撮合'), false)
  assert.equal(visibleSource.includes('采购需求'), false)
  assert.equal(visibleSource.includes('登录后同步收藏、需求'), false)
})

test('cold-start public entries hide recruitment and job seeking', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const visibleSource = [
    'pages/home/index.vue',
    'pages/publish/index.vue',
    'pages/market/index.vue',
    'pages/search/index.vue',
  ].map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('招聘'), false)
  assert.equal(visibleSource.includes('求职'), false)
})

test('unreleased purchase demand pages are removed from wxapp source', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const mySource = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')
  const messagesSource = fs.readFileSync(path.join(root, 'pages/messages/index.vue'), 'utf8')

  for (const file of ['pages/demand/index.vue', 'pages/demand-success/index.vue', 'pages/my-demands/index.vue', 'api/demand.js']) {
    assert.equal(fs.existsSync(path.join(root, file)), false)
  }
  assert.equal(mySource.includes('我的需求'), false)
  assert.equal(mySource.includes('openMyDemands'), false)
  assert.equal(messagesSource.includes('/pages/my-demands/index'), false)
})

test('launch pages do not register unreleased purchase demand routes', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const pagePaths = pagesConfig.pages.map((item) => item.path)
  const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  for (const page of ['pages/demand/index', 'pages/demand-success/index', 'pages/my-demands/index']) {
    assert.equal(pagePaths.includes(page), false)
  }
  assert.equal(homeSource.includes("item.jumpType === 'demand'"), false)
  assert.equal(homeSource.includes('/pages/demand/index'), false)
})

test('launch pages do not register merchant certification route', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const pagePaths = pagesConfig.pages.map((item) => item.path)
  const visibleSource = [
    'pages/home/index.vue',
    'pages/my/index.vue',
    'pages/merchant/detail.vue',
    'pages/merchant/profile.vue',
    'pages/search/index.vue',
    'pages/market/index.vue',
  ].map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(pagePaths.includes('pages/verification/index'), false)
  assert.equal(fs.existsSync(path.join(root, 'pages/verification/index.vue')), false)
  assert.equal(fs.existsSync(path.join(root, 'api/verification.js')), false)
  assert.equal(visibleSource.includes('商家认证'), false)
  assert.equal(visibleSource.includes('认证状态'), false)
})

test('home banner only overlays labels and title on image', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes('banner-pill'), false)
  assert.equal(source.includes('banner-subtitle'), false)
  assert.match(source, /<image[^>]+class="banner-image"/)
  assert.match(source, /banner-kicker/)
  assert.match(source, /banner-title/)
})

test('home banner auto scrolls and shows bottom-right dots', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.match(source, /<swiper[\s\S]*class="banner-swiper"[\s\S]*autoplay[\s\S]*circular[\s\S]*@change="handleBannerChange"/)
  assert.match(source, /<swiper-item[\s\S]*v-for="\(\s*item,\s*index\s*\) in displayBanners"/)
  assert.match(source, /banner-dots/)
  assert.match(source, /activeBannerIndex === index/)
  assert.match(source, /\.banner-dots \{[\s\S]*right: 28rpx;[\s\S]*bottom: 24rpx;/)
})

test('home banner keeps slide spacing and native autoplay smooth', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.match(source, /<swiper[\s\S]*easing-function="easeInOutCubic"[\s\S]*@change="handleBannerChange"/)
  assert.doesNotMatch(source, /:current="activeBannerIndex"/)
  assert.match(source, /<swiper-item[^>]*class="banner-slide"[^>]*v-for="\(\s*item,\s*index\s*\) in displayBanners"/)
  assert.match(source, /\.banner-swiper \{[\s\S]*width: calc\(100% \+ 18rpx\);/)
  assert.match(source, /\.banner-slide \{[\s\S]*padding-right: 18rpx;/)
  assert.match(source, /\.banner-card \{[\s\S]*box-sizing: border-box;/)
})

test('home operation config loads banners and recommend cards together', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
  const discoverySource = fs.readFileSync(path.join(root, 'api/discovery.js'), 'utf8')

  assert.match(discoverySource, /listHomeOperationConfig/)
  assert.match(discoverySource, /\/api\/v1\/home\/operation-config/)
  assert.match(homeSource, /const recommendCards = ref\(\[\]\)/)
  assert.match(homeSource, /loadHomeOperationConfig/)
  assert.match(homeSource, /banners\.value = resp\.banners \|\| \[\]/)
  assert.match(homeSource, /recommendCards\.value = resp\.recommendCards \|\| \[\]/)
  assert.match(homeSource, /displayRecommendCard/)
  assert.match(homeSource, /openRecommendCard\(displayRecommendCard\)/)
  assert.match(homeSource, /recommend-card" v-if="displayRecommendCard"/)
  assert.equal(discoverySource.includes('/api/v1/home/banners'), false)
  assert.equal(discoverySource.includes('/api/v1/home/recommend-cards'), false)
  assert.equal(homeSource.includes("openSearch('小单快返')"), false)
  assert.equal(homeSource.includes('本周空档工厂：4 条针织生产线'), false)
})

test('home resources use dedicated discovery endpoint', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
  const discoverySource = fs.readFileSync(path.join(root, 'api/discovery.js'), 'utf8')

  assert.match(discoverySource, /listHomeResources/)
  assert.match(discoverySource, /\/api\/v1\/home\/resources/)
  assert.match(homeSource, /listHomeResources\(\{ cityCode: DEFAULT_CITY_CODE \}\)/)
  assert.match(homeSource, /loadFallbackHomeResources/)
  assert.match(homeSource, /listResources\([\s\S]*suppressErrorToast: true[\s\S]*\)/)
  assert.equal(homeSource.includes('pageSize: 2'), false)
})

test('home page keeps custom brand first screen structure', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes('search-divider'), false)
  assert.equal(source.includes('voice-icon'), false)
  assert.equal(source.includes('search-action-icon'), false)
  assert.equal(source.includes('brand-roof'), false)
  assert.equal(source.includes('brand-window-row'), false)
  assert.equal(source.includes('brand-icon'), false)
  assert.equal(source.includes('brand-hanger-hook'), false)
  assert.equal(source.includes('brand-hanger-line'), false)
  assert.equal(source.includes('brand-cargo-box'), false)
  assert.equal(source.includes('brand-cargo-tape'), false)

  for (const token of [
    'home-fixed-header',
    'custom-title-bar',
    'home-brand',
    'brand-logo',
    '/static/brand/yihuotong-header-logo.png',
    'mode="aspectFit"',
    'getMenuButtonBoundingClientRect',
    'homeContentStyle',
    '搜索供应、需求、工厂或服务',
    'factory-hero',
    '织里站 · 产业供需',
	    '童装产业带供需服务平台',
	    'quick-action-grid',
	    '童装批发',
	    '面料辅料',
	    '加工生产',
	    '商铺办公',
	    '住宅公寓',
	    '厂房仓库',
	    '本地服务',
	    '拿货档口',
	  ]) {
    assert.match(source, new RegExp(token))
  }
  assert.equal(source.includes('发需求'), false)
})

test('home search entry keeps a crisp icon and clear input surface', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.match(source, /<view class="search-icon" aria-hidden="true"><\/view>/)
  assert.match(source, /\.search-entry \{[\s\S]*background: #ffffff;[\s\S]*box-shadow: inset 0 0 0 1rpx rgba\(176, 186, 200, 0\.56\), 0 8rpx 18rpx rgba\(15, 23, 42, 0\.04\);/)
  assert.match(source, /\.search-icon::after \{[\s\S]*transform: rotate\(45deg\);/)
  assert.match(source, /\.search-placeholder \{[\s\S]*font-weight: 600;/)
})

test('resource tab separates recommendation discovery from keyword search page', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const resourceTab = pagesConfig.tabBar.list.find((item) => item.pagePath === 'pages/market/index')
  const resourceSource = fs.readFileSync(path.join(root, 'pages/market/index.vue'), 'utf8')
  const searchSource = fs.readFileSync(path.join(root, 'pages/search/index.vue'), 'utf8')
  const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
  const detailSource = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')
  const favoritesSource = fs.readFileSync(path.join(root, 'pages/favorites/index.vue'), 'utf8')

  assert.ok(pagesConfig.pages.some((item) => item.path === 'pages/search/index'))
  assert.equal(resourceTab?.text, '供需')

  for (const token of ['供需市场', 'openSearchPage', 'loadRecommendedResources', 'listResources', 'selectGroup', 'selectType', 'groupResourceTypes', 'ResourceFeedCard']) {
    assert.match(resourceSource, new RegExp(token))
  }
  for (const removedToken of ['direction-filter-row', 'directionFilterOptions', 'chooseResourceDirection', 'filters.direction', 'direction: filters.direction']) {
    assert.equal(resourceSource.includes(removedToken), false)
  }
  assert.equal(resourceSource.includes('activeDirection'), false)
  for (const removedToken of ['createSavedSearch', 'applySavedSearch', 'saveCurrentSearch']) {
    assert.equal(resourceSource.includes(removedToken), false)
  }

  for (const token of ['searchResources', '暂无匹配内容', 'emptyPrimaryActionLabel', 'selectGroup', 'groupResourceTypes', 'ResourceFeedCard']) {
    assert.match(searchSource, new RegExp(token))
  }
  for (const removedToken of ['direction-filter-row', 'directionFilterOptions', 'chooseResourceDirection', 'filters.direction', "direction: ''", 'normalizeResourceDirection']) {
    assert.equal(searchSource.includes(removedToken), false)
  }
  assert.equal(searchSource.includes('activeDirection'), false)
  for (const removedToken of ['提交采购需求', 'openDemand', '/pages/demand/index']) {
    assert.equal(searchSource.includes(removedToken), false)
  }
  for (const removedToken of ['DemandCard', 'ResourceCard', 'RESOURCE_DIRECTION_DEMAND']) {
    assert.equal(searchSource.includes(removedToken), false)
  }
  for (const removedToken of ['createSavedSearch', 'listSavedSearches', 'applySavedSearch', 'saveCurrentSearch', '保存搜索']) {
    assert.equal(searchSource.includes(removedToken), false)
  }

  assert.match(homeSource, /uni\.navigateTo\(\{ url: '\/pages\/search\/index' \}\)/)
  assert.match(detailSource, /uni\.navigateTo\(\{ url: '\/pages\/search\/index' \}\)/)
  for (const removedToken of ['searches', 'savedSearches', 'applySavedSearch', 'deleteSavedSearch', '暂无保存搜索']) {
    assert.equal(favoritesSource.includes(removedToken), false)
  }
  assert.match(favoritesSource, /<ResourceFeedCard v-for="item in favoriteResources"[\s\S]*@open="openResource"/)
})

test('publish page supports custom select fields from resource type schema', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.match(source, /allowCustom:\s*field\?\.allowCustom === true/)
  assert.match(source, /field\.type === 'select' && field\.allowCustom/)
  assert.match(source, /setDynamicFieldCustomSelect/)
  assert.match(source, /field\.options\.length/)
})

test('publish custom select fields reveal manual input only after choosing custom option', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.match(source, /getDynamicFieldSelectOptions/)
  assert.match(source, /isDynamicFieldCustomSelectActive\(field\)/)
  assert.match(source, /v-if="isDynamicFieldCustomSelectActive\(field\)"/)
  assert.doesNotMatch(
    source,
    /v-else-if="field\.type === 'select' && field\.allowCustom" class="select-with-custom"[\s\S]*@input="setDynamicFieldCustomSelect\(field, \$event\.detail\.value\)"/,
  )
})

test('publish tab uses primary category entry and secondary type sheet', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const tabSource = fs.readFileSync(path.join(root, 'pages/publish/index.vue'), 'utf8')
  const formSource = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  for (const token of [
    'listCityResourceTypes',
    'groupResourceTypes',
    'categoryGroups',
    '童装批发',
    '厂房仓库',
    '本地服务',
    'startPublishGroup',
    'startPublishCategory',
    'selectedCategoryGroup',
    'selectedCategoryItems',
    'showTypeSheet',
    '选择具体发布类型',
    'navigateToPublishForm',
    'requireLogin',
  ]) {
    assert.match(tabSource, new RegExp(token))
  }

  assert.equal(tabSource.includes('<ResourcePublishForm'), false)
  assert.equal(tabSource.includes('selectedDirection'), false)
  assert.equal(tabSource.includes('我能提供'), false)
  assert.equal(tabSource.includes('我想寻找'), false)
  assert.match(tabSource, /v-for="group in categoryGroups"[\s\S]*@click="startPublishGroup\(group\)"/)
  assert.match(tabSource, /v-for="item in selectedCategoryItems"[\s\S]*@click="startPublishCategory\(item\)"/)
  assert.doesNotMatch(tabSource, /v-for="group in categoryGroups"[\s\S]*v-for="item in group\.items"[\s\S]*@click="startPublishCategory\(item\)"/)
  assert.match(tabSource, /function startPublishGroup\(group\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*const items = group\?\.items \|\| \[\][\s\S]*if \(items\.length === 1\)[\s\S]*showTypeSheet\.value = true[\s\S]*\}/)
  assert.match(tabSource, /function startPublishCategory\(item\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm\(\{ typeCode: item\.typeCode \}\)[\s\S]*\}/)
  assert.match(tabSource, /async function applyPendingPublishType\(\) \{[\s\S]*const pendingPublish = uni\.getStorageSync\(PUBLISH_TYPE_KEY\)[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*uni\.removeStorageSync\(PUBLISH_TYPE_KEY\)[\s\S]*navigateToPublishForm\(\{ typeCode: pendingTypeCode \}\)/)
  assert.match(tabSource, /function navigateToPublishForm\(options = \{\}\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?\$\{query\.join\('&'\)\}` \}\)/)
  assert.match(tabSource, /initialPublishOptions\.typeCode && query\.push\(`typeCode=\$\{encodeURIComponent\(initialPublishOptions\.typeCode\)\}`\)/)
  assert.match(formSource, /direction:\s*RESOURCE_DIRECTION_SUPPLY/)
  assert.match(formSource, /listCityResourceTypes\(form\.cityCode\)/)
  assert.match(formSource, /form\.direction = normalizePublishDirection\(current\.direction \|\| ''\) \|\| RESOURCE_DIRECTION_SUPPLY/)
  assert.match(formSource, /需求描述/)
  assert.equal(formSource.includes('需求说明'), false)
  assert.match(formSource, /参考图片/)
  assert.equal(formSource.includes('采购要求'), false)
})

test('publish form renders type fields from schema instead of fixed category purchase fields', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const formSource = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.equal(formSource.includes('<text class="field-label">品类</text>'), false)
  assert.equal(formSource.includes("detailTitle: '采购要求'"), false)
  assert.equal(formSource.includes("quantityLabel: '需求数量'"), false)
  assert.equal(formSource.includes("priceLabel: '预算描述'"), false)
  assert.equal(formSource.includes("['typeCode', 'title', 'category', 'contactName', 'contactPhone'"), false)
  assert.equal(formSource.includes("uni.showToast({ title: '请填写品类'"), false)
  assert.match(formSource, /buildResourcePublishPayload/)
  assert.match(formSource, /applySummaryFieldsToPayload/)
  assert.match(formSource, /displayTemplate\?\.summary/)
})

test('resource cards and detail display configured summaries without fixed category quantity wording', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const resourceCardSource = fs.readFileSync(path.join(root, 'components/ResourceCard.vue'), 'utf8')
  const demandCardSource = fs.readFileSync(path.join(root, 'components/DemandCard.vue'), 'utf8')
  const detailSource = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')

  for (const source of [resourceCardSource, demandCardSource]) {
    assert.match(source, /resourceSummaryText/)
    assert.equal(source.includes('品类待沟通'), false)
    assert.equal(source.includes('数量待沟通'), false)
    assert.equal(source.includes('VIP'), false)
    assert.equal(source.includes('平台核实'), false)
    assert.equal(source.includes('已认证'), false)
  }
  assert.equal(demandCardSource.includes('预算面议'), false)
  assert.match(resourceCardSource, /v-if="resource\.priceText"/)
  assert.match(demandCardSource, /v-if="resource\.priceText"/)

  assert.match(detailSource, /resourceFeatureTags/)
  assert.match(detailSource, /buildResourceDetailPresentation/)
  assert.match(detailSource, /detailPresentation\.facts/)
  assert.match(detailSource, /<text class="desc">\{\{ resource\.description/)
  assert.match(detailSource, /<text class="section-title">补充说明<\/text>/)
  assert.equal(detailSource.includes('<text class="section-title">供应说明</text>'), false)
  assert.equal(detailSource.includes('summaryFactItems'), false)
  assert.equal(detailSource.includes('detail-price'), false)
  assert.match(detailSource, /summary-type/)
  assert.equal(detailSource.includes('summarySpecItems'), false)
  assert.equal(detailSource.includes('attributeSpecValues'), false)
  assert.equal(detailSource.includes("{ label: '品类', value: resource.value.category || '待沟通' }"), false)
  assert.equal(detailSource.includes("{ label: '数量', value: resource.value.quantityText || '待沟通' }"), false)
  assert.equal(detailSource.includes("{ label: '价格', value: resource.value.priceText || '面议' }"), false)
})

test('home quick actions map to supply and demand resource flows', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes("const RESOURCE_DIRECTION_SUPPLY = 'supply'"), false)
  assert.equal(source.includes("const RESOURCE_DIRECTION_DEMAND = 'demand'"), false)
  assert.match(source, /\{ title: '童装批发'[\s\S]*icon: 'market'[\s\S]*groupCode: 'kids_wholesale'/)
  assert.match(source, /\{ title: '面料辅料'[\s\S]*icon: 'clearance'[\s\S]*groupCode: 'materials'/)
  assert.match(source, /\{ title: '加工生产'[\s\S]*icon: 'factory'[\s\S]*groupCode: 'production'/)
  assert.doesNotMatch(source, /\{ title: '招聘求职'[\s\S]*groupCode: 'jobs'/)
  assert.match(source, /\{ title: '厂房仓库'[\s\S]*groupCode: 'factory_warehouse'/)
  assert.match(source, /\{ title: '拿货档口'[\s\S]*icon: 'map'[\s\S]*action: 'sourcing-map'/)
  assert.match(source, /\.quick-action-grid \{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/)
  assert.match(source, /item\.icon === 'market'/)
  assert.match(source, /item\.icon === 'clearance'/)
  assert.match(source, /item\.icon === 'orders'/)
  assert.match(source, /item\.icon === 'map'/)
  assert.equal(source.includes("item.icon === 'demand'"), false)
  assert.equal(source.includes("action: 'publish-demand'"), false)
  assert.equal(source.includes("item.action === 'publish-demand'"), false)
  assert.match(source, /function openSearch\(options = \{\}\) \{[\s\S]*typeof options === 'string'[\s\S]*uni\.setStorageSync\(SEARCH_KEY, searchOptions\)[\s\S]*uni\.navigateTo\(\{ url: '\/pages\/search\/index' \}\)/)
  assert.match(source, /function openPublish\(options = \{\}\) \{[\s\S]*typeof options === 'string'[\s\S]*uni\.setStorageSync\(PUBLISH_TYPE_KEY, publishOptions\)[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)/)

  for (const removedToken of ["action: 'demand'", 'openDemand', "action: 'publish'"]) {
    assert.equal(source.includes(removedToken), false)
  }
  for (const removedText of ['quick-desc', "desc: '现货货源'", "desc: '发布库存'", "desc: '工厂产能'", "desc: '订单需求'", '我要找货', '我要清货', '我要找厂', '我要接单']) {
    assert.equal(source.includes(removedText), false)
  }
})

test('resource type display names use buyer-friendly wording', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const enumSource = fs.readFileSync(path.join(root, 'common/enums.js'), 'utf8')

  for (const token of [
    "stock_clearance: '库存出售'",
    "factory_direct: '工厂直批'",
    "buy_kids_goods: '求购尾货'",
    "job_hiring: '招聘员工'",
    "job_seeking: '我要求职'",
    "factory_warehouse_rental: '厂房/仓库出租'",
    "seek_factory_warehouse: '我要求租'",
    "find_factory: '招加工厂'",
  ]) {
    assert.match(enumSource, new RegExp(token))
  }
  for (const oldText of ["goods: '货源'", "factory: '工厂产能'", "order: '订单找厂'", "order: '订单需求'", "job: '招聘'", "rental: '出租'", "service: '服务'"]) {
    assert.equal(enumSource.includes(oldText), false)
  }
})

test('topic empty state does not expose demand submission in MVP', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/topic/index.vue'), 'utf8')

  for (const token of ['getTopicResources', 'ResourceFeedCard', 'Banner 专题', 'topicStats', '继续浏览供应', 'openSearch']) {
    assert.match(source, new RegExp(token))
  }

  for (const removedToken of ['demandEntry', 'openDemand', '提交找货需求', '提交需求', '/pages/demand/index', '可提交需求']) {
    assert.equal(source.includes(removedToken), false)
  }
})

test('my page separates guest and logged-in account states without merchant binding', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  for (const token of [
    'isLoggedIn',
    '未登录',
    '微信登录',
    '我的账号',
    'merchantNo',
    'merchantNoVisible',
    'account-number-row',
    '编号：',
    '复制',
    'copyMerchantNo',
    '编号已复制',
    'accountStatusText',
    '待完善',
    'openAccountCard',
    'openMerchantHome',
    '我的主页',
    '查看对外展示资料',
    '我的权益',
    '暂无可领取权益',
    'ensureMerchantProfileReady',
    'openMessages',
    'requireLogin',
  ]) {
    assert.match(source, new RegExp(token))
  }

  for (const hiddenToken of ['保存身份', '商家 ID', '用户 ID：', '主页配置', 'merchant-actions', '权益提醒', '手机号绑定', '登录后可用', '同步收藏关注', '接收审核和联系消息', '我的需求', 'openMyDemands', '登录后管理收藏和发布记录', '已登录，可管理收藏和消息', 'VIP 权益', '查看额度和限时特价', 'openVIP', '/pages/vip/index?merchantId=']) {
    assert.equal(source.includes(hiddenToken), false)
  }
})

test('my page presents merchant workspace and grouped service entries', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  for (const token of [
    'account-shell',
    'account-side',
    'account-title-main',
    'account-number-row',
    'account-number-copy',
    'width: 104rpx',
    'font-size: 38rpx',
    'common-service-section',
    '我的发布',
    '状态、数据、推广',
    'openMyResources',
    '/pages/my-resources/index',
    'entry-arrow',
    'verification-status::before',
    'border-radius: 999rpx',
    'min-height: 32rpx',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('grid-template-columns: 104rpx minmax(0, 1fr)'), true)
  assert.equal(source.includes('常用服务'), false)
  assert.match(source, /<view class="action-list">\s*<view class="action-item" @click="openMyResources">[\s\S]*<text class="action-title">我的发布<\/text>/)

  for (const verboseCopy of [
    '进入我的发布查看每条供需信息的表现',
    '进入我的发布查看每条供需信息表现',
    'quick-entry-grid',
    'quick-entry',
    '发布和推广',
    '发布供需信息',
    'openPublish',
    'quick-entry.primary {\n  background: $wplink-primary',
    'quick-entry.primary .quick-desc {\n  color: rgba(255, 255, 255',
    '商家资料待完善',
    '认证审核中',
    '认证未通过',
    '认证已撤销',
    '资料和认证',
    '个人服务',
    'class="workspace-section section-card"',
    'class="service-section section-card"',
    '优先处理发布、管理和曝光相关动作',
    '完善资料后获取更完整的商家展示和认证能力',
    '集中查看采购、收藏和平台提醒',
    'merchant-effect-card',
    'merchantEffectVisible',
    'merchantEffectItems',
    'getMerchantMetricsSummary',
    '商家本周效果',
    '近 7 天',
    '曝光',
    '浏览',
  ]) {
    assert.equal(source.includes(verboseCopy), false)
  }
})

test('my resources page keeps list concise and dates day-only', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const myResourcesPage = pagesConfig.pages.find((item) => item.path === 'pages/my-resources/index')

  assert.equal(myResourcesPage?.style?.enablePullDownRefresh, true)

  for (const token of [
    'displayDateOrPlaceholder(item.publishedAt)',
    'displayDateOrPlaceholder(item.expiresAt)',
    'function displayDateOrPlaceholder(value)',
    "return value ? formatDateToDay(value) : '-'",
    'class="publish-fab"',
    'position: fixed;',
    'top: 0;',
    'padding-top: 132rpx;',
    'background: $wplink-card;',
    'box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);',
    'displayStatusText(item)',
    'isActivePublished(item)',
    'isExpiredResource(item)',
    '<view class="filter-row">',
    'grid-template-columns: repeat(4, minmax(0, 1fr));',
    'onPullDownRefresh',
    'onReachBottom',
    'uni.stopPullDownRefresh()',
    'loadRows({ reset: true })',
    'loadRows({ reset: false })',
    'page.value',
    'hasMore.value',
    'loading.value',
    'class="empty-state"',
    '暂无发布内容',
    '继续发布',
    'load-more-text',
    'padding-bottom: calc(128rpx + env(safe-area-inset-bottom));',
    'flex: 1;',
    'min-height: 420rpx;',
    "{ label: '待跟进', value: 'needs_action' }",
    "{ label: '展示中', value: 'showing' }",
    "{ label: '已结束', value: 'ended' }",
    'rejectReason',
    'openRejectedEditor',
    '驳回原因',
    "if (item.dealtAt) return '已成交'",
    "if (isExpiredResource(item)) return '已过期'",
    "return isExpiredResource(item) || Boolean(item.dealtAt)",
  ]) {
    assert.equal(source.includes(token), true)
  }

  for (const removedToken of [
    'markResourceDeal',
    'markDealt',
    '>成交</button>',
    'lifecycle-note',
    'effectAdvice',
    'effect-advice',
    '<scroll-view class="filter-row" scroll-x>',
    '已发布供需信息可刷新、置顶或下架，过期后可再发类似。',
    '审核通过后开始展示。',
    '可再发类似供需信息继续曝光。',
    '根据曝光和联系情况刷新或置顶。',
    'directionOptions',
    '全部发布',
    '供应发布',
    '需求发布',
    'direction: filters.direction',
    'selectDirection',
    'direction-row',
    'direction-button',
    '完善信息有助于买家判断。',
    '管理供需信息状态、效果数据和推广权益',
    '管理供需信息状态和推广效果',
    'resource-manager-head',
    'manager-title',
    'manager-desc',
    "{ label: '草稿', value: 'draft' }",
    "{ label: '待审核', value: 'pending' }",
    "{ label: '已驳回', value: 'rejected' }",
    "{ label: '已发布', value: 'published' }",
    "{ label: '即将过期', value: 'expiring_soon' }",
    "{ label: '已过期', value: 'expired' }",
    "{ label: '已成交', value: 'dealt' }",
    "{ label: '已下架', value: 'taken_down' }",
  ]) {
    assert.equal(source.includes(removedToken), false)
  }

  assert.match(source, /\.filter-button \{[\s\S]*border: 2rpx solid transparent;[\s\S]*background: #f4f7fd;[\s\S]*transition: background 0\.18s ease, color 0\.18s ease, border-color 0\.18s ease, box-shadow 0\.18s ease;/)
  assert.match(source, /\.filter-button\.active \{[\s\S]*border-color: \$wplink-primary;[\s\S]*background: \$wplink-primary;[\s\S]*color: \$wplink-card;[\s\S]*font-weight: 700;[\s\S]*box-shadow: 0 8rpx 18rpx rgba\(194, 58, 0, 0\.18\);/)
})

test('favorites page matches my resources filter and supports refresh pagination empty states', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/favorites/index.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const favoritesPage = pagesConfig.pages.find((item) => item.path === 'pages/favorites/index')

  assert.equal(favoritesPage?.style?.enablePullDownRefresh, true)

  for (const token of [
    '<view class="filter-row">',
    'filter-button',
    'selectTab(item.value)',
    'position: fixed;',
    'top: 0;',
    'padding-top: 132rpx;',
    'background: $wplink-card;',
    'box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.06);',
    'onPullDownRefresh',
    'onReachBottom',
    'uni.stopPullDownRefresh()',
    'loadRows({ reset: true })',
    'loadRows({ reset: false })',
    'page.value',
    'hasMore.value',
    'loading.value',
    'class="empty-state"',
    'emptyTitle',
    'emptyDesc',
    'emptyActionText',
    'openEmptyAction',
    '暂无收藏供应',
    '暂无关注商家',
    '去找供应',
    '去找商家',
    'load-more-text',
    'listFavoriteResources({ page: nextPage, pageSize })',
    'listFollowedMerchants({ page: nextPage, pageSize })',
    "import MerchantBadge from '../../components/MerchantBadge.vue'",
    '<MerchantBadge :merchant="item" />',
    'merchantAvatarUrl(item)',
    'merchantAvatarText(item)',
    'merchantBusinessText(item)',
    'mainCategories',
    'merchant-avatar',
    'merchant-arrow',
  ]) {
    assert.equal(source.includes(token), true)
  }

  assert.match(source, /\.filter-row \{[\s\S]*position: fixed;[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);[\s\S]*background: \$wplink-card;[\s\S]*box-shadow: 0 8rpx 20rpx rgba\(15, 23, 42, 0\.06\);/)
  assert.match(source, /\.filter-button \{[\s\S]*border: 2rpx solid transparent;[\s\S]*background: #f4f7fd;[\s\S]*transition: background 0\.18s ease, color 0\.18s ease, border-color 0\.18s ease, box-shadow 0\.18s ease;/)
  assert.match(source, /\.filter-button\.active \{[\s\S]*border-color: \$wplink-primary;[\s\S]*background: \$wplink-primary;[\s\S]*color: \$wplink-card;[\s\S]*font-weight: 700;[\s\S]*box-shadow: 0 8rpx 18rpx rgba\(194, 58, 0, 0\.18\);/)
  assert.match(source, /\.merchant-item \{[\s\S]*grid-template-columns: 88rpx minmax\(0, 1fr\) 28rpx;[\s\S]*align-items: center;[\s\S]*gap: 18rpx;/)
  assert.match(source, /\.merchant-info :deep\(\.merchant-badge\) \{[\s\S]*min-width: 0;[\s\S]*flex-wrap: wrap;/)
})

test('my resources draft resources open editor instead of direct submit', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')

  assert.match(source, /item\.status === 'draft'[\s\S]*@click="openDraftEditor\(item\)"[\s\S]*>编辑<\/button>/)
  assert.match(source, /function openDraftEditor\(item\) \{[\s\S]*openPublishEditor\(item\)/)
  assert.match(source, /function openPublishEditor\(item\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}&resourceId=\$\{item\.id\}` \}\)/)
  assert.equal(source.includes('submitDraft'), false)
  assert.equal(source.includes('submitResource'), false)
  assert.equal(source.includes('已提交审核'), false)
  assert.equal(source.includes('publish:pending-edit-context'), false)
  assert.match(source, /async function openPublish\(\) \{[\s\S]*ensurePageMerchantProfile\(\)[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)[\s\S]*\}/)
  assert.equal(source.includes("uni.switchTab({ url: '/pages/publish/index' })"), true)
})

test('my resources repost similar opens a new resource form with old resource defaults', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')
  const formSource = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.match(source, /getOwnResource/)
  assert.match(source, /function buildRepostInitialForm\(detail\)/)
  assert.match(source, /repostInitialForm/)
  assert.match(source, /uni\.setStorageSync\('publish:repost-initial-form'/)
  assert.match(source, /uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}&repost=1` \}\)/)
  assert.equal(source.includes('repostSimilarResource'), false)
  assert.equal(source.includes('已复制为草稿'), false)

  assert.match(formSource, /restoreRepostInitialForm/)
  assert.match(formSource, /applyInitialPublishForm/)
  assert.match(formSource, /editingResourceId\.value = ''/)
  assert.match(formSource, /uni\.removeStorageSync\('publish:repost-initial-form'\)/)
})

test('publish pages split tab creation and independent editing', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const tabSource = fs.readFileSync(path.join(root, 'pages/publish/index.vue'), 'utf8')
  const editSource = fs.readFileSync(path.join(root, 'pages/publish/edit.vue'), 'utf8')
  const myResourcesSource = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')

  assert.equal(tabSource.includes('<ResourcePublishForm'), false)
  assert.match(tabSource, /PUBLISH_TYPE_KEY/)
  assert.match(tabSource, /onShow\(applyPendingPublishType\)/)
  assert.match(tabSource, /async function applyPendingPublishType\(\) \{[\s\S]*uni\.getStorageSync\(PUBLISH_TYPE_KEY\)[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm\(\{ typeCode: pendingTypeCode \}\)/)
  assert.match(editSource, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(editSource, /import \{ getMerchantId \} from '\.\.\/\.\.\/store\/session'/)
  assert.match(editSource, /onLoad\(\(options\)/)
  assert.match(editSource, /direction: ''/)
  assert.match(editSource, /if \(!requireLogin\(\)\) return/)
  assert.match(editSource, /routeOptions\.merchantId = options\.merchantId \|\| getMerchantId\(\)/)
  assert.match(editSource, /routeOptions\.direction = options\.direction \|\| ''/)
  assert.doesNotMatch(editSource, /ensureMerchantProfileReady/)
  assert.match(editSource, /const publishFormMode = computed\(\(\) => routeOptions\.resourceId \? 'edit' : 'create'\)/)
  assert.match(editSource, /<ResourcePublishForm[\s\S]*:mode="publishFormMode"[\s\S]*:initial-options="routeOptions"/)
  assert.equal(tabSource.includes('publish:pending-edit-context'), false)
  assert.match(myResourcesSource, /async function openPublish\(\) \{[\s\S]*ensurePageMerchantProfile\(\)[\s\S]*uni\.removeStorageSync\('wplink_pending_publish_type_code'\)[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)/)
})

test('publish success page highlights my resources as the primary action', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/publish-success/index.vue'), 'utf8')

  assert.match(source, /<button class="wplink-primary-button action-button" @click="openMyResources">查看我的发布<\/button>/)
  assert.match(source, /<button class="wplink-secondary-button action-button" @click="openMessages">查看消息<\/button>/)
})

test('merchant profile page labels every field and removes manual image url entry', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    '展示名称',
    '主要身份',
    '主营内容',
    '联系人',
    '联系电话',
    '微信',
    '经营地址',
    '简介',
    '展示图片',
    '公开展示图片',
    'form-field',
    'field-label',
    'image-helper',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('也可粘贴图片 URL'), false)
  assert.equal(source.includes('图片 URL'), false)
})

test('merchant profile page uses neutral profile setup wording and keeps basic field order', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const profilePage = pagesConfig.pages.find((entry) => entry.path === 'pages/merchant/profile')

  assert.equal(profilePage?.style?.navigationBarTitleText, '资料设置')
  for (const token of [
    '展示名称',
    '请输入展示名称',
    '主要身份',
    '主营内容',
    '如：童装现货、女装尾货、厂房出租、档口转让、设备转让',
    '请填写主营内容',
    '简介',
    '经营地址',
    '头像 / LOGO',
    '展示图片',
    '资料加载失败',
    '资料已保存',
    '资料保存失败',
  ]) {
    assert.match(source, new RegExp(token))
  }

  const basicSectionEndIndex = source.indexOf('<view class="form-section contact-section">')
  const basicSectionSource = source.slice(0, basicSectionEndIndex)
  const logoIndex = basicSectionSource.indexOf('头像 / LOGO')
  const nameIndex = source.indexOf('展示名称')
  const identityIndex = source.indexOf('主要身份')
  const categoryIndex = source.indexOf('主营内容')
  const introIndex = source.indexOf('简介')
  const imageIndex = source.indexOf('展示图片')
  assert.ok(logoIndex > -1 && logoIndex < nameIndex)
  assert.ok(nameIndex < identityIndex)
  assert.ok(identityIndex < categoryIndex)
  assert.ok(categoryIndex < introIndex)
  assert.ok(introIndex < imageIndex && imageIndex < basicSectionEndIndex)
  assert.doesNotMatch(source, /:disabled="Boolean\(merchantId\)" placeholder="请输入展示名称"/)
  assert.match(source, /const patch = \{[\s\S]*name: form\.name\.trim\(\),[\s\S]*mainCategories,/)

  for (const token of [
    '商家名称',
    '商家介绍',
    '商家地址',
    '商家 LOGO',
    '主营品类',
    '选择最主要的经营身份',
    '经营简介',
    '品牌展示',
    '主页展示',
    '补充公开展示图片',
    '提交入驻',
    '商家资料加载失败',
    '商家资料已保存',
    '商家资料保存失败',
    '选择最接近你发布供需信息的身份',
    '包装、辅料、印花绣花、物流等选配套服务',
  ]) {
    assert.equal(source.includes(token), false)
  }
})

test('merchant identity wording is unified across profile and display pages', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const enumSource = fs.readFileSync(path.join(root, 'common/enums.js'), 'utf8')
  const displaySources = [
    'pages/merchant/detail.vue',
    'pages/resource/detail.vue',
    'pages/favorites/index.vue',
  ].map((file) => fs.readFileSync(path.join(root, file), 'utf8'))

  for (const token of [
    '主要身份',
    '个人',
    '源头工厂',
    '库存货源',
    '配套服务',
    '采购',
  ]) {
    assert.match(profileSource, new RegExp(token))
  }
  assert.match(profileSource, /const DEFAULT_MERCHANT_TYPE = 'individual'/)
  assert.doesNotMatch(profileSource, /\{ label: '现货档口', value: 'stall' \}/)
  assert.doesNotMatch(profileSource, /\{ label: '场地\/设备方', value: 'rental_provider' \}/)
  assert.doesNotMatch(profileSource, /merchantTypeOptions = \[[\s\S]*value: 'stall'/)
  assert.doesNotMatch(profileSource, /merchantTypeOptions = \[[\s\S]*value: 'rental_provider'/)
  assert.doesNotMatch(profileSource, /选择最接近你发布供需信息的身份/)
  assert.doesNotMatch(profileSource, /包装、辅料、印花绣花、物流等选配套服务/)

  for (const source of displaySources) {
    for (const token of ['个人', '场地/设备方', '源头工厂', '现货档口', '库存货源', '配套服务', '采购']) {
      assert.match(source, new RegExp(token))
    }
  }
  for (const token of ['个人', '场地/设备方', '源头工厂', '现货档口', '库存货源', '配套服务', '采购']) {
    assert.match(enumSource, new RegExp(token))
  }

  assert.equal(profileSource.includes("label: '工厂'"), false)
  assert.equal(profileSource.includes("label: '档口'"), false)
  assert.equal(profileSource.includes("label: '库存商'"), false)
  assert.equal(profileSource.includes("label: '服务商'"), false)
  assert.equal(enumSource.includes("factory: '工厂'"), false)
  assert.equal(enumSource.includes("stall: '档口'"), false)
  assert.equal(enumSource.includes("stockist: '库存商'"), false)
  assert.equal(enumSource.includes("service_provider: '服务商'"), false)
})

test('publish page presents grouped fast publishing workflow', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  for (const token of [
    'form-section basic-section',
    'form-section image-section',
    'form-section contact-section',
    'field-label',
    'UniGrid',
    'UniGridItem',
    'resourceImageGridItems',
    'resourceImageMaxCount',
    'onResourceImageGridItemClick',
    'image-count',
    'image-grid-wrap',
    'upload-img-item',
    'upload-img-add-container',
    'img-del',
    'previewResourceImage',
    'removeResourceImage',
    'getMerchant',
    'getEditableResource',
    'updateResourceDraft',
    'submitResource',
    'editingResourceId',
    'editingResourceStatus',
    'loadEditableResource',
    'loadMerchantContact',
    'applyMerchantContactDefaults',
    'contact.name',
    'contact.phone',
    'contact.phoneMasked',
    'contact.wechat',
    'sanitizeContactWechat',
    'fixed-save-spacer',
    'fixed-save-bar',
    'fixed-save-actions',
    '供应描述',
    '类型字段',
    '供应图片',
    '联系信息',
    '保存草稿',
    '提交审核',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('class="form-card"'), false)
  assert.equal(source.includes('供应说明'), false)
  assert.equal(source.includes('class="image-url"'), false)
  assert.equal(source.includes('type-section section-card'), false)
  assert.equal(source.includes('publish-hero'), false)
  assert.equal(source.includes('publish-benefits'), false)
  assert.equal(source.includes('认证商家权益'), false)
  assert.equal(source.includes('发布后可获得'), false)
  assert.equal(source.includes('发布类型'), false)
  assert.equal(source.includes('publishTypeOptions'), false)
  assert.equal(source.includes('selectTypeByCode'), false)
  assert.equal(source.includes('category-lock-card'), false)
  assert.equal(source.includes('category-lock-main'), false)
  assert.equal(source.includes('category-lock-sub'), false)
  assert.equal(source.includes('category-lock-label'), false)
  assert.equal(source.includes('category-lock-badge'), false)
  assert.equal(source.includes('selectedGroupName'), false)
  assert.equal(source.includes('selectedTypeLabel'), false)
  assert.equal(source.includes('发布类目'), false)
  assert.equal(source.includes('已选择'), false)
  assert.equal(source.includes('basic-progress'), false)
  assert.equal(source.includes('completion-percent'), false)
  assert.equal(source.includes('completion-bar-fill'), false)
  assert.equal(source.includes('<picker :range="resourceTypeNames"'), false)
  assert.equal(source.includes('function selectType(event)'), false)
  assert.equal(source.includes('publish-types-in-basic'), false)
  assert.equal(source.includes('image-upload-tile'), false)
  assert.equal(source.includes('image-preview-grid'), false)
  assert.equal(source.includes('sticky-action-bar'), false)
  assert.equal(source.includes('padding: 24rpx 24rpx 176rpx'), false)
  assert.equal(source.includes('点击图片预览，点击最后一格添加'), false)
  assert.equal(source.includes('form-section supply-section'), false)
  assert.match(source, /reserveBottomSafeArea/)
  assert.match(source, /fixed-save-spacer', \{ 'no-safe-area': !reserveBottomSafeArea \}/)
  assert.match(source, /fixed-save-bar', \{ 'no-safe-area': !reserveBottomSafeArea \}/)
  assert.match(source, /\.fixed-save-bar \{[\s\S]*position: fixed;[\s\S]*right: 0;[\s\S]*bottom: 0;[\s\S]*left: 0;[\s\S]*padding: 10rpx 24rpx calc\(4rpx \+ env\(safe-area-inset-bottom\)\);[\s\S]*border-top: 1rpx solid \$wplink-line;[\s\S]*background: rgba\(255, 255, 255, 0\.96\);[\s\S]*\}/)
  assert.match(source, /\.fixed-save-spacer \{[\s\S]*height: calc\(102rpx \+ env\(safe-area-inset-bottom\)\);[\s\S]*\}/)
  assert.match(source, /\.fixed-save-spacer\.no-safe-area \{[\s\S]*height: 102rpx;[\s\S]*\}/)
  assert.match(source, /\.fixed-save-bar\.no-safe-area \{[\s\S]*padding-bottom: 4rpx;[\s\S]*\}/)
})

test('publish page renders resource type field schema into attributes', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  for (const token of [
    'dynamicFieldItems',
    'fieldSchema',
    'form.attributes[field.key]',
    'getDynamicFieldValue',
    'setDynamicFieldValue',
    'setDynamicFieldBoolean',
    "field.type === 'boolean'",
    'syncAttributesWithSelectedType',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.match(source, /v-for="field in dynamicFieldItems"/)
  assert.match(source, /:placeholder="field\.placeholder \|\| `请填写\$\{field\.label\}`"/)
  assert.match(source, /:class="\['toggle-option', getDynamicFieldValue\(field\.key\) === true \? 'active' : ''\]"/)
  assert.match(source, /:class="\['toggle-option', getDynamicFieldValue\(field\.key\) === false \? 'active' : ''\]"/)
  assert.match(source, /function syncAttributesWithSelectedType\(\) \{[\s\S]*delete nextAttributes\[key\][\s\S]*\}/)
})

test('publish tab page does not reserve bottom safe area for fixed save bar', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const publishSource = fs.readFileSync(path.join(root, 'pages/publish/index.vue'), 'utf8')
  const editSource = fs.readFileSync(path.join(root, 'pages/publish/edit.vue'), 'utf8')

  assert.equal(publishSource.includes('reserve-bottom-safe-area'), false)
  assert.equal(editSource.includes('reserve-bottom-safe-area'), false)
})

test('publish page defaults contact phone from merchant profile masked phone', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  assert.match(source, /const detail = await getMerchant\(form\.merchantId,\s*\{ suppressErrorToast: true \}\)/)
  assert.match(source, /contact\.phone \|\| contact\.phoneMasked/)
  assert.match(source, /form\.contact\.phone = contact\.phone \|\| contact\.phoneMasked/)
})

test('publish page autosaves local draft and clears it after successful save or submit', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  for (const token of [
    'watch',
    'onUnmounted',
    'publishLocalDraftStorageKey',
    'buildPublishLocalDraftStorageKey',
    'restorePublishLocalDraft',
    'scheduleSavePublishLocalDraft',
    'savePublishLocalDraft',
    'flushPublishLocalDraft',
    'clearPublishLocalDraft',
    'resetPublishForm',
    'autosaveReady',
    'localDraftSaveTimer',
    'uni.setStorageSync',
    'uni.getStorageSync',
    'uni.removeStorageSync',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.match(source, /watch\(\s*form,[\s\S]*scheduleSavePublishLocalDraft[\s\S]*deep: true/)
  assert.match(source, /watch\(\s*resourceImageEntries,[\s\S]*scheduleSavePublishLocalDraft[\s\S]*deep: true/)
  assert.match(source, /onUnmounted\(\(\) => \{[\s\S]*flushPublishLocalDraft\(\)[\s\S]*\}\)/)

  const submitMatched = source.match(/async function submit\(\) \{([\s\S]*?)\n\}/)
  const saveDraftMatched = source.match(/async function saveDraft\(\) \{([\s\S]*?)\n\}/)
  assert.ok(submitMatched, 'submit should exist')
  assert.ok(saveDraftMatched, 'saveDraft should exist')
  assert.match(submitMatched[1], /await createResource[\s\S]*clearPublishLocalDraft\(\)[\s\S]*resetPublishForm\(\)|await submitResource/)
  assert.match(saveDraftMatched[1], /await saveResourceDraftPayload[\s\S]*clearPublishLocalDraft\(\)[\s\S]*resetPublishForm\(\)/)
})

test('publish page stages images and uploads them only when saving or submitting', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'components/ResourcePublishForm.vue'), 'utf8')

  for (const token of [
    'chooseImageFile',
    'uploadSelectedImage',
    'resourceImageEntries',
    'createPendingResourceImageEntry',
    'createStoredResourceImageEntry',
    'getResourceImagePreviewUrls',
    'uploadPendingResourceImages',
    'await uploadPendingResourceImages()',
    "await uploadSelectedImage(entry.file, 'resource')",
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  const submitMatched = source.match(/async function submit\(\) \{([\s\S]*?)\n\}/)
  const saveDraftMatched = source.match(/async function saveDraft\(\) \{([\s\S]*?)\n\}/)
  const uploadResourceImageMatched = source.match(/async function uploadResourceImage\(\) \{([\s\S]*?)\n\}/)

  assert.ok(submitMatched, 'submit should exist')
  assert.ok(saveDraftMatched, 'saveDraft should exist')
  assert.ok(uploadResourceImageMatched, 'uploadResourceImage should exist')
  assert.match(submitMatched[1], /await uploadPendingResourceImages\(\)[\s\S]*(await createResource|await submitResource)/)
  assert.match(saveDraftMatched[1], /await uploadPendingResourceImages\(\)[\s\S]*await saveResourceDraftPayload/)
  assert.equal(uploadResourceImageMatched[1].includes('uploadSelectedImage'), false)
  assert.equal(source.includes('chooseAndUploadImage'), false)
  assert.equal(source.includes('图片已上传'), false)
})

test('success pages explain the result and next step consistently', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const publishSource = fs.readFileSync(path.join(root, 'pages/publish-success/index.vue'), 'utf8')

  for (const token of [
    'success-icon',
    'success-result-list',
    'success-result-item',
    'result-label',
    'result-value',
    'wplink-primary-button',
    'wplink-secondary-button',
  ]) {
    assert.match(publishSource, new RegExp(token))
  }

  for (const token of ['发布状态', '自动检测中', '发布后曝光', '搜索、推荐和供应方资料']) {
    assert.match(publishSource, new RegExp(token))
  }
})

test('merchant profile page does not require sms verification for contact phone', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(source, /sanitizeContactPhone/)
  assert.match(source, /isValidContactPhone/)
  assert.match(source, /手机号需为 6-20 位数字/)
  assert.equal(source.includes('sendSmsCodeForHomepagePhone'), false)
  assert.equal(source.includes('sendSmsCode'), false)
  assert.equal(source.includes('smsCode'), false)
  assert.equal(source.includes('短信验证码'), false)
  assert.equal(source.includes('验证码'), false)
})

test('merchant profile page keeps textarea aligned with inputs', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(source, /\.field,\n\.textarea \{[\s\S]*width: 100%;[\s\S]*box-sizing: border-box;[\s\S]*\}/)
})

test('merchant profile page supports independent merchant logo upload', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    '头像 / LOGO',
    '正方形头像或 LOGO',
    'logoUrl',
    'open-type="chooseAvatar"',
    'onChooseMerchantLogoAvatar',
    'previewMerchantLogo',
    'uni.previewImage',
    'merchant-logo',
    'logo-layout',
    'logo-preview-wrap',
    'logo-change-button',
    'logo-upload-tile',
    'logo-preview-tile',
    'logo-preview',
    'logo-plus',
    'logo-plus-icon',
    '更换图片',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('暂无 LOGO'), false)
  assert.equal(source.includes('class="secondary-button logo-action-button"'), false)
  assert.equal(source.includes('移除 LOGO'), false)
  assert.equal(source.includes('removeMerchantLogo'), false)
  assert.equal(source.includes('logo-action-button'), false)
  assert.equal(source.includes('logo-actions'), false)
  assert.match(source, /<view v-if="logoPreviewUrl" class="logo-preview-wrap">[\s\S]*class="logo-preview-tile"[\s\S]*@click="previewMerchantLogo"[\s\S]*class="logo-change-button"[\s\S]*open-type="chooseAvatar"[\s\S]*@chooseavatar="onChooseMerchantLogoAvatar"[\s\S]*更换图片[\s\S]*<\/view>/)
  assert.match(source, /\.logo-change-button \{[\s\S]*position: absolute;[\s\S]*bottom: 0;[\s\S]*background: rgba\(0, 0, 0, 0\.5\);[\s\S]*color: #fff;[\s\S]*\}/)
})

test('merchant profile page stages images and uploads them on save', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const uploadSource = fs.readFileSync(path.join(root, 'common/upload.js'), 'utf8')

  for (const token of [
    'UniGrid',
    'UniGridItem',
    'chooseMerchantImageFiles',
    'uploadSelectedImage',
    'pendingLogoFile',
    'merchantImageEntries',
    'logoPreviewUrl',
    'merchantImageGridItems',
    'appendMerchantImageFiles',
    'compressMerchantImageFile',
    'resolveImageCompressionOptions',
    'uploadPendingMerchantImages',
    'await uploadPendingMerchantImages()',
    '上传并保存',
  ]) {
    assert.match(profileSource, new RegExp(token))
  }

  for (const token of ['chooseImageFile', 'uploadSelectedImage', 'chooseAndUploadImage']) {
    assert.match(uploadSource, new RegExp(token))
  }

  assert.equal(profileSource.includes('chooseAndUploadImage'), false)
  assert.equal(profileSource.includes('pendingImageFiles'), false)
  assert.equal(profileSource.includes('图片已上传'), false)
  assert.equal(profileSource.includes('LOGO 已上传'), false)
  assert.equal(profileSource.includes('LOGO 上传中'), false)
  assert.equal(profileSource.includes('图片将在保存时上传'), false)
})

test('merchant profile page does not toast after selecting display images', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const homepageMatched = profileSource.match(/async function uploadMerchantImage\(\) \{([\s\S]*?)\n\}/)
  const logoMatched = profileSource.match(/function onChooseMerchantLogoAvatar\(e\) \{([\s\S]*?)\n\}/)

  assert.ok(homepageMatched, 'uploadMerchantImage should exist')
  assert.ok(logoMatched, 'onChooseMerchantLogoAvatar should exist')
  assert.equal(homepageMatched[1].includes('已选择，保存时上传'), false)
  assert.equal(logoMatched[1].includes('已选择，保存时上传'), false)
})

test('merchant profile homepage image delete button is placed at top right', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(profileSource, /\.img-del \{[\s\S]*right: 8rpx;[\s\S]*top: 8rpx;/)
  assert.match(profileSource, /\.img-del\[disabled\] \{[\s\S]*background: rgba\(15, 23, 42, 0\.72\);[\s\S]*opacity: 1;/)
  assert.doesNotMatch(profileSource, /\.img-del \{[\s\S]*bottom: 8rpx;/)
})

test('merchant profile page uses wechat avatar picker for merchant logo', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const uploadSource = fs.readFileSync(path.join(root, 'common/upload.js'), 'utf8')

  for (const token of [
    '正方形头像或 LOGO',
    'open-type="chooseAvatar"',
    '@chooseavatar="onChooseMerchantLogoAvatar"',
    'onChooseMerchantLogoAvatar',
    'e.detail.avatarUrl',
    'createImageFileFromPath',
  ]) {
    assert.match(profileSource, new RegExp(token))
  }

  for (const token of [
    '调整 LOGO',
    'logoCropOpen',
    'confirmLogoCrop',
    'logo-crop-modal',
    'merchantLogoCropCanvas',
  ]) {
    assert.equal(profileSource.includes(token), false)
  }

  for (const token of [
    'createImageFileFromPath',
    'chooseAndCropSquareImageFile',
    'cropImageToSquare',
    'centerCropImageToSquare',
    'uni.cropImage',
    'wx.cropImage',
    'uni.getImageInfo',
    'uni.createCanvasContext',
    'uni.canvasToTempFilePath',
    'setTimeout',
    'finishCanvasCrop',
    "cropScale: '1:1'",
    'tempFilePath',
  ]) {
    assert.match(uploadSource, new RegExp(token))
  }

  assert.match(profileSource, /pendingLogoFile\.value = createImageFileFromPath\(avatarUrl\)/)
  assert.match(profileSource, /grid-template-columns: minmax\(0, 1fr\) 112rpx/)
  assert.match(profileSource, /width: 112rpx/)
  assert.match(profileSource, /height: 112rpx/)
  assert.match(profileSource, /width: 22rpx/)
  assert.match(profileSource, /height: 2rpx/)
  assert.equal(profileSource.includes('<text class="logo-plus">+</text>'), false)
  assert.equal(profileSource.includes('left: -9999px'), false)
  assert.equal(profileSource.includes('top: -9999px'), false)
})

test('merchant profile page keeps display images inline and collapses contact fields', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    '基础资料',
    '联系方式',
    '买家联系和导航',
    '展示图片',
    '公开展示图片，点击图片预览，点击最后一格添加',
    'fixed-save-bar',
    'section-toggle',
    'section-body',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('主页展示'), false)
  assert.equal(source.includes('补充公开展示图片'), false)
  assert.equal(source.includes('brandSectionOpen'), false)
  assert.equal(source.includes('toggleBrandSection'), false)
  assert.match(source, /<text class="field-label">简介<\/text>[\s\S]*?<text class="field-label">展示图片<\/text>[\s\S]*?<view class="form-section contact-section">/)
  assert.equal(source.includes('optional-badge'), false)
  assert.equal(source.includes('选填'), false)
  assert.equal(source.includes('page-head'), false)
  assert.equal(source.includes('page-title'), false)
  assert.equal(source.includes('page-tip'), false)
  assert.equal(source.includes('主页配置'), false)
  assert.equal(source.includes('先完善基础资料，LOGO 和图片可稍后补充'), false)
  assert.equal(source.includes('可稍后完善'), false)
  assert.equal(source.includes('展示买家联系入口和地图导航信息'), false)
  assert.equal(source.includes('展示主页头像、商家卡片和主页图片'), false)
})

test('merchant profile page keeps contact fields optional and collapsed', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    'contactSectionOpen',
    'toggleContactSection',
    '买家联系和导航',
    'contact-section',
    'section-toggle',
    '联系人',
    '联系电话',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('请填写联系人和电话'), false)
  assert.equal(source.includes('<text class="required-badge">必填</text>\\n          </view>\\n        </view>\\n        <view class="section-body">\\n          <view class="form-field">\\n            <text class="field-label">联系人</text>'), false)
})

test('merchant profile page echoes existing contact values in inputs and sanitizes contact input', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    'form.contactPhone = sanitizeContactPhoneValue(contact.phone || \'\')',
    'form.contactWechat = sanitizeContactWechatValue(contact.wechat || \'\')',
    'contactSectionOpen.value = hasExistingContactInfo()',
    'function hasExistingContactInfo()',
    '@input="sanitizeContactPhone"',
    '@input="sanitizeContactWechat"',
    'sanitizeContactWechatValue',
    "replace(/[^a-zA-Z0-9_-]/g, '')",
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.equal(source.includes('contactPhoneHint'), false)
  assert.equal(source.includes('contactWechatHint'), false)
  assert.equal(source.includes('当前：'), false)
  assert.equal(source.includes('填新号可更换'), false)
  assert.equal(source.includes('填新微信可更换'), false)
})

test('merchant profile page reserves space above fixed save bar', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    'fixed-save-spacer',
    'height: calc(156rpx + env(safe-area-inset-bottom))',
    'padding: 24rpx 24rpx 0',
    'fixed-save-bar',
  ]) {
    assert.equal(source.includes(token), true)
  }
})

test('merchant profile page allows merchant type changes without certification warning', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(source, /merchantType: form\.merchantType/)
  assert.equal(source.includes('merchantVerificationStatus'), false)
  assert.equal(source.includes('merchantTypeChangeNeedsReverify'), false)
  assert.equal(source.includes('修改后可能需要重新认证'), false)
  assert.equal(source.includes('保存后需重新提交认证'), false)
  assert.equal(source.includes('getLatestVerification'), false)

  assert.equal(source.includes('merchant-type-readonly'), false)
  assert.equal(source.includes('showLockedMerchantTypeTip'), false)
  assert.equal(source.includes('商家类型创建后暂不支持直接修改'), false)
})

test('merchant profile page supports map based address selection', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    'chooseMerchantLocation',
    'clearMerchantLocation',
    'uni.chooseLocation',
    'locationSelected',
    'form.location',
    '地图选择',
    '清除位置',
    '已选地图位置',
    '可选地图定位',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes(':disabled="Boolean(merchantId)" placeholder="请输入地址"'), false)
})

test('merchant profile map selection writes a detailed address and reverse-geocodes empty results', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(source, /import \{ reverseGeocodeLocation \} from '\.\.\/\.\.\/api\/location'/)
  assert.match(source, /success: async \(result\) => \{[\s\S]*await resolveMerchantLocationAddress\(result, latitude, longitude\)[\s\S]*form\.addressText = resolvedAddress\.address/)
  assert.match(source, /async function resolveMerchantLocationAddress\(result, latitude, longitude\) \{[\s\S]*buildMerchantLocationAddressText\(result\)[\s\S]*reverseGeocodeMerchantLocation\(latitude, longitude, selectedName\)/)
  assert.match(source, /function buildMerchantLocationAddressText\(result\) \{[\s\S]*if \(address && name && !address\.includes\(name\) && !name\.includes\(address\)\) \{[\s\S]*return `\$\{address\}\$\{name\}`/)
})

test('merchant profile clears stale map coordinates when the address is edited manually', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  assert.match(source, /v-model="form\.addressText"[^>]*@input="handleAddressTextInput"/)
  assert.match(source, /const addressLocationNeedsReselection = ref\(false\)/)
  assert.match(source, /addressLocationNeedsReselection \? '地址已修改，请重新地图选择' : '可选地图定位'/)
  assert.match(source, /function handleAddressTextInput\(event\) \{[\s\S]*form\.addressText = String\(event\?\.detail\?\.value \?\? ''\)[\s\S]*if \(!hasValidLocation\(form\.location\)\) return[\s\S]*form\.location = \{\}[\s\S]*addressLocationNeedsReselection\.value = true/)
  assert.match(source, /form\.addressText = resolvedAddress\.address[\s\S]*addressLocationNeedsReselection\.value = false/)
})

test('merchant profile page hides sourcing map binding entry while keeping the page registered', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

  assert.ok(pagesConfig.pages.some((entry) => entry.path === 'pages/merchant/map-binding'))
  for (const token of [
    '拿货地图档口',
    'map-binding-section',
    'mapBindingStatus',
    'getMerchantMapBinding',
    'openMapBindingPage',
    '/pages/merchant/map-binding',
  ]) {
    assert.equal(source.includes(token), false)
  }
})

test('merchant detail page opens an independent location page when coordinates exist', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    'buildMerchantAddressLocation',
    'openMerchantLocation',
    '/pages/merchant/location',
    '查看位置',
    '地址',
  ]) {
    assert.match(source, new RegExp(token))
  }
  assert.doesNotMatch(source, /<map|uni\.openLocation/)
})

test('merchant detail page previews merchant images from tapped image', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    '@click="previewMerchantImage(url)"',
    'previewMerchantImage',
    'uni.previewImage',
    'urls: merchantImages.value',
    'current: url',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('merchant detail page does not show misleading direct contact buttons', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    'class="contact-bar"',
    'contact-spacer',
    'copyWechat',
    'callPhone',
    '微信联系',
    '电话联系',
    '请到供需信息详情页查看电话',
    '请到供需信息详情页查看微信',
  ]) {
    assert.equal(source.includes(token), false)
  }
})

test('merchant detail page paginates published resources with reusable resource list', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const detailSource = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')
  const listSource = fs.existsSync(path.join(root, 'components/ResourceList.vue'))
    ? fs.readFileSync(path.join(root, 'components/ResourceList.vue'), 'utf8')
    : ''

  for (const token of [
    'ResourceList',
    'loadMerchantResources',
    'merchantResourcePage',
    'merchantResourcePageSize',
    'merchantResourceTotal',
    'merchantResourcesLoading',
    'hasMoreMerchantResources',
    'onReachBottom',
    'resetMerchantResources',
    '@load-more="loadMerchantResources"',
    ':loading="merchantResourcesLoading"',
    ':has-more="hasMoreMerchantResources"',
    'merchantResourceCountText',
  ]) {
    assert.match(detailSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.equal(detailSource.includes('<ResourceCard v-for="item in merchantResources"'), false)

  for (const token of [
    'ResourceCard',
    'defineProps',
    'defineEmits',
    'resources',
    'emptyText',
    'loading',
    'hasMore',
    'loadMoreText',
    'resource-list',
    'load-more',
    "emit('load-more')",
  ]) {
    assert.match(listSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('resource detail merchant row uses profile logo without extra title', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')

  for (const token of [
    "import { getMerchant } from '../../api/merchant'",
    'merchantProfile',
    'loadMerchantProfile',
    'merchantProfile.value.logoUrl',
    'merchantAvatarUrl',
    'merchantBusinessText',
    'mainCategories',
    'merchantTypeText',
    'merchant-avatar',
    'merchant-arrow',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.equal(source.includes('merchant-card-title'), false)
  assert.equal(source.includes('商家信息</text>'), false)
  assert.equal(source.includes('查看商家认证、发布记录和信用信息'), false)
})

test('resource detail related resources reuse the market feed card', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')
  const listSource = fs.readFileSync(path.join(root, 'components/ResourceList.vue'), 'utf8')

  for (const token of [
    "import ResourceList from '../../components/ResourceList.vue'",
    ':resources="relatedResources"',
    'variant="feed"',
    ':empty-text="`暂无同类${resourceNoun}`"',
    '@open="openRelatedResource"',
  ]) {
    assert.match(source, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.equal(source.includes('<ResourceCard v-for="item in relatedResources"'), false)
  assert.match(listSource, /<ResourceFeedCard\s+v-if="variant === 'feed'"/)
})

test('resource detail contact bar secondary buttons look independent', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')

  assert.match(source, /\.contact-bar button \{[\s\S]*border: 1rpx solid \$wplink-line;[\s\S]*background: #f8fafc;[\s\S]*box-shadow: 0 8rpx 20rpx rgba\(15, 23, 42, 0\.06\);/)
  assert.match(source, /\.contact-bar button::after \{[\s\S]*border: 0;[\s\S]*\}/)
  assert.match(source, /\.contact-bar \.primary-button \{[\s\S]*border-color: \$wplink-primary;[\s\S]*background: \$wplink-primary;[\s\S]*box-shadow: 0 10rpx 24rpx rgba\(6, 22, 37, 0\.14\);/)
})

test('login page provides reusable wechat login and redirect guard', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const loginSource = fs.existsSync(path.join(root, 'pages/login/index.vue'))
    ? fs.readFileSync(path.join(root, 'pages/login/index.vue'), 'utf8')
    : ''
  const authSource = fs.existsSync(path.join(root, 'common/auth.js'))
    ? fs.readFileSync(path.join(root, 'common/auth.js'), 'utf8')
    : ''
  const mySource = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const pagePaths = (pagesConfig.pages || []).map((item) => item.path)

  assert.equal(pagePaths.includes('pages/login/index'), true)

  for (const token of ['wechatLogin', 'saveToken', 'saveUserId', 'redirectUrl', 'goAfterLogin', 'DEFAULT_CITY_CODE']) {
    assert.match(loginSource, new RegExp(token))
  }

  for (const token of ['requireLogin', 'buildLoginUrl', 'getCurrentPageUrl', 'encodeURIComponent', 'getSession']) {
    assert.match(authSource, new RegExp(token))
  }

  assert.match(mySource, /buildLoginUrl/)
  assert.equal(mySource.includes("import { bindPhone, sendSmsCode, wechatLogin }"), false)
})

test('merchant profile supports filling contact phone from wechat authorization', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const profileSource = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')
  const authSource = fs.readFileSync(path.join(root, 'api/auth.js'), 'utf8')

  assert.match(authSource, /bindWechatPhone/)
  assert.match(authSource, /\/api\/v1\/me\/wechat-phone/)

  for (const token of [
    "import { bindWechatPhone } from '../../api/auth'",
    'phoneAuthorizing',
    'open-type="getPhoneNumber"',
    '@getphonenumber="useWechatPhoneNumber"',
    '微信手机号',
    'bindWechatPhone({ code })',
    '已填入微信手机号',
  ]) {
    assert.match(profileSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})

test('wxapp uses industrial b2b theme colors without legacy green primary', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const sources = collectSourceFiles(root, ['.vue', '.scss', '.js', '.json'])
    .filter((file) => !file.includes(`${path.sep}dist${path.sep}`))
    .filter((file) => !file.includes(`${path.sep}node_modules${path.sep}`))
    .map((file) => fs.readFileSync(file, 'utf8'))
    .join('\n')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

  for (const token of ['#061625', '#c23a00', '#fff0e8', '#f4f7fd', '#d8e0ec', '#16a36a']) {
    assert.match(sources, new RegExp(token, 'i'))
  }

  for (const legacyColor of ['#0f766e', '#e6f4f1']) {
    assert.equal(sources.toLowerCase().includes(legacyColor), false)
  }

  assert.equal(pagesConfig.tabBar.selectedColor, '#c23a00')
  assert.equal(pagesConfig.globalStyle.backgroundColor, '#f4f7fd')
})

test('merchant detail page uses trust-first homepage layout', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    'merchant-hero-card',
    'merchant-identity-row',
    'merchant-summary',
    'hero-stats',
    'profile-panel',
    'profile-chip-row',
    'profile-description',
    'media-section',
    'merchant-gallery',
    'trust-note-section',
    'merchantCategoryTags',
    'merchantTypeLabel',
    'merchantTypeText',
    'profileDescription',
    'statCards',
    'heatScore',
    "' · '",
    '商家档案',
    'profile-chip category',
    'profile-chip.category',
    '热度',
    '主营待补充',
    '联系方式仅随有效供需信息展示',
  ]) {
    assert.match(source, new RegExp(token))
  }

  for (const removedToken of [
    'class="merchant-stats"',
    '商家画像',
    'v-for="tag in creditTags"',
    'profile-chip verified',
    '信用标签',
    '成交反馈',
    'class="contact-bar"',
    '微信联系',
    '电话联系',
    '发布概况</text>',
    'benefit-section',
  ]) {
    assert.equal(source.includes(removedToken), false)
  }
})

test('reports missing API call in required page flow', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-wxapp-flow-'))
  fs.mkdirSync(path.join(root, 'pages/home'), { recursive: true })
  fs.writeFileSync(path.join(root, 'pages/home/index.vue'), '<script setup>function noop(){}</script>')

  const issues = validateFlows(root, [
    {
      file: 'pages/home/index.vue',
      checks: ['listHomeOperationConfig'],
      description: '首页加载运营配置',
    },
  ])

  assert.deepEqual(issues, ['pages/home/index.vue 缺少 首页加载运营配置: listHomeOperationConfig'])
})

function collectSourceFiles(dir, extensions) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...collectSourceFiles(fullPath, extensions))
    } else if (entry.isFile() && extensions.includes(path.extname(entry.name))) {
      files.push(fullPath)
    }
  }
  return files
}
