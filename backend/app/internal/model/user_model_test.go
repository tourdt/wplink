package model

import (
	"strings"
	"testing"
)

func TestManagedMerchantQueriesOnlyReturnActiveMerchants(t *testing.T) {
	queries := map[string]string{
		"list managed merchants": listManagedMerchantsSQL,
		"first managed merchant": getFirstManagedMerchantSQL,
	}
	for name, query := range queries {
		if !strings.Contains(query, "AND m.status = 'active'") {
			t.Fatalf("%s query must filter inactive merchants:\n%s", name, query)
		}
	}
}
