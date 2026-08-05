package discovery

import (
	"net/http"

	discoverylogic "wplink/backend/app/internal/logic/discovery"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func GetTopicResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TopicResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "专题资源查询参数格式不正确"))
			return
		}
		// topicId 不属于生成查询 DTO，必须从 go-zero 注入的路径变量读取，不能回退到手工拆分 URL。
		topicID := pathvar.Vars(r)["topicId"]
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(svcCtx.APIStore).GetTopicResources(r.Context(), discoverylogic.TopicResourcesReq{
			TopicID:  topicID,
			CityCode: req.CityCode,
			Page:     req.Page,
			PageSize: req.PageSize,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		items := make([]types.TopicResourceItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.TopicResourceItem{
				Id:           item.ID,
				TypeCode:     item.TypeCode,
				Title:        item.Title,
				Category:     item.Category,
				CoverUrl:     item.CoverURL,
				District:     item.District,
				PriceText:    item.PriceText,
				QuantityText: item.QuantityText,
				MerchantName: item.MerchantName,
				DealtAt:      item.DealtAt,
			})
		}
		response.JSON(w, types.TopicResourcesResp{
			Topic: types.TopicInfoResp{
				Id:       resp.Topic.ID,
				Title:    resp.Topic.Title,
				Subtitle: resp.Topic.Subtitle,
				CoverUrl: resp.Topic.CoverURL,
				Tags:     append([]string{}, resp.Topic.Tags...),
			},
			Items:    items,
			Page:     resp.Page,
			PageSize: resp.PageSize,
			Total:    resp.Total,
		}, nil)
	}
}
