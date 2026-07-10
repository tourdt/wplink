package admin

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type VIPConfigAdminStore interface {
	ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error)
	SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error)
	ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error)
	SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error)
	ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error)
	SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error)
}

type VIPConfigAdminLogic struct {
	store VIPConfigAdminStore
}

func NewVIPConfigAdminLogic(store VIPConfigAdminStore) *VIPConfigAdminLogic {
	return &VIPConfigAdminLogic{store: store}
}

type VIPBenefitConfig struct {
	PublishPolicy      string `json:"publishPolicy,omitempty"`
	PublishQuota       int64  `json:"publishQuota"`
	RefreshQuota       int64  `json:"refreshQuota"`
	TopVoucherCount    int64  `json:"topVoucherCount"`
	TopDurationHours   int64  `json:"topDurationHours"`
	HomepageImageLimit int64  `json:"homepageImageLimit,omitempty"`
}

type VIPPlanConfigItem struct {
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	DurationMonths    int64            `json:"durationMonths"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	Status            string           `json:"status"`
	DisplayOrder      int64            `json:"displayOrder"`
	Benefits          VIPBenefitConfig `json:"benefits"`
	UpdatedAt         string           `json:"updatedAt,omitempty"`
}

type AdminListVIPPlansResp struct {
	Items []VIPPlanConfigItem `json:"items"`
}

type SaveVIPPlanConfigReq struct {
	Code              string           `json:"code,omitempty"`
	Name              string           `json:"name"`
	DurationMonths    int64            `json:"durationMonths"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	Status            string           `json:"status,omitempty"`
	DisplayOrder      int64            `json:"displayOrder,omitempty"`
	Benefits          VIPBenefitConfig `json:"benefits"`
}

type QuotaPackConfigItem struct {
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	SalePriceCent     int64            `json:"salePriceCent,omitempty"`
	SaleLabel         string           `json:"saleLabel,omitempty"`
	Status            string           `json:"status"`
	DisplayOrder      int64            `json:"displayOrder"`
	Benefits          VIPBenefitConfig `json:"benefits"`
	UpdatedAt         string           `json:"updatedAt,omitempty"`
}

type AdminListQuotaPacksResp struct {
	Items []QuotaPackConfigItem `json:"items"`
}

type SaveQuotaPackConfigReq struct {
	Code              string           `json:"code,omitempty"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	StandardPriceCent int64            `json:"standardPriceCent"`
	SalePriceCent     int64            `json:"salePriceCent,omitempty"`
	SaleLabel         string           `json:"saleLabel,omitempty"`
	Status            string           `json:"status,omitempty"`
	DisplayOrder      int64            `json:"displayOrder,omitempty"`
	Benefits          VIPBenefitConfig `json:"benefits"`
}

type VIPPromotionConfigItem struct {
	Code          string `json:"code"`
	PlanCode      string `json:"planCode"`
	PlanName      string `json:"planName,omitempty"`
	PromotionType string `json:"promotionType"`
	SalePriceCent int64  `json:"salePriceCent"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt,omitempty"`
	QuotaLimit    int64  `json:"quotaLimit,omitempty"`
	UsedCount     int64  `json:"usedCount"`
	Status        string `json:"status"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type AdminListVIPPromotionsResp struct {
	Items []VIPPromotionConfigItem `json:"items"`
}

type SaveVIPPromotionConfigReq struct {
	Code          string `json:"code,omitempty"`
	PlanCode      string `json:"planCode"`
	PromotionType string `json:"promotionType"`
	SalePriceCent int64  `json:"salePriceCent"`
	StartsAt      string `json:"startsAt,omitempty"`
	EndsAt        string `json:"endsAt,omitempty"`
	QuotaLimit    int64  `json:"quotaLimit,omitempty"`
	Status        string `json:"status,omitempty"`
}

type SaveVIPConfigResp struct {
	Code      string `json:"code"`
	UpdatedAt string `json:"updatedAt"`
}

func (l *VIPConfigAdminLogic) ListVIPPlans(ctx context.Context) (AdminListVIPPlansResp, error) {
	plans, err := l.store.ListAdminVIPPlans(ctx)
	if err != nil {
		logx.Errorf("查询后台 VIP 套餐配置失败: err=%+v", err)
		return AdminListVIPPlansResp{}, err
	}
	items := make([]VIPPlanConfigItem, 0, len(plans))
	for _, plan := range plans {
		items = append(items, VIPPlanConfigItem{
			Code:              plan.Code,
			Name:              plan.Name,
			DurationMonths:    plan.DurationMonths,
			StandardPriceCent: plan.StandardPriceCent,
			Status:            plan.Status,
			DisplayOrder:      plan.DisplayOrder,
			Benefits:          mapVIPBenefitConfig(plan.Benefits),
			UpdatedAt:         plan.UpdatedAt,
		})
	}
	return AdminListVIPPlansResp{Items: items}, nil
}

func (l *VIPConfigAdminLogic) SaveVIPPlan(ctx context.Context, pathCode string, req SaveVIPPlanConfigReq, operatorID string) (SaveVIPConfigResp, error) {
	input := model.SaveAdminVIPPlanInput{
		Code:              configCode(pathCode, req.Code),
		Name:              strings.TrimSpace(req.Name),
		DurationMonths:    req.DurationMonths,
		StandardPriceCent: req.StandardPriceCent,
		Status:            normalizeActiveInactive(req.Status),
		DisplayOrder:      req.DisplayOrder,
		Benefits:          mapAdminVIPBenefits(req.Benefits),
		OperatorID:        strings.TrimSpace(operatorID),
	}
	if err := validateVIPPlanInput(input); err != nil {
		return SaveVIPConfigResp{}, err
	}
	result, err := l.store.SaveAdminVIPPlan(ctx, input)
	if err != nil {
		logx.Errorf("保存 VIP 套餐配置失败: operatorId=%s planCode=%s err=%+v", input.OperatorID, input.Code, err)
		return SaveVIPConfigResp{}, errx.New(errx.CodeInternalError, "保存失败，请稍后重试")
	}
	return SaveVIPConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}

func (l *VIPConfigAdminLogic) ListQuotaPacks(ctx context.Context) (AdminListQuotaPacksResp, error) {
	packs, err := l.store.ListAdminQuotaPacks(ctx)
	if err != nil {
		logx.Errorf("查询后台次数包配置失败: err=%+v", err)
		return AdminListQuotaPacksResp{}, err
	}
	items := make([]QuotaPackConfigItem, 0, len(packs))
	for _, pack := range packs {
		items = append(items, QuotaPackConfigItem{
			Code:              pack.Code,
			Name:              pack.Name,
			Description:       pack.Description,
			StandardPriceCent: pack.StandardPriceCent,
			SalePriceCent:     pack.SalePriceCent,
			SaleLabel:         pack.SaleLabel,
			Status:            pack.Status,
			DisplayOrder:      pack.DisplayOrder,
			Benefits:          mapVIPBenefitConfig(pack.Benefits),
			UpdatedAt:         pack.UpdatedAt,
		})
	}
	return AdminListQuotaPacksResp{Items: items}, nil
}

func (l *VIPConfigAdminLogic) SaveQuotaPack(ctx context.Context, pathCode string, req SaveQuotaPackConfigReq, operatorID string) (SaveVIPConfigResp, error) {
	input := model.SaveAdminQuotaPackInput{
		Code:              configCode(pathCode, req.Code),
		Name:              strings.TrimSpace(req.Name),
		Description:       strings.TrimSpace(req.Description),
		StandardPriceCent: req.StandardPriceCent,
		SalePriceCent:     req.SalePriceCent,
		SaleLabel:         strings.TrimSpace(req.SaleLabel),
		Status:            normalizeActiveInactive(req.Status),
		DisplayOrder:      req.DisplayOrder,
		Benefits:          mapAdminVIPBenefits(req.Benefits),
		OperatorID:        strings.TrimSpace(operatorID),
	}
	if err := validateQuotaPackInput(input); err != nil {
		return SaveVIPConfigResp{}, err
	}
	result, err := l.store.SaveAdminQuotaPack(ctx, input)
	if err != nil {
		logx.Errorf("保存次数包配置失败: operatorId=%s packCode=%s err=%+v", input.OperatorID, input.Code, err)
		return SaveVIPConfigResp{}, errx.New(errx.CodeInternalError, "保存失败，请稍后重试")
	}
	return SaveVIPConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}

func (l *VIPConfigAdminLogic) ListVIPPromotions(ctx context.Context) (AdminListVIPPromotionsResp, error) {
	promotions, err := l.store.ListAdminVIPPromotions(ctx)
	if err != nil {
		logx.Errorf("查询 VIP 优惠配置失败: err=%+v", err)
		return AdminListVIPPromotionsResp{}, err
	}
	items := make([]VIPPromotionConfigItem, 0, len(promotions))
	for _, promotion := range promotions {
		items = append(items, VIPPromotionConfigItem{
			Code:          promotion.Code,
			PlanCode:      promotion.PlanCode,
			PlanName:      promotion.PlanName,
			PromotionType: promotion.PromotionType,
			SalePriceCent: promotion.SalePriceCent,
			StartsAt:      promotion.StartsAt,
			EndsAt:        promotion.EndsAt,
			QuotaLimit:    promotion.QuotaLimit,
			UsedCount:     promotion.UsedCount,
			Status:        promotion.Status,
			UpdatedAt:     promotion.UpdatedAt,
		})
	}
	return AdminListVIPPromotionsResp{Items: items}, nil
}

func (l *VIPConfigAdminLogic) SaveVIPPromotion(ctx context.Context, pathCode string, req SaveVIPPromotionConfigReq, operatorID string) (SaveVIPConfigResp, error) {
	input := model.SaveAdminVIPPromotionInput{
		Code:          configCode(pathCode, req.Code),
		PlanCode:      strings.TrimSpace(req.PlanCode),
		PromotionType: strings.TrimSpace(req.PromotionType),
		SalePriceCent: req.SalePriceCent,
		StartsAt:      strings.TrimSpace(req.StartsAt),
		EndsAt:        strings.TrimSpace(req.EndsAt),
		QuotaLimit:    req.QuotaLimit,
		Status:        normalizeActiveInactive(req.Status),
		OperatorID:    strings.TrimSpace(operatorID),
	}
	if err := validateVIPPromotionInput(input); err != nil {
		return SaveVIPConfigResp{}, err
	}
	result, err := l.store.SaveAdminVIPPromotion(ctx, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SaveVIPConfigResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的 VIP 套餐")
		}
		logx.Errorf("保存 VIP 优惠配置失败: operatorId=%s promotionCode=%s planCode=%s err=%+v", input.OperatorID, input.Code, input.PlanCode, err)
		return SaveVIPConfigResp{}, errx.New(errx.CodeInternalError, "保存失败，请稍后重试")
	}
	return SaveVIPConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}

func configCode(pathCode string, bodyCode string) string {
	if trimmed := strings.TrimSpace(pathCode); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(bodyCode)
}

func normalizeActiveInactive(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "active"
	}
	return status
}

func mapAdminVIPBenefits(req VIPBenefitConfig) model.VIPBenefitSnapshot {
	policy := strings.TrimSpace(req.PublishPolicy)
	if policy == "" {
		policy = model.VIPPublishPolicyQuota
	}
	return model.VIPBenefitSnapshot{
		PublishPolicy:      policy,
		PublishQuota:       req.PublishQuota,
		RefreshQuota:       req.RefreshQuota,
		TopVoucherCount:    req.TopVoucherCount,
		TopDurationHours:   req.TopDurationHours,
		HomepageImageLimit: req.HomepageImageLimit,
	}
}

func mapVIPBenefitConfig(snapshot model.VIPBenefitSnapshot) VIPBenefitConfig {
	policy := strings.TrimSpace(snapshot.PublishPolicy)
	if policy == "" {
		policy = model.VIPPublishPolicyQuota
	}
	return VIPBenefitConfig{
		PublishPolicy:      policy,
		PublishQuota:       snapshot.PublishQuota,
		RefreshQuota:       snapshot.RefreshQuota,
		TopVoucherCount:    snapshot.TopVoucherCount,
		TopDurationHours:   snapshot.TopDurationHours,
		HomepageImageLimit: snapshot.HomepageImageLimit,
	}
}

func validateVIPPlanInput(input model.SaveAdminVIPPlanInput) error {
	if input.Code == "" {
		return errx.New(errx.CodeValidationFailed, "请填写配置编码")
	}
	if input.Name == "" {
		return errx.New(errx.CodeValidationFailed, "请填写配置名称")
	}
	if input.DurationMonths <= 0 {
		return errx.New(errx.CodeValidationFailed, "套餐周期必须大于 0")
	}
	if input.StandardPriceCent < 0 {
		return errx.New(errx.CodeValidationFailed, "价格不能小于 0")
	}
	if err := validateActiveInactiveStatus(input.Status); err != nil {
		return err
	}
	return validateVIPBenefits(input.Benefits)
}

func validateQuotaPackInput(input model.SaveAdminQuotaPackInput) error {
	if input.Code == "" {
		return errx.New(errx.CodeValidationFailed, "请填写配置编码")
	}
	if input.Name == "" {
		return errx.New(errx.CodeValidationFailed, "请填写配置名称")
	}
	if input.StandardPriceCent < 0 || input.SalePriceCent < 0 {
		return errx.New(errx.CodeValidationFailed, "价格不能小于 0")
	}
	if input.SalePriceCent > 0 && input.SalePriceCent > input.StandardPriceCent {
		return errx.New(errx.CodeValidationFailed, "优惠价不能高于标准价")
	}
	if err := validateActiveInactiveStatus(input.Status); err != nil {
		return err
	}
	return validateVIPBenefits(input.Benefits)
}

func validateVIPPromotionInput(input model.SaveAdminVIPPromotionInput) error {
	if input.Code == "" {
		return errx.New(errx.CodeValidationFailed, "请填写配置编码")
	}
	if input.PlanCode == "" {
		return errx.New(errx.CodeValidationFailed, "请选择有效的 VIP 套餐")
	}
	if input.PromotionType != "first_purchase" && input.PromotionType != "launch" {
		return errx.New(errx.CodeValidationFailed, "优惠类型不正确")
	}
	if input.SalePriceCent < 0 {
		return errx.New(errx.CodeValidationFailed, "价格不能小于 0")
	}
	if input.QuotaLimit < 0 {
		return errx.New(errx.CodeValidationFailed, "优惠名额不能小于 0")
	}
	if err := validateActiveInactiveStatus(input.Status); err != nil {
		return err
	}
	return validatePromotionPeriod(input.StartsAt, input.EndsAt)
}

func validateVIPBenefits(benefits model.VIPBenefitSnapshot) error {
	if benefits.PublishPolicy != model.VIPPublishPolicyQuota {
		return errx.New(errx.CodeValidationFailed, "发布权益策略不正确")
	}
	if benefits.PublishQuota < 0 || benefits.RefreshQuota < 0 || benefits.TopVoucherCount < 0 || benefits.TopDurationHours < 0 || benefits.HomepageImageLimit < 0 {
		return errx.New(errx.CodeValidationFailed, "权益数量不能小于 0")
	}
	if benefits.TopVoucherCount > 0 && benefits.TopDurationHours <= 0 {
		return errx.New(errx.CodeValidationFailed, "配置置顶券时请填写置顶时长")
	}
	return nil
}

func validateActiveInactiveStatus(status string) error {
	if status != "active" && status != "inactive" {
		return errx.New(errx.CodeValidationFailed, "配置状态不正确")
	}
	return nil
}

func validatePromotionPeriod(startsAt string, endsAt string) error {
	if strings.TrimSpace(startsAt) == "" || strings.TrimSpace(endsAt) == "" {
		return nil
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(startsAt))
	if err != nil {
		return errx.New(errx.CodeValidationFailed, "开始时间格式不正确")
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(endsAt))
	if err != nil {
		return errx.New(errx.CodeValidationFailed, "结束时间格式不正确")
	}
	if !end.After(start) {
		return errx.New(errx.CodeValidationFailed, "结束时间必须晚于开始时间")
	}
	return nil
}
