package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxExposureBatchSize = 50

var allowedExposureSources = map[string]struct{}{
	"home":     {},
	"list":     {},
	"search":   {},
	"topic":    {},
	"merchant": {},
}

type ResourceExposureStore interface {
	RecordResourceExposures(ctx context.Context, input model.RecordResourceExposuresInput) (int64, error)
}

type ResourceExposureItem struct {
	ResourceID        string `json:"resourceId"`
	VisibleDurationMS int64  `json:"visibleDurationMs"`
}

type RecordResourceExposuresReq struct {
	VisitorKey string                 `json:"visitorKey"`
	SessionID  string                 `json:"sessionId"`
	Source     string                 `json:"source"`
	Items      []ResourceExposureItem `json:"items"`
	UserID     string                 `json:"-"`
}

type RecordResourceExposuresResp struct {
	RecordedCount int64 `json:"recordedCount"`
}

type RecordResourceExposuresLogic struct {
	store ResourceExposureStore
}

func NewRecordResourceExposuresLogic(store ResourceExposureStore) *RecordResourceExposuresLogic {
	return &RecordResourceExposuresLogic{store: store}
}

func (l *RecordResourceExposuresLogic) RecordResourceExposures(ctx context.Context, req RecordResourceExposuresReq) (RecordResourceExposuresResp, error) {
	visitorKey := strings.TrimSpace(req.VisitorKey)
	sessionID := strings.TrimSpace(req.SessionID)
	source := strings.TrimSpace(req.Source)
	if visitorKey == "" || len(visitorKey) > 96 || sessionID == "" || len(sessionID) > 96 {
		return RecordResourceExposuresResp{}, errx.New(errx.CodeValidationFailed, "曝光会话参数无效")
	}
	if _, ok := allowedExposureSources[source]; !ok {
		return RecordResourceExposuresResp{}, errx.New(errx.CodeValidationFailed, "曝光来源无效")
	}
	if len(req.Items) == 0 || len(req.Items) > maxExposureBatchSize {
		return RecordResourceExposuresResp{}, errx.New(errx.CodeValidationFailed, "单次曝光记录数量应为 1 至 50 条")
	}

	items := make([]model.ResourceExposureItem, 0, len(req.Items))
	seen := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		resourceID := strings.TrimSpace(item.ResourceID)
		if resourceID == "" {
			continue
		}
		if _, exists := seen[resourceID]; exists {
			continue
		}
		seen[resourceID] = struct{}{}
		duration := item.VisibleDurationMS
		if duration < 800 {
			continue
		}
		if duration > 600000 {
			duration = 600000
		}
		items = append(items, model.ResourceExposureItem{
			ResourceID:        resourceID,
			VisibleDurationMS: duration,
		})
	}
	if len(items) == 0 {
		return RecordResourceExposuresResp{RecordedCount: 0}, nil
	}

	recorded, err := l.store.RecordResourceExposures(ctx, model.RecordResourceExposuresInput{
		UserID:     strings.TrimSpace(req.UserID),
		VisitorKey: visitorKey,
		SessionID:  sessionID,
		Source:     source,
		Items:      items,
	})
	if err != nil {
		logx.Errorf("批量记录资源曝光失败: userId=%s source=%s itemCount=%d err=%+v", strings.TrimSpace(req.UserID), source, len(items), err)
		return RecordResourceExposuresResp{}, errx.New(errx.CodeInternalError, "曝光记录失败，请稍后重试")
	}
	return RecordResourceExposuresResp{RecordedCount: recorded}, nil
}
