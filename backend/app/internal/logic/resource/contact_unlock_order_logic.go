package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContactUnlockOrderStore interface {
	CreateContactUnlockOrder(ctx context.Context, input model.CreateContactUnlockOrderInput) (model.ContactUnlockOrderResult, error)
}

type CreateContactUnlockOrderReq struct {
	ResourceID       string
	UserID           string
	ViewerMerchantID string
	Action           string
}

type CreateContactUnlockOrderResp struct {
	OrderID         string `json:"orderId,omitempty"`
	Status          string `json:"status"`
	AlreadyUnlocked bool   `json:"alreadyUnlocked"`
	PriceCent       int64  `json:"priceCent"`
	Currency        string `json:"currency"`
	Message         string `json:"message"`
}

type CreateContactUnlockOrderLogic struct {
	store ContactUnlockOrderStore
}

func NewCreateContactUnlockOrderLogic(store ContactUnlockOrderStore) *CreateContactUnlockOrderLogic {
	return &CreateContactUnlockOrderLogic{store: store}
}

func (l *CreateContactUnlockOrderLogic) CreateContactUnlockOrder(ctx context.Context, req CreateContactUnlockOrderReq) (CreateContactUnlockOrderResp, error) {
	resourceID := strings.TrimSpace(req.ResourceID)
	userID := strings.TrimSpace(req.UserID)
	action := strings.TrimSpace(req.Action)
	if resourceID == "" {
		return CreateContactUnlockOrderResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if userID == "" {
		return CreateContactUnlockOrderResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看联系方式")
	}
	if action != "" && action != "phone" && action != "wechat" {
		return CreateContactUnlockOrderResp{}, errx.New(errx.CodeValidationFailed, "联系动作不正确")
	}
	result, err := l.store.CreateContactUnlockOrder(ctx, model.CreateContactUnlockOrderInput{
		ResourceID:       resourceID,
		UserID:           userID,
		ViewerMerchantID: strings.TrimSpace(req.ViewerMerchantID),
	})
	if err != nil {
		logx.Errorf("创建联系方式查看订单失败: resourceId=%s userId=%s viewerMerchantId=%s err=%+v", resourceID, userID, strings.TrimSpace(req.ViewerMerchantID), err)
		return CreateContactUnlockOrderResp{}, err
	}
	return CreateContactUnlockOrderResp{
		OrderID:         result.ID,
		Status:          result.Status,
		AlreadyUnlocked: result.AlreadyUnlocked,
		PriceCent:       result.PriceCent,
		Currency:        result.Currency,
		Message:         contactUnlockOrderMessage(result),
	}, nil
}

func contactUnlockOrderMessage(result model.ContactUnlockOrderResult) string {
	if result.AlreadyUnlocked {
		return "联系方式已解锁"
	}
	if result.PriceCent <= 0 {
		return "无需支付，可直接查看联系方式"
	}
	return "订单已创建，请完成支付后查看联系方式"
}
