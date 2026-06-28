package city

import (
	"net/http"

	citylogic "wplink/backend/app/internal/logic/city"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func ListCityStationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := citylogic.NewListCityStationsLogic(svcCtx.CityStore).ListCityStations(r.Context())
		response.JSON(w, resp, err)
	}
}

func ListCityResourceTypesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cityCode := pathvar.Vars(r)["cityCode"]
		resp, err := citylogic.NewListResourceTypesLogic(svcCtx.CityStore).ListResourceTypes(r.Context(), cityCode)
		response.JSON(w, resp, err)
	}
}
