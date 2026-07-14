package message

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type Store interface {
	ListMessages(ctx context.Context, filter model.ListMessagesFilter) (model.ListMessagesResult, error)
	ReadMessage(ctx context.Context, userID string, roleCodes []string, messageID string) (model.ReadMessageResult, error)
}

type ListMessagesReq struct {
	UserID    string
	RoleCode  string
	RoleCodes []string
	Type      string
	Status    string
	Page      int64
	PageSize  int64
}

type MessageListItem struct {
	ID          string `json:"id"`
	MessageType string `json:"messageType"`
	TriggerID   string `json:"triggerId,omitempty"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	TargetURL   string `json:"targetUrl,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
}

type ListMessagesResp struct {
	Items    []MessageListItem `json:"items"`
	Page     int64             `json:"page"`
	PageSize int64             `json:"pageSize"`
	Total    int64             `json:"total"`
}

type ReadMessageReq struct {
	UserID    string
	RoleCode  string
	RoleCodes []string
	MessageID string
}

type ReadMessageResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ListMessagesLogic struct {
	store Store
}

func NewListMessagesLogic(store Store) *ListMessagesLogic {
	return &ListMessagesLogic{store: store}
}

func (l *ListMessagesLogic) ListMessages(ctx context.Context, req ListMessagesReq) (ListMessagesResp, error) {
	roleCodes := normalizeRoleCodes(req.RoleCode, req.RoleCodes)
	filter := model.ListMessagesFilter{
		UserID:    strings.TrimSpace(req.UserID),
		RoleCode:  firstRoleCode(roleCodes),
		RoleCodes: roleCodes,
		Type:      strings.TrimSpace(req.Type),
		Status:    strings.TrimSpace(req.Status),
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
	if filter.UserID == "" && len(filter.RoleCodes) == 0 {
		return ListMessagesResp{}, errx.New(errx.CodeValidationFailed, "请先登录后查看消息")
	}
	result, err := l.store.ListMessages(ctx, filter)
	if err != nil {
		return ListMessagesResp{}, err
	}
	items := make([]MessageListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MessageListItem{
			ID: item.ID, MessageType: item.MessageType, TriggerID: item.TriggerID, Title: item.Title, Content: item.Content,
			TargetURL: item.TargetURL, Status: item.Status, CreatedAt: item.CreatedAt,
		})
	}
	return ListMessagesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

type ReadMessageLogic struct {
	store Store
}

func NewReadMessageLogic(store Store) *ReadMessageLogic {
	return &ReadMessageLogic{store: store}
}

func (l *ReadMessageLogic) ReadMessage(ctx context.Context, req ReadMessageReq) (ReadMessageResp, error) {
	userID := strings.TrimSpace(req.UserID)
	roleCodes := normalizeRoleCodes(req.RoleCode, req.RoleCodes)
	messageID := strings.TrimSpace(req.MessageID)
	if (userID == "" && len(roleCodes) == 0) || messageID == "" {
		return ReadMessageResp{}, errx.New(errx.CodeValidationFailed, "消息不存在")
	}
	result, err := l.store.ReadMessage(ctx, userID, roleCodes, messageID)
	if err != nil {
		return ReadMessageResp{}, err
	}
	return ReadMessageResp{ID: result.ID, Status: result.Status}, nil
}

func normalizeRoleCodes(primary string, values []string) []string {
	seen := map[string]struct{}{}
	roleCodes := make([]string, 0, len(values)+1)
	for _, value := range append([]string{primary}, values...) {
		roleCode := strings.TrimSpace(value)
		if roleCode == "" {
			continue
		}
		if _, ok := seen[roleCode]; ok {
			continue
		}
		seen[roleCode] = struct{}{}
		roleCodes = append(roleCodes, roleCode)
	}
	return roleCodes
}

func firstRoleCode(roleCodes []string) string {
	if len(roleCodes) == 1 {
		return roleCodes[0]
	}
	return ""
}
