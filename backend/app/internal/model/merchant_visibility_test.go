package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMerchantDirectoryQueriesRespectOwnerVisibility(t *testing.T) {
	where, _ := buildMerchantDirectoryWhere(MerchantDirectoryFilter{})
	for _, query := range []string{where, buildNearbyVisibleMerchantsSQLForTest(t)} {
		if !strings.Contains(query, "m.owner_visible = TRUE") || !strings.Contains(query, "m.map_visible = TRUE") {
			t.Fatalf("missing owner and map visibility conditions: %s", query)
		}
	}
}

func buildNearbyVisibleMerchantsSQLForTest(t *testing.T) string {
	t.Helper()
	query, _ := buildNearbyVisibleMerchantsQuery(MerchantDirectoryItem{MerchantID: "101", Latitude: 30, Longitude: 120}, 1000)
	return query
}

func TestSetMerchantVisibilityUpdatesOnlyOwnerSetting(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`UPDATE merchants SET owner_visible = \$2, updated_at = \$3 WHERE id = \$1 AND deleted_at IS NULL`).WithArgs("101", false, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewMerchantModel(db).SetMerchantVisibility(context.Background(), "101", false); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetMerchantVisibilityMissingMerchant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`UPDATE merchants`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := NewMerchantModel(db).SetMerchantVisibility(context.Background(), "101", false); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v", err)
	}
}
