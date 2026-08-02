package model

import "fmt"

const (
	PaymentOrderStatusPending = "pending"
	PaymentOrderStatusPaid    = "paid"
	PaymentOrderStatusClosed  = "closed"
)

func boolFromJSON(value interface{}) bool {
	result, _ := value.(bool)
	return result
}

func stringFromJSON(value interface{}) string {
	result, _ := value.(string)
	return result
}

func int64FromJSON(value interface{}) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		var result int64
		_, _ = fmt.Sscanf(typed, "%d", &result)
		return result
	default:
		return 0
	}
}
