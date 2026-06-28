package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type MatchCaseStore interface {
	CreateMatchCase(ctx context.Context, input model.CreateMatchCaseInput) (model.MatchCaseResult, error)
	ListMatchCases(ctx context.Context, filter model.ListMatchCasesFilter) (model.ListMatchCasesResult, error)
	UpdateMatchCaseStatus(ctx context.Context, input model.UpdateMatchCaseStatusInput) (model.MatchCaseResult, error)
	AddMatchCaseResources(ctx context.Context, input model.AddMatchCaseResourcesInput) error
	AddMatchCaseParticipants(ctx context.Context, input model.AddMatchCaseParticipantsInput) error
}

type CreateMatchCaseReq struct {
	PurchaseDemandID       string
	OperatorID             string
	ResourceIDs            []string
	ParticipantMerchantIDs []string
	ResultNote             string
}

type MatchCaseResp struct {
	ID      string `json:"id"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
}

type ListMatchCasesReq struct {
	Status   string
	Page     int64
	PageSize int64
}

type MatchCaseItem struct {
	ID               string `json:"id"`
	PurchaseDemandID string `json:"purchaseDemandId"`
	DemandTitle      string `json:"demandTitle"`
	Status           string `json:"status"`
	Source           string `json:"source"`
	ResultNote       string `json:"resultNote,omitempty"`
	ResourceCount    int64  `json:"resourceCount"`
	ParticipantCount int64  `json:"participantCount"`
	CreatedAt        string `json:"createdAt"`
}

type ListMatchCasesResp struct {
	Items    []MatchCaseItem `json:"items"`
	Page     int64           `json:"page"`
	PageSize int64           `json:"pageSize"`
	Total    int64           `json:"total"`
}

type UpdateMatchCaseStatusReq struct {
	MatchCaseID string
	OperatorID  string
	Status      string
	ResultNote  string
}

type AddMatchCaseResourcesReq struct {
	MatchCaseID string
	OperatorID  string
	ResourceIDs []string
}

type AddMatchCaseParticipantsReq struct {
	MatchCaseID string
	OperatorID  string
	MerchantIDs []string
}

type MatchCaseLogic struct {
	store MatchCaseStore
}

func NewMatchCaseLogic(store MatchCaseStore) *MatchCaseLogic {
	return &MatchCaseLogic{store: store}
}

func (l *MatchCaseLogic) CreateMatchCase(ctx context.Context, req CreateMatchCaseReq) (MatchCaseResp, error) {
	input := model.CreateMatchCaseInput{
		PurchaseDemandID:       strings.TrimSpace(req.PurchaseDemandID),
		OperatorID:             strings.TrimSpace(req.OperatorID),
		ResourceIDs:            trimNonEmptyStrings(req.ResourceIDs),
		ParticipantMerchantIDs: trimNonEmptyStrings(req.ParticipantMerchantIDs),
		ResultNote:             strings.TrimSpace(req.ResultNote),
	}
	if input.PurchaseDemandID == "" {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "请选择采购需求")
	}
	result, err := l.store.CreateMatchCase(ctx, input)
	if err != nil {
		return MatchCaseResp{}, err
	}
	return MatchCaseResp{ID: result.ID, Status: result.Status, Message: "撮合单已创建"}, nil
}

func (l *MatchCaseLogic) ListMatchCases(ctx context.Context, req ListMatchCasesReq) (ListMatchCasesResp, error) {
	result, err := l.store.ListMatchCases(ctx, model.ListMatchCasesFilter{
		Status:   strings.TrimSpace(req.Status),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListMatchCasesResp{}, err
	}
	items := make([]MatchCaseItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MatchCaseItem{
			ID:               item.ID,
			PurchaseDemandID: item.PurchaseDemandID,
			DemandTitle:      item.DemandTitle,
			Status:           item.Status,
			Source:           item.Source,
			ResultNote:       item.ResultNote,
			ResourceCount:    item.ResourceCount,
			ParticipantCount: item.ParticipantCount,
			CreatedAt:        item.CreatedAt,
		})
	}
	return ListMatchCasesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *MatchCaseLogic) UpdateMatchCaseStatus(ctx context.Context, req UpdateMatchCaseStatusReq) (MatchCaseResp, error) {
	input := model.UpdateMatchCaseStatusInput{
		MatchCaseID: strings.TrimSpace(req.MatchCaseID),
		OperatorID:  strings.TrimSpace(req.OperatorID),
		Status:      strings.TrimSpace(req.Status),
		ResultNote:  strings.TrimSpace(req.ResultNote),
	}
	if input.MatchCaseID == "" {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "撮合单不存在")
	}
	if !isSupportedMatchStatus(input.Status) {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "撮合状态不正确")
	}
	if (input.Status == model.MatchCaseStatusSucceeded || input.Status == model.MatchCaseStatusFailed) && input.ResultNote == "" {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "请填写撮合结果说明")
	}
	result, err := l.store.UpdateMatchCaseStatus(ctx, input)
	if err != nil {
		return MatchCaseResp{}, err
	}
	return MatchCaseResp{ID: result.ID, Status: result.Status, Message: "撮合状态已更新"}, nil
}

func (l *MatchCaseLogic) AddMatchCaseResources(ctx context.Context, req AddMatchCaseResourcesReq) (MatchCaseResp, error) {
	input := model.AddMatchCaseResourcesInput{
		MatchCaseID: strings.TrimSpace(req.MatchCaseID),
		OperatorID:  strings.TrimSpace(req.OperatorID),
		ResourceIDs: trimNonEmptyStrings(req.ResourceIDs),
	}
	if input.MatchCaseID == "" || len(input.ResourceIDs) == 0 {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "请选择撮合单和候选资源")
	}
	if err := l.store.AddMatchCaseResources(ctx, input); err != nil {
		return MatchCaseResp{}, err
	}
	return MatchCaseResp{ID: input.MatchCaseID, Message: "候选资源已加入"}, nil
}

func (l *MatchCaseLogic) AddMatchCaseParticipants(ctx context.Context, req AddMatchCaseParticipantsReq) (MatchCaseResp, error) {
	input := model.AddMatchCaseParticipantsInput{
		MatchCaseID: strings.TrimSpace(req.MatchCaseID),
		OperatorID:  strings.TrimSpace(req.OperatorID),
		MerchantIDs: trimNonEmptyStrings(req.MerchantIDs),
	}
	if input.MatchCaseID == "" || len(input.MerchantIDs) == 0 {
		return MatchCaseResp{}, errx.New(errx.CodeValidationFailed, "请选择撮合单和参与商家")
	}
	if err := l.store.AddMatchCaseParticipants(ctx, input); err != nil {
		return MatchCaseResp{}, err
	}
	return MatchCaseResp{ID: input.MatchCaseID, Message: "参与商家已加入"}, nil
}

func isSupportedMatchStatus(status string) bool {
	switch status {
	case model.MatchCaseStatusOpen, model.MatchCaseStatusContacted, model.MatchCaseStatusSucceeded, model.MatchCaseStatusFailed, model.MatchCaseStatusClosed:
		return true
	default:
		return false
	}
}

func trimNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
