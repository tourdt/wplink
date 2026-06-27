package message

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListMessagesRequiresRecipient(t *testing.T) {
	logic := NewListMessagesLogic(&fakeMessageStore{})

	_, err := logic.ListMessages(context.Background(), ListMessagesReq{})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ListMessages() error = %v, want validation error", err)
	}
}

func TestListMessagesPassesFiltersToStore(t *testing.T) {
	store := &fakeMessageStore{
		listResult: model.ListMessagesResult{
			Items: []model.MessageItem{{ID: "message-1", MessageType: "resource_review", Title: "审核通过", Status: "unread"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListMessagesLogic(store)

	resp, err := logic.ListMessages(context.Background(), ListMessagesReq{UserID: " user-1 ", Type: " resource_review ", Status: " unread "})
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if store.filter.UserID != "user-1" || store.filter.Type != "resource_review" || store.filter.Status != "unread" {
		t.Fatalf("filter = %#v, want trimmed filters", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "审核通过" {
		t.Fatalf("resp = %#v, want message item", resp)
	}
}

func TestReadMessageRequiresUserAndMessage(t *testing.T) {
	logic := NewReadMessageLogic(&fakeMessageStore{})

	_, err := logic.ReadMessage(context.Background(), ReadMessageReq{UserID: "user-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ReadMessage() error = %v, want validation error", err)
	}
}

func TestReadMessagePassesIDsToStore(t *testing.T) {
	store := &fakeMessageStore{readResult: model.ReadMessageResult{ID: "message-1", Status: "read"}}
	logic := NewReadMessageLogic(store)

	resp, err := logic.ReadMessage(context.Background(), ReadMessageReq{UserID: " user-1 ", MessageID: " message-1 "})
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	if store.readUserID != "user-1" || store.readMessageID != "message-1" || resp.Status != "read" {
		t.Fatalf("readUserID = %q, readMessageID = %q, resp = %#v", store.readUserID, store.readMessageID, resp)
	}
}

type fakeMessageStore struct {
	filter        model.ListMessagesFilter
	listResult    model.ListMessagesResult
	readUserID    string
	readMessageID string
	readResult    model.ReadMessageResult
}

func (s *fakeMessageStore) ListMessages(ctx context.Context, filter model.ListMessagesFilter) (model.ListMessagesResult, error) {
	s.filter = filter
	return s.listResult, nil
}

func (s *fakeMessageStore) ReadMessage(ctx context.Context, userID string, messageID string) (model.ReadMessageResult, error) {
	s.readUserID = userID
	s.readMessageID = messageID
	return s.readResult, nil
}
