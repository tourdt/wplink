package admin

// nonNilAdminStringSlice 保持 admin.api 中非 optional 数组的运行时契约：
// 即使持久化层返回 nil，JSON 也必须编码为 []，避免生成客户端把数组字段读成 null。
func nonNilAdminStringSlice(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}
