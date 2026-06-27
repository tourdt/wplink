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
