package admin

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type DemandAdminStore interface {
	ListDemands(ctx context.Context, filter model.ListDemandsFilter) (model.ListDemandsResult, error)
	GetDemand(ctx context.Context, demandID string) (model.DemandDetail, error)
	UpdateDemandStatus(ctx context.Context, demandID string, status string) (model.UpdateDemandStatusResult, error)
}

type ListDemandsReq struct {
	CityCode   string
	DemandType string
	Status     string
	Page       int64
	PageSize   int64
}

type AdminDemandItem struct {
	ID          string
	Title       string
	DemandType  string
	Category    string
	ContactName string
	Status      string
	CreatedAt   string
}

type ListDemandsResp struct {
	Items    []AdminDemandItem
	Page     int64
	PageSize int64
	Total    int64
}

type GetDemandReq struct {
	DemandID string
}

type DemandContact struct {
	Name   string
	Phone  string
	Wechat string
}

type DemandDetailResp struct {
	ID                  string
	Title               string
	DemandType          string
	Category            string
	PriceRange          model.JSONMap
	QuantityRequirement model.JSONMap
	Attributes          model.JSONMap
	Contact             DemandContact
	Status              string
	CreatedAt           string
}

type UpdateDemandStatusReq struct {
	DemandID string
	Status   string
}

type UpdateDemandStatusResp struct {
	ID     string
	Status string
}

type DemandAdminLogic struct {
	store DemandAdminStore
}

func NewDemandAdminLogic(store DemandAdminStore) *DemandAdminLogic {
	return &DemandAdminLogic{store: store}
}

func (l *DemandAdminLogic) ListDemands(ctx context.Context, req ListDemandsReq) (ListDemandsResp, error) {
	result, err := l.store.ListDemands(ctx, model.ListDemandsFilter{
		CityCode:   strings.TrimSpace(req.CityCode),
		DemandType: strings.TrimSpace(req.DemandType),
		Status:     strings.TrimSpace(req.Status),
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ListDemandsResp{}, err
	}
	items := make([]AdminDemandItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, AdminDemandItem{
			ID: item.ID, Title: item.Title, DemandType: item.DemandType, Category: item.Category,
			ContactName: item.ContactName, Status: item.Status, CreatedAt: item.CreatedAt,
		})
	}
	return ListDemandsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *DemandAdminLogic) GetDemand(ctx context.Context, req GetDemandReq) (DemandDetailResp, error) {
	demandID := strings.TrimSpace(req.DemandID)
	if demandID == "" {
		return DemandDetailResp{}, errx.New(errx.CodeValidationFailed, "需求不存在")
	}
	detail, err := l.store.GetDemand(ctx, demandID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DemandDetailResp{}, errx.New(errx.CodeResourceNotFound, "需求不存在")
		}
		return DemandDetailResp{}, err
	}
	return DemandDetailResp{
		ID:                  detail.ID,
		Title:               detail.Title,
		DemandType:          detail.DemandType,
		Category:            detail.Category,
		PriceRange:          detail.PriceRange,
		QuantityRequirement: detail.QuantityRequirement,
		Attributes:          detail.Attributes,
		Contact:             DemandContact{Name: detail.ContactName, Phone: detail.ContactPhone, Wechat: detail.ContactWechat},
		Status:              detail.Status,
		CreatedAt:           detail.CreatedAt,
	}, nil
}

func (l *DemandAdminLogic) UpdateDemandStatus(ctx context.Context, req UpdateDemandStatusReq) (UpdateDemandStatusResp, error) {
	demandID := strings.TrimSpace(req.DemandID)
	status := strings.TrimSpace(req.Status)
	if demandID == "" {
		return UpdateDemandStatusResp{}, errx.New(errx.CodeValidationFailed, "需求不存在")
	}
	if !isSupportedDemandStatus(status) {
		return UpdateDemandStatusResp{}, errx.New(errx.CodeValidationFailed, "需求状态不正确")
	}
	result, err := l.store.UpdateDemandStatus(ctx, demandID, status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateDemandStatusResp{}, errx.New(errx.CodeResourceNotFound, "需求不存在")
		}
		return UpdateDemandStatusResp{}, err
	}
	return UpdateDemandStatusResp{ID: result.ID, Status: result.Status}, nil
}

func isSupportedDemandStatus(status string) bool {
	switch status {
	case "pending", "matching", "closed":
		return true
	default:
		return false
	}
}
