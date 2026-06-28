package server

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

func registerCompatAPIRoutes(srv *rest.Server, apiHandler http.Handler) {
	if apiHandler == nil {
		return
	}
	handler := apiHandler.ServeHTTP
	// 过渡期先按 app.api/goctl 生成结果注册 go-zero 路由，再委托到现有 APIRouter，避免一次迁移全部 handler 造成行为回归。
	srv.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/wechat-login", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/me", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/me/phone", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/uploads/token", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/merchants", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/merchants/:merchantId", Handler: handler},
		{Method: http.MethodPatch, Path: "/api/v1/merchants/:merchantId", Handler: handler},

		{Method: http.MethodPost, Path: "/api/v1/resources", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/drafts", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/submit", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/resources", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/resource-search", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/me/resources", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/resources/:resourceId", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/detail-view", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/refresh", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/deal-feedback", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/take-down", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/repost-similar", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/resources/:resourceId/contact-events", Handler: handler},

		{Method: http.MethodPost, Path: "/api/v1/purchase-demands", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/me/purchase-demands", Handler: handler},

		{Method: http.MethodGet, Path: "/api/v1/home/banners", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/topics/:topicId/resources", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/webview/validate", Handler: handler},

		{Method: http.MethodPost, Path: "/api/v1/merchants/:merchantId/verifications", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/merchants/:merchantId/verifications/latest", Handler: handler},

		{Method: http.MethodGet, Path: "/api/v1/merchants/:merchantId/entitlements", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/merchants/:merchantId/top-vouchers", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/top-vouchers/:voucherId/redeem", Handler: handler},

		{Method: http.MethodGet, Path: "/api/v1/resources/:resourceId/metrics", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/merchants/:merchantId/metrics/summary", Handler: handler},

		{Method: http.MethodGet, Path: "/api/v1/messages", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/messages/:messageId/read", Handler: handler},

		{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/overview", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/resources/pending", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/resources/:resourceId/review", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/verifications/pending", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/verifications/:verificationId/review", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/merchants/:merchantId/entitlements", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/match-cases", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/match-cases", Handler: handler},
		{Method: http.MethodPatch, Path: "/api/v1/admin/match-cases/:matchCaseId/status", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/match-cases/:matchCaseId/resources", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/match-cases/:matchCaseId/participants", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/operation-logs", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/search-logs", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/tasks/resource-lifecycle/run", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/merchants", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/purchase-demands", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/purchase-demands/:demandId", Handler: handler},
		{Method: http.MethodPatch, Path: "/api/v1/admin/purchase-demands/:demandId/status", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/banner-topics", Handler: handler},
		{Method: http.MethodPost, Path: "/api/v1/admin/banner-topics", Handler: handler},
		{Method: http.MethodPatch, Path: "/api/v1/admin/banner-topics/:configId", Handler: handler},
		{Method: http.MethodGet, Path: "/api/v1/admin/resource-type-configs", Handler: handler},
		{Method: http.MethodPatch, Path: "/api/v1/admin/resource-type-configs/:configId", Handler: handler},
	})
}
