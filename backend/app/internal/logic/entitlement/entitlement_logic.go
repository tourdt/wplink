package entitlement

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type EntitlementStore interface {
	ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error)
	ListTopVouchers(ctx context.Context, merchantID string) ([]model.TopVoucher, error)
	RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (model.RedeemTopVoucherResult, error)
}

type MerchantEntitlementInfo struct {
	Type            string `json:"type"`
	SourceType      string `json:"sourceType"`
	TotalAmount     int64  `json:"totalAmount"`
	UsedAmount      int64  `json:"usedAmount"`
	RemainingAmount int64  `json:"remainingAmount"`
	ExpiresAt       string `json:"expiresAt,omitempty"`
}

type ListMerchantEntitlementsResp struct {
	Items []MerchantEntitlementInfo `json:"items"`
}

type TopVoucherInfo struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	TopDurationHours int64    `json:"topDurationHours"`
	AllowedTypeCodes []string `json:"allowedTypeCodes"`
	ExpiresAt        string   `json:"expiresAt,omitempty"`
}

type ListTopVouchersResp struct {
	Items []TopVoucherInfo `json:"items"`
}

type RedeemTopVoucherReq struct {
	VoucherID  string
	ResourceID string
}

type RedeemTopVoucherResp struct {
	VoucherID  string `json:"voucherId"`
	ResourceID string `json:"resourceId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type ListEntitlementsLogic struct {
	store EntitlementStore
}

func NewListEntitlementsLogic(store EntitlementStore) *ListEntitlementsLogic {
	return &ListEntitlementsLogic{store: store}
}

func (l *ListEntitlementsLogic) ListEntitlements(ctx context.Context, merchantID string) (ListMerchantEntitlementsResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return ListMerchantEntitlementsResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	entitlements, err := l.store.ListMerchantEntitlements(ctx, merchantID)
	if err != nil {
		return ListMerchantEntitlementsResp{}, err
	}
	items := make([]MerchantEntitlementInfo, 0, len(entitlements))
	for _, item := range entitlements {
		items = append(items, MerchantEntitlementInfo{
			Type: item.Type, SourceType: item.SourceType, TotalAmount: item.TotalAmount,
			UsedAmount: item.UsedAmount, RemainingAmount: item.RemainingAmount, ExpiresAt: item.ExpiresAt,
		})
	}
	return ListMerchantEntitlementsResp{Items: items}, nil
}

type ListTopVouchersLogic struct {
	store EntitlementStore
}

func NewListTopVouchersLogic(store EntitlementStore) *ListTopVouchersLogic {
	return &ListTopVouchersLogic{store: store}
}

func (l *ListTopVouchersLogic) ListTopVouchers(ctx context.Context, merchantID string) (ListTopVouchersResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return ListTopVouchersResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	vouchers, err := l.store.ListTopVouchers(ctx, merchantID)
	if err != nil {
		return ListTopVouchersResp{}, err
	}
	items := make([]TopVoucherInfo, 0, len(vouchers))
	for _, item := range vouchers {
		items = append(items, TopVoucherInfo{
			ID: item.ID, Status: item.Status, TopDurationHours: item.TopDurationHours,
			AllowedTypeCodes: append([]string(nil), item.AllowedTypeCodes...), ExpiresAt: item.ExpiresAt,
		})
	}
	return ListTopVouchersResp{Items: items}, nil
}

type RedeemTopVoucherLogic struct {
	store EntitlementStore
}

func NewRedeemTopVoucherLogic(store EntitlementStore) *RedeemTopVoucherLogic {
	return &RedeemTopVoucherLogic{store: store}
}

func (l *RedeemTopVoucherLogic) RedeemTopVoucher(ctx context.Context, req RedeemTopVoucherReq) (RedeemTopVoucherResp, error) {
	voucherID := strings.TrimSpace(req.VoucherID)
	resourceID := strings.TrimSpace(req.ResourceID)
	if voucherID == "" || resourceID == "" {
		return RedeemTopVoucherResp{}, errx.New(errx.CodeValidationFailed, "请选择置顶券和已发布资源")
	}
	result, err := l.store.RedeemTopVoucher(ctx, voucherID, resourceID)
	if err != nil {
		return RedeemTopVoucherResp{}, err
	}
	return RedeemTopVoucherResp{
		VoucherID: result.VoucherID, ResourceID: result.ResourceID, Status: result.Status, Message: "置顶券已使用",
	}, nil
}
