package resource

import (
	"context"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type CreateResourceStore interface {
	GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error)
	GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error)
	CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error)
	RecordOperationLog(ctx context.Context, input model.OperationLogInput) error
}

type ResourceContactReq struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Wechat string `json:"wechat,omitempty"`
}

type CreateResourceReq struct {
	MerchantID    string
	CityCode      string
	TypeCode      string
	Title         string
	Category      string
	District      string
	PriceText     string
	QuantityText  string
	Description   string
	Attributes    model.JSONMap
	Tags          []string
	Images        []string
	Contact       ResourceContactReq
	CreatedByUser string
	CreatedByRole string
}

type CreateResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type CreateResourceLogic struct {
	store CreateResourceStore
}

func NewCreateResourceLogic(store CreateResourceStore) *CreateResourceLogic {
	return &CreateResourceLogic{store: store}
}

func (l *CreateResourceLogic) CreateResource(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusPending, "已提交审核，审核通过后将展示给买家")
}

func (l *CreateResourceLogic) CreateResourceDraft(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusDraft, "草稿已保存")
}

func (l *CreateResourceLogic) create(ctx context.Context, req CreateResourceReq, status string, message string) (CreateResourceResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	typeCode := strings.TrimSpace(req.TypeCode)
	config, err := l.store.GetResourcePublishConfig(ctx, cityCode, typeCode)
	if err != nil {
		return CreateResourceResp{}, err
	}

	values := map[string]string{
		"merchantId":   strings.TrimSpace(req.MerchantID),
		"cityCode":     cityCode,
		"typeCode":     typeCode,
		"title":        strings.TrimSpace(req.Title),
		"category":     strings.TrimSpace(req.Category),
		"quantityText": strings.TrimSpace(req.QuantityText),
		"priceText":    strings.TrimSpace(req.PriceText),
		"contactName":  strings.TrimSpace(req.Contact.Name),
		"contactPhone": strings.TrimSpace(req.Contact.Phone),
		"description":  strings.TrimSpace(req.Description),
	}
	for _, field := range config.RequiredFields {
		if strings.TrimSpace(values[field]) == "" {
			return CreateResourceResp{}, errx.New(errx.CodeValidationFailed, fmt.Sprintf("请补充%s", field))
		}
	}
	merchantStatus, err := l.store.GetMerchantPublishStatus(ctx, values["merchantId"])
	if err != nil {
		return CreateResourceResp{}, err
	}
	if merchantStatus != model.MerchantStatusActive {
		return CreateResourceResp{}, errx.New(errx.CodeValidationFailed, "商家已停用，不能发布资源")
	}

	result, err := l.store.CreateResource(ctx, model.CreateResourceInput{
		MerchantID:           values["merchantId"],
		CityCode:             cityCode,
		ResourceTypeConfigID: config.ID,
		TypeCode:             typeCode,
		Status:               status,
		Title:                values["title"],
		Category:             values["category"],
		District:             strings.TrimSpace(req.District),
		PriceText:            values["priceText"],
		QuantityText:         values["quantityText"],
		Description:          values["description"],
		Attributes:           req.Attributes,
		Tags:                 append([]string(nil), req.Tags...),
		Images:               append([]string(nil), req.Images...),
		ContactName:          values["contactName"],
		ContactPhone:         values["contactPhone"],
		ContactWechat:        strings.TrimSpace(req.Contact.Wechat),
		CreatedByUser:        strings.TrimSpace(req.CreatedByUser),
	})
	if err != nil {
		return CreateResourceResp{}, err
	}
	if isOperatorProxy(req.CreatedByRole) && strings.TrimSpace(req.CreatedByUser) != "" {
		if err := l.store.RecordOperationLog(ctx, model.OperationLogInput{
			OperatorID:     strings.TrimSpace(req.CreatedByUser),
			OperatorRole:   strings.TrimSpace(req.CreatedByRole),
			Action:         "proxy_create_resource",
			ObjectType:     "resource",
			ObjectID:       result.ID,
			BeforeSnapshot: model.JSONMap{},
			AfterSnapshot:  model.JSONMap{"status": result.Status, "typeCode": typeCode},
		}); err != nil {
			return CreateResourceResp{}, err
		}
	}
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: message}, nil
}

func isOperatorProxy(role string) bool {
	role = strings.TrimSpace(role)
	return role == "platform_operator" || role == "super_admin"
}
