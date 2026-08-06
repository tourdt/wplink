package discovery

import (
	"net/http"

	discoverylogic "wplink/backend/app/internal/logic/discovery"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ValidateWebviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ValidateWebviewReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "活动链接参数格式不正确"))
			return
		}
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(svcCtx.APIStore).ValidateWebviewURL(r.Context(), discoverylogic.ValidateWebviewURLReq{
			URL: req.Url,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ValidateWebviewResp{
			Allowed: resp.Allowed,
			Url:     resp.URL,
		}, nil)
	}
}
