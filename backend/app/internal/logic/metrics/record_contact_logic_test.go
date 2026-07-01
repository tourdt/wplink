package metrics

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestRecordContactRejectsUnsupportedAction(t *testing.T) {
	logic := NewRecordContactLogic(&fakeContactStore{})

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "email"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("RecordContact() error = %v, want validation error", err)
	}
}

func TestRecordContactRecordsPhoneEventAndMetric(t *testing.T) {
	store := &fakeContactStore{
		contact:     model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Phone: "18800000002"},
		eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"},
	}
	logic := NewRecordContactLogic(store)

	resp, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: " resource-1 ", UserID: " user-1 ", Action: "phone"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}

	if store.eventInput.ResourceID != "resource-1" || store.eventInput.Action != "phone" || store.eventInput.UserID != "user-1" {
		t.Fatalf("eventInput = %#v, want trimmed phone event", store.eventInput)
	}
	if store.metricDelta.ContactClickCount != 1 || store.metricDelta.PhoneClickCount != 1 {
		t.Fatalf("metricDelta = %#v, want phone contact counters", store.metricDelta)
	}
	if resp.Phone != "18800000002" || resp.Action != "phone" || resp.Message != "电话已解锁" {
		t.Fatalf("resp = %#v, want unlocked phone", resp)
	}
}

func TestRecordContactReturnsWechatOnlyAfterSuccessfulUnlock(t *testing.T) {
	store := &fakeContactStore{
		contact:     model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Wechat: "stock-demo"},
		eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"},
	}
	logic := NewRecordContactLogic(store)

	resp, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: " resource-1 ", UserID: " user-1 ", Action: "wechat"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}
	if resp.Wechat != "stock-demo" || resp.Action != "wechat" || resp.Message != "微信号已解锁" {
		t.Fatalf("resp = %#v, want unlocked wechat", resp)
	}
	if store.metricDelta.WechatCopyCount != 1 || store.metricDelta.ContactClickCount != 1 {
		t.Fatalf("metricDelta = %#v, want wechat contact metric", store.metricDelta)
	}
}

func TestRecordContactRequiresLoginForWechatUnlock(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Wechat: "stock-demo"},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "wechat"})
	if err == nil || errx.CodeOf(err) != errx.CodeUnauthorized {
		t.Fatalf("RecordContact() error = %v, want unauthorized", err)
	}
	if store.eventInput.ResourceID != "" || store.metricDelta.ContactClickCount != 0 {
		t.Fatalf("eventInput = %#v metricDelta = %#v, want no writes", store.eventInput, store.metricDelta)
	}
}

func TestRecordContactRejectsWechatWithoutPersistingMetricWhenWechatMissing(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-1", Action: "wechat"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("RecordContact() error = %v, want validation failed", err)
	}
	if store.eventInput.ResourceID != "" || store.metricDelta.ContactClickCount != 0 {
		t.Fatalf("eventInput = %#v metricDelta = %#v, want no writes", store.eventInput, store.metricDelta)
	}
}

func TestRecordContactReturnsOwnResourceContactWithoutPersistingMetric(t *testing.T) {
	tests := []struct {
		name        string
		action      string
		contact     model.ResourceContactUnlockInfo
		wantPhone   string
		wantWechat  string
		wantMessage string
	}{
		{
			name:        "phone",
			action:      "phone",
			contact:     model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Phone: "18800000002"},
			wantPhone:   "18800000002",
			wantMessage: "电话已解锁",
		},
		{
			name:        "wechat",
			action:      "wechat",
			contact:     model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Wechat: "stock-demo"},
			wantWechat:  "stock-demo",
			wantMessage: "微信号已解锁",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeContactStore{contact: tt.contact, userManagedMerchant: true}
			logic := NewRecordContactLogic(store)

			resp, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-1", Action: tt.action})
			if err != nil {
				t.Fatalf("RecordContact() error = %v", err)
			}
			if resp.Phone != tt.wantPhone || resp.Wechat != tt.wantWechat || resp.Message != tt.wantMessage {
				t.Fatalf("resp = %#v, want phone=%q wechat=%q message=%q", resp, tt.wantPhone, tt.wantWechat, tt.wantMessage)
			}
			if store.eventInput.ResourceID != "" || store.metricDelta.ContactClickCount != 0 {
				t.Fatalf("eventInput = %#v metricDelta = %#v, want no writes", store.eventInput, store.metricDelta)
			}
		})
	}
}

func TestRecordContactRecordsShareMetric(t *testing.T) {
	store := &fakeContactStore{eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"}}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "share"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}

	if store.metricDelta.ContactClickCount != 1 || store.metricDelta.ShareCount != 1 {
		t.Fatalf("metricDelta = %#v, want share counters", store.metricDelta)
	}
}

func TestRecordContactAcceptsMerchantProfileAlias(t *testing.T) {
	store := &fakeContactStore{eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"}}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "merchant_profile"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v, want merchant_profile accepted", err)
	}
	if store.metricDelta.ContactClickCount != 1 {
		t.Fatalf("metricDelta = %#v, want contact counter", store.metricDelta)
	}
}

type fakeContactStore struct {
	contact             model.ResourceContactUnlockInfo
	eventInput          model.ResourceContactEventInput
	eventResult         model.ResourceContactEventResult
	metricDelta         model.ResourceMetricDelta
	userManagedMerchant bool
}

func (s *fakeContactStore) GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (model.ResourceContactUnlockInfo, error) {
	s.contact.ResourceID = resourceID
	return s.contact, nil
}

func (s *fakeContactStore) UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error) {
	return s.userManagedMerchant, nil
}

func (s *fakeContactStore) RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error) {
	s.eventInput = input
	return s.eventResult, nil
}

func (s *fakeContactStore) UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error {
	s.metricDelta = delta
	return nil
}
