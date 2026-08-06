package permission

import "strings"

// AdminModuleForPath 将后台路由绑定到权限模块；未登记路径返回空值并由权限层默认拒绝。
func AdminModuleForPath(path string) string {
	switch {
	case adminPathMatches(path, "/api/v1/admin/dashboard"):
		return AdminModuleDashboard
	case adminPathMatches(path, "/api/v1/admin/resources"):
		return AdminModuleResourceReview
	case adminPathMatches(path, "/api/v1/admin/resource-reports"):
		return AdminModuleResourceReports
	case adminPathMatches(path, "/api/v1/admin/merchants") && adminPathHasSegment(path, "entitlements"):
		return AdminModuleEntitlements
	case adminPathMatches(path, "/api/v1/admin/merchants"):
		return AdminModuleMerchants
	case adminPathMatches(path, "/api/v1/admin/banner-topics"):
		return AdminModuleBannerTopics
	case adminPathMatches(path, "/api/v1/admin/hot-search-keywords"):
		return AdminModuleHotSearchKeywords
	case adminPathMatches(path, "/api/v1/admin/vip"):
		return AdminModuleVIPConfigs
	case adminPathMatches(path, "/api/v1/admin/growth-campaigns"):
		return AdminModuleGrowthCampaigns
	case adminPathMatches(path, "/api/v1/admin/map"):
		return AdminModuleSourcingMap
	case adminPathMatches(path, "/api/v1/admin/resource-type-configs"):
		return AdminModuleResourceTypeConfigs
	case adminPathMatches(path, "/api/v1/admin/operators"), adminPathMatches(path, "/api/v1/admin/module-permissions"):
		return AdminModuleAdminPermissions
	case adminPathMatches(path, "/api/v1/admin/operation-logs"), adminPathMatches(path, "/api/v1/admin/tasks/resource-lifecycle"):
		return AdminModuleOperationLogs
	case adminPathMatches(path, "/api/v1/admin/search-logs"):
		return AdminModuleSearchLogs
	default:
		return ""
	}
}

func adminPathMatches(path string, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func adminPathHasSegment(path string, segment string) bool {
	return strings.Contains("/"+strings.Trim(path, "/")+"/", "/"+segment+"/")
}
