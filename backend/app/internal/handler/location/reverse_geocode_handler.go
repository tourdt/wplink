package location

import (
	"net/http"

	locationlogic "wplink/backend/app/internal/logic/location"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ReverseGeocodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ReverseGeocodeReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "位置参数格式不正确"))
			return
		}
		resp, err := locationlogic.NewLogic(svcCtx.LocationGeocoder).ReverseGeocode(r.Context(), locationlogic.ReverseGeocodeReq{
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ReverseGeocodeResp{
			Address:  resp.Address,
			Name:     resp.Name,
			Province: resp.Province,
		}, nil)
	}
}
