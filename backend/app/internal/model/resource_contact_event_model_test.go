package model

import (
	"strings"
	"testing"
)

func TestResourceContactUnlockInfoSQLReturnsCommercialRules(t *testing.T) {
	requiredSnippets := []string{
		"r.resource_type_snapshot -> 'commercialRules'",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(resourceContactUnlockInfoSQL, snippet) {
			t.Fatalf("resourceContactUnlockInfoSQL missing %q:\n%s", snippet, resourceContactUnlockInfoSQL)
		}
	}
}
