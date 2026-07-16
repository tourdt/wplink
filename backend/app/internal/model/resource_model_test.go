package model

import (
	"strings"
	"testing"
)

func TestListResourcesSQLAllowsEmptyMerchantID(t *testing.T) {
	requiredSnippets := []string{
		"NULLIF($3, '')::bigint",
		"r.merchant_id = NULLIF($3, '')::bigint",
		"r.direction",
		"rtc.type_name",
		"rtc.display_template #>> '{group,code}' = $4",
		"($6 = '' OR r.direction = $6)",
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

func TestListResourcesSQLPrioritizesActiveTopResources(t *testing.T) {
	requiredSnippets := []string{
		"r.top_expires_at",
		"r.top_expires_at > now()",
		"CASE WHEN r.top_expires_at IS NOT NULL AND r.top_expires_at > now() THEN 1 ELSE 0 END DESC",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listResourcesSQL, snippet) {
			t.Fatalf("listResourcesSQL missing top priority snippet %q:\n%s", snippet, listResourcesSQL)
		}
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
		"rtc.commercial_rules",
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
		"$2 = 'needs_action' AND r.status IN ('draft', 'pending', 'manual_review', 'audit_retry', 'rejected')",
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

func TestProfileMonthlyBenefitsForStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		publishQuota int64
		refreshQuota int64
	}{
		{name: "incomplete merchant no longer gets profile monthly quota", status: MerchantProfileStatusIncomplete, publishQuota: 0, refreshQuota: 0},
		{name: "completed merchant no longer gets more quota from merchant card", status: MerchantProfileStatusCompleted, publishQuota: 0, refreshQuota: 0},
		{name: "unknown status no longer falls back to profile quota", status: " ", publishQuota: 0, refreshQuota: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publishQuota, refreshQuota := profileMonthlyBenefitsForStatus(tt.status)

			if publishQuota != tt.publishQuota || refreshQuota != tt.refreshQuota {
				t.Fatalf("profileMonthlyBenefitsForStatus(%q) = (%d, %d), want (%d, %d)", tt.status, publishQuota, refreshQuota, tt.publishQuota, tt.refreshQuota)
			}
		})
	}
}

func TestQuotaConsumeSQLGuardsOuterBalance(t *testing.T) {
	for name, query := range map[string]string{
		"publish": consumePublishQuotaSQL,
		"refresh": consumeRefreshQuotaSQL,
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(query, "AND remaining_amount > 0\nRETURNING") {
				t.Fatalf("%s quota consume sql should guard remaining_amount on the updated row: %s", name, query)
			}
			if !strings.Contains(query, "AND starts_at <= now()") {
				t.Fatalf("%s quota consume sql should ignore future entitlements: %s", name, query)
			}
		})
	}
}

func TestSubmitResourceForReviewLocksTypeCommercialRules(t *testing.T) {
	requiredSnippets := []string{
		"JOIN resource_type_configs rtc ON rtc.id = r.resource_type_config_id",
		"rtc.commercial_rules",
		"FOR UPDATE",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(submitResourceForReviewLockSQL, snippet) {
			t.Fatalf("submitResourceForReviewLockSQL missing %q:\n%s", snippet, submitResourceForReviewLockSQL)
		}
	}
}
