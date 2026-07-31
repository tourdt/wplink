package model

import (
	"strings"
	"testing"
)

func TestListHomeRecentMerchantsSQLScopesEligibleSelfOnboardedMerchants(t *testing.T) {
	for _, token := range []string{
		"m.status = 'active'",
		"m.profile_status = 'completed'",
		"m.deleted_at IS NULL",
		"m.onboarded_at IS NOT NULL",
		"mab.status = 'active'",
		"mab.role = 'owner'",
		"cs.code = $1",
		"ORDER BY m.onboarded_at DESC, m.id DESC",
		"LIMIT $2",
	} {
		if !strings.Contains(listHomeRecentMerchantsSQL, token) {
			t.Fatalf("listHomeRecentMerchantsSQL missing %q:\n%s", token, listHomeRecentMerchantsSQL)
		}
	}
}

func TestUpdateMerchantSQLKeepsFirstOnboardedTime(t *testing.T) {
	if !strings.Contains(updateMerchantSQL, "onboarded_at = COALESCE(onboarded_at, $9)") {
		t.Fatalf("updateMerchantSQL must preserve the first onboarded time:\n%s", updateMerchantSQL)
	}
	if !strings.Contains(updateMerchantSQL, "profile_status = 'completed'") {
		t.Fatalf("updateMerchantSQL must complete the public merchant profile:\n%s", updateMerchantSQL)
	}
}
