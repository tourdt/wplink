export const defaultPlatformOperatorModules = ['resource_review', 'resource_reports', 'verification_review']

export function userModules(user) {
  if (user?.roles?.includes('super_admin')) {
    return ['*']
  }
  if (Array.isArray(user?.modules) && user.modules.length > 0) {
    return user.modules
  }
  if (user?.roles?.includes('platform_operator')) {
    return defaultPlatformOperatorModules
  }
  return []
}

export function canAccessAdminModule(user, moduleCode) {
  if (!moduleCode) return true
  const modules = userModules(user)
  return modules.includes('*') || modules.includes(moduleCode)
}
