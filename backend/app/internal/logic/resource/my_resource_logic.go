package resource

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type MyResourceStore interface {
	ListMyResources(ctx context.Context, filter model.ListMyResourcesFilter) (model.ListMyResourcesResult, error)
	GetResourceOwnershipStatus(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error)
	GetEditableResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.EditableResourceDetail, error)
	GetOwnResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.ResourceDetail, error)
	RefreshResource(ctx context.Context, merchantID string, resourceID string) (model.RefreshResourceResult, error)
	MarkDealt(ctx context.Context, input model.MarkDealtInput) (model.DealFeedbackResult, error)
	TakeDownOwnResource(ctx context.Context, input model.TakeDownOwnResourceInput) (model.TakeDownOwnResourceResult, error)
	DeleteTakenDownResource(ctx context.Context, merchantID string, resourceID string) (model.DeleteTakenDownResourceResult, error)
	RepostSimilar(ctx context.Context, merchantID string, resourceID string) (model.RepostSimilarResult, error)
}

type ListMyResourcesReq struct {
	MerchantID string
	Status     string
	Page       int64
	PageSize   int64
}

type MyResourceMetrics struct {
	ExposureCount   int64 `json:"exposureCount"`
	DetailViewCount int64 `json:"detailViewCount"`
	PhoneClickCount int64 `json:"phoneClickCount"`
	WechatCopyCount int64 `json:"wechatCopyCount"`
}

type MyResourceItem struct {
	ID           string            `json:"id"`
	TypeCode     string            `json:"typeCode"`
	Title        string            `json:"title"`
	Category     string            `json:"category"`
	Status       string            `json:"status"`
	RejectReason string            `json:"rejectReason,omitempty"`
	PublishedAt  string            `json:"publishedAt,omitempty"`
	ExpiresAt    string            `json:"expiresAt,omitempty"`
	DealtAt      string            `json:"dealtAt,omitempty"`
	Metrics      MyResourceMetrics `json:"metrics"`
}

type ListMyResourcesResp struct {
	Items    []MyResourceItem `json:"items"`
	Page     int64            `json:"page"`
	PageSize int64            `json:"pageSize"`
	Total    int64            `json:"total"`
}

type EditableResourceContact struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Wechat string `json:"wechat,omitempty"`
}

type GetEditableResourceReq struct {
	MerchantID string
	ResourceID string
}

type GetOwnResourceReq struct {
	MerchantID string
	ResourceID string
}

type GetEditableResourceResp struct {
	ID           string                  `json:"id"`
	MerchantID   string                  `json:"merchantId"`
	CityCode     string                  `json:"cityCode"`
	TypeCode     string                  `json:"typeCode"`
	Status       string                  `json:"status"`
	Title        string                  `json:"title"`
	Category     string                  `json:"category"`
	District     string                  `json:"district,omitempty"`
	PriceText    string                  `json:"priceText,omitempty"`
	QuantityText string                  `json:"quantityText,omitempty"`
	Description  string                  `json:"description"`
	Attributes   model.JSONMap           `json:"attributes"`
	Tags         []string                `json:"tags"`
	Images       []string                `json:"images"`
	Contact      EditableResourceContact `json:"contact"`
	RejectReason string                  `json:"rejectReason,omitempty"`
}

type RefreshResourceReq struct {
	MerchantID string
	ResourceID string
}

type RefreshResourceResp struct {
	ID                    string `json:"id"`
	RefreshedAt           string `json:"refreshedAt"`
	RemainingRefreshQuota int64  `json:"remainingRefreshQuota"`
}

type MarkDealtReq struct {
	MerchantID              string
	ResourceID              string
	IsDealt                 bool
	IsReal                  bool
	ResponseTimely          bool
	WillingToCooperateAgain bool
	Note                    string
}

type DealFeedbackResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TakeDownOwnResourceReq struct {
	MerchantID string
	ResourceID string
	Reason     string
}

type TakeDownOwnResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type DeleteTakenDownResourceReq struct {
	MerchantID string
	ResourceID string
}

type DeleteTakenDownResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RepostSimilarReq struct {
	MerchantID string
	ResourceID string
}

type RepostSimilarResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ListMyResourcesLogic struct {
	store MyResourceStore
}

func NewListMyResourcesLogic(store MyResourceStore) *ListMyResourcesLogic {
	return &ListMyResourcesLogic{store: store}
}

func (l *ListMyResourcesLogic) ListMyResources(ctx context.Context, req ListMyResourcesReq) (ListMyResourcesResp, error) {
	merchantID := strings.TrimSpace(req.MerchantID)
	if merchantID == "" {
		return ListMyResourcesResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	result, err := l.store.ListMyResources(ctx, model.ListMyResourcesFilter{
		MerchantID: merchantID,
		Status:     strings.TrimSpace(req.Status),
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ListMyResourcesResp{}, err
	}
	items := make([]MyResourceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MyResourceItem{
			ID: item.ID, TypeCode: item.TypeCode, Title: item.Title, Category: item.Category, Status: item.Status,
			RejectReason: item.RejectReason, PublishedAt: item.PublishedAt, ExpiresAt: item.ExpiresAt, DealtAt: item.DealtAt,
			Metrics: MyResourceMetrics{
				ExposureCount: item.Metrics.ExposureCount, DetailViewCount: item.Metrics.DetailViewCount,
				PhoneClickCount: item.Metrics.PhoneClickCount, WechatCopyCount: item.Metrics.WechatCopyCount,
			},
		})
	}
	return ListMyResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

type GetEditableResourceLogic struct {
	store MyResourceStore
}

func NewGetEditableResourceLogic(store MyResourceStore) *GetEditableResourceLogic {
	return &GetEditableResourceLogic{store: store}
}

func (l *GetEditableResourceLogic) Get(ctx context.Context, req GetEditableResourceReq) (GetEditableResourceResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	if merchantID == "" || resourceID == "" {
		return GetEditableResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	detail, err := l.store.GetEditableResourceDetail(ctx, merchantID, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetEditableResourceResp{}, errx.New(errx.CodeStateConflict, "资源不存在或当前状态不可编辑")
		}
		return GetEditableResourceResp{}, err
	}
	return GetEditableResourceResp{
		ID:           detail.ID,
		MerchantID:   detail.MerchantID,
		CityCode:     detail.CityCode,
		TypeCode:     detail.TypeCode,
		Status:       detail.Status,
		Title:        detail.Title,
		Category:     detail.Category,
		District:     detail.District,
		PriceText:    detail.PriceText,
		QuantityText: detail.QuantityText,
		Description:  detail.Description,
		Attributes:   detail.Attributes,
		Tags:         append([]string(nil), detail.Tags...),
		Images:       append([]string(nil), detail.Images...),
		Contact: EditableResourceContact{
			Name: detail.ContactName, Phone: detail.ContactPhone, Wechat: detail.ContactWechat,
		},
		RejectReason: detail.RejectReason,
	}, nil
}

type GetOwnResourceLogic struct {
	store MyResourceStore
}

func NewGetOwnResourceLogic(store MyResourceStore) *GetOwnResourceLogic {
	return &GetOwnResourceLogic{store: store}
}

func (l *GetOwnResourceLogic) Get(ctx context.Context, req GetOwnResourceReq) (ResourceDetailResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	if merchantID == "" || resourceID == "" {
		return ResourceDetailResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	detail, err := l.store.GetOwnResourceDetail(ctx, merchantID, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResourceDetailResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
		}
		return ResourceDetailResp{}, err
	}
	return resourceDetailRespFromModel(detail), nil
}

type RefreshResourceLogic struct {
	store MyResourceStore
}

func NewRefreshResourceLogic(store MyResourceStore) *RefreshResourceLogic {
	return &RefreshResourceLogic{store: store}
}

func (l *RefreshResourceLogic) RefreshResource(ctx context.Context, req RefreshResourceReq) (RefreshResourceResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	status, err := l.requirePublished(ctx, merchantID, resourceID)
	if err != nil {
		return RefreshResourceResp{}, err
	}
	if status.IsExpired {
		return RefreshResourceResp{}, errx.New(errx.CodeStateConflict, "资源已过期，请再发类似资源")
	}
	result, err := l.store.RefreshResource(ctx, merchantID, resourceID)
	if err != nil {
		return RefreshResourceResp{}, err
	}
	return RefreshResourceResp{ID: result.ID, RefreshedAt: result.RefreshedAt, RemainingRefreshQuota: result.RemainingRefreshQuota}, nil
}

type MarkDealtLogic struct {
	store MyResourceStore
}

func NewMarkDealtLogic(store MyResourceStore) *MarkDealtLogic {
	return &MarkDealtLogic{store: store}
}

func (l *MarkDealtLogic) MarkDealt(ctx context.Context, req MarkDealtReq) (DealFeedbackResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	if _, err := l.requirePublished(ctx, merchantID, resourceID); err != nil {
		return DealFeedbackResp{}, err
	}
	result, err := l.store.MarkDealt(ctx, model.MarkDealtInput{
		MerchantID: merchantID, ResourceID: resourceID, IsDealt: req.IsDealt,
		IsReal: req.IsReal, ResponseTimely: req.ResponseTimely,
		WillingToCooperateAgain: req.WillingToCooperateAgain, Note: strings.TrimSpace(req.Note),
	})
	if err != nil {
		return DealFeedbackResp{}, err
	}
	return DealFeedbackResp{ID: result.ID, Status: result.Status, Message: "成交反馈已记录"}, nil
}

type TakeDownOwnResourceLogic struct {
	store MyResourceStore
}

func NewTakeDownOwnResourceLogic(store MyResourceStore) *TakeDownOwnResourceLogic {
	return &TakeDownOwnResourceLogic{store: store}
}

func (l *TakeDownOwnResourceLogic) TakeDown(ctx context.Context, req TakeDownOwnResourceReq) (TakeDownOwnResourceResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return TakeDownOwnResourceResp{}, errx.New(errx.CodeValidationFailed, "请填写下架原因")
	}
	if _, err := l.requirePublished(ctx, merchantID, resourceID); err != nil {
		return TakeDownOwnResourceResp{}, err
	}
	result, err := l.store.TakeDownOwnResource(ctx, model.TakeDownOwnResourceInput{MerchantID: merchantID, ResourceID: resourceID, Reason: reason})
	if err != nil {
		return TakeDownOwnResourceResp{}, err
	}
	return TakeDownOwnResourceResp{ID: result.ID, Status: result.Status, Message: "资源已下架"}, nil
}

type DeleteTakenDownResourceLogic struct {
	store MyResourceStore
}

func NewDeleteTakenDownResourceLogic(store MyResourceStore) *DeleteTakenDownResourceLogic {
	return &DeleteTakenDownResourceLogic{store: store}
}

func (l *DeleteTakenDownResourceLogic) Delete(ctx context.Context, req DeleteTakenDownResourceReq) (DeleteTakenDownResourceResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	if merchantID == "" || resourceID == "" {
		return DeleteTakenDownResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在")
	}
	status, err := l.store.GetResourceOwnershipStatus(ctx, merchantID, resourceID)
	if err != nil {
		return DeleteTakenDownResourceResp{}, err
	}
	// 删除只对已下架资源开放，避免误删正在审核或公开展示的资源。
	if status.Status != model.ResourceStatusTakenDown {
		return DeleteTakenDownResourceResp{}, errx.New(errx.CodeStateConflict, "仅已下架资源可以删除")
	}
	result, err := l.store.DeleteTakenDownResource(ctx, merchantID, resourceID)
	if err != nil {
		return DeleteTakenDownResourceResp{}, err
	}
	return DeleteTakenDownResourceResp{ID: result.ID, Status: result.Status, Message: "资源已删除"}, nil
}

type RepostSimilarLogic struct {
	store MyResourceStore
}

func NewRepostSimilarLogic(store MyResourceStore) *RepostSimilarLogic {
	return &RepostSimilarLogic{store: store}
}

func (l *RepostSimilarLogic) RepostSimilar(ctx context.Context, req RepostSimilarReq) (RepostSimilarResp, error) {
	merchantID, resourceID := trimMerchantResource(req.MerchantID, req.ResourceID)
	status, err := l.store.GetResourceOwnershipStatus(ctx, merchantID, resourceID)
	if err != nil {
		return RepostSimilarResp{}, err
	}
	if !status.IsExpired && !status.IsDealt {
		return RepostSimilarResp{}, errx.New(errx.CodeStateConflict, "仅已过期或已成交资源可以再发类似")
	}
	result, err := l.store.RepostSimilar(ctx, merchantID, resourceID)
	if err != nil {
		return RepostSimilarResp{}, err
	}
	return RepostSimilarResp{ID: result.ID, Status: result.Status, Message: "已复制为草稿"}, nil
}

func (l *RefreshResourceLogic) requirePublished(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	return requirePublishedResource(ctx, l.store, merchantID, resourceID)
}

func (l *MarkDealtLogic) requirePublished(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	return requirePublishedResource(ctx, l.store, merchantID, resourceID)
}

func (l *TakeDownOwnResourceLogic) requirePublished(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	return requirePublishedResource(ctx, l.store, merchantID, resourceID)
}

func requirePublishedResource(ctx context.Context, store MyResourceStore, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	if merchantID == "" || resourceID == "" {
		return model.ResourceOwnershipStatus{}, errx.New(errx.CodeValidationFailed, "资源不存在")
	}
	status, err := store.GetResourceOwnershipStatus(ctx, merchantID, resourceID)
	if err != nil {
		return model.ResourceOwnershipStatus{}, err
	}
	if status.Status != model.ResourceStatusPublished {
		return model.ResourceOwnershipStatus{}, errx.New(errx.CodeStateConflict, "当前资源状态不能执行该操作")
	}
	return status, nil
}

func trimMerchantResource(merchantID string, resourceID string) (string, string) {
	return strings.TrimSpace(merchantID), strings.TrimSpace(resourceID)
}
