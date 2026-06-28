import fs from 'node:fs'
import path from 'node:path'

export const defaultFlowChecks = [
  {
    file: 'pages/home/index.vue',
    description: '首页 Banner 和入口跳转',
    checks: ['listHomeBanners', "item.jumpType === 'topic'", "item.jumpType === 'resource'", "item.jumpType === 'merchant'", "item.jumpType === 'webview'", 'openDemand', 'openPublish'],
  },
  {
    file: 'pages/search/index.vue',
    description: '搜索和无结果提交需求',
    checks: ['listCityResourceTypes', 'searchResources', 'ResourceCard', 'openDemand', '暂未找到合适资源'],
  },
  {
    file: 'pages/resource/detail.vue',
    description: '详情浏览和联系行为',
    checks: [
      'getResource',
      'recordResourceDetailView',
      "recordContact('phone')",
      "recordContact('wechat')",
      "recordContact('merchant_home')",
      "recordContact('share')",
      'open-type="share"',
      'onShareAppMessage',
      'uni.makePhoneCall',
      'uni.setClipboardData',
    ],
  },
  {
    file: 'pages/publish/index.vue',
    description: '资源发布和提交审核',
    checks: [
      'listCityResourceTypes',
      'createResource',
      'submitResource',
      'createResourceDraft',
      'validatePublishForm',
      '请填写标题',
      '请填写品类',
      '请填写联系人',
      '请填写联系电话',
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
    file: 'pages/my-resources/index.vue',
    description: '我的发布管理动作和指标',
    checks: ['listMyResources', 'MetricStrip', 'refreshResource', 'listTopVouchers', 'redeemTopVoucher', 'markResourceDeal', 'takeDownResource', 'repostSimilarResource', 'wechatCopyCount'],
  },
  {
    file: 'pages/messages/index.vue',
    description: '消息筛选和已读',
    checks: ['listMessages', 'readMessage', "selectStatus('unread')", "selectStatus('read')", 'markRead'],
  },
  {
    file: 'pages/topic/index.vue',
    description: '专题资源和需求兜底',
    checks: ['getTopicResources', 'ResourceCard', 'demandEntry', 'openDemand'],
  },
  {
    file: 'pages/webview/index.vue',
    description: 'web-view 白名单校验和阻断态',
    checks: ['validateWebview', 'allowedUrl', '链接不可访问'],
  },
  {
    file: 'pages/verification/index.vue',
    description: '商家认证提交',
    checks: ['submitVerification', 'getUserId', 'businessName', 'licenseUrl', 'storefrontUrl'],
  },
  {
    file: 'pages/my/index.vue',
    description: '身份保存和核心入口',
    checks: ['wechatLogin', 'saveUserId', 'saveMerchantId', 'openMyResources', 'openVerification', 'openPublish'],
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
  return issues
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
