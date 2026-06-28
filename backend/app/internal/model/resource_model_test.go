package model

import (
	"strings"
	"testing"
)

func TestListResourcesSQLAllowsEmptyMerchantID(t *testing.T) {
	requiredSnippets := []string{
		"NULLIF($3, '')::bigint",
		"r.merchant_id = NULLIF($3, '')::bigint",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listResourcesSQL, snippet) {
			t.Fatalf("listResourcesSQL missing %q:\n%s", snippet, listResourcesSQL)
		}
	}
}
