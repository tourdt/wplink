package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type MerchantAdminStore interface {
	ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error)
}

type ListMerchantsReq struct {
	CityCode     string
	MerchantType string
	Status       string
	Page         int64
	PageSize     int64
}

type MerchantListItem struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MerchantType       string `json:"merchantType"`
	VerificationStatus string `json:"verificationStatus"`
	Status             string `json:"status"`
	LastActiveAt       string `json:"lastActiveAt,omitempty"`
}

type ListMerchantsResp struct {
	Items    []MerchantListItem `json:"items"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"pageSize"`
	Total    int64              `json:"total"`
}

type MerchantAdminLogic struct {
	store MerchantAdminStore
}

func NewMerchantAdminLogic(store MerchantAdminStore) *MerchantAdminLogic {
	return &MerchantAdminLogic{store: store}
}

func (l *MerchantAdminLogic) ListMerchants(ctx context.Context, req ListMerchantsReq) (ListMerchantsResp, error) {
	result, err := l.store.ListMerchants(ctx, model.ListMerchantsFilter{
		CityCode:     strings.TrimSpace(req.CityCode),
		MerchantType: strings.TrimSpace(req.MerchantType),
		Status:       strings.TrimSpace(req.Status),
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return ListMerchantsResp{}, err
	}

	items := make([]MerchantListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MerchantListItem{
			ID:                 item.ID,
			Name:               item.Name,
			MerchantType:       item.MerchantType,
			VerificationStatus: item.VerificationStatus,
			Status:             item.Status,
			LastActiveAt:       item.LastActiveAt,
		})
	}
	return ListMerchantsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
