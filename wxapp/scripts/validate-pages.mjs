import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const pagesPath = path.join(root, 'pages.json')
const manifestPath = path.join(root, 'manifest.json')

if (!fs.existsSync(pagesPath)) {
  throw new Error('缺少 pages.json')
}
if (!fs.existsSync(manifestPath)) {
  throw new Error('缺少 manifest.json')
}

const pagesConfig = JSON.parse(fs.readFileSync(pagesPath, 'utf8'))
const manifestConfig = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
const pagePaths = (pagesConfig.pages || []).map((item) => item.path)
const requiredPages = [
  'pages/home/index',
  'pages/market/index',
  'pages/search/index',
  'pages/sourcing-map/index',
  'pages/publish/index',
  'pages/publish/edit',
  'pages/publish-success/index',
  'pages/messages/index',
  'pages/my/index',
  'pages/login/index',
  'pages/legal/index',
  'pages/account/settings',
  'pages/my-resources/index',
  'pages/favorites/index',
  'pages/merchant/profile',
  'pages/merchant/map-binding',
  'pages/resource/detail',
  'pages/resource/report',
  'pages/merchant/detail',
  'pages/merchant/location',
  'pages/topic/index',
  'pages/webview/index',
]

for (const page of requiredPages) {
  if (!pagePaths.includes(page)) {
    throw new Error(`pages.json 缺少页面: ${page}`)
  }
  const vuePath = path.join(root, `${page}.vue`)
  if (!fs.existsSync(vuePath)) {
    throw new Error(`页面文件不存在: ${page}.vue`)
  }
}

const tabBarPages = new Set((pagesConfig.tabBar?.list || []).map((item) => item.pagePath))
for (const page of ['pages/home/index', 'pages/sourcing-map/index', 'pages/publish/index', 'pages/market/index', 'pages/my/index']) {
  if (!tabBarPages.has(page)) {
    throw new Error(`tabBar 缺少页面: ${page}`)
  }
}

for (const item of pagesConfig.tabBar?.list || []) {
  for (const key of ['iconPath', 'selectedIconPath']) {
    const iconPath = item[key]
    if (!iconPath) {
      throw new Error(`tabBar ${item.text} 缺少 ${key}`)
    }
    if (!iconPath.startsWith('static/tabbar/')) {
      throw new Error(`tabBar ${item.text} ${key} 必须使用 static/tabbar/ 相对路径: ${iconPath}`)
    }
    const iconFile = path.join(root, iconPath)
    if (!fs.existsSync(iconFile)) {
      throw new Error(`tabBar ${item.text} ${key} 文件不存在: ${iconPath}`)
    }
  }
}

for (const page of ['pages/market/index', 'pages/search/index']) {
  const vuePath = path.join(root, `${page}.vue`)
  const source = fs.readFileSync(vuePath, 'utf8')
  if (!source.includes('listCityResourceTypes')) {
    throw new Error(`${page}.vue 未从城市供需类型配置加载供需类型`)
  }
}

const publishFormPath = path.join(root, 'components/ResourcePublishForm.vue')
if (!fs.existsSync(publishFormPath)) {
  throw new Error('缺少供需信息发布表单组件: components/ResourcePublishForm.vue')
}
if (!fs.readFileSync(publishFormPath, 'utf8').includes('listCityResourceTypes')) {
  throw new Error('components/ResourcePublishForm.vue 未从城市供需类型配置加载供需类型')
}

if (manifestConfig.uniStatistics?.enable !== false || manifestConfig['mp-weixin']?.uniStatistics?.enable !== false) {
  throw new Error('manifest.json 必须显式关闭 uniStatistics，避免微信开发者工具启动期注入统计运行时')
}

const weixinConfig = manifestConfig['mp-weixin'] || {}
if (!Array.isArray(weixinConfig.requiredPrivateInfos) || !weixinConfig.requiredPrivateInfos.includes('chooseLocation')) {
  throw new Error('manifest.json mp-weixin.requiredPrivateInfos 必须声明 chooseLocation，供需表单地图选点依赖该微信隐私接口')
}
if (!weixinConfig.requiredPrivateInfos.includes('getLocation')) {
  throw new Error('manifest.json mp-weixin.requiredPrivateInfos 必须声明 getLocation，供需表单需按当前位置判断地图默认中心')
}
const userLocationDesc = weixinConfig.permission?.['scope.userLocation']?.desc || ''
if (!userLocationDesc.includes('地图位置')) {
  throw new Error('manifest.json mp-weixin.permission.scope.userLocation.desc 必须说明地图位置用途')
}

console.log('wxapp pages ok')
