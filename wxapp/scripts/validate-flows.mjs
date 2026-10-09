import fs from 'node:fs'
import path from 'node:path'

// 只验收本次实际注册的地图页面；资源模块源码测试继续保留，避免把旧首页规则套到新首发。
export const defaultFlowChecks = [
 { file: 'pages/home/index.vue', description: '旧首页兼容跳转', checks: ['uni.switchTab', '/pages/sourcing-map/index'] },
 { file: 'pages/sourcing-map/index.vue', description: '找档口列表', checks: ['listMerchantDirectory', 'MerchantPlaceCard', 'openOnboarding', 'MERCHANT_CATEGORY_OPTIONS', 'pendingDirectoryRequests', 'requestVersion', '首批档口入驻中'] },
 { file: 'pages/merchant/location.vue', description: '店铺位置与导航', checks: ['getMerchantLocationContext', 'uni.openLocation'] },
 { file: 'pages/merchant/profile.vue', description: '入驻与店铺维护', checks: ['getMerchant', 'updateMerchant', 'chooseMerchantLocation', 'uploadPendingMerchantImages', 'publicPhone', 'publicWechat'] },
 { file: 'pages/merchant/detail.vue', description: '店铺名片', checks: ['getMerchant', 'onShareAppMessage', 'uni.openLocation'], absentChecks: ['listResources', 'getMerchantFollowState', 'ResourceList'] },
 { file: 'pages/my/index.vue', description: '店铺管理', checks: ['getMerchant', 'setMerchantVisibility', 'getMerchantDraft'], absentChecks: ['getMerchantEntitlements', 'getActiveGrowthCampaigns', 'openMyResources'] },
 { file: 'pages/login/index.vue', description: '登录回跳', checks: ['resolveMapRedirect', 'saveToken', 'getWechatLoginCode'] },
 { file: 'pages/account/settings.vue', description: '退出清理本机草稿', checks: ['clearMerchantDraft', 'clearSession', '/pages/sourcing-map/index'] },
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
    for (const snippet of flow.absentChecks || []) {
      if (source.includes(snippet)) {
        issues.push(`${flow.file} 不应包含 ${flow.description}: ${snippet}`)
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
