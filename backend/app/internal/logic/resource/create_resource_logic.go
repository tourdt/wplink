package resource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateResourceStore interface {
	GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error)
	GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error)
	GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error)
	CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error)
	UpdateResourceDraft(ctx context.Context, resourceID string, input model.CreateResourceInput) (model.CreateResourceResult, error)
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
	input, typeCode, err := l.buildResourceInput(ctx, req, status)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.CreateResource(ctx, input)
	if err != nil {
		logx.Errorf("创建资源失败: merchantId=%s typeCode=%s targetStatus=%s err=%+v", strings.TrimSpace(req.MerchantID), typeCode, status, err)
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
			logx.Errorf("记录代发布资源操作日志失败: operatorId=%s resourceId=%s status=%s err=%+v", strings.TrimSpace(req.CreatedByUser), result.ID, result.Status, err)
			return CreateResourceResp{}, err
		}
	}
	// 新建资源可能直接进入审核队列，也可能只是草稿；日志记录目标状态，避免排查时只看到写入成功但不知道用户路径。
	logx.Infof("创建资源成功: merchantId=%s resourceId=%s typeCode=%s status=%s createdByRole=%s", strings.TrimSpace(req.MerchantID), result.ID, typeCode, result.Status, strings.TrimSpace(req.CreatedByRole))
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: message}, nil
}

func (l *CreateResourceLogic) UpdateResourceDraft(ctx context.Context, resourceID string, req CreateResourceReq) (CreateResourceResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return CreateResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	input, _, err := l.buildResourceInput(ctx, req, model.ResourceStatusDraft)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.UpdateResourceDraft(ctx, resourceID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("更新资源草稿被拦截: merchantId=%s resourceId=%s reason=not_editable", strings.TrimSpace(req.MerchantID), resourceID)
			return CreateResourceResp{}, errx.New(errx.CodeStateConflict, "资源不存在或当前状态不可编辑")
		}
		logx.Errorf("更新资源草稿失败: merchantId=%s resourceId=%s err=%+v", strings.TrimSpace(req.MerchantID), resourceID, err)
		return CreateResourceResp{}, err
	}
	logx.Infof("更新资源草稿成功: merchantId=%s resourceId=%s status=%s", strings.TrimSpace(req.MerchantID), result.ID, result.Status)
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: "草稿已保存，请重新提交审核"}, nil
}

func (l *CreateResourceLogic) buildResourceInput(ctx context.Context, req CreateResourceReq, status string) (model.CreateResourceInput, string, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	typeCode := strings.TrimSpace(req.TypeCode)
	config, err := l.store.GetResourcePublishConfig(ctx, cityCode, typeCode)
	if err != nil {
		return model.CreateResourceInput{}, "", err
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
	if shouldUseMerchantContactPhone(values["contactPhone"]) {
		// 发布页只能从公开商家资料拿到脱敏手机号，保存资源前用商家资料真实电话兜底。
		merchantPhone, err := l.store.GetMerchantContactPhone(ctx, values["merchantId"])
		if err != nil {
			return model.CreateResourceInput{}, "", err
		}
		values["contactPhone"] = strings.TrimSpace(merchantPhone)
	}
	for _, field := range config.RequiredFields {
		if strings.TrimSpace(values[field]) == "" {
			return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, fmt.Sprintf("请补充%s", field))
		}
	}
	merchantStatus, err := l.store.GetMerchantPublishStatus(ctx, values["merchantId"])
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if merchantStatus != model.MerchantStatusActive {
		return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, "商家已停用，不能发布资源")
	}

	return model.CreateResourceInput{
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
		CoverURL:             firstResourceImage(req.Images),
		Description:          values["description"],
		Attributes:           req.Attributes,
		Tags:                 append([]string(nil), req.Tags...),
		Images:               append([]string(nil), req.Images...),
		ContactName:          values["contactName"],
		ContactPhone:         values["contactPhone"],
		ContactWechat:        strings.TrimSpace(req.Contact.Wechat),
		CreatedByUser:        strings.TrimSpace(req.CreatedByUser),
	}, typeCode, nil
}

func isOperatorProxy(role string) bool {
	role = strings.TrimSpace(role)
	return role == "platform_operator" || role == "super_admin"
}

func firstResourceImage(images []string) string {
	for _, image := range images {
		if trimmed := strings.TrimSpace(image); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func shouldUseMerchantContactPhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phone == "" || strings.Contains(phone, "*")
}
