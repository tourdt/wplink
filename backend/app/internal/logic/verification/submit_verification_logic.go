package verification

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type VerificationStore interface {
	SubmitVerification(ctx context.Context, input model.SubmitVerificationInput) (model.VerificationResult, error)
	GetLatestVerification(ctx context.Context, merchantID string) (model.VerificationBrief, error)
}

type SubmitVerificationReq struct {
	MerchantID       string
	ApplicantUserID  string
	VerificationType string
	BusinessName     string
	LicenseURL       string
	StorefrontURL    string
	Materials        model.JSONMap
}

type SubmitVerificationResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type LatestVerificationResp struct {
	ID               string `json:"id"`
	VerificationType string `json:"verificationType"`
	Status           string `json:"status"`
	ReviewedAt       string `json:"reviewedAt,omitempty"`
}

type SubmitVerificationLogic struct {
	store VerificationStore
}

func NewSubmitVerificationLogic(store VerificationStore) *SubmitVerificationLogic {
	return &SubmitVerificationLogic{store: store}
}

func (l *SubmitVerificationLogic) SubmitVerification(ctx context.Context, req SubmitVerificationReq) (SubmitVerificationResp, error) {
	input := model.SubmitVerificationInput{
		MerchantID:       strings.TrimSpace(req.MerchantID),
		ApplicantUserID:  strings.TrimSpace(req.ApplicantUserID),
		VerificationType: strings.TrimSpace(req.VerificationType),
		BusinessName:     strings.TrimSpace(req.BusinessName),
		LicenseURL:       strings.TrimSpace(req.LicenseURL),
		StorefrontURL:    strings.TrimSpace(req.StorefrontURL),
		Materials:        req.Materials,
	}
	if input.MerchantID == "" || input.ApplicantUserID == "" {
		return SubmitVerificationResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或未登录")
	}
	if !isSupportedVerificationType(input.VerificationType) {
		return SubmitVerificationResp{}, errx.New(errx.CodeValidationFailed, "认证类型不正确")
	}
	result, err := l.store.SubmitVerification(ctx, input)
	if err != nil {
		return SubmitVerificationResp{}, err
	}
	return SubmitVerificationResp{ID: result.ID, Status: result.Status, Message: "认证资料已提交，请等待审核"}, nil
}

type GetLatestVerificationLogic struct {
	store VerificationStore
}

func NewGetLatestVerificationLogic(store VerificationStore) *GetLatestVerificationLogic {
	return &GetLatestVerificationLogic{store: store}
}

func (l *GetLatestVerificationLogic) GetLatestVerification(ctx context.Context, merchantID string) (LatestVerificationResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return LatestVerificationResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	result, err := l.store.GetLatestVerification(ctx, merchantID)
	if errors.Is(err, sql.ErrNoRows) {
		return LatestVerificationResp{Status: "none"}, nil
	}
	if err != nil {
		return LatestVerificationResp{}, err
	}
	return LatestVerificationResp{
		ID: result.ID, VerificationType: result.VerificationType, Status: result.Status, ReviewedAt: result.ReviewedAt,
	}, nil
}

func isSupportedVerificationType(verificationType string) bool {
	switch verificationType {
	case "factory", "stall", "stockist", "service_provider":
		return true
	default:
		return false
	}
}
