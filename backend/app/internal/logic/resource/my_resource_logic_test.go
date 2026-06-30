package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListMyResourcesPassesFiltersToStore(t *testing.T) {
	store := &fakeMyResourceStore{
		listResult: model.ListMyResourcesResult{
			Items: []model.MyResourceItem{{
				ID: "resource-1", Title: "童装库存", Status: "rejected", RejectReason: "图片不清晰",
				CoverURL: "https://img.example.com/resource-cover.jpg",
				Metrics:  model.MyResourceMetrics{ExposureCount: 10, DetailViewCount: 3, PhoneClickCount: 2, WechatCopyCount: 1},
			}},
			Page: 1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListMyResourcesLogic(store)

	resp, err := logic.ListMyResources(context.Background(), ListMyResourcesReq{MerchantID: " merchant-1 ", Status: "published", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMyResources() error = %v", err)
	}

	if store.listFilter.MerchantID != "merchant-1" || store.listFilter.Status != "published" {
		t.Fatalf("filter = %#v, want trimmed merchant and status", store.listFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Metrics.DetailViewCount != 3 || resp.Items[0].RejectReason != "图片不清晰" {
		t.Fatalf("resp = %#v, want metrics item with reject reason", resp)
	}
	if resp.Items[0].CoverURL != "https://img.example.com/resource-cover.jpg" {
		t.Fatalf("coverUrl = %q, want list cover image", resp.Items[0].CoverURL)
	}
}

func TestGetOwnResourceReturnsUnpublishedDetail(t *testing.T) {
	store := &fakeMyResourceStore{
		ownDetail: model.ResourceDetail{
			ID: "resource-1", Status: model.ResourceStatusPending, TypeCode: "inventory", Title: "待审核童装库存",
			Category: "童装卫衣", Description: "可拿样", PriceText: "18元/件", QuantityText: "3800件",
			Attributes: model.JSONMap{"season": "春季"}, MerchantID: "merchant-1", MerchantName: "织里云仓",
			MerchantVerificationStatus: "verified", ContactName: "周经理", PhoneMasked: "18800000002", WechatMasked: "stock-demo",
			Tags: []string{"春款"}, Images: []string{"https://img.example.com/resource.jpg"},
		},
	}
	logic := NewGetOwnResourceLogic(store)

	resp, err := logic.Get(context.Background(), GetOwnResourceReq{MerchantID: " merchant-1 ", ResourceID: " resource-1 "})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if store.ownDetailMerchantID != "merchant-1" || store.ownDetailResourceID != "resource-1" {
		t.Fatalf("store args = %q/%q, want trimmed merchant/resource", store.ownDetailMerchantID, store.ownDetailResourceID)
	}
	if resp.Status != model.ResourceStatusPending || resp.Title != "待审核童装库存" {
		t.Fatalf("resp = %#v, want pending own resource detail", resp)
	}
	if len(resp.Tags) != 1 || resp.Tags[0] != "春款" || len(resp.Images) != 1 || resp.Images[0] != "https://img.example.com/resource.jpg" {
		t.Fatalf("tags/images = %#v/%#v, want repost defaults", resp.Tags, resp.Images)
	}
}

func TestRefreshResourceRejectsPendingStatus(t *testing.T) {
	store := &fakeMyResourceStore{resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPending}}
	logic := NewRefreshResourceLogic(store)

	_, err := logic.RefreshResource(context.Background(), RefreshResourceReq{MerchantID: "merchant-1", ResourceID: "resource-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("RefreshResource() error = %v, want state conflict", err)
	}
}

func TestRefreshResourceUpdatesPublishedResource(t *testing.T) {
	store := &fakeMyResourceStore{
		resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished},
		refreshResult:  model.RefreshResourceResult{ID: "resource-1", RefreshedAt: "2026-06-27T10:00:00Z", RemainingRefreshQuota: 9},
	}
	logic := NewRefreshResourceLogic(store)

	resp, err := logic.RefreshResource(context.Background(), RefreshResourceReq{MerchantID: " merchant-1 ", ResourceID: " resource-1 "})
	if err != nil {
		t.Fatalf("RefreshResource() error = %v", err)
	}

	if store.refreshedResourceID != "resource-1" || resp.RemainingRefreshQuota != 9 {
		t.Fatalf("refreshedResourceID = %q, resp = %#v", store.refreshedResourceID, resp)
	}
}

func TestMarkDealtRequiresPublishedResource(t *testing.T) {
	store := &fakeMyResourceStore{resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusRejected}}
	logic := NewMarkDealtLogic(store)

	_, err := logic.MarkDealt(context.Background(), MarkDealtReq{MerchantID: "merchant-1", ResourceID: "resource-1", IsDealt: true})
	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("MarkDealt() error = %v, want state conflict", err)
	}
}

func TestTakeDownRequiresReason(t *testing.T) {
	logic := NewTakeDownOwnResourceLogic(&fakeMyResourceStore{})

	_, err := logic.TakeDown(context.Background(), TakeDownOwnResourceReq{MerchantID: "merchant-1", ResourceID: "resource-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("TakeDown() error = %v, want validation error", err)
	}
}

func TestDeleteTakenDownRejectsPublishedResource(t *testing.T) {
	store := &fakeMyResourceStore{resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished}}
	logic := NewDeleteTakenDownResourceLogic(store)

	_, err := logic.Delete(context.Background(), DeleteTakenDownResourceReq{MerchantID: "merchant-1", ResourceID: "resource-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("Delete() error = %v, want state conflict", err)
	}
}

func TestDeleteTakenDownSoftDeletesTakenDownResource(t *testing.T) {
	store := &fakeMyResourceStore{
		resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusTakenDown},
		deleteResult:   model.DeleteTakenDownResourceResult{ID: "resource-1", Status: model.ResourceStatusTakenDown},
	}
	logic := NewDeleteTakenDownResourceLogic(store)

	resp, err := logic.Delete(context.Background(), DeleteTakenDownResourceReq{MerchantID: " merchant-1 ", ResourceID: " resource-1 "})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if store.deletedResourceID != "resource-1" || resp.Message != "资源已删除" {
		t.Fatalf("deletedResourceID = %q, resp = %#v", store.deletedResourceID, resp)
	}
}

func TestRepostSimilarCreatesDraftFromExpiredOrDealt(t *testing.T) {
	store := &fakeMyResourceStore{
		resourceStatus: model.ResourceOwnershipStatus{ID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, IsExpired: true},
		repostResult:   model.RepostSimilarResult{ID: "resource-copy", Status: model.ResourceStatusDraft},
	}
	logic := NewRepostSimilarLogic(store)

	resp, err := logic.RepostSimilar(context.Background(), RepostSimilarReq{MerchantID: "merchant-1", ResourceID: "resource-1"})
	if err != nil {
		t.Fatalf("RepostSimilar() error = %v", err)
	}

	if store.repostResourceID != "resource-1" || resp.Status != model.ResourceStatusDraft {
		t.Fatalf("repostResourceID = %q, resp = %#v", store.repostResourceID, resp)
	}
}

type fakeMyResourceStore struct {
	listFilter          model.ListMyResourcesFilter
	listResult          model.ListMyResourcesResult
	resourceStatus      model.ResourceOwnershipStatus
	editableDetail      model.EditableResourceDetail
	ownDetail           model.ResourceDetail
	ownDetailMerchantID string
	ownDetailResourceID string
	refreshedResourceID string
	refreshResult       model.RefreshResourceResult
	dealInput           model.MarkDealtInput
	dealResult          model.DealFeedbackResult
	takeDownInput       model.TakeDownOwnResourceInput
	takeDownResult      model.TakeDownOwnResourceResult
	deletedResourceID   string
	deleteResult        model.DeleteTakenDownResourceResult
	repostResourceID    string
	repostResult        model.RepostSimilarResult
}

func (s *fakeMyResourceStore) ListMyResources(ctx context.Context, filter model.ListMyResourcesFilter) (model.ListMyResourcesResult, error) {
	s.listFilter = filter
	return s.listResult, nil
}

func (s *fakeMyResourceStore) GetResourceOwnershipStatus(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	return s.resourceStatus, nil
}

func (s *fakeMyResourceStore) GetEditableResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.EditableResourceDetail, error) {
	return s.editableDetail, nil
}

func (s *fakeMyResourceStore) GetOwnResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.ResourceDetail, error) {
	s.ownDetailMerchantID = merchantID
	s.ownDetailResourceID = resourceID
	return s.ownDetail, nil
}

func (s *fakeMyResourceStore) RefreshResource(ctx context.Context, merchantID string, resourceID string) (model.RefreshResourceResult, error) {
	s.refreshedResourceID = resourceID
	return s.refreshResult, nil
}

func (s *fakeMyResourceStore) MarkDealt(ctx context.Context, input model.MarkDealtInput) (model.DealFeedbackResult, error) {
	s.dealInput = input
	return s.dealResult, nil
}

func (s *fakeMyResourceStore) TakeDownOwnResource(ctx context.Context, input model.TakeDownOwnResourceInput) (model.TakeDownOwnResourceResult, error) {
	s.takeDownInput = input
	return s.takeDownResult, nil
}

func (s *fakeMyResourceStore) DeleteTakenDownResource(ctx context.Context, merchantID string, resourceID string) (model.DeleteTakenDownResourceResult, error) {
	s.deletedResourceID = resourceID
	return s.deleteResult, nil
}

func (s *fakeMyResourceStore) RepostSimilar(ctx context.Context, merchantID string, resourceID string) (model.RepostSimilarResult, error) {
	s.repostResourceID = resourceID
	return s.repostResult, nil
}
