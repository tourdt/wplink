package merchant

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func UpdateMerchantHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMerchantHandlerStore(r, svcCtx, true); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{
			UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore,
		}, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		// goctl 将 optional float64 生成为值类型；路由专属 Payload 用指针保留“字段缺失”与合法 0 坐标的区别。
		var req updateMerchantPayload
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "商家资料参数格式不正确"))
			return
		}
		resp, err := merchantlogic.NewUpdateMerchantLogic(svcCtx.APIStore, svcCtx.SMSVerifier).UpdateMerchant(r.Context(), merchantID, merchantlogic.UpdateMerchantReq{
			Name: req.Name, MainCategories: append([]string(nil), req.MainCategories...),
			Description: req.Description, LogoURL: req.LogoURL, Images: append([]string(nil), req.Images...),
			ContactName: req.ContactName, ContactPhone: req.ContactPhone, ContactWechat: req.ContactWechat,
			PublicPhone: req.PublicPhone, PublicWechat: req.PublicWechat,
			BusinessProfile: mapBusinessProfileInput(req.BusinessProfile), MapVisible: req.MapVisible,
			AddressText: req.AddressText, LocationAction: req.LocationAction,
			MarketName: req.MarketName, BuildingName: req.BuildingName, FloorNo: req.FloorNo, BoothNo: req.BoothNo,
			Latitude: req.Latitude, Longitude: req.Longitude, LocationName: req.LocationName, SmsCode: req.SmsCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.UpdateMerchantResp{Id: resp.ID, UpdatedAt: resp.UpdatedAt}, nil)
	}
}

type updateMerchantPayload struct {
	Name            string                      `json:"name,optional"`
	MainCategories  []string                    `json:"mainCategories,optional"`
	Description     string                      `json:"description,optional"`
	LogoURL         string                      `json:"logoUrl,optional"`
	Images          []string                    `json:"images,optional"`
	ContactName     string                      `json:"contactName,optional"`
	ContactPhone    string                      `json:"contactPhone,optional"`
	ContactWechat   string                      `json:"contactWechat,optional"`
	PublicPhone     *bool                       `json:"publicPhone,optional"`
	PublicWechat    *bool                       `json:"publicWechat,optional"`
	BusinessProfile *types.BusinessProfileInput `json:"businessProfile"`
	MapVisible      *bool                       `json:"mapVisible"`
	AddressText     string                      `json:"addressText,optional"`
	LocationAction  string                      `json:"locationAction"`
	MarketName      string                      `json:"marketName,optional"`
	BuildingName    string                      `json:"buildingName,optional"`
	FloorNo         string                      `json:"floorNo,optional"`
	BoothNo         string                      `json:"boothNo,optional"`
	Latitude        *float64                    `json:"latitude,optional"`
	Longitude       *float64                    `json:"longitude,optional"`
	LocationName    string                      `json:"locationName,optional"`
	SmsCode         string                      `json:"smsCode,optional"`
}

func mapBusinessProfileInput(input *types.BusinessProfileInput) *model.BusinessProfile {
	if input == nil {
		return nil
	}
	return &model.BusinessProfile{SchemaVersion: input.SchemaVersion, EntityKind: input.EntityKind, PrimaryRole: input.PrimaryRole,
		Roles: append([]string(nil), input.Roles...), ServiceArea: input.ServiceArea,
		ProcessingMethods: append([]string(nil), input.ProcessingMethods...), PropertyTypes: append([]string(nil), input.PropertyTypes...), ServiceItems: append([]string(nil), input.ServiceItems...)}
}
