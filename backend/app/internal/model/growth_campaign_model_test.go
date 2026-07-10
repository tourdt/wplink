package model

import (
	"os"
	"strings"
	"testing"
)

func TestGrowthCampaignModelUsesCampaignSourceAndIdempotency(t *testing.T) {
	source, err := os.ReadFile("growth_campaign_model.go")
	if err != nil {
		t.Fatalf("ReadFile(growth_campaign_model.go) error = %v", err)
	}
	text := string(source)

	for _, snippet := range []string{
		`EntitlementSourceGrowthCampaign = "growth_campaign"`,
		"growth_campaigns",
		"growth_campaign_rules",
		"growth_reward_grants",
		"rule_name",
		"idempotency_key",
		"ON CONFLICT (idempotency_key) DO NOTHING",
		"INSERT INTO merchant_entitlements",
		"source_type",
		"expires_at",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("growth campaign model missing snippet %q", snippet)
		}
	}
}

func TestGrowthCampaignModelClaimsIdempotencyBeforeGrantingEntitlement(t *testing.T) {
	source, err := os.ReadFile("growth_campaign_model.go")
	if err != nil {
		t.Fatalf("ReadFile(growth_campaign_model.go) error = %v", err)
	}
	text := string(source)

	claimIndex := strings.Index(text, "INSERT INTO growth_reward_grants")
	entitlementIndex := strings.Index(text, "INSERT INTO merchant_entitlements")
	if claimIndex < 0 || entitlementIndex < 0 {
		t.Fatalf("growth campaign model should insert both grant record and merchant entitlement")
	}
	if claimIndex > entitlementIndex {
		t.Fatalf("growth campaign model must claim idempotency before inserting entitlement:\n%s", text)
	}
	for _, snippet := range []string{
		"UPDATE growth_reward_grants",
		"SET entitlement_id = $2",
		"WHERE id = $1",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("growth campaign model missing grant entitlement backfill snippet %q", snippet)
		}
	}
}

func TestGrowthCampaignModelIgnoresInactiveCampaignStatuses(t *testing.T) {
	source, err := os.ReadFile("growth_campaign_model.go")
	if err != nil {
		t.Fatalf("ReadFile(growth_campaign_model.go) error = %v", err)
	}
	text := string(source)

	for _, snippet := range []string{
		"gc.status = 'active'",
		"gcr.status = 'active'",
		"gc.starts_at <= now()",
		"(gc.ends_at IS NULL OR gc.ends_at > now())",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("growth campaign active query missing snippet %q", snippet)
		}
	}
}

func TestGrowthTaskProgressSQLUsesResourceAndContactSignals(t *testing.T) {
	checks := []string{
		"COUNT(*) FILTER (WHERE status = 'published')",
		"COUNT(*) FILTER (WHERE status = 'pending')",
		"created_at >= date_trunc('day', now())",
		"action IN ('phone', 'wechat')",
		"action = 'share_view'",
	}
	for _, snippet := range checks {
		if !strings.Contains(growthTaskProgressSQL, snippet) {
			t.Fatalf("growthTaskProgressSQL missing %q:\n%s", snippet, growthTaskProgressSQL)
		}
	}
}

func TestMerchantGrowthRewardGrantsSQLFiltersGrantedByMerchant(t *testing.T) {
	for _, snippet := range []string{"FROM growth_reward_grants", "merchant_id = $1", "status = 'granted'"} {
		if !strings.Contains(listMerchantGrowthRewardGrantsSQL, snippet) {
			t.Fatalf("listMerchantGrowthRewardGrantsSQL missing %q:\n%s", snippet, listMerchantGrowthRewardGrantsSQL)
		}
	}
}
