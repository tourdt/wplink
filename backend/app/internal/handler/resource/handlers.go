package resource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/handler/handlerx"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

type resourceDependencies struct {
	resource      bool
	merchant      bool
	user          bool
	searchLog     bool
	contactEvent  bool
	contactUnlock bool
	metric        bool
	operationLog  bool
	growth        bool
	adminToken    bool
	userToken     bool
}

func requireResourceDependencies(r *http.Request, svcCtx *svc.ServiceContext, deps resourceDependencies) error {
	missing := svcCtx == nil || svcCtx.APIStore == nil
	if !missing {
		store := svcCtx.APIStore
		missing = (deps.resource && store.ResourceModel == nil) ||
			(deps.merchant && store.MerchantModel == nil) ||
			(deps.user && store.UserModel == nil) ||
			(deps.searchLog && store.SearchLogModel == nil) ||
			(deps.contactEvent && store.ResourceContactEventModel == nil) ||
			(deps.contactUnlock && store.ResourceContactUnlockModel == nil) ||
			(deps.metric && store.ResourceMetricDailyModel == nil) ||
			(deps.operationLog && store.OperationLogModel == nil) ||
			(deps.growth && store.GrowthCampaignModel == nil)
	}
	if !missing {
		missing = (deps.adminToken && nilLike(svcCtx.AdminTokenService)) ||
			(deps.userToken && nilLike(svcCtx.UserTokenService))
	}
	if !missing {
		return nil
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	logx.WithContext(ctx).Error("资源 Handler 依赖未配置")
	return errx.New(errx.CodeInternalError, "资源服务暂不可用，请稍后重试")
}

func nilLike(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func parseResourceJSON(r *http.Request, target any) error {
	if r == nil || r.Body == nil {
		return errx.New(errx.CodeValidationFailed, "请求参数格式不正确")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errx.New(errx.CodeValidationFailed, "请求参数格式不正确")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errx.New(errx.CodeValidationFailed, "请求参数格式不正确")
	}
	return nil
}

func merchantPermissionDeps(svcCtx *svc.ServiceContext) handlerx.MerchantPermissionDeps {
	return handlerx.MerchantPermissionDeps{
		UserTokenService:  svcCtx.UserTokenService,
		AdminTokenService: svcCtx.AdminTokenService,
		Store:             svcCtx.APIStore,
	}
}

func resourceMerchantID(ctx context.Context, store *svc.APIStore, resourceID string) (string, error) {
	merchantID, err := store.GetResourceMerchantID(ctx, strings.TrimSpace(resourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return "", errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
	}
	if err != nil {
		logx.WithContext(ctx).Errorw("读取资源所属商家失败", logx.Field("resourceId", strings.TrimSpace(resourceID)), logx.Field("errorType", fmt.Sprintf("%T", err)))
		return "", errx.New(errx.CodeInternalError, "资源信息加载失败，请稍后重试")
	}
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return "", errx.New(errx.CodeForbidden, "您没有权限操作该资源")
	}
	return merchantID, nil
}

func requireResourceOwner(r *http.Request, svcCtx *svc.ServiceContext, resourceID string) (string, error) {
	merchantID, err := resourceMerchantID(r.Context(), svcCtx.APIStore, resourceID)
	if err != nil {
		return "", err
	}
	if err := handlerx.RequireMerchant(r, merchantPermissionDeps(svcCtx), merchantID); err != nil {
		return "", err
	}
	return merchantID, nil
}

func bindResourceCreator(r *http.Request, svcCtx *svc.ServiceContext, req *resourcelogic.CreateResourceReq) error {
	if admin, ok := handlerx.OptionalAdmin(r, svcCtx.AdminTokenService); ok && permission.CanAccessAdmin(admin.Roles) {
		req.CreatedByOperator = strings.TrimSpace(admin.OperatorID)
		req.CreatedByRole = firstRole(admin.Roles, "platform_operator")
		return nil
	}
	user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
	if err != nil {
		return err
	}
	req.CreatedByUser = strings.TrimSpace(user.UserID)
	req.CreatedByRole = firstRole(user.Roles, "merchant_admin")
	return nil
}

func firstRole(roles []string, fallback string) string {
	for _, role := range roles {
		if role = strings.TrimSpace(role); role != "" {
			return role
		}
	}
	return fallback
}

func createResourceRequest(req types.CreateResourceReq) resourcelogic.CreateResourceReq {
	return resourcelogic.CreateResourceReq{
		MerchantID: req.MerchantId, CityCode: req.CityCode, TypeCode: req.TypeCode, Direction: req.Direction,
		Title: req.Title, Category: req.Category, District: req.District, PriceText: req.PriceText,
		QuantityText: req.QuantityText, Description: req.Description, Attributes: model.JSONMap(req.Attributes),
		Tags: append([]string(nil), req.Tags...), Images: append([]string(nil), req.Images...),
		Contact: resourcelogic.ResourceContactReq{Name: req.Contact.Name, Phone: req.Contact.Phone, Wechat: req.Contact.Wechat},
	}
}

func createResourceHTTPHandler(svcCtx *svc.ServiceContext, draft bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps := resourceDependencies{resource: true, merchant: true, user: true, operationLog: true, adminToken: true, userToken: true}
		if err := requireResourceDependencies(r, svcCtx, deps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body types.CreateResourceReq
		if err := parseResourceJSON(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		req := createResourceRequest(body)
		if err := handlerx.RequireMerchant(r, merchantPermissionDeps(svcCtx), req.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := bindResourceCreator(r, svcCtx, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		auditor := svcCtx.ContentAuditor
		if nilLike(auditor) {
			auditor = nil
		}
		logic := resourcelogic.NewCreateResourceLogic(svcCtx.APIStore, auditor)
		var resp resourcelogic.CreateResourceResp
		var err error
		if draft {
			resp, err = logic.CreateResourceDraft(r.Context(), req)
		} else {
			resp, err = logic.CreateResource(r.Context(), req)
		}
		response.JSON(w, resp, err)
	}
}

func updateResourceDraftHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, merchant: true, user: true, adminToken: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID := pathvar.Vars(r)["resourceId"]
		merchantID, err := requireResourceOwner(r, svcCtx, resourceID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body types.CreateResourceReq
		if err := parseResourceJSON(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		req := createResourceRequest(body)
		req.MerchantID = merchantID
		auditor := svcCtx.ContentAuditor
		if nilLike(auditor) {
			auditor = nil
		}
		resp, err := resourcelogic.NewCreateResourceLogic(svcCtx.APIStore, auditor).UpdateResourceDraft(r.Context(), resourceID, req)
		response.JSON(w, resp, err)
	}
}

func submitResourceHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, user: true, adminToken: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID := pathvar.Vars(r)["resourceId"]
		if _, err := requireResourceOwner(r, svcCtx, resourceID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		auditor := svcCtx.ContentAuditor
		if nilLike(auditor) {
			auditor = nil
		}
		resp, err := resourcelogic.NewSubmitResourceLogic(svcCtx.APIStore, auditor).SubmitResource(r.Context(), resourceID)
		response.JSON(w, resp, err)
	}
}

func listResourcesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "资源筛选参数格式不正确"))
			return
		}
		resp, err := resourcelogic.NewListResourcesLogic(svcCtx.APIStore).ListResources(r.Context(), resourcelogic.ListResourcesReq{
			CityCode: req.CityCode, MerchantID: req.MerchantId, GroupCode: req.GroupCode, TypeCode: req.TypeCode,
			Direction: req.Direction, Keyword: req.Keyword, Category: req.Category, Tags: resourceTags(r.URL.Query()),
			Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func relatedResourcesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID, err := resourcelogic.NormalizeRelatedResourceID(pathvar.Vars(r)["resourceId"])
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		source, err := svcCtx.APIStore.GetRelatedResourceSource(r.Context(), resourceID)
		if errors.Is(err, sql.ErrNoRows) {
			response.JSON(w, nil, errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看"))
			return
		}
		if err != nil {
			logx.WithContext(r.Context()).Errorw("读取相关推荐源资源失败", logx.Field("resourceId", resourceID), logx.Field("errorType", fmt.Sprintf("%T", err)))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "相关推荐加载失败，请稍后重试"))
			return
		}
		if strings.TrimSpace(source.Status) != model.ResourceStatusPublished {
			if err := authorizePrivateRelatedResource(r, svcCtx, source); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		var req types.RelatedResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "相关推荐参数格式不正确"))
			return
		}
		resp, err := resourcelogic.NewListRelatedResourcesLogic(svcCtx.APIStore).ListRelatedResources(r.Context(), resourceID, resourcelogic.RelatedResourcesReq{PageSize: req.PageSize})
		response.JSON(w, resp, err)
	}
}

func authorizePrivateRelatedResource(r *http.Request, svcCtx *svc.ServiceContext, source model.RelatedResourceSource) error {
	if svcCtx.APIStore.UserModel == nil || nilLike(svcCtx.UserTokenService) {
		return errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
	}
	user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
	if err != nil {
		return errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
	}
	allowed, err := svcCtx.APIStore.UserCanManageMerchant(r.Context(), user.UserID, source.MerchantID)
	if err != nil {
		logx.WithContext(r.Context()).Errorw("校验相关推荐源资源权限失败", logx.Field("merchantId", source.MerchantID), logx.Field("errorType", fmt.Sprintf("%T", err)))
		return errx.New(errx.CodeInternalError, "相关推荐加载失败，请稍后重试")
	}
	if !allowed {
		return errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
	}
	return nil
}

func searchResourcesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, searchLog: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SearchResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "资源搜索参数格式不正确"))
			return
		}
		user, authenticated, err := handlerx.OptionalUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		userID := ""
		if authenticated {
			userID = user.UserID
		}
		resp, err := resourcelogic.NewSearchResourcesLogic(svcCtx.APIStore).SearchResources(r.Context(), resourcelogic.SearchResourcesReq{
			UserID: userID, CityCode: req.CityCode, GroupCode: req.GroupCode, TypeCode: req.TypeCode,
			Direction: req.Direction, Keyword: req.Keyword, Category: req.Category, Tags: resourceTags(r.URL.Query()),
			VerifiedOnly: req.VerifiedOnly, Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func resourceTags(query url.Values) []string {
	values := query["tags"]
	if len(values) == 0 {
		return nil
	}
	tags := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '，' }) {
			if item = strings.TrimSpace(item); item != "" {
				tags = append(tags, item)
			}
		}
	}
	return tags
}

func listMyResourcesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, user: true, adminToken: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMyResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "我的资源查询参数格式不正确"))
			return
		}
		if err := handlerx.RequireMerchant(r, merchantPermissionDeps(svcCtx), req.MerchantId); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewListMyResourcesLogic(svcCtx.APIStore).ListMyResources(r.Context(), resourcelogic.ListMyResourcesReq{
			MerchantID: req.MerchantId, Status: req.Status, Direction: req.Direction, Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func ownResourceHTTPHandler(svcCtx *svc.ServiceContext, editable bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, user: true, adminToken: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID := pathvar.Vars(r)["resourceId"]
		merchantID, err := requireResourceOwner(r, svcCtx, resourceID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if editable {
			resp, logicErr := resourcelogic.NewGetEditableResourceLogic(svcCtx.APIStore).Get(r.Context(), resourcelogic.GetEditableResourceReq{MerchantID: merchantID, ResourceID: resourceID})
			response.JSON(w, resp, logicErr)
			return
		}
		resp, logicErr := resourcelogic.NewGetOwnResourceLogic(svcCtx.APIStore).Get(r.Context(), resourcelogic.GetOwnResourceReq{MerchantID: merchantID, ResourceID: resourceID})
		response.JSON(w, resp, logicErr)
	}
}

func getResourceHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewGetResourceLogic(svcCtx.APIStore).GetResource(r.Context(), pathvar.Vars(r)["resourceId"])
		response.JSON(w, resp, err)
	}
}

func reportResourceHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后举报"))
			return
		}
		var req types.ReportResourceReq
		if err := parseResourceJSON(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewReportResourceLogic(svcCtx.APIStore).ReportResource(r.Context(), resourcelogic.ReportResourceReq{
			ResourceID: pathvar.Vars(r)["resourceId"], ReporterUserID: user.UserID,
			ReasonCode: req.ReasonCode, ReasonText: req.ReasonText, Evidence: model.JSONMap(req.Evidence),
		})
		response.JSON(w, resp, err)
	}
}

func recordDetailViewHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{metric: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		err := metricslogic.NewRecordDetailViewLogic(svcCtx.APIStore).RecordDetailView(r.Context(), metricslogic.RecordDetailViewReq{ResourceID: pathvar.Vars(r)["resourceId"]})
		response.JSON(w, types.DetailViewResp{Message: "浏览行为已记录"}, err)
	}
}

func ownerActionHTTPHandler(svcCtx *svc.ServiceContext, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{resource: true, user: true, adminToken: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID := pathvar.Vars(r)["resourceId"]
		merchantID, err := requireResourceOwner(r, svcCtx, resourceID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		switch action {
		case "refresh":
			resp, logicErr := resourcelogic.NewRefreshResourceLogic(svcCtx.APIStore).RefreshResource(r.Context(), resourcelogic.RefreshResourceReq{MerchantID: merchantID, ResourceID: resourceID})
			response.JSON(w, resp, logicErr)
		case "deal":
			var req types.DealFeedbackReq
			if err := parseResourceJSON(r, &req); err != nil {
				response.JSON(w, nil, err)
				return
			}
			resp, logicErr := resourcelogic.NewMarkDealtLogic(svcCtx.APIStore).MarkDealt(r.Context(), resourcelogic.MarkDealtReq{
				MerchantID: merchantID, ResourceID: resourceID, IsDealt: req.IsDealt, IsReal: req.IsReal,
				ResponseTimely: req.ResponseTimely, WillingToCooperateAgain: req.WillingToCooperateAgain, Note: req.Note,
			})
			response.JSON(w, resp, logicErr)
		case "take_down":
			var req types.TakeDownOwnResourceReq
			if err := parseResourceJSON(r, &req); err != nil {
				response.JSON(w, nil, err)
				return
			}
			resp, logicErr := resourcelogic.NewTakeDownOwnResourceLogic(svcCtx.APIStore).TakeDown(r.Context(), resourcelogic.TakeDownOwnResourceReq{MerchantID: merchantID, ResourceID: resourceID, Reason: req.Reason})
			response.JSON(w, resp, logicErr)
		case "delete":
			resp, logicErr := resourcelogic.NewDeleteTakenDownResourceLogic(svcCtx.APIStore).Delete(r.Context(), resourcelogic.DeleteTakenDownResourceReq{MerchantID: merchantID, ResourceID: resourceID})
			response.JSON(w, resp, logicErr)
		case "repost":
			resp, logicErr := resourcelogic.NewRepostSimilarLogic(svcCtx.APIStore).RepostSimilar(r.Context(), resourcelogic.RepostSimilarReq{MerchantID: merchantID, ResourceID: resourceID})
			response.JSON(w, resp, logicErr)
		default:
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "资源操作暂不可用，请稍后重试"))
		}
	}
}

func contactEventHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps := resourceDependencies{user: true, contactEvent: true, contactUnlock: true, metric: true, growth: true, userToken: true}
		if err := requireResourceDependencies(r, svcCtx, deps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后联系商家"))
			return
		}
		var req types.ContactEventReq
		if err := parseResourceJSON(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := metricslogic.NewRecordContactLogic(svcCtx.APIStore).RecordContact(r.Context(), metricslogic.RecordContactReq{
			ResourceID: pathvar.Vars(r)["resourceId"], UserID: user.UserID, Action: req.Action,
		})
		response.JSON(w, resp, err)
	}
}

func contactUnlockOrderHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps := resourceDependencies{user: true, contactUnlock: true, adminToken: true, userToken: true}
		if err := requireResourceDependencies(r, svcCtx, deps); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后查看联系方式"))
			return
		}
		var req types.CreateContactUnlockOrderReq
		if err := parseResourceJSON(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if strings.TrimSpace(req.ViewerMerchantId) != "" {
			if err := handlerx.RequireMerchant(r, merchantPermissionDeps(svcCtx), req.ViewerMerchantId); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := resourcelogic.NewCreateContactUnlockOrderLogic(svcCtx.APIStore).CreateContactUnlockOrder(r.Context(), resourcelogic.CreateContactUnlockOrderReq{
			ResourceID: pathvar.Vars(r)["resourceId"], UserID: user.UserID,
			ViewerMerchantID: req.ViewerMerchantId, Action: req.Action,
		})
		response.JSON(w, resp, err)
	}
}

func contactUnlockPaymentHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireResourceDependencies(r, svcCtx, resourceDependencies{contactUnlock: true, userToken: true}); err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后支付"))
			return
		}
		var req types.CreateContactUnlockPaymentReq
		if err := parseResourceJSON(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		gateway := svcCtx.WechatPayGateway
		if nilLike(gateway) {
			gateway = nil
		}
		devMock := svcCtx.Config.WechatPay.DevMockEnabled && !config.IsProductionMode(svcCtx.Config.RuntimeMode)
		resp, err := paymentlogic.NewCreateContactUnlockPaymentLogic(svcCtx.APIStore, gateway, devMock).CreateContactUnlockPayment(r.Context(), paymentlogic.CreateContactUnlockPaymentReq{
			ResourceID: pathvar.Vars(r)["resourceId"], OrderID: pathvar.Vars(r)["orderId"], UserID: user.UserID,
		})
		response.JSON(w, resp, err)
	}
}
