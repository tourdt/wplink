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
	if !strings.Contains(updateMerchantSQL, "onboarded_at = COALESCE(onboarded_at, $8)") {
		t.Fatalf("updateMerchantSQL must preserve the first onboarded time:\n%s", updateMerchantSQL)
	}
	if !strings.Contains(updateMerchantSQL, "profile_status = 'completed'") {
		t.Fatalf("updateMerchantSQL must complete the public merchant profile:\n%s", updateMerchantSQL)
	}
}

func TestUpdateMerchantSQLSetsOrClearsAllLocationColumns(t *testing.T) {
	for _, token := range []string{
		"market_name = CASE WHEN $13 = 'set' THEN NULLIF($14, '') WHEN $13 = 'clear' THEN NULL",
		"building_name = CASE WHEN $13 = 'set' THEN NULLIF($15, '') WHEN $13 = 'clear' THEN NULL",
		"floor_no = CASE WHEN $13 = 'set' THEN NULLIF($16, '') WHEN $13 = 'clear' THEN NULL",
		"booth_no = CASE WHEN $13 = 'set' THEN NULLIF($17, '') WHEN $13 = 'clear' THEN NULL",
		"latitude = CASE WHEN $13 = 'set' THEN $18 WHEN $13 = 'clear' THEN NULL",
		"longitude = CASE WHEN $13 = 'set' THEN $19 WHEN $13 = 'clear' THEN NULL",
		"location_name = CASE WHEN $13 = 'set' THEN NULLIF($20, '') WHEN $13 = 'clear' THEN NULL",
		"location_status = CASE WHEN $13 = 'set' THEN $21 WHEN $13 = 'clear' THEN 'hidden'",
		"location_updated_at = CASE WHEN $13 = 'set' THEN $8 WHEN $13 = 'clear' THEN NULL",
	} {
		if !strings.Contains(updateMerchantSQL, token) {
			t.Fatalf("updateMerchantSQL missing %q:\n%s", token, updateMerchantSQL)
		}
	}
}
