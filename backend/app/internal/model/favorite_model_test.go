package model

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestListFollowedMerchantsReadsProfileStatusFromCurrentSchema 防止关注列表查询引用
// merchants 表中不存在的验证状态字段，导致用户打开收藏页时直接收到 500。
func TestListFollowedMerchantsReadsProfileStatusFromCurrentSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	followedAt := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT.*m\.profile_status.*FROM user_followed_merchants`).
		WithArgs("101", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "merchant_type", "profile_status", "main_categories", "logo_url", "followed_at", "total",
		}).AddRow(
			"201", "织里童装工厂", "factory", "completed", []byte(`["童装"]`), "https://img.example.com/logo.png", followedAt, 1,
		))

	result, err := NewFavoriteModel(db).ListFollowedMerchants(context.Background(), "101", ListInteractionFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListFollowedMerchants() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.ID != "201" || item.VerificationStatus != "completed" || len(item.MainCategories) != 1 || item.MainCategories[0] != "童装" || item.FollowedAt != "2026-07-01T08:00:00Z" {
		t.Fatalf("item = %+v, want mapped merchant profile", item)
	}
	if result.Total != 1 || result.Page != 1 || result.PageSize != 20 {
		t.Fatalf("result = %+v, want pagination total 1", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}
