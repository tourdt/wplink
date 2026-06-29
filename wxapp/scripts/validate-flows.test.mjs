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
    'pages/messages/index.vue',
    'pages/demand-success/index.vue',
    'pages/my/index.vue',
    'pages/search/index.vue',
    'pages/my-demands/index.vue',
  ]

  const visibleSource = files.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('撮合'), false)
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

test('home page keeps custom brand first screen structure', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes('search-divider'), false)
  assert.equal(source.includes('voice-icon'), false)
  assert.equal(source.includes('search-action-icon'), false)
  assert.equal(source.includes('brand-roof'), false)
  assert.equal(source.includes('brand-window-row'), false)

  for (const token of [
    'home-fixed-header',
    'custom-title-bar',
    'home-brand',
    'brand-icon',
    'brand-hanger-hook',
    'brand-hanger-line',
    'brand-cargo-box',
    'brand-cargo-tape',
    '衣货通',
    'getMenuButtonBoundingClientRect',
    'homeContentStyle',
    '搜索现货、厂家或求购需求',
    'factory-hero',
    '织里站 · 精选工厂',
    '童装产业带数字化撮合中心',
    'quick-action-grid',
    '我要找货',
    '我要清货',
    '我要找厂',
    '我要接单',
  ]) {
    assert.match(source, new RegExp(token))
  }
})

test('my page separates guest and logged-in account states without merchant binding', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  for (const token of [
    'isLoggedIn',
    '未登录',
    '登录后管理需求、收藏和发布记录',
    '微信登录',
    '我的账号',
    '已登录，可管理需求、收藏和消息',
    'getLatestVerification',
    'verificationStatusText',
    '待完善',
    'openAccountCard',
    'openMerchantHome',
    '商家主页',
    '商家认证',
    'openMerchantVerification',
    '/pages/verification/index\\?merchantId=',
    '请先完善商家资料',
    'openMessages',
    'requireLogin',
  ]) {
    assert.match(source, new RegExp(token))
  }

  for (const hiddenToken of ['保存身份', '商家 ID', '用户 ID：', '主页配置', 'merchant-actions', '我的权益', '权益提醒', '手机号绑定', '登录后可用', '同步收藏关注', '接收审核和联系消息']) {
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
    'width: 104rpx',
    'font-size: 38rpx',
    'quick-entry-grid',
    'common-service-section',
    '我的发布',
    '商家资料',
    'openMyResources',
    '/pages/my-resources/index',
    'entry-arrow',
    '发布和推广',
    '常用服务',
    'verification-status::before',
    'border-radius: 999rpx',
    'min-height: 32rpx',
    'quick-entry.primary .quick-icon',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('grid-template-columns: 104rpx minmax(0, 1fr)'), true)
  assert.equal(source.includes('.quick-entry.primary .quick-icon {\n  background: $wplink-warning-soft'), true)

  for (const verboseCopy of [
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
  ]) {
    assert.equal(source.includes(verboseCopy), false)
  }
})

test('merchant profile page labels every field and removes manual image url entry', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    '商家名称',
    '商家类型',
    '主营品类',
    '主页联系人',
    '主页联系电话',
    '主页微信',
    '商家地址',
    '商家介绍',
    '商家主页图片',
    '主页展示图',
    'form-field',
    'field-label',
    'image-helper',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('也可粘贴图片 URL'), false)
  assert.equal(source.includes('图片 URL'), false)
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
    '商家 LOGO',
    '正方形 LOGO',
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
    '更换 LOGO',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('暂无 LOGO'), false)
  assert.equal(source.includes('class="secondary-button logo-action-button"'), false)
  assert.equal(source.includes('移除 LOGO'), false)
  assert.equal(source.includes('removeMerchantLogo'), false)
  assert.equal(source.includes('logo-action-button'), false)
  assert.equal(source.includes('logo-actions'), false)
  assert.match(source, /<view v-if="logoPreviewUrl" class="logo-preview-wrap">[\s\S]*class="logo-preview-tile"[\s\S]*@click="previewMerchantLogo"[\s\S]*class="logo-change-button"[\s\S]*open-type="chooseAvatar"[\s\S]*@chooseavatar="onChooseMerchantLogoAvatar"[\s\S]*更换 LOGO[\s\S]*<\/view>/)
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
    '正方形 LOGO',
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

test('merchant profile page groups long form and collapses optional brand section', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    '基础资料',
    '联系方式',
    '品牌展示',
    '买家联系和导航',
    '头像、卡片、主页图',
    'brandSectionOpen',
    'toggleBrandSection',
    'fixed-save-bar',
    'section-toggle',
    'section-body',
  ]) {
    assert.match(source, new RegExp(token))
  }

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
    '主页联系人',
    '主页联系电话',
  ]) {
    assert.match(source, new RegExp(token))
  }

  assert.equal(source.includes('请填写联系人和电话'), false)
  assert.equal(source.includes('<text class="required-badge">必填</text>\\n          </view>\\n        </view>\\n        <view class="section-body">\\n          <view class="form-field">\\n            <text class="field-label">主页联系人</text>'), false)
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

test('merchant profile page allows merchant type changes with re-verification warning', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/profile.vue'), 'utf8')

  for (const token of [
    'merchantVerificationStatus',
    'merchantTypeChanged',
    'merchantTypeChangeNeedsReverify',
    '修改后可能需要重新认证',
    '保存后需重新提交认证',
    'merchantType: form.merchantType',
  ]) {
    assert.match(source, new RegExp(token))
  }

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

test('merchant detail page exposes map navigation when location exists', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    'merchantLocation',
    'hasMerchantLocation',
    'openMerchantLocation',
    'uni.openLocation',
    '地图导航',
    '商家地址',
    'addressText',
  ]) {
    assert.match(source, new RegExp(token))
  }
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

test('merchant detail page keeps fixed contact bar above phone safe area', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  assert.match(source, /\.contact-bar \{[\s\S]*bottom: 0;[\s\S]*padding: 18rpx 24rpx calc\(18rpx \+ env\(safe-area-inset-bottom\)\);/)
  assert.match(source, /\.contact-bar \{[\s\S]*border-top: 1rpx solid \$wplink-line;[\s\S]*background: rgba\(255, 255, 255, 0\.96\);/)
  assert.match(source, /\.contact-spacer \{[\s\S]*height: calc\(124rpx \+ env\(safe-area-inset-bottom\)\);/)
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
    '商家信息',
    'profile-chip category',
    'profile-chip.category',
    '商家热度',
    '主营品类待补充',
    '从资源详情进入可查看完整联系方式',
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
      checks: ['listHomeBanners'],
      description: '首页加载 Banner',
    },
  ])

  assert.deepEqual(issues, ['pages/home/index.vue 缺少 首页加载 Banner: listHomeBanners'])
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
