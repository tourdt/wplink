package demand

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListMyDemandsRejectsEmptyUser(t *testing.T) {
	logic := NewListMyDemandsLogic(&fakeMyDemandStore{})

	_, err := logic.ListMyDemands(context.Background(), "", ListMyDemandsReq{})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestListMyDemandsPassesUserAndPaging(t *testing.T) {
	store := &fakeMyDemandStore{
		result: model.ListDemandsResult{
			Items: []model.DemandListItem{{ID: "demand-1", Title: "找童装库存", Status: "matching", CreatedAt: "2026-06-27T10:00:00+08:00"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListMyDemandsLogic(store)

	resp, err := logic.ListMyDemands(context.Background(), " user-1 ", ListMyDemandsReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMyDemands() error = %v", err)
	}

	if store.userID != "user-1" {
		t.Fatalf("userID = %q, want trimmed user-1", store.userID)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "demand-1" {
		t.Fatalf("items = %#v, want demand item", resp.Items)
	}
}

type fakeMyDemandStore struct {
	userID string
	filter model.ListDemandsFilter
	result model.ListDemandsResult
}

func (s *fakeMyDemandStore) ListMyDemands(ctx context.Context, userID string, filter model.ListDemandsFilter) (model.ListDemandsResult, error) {
	s.userID = userID
	s.filter = filter
	return s.result, nil
}
