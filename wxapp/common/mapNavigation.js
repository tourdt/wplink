export const MAP_HOME = '/pages/sourcing-map/index'
export const MAP_TAB_PATHS = [MAP_HOME, '/pages/my/index']
const PUBLIC_PAGE_PATHS = [
  '/pages/merchant/detail', '/pages/merchant/location', '/pages/merchant/profile',
  '/pages/legal/index', '/pages/account/settings',
]

// 登录回跳只允许当前版本已注册的业务页，旧资源链接和外部 URL 回到找档口。
export function resolveMapRedirect(value) {
  const target = String(value || '').trim()
  const path = target.split('?')[0]
  if (MAP_TAB_PATHS.includes(path)) return { url: path, tab: true }
  if (PUBLIC_PAGE_PATHS.includes(path)) return { url: target, tab: false }
  return { url: MAP_HOME, tab: true }
}
