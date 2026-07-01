package task

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

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

func TestResourceLifecycleTaskExpiresVerificationsAndCreatesMessages(t *testing.T) {
	store := &fakeLifecycleStore{
		expiredVerifications: []model.LifecycleResource{{ID: "verification-1", MerchantID: "merchant-1", Title: "源头工厂认证"}},
	}
	task := NewResourceLifecycleTask(store)

	resp, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if resp.VerificationExpiredCount != 1 || len(store.messages) != 1 || store.messages[0].MessageType != "verification_expired" {
		t.Fatalf("resp = %#v, messages = %#v, want expired verification message", resp, store.messages)
	}
	if store.messages[0].TargetURL != "/pages/verification/index?merchantId=merchant-1" {
		t.Fatalf("targetURL = %q, want merchant scoped verification page", store.messages[0].TargetURL)
	}
}

func TestResourceLifecycleTaskRemindsExpiringVerifications(t *testing.T) {
	store := &fakeLifecycleStore{
		expiringVerifications: []model.LifecycleResource{{ID: "verification-2", MerchantID: "merchant-2", Title: "现货档口认证"}},
	}
	task := NewResourceLifecycleTask(store)

	resp, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if resp.VerificationExpiringReminderCount != 1 || len(store.messages) != 1 || store.messages[0].TriggerType != "verification_expiring" {
		t.Fatalf("resp = %#v, messages = %#v, want expiring verification message", resp, store.messages)
	}
	if store.messages[0].TargetURL != "/pages/verification/index?merchantId=merchant-2" {
		t.Fatalf("targetURL = %q, want merchant scoped verification page", store.messages[0].TargetURL)
	}
}

type fakeLifecycleStore struct {
	expired               []model.LifecycleResource
	expiring              []model.LifecycleResource
	expiredVerifications  []model.LifecycleResource
	expiringVerifications []model.LifecycleResource
	messages              []model.CreateMessageInput
}

func (s *fakeLifecycleStore) MarkExpiredResources(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expired, nil
}

func (s *fakeLifecycleStore) ListResourcesExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expiring, nil
}

func (s *fakeLifecycleStore) MarkExpiredVerifications(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expiredVerifications, nil
}

func (s *fakeLifecycleStore) ListVerificationsExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error) {
	return s.expiringVerifications, nil
}

func (s *fakeLifecycleStore) CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error) {
	s.messages = append(s.messages, input)
	return model.CreateMessageResult{ID: "message"}, nil
}
