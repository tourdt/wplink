package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListOperationLogsPassesFiltersToStore(t *testing.T) {
	store := &fakeOperationLogStore{
		result: model.ListOperationLogsResult{
			Items: []model.OperationLogItem{{ID: "log-1", OperatorID: "admin-1", ObjectType: "resource", Action: "approve"}},
			Page: 1, PageSize: 20, Total: 1,
		},
	}
	logic := NewOperationLogLogic(store)

	resp, err := logic.ListOperationLogs(context.Background(), OperationLogsReq{ObjectType: " resource ", OperatorID: " admin-1 "})
	if err != nil {
		t.Fatalf("ListOperationLogs() error = %v", err)
	}

	if store.filter.ObjectType != "resource" || store.filter.OperatorID != "admin-1" {
		t.Fatalf("filter = %#v, want trimmed filters", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Action != "approve" {
		t.Fatalf("resp = %#v, want operation log", resp)
	}
}

type fakeOperationLogStore struct {
	filter model.OperationLogFilter
	result model.ListOperationLogsResult
}

func (s *fakeOperationLogStore) ListOperationLogs(ctx context.Context, filter model.OperationLogFilter) (model.ListOperationLogsResult, error) {
	s.filter = filter
	return s.result, nil
}
