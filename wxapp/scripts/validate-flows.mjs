import fs from 'node:fs'
import path from 'node:path'

export const defaultFlowChecks = [
  {
    file: 'pages/home/index.vue',
    description: '首页 Banner 和入口跳转',
    checks: [
      'listHomeBanners',
      'banners.value.length ? banners.value : defaultBanners',
      "item.jumpType === 'topic'",
      "item.jumpType === 'resource'",
      "item.jumpType === 'merchant'",
      "item.jumpType === 'publish'",
      "item.jumpType === 'search'",
      "item.jumpType === 'webview'",
      'openPublish',
      'sceneEntries',
      'PUBLISH_TYPE_KEY',
      '织里站 · 精选工厂',
      '精选资源',
      'ResourceCard',
      'listResources',
      'homeResources',
    ],
  },
  {
    file: 'pages/search/index.vue',
    description: '资源 tab 推荐和类型筛选',
    checks: ['资源推荐', 'listCityResourceTypes', 'listResources', 'loadRecommendedResources', 'ResourceCard', 'openSearchPage', 'selectType', 'visibleResourceTypes', 'showTypeDrawer', 'onPullDownRefresh', 'onReachBottom'],
  },
  {
    file: 'pages/search/result.vue',
    description: '独立搜索和无结果换条件',
    checks: ['listCityResourceTypes', 'searchResources', 'ResourceCard', '暂无匹配资源', 'hotKeywords', '换个条件'],
  },
  {
    file: 'pages/resource/detail.vue',
    description: '详情浏览和联系行为',
    checks: [
      'getResource',
      'getOwnResource',
      'recordResourceDetailView',
      "recordContact('phone')",
      "recordContact('wechat')",
      "recordContact('merchant_home')",
      "recordContact('share')",
      'open-type="share"',
      'onShareAppMessage',
      'uni.makePhoneCall',
      'uni.setClipboardData',
      '友情提示',
      '联系商家前，建议先确认实物、价格、数量和交付方式。',
      '同类推荐',
      'getResourceFavoriteState',
      'setResourceFavorite',
      'toggleFavorite',
      '不能收藏自己发布的资源',
      'loadOwnResourceIfCurrentMerchant',
      'resourceUnavailable',
      '资源暂不可查看',
      '去找其他资源',
    ],
  },
  {
    file: 'pages/publish/index.vue',
    description: 'tab 发布页入口',
    checks: ['ResourcePublishForm', ':initial-options', 'mode="create"', 'PUBLISH_TYPE_KEY', 'applyPendingPublishType'],
  },
  {
    file: 'pages/publish/edit.vue',
    description: '独立资源编辑页入口',
    checks: ['ResourcePublishForm', 'onLoad', 'routeOptions', 'mode="edit"'],
  },
  {
    file: 'components/ResourcePublishForm.vue',
    description: '资源发布和草稿保存',
    checks: [
      'listCityResourceTypes',
      'createResource',
      'createResourceDraft',
      'chooseImageFile',
      'uploadSelectedImage',
      'uploadResourceImage',
      'uploadPendingResourceImages',
      'validatePublishForm',
      '请填写标题',
      '请填写品类',
      '请填写联系人',
      '请填写联系电话',
      'basic-progress',
      'completionPercent',
      'safe-area-inset-bottom',
    ],
  },
  {
    file: 'pages/publish-success/index.vue',
    description: '发布成功跳转消息和我的发布',
    checks: ['openMessages', 'openMyResources', '/pages/messages/index', '/pages/my-resources/index'],
  },
  {
    file: 'pages/demand/index.vue',
    description: '采购需求提交',
    checks: ['createDemand', 'getUserId', 'quantityRequirement', '/pages/demand-success/index'],
  },
  {
    file: 'pages/demand-success/index.vue',
    description: '需求成功跳转消息和首页',
    checks: ['openMessages', 'backHome', '/pages/messages/index', '/pages/home/index'],
  },
  {
    file: 'pages/my-demands/index.vue',
    description: '我的采购需求列表',
    checks: ['listMyDemands', 'options.userId', 'getUserId', 'statusLabel', 'openDemand', 'openMessages'],
  },
  {
    file: 'pages/my-resources/index.vue',
    description: '我的发布管理动作和指标',
    checks: ['listMyResources', 'MetricStrip', 'refreshResource', 'listTopVouchers', 'redeemTopVoucher', 'takeDownResource', 'deleteTakenDownResource', 'canDeleteTakenDown', 'openDraftEditor', 'openRejectedEditor', 'openPublishEditor', '/pages/publish/edit?merchantId=', 'uni.navigateTo', 'rejectReason', '驳回原因', 'getOwnResource', 'buildRepostInitialForm', 'repostInitialForm', 'wechatCopyCount', 'formatDateToDay', 'publish-fab', 'position: fixed', 'canTopResource', '再发类似', 'from=my-resources'],
  },
  {
    file: 'pages/messages/index.vue',
    description: '消息筛选和已读',
    checks: [
      'listMessages',
      'readMessage',
      "selectStatus('unread')",
      "selectStatus('read')",
      'markRead',
      'roleCode.value',
      'openMessageTarget',
      'targetUrl',
      'tabPagePaths',
      'uni.switchTab',
      'uni.navigateTo',
      'messageTabs',
    ],
  },
  {
    file: 'pages/favorites/index.vue',
    description: '收藏关注筛选、空态和分页',
    checks: [
      'listFavoriteResources',
      'listFollowedMerchants',
      'MerchantBadge',
      '<view class="filter-row">',
      'filter-button',
      'position: fixed;',
      'padding-top: 132rpx;',
      'onPullDownRefresh',
      'onReachBottom',
      'uni.stopPullDownRefresh()',
      'loadRows({ reset: true })',
      'loadRows({ reset: false })',
      'page.value',
      'hasMore.value',
      'loading.value',
      'empty-state',
      'emptyTitle',
      'emptyDesc',
      'emptyActionText',
      'load-more-text',
      'merchantAvatarUrl',
      'merchantAvatarText',
      'merchantBusinessText',
      'merchant-avatar',
      'merchant-arrow',
    ],
  },
  {
    file: 'pages/topic/index.vue',
    description: '专题资源和继续浏览兜底',
    checks: ['getTopicResources', 'ResourceCard', 'openSearch', 'Banner 专题', 'topicStats', '继续浏览资源'],
  },
  {
    file: 'pages/webview/index.vue',
    description: 'web-view 白名单校验和阻断态',
    checks: ['validateWebview', 'allowedUrl', '链接不可访问'],
  },
  {
    file: 'pages/merchant/detail.vue',
    description: '商家主页认证和发布记录',
    checks: ['getMerchant', 'listResources', 'ResourceList', 'merchantResources', 'loadMerchantResources', 'hasMoreMerchantResources', 'onReachBottom', 'openResource', 'verificationStatus', 'resourcesSummary', 'heatScore', '商家热度', 'merchantLogo', 'merchantImages', 'merchantLocation', 'openMerchantLocation', 'uni.openLocation', '地图导航', 'merchant-hero-card', 'merchant-gallery', 'merchant-image', 'previewMerchantImage', 'uni.previewImage', 'trust-note-section', '从资源详情进入可查看完整联系方式', 'getMerchantFollowState', 'setMerchantFollow', 'toggleFollow', 'isOwnMerchant', 'openMerchantEditor', '编辑', '/pages/merchant/profile?merchantId='],
  },
  {
    file: 'api/favorite.js',
    description: '收藏关注 API',
    checks: ['getResourceFavoriteState', 'setResourceFavorite', 'listFavoriteResources', 'getMerchantFollowState', 'setMerchantFollow', 'listFollowedMerchants'],
  },
  {
    file: 'common/auth.js',
    description: '通用登录跳转工具',
    checks: ['getSession', 'isLoggedIn', 'getCurrentPageUrl', 'buildLoginUrl', 'requireLogin', 'encodeURIComponent', '/pages/login/index'],
  },
  {
    file: 'components/ResourceCard.vue',
    description: '资源卡四行信息和资源类型角标',
    checks: ['isVerifiedMerchant', 'verified-badge', 'merchant-line', 'resource-title', 'resource-meta', 'resource-price', 'type-corner', 'formatRefreshedAt'],
  },
  {
    file: 'components/ResourceList.vue',
    description: '通用资源列表、空态和加载更多',
    checks: ['ResourceCard', 'resources', 'emptyText', 'loading', 'hasMore', 'loadMoreText', "emit('load-more')"],
  },
  {
    file: 'pages/merchant/profile.vue',
    description: '商家入驻和资料维护',
    checks: [
      'createMerchant',
      'getMerchant',
      'updateMerchant',
      'saveMerchantId',
      'loadMerchant',
      'submitMerchantProfile',
      'sanitizeContactPhone',
      'isValidContactPhone',
      '手机号需为 6-20 位数字',
      'contactPhoneHint',
      'chooseMerchantImageFiles',
      'createImageFileFromPath',
      'uploadSelectedImage',
      'uploadMerchantImage',
      'removeMerchantImage',
      'merchantImageEntries',
      'appendMerchantImageFiles',
      'compressMerchantImageFile',
      'UniGrid',
      'UniGridItem',
      'open-type="chooseAvatar"',
      'onChooseMerchantLogoAvatar',
      'pendingLogoFile',
      'uploadPendingMerchantImages',
      'previewMerchantLogo',
      'uni.previewImage',
      'logo-upload-tile',
      'logo-preview-tile',
      'logo-change-button',
      'merchantVerificationStatus',
      'merchantTypeChanged',
      'merchantTypeChangeNeedsReverify',
      '修改后可能需要重新认证',
      'chooseMerchantLocation',
      'clearMerchantLocation',
      'uni.chooseLocation',
      'locationSelected',
      'form.location',
      'mainCategoriesText',
      'DEFAULT_CITY_CODE',
      '主页联系电话',
      '地图选择',
      '已选地图位置',
      'contactSectionOpen',
      'toggleContactSection',
      'validateMerchantName',
      'merchantNameMessage',
      '请填写主营品类',
    ],
  },
  {
    file: 'pages/verification/index.vue',
    description: '商家认证提交',
    checks: [
      'submitVerification',
      'getLatestVerification',
      'latestVerification',
      'statusLabel',
      'options.merchantId',
      'getUserId',
      'businessName',
      'licenseUrl',
      'storefrontUrl',
      'chooseAndUploadImage',
      'uploadLicense',
      'uploadStorefront',
    ],
  },
  {
    file: 'common/upload.js',
    description: '图片直传对象存储',
    checks: ['createUploadToken', 'uni.chooseImage', 'uni.uploadFile', 'token.uploadToken', 'token.objectKey'],
  },
  {
    file: 'pages/my/index.vue',
    description: '我的页登录态和核心入口',
    checks: [
      'isLoggedIn',
      'buildLoginUrl',
      'requireLogin',
      'openLogin',
      'openMessages',
      'openMerchantHome',
      'openMerchantVerification',
      'openAccountCard',
      'getLatestVerification',
      'getMerchantMetricsSummary',
      'verificationStatusText',
      'merchantEffectVisible',
      '商家本周效果',
      '近 7 天',
      'openFavorites',
      '未登录',
      '登录后管理收藏和发布记录',
      '我的账号',
      '已登录，可管理收藏和消息',
      '待完善',
      '商家主页',
      '查看自己的公开页',
      '商家认证',
      '我的发布',
      '状态、数据、推广',
      '/pages/merchant/detail?id=',
      '/pages/verification/index?merchantId=',
      '/pages/my-resources/index',
    ],
  },
  {
    file: 'pages/login/index.vue',
    description: '独立登录页和回跳',
    checks: [
      'wechatLogin',
      'saveToken',
      'saveUserId',
      'DEFAULT_CITY_CODE',
      'redirectUrl',
      'goAfterLogin',
      'switchTab',
      'redirectTo',
      'localDevLoginCode',
      '登录后同步收藏、需求、消息和发布记录',
    ],
  },
]

export function validateFlows(root, checks = defaultFlowChecks) {
  const issues = []
  for (const flow of checks) {
    const filePath = path.join(root, flow.file)
    if (!fs.existsSync(filePath)) {
      issues.push(`${flow.file} 不存在`)
      continue
    }
    const source = fs.readFileSync(filePath, 'utf8')
    for (const snippet of flow.checks) {
      if (!source.includes(snippet)) {
        issues.push(`${flow.file} 缺少 ${flow.description}: ${snippet}`)
      }
    }
  }
  issues.push(...validateTemplateCompatibility(root))
  return issues
}

function validateTemplateCompatibility(root) {
  const issues = []
  for (const file of listVueFiles(root)) {
    const source = fs.readFileSync(file, 'utf8')
    const template = source.match(/<template>([\s\S]*?)<\/template>/)?.[1] || ''
    if (template.includes('?.')) {
      issues.push(`${path.relative(root, file)} 模板不兼容微信编译: 请避免使用可选链 ?.`)
    }
  }
  return issues
}

function listVueFiles(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...listVueFiles(fullPath))
    } else if (entry.isFile() && entry.name.endsWith('.vue')) {
      files.push(fullPath)
    }
  }
  return files
}

function main() {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const issues = validateFlows(root)
  if (issues.length > 0) {
    console.error('wxapp flow check failed:')
    for (const issue of issues) {
      console.error(`- ${issue}`)
    }
    process.exitCode = 1
    return
  }
  console.log('wxapp flows ok')
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main()
}
