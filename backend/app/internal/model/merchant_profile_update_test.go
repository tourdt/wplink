package model

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMerchantProfileUpdateCompletesFullProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	profile := supplierProfileForUpdateTest()
	mapVisible := false
	mock.ExpectBegin()
	expectMerchantProfileSnapshot(mock)
	mock.ExpectExec("UPDATE merchants").WithArgs(profileUpdateTestArgs([]string{"童装"}, MerchantProfileStatusCompleted)...).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "童装店", MainCategories: []string{"童装"}, BusinessProfile: &profile, MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: "clear", Status: MerchantLocationStatusHidden}})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMerchantProfileTransactionFailuresKeepStageAndCause(t *testing.T) {
	for _, stage := range []string{"begin", "lock_merchant", "save_profile", "resolve_reports", "record_type_change", "commit"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			cause := errors.New("sensitive-address-13812345678")
			profile := supplierProfileForUpdateTest()
			mapVisible := false
			patch := UpdateMerchantPatch{Name: "童装店", MainCategories: []string{"童装"}, BusinessProfile: &profile, MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: "clear", Status: MerchantLocationStatusHidden}}
			if stage == "resolve_reports" {
				patch.Location = MerchantLocationPatch{Action: "set", Latitude: 30, Longitude: 120, Status: MerchantLocationStatusVisible}
			}
			if stage == "record_type_change" {
				profile = factoryProfileForUpdateTest()
			}
			if stage == "begin" {
				mock.ExpectBegin().WillReturnError(cause)
			} else {
				mock.ExpectBegin()
				if stage == "lock_merchant" {
					mock.ExpectQuery("SELECT").WillReturnError(cause)
				} else {
					expectMerchantProfileSnapshot(mock)
					update := mock.ExpectExec("UPDATE merchants")
					if stage == "save_profile" {
						update.WillReturnError(cause)
					} else {
						update.WillReturnResult(sqlmock.NewResult(0, 1))
						if stage == "resolve_reports" {
							mock.ExpectExec("UPDATE merchant_location_reports").WillReturnError(cause)
						}
						if stage == "record_type_change" {
							mock.ExpectExec("INSERT INTO merchant_type_change_logs").WillReturnError(cause)
						}
					}
				}
				if stage == "commit" {
					mock.ExpectCommit().WillReturnError(cause)
				} else {
					mock.ExpectRollback()
				}
			}
			_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", patch)
			var operationErr *MerchantOperationError
			if !errors.As(err, &operationErr) || operationErr.Stage != stage || !errors.Is(err, cause) {
				t.Fatalf("stage=%s err=%v", stage, err)
			}
			if strings.Contains(err.Error(), "13812345678") {
				t.Fatal("error exposes sensitive cause")
			}
			if err = mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func expectMerchantProfileSnapshot(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`(?s)SELECT.*merchant_type.*location_status.*profile_status.*business_profile.*FOR UPDATE`).WithArgs("101").WillReturnRows(sqlmock.NewRows([]string{"merchant_type", "location_status", "name", "profile_status", "business_profile", "map_visible", "address_text", "latitude", "longitude"}).AddRow("supplier", MerchantLocationStatusHidden, "原店名", MerchantProfileStatusIncomplete, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), true, "", nil, nil))
}

func profileUpdateTestArgs(categories []string, status string) []driver.Value {
	return []driver.Value{"101", "童装店", JSONStringSlice(categories), "supplier", "", "", JSONStringSlice(nil), sqlmock.AnyArg(), "", "", "", "", "clear", "", "", "", "", float64(0), float64(0), "", MerchantLocationStatusHidden, status, nil, nil, sqlmock.AnyArg(), false}
}
