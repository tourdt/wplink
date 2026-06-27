package demand

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type CreateDemandStore interface {
	CreateDemand(ctx context.Context, input model.CreateDemandInput) (model.CreateDemandResult, error)
}

type DemandContactReq struct {
	Name   string
	Phone  string
	Wechat string
}

type CreateDemandReq struct {
	UserID              string
	CityCode            string
	DemandType          string
	Title               string
	Category            string
	PriceRange          model.JSONMap
	QuantityRequirement model.JSONMap
	Attributes          model.JSONMap
	Contact             DemandContactReq
}

type CreateDemandResp struct {
	ID      string
	Status  string
	Message string
}

type CreateDemandLogic struct {
	store CreateDemandStore
}

func NewCreateDemandLogic(store CreateDemandStore) *CreateDemandLogic {
	return &CreateDemandLogic{store: store}
}

func (l *CreateDemandLogic) CreateDemand(ctx context.Context, req CreateDemandReq) (CreateDemandResp, error) {
	input := model.CreateDemandInput{
		UserID:              strings.TrimSpace(req.UserID),
		CityCode:            strings.TrimSpace(req.CityCode),
		DemandType:          strings.TrimSpace(req.DemandType),
		Title:               strings.TrimSpace(req.Title),
		Category:            strings.TrimSpace(req.Category),
		PriceRange:          req.PriceRange,
		QuantityRequirement: req.QuantityRequirement,
		Attributes:          req.Attributes,
		ContactName:         strings.TrimSpace(req.Contact.Name),
		ContactPhone:        strings.TrimSpace(req.Contact.Phone),
		ContactWechat:       strings.TrimSpace(req.Contact.Wechat),
	}
	if input.UserID == "" || input.DemandType == "" || input.Title == "" || input.Category == "" || input.ContactName == "" || input.ContactPhone == "" {
		return CreateDemandResp{}, errx.New(errx.CodeValidationFailed, "请补充需求类型、品类和联系方式")
	}
	result, err := l.store.CreateDemand(ctx, input)
	if err != nil {
		return CreateDemandResp{}, err
	}
	return CreateDemandResp{ID: result.ID, Status: result.Status, Message: "需求已提交，平台会尽快为您匹配"}, nil
}
