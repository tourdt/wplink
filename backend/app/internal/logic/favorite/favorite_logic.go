package favorite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type InteractionStore interface {
	SetResourceFavorite(ctx context.Context, input model.ResourceFavoriteInput) (model.ResourceFavoriteState, error)
	ResourceBelongsToUser(ctx context.Context, userID string, resourceID string) (bool, error)
	GetResourceFavoriteState(ctx context.Context, userID string, resourceID string) (model.ResourceFavoriteState, error)
	ListFavoriteResources(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListResourcesResult, error)
	SetMerchantFollow(ctx context.Context, input model.MerchantFollowInput) (model.MerchantFollowState, error)
	GetMerchantFollowState(ctx context.Context, userID string, merchantID string) (model.MerchantFollowState, error)
	ListFollowedMerchants(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListFollowedMerchantsResult, error)
	CreateSavedSearch(ctx context.Context, input model.SavedSearchInput) (model.SavedSearchResult, error)
	ListSavedSearches(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListSavedSearchesResult, error)
	DeleteSavedSearch(ctx context.Context, userID string, savedSearchID string) error
}

type InteractionLogic struct {
	store InteractionStore
}

func NewInteractionLogic(store InteractionStore) *InteractionLogic {
	return &InteractionLogic{store: store}
}

type SetResourceFavoriteReq struct {
	ResourceID string `json:"resourceId"`
	Favorited  bool   `json:"favorited"`
}

type ResourceFavoriteStateResp struct {
	ResourceID string `json:"resourceId"`
	Favorited  bool   `json:"favorited"`
}

type SetMerchantFollowReq struct {
	MerchantID string `json:"merchantId"`
	Followed   bool   `json:"followed"`
}

type MerchantFollowStateResp struct {
	MerchantID string `json:"merchantId"`
	Followed   bool   `json:"followed"`
}

type ListInteractionReq struct {
	Page     int64
	PageSize int64
}

type FollowedMerchantItem struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	MerchantType       string   `json:"merchantType"`
	VerificationStatus string   `json:"verificationStatus"`
	MainCategories     []string `json:"mainCategories"`
	LogoUrl            string   `json:"logoUrl,omitempty"`
	FollowedAt         string   `json:"followedAt"`
}

type ListFollowedMerchantsResp struct {
	Items    []FollowedMerchantItem `json:"items"`
	Page     int64                  `json:"page"`
	PageSize int64                  `json:"pageSize"`
	Total    int64                  `json:"total"`
}

type CreateSavedSearchReq struct {
	Name         string `json:"name"`
	CityCode     string `json:"cityCode"`
	TypeCode     string `json:"typeCode"`
	Keyword      string `json:"keyword"`
	Category     string `json:"category"`
	VerifiedOnly bool   `json:"verifiedOnly"`
}

type SavedSearchItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CityCode     string `json:"cityCode"`
	TypeCode     string `json:"typeCode"`
	Keyword      string `json:"keyword"`
	Category     string `json:"category"`
	VerifiedOnly bool   `json:"verifiedOnly"`
	CreatedAt    string `json:"createdAt"`
}

type SavedSearchResp struct {
	ID string `json:"id"`
}

type ListSavedSearchesResp struct {
	Items    []SavedSearchItem `json:"items"`
	Page     int64             `json:"page"`
	PageSize int64             `json:"pageSize"`
	Total    int64             `json:"total"`
}

type DeleteSavedSearchResp struct {
	Message string `json:"message"`
}

func (l *InteractionLogic) SetResourceFavorite(ctx context.Context, userID string, req SetResourceFavoriteReq) (ResourceFavoriteStateResp, error) {
	userID = strings.TrimSpace(userID)
	resourceID := strings.TrimSpace(req.ResourceID)
	if userID == "" {
		return ResourceFavoriteStateResp{}, errx.New(errx.CodeUnauthorized, "请先登录后收藏资源")
	}
	if resourceID == "" {
		return ResourceFavoriteStateResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if req.Favorited {
		// 收藏数据代表买家意向，商家不能收藏自己管理的资源，避免污染运营和推荐信号。
		owned, err := l.store.ResourceBelongsToUser(ctx, userID, resourceID)
		if err != nil {
			return ResourceFavoriteStateResp{}, err
		}
		if owned {
			return ResourceFavoriteStateResp{}, errx.New(errx.CodeForbidden, "不能收藏自己发布的资源")
		}
	}
	state, err := l.store.SetResourceFavorite(ctx, model.ResourceFavoriteInput{UserID: userID, ResourceID: resourceID, Favorited: req.Favorited})
	if errors.Is(err, sql.ErrNoRows) {
		return ResourceFavoriteStateResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
	}
	if err != nil {
		return ResourceFavoriteStateResp{}, err
	}
	return ResourceFavoriteStateResp{ResourceID: state.ResourceID, Favorited: state.Favorited}, nil
}

func (l *InteractionLogic) GetResourceFavoriteState(ctx context.Context, userID string, resourceID string) (ResourceFavoriteStateResp, error) {
	userID = strings.TrimSpace(userID)
	resourceID = strings.TrimSpace(resourceID)
	if userID == "" {
		return ResourceFavoriteStateResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看收藏状态")
	}
	if resourceID == "" {
		return ResourceFavoriteStateResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	state, err := l.store.GetResourceFavoriteState(ctx, userID, resourceID)
	if err != nil {
		return ResourceFavoriteStateResp{}, err
	}
	return ResourceFavoriteStateResp{ResourceID: state.ResourceID, Favorited: state.Favorited}, nil
}

func (l *InteractionLogic) ListFavoriteResources(ctx context.Context, userID string, req ListInteractionReq) (resource.ListResourcesResp, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return resource.ListResourcesResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看收藏资源")
	}
	result, err := l.store.ListFavoriteResources(ctx, userID, model.ListInteractionFilter{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return resource.ListResourcesResp{}, err
	}
	return resourcesRespFromModel(result), nil
}

func (l *InteractionLogic) SetMerchantFollow(ctx context.Context, userID string, req SetMerchantFollowReq) (MerchantFollowStateResp, error) {
	userID = strings.TrimSpace(userID)
	merchantID := strings.TrimSpace(req.MerchantID)
	if userID == "" {
		return MerchantFollowStateResp{}, errx.New(errx.CodeUnauthorized, "请先登录后关注商家")
	}
	if merchantID == "" {
		return MerchantFollowStateResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}
	state, err := l.store.SetMerchantFollow(ctx, model.MerchantFollowInput{UserID: userID, MerchantID: merchantID, Followed: req.Followed})
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantFollowStateResp{}, errx.New(errx.CodeMerchantNotFound, "商家不存在或已停用")
	}
	if err != nil {
		return MerchantFollowStateResp{}, err
	}
	return MerchantFollowStateResp{MerchantID: state.MerchantID, Followed: state.Followed}, nil
}

func (l *InteractionLogic) GetMerchantFollowState(ctx context.Context, userID string, merchantID string) (MerchantFollowStateResp, error) {
	userID = strings.TrimSpace(userID)
	merchantID = strings.TrimSpace(merchantID)
	if userID == "" {
		return MerchantFollowStateResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看关注状态")
	}
	if merchantID == "" {
		return MerchantFollowStateResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}
	state, err := l.store.GetMerchantFollowState(ctx, userID, merchantID)
	if err != nil {
		return MerchantFollowStateResp{}, err
	}
	return MerchantFollowStateResp{MerchantID: state.MerchantID, Followed: state.Followed}, nil
}

func (l *InteractionLogic) ListFollowedMerchants(ctx context.Context, userID string, req ListInteractionReq) (ListFollowedMerchantsResp, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ListFollowedMerchantsResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看关注商家")
	}
	result, err := l.store.ListFollowedMerchants(ctx, userID, model.ListInteractionFilter{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return ListFollowedMerchantsResp{}, err
	}
	items := make([]FollowedMerchantItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, FollowedMerchantItem{
			ID: item.ID, Name: item.Name, MerchantType: item.MerchantType, VerificationStatus: item.VerificationStatus,
			MainCategories: append([]string(nil), item.MainCategories...), LogoUrl: item.LogoUrl, FollowedAt: item.FollowedAt,
		})
	}
	return ListFollowedMerchantsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *InteractionLogic) CreateSavedSearch(ctx context.Context, userID string, req CreateSavedSearchReq) (SavedSearchResp, error) {
	input := model.SavedSearchInput{
		UserID:       strings.TrimSpace(userID),
		Name:         strings.TrimSpace(req.Name),
		CityCode:     strings.TrimSpace(req.CityCode),
		TypeCode:     strings.TrimSpace(req.TypeCode),
		Keyword:      strings.TrimSpace(req.Keyword),
		Category:     strings.TrimSpace(req.Category),
		VerifiedOnly: req.VerifiedOnly,
	}
	if input.UserID == "" {
		return SavedSearchResp{}, errx.New(errx.CodeUnauthorized, "请先登录后保存搜索")
	}
	if input.Keyword == "" && input.TypeCode == "" && input.Category == "" && !input.VerifiedOnly {
		return SavedSearchResp{}, errx.New(errx.CodeValidationFailed, "请先输入关键词或选择筛选条件")
	}
	if input.Name == "" {
		input.Name = defaultSavedSearchName(input)
	}
	result, err := l.store.CreateSavedSearch(ctx, input)
	if err != nil {
		return SavedSearchResp{}, err
	}
	return SavedSearchResp{ID: result.ID}, nil
}

func (l *InteractionLogic) ListSavedSearches(ctx context.Context, userID string, req ListInteractionReq) (ListSavedSearchesResp, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ListSavedSearchesResp{}, errx.New(errx.CodeUnauthorized, "请先登录后查看保存的搜索")
	}
	result, err := l.store.ListSavedSearches(ctx, userID, model.ListInteractionFilter{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return ListSavedSearchesResp{}, err
	}
	items := make([]SavedSearchItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, SavedSearchItem{
			ID: item.ID, Name: item.Name, CityCode: item.CityCode, TypeCode: item.TypeCode, Keyword: item.Keyword,
			Category: item.Category, VerifiedOnly: item.VerifiedOnly, CreatedAt: item.CreatedAt,
		})
	}
	return ListSavedSearchesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *InteractionLogic) DeleteSavedSearch(ctx context.Context, userID string, savedSearchID string) (DeleteSavedSearchResp, error) {
	userID = strings.TrimSpace(userID)
	savedSearchID = strings.TrimSpace(savedSearchID)
	if userID == "" {
		return DeleteSavedSearchResp{}, errx.New(errx.CodeUnauthorized, "请先登录后删除保存的搜索")
	}
	if savedSearchID == "" {
		return DeleteSavedSearchResp{}, errx.New(errx.CodeValidationFailed, "保存的搜索不存在")
	}
	if err := l.store.DeleteSavedSearch(ctx, userID, savedSearchID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeleteSavedSearchResp{}, errx.New(errx.CodeResourceNotFound, "保存的搜索不存在或已删除")
		}
		return DeleteSavedSearchResp{}, err
	}
	return DeleteSavedSearchResp{Message: "已删除保存的搜索"}, nil
}

func defaultSavedSearchName(input model.SavedSearchInput) string {
	if input.Keyword != "" {
		return input.Keyword
	}
	if input.Category != "" {
		return input.Category
	}
	if input.TypeCode != "" {
		return input.TypeCode
	}
	return "认证资源"
}

func resourcesRespFromModel(result model.ListResourcesResult) resource.ListResourcesResp {
	items := make([]resource.ResourceListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, resource.ResourceListItem{
			ID: item.ID, TypeCode: item.TypeCode, Title: item.Title, Category: item.Category, District: item.District,
			PriceText: item.PriceText, QuantityText: item.QuantityText,
			Merchant:   resource.ResourceMerchantBrief{ID: item.Merchant.ID, Name: item.Merchant.Name, VerificationStatus: item.Merchant.VerificationStatus},
			CreditTags: append([]string(nil), item.CreditTags...), RefreshedAt: item.RefreshedAt,
		})
	}
	return resource.ListResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}
}
