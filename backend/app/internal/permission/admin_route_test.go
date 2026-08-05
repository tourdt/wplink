package permission

import "testing"

func TestAdminModuleForPathMapsEveryCurrentAdminModule(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "dashboard", path: "/api/v1/admin/dashboard/overview", want: AdminModuleDashboard},
		{name: "resources", path: "/api/v1/admin/resources/pending", want: AdminModuleResourceReview},
		{name: "resource reports", path: "/api/v1/admin/resource-reports/report-1/review", want: AdminModuleResourceReports},
		{name: "merchant entitlements", path: "/api/v1/admin/merchants/merchant-1/entitlements", want: AdminModuleEntitlements},
		{name: "merchants", path: "/api/v1/admin/merchants/merchant-1", want: AdminModuleMerchants},
		{name: "banner", path: "/api/v1/admin/banner-topics/banner-1", want: AdminModuleBannerTopics},
		{name: "hot search", path: "/api/v1/admin/hot-search-keywords/keyword-1", want: AdminModuleHotSearchKeywords},
		{name: "VIP", path: "/api/v1/admin/vip/plans", want: AdminModuleVIPConfigs},
		{name: "growth", path: "/api/v1/admin/growth-campaigns/invite", want: AdminModuleGrowthCampaigns},
		{name: "map", path: "/api/v1/admin/map/scenes", want: AdminModuleSourcingMap},
		{name: "resource type", path: "/api/v1/admin/resource-type-configs/config-1", want: AdminModuleResourceTypeConfigs},
		{name: "operators", path: "/api/v1/admin/operators/operator-1", want: AdminModuleAdminPermissions},
		{name: "module permissions", path: "/api/v1/admin/module-permissions/platform_operator", want: AdminModuleAdminPermissions},
		{name: "operation log", path: "/api/v1/admin/operation-logs", want: AdminModuleOperationLogs},
		{name: "operation task", path: "/api/v1/admin/tasks/resource-lifecycle", want: AdminModuleOperationLogs},
		{name: "search log", path: "/api/v1/admin/search-logs", want: AdminModuleSearchLogs},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AdminModuleForPath(tc.path); got != tc.want {
				t.Fatalf("AdminModuleForPath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestAdminModuleForPathDefaultsUnknownRouteToDenied(t *testing.T) {
	module := AdminModuleForPath("/api/v1/admin/new-module")
	if module != "" {
		t.Fatalf("module = %q, want empty", module)
	}
	if CanAccessAdminModule([]string{RoleSuperAdmin}, AllAdminModules(), module) {
		t.Fatal("super admin should be denied for an unmapped route")
	}
}

func TestAdminModuleForPathDoesNotMatchLookalikePrefixes(t *testing.T) {
	for _, path := range []string{
		"/api/v1/admin/merchants-malicious",
		"/api/v1/admin/operators-extra",
		"/api/v1/admin/search-logs-archive",
	} {
		if got := AdminModuleForPath(path); got != "" {
			t.Fatalf("AdminModuleForPath(%q) = %q, want default deny", path, got)
		}
	}
}

func TestAdminModuleForPathDoesNotTreatEntitlementLookalikeAsEntitlementRoute(t *testing.T) {
	path := "/api/v1/admin/merchants/merchant-1/entitlements-archive"
	if got := AdminModuleForPath(path); got != AdminModuleMerchants {
		t.Fatalf("AdminModuleForPath(%q) = %q, want merchant module", path, got)
	}
}
