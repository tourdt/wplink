package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

const claimDueResourceAuditRetriesFullCasePattern = `(?s)WITH candidates AS \(.*WHERE status = 'audit_retry'\s+AND audit_retry_at <= NOW\(\)\s+AND \(audit_lease_until IS NULL OR audit_lease_until <= NOW\(\)\).*FOR UPDATE SKIP LOCKED.*UPDATE resources r\s+SET\s+status = CASE WHEN r\.audit_retry_count >= \$2 THEN 'manual_review' ELSE 'audit_retry' END,\s+audit_processing_by = CASE WHEN r\.audit_retry_count >= \$2 THEN NULL ELSE \$3 END,\s+audit_lease_until = CASE\s+WHEN r\.audit_retry_count >= \$2 THEN NULL\s+ELSE NOW\(\) \+ \(\$4 \* INTERVAL '1 millisecond'\)\s+END,\s+audit_retry_count = CASE WHEN r\.audit_retry_count >= \$2 THEN r\.audit_retry_count ELSE r\.audit_retry_count \+ 1 END,\s+audit_retry_at = CASE WHEN r\.audit_retry_count >= \$2 THEN NULL ELSE r\.audit_retry_at END,.*COALESCE\(audit_processing_by, ''\)`

func TestClaimDueResourceAuditRetriesSetsLeaseWithoutChangingStatus(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(claimDueResourceAuditRetriesFullCasePattern).
		WithArgs(int64(20), int64(3), "api-a", int64(120000)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "audit_retry_count", "manual_review", "audit_processing_by"}).
			AddRow("resource-1", int64(1), false, "api-a"))

	claims, err := NewResourceModel(db).ClaimDueResourceAuditRetries(
		context.Background(),
		20,
		3,
		"api-a",
		2*time.Minute,
	)
	if err != nil {
		t.Fatalf("领取内容审核重试租约失败: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("领取结果数量错误: got=%d want=1", len(claims))
	}
	want := ResourceAuditRetryClaim{ResourceID: "resource-1", RetryCount: 1, ProcessingBy: "api-a"}
	if claims[0] != want {
		t.Fatalf("领取结果错误: got=%+v want=%+v", claims[0], want)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestClaimDueResourceAuditRetriesRejectsEmptyProcessingBy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	_, err = NewResourceModel(db).ClaimDueResourceAuditRetries(context.Background(), 20, 3, "   ", 2*time.Minute)
	if err == nil || !strings.Contains(err.Error(), "实例标识不能为空") {
		t.Fatalf("空实例标识错误不符合预期: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestClaimDueResourceAuditRetriesRejectsNonPositiveLease(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{name: "zero", duration: 0},
		{name: "negative", duration: -time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %v", err)
			}
			defer db.Close()

			_, err = NewResourceModel(db).ClaimDueResourceAuditRetries(context.Background(), 20, 3, "api-a", tt.duration)
			if err == nil || !strings.Contains(err.Error(), "租约时长必须大于 0") {
				t.Fatalf("非正租约时长错误不符合预期: duration=%s err=%v", tt.duration, err)
			}
			assertResourceModelSQLExpectations(t, mock)
		})
	}
}

func TestClaimDueResourceAuditRetriesCanReclaimExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)WHERE status = 'audit_retry'\s+AND audit_retry_at <= NOW\(\)\s+AND \(audit_lease_until IS NULL OR audit_lease_until <= NOW\(\)\).*FOR UPDATE SKIP LOCKED`).
		WithArgs(int64(10), int64(5), "api-b", int64(60000)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "audit_retry_count", "manual_review", "audit_processing_by"}).
			AddRow("resource-expired", int64(2), false, "api-b"))

	claims, err := NewResourceModel(db).ClaimDueResourceAuditRetries(context.Background(), 10, 5, "api-b", time.Minute)
	if err != nil {
		t.Fatalf("重新领取过期租约失败: %v", err)
	}
	if len(claims) != 1 || claims[0].ProcessingBy != "api-b" || claims[0].ManualReview {
		t.Fatalf("过期租约重新领取结果错误: %+v", claims)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestClaimDueResourceAuditRetriesMovesMaxedClaimToManualReviewWithoutLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(claimDueResourceAuditRetriesFullCasePattern).
		WithArgs(int64(20), int64(5), "api-a", int64(120000)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "audit_retry_count", "manual_review", "audit_processing_by"}).
			AddRow("resource-manual", int64(5), true, ""))

	claims, err := NewResourceModel(db).ClaimDueResourceAuditRetries(context.Background(), 20, 5, "api-a", 2*time.Minute)
	if err != nil {
		t.Fatalf("领取达到上限的审核重试失败: %v", err)
	}
	want := ResourceAuditRetryClaim{ResourceID: "resource-manual", RetryCount: 5, ManualReview: true, ProcessingBy: ""}
	if len(claims) != 1 || claims[0] != want {
		t.Fatalf("转人工领取结果错误: got=%+v want=%+v", claims, want)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestGetLeasedResourceAuditSnapshotChecksOwnerAndUnexpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	mock.ExpectQuery(`(?s)FROM resources r.*WHERE r.id = \$1\s+AND r.status = 'audit_retry'\s+AND r.audit_processing_by = \$2\s+AND r.audit_lease_until > NOW\(\)\s+AND r.deleted_at IS NULL`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnRows(resourceAuditSnapshotRows("resource-1", "openid-1"))

	snapshot, err := NewResourceModel(db).GetLeasedResourceAuditSnapshot(context.Background(), guard)
	if err != nil {
		t.Fatalf("读取持有租约的审核快照失败: %v", err)
	}
	if snapshot.ID != "resource-1" || snapshot.OpenID != "openid-1" {
		t.Fatalf("审核快照错误: %+v", snapshot)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestGetLeasedResourceAuditSnapshotReturnsLeaseLostForWrongOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	mock.ExpectQuery(`(?s)FROM resources r.*audit_processing_by = \$2.*audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnError(sql.ErrNoRows)

	_, err = NewResourceModel(db).GetLeasedResourceAuditSnapshot(context.Background(), guard)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("错误实例读取租约应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestCreateLeasedResourceContentAuditTasksClearsLeaseAndMovesToPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	expectResourceAuditLeaseLock(mock, guard, true)
	mock.ExpectExec(`DELETE FROM resource_content_audit_tasks WHERE resource_id = \$1`).
		WithArgs(guard.ResourceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO resource_content_audit_tasks .*VALUES \(\$1, \$2, \$3, \$4, 'pending'\)`).
		WithArgs(guard.ResourceID, "trace-1", "image", "https://example.com/image.png").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE resources\s+SET status = 'pending',\s+audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE id = \$1\s+AND status = 'audit_retry'\s+AND audit_processing_by = \$2\s+AND audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = NewResourceModel(db).CreateLeasedResourceContentAuditTasks(context.Background(), guard, []ResourceContentAuditTaskInput{{
		TraceID: "trace-1", AuditType: "image", MediaURL: "https://example.com/image.png",
	}})
	if err != nil {
		t.Fatalf("创建持有租约的媒体审核任务失败: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestCreateLeasedResourceContentAuditTasksRejectsExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	expectResourceAuditLeaseLock(mock, guard, false)
	mock.ExpectRollback()

	err = NewResourceModel(db).CreateLeasedResourceContentAuditTasks(context.Background(), guard, nil)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("过期租约创建媒体任务应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestCreateLeasedResourceContentAuditTasksReturnsLeaseLostWhenFinalGuardUpdatesNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	expectResourceAuditLeaseLock(mock, guard, true)
	mock.ExpectExec(`DELETE FROM resource_content_audit_tasks WHERE resource_id = \$1`).
		WithArgs(guard.ResourceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE resources.*status = 'audit_retry'.*audit_processing_by = \$2.*audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = NewResourceModel(db).CreateLeasedResourceContentAuditTasks(context.Background(), guard, nil)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("最终条件更新未命中应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishResourceAfterMediaAuditLocksResourceBeforeValidatingCurrentTrace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-current"
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'.*WHERE r.id = \$1\s+AND r.status = 'pending'.*FOR UPDATE`).
		WithArgs(resourceID).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title", "commercial_rules"}).
			AddRow("merchant-1", "测试资源", []byte(`{"publish":{"mode":"free"}}`)))
	mock.ExpectQuery(`(?s)SELECT 1\s+WHERE EXISTS \(.*resource_id = \$1.*trace_id = \$2.*status = 'pass'.*\)\s+AND NOT EXISTS \(.*resource_id = \$1.*status IN \('pending', 'rejected', 'failed'\).*\)`).
		WithArgs(resourceID, traceID).
		WillReturnRows(sqlmock.NewRows([]string{"valid"}).AddRow(1))
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'published'.*WHERE resources.id = \$1\s+AND resources.status = 'pending'`).
		WithArgs(resourceID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(resourceID, ResourceStatusPublished))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_approve`).
		WithArgs("merchant:merchant-1", resourceID, "测试资源 已通过内容审核并公开展示", MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewResourceModel(db).PublishResourceAfterMediaAudit(context.Background(), resourceID, traceID)
	if err != nil {
		t.Fatalf("当前图片审核任务发布资源失败: %v", err)
	}
	if result.ID != resourceID || result.Status != ResourceStatusPublished {
		t.Fatalf("发布结果错误: %+v", result)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishResourceAfterMediaAuditRejectsTraceRemovedByNewGeneration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-old"
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'.*WHERE r.id = \$1\s+AND r.status = 'pending'.*FOR UPDATE`).
		WithArgs(resourceID).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title", "commercial_rules"}).
			AddRow("merchant-1", "测试资源", []byte(`{"publish":{"mode":"free"}}`)))
	mock.ExpectQuery(`(?s)SELECT 1\s+WHERE EXISTS \(.*resource_id = \$1.*trace_id = \$2.*status = 'pass'.*\)\s+AND NOT EXISTS \(.*resource_id = \$1.*status IN \('pending', 'rejected', 'failed'\).*\)`).
		WithArgs(resourceID, traceID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).PublishResourceAfterMediaAudit(context.Background(), resourceID, traceID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("新一代任务已删除旧 trace 时应保持资源状态不变: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectResourceAfterMediaAuditLocksResourceBeforeValidatingCurrentTrace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-current"
	reason := "命中风险"
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title\s+FROM resources r.*WHERE r.id = \$1\s+AND r.status = 'pending'.*FOR UPDATE`).
		WithArgs(resourceID).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title"}).AddRow("merchant-1", "测试资源"))
	mock.ExpectQuery(`(?s)SELECT 1\s+WHERE EXISTS \(.*resource_id = \$1.*trace_id = \$2.*status IN \('pass', 'rejected'\).*\)`).
		WithArgs(resourceID, traceID).
		WillReturnRows(sqlmock.NewRows([]string{"valid"}).AddRow(1))
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET status = 'rejected', reject_reason = \$2.*WHERE id = \$1\s+AND status = 'pending'.*RETURNING id::text, status`).
		WithArgs(resourceID, reason).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(resourceID, ResourceStatusRejected))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_reject`).
		WithArgs("merchant:merchant-1", resourceID, "测试资源 "+reason, MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewResourceModel(db).RejectResourceAfterMediaAudit(context.Background(), resourceID, traceID, reason)
	if err != nil {
		t.Fatalf("当前图片审核任务驳回资源失败: %v", err)
	}
	if result.ID != resourceID || result.Status != ResourceStatusRejected {
		t.Fatalf("驳回结果错误: %+v", result)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectResourceAfterMediaAuditRejectsTraceRemovedByNewGeneration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-old"
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title\s+FROM resources r.*WHERE r.id = \$1\s+AND r.status = 'pending'.*FOR UPDATE`).
		WithArgs(resourceID).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title"}).AddRow("merchant-1", "测试资源"))
	mock.ExpectQuery(`(?s)SELECT 1\s+WHERE EXISTS \(.*resource_id = \$1.*trace_id = \$2.*status IN \('pass', 'rejected'\).*\)`).
		WithArgs(resourceID, traceID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).RejectResourceAfterMediaAudit(context.Background(), resourceID, traceID, "命中风险")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("新一代任务已删除旧 trace 时应保持资源状态不变: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkResourceAuditRetryAfterMediaAuditLocksResourceAndValidatesFailedTrace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-current"
	reason := "微信审核服务超时"
	mock.ExpectBegin()
	expectPendingMediaAuditResourceLock(mock, resourceID)
	expectCurrentFailedMediaAuditTrace(mock, resourceID, traceID, true)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'audit_retry',.*WHERE id = \$1\s+AND status = 'pending'.*EXISTS \(.*resource_id = resources.id.*trace_id = \$2.*status = 'failed'.*\).*RETURNING audit_retry_count`).
		WithArgs(resourceID, traceID, reason).
		WillReturnRows(sqlmock.NewRows([]string{"audit_retry_count"}).AddRow(int64(2)))
	mock.ExpectCommit()

	retryCount, err := NewResourceModel(db).MarkResourceAuditRetryAfterMediaAudit(context.Background(), resourceID, traceID, reason)
	if err != nil {
		t.Fatalf("当前失败图片审核任务进入重试失败: %v", err)
	}
	if retryCount != 2 {
		t.Fatalf("重试次数错误: got=%d want=2", retryCount)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkResourceAuditRetryAfterMediaAuditRejectsTraceRemovedByNewGeneration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-old"
	mock.ExpectBegin()
	expectPendingMediaAuditResourceLock(mock, resourceID)
	expectCurrentFailedMediaAuditTrace(mock, resourceID, traceID, false)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).MarkResourceAuditRetryAfterMediaAudit(context.Background(), resourceID, traceID, "微信审核服务超时")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("新一代任务已删除旧 failed trace 时应保持资源状态不变: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkResourceAuditRetryAfterMediaAuditRollsBackWhenFinalTraceGuardMisses(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	resourceID := "resource-1"
	traceID := "trace-old"
	reason := "微信审核服务超时"
	mock.ExpectBegin()
	expectPendingMediaAuditResourceLock(mock, resourceID)
	expectCurrentFailedMediaAuditTrace(mock, resourceID, traceID, true)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'audit_retry',.*WHERE id = \$1\s+AND status = 'pending'.*trace_id = \$2.*status = 'failed'.*RETURNING audit_retry_count`).
		WithArgs(resourceID, traceID, reason).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).MarkResourceAuditRetryAfterMediaAudit(context.Background(), resourceID, traceID, reason)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("最终 trace guard 未命中时应回滚并保持资源状态不变: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishLeasedResourceAfterAuditClearsLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'.*WHERE r.id = \$1\s+AND r.status = 'audit_retry'\s+AND r.audit_processing_by = \$2\s+AND r.audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title", "commercial_rules"}).
			AddRow("merchant-1", "测试资源", []byte(`{"publish":{"mode":"free"}}`)))
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'published',.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE resources.id = \$1\s+AND resources.status = 'audit_retry'\s+AND resources.audit_processing_by = \$3\s+AND resources.audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, sqlmock.AnyArg(), guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("resource-1", ResourceStatusPublished))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_approve`).
		WithArgs("merchant:merchant-1", guard.ResourceID, "测试资源 已通过内容审核并公开展示", MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewResourceModel(db).PublishLeasedResourceAfterAudit(context.Background(), guard)
	if err != nil {
		t.Fatalf("持有租约发布资源失败: %v", err)
	}
	if result.ID != guard.ResourceID || result.Status != ResourceStatusPublished {
		t.Fatalf("发布结果错误: %+v", result)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishLeasedResourceAfterAuditRejectsWrongOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'.*audit_processing_by = \$2.*audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).PublishLeasedResourceAfterAudit(context.Background(), guard)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("错误持有者发布应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishLeasedResourceAfterAuditConsumesQuotaAndRecordsUsageInTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	mock.ExpectBegin()
	expectLeasedPublishSource(mock, guard, `{"publish":{"mode":"consume_quota"}}`)
	expectPublishQuotaUsage(mock, "merchant-1", guard.ResourceID)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'published',.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE resources.id = \$1\s+AND resources.status = 'audit_retry'.*resources.audit_processing_by = \$3.*resources.audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, sqlmock.AnyArg(), guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(guard.ResourceID, ResourceStatusPublished))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_approve`).
		WithArgs("merchant:merchant-1", guard.ResourceID, "测试资源 已通过内容审核并公开展示", MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewResourceModel(db).PublishLeasedResourceAfterAudit(context.Background(), guard)
	if err != nil {
		t.Fatalf("持有租约发布资源时消耗额度失败: %v", err)
	}
	if result.ID != guard.ResourceID || result.Status != ResourceStatusPublished {
		t.Fatalf("发布结果错误: %+v", result)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishLeasedResourceAfterAuditRollsBackWhenMessageInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	wantErr := errors.New("写入发布消息失败")
	mock.ExpectBegin()
	expectLeasedPublishSource(mock, guard, `{"publish":{"mode":"free"}}`)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'published'.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*resources.audit_processing_by = \$3.*resources.audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, sqlmock.AnyArg(), guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(guard.ResourceID, ResourceStatusPublished))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_approve`).
		WithArgs("merchant:merchant-1", guard.ResourceID, "测试资源 已通过内容审核并公开展示", MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).PublishLeasedResourceAfterAudit(context.Background(), guard)
	if !errors.Is(err, wantErr) {
		t.Fatalf("发布消息写入失败应回滚资源状态: got=%v want=%v", err, wantErr)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestPublishLeasedResourceAfterAuditRollsBackQuotaWhenFinalGuardMisses(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	mock.ExpectBegin()
	expectLeasedPublishSource(mock, guard, `{"publish":{"mode":"consume_quota"}}`)
	// 先消耗额度并记录用量，再模拟最终 guard 未命中，以验证所有前置写入都随事务回滚。
	expectPublishQuotaUsage(mock, "merchant-1", guard.ResourceID)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'published'.*resources.audit_processing_by = \$3.*resources.audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, sqlmock.AnyArg(), guard.ProcessingBy).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).PublishLeasedResourceAfterAudit(context.Background(), guard)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("最终租约 guard 未命中应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectLeasedResourceAfterAuditClearsLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title\s+FROM resources r.*WHERE r.id = \$1\s+AND r.status = 'audit_retry'\s+AND r.audit_processing_by = \$2\s+AND r.audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title"}).AddRow("merchant-1", "测试资源"))
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET status = 'rejected', reject_reason = \$3,\s+audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE id = \$1\s+AND status = 'audit_retry'\s+AND audit_processing_by = \$2\s+AND audit_lease_until > NOW\(\).*RETURNING id::text, status`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, "命中风险").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("resource-1", ResourceStatusRejected))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_reject`).
		WithArgs("merchant:merchant-1", guard.ResourceID, "测试资源 命中风险", MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewResourceModel(db).RejectLeasedResourceAfterAudit(context.Background(), guard, "命中风险")
	if err != nil {
		t.Fatalf("持有租约拒绝资源失败: %v", err)
	}
	if result.ID != guard.ResourceID || result.Status != ResourceStatusRejected {
		t.Fatalf("拒绝结果错误: %+v", result)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectLeasedResourceAfterAuditRejectsExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title.*audit_processing_by = \$2.*audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).RejectLeasedResourceAfterAudit(context.Background(), guard, "命中风险")
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("过期租约拒绝应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectLeasedResourceAfterAuditRollsBackWhenMessageInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	reason := "命中风险"
	wantErr := errors.New("写入驳回消息失败")
	mock.ExpectBegin()
	expectLeasedRejectSource(mock, guard)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET status = 'rejected', reject_reason = \$3,.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*audit_processing_by = \$2.*audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, reason).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(guard.ResourceID, ResourceStatusRejected))
	mock.ExpectExec(`(?s)INSERT INTO messages .*resource_auto_reject`).
		WithArgs("merchant:merchant-1", guard.ResourceID, "测试资源 "+reason, MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).RejectLeasedResourceAfterAudit(context.Background(), guard, reason)
	if !errors.Is(err, wantErr) {
		t.Fatalf("驳回消息写入失败应回滚资源状态: got=%v want=%v", err, wantErr)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestRejectLeasedResourceAfterAuditReturnsLeaseLostWhenFinalGuardMisses(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	reason := "命中风险"
	mock.ExpectBegin()
	expectLeasedRejectSource(mock, guard)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET status = 'rejected', reject_reason = \$3,.*audit_processing_by = \$2.*audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, reason).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).RejectLeasedResourceAfterAudit(context.Background(), guard, reason)
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("最终租约 guard 未命中应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkLeasedResourceAuditRetryClearsLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	expectResourceAuditLeaseLock(mock, guard, true)
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET\s+status = 'audit_retry',.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE id = \$1\s+AND status = 'audit_retry'\s+AND audit_processing_by = \$2\s+AND audit_lease_until > NOW\(\).*RETURNING audit_retry_count`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, "微信审核超时").
		WillReturnRows(sqlmock.NewRows([]string{"audit_retry_count"}).AddRow(int64(2)))
	mock.ExpectCommit()

	retryCount, err := NewResourceModel(db).MarkLeasedResourceAuditRetry(context.Background(), guard, "微信审核超时")
	if err != nil {
		t.Fatalf("持有租约再次标记重试失败: %v", err)
	}
	if retryCount != 2 {
		t.Fatalf("重试次数错误: got=%d want=2", retryCount)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkLeasedResourceAuditRetryRejectsLeaseLostDuringUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	expectResourceAuditLeaseLock(mock, guard, true)
	mock.ExpectQuery(`(?s)UPDATE resources.*audit_processing_by = \$2.*audit_lease_until > NOW\(\).*RETURNING audit_retry_count`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, "微信审核超时").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewResourceModel(db).MarkLeasedResourceAuditRetry(context.Background(), guard, "微信审核超时")
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("写入前租约失效应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkLeasedResourceManualReviewClearsLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	expectResourceAuditLeaseLock(mock, guard, true)
	mock.ExpectExec(`(?s)UPDATE resources\s+SET\s+status = 'manual_review',.*audit_lease_until = NULL,\s+audit_processing_by = NULL,.*WHERE id = \$1\s+AND status = 'audit_retry'\s+AND audit_processing_by = \$2\s+AND audit_lease_until > NOW\(\)`).
		WithArgs(guard.ResourceID, guard.ProcessingBy, "缺少审核身份").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = NewResourceModel(db).MarkLeasedResourceManualReview(context.Background(), guard, "缺少审核身份")
	if err != nil {
		t.Fatalf("持有租约转人工失败: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkLeasedResourceManualReviewRejectsWrongOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	guard := ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	expectResourceAuditLeaseLock(mock, guard, false)
	mock.ExpectRollback()

	err = NewResourceModel(db).MarkLeasedResourceManualReview(context.Background(), guard, "缺少审核身份")
	if !errors.Is(err, ErrResourceAuditLeaseLost) {
		t.Fatalf("错误持有者转人工应返回 ErrResourceAuditLeaseLost: %v", err)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func TestMarkStaleContentAuditTasksForRetryClearsLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`(?s)UPDATE resources\s+SET\s+status = 'audit_retry',\s+audit_retry_at = now\(\),\s+audit_last_error = '图片审核回调超时',\s+audit_lease_until = NULL,\s+audit_processing_by = NULL,`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	count, err := NewResourceModel(db).MarkStaleContentAuditTasksForRetry(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("修复过期媒体审核任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("修复数量错误: got=%d want=1", count)
	}
	assertResourceModelSQLExpectations(t, mock)
}

func expectResourceAuditLeaseLock(mock sqlmock.Sqlmock, guard ResourceAuditGuard, held bool) {
	mock.ExpectBegin()
	expectation := mock.ExpectQuery(`(?s)SELECT 1\s+FROM resources\s+WHERE id = \$1\s+AND status = 'audit_retry'\s+AND audit_processing_by = \$2\s+AND audit_lease_until > NOW\(\)\s+AND deleted_at IS NULL\s+FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy)
	if held {
		expectation.WillReturnRows(sqlmock.NewRows([]string{"lease_held"}).AddRow(1))
		return
	}
	expectation.WillReturnError(sql.ErrNoRows)
}

func expectLeasedPublishSource(mock sqlmock.Sqlmock, guard ResourceAuditGuard, commercialRules string) {
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'.*WHERE r.id = \$1\s+AND r.status = 'audit_retry'\s+AND r.audit_processing_by = \$2\s+AND r.audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title", "commercial_rules"}).
			AddRow("merchant-1", "测试资源", []byte(commercialRules)))
}

func expectPublishQuotaUsage(mock sqlmock.Sqlmock, merchantID string, resourceID string) {
	mock.ExpectQuery(`(?s)SELECT\s+COALESCE\(profile_status, 'incomplete'\) AS profile_status,.*AS has_active_vip\s+FROM merchants\s+WHERE id = \$1.*FOR UPDATE`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{"profile_status", "has_active_vip"}).AddRow(MerchantProfileStatusCompleted, true))
	mock.ExpectQuery(`(?s)UPDATE merchant_entitlements\s+SET used_amount = used_amount \+ 1,.*entitlement_type = 'publish_quota'.*RETURNING id::text, remaining_amount \+ 1, remaining_amount`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "before_remaining_amount", "after_remaining_amount"}).
			AddRow("entitlement-1", int64(3), int64(2)))
	mock.ExpectExec(`(?s)INSERT INTO merchant_entitlement_usage_records .*VALUES`).
		WithArgs(
			"entitlement-1",
			merchantID,
			EntitlementTypePublishQuota,
			ActionTypePublishResource,
			int64(1),
			resourceID,
			int64(3),
			int64(2),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectLeasedRejectSource(mock sqlmock.Sqlmock, guard ResourceAuditGuard) {
	mock.ExpectQuery(`(?s)SELECT r.merchant_id::text, r.title\s+FROM resources r.*WHERE r.id = \$1\s+AND r.status = 'audit_retry'\s+AND r.audit_processing_by = \$2\s+AND r.audit_lease_until > NOW\(\).*FOR UPDATE`).
		WithArgs(guard.ResourceID, guard.ProcessingBy).
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "title"}).AddRow("merchant-1", "测试资源"))
}

func expectPendingMediaAuditResourceLock(mock sqlmock.Sqlmock, resourceID string) {
	mock.ExpectQuery(`(?s)SELECT 1\s+FROM resources\s+WHERE id = \$1\s+AND status = 'pending'\s+AND deleted_at IS NULL\s+FOR UPDATE`).
		WithArgs(resourceID).
		WillReturnRows(sqlmock.NewRows([]string{"resource_locked"}).AddRow(1))
}

func expectCurrentFailedMediaAuditTrace(mock sqlmock.Sqlmock, resourceID string, traceID string, exists bool) {
	expectation := mock.ExpectQuery(`(?s)SELECT 1\s+WHERE EXISTS \(.*resource_id = \$1.*trace_id = \$2.*status = 'failed'.*\)`).
		WithArgs(resourceID, traceID)
	if exists {
		expectation.WillReturnRows(sqlmock.NewRows([]string{"valid"}).AddRow(1))
		return
	}
	expectation.WillReturnError(sql.ErrNoRows)
}

func resourceAuditSnapshotRows(resourceID string, openID string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "merchant_id", "type_code", "wechat_openid", "title", "category", "district", "price_text",
		"quantity_text", "description", "attributes", "tags", "images", "contact_name", "contact_wechat",
	}).AddRow(
		resourceID, "merchant-1", "supply", openID, "测试资源", "分类", "区域", "面议",
		"10件", "描述", []byte(`{}`), []byte(`[]`), []byte(`[]`), "联系人", "wechat-id",
	)
}

func assertResourceModelSQLExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}
}
