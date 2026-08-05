package merchant

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func GetMerchantHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMerchantHandlerStore(r, svcCtx, false); err != nil {
			response.JSON(w, nil, err)
			return
		}
		subject, authenticated, err := handlerx.OptionalUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if authenticated && svcCtx.APIStore.UserModel == nil {
			logx.WithContext(r.Context()).Error("商家详情权限存储依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "商家资料服务暂不可用，请稍后重试"))
			return
		}
		viewerUserID := ""
		if authenticated {
			viewerUserID = subject.UserID
		}
		resp, err := merchantlogic.NewGetMerchantLogic(svcCtx.APIStore).GetMerchant(r.Context(), pathvar.Vars(r)["merchantId"], viewerUserID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		creditTags := make([]types.CreditTagInfo, 0, len(resp.CreditTags))
		for _, tag := range resp.CreditTags {
			creditTags = append(creditTags, types.CreditTagInfo{Code: tag.Code, Label: tag.Label})
		}
		var contact *types.MerchantContactInfo
		if resp.Contact != nil {
			contact = &types.MerchantContactInfo{
				Name: resp.Contact.Name, Phone: resp.Contact.Phone, Wechat: resp.Contact.Wechat,
				PhoneMasked: resp.Contact.PhoneMasked, WechatMasked: resp.Contact.WechatMasked,
			}
		}
		response.JSON(w, merchantDetailPayload{MerchantDetailResp: types.MerchantDetailResp{
			Id: resp.ID, MerchantNo: resp.MerchantNo, Name: resp.Name, MerchantType: resp.MerchantType,
			CityCode: resp.CityCode, MainCategories: append([]string{}, resp.MainCategories...), ProfileStatus: resp.ProfileStatus,
			VipStatus: resp.VIPStatus, CreditTags: creditTags,
			ResourcesSummary: types.MerchantResourcesSummary{PublishedCount: resp.ResourcesSummary.PublishedCount, DealtCount: resp.ResourcesSummary.DealtCount},
			HeatScore:        resp.HeatScore, AddressText: resp.AddressText, Location: map[string]interface{}(resp.Location),
			Description: resp.Description, LogoUrl: resp.LogoURL, Images: append([]string{}, resp.Images...), LastActiveAt: resp.LastActiveAt,
		}, Contact: contact}, nil)
	}
}

// goctl 保留了契约中的 optional 标记但 encoding/json 不识别该选项；外层字段确保匿名详情继续彻底省略联系方式。
type merchantDetailPayload struct {
	types.MerchantDetailResp
	Contact *types.MerchantContactInfo `json:"contact,omitempty"`
}

func requireMerchantHandlerStore(r *http.Request, svcCtx *svc.ServiceContext, requirePermissionStore bool) error {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantModel == nil || (requirePermissionStore && svcCtx.APIStore.UserModel == nil) {
		logx.WithContext(r.Context()).Errorw("商家 Handler 存储依赖未配置", logx.Field("permissionStoreRequired", requirePermissionStore))
		return errx.New(errx.CodeInternalError, "商家资料服务暂不可用，请稍后重试")
	}
	return nil
}
