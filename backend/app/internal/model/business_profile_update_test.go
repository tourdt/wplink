package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectBusinessProfileSnapshot(mock sqlmock.Sqlmock, raw []byte, locationStatus, address string, latitude, longitude interface{}) {
	mock.ExpectQuery(`(?s)SELECT.*merchant_type.*business_profile.*map_visible.*FOR UPDATE`).WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_type", "location_status", "name", "profile_status", "business_profile", "map_visible", "address_text", "latitude", "longitude"}).
			AddRow("supplier", locationStatus, "旧主体", MerchantProfileStatusIncomplete, raw, true, address, latitude, longitude))
}

func TestNewLandlordProfileCompletesWithoutGPSAndStoresPrimaryRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mapVisible := false
	profile := BusinessProfile{SchemaVersion: 1, EntityKind: "individual", PrimaryRole: "landlord", Roles: []string{"landlord"}, PropertyTypes: []string{"商铺"}}
	mock.ExpectBegin()
	expectBusinessProfileSnapshot(mock, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), MerchantLocationStatusHidden, "", nil, nil)
	mock.ExpectExec(`UPDATE merchants`).WithArgs(
		"101", "新房东", JSONStringSlice(nil), "landlord", "", "", JSONStringSlice(nil), sqlmock.AnyArg(),
		"", "", "", "", "keep", "", "", "", "", float64(0), float64(0), "", "", MerchantProfileStatusCompleted,
		nil, nil, sqlmock.AnyArg(), false,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO merchant_type_change_logs`).WithArgs("101", "supplier", "landlord", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "新房东", BusinessProfile: &profile, MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNewBusinessProfileRejectsMissingCategoriesUnderLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	profile := factoryProfileForUpdateTest()
	mapVisible := false
	mock.ExpectBegin()
	expectBusinessProfileSnapshot(mock, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), MerchantLocationStatusHidden, "", nil, nil)
	mock.ExpectRollback()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "新厂名", BusinessProfile: &profile, MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	var validationErr *MerchantValidationError
	if !errors.As(err, &validationErr) || !strings.Contains(validationErr.Message, "经营品类") {
		t.Fatalf("err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessProfileSaveRequiresFullForm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "供货主体", Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	var validationErr *MerchantValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func factoryProfileForUpdateTest() BusinessProfile {
	return BusinessProfile{SchemaVersion: 1, EntityKind: "organization", PrimaryRole: "factory", Roles: []string{"factory"}, ProcessingMethods: []string{"车缝"}}
}
func supplierProfileForUpdateTest() BusinessProfile {
	return BusinessProfile{SchemaVersion: 1, EntityKind: "organization", PrimaryRole: "supplier", Roles: []string{"supplier"}}
}

func TestExplicitMapEnableRequiresAddressAndCoordinatesUnderLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mapVisible := true
	mock.ExpectBegin()
	expectBusinessProfileSnapshot(mock, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), MerchantLocationStatusHidden, "", nil, nil)
	mock.ExpectRollback()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "旧主体", MainCategories: []string{"童装"}, BusinessProfile: func() *BusinessProfile { p := supplierProfileForUpdateTest(); return &p }(), MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	var validationErr *MerchantValidationError
	if !errors.As(err, &validationErr) || !strings.Contains(validationErr.Message, "地图位置") {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestKeepRiskLocationDoesNotResolveReportsOrChangeStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mapVisible := false
	mock.ExpectBegin()
	expectBusinessProfileSnapshot(mock, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), MerchantLocationStatusRiskHidden, "旧私有地址", 30.1, 120.2)
	mock.ExpectExec(`(?s)UPDATE merchants.*address_text = CASE WHEN \$13 = 'keep' THEN address_text.*location_status = CASE WHEN \$13 = 'set'`).
		WithArgs("101", "旧主体", JSONStringSlice([]string{"童装"}), "supplier", "", "", JSONStringSlice(nil), sqlmock.AnyArg(), "", "", "", "新地址不得覆盖", "keep", "", "", "", "", float64(0), float64(0), "", "", MerchantProfileStatusCompleted, nil, nil, sqlmock.AnyArg(), false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "旧主体", MainCategories: []string{"童装"}, AddressText: "新地址不得覆盖", BusinessProfile: func() *BusinessProfile { p := supplierProfileForUpdateTest(); return &p }(), MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiskHiddenKeepCanRetainMapChoiceWithoutResolvingReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mapVisible := true
	mock.ExpectBegin()
	expectBusinessProfileSnapshot(mock, []byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`), MerchantLocationStatusRiskHidden, "旧详细地址", 30.1, 120.2)
	mock.ExpectExec(`(?s)UPDATE merchants.*location_status = CASE WHEN \$13 = 'set' THEN \$21`).
		WithArgs("101", "旧主体", JSONStringSlice([]string{"童装"}), "supplier", "", "", JSONStringSlice(nil), sqlmock.AnyArg(), "", "", "", "", "keep", "", "", "", "", float64(0), float64(0), "", "", MerchantProfileStatusCompleted, nil, nil, sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err = NewMerchantModel(db).UpdateMerchant(context.Background(), "101", UpdateMerchantPatch{Name: "旧主体", MainCategories: []string{"童装"}, BusinessProfile: func() *BusinessProfile { p := supplierProfileForUpdateTest(); return &p }(), MapVisible: &mapVisible, Location: MerchantLocationPatch{Action: MerchantLocationActionKeep}})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessProfileValidationErrorRemainsInspectableThroughTransaction(t *testing.T) {
	err := &MerchantOperationError{Stage: "validate_profile", Cause: &MerchantValidationError{Message: "请填写出租类型"}}
	var validationErr *MerchantValidationError
	if !errors.As(err, &validationErr) || validationErr.Message != "请填写出租类型" {
		t.Fatalf("err = %v", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatal("validation is not missing merchant")
	}
}
