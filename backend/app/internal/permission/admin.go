package permission

const (
	RolePlatformOperator = "platform_operator"
	RoleSuperAdmin       = "super_admin"
	RoleMerchantAdmin    = "merchant_admin"
)

func CanAccessAdmin(roles []string) bool {
	for _, role := range roles {
		if role == RolePlatformOperator || role == RoleSuperAdmin {
			return true
		}
	}
	return false
}
