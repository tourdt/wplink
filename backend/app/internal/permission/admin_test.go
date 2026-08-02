package permission

import "testing"

func TestCanAccessAdminReturnsTrueForOperators(t *testing.T) {
	if !CanAccessAdmin([]string{RolePlatformOperator}) {
		t.Fatal("platform operator should access admin")
	}
	if !CanAccessAdmin([]string{RoleSuperAdmin}) {
		t.Fatal("super admin should access admin")
	}
}

func TestCanAccessAdminRejectsMerchantAdmin(t *testing.T) {
	if CanAccessAdmin([]string{RoleMerchantAdmin}) {
		t.Fatal("merchant admin should not access platform admin")
	}
}

func TestCanManageAdminPermissionsRequiresSuperAdmin(t *testing.T) {
	if !CanManageAdminPermissions([]string{RoleSuperAdmin}) {
		t.Fatal("super admin should manage admin permissions")
	}
	if CanManageAdminPermissions([]string{RolePlatformOperator}) {
		t.Fatal("platform operator should not manage admin permissions")
	}
}

func TestResolveAdminModulesDefaultsPlatformOperatorToAuditModules(t *testing.T) {
	modules := ResolveAdminModules([]string{RolePlatformOperator}, nil)
	want := []string{AdminModuleResourceReview, AdminModuleResourceReports}
	if len(modules) != len(want) {
		t.Fatalf("modules = %#v, want %#v", modules, want)
	}
	for i := range want {
		if modules[i] != want[i] {
			t.Fatalf("modules = %#v, want %#v", modules, want)
		}
	}
}

func TestResolveAdminModulesAllowsConfiguredPlatformOperatorModules(t *testing.T) {
	modules := ResolveAdminModules([]string{RolePlatformOperator}, map[string][]string{
		RolePlatformOperator: {AdminModuleDashboard, AdminModuleMerchants, AdminModuleAdminPermissions},
	})
	if len(modules) != 2 || modules[0] != AdminModuleDashboard || modules[1] != AdminModuleMerchants {
		t.Fatalf("modules = %#v, want configurable non-super modules", modules)
	}
}

func TestCanAccessAdminModuleAllowsSuperAdmin(t *testing.T) {
	if !CanAccessAdminModule([]string{RoleSuperAdmin}, nil, AdminModuleAdminPermissions) {
		t.Fatal("super admin should access admin permission module")
	}
	if CanAccessAdminModule([]string{RolePlatformOperator}, []string{AdminModuleResourceReview}, AdminModuleMerchants) {
		t.Fatal("platform operator should not access module outside configured list")
	}
}

func TestCanAccessAdminModuleRejectsUnmappedPathForEveryRole(t *testing.T) {
	if CanAccessAdminModule([]string{RoleSuperAdmin}, AllAdminModules(), "") {
		t.Fatal("super admin should not access an unmapped admin route")
	}
	if CanAccessAdminModule([]string{RolePlatformOperator}, DefaultPlatformOperatorAdminModules(), "") {
		t.Fatal("platform operator should not access an unmapped admin route")
	}
}
