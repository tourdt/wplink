package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type EntitlementAdminStore interface {
	GrantMerchantEntitlement(ctx context.Context, input model.GrantEntitlementInput) (model.GrantEntitlementResult, error)
}

type GrantEntitlementReq struct {
	MerchantID      string
	OperatorID      string
	EntitlementType string
	SourceType      string
	TotalAmount     int64
	ExpiresAt       string
	Reason          string
}

type GrantEntitlementResp struct {
	ID      string
	Message string
}

type EntitlementAdminLogic struct {
	store EntitlementAdminStore
}

func NewEntitlementAdminLogic(store EntitlementAdminStore) *EntitlementAdminLogic {
	return &EntitlementAdminLogic{store: store}
}

func (l *EntitlementAdminLogic) GrantMerchantEntitlement(ctx context.Context, req GrantEntitlementReq) (GrantEntitlementResp, error) {
	input := model.GrantEntitlementInput{
		MerchantID:      strings.TrimSpace(req.MerchantID),
		OperatorID:      strings.TrimSpace(req.OperatorID),
		EntitlementType: strings.TrimSpace(req.EntitlementType),
		SourceType:      strings.TrimSpace(req.SourceType),
		TotalAmount:     req.TotalAmount,
		ExpiresAt:       strings.TrimSpace(req.ExpiresAt),
		Reason:          strings.TrimSpace(req.Reason),
	}
	if input.MerchantID == "" {
		return GrantEntitlementResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	if input.EntitlementType != "publish_quota" && input.EntitlementType != "refresh_quota" {
		return GrantEntitlementResp{}, errx.New(errx.CodeValidationFailed, "权益类型不正确")
	}
	if input.SourceType == "" || input.Reason == "" {
		return GrantEntitlementResp{}, errx.New(errx.CodeValidationFailed, "请填写发放来源和原因")
	}
	if input.TotalAmount <= 0 {
		return GrantEntitlementResp{}, errx.New(errx.CodeValidationFailed, "发放数量必须大于 0")
	}
	result, err := l.store.GrantMerchantEntitlement(ctx, input)
	if err != nil {
		return GrantEntitlementResp{}, err
	}
	return GrantEntitlementResp{ID: result.ID, Message: "权益已发放"}, nil
}
