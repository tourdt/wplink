package vip

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type Store interface {
	ListVIPPlans(ctx context.Context) ([]model.VIPPlan, error)
	ListQuotaPacks(ctx context.Context) ([]model.QuotaPack, error)
	GetMerchantVIPSummary(ctx context.Context, merchantID string) (model.MerchantVIPSummary, error)
	CreateVIPOrder(ctx context.Context, input model.CreateVIPOrderInput) (model.VIPOrder, error)
	CreateQuotaPackOrder(ctx context.Context, input model.CreateQuotaPackOrderInput) (model.VIPOrder, error)
}

type VIPBenefitInfo struct {
	PublishPolicy      string `json:"publishPolicy"`
	PublishQuota       int64  `json:"publishQuota"`
	RefreshQuota       int64  `json:"refreshQuota"`
	TopVoucherCount    int64  `json:"topVoucherCount"`
	TopDurationHours   int64  `json:"topDurationHours"`
	HomepageImageLimit int64  `json:"homepageImageLimit,omitempty"`
}

type VIPPlanInfo struct {
	Code              string         `json:"code"`
	Name              string         `json:"name"`
	DurationMonths    int64          `json:"durationMonths"`
	StandardPriceCent int64          `json:"standardPriceCent"`
	SalePriceCent     int64          `json:"salePriceCent,omitempty"`
	SaleLabel         string         `json:"saleLabel,omitempty"`
	Benefits          VIPBenefitInfo `json:"benefits"`
}

type ListVIPPlansResp struct {
	Items []VIPPlanInfo `json:"items"`
}

type QuotaPackInfo struct {
	Code              string         `json:"code"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	StandardPriceCent int64          `json:"standardPriceCent"`
	SalePriceCent     int64          `json:"salePriceCent,omitempty"`
	SaleLabel         string         `json:"saleLabel,omitempty"`
	Benefits          VIPBenefitInfo `json:"benefits"`
}

type ListQuotaPacksResp struct {
	Items []QuotaPackInfo `json:"items"`
}

type MerchantVIPResp struct {
	MerchantID            string `json:"merchantId"`
	Status                string `json:"status"`
	PlanCode              string `json:"planCode,omitempty"`
	PlanName              string `json:"planName,omitempty"`
	StartsAt              string `json:"startsAt,omitempty"`
	ExpiresAt             string `json:"expiresAt,omitempty"`
	PublishQuotaRemaining int64  `json:"publishQuotaRemaining"`
	RefreshQuotaRemaining int64  `json:"refreshQuotaRemaining"`
	TopVoucherCount       int64  `json:"topVoucherCount"`
}

type CreateVIPOrderReq struct {
	MerchantID  string
	UserID      string
	ProductType string `json:"productType,omitempty"`
	ProductCode string `json:"productCode,omitempty"`
	PlanCode    string `json:"planCode,omitempty"`
	ResourceID  string `json:"resourceId,omitempty"`
}

type CreateVIPOrderResp struct {
	OrderID           string         `json:"orderId"`
	Status            string         `json:"status"`
	ProductType       string         `json:"productType,omitempty"`
	ProductCode       string         `json:"productCode,omitempty"`
	ProductName       string         `json:"productName,omitempty"`
	PlanCode          string         `json:"planCode,omitempty"`
	PlanName          string         `json:"planName,omitempty"`
	StandardPriceCent int64          `json:"standardPriceCent"`
	ActualPriceCent   int64          `json:"actualPriceCent"`
	PromotionCode     string         `json:"promotionCode,omitempty"`
	Benefits          VIPBenefitInfo `json:"benefits"`
}

type ListVIPPlansLogic struct {
	store Store
}

func NewListVIPPlansLogic(store Store) *ListVIPPlansLogic {
	return &ListVIPPlansLogic{store: store}
}

func (l *ListVIPPlansLogic) ListVIPPlans(ctx context.Context) (ListVIPPlansResp, error) {
	plans, err := l.store.ListVIPPlans(ctx)
	if err != nil {
		logx.Errorf("查询 VIP 套餐失败: err=%+v", err)
		return ListVIPPlansResp{}, err
	}
	items := make([]VIPPlanInfo, 0, len(plans))
	for _, plan := range plans {
		items = append(items, VIPPlanInfo{
			Code:              plan.Code,
			Name:              plan.Name,
			DurationMonths:    plan.DurationMonths,
			StandardPriceCent: plan.StandardPriceCent,
			SalePriceCent:     plan.SalePriceCent,
			SaleLabel:         plan.SaleLabel,
			Benefits:          mapVIPBenefit(plan.Benefits),
		})
	}
	return ListVIPPlansResp{Items: items}, nil
}

type ListQuotaPacksLogic struct {
	store Store
}

func NewListQuotaPacksLogic(store Store) *ListQuotaPacksLogic {
	return &ListQuotaPacksLogic{store: store}
}

func (l *ListQuotaPacksLogic) ListQuotaPacks(ctx context.Context) (ListQuotaPacksResp, error) {
	packs, err := l.store.ListQuotaPacks(ctx)
	if err != nil {
		logx.Errorf("查询次数包商品失败: err=%+v", err)
		return ListQuotaPacksResp{}, err
	}
	items := make([]QuotaPackInfo, 0, len(packs))
	for _, pack := range packs {
		items = append(items, QuotaPackInfo{
			Code:              pack.Code,
			Name:              pack.Name,
			Description:       pack.Description,
			StandardPriceCent: pack.StandardPriceCent,
			SalePriceCent:     pack.SalePriceCent,
			SaleLabel:         pack.SaleLabel,
			Benefits:          mapVIPBenefit(pack.Benefits),
		})
	}
	return ListQuotaPacksResp{Items: items}, nil
}

type GetMerchantVIPLogic struct {
	store Store
}

func NewGetMerchantVIPLogic(store Store) *GetMerchantVIPLogic {
	return &GetMerchantVIPLogic{store: store}
}

func (l *GetMerchantVIPLogic) GetMerchantVIP(ctx context.Context, merchantID string) (MerchantVIPResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantVIPResp{}, errx.New(errx.CodeMerchantNotFound, "商家不存在")
	}
	summary, err := l.store.GetMerchantVIPSummary(ctx, merchantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantVIPResp{}, errx.New(errx.CodeMerchantNotFound, "商家不存在")
		}
		logx.Errorf("查询商家 VIP 状态失败: merchantId=%s err=%+v", merchantID, err)
		return MerchantVIPResp{}, err
	}
	return MerchantVIPResp{
		MerchantID:            summary.MerchantID,
		Status:                summary.Status,
		PlanCode:              summary.PlanCode,
		PlanName:              summary.PlanName,
		StartsAt:              summary.StartsAt,
		ExpiresAt:             summary.ExpiresAt,
		PublishQuotaRemaining: summary.PublishQuotaRemaining,
		RefreshQuotaRemaining: summary.RefreshQuotaRemaining,
		TopVoucherCount:       summary.TopVoucherCount,
	}, nil
}

type CreateVIPOrderLogic struct {
	store Store
}

func NewCreateVIPOrderLogic(store Store) *CreateVIPOrderLogic {
	return &CreateVIPOrderLogic{store: store}
}

func (l *CreateVIPOrderLogic) CreateVIPOrder(ctx context.Context, req CreateVIPOrderReq) (CreateVIPOrderResp, error) {
	merchantID := strings.TrimSpace(req.MerchantID)
	userID := strings.TrimSpace(req.UserID)
	productType := strings.TrimSpace(req.ProductType)
	if productType == "" {
		productType = model.VIPProductTypeVIPPlan
	}
	if merchantID == "" {
		return CreateVIPOrderResp{}, errx.New(errx.CodeMerchantNotFound, "商家不存在")
	}
	if userID == "" {
		return CreateVIPOrderResp{}, errx.New(errx.CodeUnauthorized, "请先登录后再购买权益")
	}
	if productType == model.VIPProductTypeQuotaPack {
		return l.createQuotaPackOrder(ctx, merchantID, userID, req)
	}
	if productType != model.VIPProductTypeVIPPlan {
		return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的权益商品")
	}
	return l.createVIPPlanOrder(ctx, merchantID, userID, req)
}

func (l *CreateVIPOrderLogic) createVIPPlanOrder(ctx context.Context, merchantID string, userID string, req CreateVIPOrderReq) (CreateVIPOrderResp, error) {
	input := model.CreateVIPOrderInput{
		MerchantID: merchantID,
		UserID:     userID,
		PlanCode:   strings.TrimSpace(req.PlanCode),
	}
	if input.PlanCode == "" {
		return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的 VIP 套餐")
	}
	order, err := l.store.CreateVIPOrder(ctx, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("创建 VIP 订单被拦截: merchantId=%s userId=%s planCode=%s reason=invalid_plan", input.MerchantID, input.UserID, input.PlanCode)
			return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的 VIP 套餐")
		}
		logx.Errorf("创建 VIP 订单失败: merchantId=%s userId=%s planCode=%s err=%+v", input.MerchantID, input.UserID, input.PlanCode, err)
		return CreateVIPOrderResp{}, err
	}
	logx.Infof("创建 VIP 订单成功: merchantId=%s userId=%s planCode=%s orderId=%s actualPriceCent=%d", input.MerchantID, input.UserID, input.PlanCode, order.ID, order.ActualPriceCent)
	return mapOrderResp(order), nil
}

func (l *CreateVIPOrderLogic) createQuotaPackOrder(ctx context.Context, merchantID string, userID string, req CreateVIPOrderReq) (CreateVIPOrderResp, error) {
	packCode := strings.TrimSpace(req.ProductCode)
	if packCode == "" {
		packCode = strings.TrimSpace(req.PlanCode)
	}
	input := model.CreateQuotaPackOrderInput{
		MerchantID: merchantID,
		UserID:     userID,
		PackCode:   packCode,
		ResourceID: strings.TrimSpace(req.ResourceID),
	}
	if input.PackCode == "" {
		return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的次数包")
	}
	order, err := l.store.CreateQuotaPackOrder(ctx, input)
	if err != nil {
		if errors.Is(err, model.ErrTopServiceResourceRequired) {
			logx.Infof("创建置顶服务订单被拦截: merchantId=%s userId=%s packCode=%s reason=missing_resource", input.MerchantID, input.UserID, input.PackCode)
			return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择要置顶的供需信息")
		}
		if errors.Is(err, model.ErrTopServiceResourceInvalid) {
			logx.Infof("创建置顶服务订单被拦截: merchantId=%s userId=%s packCode=%s resourceId=%s reason=resource_not_topable", input.MerchantID, input.UserID, input.PackCode, input.ResourceID)
			return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "资源当前不可置顶，请刷新后重试")
		}
		if errors.Is(err, model.ErrTopServiceProductInvalid) {
			logx.Infof("创建置顶服务订单被拦截: merchantId=%s userId=%s packCode=%s resourceId=%s reason=invalid_top_service_product", input.MerchantID, input.UserID, input.PackCode, input.ResourceID)
			return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择单次置顶服务")
		}
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("创建次数包订单被拦截: merchantId=%s userId=%s packCode=%s reason=invalid_pack", input.MerchantID, input.UserID, input.PackCode)
			return CreateVIPOrderResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的次数包")
		}
		logx.Errorf("创建次数包订单失败: merchantId=%s userId=%s packCode=%s resourceId=%s err=%+v", input.MerchantID, input.UserID, input.PackCode, input.ResourceID, err)
		return CreateVIPOrderResp{}, err
	}
	logx.Infof("创建次数包订单成功: merchantId=%s userId=%s packCode=%s resourceId=%s orderId=%s actualPriceCent=%d", input.MerchantID, input.UserID, input.PackCode, input.ResourceID, order.ID, order.ActualPriceCent)
	return mapOrderResp(order), nil
}

func mapOrderResp(order model.VIPOrder) CreateVIPOrderResp {
	return CreateVIPOrderResp{
		OrderID:           order.ID,
		Status:            order.Status,
		ProductType:       order.ProductType,
		ProductCode:       order.ProductCode,
		ProductName:       order.ProductName,
		PlanCode:          order.PlanCode,
		PlanName:          order.PlanName,
		StandardPriceCent: order.StandardPriceCent,
		ActualPriceCent:   order.ActualPriceCent,
		PromotionCode:     order.PromotionCode,
		Benefits:          mapVIPBenefit(order.Benefits),
	}
}

func mapVIPBenefit(snapshot model.VIPBenefitSnapshot) VIPBenefitInfo {
	policy := strings.TrimSpace(snapshot.PublishPolicy)
	if policy == "" {
		policy = model.VIPPublishPolicyQuota
	}
	return VIPBenefitInfo{
		PublishPolicy:      policy,
		PublishQuota:       snapshot.PublishQuota,
		RefreshQuota:       snapshot.RefreshQuota,
		TopVoucherCount:    snapshot.TopVoucherCount,
		TopDurationHours:   snapshot.TopDurationHours,
		HomepageImageLimit: snapshot.HomepageImageLimit,
	}
}
