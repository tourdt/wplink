package entitlement

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListMerchantEntitlementsRequiresMerchantID(t *testing.T) {
	logic := NewListEntitlementsLogic(&fakeEntitlementStore{})

	_, err := logic.ListEntitlements(context.Background(), " ")
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ListEntitlements() error = %v, want validation error", err)
	}
}

func TestListMerchantEntitlementsMapsItems(t *testing.T) {
	store := &fakeEntitlementStore{
		entitlements: []model.MerchantEntitlement{{ID: "entitlement-1", Type: "publish_quota", SourceType: "verification", Status: "active", TotalAmount: 20, RemainingAmount: 20}},
	}
	logic := NewListEntitlementsLogic(store)

	resp, err := logic.ListEntitlements(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("ListEntitlements() error = %v", err)
	}

	if store.merchantID != "merchant-1" || len(resp.Items) != 1 || resp.Items[0].ID != "entitlement-1" || resp.Items[0].Type != "publish_quota" {
		t.Fatalf("merchantID = %q, resp = %#v", store.merchantID, resp)
	}
}

func TestListEntitlementUsageRecordsMapsItems(t *testing.T) {
	store := &fakeEntitlementStore{
		usageRecords: []model.EntitlementUsageRecord{{
			ID: "usage-1", EntitlementID: "entitlement-1", EntitlementType: "top_voucher", ActionType: "top_resource",
			Amount: 1, ResourceID: "resource-1", BeforeRemainingAmount: 3, AfterRemainingAmount: 2, UsedAt: "2026-07-09T10:00:00Z",
		}},
	}
	logic := NewListEntitlementUsageRecordsLogic(store)

	resp, err := logic.ListUsageRecords(context.Background(), " merchant-1 ", " entitlement-1 ")
	if err != nil {
		t.Fatalf("ListUsageRecords() error = %v", err)
	}

	if store.merchantID != "merchant-1" || store.entitlementID != "entitlement-1" || len(resp.Items) != 1 || resp.Items[0].ActionType != "top_resource" {
		t.Fatalf("merchantID = %q entitlementID = %q resp = %#v", store.merchantID, store.entitlementID, resp)
	}
}

func TestRedeemTopVoucherRejectsEmptyResourceID(t *testing.T) {
	logic := NewRedeemTopVoucherLogic(&fakeEntitlementStore{})

	_, err := logic.RedeemTopVoucher(context.Background(), RedeemTopVoucherReq{VoucherID: "voucher-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("RedeemTopVoucher() error = %v, want validation error", err)
	}
}

func TestRedeemTopVoucherPassesIDsToStore(t *testing.T) {
	store := &fakeEntitlementStore{redeemResult: model.RedeemTopVoucherResult{VoucherID: "voucher-1", ResourceID: "resource-1", Status: "used"}}
	logic := NewRedeemTopVoucherLogic(store)

	resp, err := logic.RedeemTopVoucher(context.Background(), RedeemTopVoucherReq{VoucherID: " voucher-1 ", ResourceID: " resource-1 "})
	if err != nil {
		t.Fatalf("RedeemTopVoucher() error = %v", err)
	}

	if store.voucherID != "voucher-1" || store.resourceID != "resource-1" || resp.Status != "used" {
		t.Fatalf("voucherID = %q resourceID = %q resp = %#v", store.voucherID, store.resourceID, resp)
	}
}

type fakeEntitlementStore struct {
	merchantID    string
	entitlementID string
	voucherID     string
	resourceID    string
	entitlements  []model.MerchantEntitlement
	topVouchers   []model.TopVoucher
	usageRecords  []model.EntitlementUsageRecord
	redeemResult  model.RedeemTopVoucherResult
}

func (s *fakeEntitlementStore) ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error) {
	s.merchantID = merchantID
	return append([]model.MerchantEntitlement(nil), s.entitlements...), nil
}

func (s *fakeEntitlementStore) ListTopVouchers(ctx context.Context, merchantID string) ([]model.TopVoucher, error) {
	s.merchantID = merchantID
	return append([]model.TopVoucher(nil), s.topVouchers...), nil
}

func (s *fakeEntitlementStore) ListMerchantEntitlementUsageRecords(ctx context.Context, merchantID string, entitlementID string) ([]model.EntitlementUsageRecord, error) {
	s.merchantID = merchantID
	s.entitlementID = entitlementID
	return append([]model.EntitlementUsageRecord(nil), s.usageRecords...), nil
}

func (s *fakeEntitlementStore) RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (model.RedeemTopVoucherResult, error) {
	s.voucherID = voucherID
	s.resourceID = resourceID
	return s.redeemResult, nil
}
