package model

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListMerchantsFiltersLocationStatusAndReturnsGovernanceSummaryWithoutDuplicates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	lastActiveAt := time.Date(2026, time.August, 29, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT m.id::text, m.name, m.merchant_type, m.status, m.location_status, COALESCE\(m.market_name, ''\), COALESCE\(m.building_name, ''\), COALESCE\(m.floor_no, ''\), COALESCE\(m.booth_no, ''\), m.last_active_at, \( SELECT COUNT\(\*\) FROM merchant_location_reports mlr WHERE mlr.merchant_id = m.id AND mlr.status = 'pending' \) AS pending_location_report_count, COUNT\(\*\) OVER\(\) AS total FROM merchants m JOIN city_stations cs ON cs.id = m.city_station_id WHERE m.deleted_at IS NULL AND \(\$1 = '' OR cs.code = \$1\) AND \(\$2 = '' OR m.merchant_type = \$2\) AND \(\$3 = '' OR m.status = \$3\) AND \(\$4 = '' OR m.location_status = \$4\) AND \(\$5 = '' OR m.name ILIKE '%' \|\| \$5 \|\| '%' OR m.contact_name ILIKE '%' \|\| \$5 \|\| '%'\) ORDER BY m.created_at DESC LIMIT \$6 OFFSET \$7`).
		WithArgs("zhili", "factory", MerchantStatusActive, MerchantLocationStatusRiskHidden, "晨星", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "merchant_type", "status", "location_status", "market_name", "building_name", "floor_no", "booth_no", "last_active_at", "pending_location_report_count", "total",
		}).AddRow("101", "织里晨星童装", "factory", MerchantStatusActive, MerchantLocationStatusRiskHidden, "织里童装城", "一期", "1F", "A108", lastActiveAt, int64(3), int64(1)))

	result, err := NewMerchantModel(db).ListMerchants(context.Background(), ListMerchantsFilter{
		CityCode: "zhili", MerchantType: "factory", Status: MerchantStatusActive,
		LocationStatus: MerchantLocationStatusRiskHidden, Keyword: "晨星", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListMerchants() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = %#v, want one merchant without report JOIN duplication", result)
	}
	item := result.Items[0]
	if item.LocationStatus != MerchantLocationStatusRiskHidden || item.MarketName != "织里童装城" || item.BuildingName != "一期" || item.FloorNo != "1F" || item.BoothNo != "A108" || item.PendingLocationReportCount != 3 || item.LastActiveAt != lastActiveAt.Format(time.RFC3339) {
		t.Fatalf("item = %#v, want complete location governance summary", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("merchant list expectations: %v", err)
	}
}

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

func TestUpdateMerchantSetResolvesPendingLocationReportsInSameTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT merchant_type, location_status FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_type", "location_status"}).AddRow("stockist", MerchantLocationStatusRiskHidden))
	mock.ExpectExec(`UPDATE merchants.*location_status = CASE WHEN \$13 = 'set' THEN \$21`).
		WithArgs(
			"101", "织里晨星童装", JSONStringSlice([]string{"童装"}), "stockist", "", "", JSONStringSlice(nil), sqlmock.AnyArg(),
			"", "", "", "织里镇利济路", "set", "织里童装城", "一期", "1F", "A108", 30.8732, 120.2255, "织里童装城", MerchantLocationStatusVisible,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE merchant_location_reports.*SET status = 'resolved'.*WHERE merchant_id = \$1.*status = 'pending'`).
		WithArgs("101").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{
		Name: "织里晨星童装", MainCategories: []string{"童装"}, AddressText: "织里镇利济路",
		Location: MerchantLocationPatch{
			Action: "set", MarketName: "织里童装城", BuildingName: "一期", FloorNo: "1F", BoothNo: "A108",
			Latitude: 30.8732, Longitude: 120.2255, LocationName: "织里童装城", Status: MerchantLocationStatusVisible,
		},
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("transaction expectations: %v", err)
	}
}

func TestUpdateMerchantClearDoesNotResolvePendingLocationReports(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT merchant_type, location_status FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_type", "location_status"}).AddRow("stockist", MerchantLocationStatusVisible))
	mock.ExpectExec(`UPDATE merchants`).
		WithArgs(
			"101", "织里晨星童装", JSONStringSlice(nil), "stockist", "", "", JSONStringSlice(nil), sqlmock.AnyArg(),
			"", "", "", "", "clear", "", "", "", "", float64(0), float64(0), "", MerchantLocationStatusHidden,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{
		Name: "织里晨星童装", Location: MerchantLocationPatch{Action: "clear", Status: MerchantLocationStatusHidden},
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("transaction expectations: %v", err)
	}
}
