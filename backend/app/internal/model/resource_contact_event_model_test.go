package model

import (
	"strings"
	"testing"
)

func TestResourceContactUnlockInfoSQLReturnsCommercialRules(t *testing.T) {
	requiredSnippets := []string{
		"rtc.commercial_rules",
		"JOIN resource_type_configs rtc ON rtc.id = r.resource_type_config_id",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(resourceContactUnlockInfoSQL, snippet) {
			t.Fatalf("resourceContactUnlockInfoSQL missing %q:\n%s", snippet, resourceContactUnlockInfoSQL)
		}
	}
}
