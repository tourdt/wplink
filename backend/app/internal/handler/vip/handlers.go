package vip

import (
	"context"
	"net/http"
	"reflect"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/handler/handlerx"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	viplogic "wplink/backend/app/internal/logic/vip"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func listVIPPlansHTTPHandler(store viplogic.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireVIPStore(r, store); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := viplogic.NewListVIPPlansLogic(store).ListVIPPlans(r.Context())
		response.JSON(w, resp, err)
	}
}

func listQuotaPacksHTTPHandler(store viplogic.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireVIPStore(r, store); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := viplogic.NewListQuotaPacksLogic(store).ListQuotaPacks(r.Context())
		response.JSON(w, resp, err)
	}
}

func getMerchantVIPHTTPHandler(store viplogic.Store, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireVIPPrivateDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := viplogic.NewGetMerchantVIPLogic(store).GetMerchantVIP(r.Context(), merchantID)
		response.JSON(w, resp, err)
	}
}

func createVIPOrderHTTPHandler(store viplogic.Store, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireVIPPrivateDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.CreateVIPOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, permissionDeps.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 购买身份只取服务端 Token；契约 DTO 不接受 userId，客户端同名字段不会进入业务请求。
		resp, err := viplogic.NewCreateVIPOrderLogic(store).CreateVIPOrder(r.Context(), viplogic.CreateVIPOrderReq{
			MerchantID:  merchantID,
			UserID:      user.UserID,
			ProductType: req.ProductType,
			ProductCode: req.ProductCode,
			PlanCode:    req.PlanCode,
			ResourceID:  req.ResourceId,
		})
		response.JSON(w, resp, err)
	}
}

func createVIPPaymentHTTPHandler(store paymentlogic.VIPPaymentStore, gateway paymentlogic.WechatPayGateway, permissionDeps handlerx.MerchantPermissionDeps, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireVIPPrivateDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.CreateVIPPaymentReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		vars := pathvar.Vars(r)
		merchantID := vars["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, permissionDeps.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		requestGateway := gateway
		if vipDependencyMissing(requestGateway) {
			requestGateway = nil
		}
		devMock := cfg.WechatPay.DevMockEnabled && !config.IsProductionMode(cfg.RuntimeMode)
		resp, err := paymentlogic.NewCreateVIPPaymentLogic(store, requestGateway, devMock).CreateVIPPayment(r.Context(), paymentlogic.CreateVIPPaymentReq{
			MerchantID: merchantID,
			OrderID:    vars["orderId"],
			UserID:     user.UserID,
		})
		response.JSON(w, resp, err)
	}
}

func requireVIPStore(r *http.Request, store any) error {
	if !vipDependencyMissing(store) {
		return nil
	}
	logx.WithContext(vipRequestContext(r)).Error("VIP Handler 存储依赖未配置")
	return errx.New(errx.CodeInternalError, "VIP 服务暂不可用，请稍后重试")
}

func requireVIPPrivateDependencies(r *http.Request, store any, permissionDeps handlerx.MerchantPermissionDeps) error {
	if !vipDependencyMissing(store) && !vipDependencyMissing(permissionDeps.UserTokenService) &&
		!vipDependencyMissing(permissionDeps.AdminTokenService) && !vipDependencyMissing(permissionDeps.Store) {
		return nil
	}
	logx.WithContext(vipRequestContext(r)).Error("VIP Handler 依赖未配置")
	return errx.New(errx.CodeInternalError, "VIP 服务暂不可用，请稍后重试")
}

func vipDependencyMissing(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func vipRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func vipStoreFromServiceContext(svcCtx *svc.ServiceContext) *svc.APIStore {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.VIPModel == nil {
		return nil
	}
	return svcCtx.APIStore
}

func vipPermissionDepsFromServiceContext(svcCtx *svc.ServiceContext) handlerx.MerchantPermissionDeps {
	if svcCtx == nil {
		return handlerx.MerchantPermissionDeps{}
	}
	var permissionStore handlerx.MerchantPermissionStore
	if svcCtx.APIStore != nil && svcCtx.APIStore.UserModel != nil {
		permissionStore = svcCtx.APIStore
	}
	return handlerx.MerchantPermissionDeps{
		UserTokenService:  svcCtx.UserTokenService,
		AdminTokenService: svcCtx.AdminTokenService,
		Store:             permissionStore,
	}
}
