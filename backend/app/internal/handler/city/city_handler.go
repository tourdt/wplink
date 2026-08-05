package city

import (
	"net/http"

	citylogic "wplink/backend/app/internal/logic/city"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func ListCityStationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := citylogic.NewListCityStationsLogic(svcCtx.CityStore).ListCityStations(r.Context())
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ListCityStationsResp{
			Items: toCityStationTypes(resp.Items),
		}, nil)
	}
}

func ListCityResourceTypesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListResourceTypesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "供需分类查询参数格式不正确"))
			return
		}
		// 城市编码由生成路由的路径变量提供，查询参数 DTO 只承载可选的供需方向。
		cityCode := pathvar.Vars(r)["cityCode"]
		resp, err := citylogic.NewListResourceTypesLogic(svcCtx.CityStore).ListResourceTypes(r.Context(), citylogic.ListResourceTypesReq{
			CityCode:  cityCode,
			Direction: req.Direction,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ListResourceTypesResp{
			Items: toResourceTypeConfigTypes(resp.Items),
		}, nil)
	}
}

func toCityStationTypes(items []citylogic.CityStationInfo) []types.CityStationInfo {
	result := make([]types.CityStationInfo, 0, len(items))
	for _, item := range items {
		result = append(result, types.CityStationInfo{
			Id:              item.ID,
			Code:            item.Code,
			Name:            item.Name,
			PrimaryCategory: item.PrimaryCategory,
			Status:          item.Status,
		})
	}
	return result
}

func toResourceTypeConfigTypes(items []citylogic.ResourceTypeConfigInfo) []types.ResourceTypeConfigInfo {
	result := make([]types.ResourceTypeConfigInfo, 0, len(items))
	for _, item := range items {
		result = append(result, types.ResourceTypeConfigInfo{
			Id:               item.ID,
			Version:          item.Version,
			TypeCode:         item.TypeCode,
			TypeName:         item.TypeName,
			Direction:        item.Direction,
			DefaultValidDays: item.DefaultValidDays,
			FieldSchema:      item.FieldSchema,
			RequiredFields:   append([]string{}, item.RequiredFields...),
			FilterFields:     append([]string{}, item.FilterFields...),
			DisplayTemplate:  item.DisplayTemplate,
		})
	}
	return result
}
