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

func TestListResourcesSQLHidesInactiveMerchants(t *testing.T) {
	requiredSnippet := "m.status = 'active'"
	if !strings.Contains(listResourcesSQL, requiredSnippet) {
		t.Fatalf("listResourcesSQL missing %q:\n%s", requiredSnippet, listResourcesSQL)
	}
}

func TestReviewResourceSQLUsesConfiguredValidDays(t *testing.T) {
	requiredSnippets := []string{
		"rtc.default_valid_days",
		"make_interval(days => GREATEST(rtc.default_valid_days, 1)::int)",
		"rtc.id = resources.resource_type_config_id",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(reviewResourceSQL, snippet) {
			t.Fatalf("reviewResourceSQL missing %q:\n%s", snippet, reviewResourceSQL)
		}
	}
	if strings.Contains(reviewResourceSQL, "interval '7 days'") {
		t.Fatalf("reviewResourceSQL still hard-codes 7 days:\n%s", reviewResourceSQL)
	}
}

func TestPublishedResourceDetailSQLReturnsTypeDisplayConfigAndHidesInactiveMerchants(t *testing.T) {
	requiredSnippets := []string{
		"rtc.type_name",
		"rtc.field_schema",
		"rtc.display_template",
		"m.status = 'active'",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(publishedResourceDetailSQL, snippet) {
			t.Fatalf("publishedResourceDetailSQL missing %q:\n%s", snippet, publishedResourceDetailSQL)
		}
	}
}

func TestListMyResourcesSQLSupportsGroupedStatusFilters(t *testing.T) {
	requiredSnippets := []string{
		"$2 = 'needs_action' AND r.status IN ('draft', 'pending', 'rejected')",
		"$2 = 'showing' AND r.status = 'published' AND r.dealt_at IS NULL",
		"$2 = 'ended' AND (r.status IN ('expired', 'taken_down')",
		"OR r.dealt_at IS NOT NULL",
		"OR (r.expires_at IS NOT NULL AND r.expires_at <= now())",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listMyResourcesSQL, snippet) {
			t.Fatalf("listMyResourcesSQL missing %q:\n%s", snippet, listMyResourcesSQL)
		}
	}
}

func TestListMyResourcesSQLFallsBackToFirstImageWhenCoverURLIsEmpty(t *testing.T) {
	requiredSnippet := "COALESCE(NULLIF(r.cover_url, ''), r.images ->> 0, '')"
	if !strings.Contains(listMyResourcesSQL, requiredSnippet) {
		t.Fatalf("listMyResourcesSQL missing %q:\n%s", requiredSnippet, listMyResourcesSQL)
	}
}

func TestExpiringResourcesSQLUsesTypeMessageRules(t *testing.T) {
	requiredSnippets := []string{
		"JOIN resource_type_configs rtc ON rtc.id = r.resource_type_config_id",
		"rtc.message_rules ->> 'expiringSoonDays'",
		"NULLIF(rtc.message_rules ->> 'expiringSoonDays', '') ~ '^[0-9]+$'",
		"GREATEST((rtc.message_rules ->> 'expiringSoonDays')::int, 1)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listResourcesExpiringSoonSQL, snippet) {
			t.Fatalf("listResourcesExpiringSoonSQL missing %q:\n%s", snippet, listResourcesExpiringSoonSQL)
		}
	}
}
