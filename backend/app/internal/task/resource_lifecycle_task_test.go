package task

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestResourceLifecycleTaskPreservesStableFailureStage(t *testing.T) {
	sentinel := errors.New("password=db-secret Authorization=Bearer-secret")
	tests := []struct {
		name  string
		store *fakeLifecycleStore
		want  string
	}{
		{name: "mark expired", store: &fakeLifecycleStore{markExpiredErr: sentinel}, want: "mark_expired_resources"},
		{name: "create expired message", store: &fakeLifecycleStore{
			expired: []model.LifecycleResource{{ID: "resource-expired", MerchantID: "merchant-1"}}, messageErr: sentinel,
		}, want: "create_expired_message"},
		{name: "list expiring", store: &fakeLifecycleStore{listExpiringErr: sentinel}, want: "list_expiring_resources"},
		{name: "create expiring message", store: &fakeLifecycleStore{
			expiring: []model.LifecycleResource{{ID: "resource-expiring", MerchantID: "merchant-1"}}, messageErr: sentinel,
		}, want: "create_expiring_message"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewResourceLifecycleTask(tc.store).Run(context.Background())
			if !errors.Is(err, sentinel) {
				t.Fatalf("Run() error=%v, want preserved sentinel", err)
			}
			if got := ResourceLifecycleErrorStage(err); got != tc.want {
				t.Fatalf("ResourceLifecycleErrorStage()=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourceLifecycleTaskExpiresResourcesAndCreatesMessages(t *testing.T) {
	store := &fakeLifecycleStore{
		expired: []model.LifecycleResource{{ID: "resource-1", MerchantID: "merchant-1", Title: "童装库存"}},
	}
	task := NewResourceLifecycleTask(store)

	resp, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if resp.ExpiredCount != 1 || len(store.messages) != 1 || store.messages[0].MessageType != "resource_expired" {
		t.Fatalf("resp = %#v, messages = %#v", resp, store.messages)
	}
	if store.messages[0].TargetURL != "/pages/my-resources/index?merchantId=merchant-1" {
		t.Fatalf("targetURL = %q, want merchant scoped my resources page", store.messages[0].TargetURL)
	}
}

func TestResourceLifecycleTaskRemindsExpiringResources(t *testing.T) {
	store := &fakeLifecycleStore{
		expiring: []model.LifecycleResource{{ID: "resource-2", MerchantID: "merchant-2", Title: "即将过期资源"}},
	}
	task := NewResourceLifecycleTask(store)

	resp, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if resp.ExpiringReminderCount != 1 || store.messages[0].TriggerType != "resource_expiring" {
		t.Fatalf("resp = %#v, messages = %#v", resp, store.messages)
	}
	if store.messages[0].TargetURL != "/pages/my-resources/index?merchantId=merchant-2" {
		t.Fatalf("targetURL = %q, want merchant scoped my resources page", store.messages[0].TargetURL)
	}
}

func TestResourceLifecycleTaskDoesNotCountDuplicateMessage(t *testing.T) {
	store := &fakeLifecycleStore{
		expiring:         []model.LifecycleResource{{ID: "resource-2", MerchantID: "merchant-2", Title: "即将过期资源"}},
		messageDuplicate: true,
	}
	task := NewResourceLifecycleTask(store)

	resp, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.ExpiringReminderCount != 0 {
		t.Fatalf("ExpiringReminderCount = %d, want duplicate delivery not counted", resp.ExpiringReminderCount)
	}
}

type fakeLifecycleStore struct {
	expired          []model.LifecycleResource
	expiring         []model.LifecycleResource
	messages         []model.CreateMessageInput
	messageDuplicate bool
	markExpiredErr   error
	listExpiringErr  error
	messageErr       error
}

func (s *fakeLifecycleStore) MarkExpiredResources(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expired, s.markExpiredErr
}

func (s *fakeLifecycleStore) ListResourcesExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expiring, s.listExpiringErr
}

func (s *fakeLifecycleStore) CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error) {
	s.messages = append(s.messages, input)
	if s.messageErr != nil {
		return model.CreateMessageResult{}, s.messageErr
	}
	return model.CreateMessageResult{ID: "message", Created: !s.messageDuplicate}, nil
}
