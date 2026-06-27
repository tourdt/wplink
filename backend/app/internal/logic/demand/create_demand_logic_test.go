package demand

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateDemandRejectsMissingRequiredFields(t *testing.T) {
	logic := NewCreateDemandLogic(&fakeDemandStore{})

	_, err := logic.CreateDemand(context.Background(), CreateDemandReq{
		CityCode: "zhili",
		Title:    "找 100-140 码女童卫衣库存",
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestCreateDemandPassesInputToStore(t *testing.T) {
	store := &fakeDemandStore{createResult: model.CreateDemandResult{ID: "demand-1", Status: "pending"}}
	logic := NewCreateDemandLogic(store)

	resp, err := logic.CreateDemand(context.Background(), CreateDemandReq{
		UserID:     "user-1",
		CityCode:   " zhili ",
		DemandType: "inventory",
		Title:      "找 100-140 码女童卫衣库存",
		Category:   "童装",
		PriceRange: model.JSONMap{"min": 10, "max": 25},
		QuantityRequirement: model.JSONMap{
			"quantity": float64(2000),
			"unit":     "件",
		},
		Attributes: model.JSONMap{"season": "春款"},
		Contact:    DemandContactReq{Name: "王老板", Phone: "13800000000", Wechat: "buyer001"},
	})
	if err != nil {
		t.Fatalf("CreateDemand() error = %v", err)
	}

	if store.createInput.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.createInput.CityCode)
	}
	if resp.ID != "demand-1" || resp.Message != "需求已提交，平台会尽快为您匹配" {
		t.Fatalf("resp = %#v, want submitted demand", resp)
	}
}

type fakeDemandStore struct {
	createInput  model.CreateDemandInput
	createResult model.CreateDemandResult
}

func (s *fakeDemandStore) CreateDemand(ctx context.Context, input model.CreateDemandInput) (model.CreateDemandResult, error) {
	s.createInput = input
	return s.createResult, nil
}
