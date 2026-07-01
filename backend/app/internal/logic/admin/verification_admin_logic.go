package admin

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type VerificationAdminStore interface {
	ListPendingVerifications(ctx context.Context, filter model.PendingVerificationsFilter) (model.ListPendingVerificationsResult, error)
	ReviewVerification(ctx context.Context, input model.ReviewVerificationInput) (model.ReviewVerificationResult, error)
}

type VerificationBillingForReviewStore interface {
	GetVerificationBillingConfigForVerification(ctx context.Context, verificationID string) (model.VerificationBillingConfig, error)
}

type ListPendingVerificationsReq struct {
	Page     int64
	PageSize int64
}

type PendingVerificationItem struct {
	ID               string `json:"id"`
	MerchantID       string `json:"merchantId"`
	MerchantName     string `json:"merchantName"`
	VerificationType string `json:"verificationType"`
	Status           string `json:"status"`
	SubmittedAt      string `json:"submittedAt"`
}

type ListPendingVerificationsResp struct {
	Items    []PendingVerificationItem `json:"items"`
	Page     int64                     `json:"page"`
	PageSize int64                     `json:"pageSize"`
	Total    int64                     `json:"total"`
}

type ReviewVerificationReq struct {
	VerificationID string
	ReviewerID     string
	Action         string
	ReviewNote     string
}

type ReviewVerificationResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type VerificationAdminLogic struct {
	store VerificationAdminStore
}

func NewVerificationAdminLogic(store VerificationAdminStore) *VerificationAdminLogic {
	return &VerificationAdminLogic{store: store}
}

func (l *VerificationAdminLogic) ListPendingVerifications(ctx context.Context, req ListPendingVerificationsReq) (ListPendingVerificationsResp, error) {
	result, err := l.store.ListPendingVerifications(ctx, model.PendingVerificationsFilter{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return ListPendingVerificationsResp{}, err
	}
	items := make([]PendingVerificationItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, PendingVerificationItem{
			ID: item.ID, MerchantID: item.MerchantID, MerchantName: item.MerchantName,
			VerificationType: item.VerificationType, Status: item.Status, SubmittedAt: item.SubmittedAt,
		})
	}
	return ListPendingVerificationsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *VerificationAdminLogic) ReviewVerification(ctx context.Context, req ReviewVerificationReq) (ReviewVerificationResp, error) {
	input := model.ReviewVerificationInput{
		VerificationID: strings.TrimSpace(req.VerificationID),
		ReviewerID:     strings.TrimSpace(req.ReviewerID),
		Action:         strings.TrimSpace(req.Action),
		ReviewNote:     strings.TrimSpace(req.ReviewNote),
	}
	if input.VerificationID == "" || input.ReviewerID == "" {
		return ReviewVerificationResp{}, errx.New(errx.CodeValidationFailed, "认证记录不存在")
	}
	if input.Action != "approve" && input.Action != "reject" && input.Action != "revoke" {
		return ReviewVerificationResp{}, errx.New(errx.CodeValidationFailed, "审核动作不正确")
	}
	if (input.Action == "reject" || input.Action == "revoke") && input.ReviewNote == "" {
		return ReviewVerificationResp{}, errx.New(errx.CodeValidationFailed, "请填写处理原因")
	}
	if input.Action == "approve" {
		if billingStore, ok := l.store.(VerificationBillingForReviewStore); ok {
			billingConfig, err := billingStore.GetVerificationBillingConfigForVerification(ctx, input.VerificationID)
			if err != nil {
				return ReviewVerificationResp{}, err
			}
			// 开启认证收费且当前不处于限免窗口时，审核通过仅代表资料合格，认证需等线上支付成功后生效。
			input.RequirePayment = billingConfig.RequiresOnlinePayment(time.Now())
		}
	}
	result, err := l.store.ReviewVerification(ctx, input)
	if err != nil {
		return ReviewVerificationResp{}, err
	}
	return ReviewVerificationResp{ID: result.ID, Status: result.Status, Message: "认证审核已处理"}, nil
}
