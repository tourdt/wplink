package merchant

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateMerchantStore interface {
	UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error)
}

type UpdateMerchantReq struct {
	Name           string   `json:"name,omitempty"`
	MainCategories []string `json:"mainCategories,omitempty"`
	MerchantType   string   `json:"merchantType,omitempty"`
	Description    string   `json:"description,omitempty"`
	LogoURL        string   `json:"logoUrl,omitempty"`
	Images         []string `json:"images,omitempty"`
	ContactName    string   `json:"contactName,omitempty"`
	ContactPhone   string   `json:"contactPhone,omitempty"`
	ContactWechat  string   `json:"contactWechat,omitempty"`
	AddressText    string   `json:"addressText,omitempty"`
	LocationAction string   `json:"locationAction,omitempty"`
	MarketName     string   `json:"marketName,omitempty"`
	BuildingName   string   `json:"buildingName,omitempty"`
	FloorNo        string   `json:"floorNo,omitempty"`
	BoothNo        string   `json:"boothNo,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	LocationName   string   `json:"locationName,omitempty"`
	SmsCode        string   `json:"smsCode,omitempty"`
}

type UpdateMerchantResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type UpdateMerchantLogic struct {
	store UpdateMerchantStore
}

func NewUpdateMerchantLogic(store UpdateMerchantStore, _ ...interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}) *UpdateMerchantLogic {
	return &UpdateMerchantLogic{store: store}
}

func (l *UpdateMerchantLogic) UpdateMerchant(ctx context.Context, merchantID string, req UpdateMerchantReq) (UpdateMerchantResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}
	name := strings.TrimSpace(req.Name)
	if err := validateMerchantName(name); err != nil {
		return UpdateMerchantResp{}, err
	}
	contactPhone := strings.TrimSpace(req.ContactPhone)
	if contactPhone != "" {
		if !isValidContactPhone(contactPhone) {
			return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "手机号格式不正确")
		}
	}
	location, err := buildMerchantLocationPatch(req)
	if err != nil {
		return UpdateMerchantResp{}, err
	}

	updatedAt, err := l.store.UpdateMerchant(ctx, merchantID, model.UpdateMerchantPatch{
		Name:           name,
		MainCategories: append([]string(nil), req.MainCategories...),
		MerchantType:   strings.TrimSpace(req.MerchantType),
		Description:    strings.TrimSpace(req.Description),
		LogoURL:        strings.TrimSpace(req.LogoURL),
		Images:         append([]string(nil), req.Images...),
		ContactName:    strings.TrimSpace(req.ContactName),
		ContactPhone:   contactPhone,
		ContactWechat:  strings.TrimSpace(req.ContactWechat),
		AddressText:    strings.TrimSpace(req.AddressText),
		Location:       location,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateMerchantResp{}, errx.New(errx.CodeResourceNotFound, "商家不存在或已停用")
		}
		// 数据库根错误只进入工程日志，前端使用稳定中文提示，避免泄露 SQL 或内部表结构。
		logx.WithContext(ctx).Errorw("商家资料数据库保存失败", logx.Field("merchantId", merchantID), logx.Field("locationAction", location.Action), logx.Field("err", err))
		return UpdateMerchantResp{}, errx.New(errx.CodeInternalError, "商家资料保存失败，请稍后重试")
	}
	return UpdateMerchantResp{ID: merchantID, UpdatedAt: updatedAt}, nil
}

func buildMerchantLocationPatch(req UpdateMerchantReq) (model.MerchantLocationPatch, error) {
	action := strings.TrimSpace(req.LocationAction)
	switch action {
	case "clear":
		// clear 是完整清除操作，即使请求夹带旧坐标或档口文字也必须忽略，防止位置隐私残留。
		return model.MerchantLocationPatch{Action: action, Status: model.MerchantLocationStatusHidden}, nil
	case "set":
		// 腾讯地图允许赤道或本初子午线上的合法 0 坐标，因此必须用指针区分缺失字段和显式 0。
		if req.Latitude == nil || req.Longitude == nil || !coordinatesAreValid(*req.Latitude, *req.Longitude) {
			return model.MerchantLocationPatch{}, errx.New(errx.CodeValidationFailed, "地图位置格式不正确，请重新选择")
		}
		return model.MerchantLocationPatch{
			Action:       action,
			MarketName:   strings.TrimSpace(req.MarketName),
			BuildingName: strings.TrimSpace(req.BuildingName),
			FloorNo:      strings.TrimSpace(req.FloorNo),
			BoothNo:      strings.TrimSpace(req.BoothNo),
			Latitude:     *req.Latitude,
			Longitude:    *req.Longitude,
			LocationName: strings.TrimSpace(req.LocationName),
			Status:       model.MerchantLocationStatusVisible,
		}, nil
	default:
		return model.MerchantLocationPatch{}, errx.New(errx.CodeValidationFailed, "地图位置操作不正确，请重新选择")
	}
}

func coordinatesAreValid(latitude float64, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsInf(latitude, 0) &&
		!math.IsNaN(longitude) && !math.IsInf(longitude, 0) &&
		latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func isValidContactPhone(phone string) bool {
	if len(phone) < 6 || len(phone) > 20 {
		return false
	}
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
