package city

import (
	"net/http"

	citylogic "wplink/backend/app/internal/logic/city"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/response"

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
		cityCode := pathvar.Vars(r)["cityCode"]
		resp, err := citylogic.NewListResourceTypesLogic(svcCtx.CityStore).ListResourceTypes(r.Context(), citylogic.ListResourceTypesReq{
			CityCode:  cityCode,
			Direction: r.URL.Query().Get("direction"),
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
			TypeCode:         item.TypeCode,
			TypeName:         item.TypeName,
			Direction:        item.Direction,
			DefaultValidDays: item.DefaultValidDays,
			FieldSchema:      item.FieldSchema,
			RequiredFields:   append([]string(nil), item.RequiredFields...),
			FilterFields:     append([]string(nil), item.FilterFields...),
			DisplayTemplate:  item.DisplayTemplate,
		})
	}
	return result
}
