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
		"r.resource_type_snapshot ->> 'typeName'",
		"r.resource_type_snapshot #>> '{displayTemplate,group,code}' = $4",
		"($6 = '' OR r.direction = $6)",
		"r.tags ?& $11::text[]",
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

func TestListRelatedResourcesSQLRanksAndFiltersCandidates(t *testing.T) {
	required := []string{
		"candidate.id <> source.id",
		"candidate.status = 'published'",
		"merchant.status = 'active'",
		"merchant.deleted_at IS NULL",
		"candidate.type_code = source.type_code THEN 1",
		"candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code THEN 2",
		"candidate.direction = source.direction THEN 3",
		"candidate.expires_at IS NULL OR candidate.expires_at > now()",
		"LIMIT $2",
	}
	for _, snippet := range required {
		if !strings.Contains(listRelatedResourcesSQL, snippet) {
			t.Fatalf("listRelatedResourcesSQL missing %q:\n%s", snippet, listRelatedResourcesSQL)
		}
	}
	if strings.Contains(listRelatedResourcesSQL, "source.status = 'published'") {
		t.Fatalf("listRelatedResourcesSQL should support an authorized private source:\n%s", listRelatedResourcesSQL)
	}
}

func TestListRelatedResourcesSQLExcludesUnrelatedCandidatesAndUsesStableOrder(t *testing.T) {
	requiredFilter := []string{
		"candidate.type_code = source.type_code",
		"source.group_code <> ''",
		"candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code",
		"candidate.direction = source.direction",
	}
	for _, snippet := range requiredFilter {
		if !strings.Contains(listRelatedResourcesSQL, snippet) {
			t.Fatalf("listRelatedResourcesSQL missing related-candidate filter %q:\n%s", snippet, listRelatedResourcesSQL)
		}
	}
	if strings.Contains(listRelatedResourcesSQL, "ELSE 4") {
		t.Fatalf("listRelatedResourcesSQL must not return a fourth, unrelated candidate tier:\n%s", listRelatedResourcesSQL)
	}

	order := []string{
		"END ASC,",
		"CASE WHEN candidate.top_expires_at IS NOT NULL AND candidate.top_expires_at > now() THEN 1 ELSE 0 END DESC,",
		"COALESCE(candidate.refreshed_at, candidate.published_at, candidate.created_at) DESC,",
		"candidate.id DESC",
	}
	previousIndex := -1
	for _, snippet := range order {
		index := strings.Index(listRelatedResourcesSQL, snippet)
		if index <= previousIndex {
			t.Fatalf("listRelatedResourcesSQL must keep stable ORDER BY %q after index %d:\n%s", snippet, previousIndex, listRelatedResourcesSQL)
		}
		previousIndex = index
	}
}

func TestListRelatedResourcesSQLKeepsSimilarityPredicateAndCasePriorityAligned(t *testing.T) {
	whereClause := sqlSection(t, listRelatedResourcesSQL, "WHERE candidate.deleted_at IS NULL", "\nORDER BY")
	wherePredicate := normalizeSQLWhitespace(whereClause)
	expectedPredicate := normalizeSQLWhitespace(`
AND (
  candidate.type_code = source.type_code
  OR (
    source.group_code <> ''
    AND candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code
  )
  OR candidate.direction = source.direction
)
`)
	if !strings.Contains(wherePredicate, expectedPredicate) {
		t.Fatalf("candidate similarity predicate must remain in WHERE filters:\n%s", whereClause)
	}

	priorityCase := sqlSection(t, listRelatedResourcesSQL, "ORDER BY\n  CASE", "\n  END ASC,")
	priorityCase = normalizeSQLWhitespace(priorityCase)
	branches := []string{
		"WHEN candidate.type_code = source.type_code THEN 1",
		"WHEN source.group_code <> '' AND candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code THEN 2",
		"WHEN candidate.direction = source.direction THEN 3",
	}
	previousIndex := -1
	for _, branch := range branches {
		index := strings.Index(priorityCase, branch)
		if index <= previousIndex {
			t.Fatalf("priority CASE must keep type, non-empty group, direction order; missing or misplaced %q:\n%s", branch, priorityCase)
		}
		previousIndex = index
	}
}

func sqlSection(t *testing.T, query string, startMarker string, endMarker string) string {
	t.Helper()
	start := strings.Index(query, startMarker)
	if start < 0 {
		t.Fatalf("SQL missing section start %q:\n%s", startMarker, query)
	}
	endOffset := strings.Index(query[start:], endMarker)
	if endOffset < 0 {
		t.Fatalf("SQL missing section end %q after %q:\n%s", endMarker, startMarker, query)
	}
	return query[start : start+endOffset+len(endMarker)]
}

func normalizeSQLWhitespace(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

func TestGetRelatedResourceSourceSQLOnlyReadsAuthorizationAndRankingFields(t *testing.T) {
	required := []string{
		"r.status",
		"r.merchant_id::text",
		"r.type_code",
		"r.direction",
		"r.resource_type_snapshot #>> '{displayTemplate,group,code}'",
	}
	for _, snippet := range required {
		if !strings.Contains(getRelatedResourceSourceSQL, snippet) {
			t.Fatalf("getRelatedResourceSourceSQL missing %q:\n%s", snippet, getRelatedResourceSourceSQL)
		}
	}
	for _, forbidden := range []string{"description", "contact_name", "contact_phone", "contact_wechat"} {
		if strings.Contains(getRelatedResourceSourceSQL, forbidden) {
			t.Fatalf("getRelatedResourceSourceSQL must not read private field %q:\n%s", forbidden, getRelatedResourceSourceSQL)
		}
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

func TestListResourcesSQLUsesJSONBTagFilter(t *testing.T) {
	requiredSnippets := []string{
		"cardinality($11::text[]) = 0",
		"r.tags ?& $11::text[]",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listResourcesSQL, snippet) {
			t.Fatalf("listResourcesSQL missing tag filter snippet %q:\n%s", snippet, listResourcesSQL)
		}
	}
	if strings.Contains(listResourcesSQL, "tags::text ILIKE") {
		t.Fatalf("listResourcesSQL should not scan tags as text:\n%s", listResourcesSQL)
	}
}

func TestReviewResourceSQLUsesSnapshotValidDays(t *testing.T) {
	requiredSnippets := []string{
		"resources.resource_type_snapshot ->> 'defaultValidDays'",
		"$4::timestamptz + make_interval",
		"updated_at = $4::timestamptz",
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

func TestPublishResourceAfterAuditSQLCastsPublishTime(t *testing.T) {
	requiredSnippets := []string{
		"published_at = $2::timestamptz",
		"refreshed_at = $2::timestamptz",
		"resources.resource_type_snapshot ->> 'defaultValidDays'",
		"updated_at = $2::timestamptz",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(publishResourceAfterAuditSQL, snippet) {
			t.Fatalf("publishResourceAfterAuditSQL missing %q:\n%s", snippet, publishResourceAfterAuditSQL)
		}
	}
}

func TestPublishedResourceDetailSQLReturnsTypeSnapshotAndHidesInactiveMerchants(t *testing.T) {
	requiredSnippets := []string{
		"r.resource_type_snapshot ->> 'typeName'",
		"r.resource_type_snapshot -> 'fieldSchema'",
		"r.resource_type_snapshot -> 'displayTemplate'",
		"r.resource_type_snapshot -> 'commercialRules'",
		"m.status = 'active'",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(publishedResourceDetailSQL, snippet) {
			t.Fatalf("publishedResourceDetailSQL missing %q:\n%s", snippet, publishedResourceDetailSQL)
		}
	}
}

func TestResourceDetailSQLLoadsCityCodeThroughCityStation(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "published detail", query: publishedResourceDetailSQL},
		{name: "own detail", query: ownResourceDetailSQL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, snippet := range []string{"cs.code", "JOIN city_stations cs ON cs.id = r.city_station_id"} {
				if !strings.Contains(tt.query, snippet) {
					t.Fatalf("resource detail SQL missing %q:\n%s", snippet, tt.query)
				}
			}
			if strings.Contains(tt.query, "r.city_code") {
				t.Fatalf("resource detail SQL references nonexistent resources.city_code:\n%s", tt.query)
			}
		})
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

func TestExpiringResourcesSQLUsesSnapshotMessageRules(t *testing.T) {
	requiredSnippets := []string{
		"r.resource_type_snapshot #>> '{messageRules,expiringSoonDays}'",
		"NULLIF(r.resource_type_snapshot #>> '{messageRules,expiringSoonDays}', '') ~ '^[0-9]+$'",
		"GREATEST((r.resource_type_snapshot #>> '{messageRules,expiringSoonDays}')::int, 1)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(listResourcesExpiringSoonSQL, snippet) {
			t.Fatalf("listResourcesExpiringSoonSQL missing %q:\n%s", snippet, listResourcesExpiringSoonSQL)
		}
	}
}

func TestLifecycleMessageInsertIsIdempotent(t *testing.T) {
	if !strings.Contains(listResourcesExpiringSoonSQL, "NOT EXISTS") {
		t.Fatalf("listResourcesExpiringSoonSQL should skip delivered reminders:\n%s", listResourcesExpiringSoonSQL)
	}
}

func TestProfileMonthlyBenefitsForStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		publishQuota int64
		refreshQuota int64
	}{
		{name: "incomplete merchant gets cold-start baseline quota", status: MerchantProfileStatusIncomplete, publishQuota: 3, refreshQuota: 0},
		{name: "completed merchant gets the same baseline quota", status: MerchantProfileStatusCompleted, publishQuota: 3, refreshQuota: 0},
		{name: "unknown status still gets baseline quota", status: " ", publishQuota: 3, refreshQuota: 0},
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

func TestSubmitResourceForReviewLocksSnapshotCommercialRules(t *testing.T) {
	requiredSnippets := []string{
		"r.resource_type_snapshot -> 'commercialRules'",
		"FOR UPDATE",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(submitResourceForReviewLockSQL, snippet) {
			t.Fatalf("submitResourceForReviewLockSQL missing %q:\n%s", snippet, submitResourceForReviewLockSQL)
		}
	}
}
