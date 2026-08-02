package permission

const (
	RolePlatformOperator = "platform_operator"
	RoleSuperAdmin       = "super_admin"
	RoleMerchantAdmin    = "merchant_admin"
)

const (
	AdminModuleDashboard           = "dashboard"
	AdminModuleResourceReview      = "resource_review"
	AdminModuleResourceReports     = "resource_reports"
	AdminModuleMerchants           = "merchants"
	AdminModuleEntitlements        = "entitlements"
	AdminModuleBannerTopics        = "banner_topics"
	AdminModuleHotSearchKeywords   = "hot_search_keywords"
	AdminModuleVIPConfigs          = "vip_configs"
	AdminModuleGrowthCampaigns     = "growth_campaigns"
	AdminModuleSourcingMap         = "sourcing_map"
	AdminModuleResourceTypeConfigs = "resource_type_configs"
	AdminModuleOperationLogs       = "operation_logs"
	AdminModuleSearchLogs          = "search_logs"
	AdminModuleAdminPermissions    = "admin_permissions"
)

var allAdminModules = []string{
	AdminModuleDashboard,
	AdminModuleResourceReview,
	AdminModuleResourceReports,
	AdminModuleMerchants,
	AdminModuleEntitlements,
	AdminModuleBannerTopics,
	AdminModuleHotSearchKeywords,
	AdminModuleVIPConfigs,
	AdminModuleGrowthCampaigns,
	AdminModuleSourcingMap,
	AdminModuleResourceTypeConfigs,
	AdminModuleOperationLogs,
	AdminModuleSearchLogs,
	AdminModuleAdminPermissions,
}

var defaultPlatformOperatorAdminModules = []string{
	AdminModuleResourceReview,
	AdminModuleResourceReports,
}

func CanAccessAdmin(roles []string) bool {
	for _, role := range roles {
		if role == RolePlatformOperator || role == RoleSuperAdmin {
			return true
		}
	}
	return false
}

func AllAdminModules() []string {
	return append([]string(nil), allAdminModules...)
}

func ConfigurableAdminModules() []string {
	modules := make([]string, 0, len(allAdminModules)-1)
	for _, module := range allAdminModules {
		if module != AdminModuleAdminPermissions {
			modules = append(modules, module)
		}
	}
	return modules
}

func DefaultPlatformOperatorAdminModules() []string {
	return append([]string(nil), defaultPlatformOperatorAdminModules...)
}

func ResolveAdminModules(roles []string, roleModules map[string][]string) []string {
	if CanManageAdminPermissions(roles) {
		return AllAdminModules()
	}
	for _, role := range roles {
		if role != RolePlatformOperator {
			continue
		}
		modules := normalizeKnownAdminModules(roleModules[RolePlatformOperator], false)
		if len(modules) > 0 {
			return modules
		}
		// 平台运营默认只开放审核类模块；超级管理员可在后台调整该角色的可见模块。
		return DefaultPlatformOperatorAdminModules()
	}
	return nil
}

func CanAccessAdminModule(roles []string, modules []string, module string) bool {
	// 未映射的后台路径默认拒绝，避免新增路由忘记登记模块时意外继承后台访问权限。
	if module == "" {
		return false
	}
	if CanManageAdminPermissions(roles) {
		return true
	}
	for _, allowed := range modules {
		if allowed == module {
			return true
		}
	}
	return false
}

func NormalizeConfigurableAdminModules(modules []string) []string {
	return normalizeKnownAdminModules(modules, false)
}

func normalizeKnownAdminModules(modules []string, includeAdminPermissions bool) []string {
	known := map[string]bool{}
	for _, module := range allAdminModules {
		if includeAdminPermissions || module != AdminModuleAdminPermissions {
			known[module] = true
		}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(modules))
	for _, module := range modules {
		if !known[module] || seen[module] {
			continue
		}
		normalized = append(normalized, module)
		seen[module] = true
	}
	return normalized
}

func CanManageAdminPermissions(roles []string) bool {
	for _, role := range roles {
		if role == RoleSuperAdmin {
			return true
		}
	}
	return false
}
