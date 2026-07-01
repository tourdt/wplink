package verification

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

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
	ID               string        `json:"id"`
	VerificationType string        `json:"verificationType"`
	Status           string        `json:"status"`
	ReviewedAt       string        `json:"reviewedAt,omitempty"`
	ExpiresAt        string        `json:"expiresAt,omitempty"`
	ReviewNote       string        `json:"reviewNote,omitempty"`
	BusinessName     string        `json:"businessName,omitempty"`
	LicenseURL       string        `json:"licenseUrl,omitempty"`
	StorefrontURL    string        `json:"storefrontUrl,omitempty"`
	Materials        model.JSONMap `json:"materials,omitempty"`
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
	latest, err := l.store.GetLatestVerification(ctx, input.MerchantID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logx.Errorf("查询最新认证记录失败: merchantId=%s err=%+v", input.MerchantID, err)
		return SubmitVerificationResp{}, errx.New(errx.CodeInternalError, "认证状态查询失败，请稍后重试")
	}
	if err == nil {
		// 审核中和待支付都属于同一轮认证未结束，继续提交会覆盖支付入口或制造重复审核单。
		switch latest.Status {
		case model.VerificationStatusPending:
			logx.Infof("认证资料重复提交已拦截: merchantId=%s verificationId=%s status=%s", input.MerchantID, latest.ID, latest.Status)
			return SubmitVerificationResp{}, errx.New(errx.CodeStateConflict, "认证资料正在审核中，请等待审核结果")
		case model.VerificationStatusPaymentPending:
			logx.Infof("认证待支付重复提交已拦截: merchantId=%s verificationId=%s status=%s", input.MerchantID, latest.ID, latest.Status)
			return SubmitVerificationResp{}, errx.New(errx.CodeStateConflict, "认证资料已审核通过，请先完成认证费支付")
		}
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
		ID: result.ID, VerificationType: result.VerificationType, Status: result.Status, ReviewedAt: result.ReviewedAt, ExpiresAt: result.ExpiresAt,
		ReviewNote: result.ReviewNote, BusinessName: result.BusinessName, LicenseURL: result.LicenseURL, StorefrontURL: result.StorefrontURL,
		Materials: cloneJSONMap(result.Materials),
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

func cloneJSONMap(value model.JSONMap) model.JSONMap {
	if value == nil {
		return nil
	}
	cloned := make(model.JSONMap, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
