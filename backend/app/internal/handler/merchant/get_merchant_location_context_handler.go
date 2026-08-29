package merchant

import (
	"net/http"

	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func GetMerchantLocationContextHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMerchantDirectoryStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := merchantlogic.NewMerchantDirectoryLogic(store).GetLocationContext(r.Context(), pathvar.Vars(r)["merchantId"])
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.MerchantDirectoryLocationContextResp{
			Current:         mapMerchantDirectoryItem(resp.Current),
			Nearby:          mapMerchantDirectoryItems(resp.Nearby),
			RadiusMeters:    resp.RadiusMeters,
			NearbyAvailable: resp.NearbyAvailable,
		}, nil)
	}
}
