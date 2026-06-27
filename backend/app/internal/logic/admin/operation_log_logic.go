package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type OperationLogStore interface {
	ListOperationLogs(ctx context.Context, filter model.OperationLogFilter) (model.ListOperationLogsResult, error)
}

type OperationLogsReq struct {
	ObjectType string
	ObjectID   string
	OperatorID string
	Page       int64
	PageSize   int64
}

type OperationLogItem struct {
	ID           string
	OperatorID   string
	OperatorRole string
	ObjectType   string
	ObjectID     string
	Action       string
	Reason       string
	CreatedAt    string
}

type OperationLogsResp struct {
	Items    []OperationLogItem
	Page     int64
	PageSize int64
	Total    int64
}

type OperationLogLogic struct {
	store OperationLogStore
}

func NewOperationLogLogic(store OperationLogStore) *OperationLogLogic {
	return &OperationLogLogic{store: store}
}

func (l *OperationLogLogic) ListOperationLogs(ctx context.Context, req OperationLogsReq) (OperationLogsResp, error) {
	result, err := l.store.ListOperationLogs(ctx, model.OperationLogFilter{
		ObjectType: strings.TrimSpace(req.ObjectType),
		ObjectID:   strings.TrimSpace(req.ObjectID),
		OperatorID: strings.TrimSpace(req.OperatorID),
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return OperationLogsResp{}, err
	}
	items := make([]OperationLogItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, OperationLogItem{
			ID:           item.ID,
			OperatorID:   item.OperatorID,
			OperatorRole: item.OperatorRole,
			ObjectType:   item.ObjectType,
			ObjectID:     item.ObjectID,
			Action:       item.Action,
			Reason:       item.Reason,
			CreatedAt:    item.CreatedAt,
		})
	}
	return OperationLogsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
