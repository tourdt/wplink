package merchant

import (
	"net/http"

	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListMerchantDirectoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMerchantDirectoryStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMerchantDirectoryReq
		if err := httpx.Parse(r, &req); err != nil {
			logx.WithContext(r.Context()).Errorw("解析商家目录筛选参数失败", logx.Field("operation", "parse_merchant_directory_filter"), logx.Field("err", err))
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "商家目录筛选参数格式不正确"))
			return
		}
		resp, err := merchantlogic.NewMerchantDirectoryLogic(store).ListDirectory(r.Context(), merchantlogic.ListMerchantDirectoryReq{
			CityCode: req.CityCode, Keyword: req.Keyword, Categories: req.Categories,
			Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ListMerchantDirectoryResp{
			Items: mapMerchantDirectoryItems(resp.Items), Total: resp.Total, Page: resp.Page, PageSize: resp.PageSize,
		}, nil)
	}
}

func requireMerchantDirectoryStore(r *http.Request, svcCtx *svc.ServiceContext) (merchantlogic.MerchantDirectoryStore, error) {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantDirectoryModel == nil {
		logx.WithContext(r.Context()).Errorw("商家目录 Handler 存储依赖未配置", logx.Field("operation", "require_merchant_directory_store"))
		return nil, errx.New(errx.CodeInternalError, "商家列表加载失败，请稍后重试")
	}
	return svcCtx.APIStore.MerchantDirectoryModel, nil
}

func mapMerchantDirectoryItems(items []merchantlogic.MerchantDirectoryPublicItem) []types.MerchantDirectoryItem {
	result := make([]types.MerchantDirectoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, mapMerchantDirectoryItem(item))
	}
	return result
}

func mapMerchantDirectoryItem(item merchantlogic.MerchantDirectoryPublicItem) types.MerchantDirectoryItem {
	// 显式逐字段映射让公开边界可审查，避免未来内部 Model 新增联系方式后被整体序列化。
	return types.MerchantDirectoryItem{
		MerchantId: item.MerchantID, MerchantNo: item.MerchantNo, Name: item.Name, MerchantType: item.MerchantType,
		MainCategories: append([]string{}, item.MainCategories...), LogoUrl: item.LogoURL, AddressText: item.AddressText,
		MarketName: item.MarketName, BuildingName: item.BuildingName, FloorNo: item.FloorNo, BoothNo: item.BoothNo,
		Latitude: item.Latitude, Longitude: item.Longitude, LocationName: item.LocationName,
		LastResourceActiveAt: item.LastResourceActiveAt, DistanceMeters: item.DistanceMeters,
	}
}
