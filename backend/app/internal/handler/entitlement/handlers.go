package entitlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"wplink/backend/app/internal/handler/handlerx"
	entitlementlogic "wplink/backend/app/internal/logic/entitlement"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

type topVoucherOwnerStore interface {
	GetTopVoucherMerchantID(ctx context.Context, voucherID string) (string, error)
}

func listMerchantEntitlementsHTTPHandler(store entitlementlogic.EntitlementStore, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireEntitlementHandlerDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListEntitlementsLogic(store).ListEntitlements(r.Context(), merchantID)
		response.JSON(w, resp, err)
	}
}

func listEntitlementUsageRecordsHTTPHandler(store entitlementlogic.EntitlementStore, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireEntitlementHandlerDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		vars := pathvar.Vars(r)
		merchantID := vars["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListEntitlementUsageRecordsLogic(store).ListUsageRecords(r.Context(), merchantID, vars["entitlementId"])
		response.JSON(w, resp, err)
	}
}

func listTopVouchersHTTPHandler(store entitlementlogic.EntitlementStore, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireEntitlementHandlerDependencies(r, store, permissionDeps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListTopVouchersLogic(store).ListTopVouchers(r.Context(), merchantID)
		response.JSON(w, resp, err)
	}
}

func redeemTopVoucherHTTPHandler(store entitlementlogic.EntitlementStore, ownerStore topVoucherOwnerStore, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireEntitlementHandlerDependencies(r, store, permissionDeps, ownerStore); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.RedeemTopVoucherReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		voucherID := strings.TrimSpace(pathvar.Vars(r)["voucherId"])
		merchantID, err := ownerStore.GetTopVoucherMerchantID(r.Context(), voucherID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "置顶券不存在或不可用"))
				return
			}
			// 原始数据库错误可能包含 SQL 或连接信息，只记录安全类型与稳定券标识。
			logx.WithContext(r.Context()).Errorw("查询置顶券所属商家失败", logx.Field("voucherId", voucherID), logx.Field("errorType", fmt.Sprintf("%T", err)))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "置顶券信息加载失败，请稍后重试"))
			return
		}
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewRedeemTopVoucherLogic(store).RedeemTopVoucher(r.Context(), entitlementlogic.RedeemTopVoucherReq{
			MerchantID: merchantID,
			VoucherID:  voucherID,
			ResourceID: req.ResourceId,
		})
		response.JSON(w, resp, err)
	}
}

func requireEntitlementHandlerDependencies(r *http.Request, store entitlementlogic.EntitlementStore, permissionDeps handlerx.MerchantPermissionDeps, extra ...any) error {
	missing := entitlementDependencyMissing(store) || entitlementDependencyMissing(permissionDeps.UserTokenService) ||
		entitlementDependencyMissing(permissionDeps.AdminTokenService) || entitlementDependencyMissing(permissionDeps.Store)
	for _, dependency := range extra {
		missing = missing || entitlementDependencyMissing(dependency)
	}
	if !missing {
		return nil
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	logx.WithContext(ctx).Error("权益 Handler 依赖未配置")
	return errx.New(errx.CodeInternalError, "权益服务暂不可用，请稍后重试")
}

func entitlementDependencyMissing(dependency any) bool {
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

func entitlementStoreFromServiceContext(svcCtx *svc.ServiceContext) *svc.APIStore {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantEntitlementModel == nil {
		return nil
	}
	return svcCtx.APIStore
}

func entitlementPermissionDepsFromServiceContext(svcCtx *svc.ServiceContext) handlerx.MerchantPermissionDeps {
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
