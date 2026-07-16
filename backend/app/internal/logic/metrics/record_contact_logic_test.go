package metrics

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/lib/pq"
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
	if store.growthInput.EventType != model.GrowthEventResourceShareEffectiveContact || store.growthInput.EventID != "event-1" || store.growthInput.MerchantID != "merchant-1" {
		t.Fatalf("growthInput = %#v, want effective contact growth event", store.growthInput)
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

func TestRecordContactMapsStaleTokenForeignKeyToUnauthorized(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{
			ResourceID:      "resource-1",
			MerchantID:      "merchant-1",
			Status:          model.ResourceStatusPublished,
			Phone:           "18800000002",
			CommercialRules: model.DefaultCommercialRules(),
		},
		upsertUnlockErr: &pq.Error{
			Code:       "23503",
			Constraint: "resource_contact_unlocks_user_id_fkey",
		},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "stale-user", Action: "phone"})

	if err == nil || errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "登录状态无效，请重新登录" {
		t.Fatalf("RecordContact() error code=%q message=%q, want stale login unauthorized", errx.CodeOf(err), errx.PublicMessage(err))
	}
	if store.eventInput.ResourceID != "" || store.metricDelta.ContactClickCount != 0 {
		t.Fatalf("eventInput = %#v metricDelta = %#v, want no contact writes after stale token", store.eventInput, store.metricDelta)
	}
}

func TestRecordContactMapsUnlockLoadDependencyErrorToFriendlyInternalError(t *testing.T) {
	store := &fakeContactStore{
		contactErr: errors.New("pq: column rtc.commercial_rules does not exist"),
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-1", Action: "wechat"})

	if err == nil || errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "联系方式加载失败，请稍后重试" {
		t.Fatalf("RecordContact() error code=%q message=%q, want friendly contact load error", errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestRecordContactMapsContactEventForeignKeyToUnauthorized(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{
			ResourceID:      "resource-1",
			MerchantID:      "merchant-1",
			Status:          model.ResourceStatusPublished,
			Wechat:          "stock-demo",
			CommercialRules: model.DefaultCommercialRules(),
		},
		unlockState: model.ContactUnlockState{Unlocked: true},
		eventErr: &pq.Error{
			Code:       "23503",
			Constraint: "resource_contact_events_user_id_fkey",
		},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "stale-user", Action: "wechat"})

	if err == nil || errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "登录状态无效，请重新登录" {
		t.Fatalf("RecordContact() error code=%q message=%q, want stale login unauthorized", errx.CodeOf(err), errx.PublicMessage(err))
	}
	if store.metricDelta.ContactClickCount != 0 {
		t.Fatalf("metricDelta = %#v, want no metric after stale token event failure", store.metricDelta)
	}
}

func TestRecordContactRequiresPaymentForPaidCategory(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{
			ResourceID: "resource-1",
			MerchantID: "merchant-1",
			Status:     model.ResourceStatusPublished,
			Phone:      "18800000002",
			CommercialRules: model.JSONMap{
				"contactUnlock": model.JSONMap{
					"mode":             model.ContactUnlockModePaidOrVIP,
					"priceCent":        int64(500),
					"currency":         "CNY",
					"repeatUnlockDays": int64(30),
				},
			},
		},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-1", Action: "phone"})

	if err == nil || errx.CodeOf(err) != errx.CodePaymentRequired {
		t.Fatalf("RecordContact() error = %v, want payment required", err)
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
	if store.growthInput.EventType != "" {
		t.Fatalf("growthInput = %#v, want no reward for raw share action", store.growthInput)
	}
}

func TestRecordContactRecordsShareViewGrowthEvent(t *testing.T) {
	store := &fakeContactStore{eventResult: model.ResourceContactEventResult{ID: "event-2", MerchantID: "merchant-1"}}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-2", Action: "share_view"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}

	if store.growthInput.EventType != model.GrowthEventResourceShareEffectiveView || store.growthInput.EventID != "event-2" {
		t.Fatalf("growthInput = %#v, want share view growth event", store.growthInput)
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
	contactErr          error
	manageErr           error
	unlockState         model.ContactUnlockState
	unlockErr           error
	vipManagedMerchant  model.VIPManagedMerchant
	vipErr              error
	unlockInput         model.ContactUnlockInput
	upsertUnlockErr     error
	eventInput          model.ResourceContactEventInput
	eventResult         model.ResourceContactEventResult
	eventErr            error
	metricDelta         model.ResourceMetricDelta
	metricErr           error
	growthInput         model.GrowthEventInput
	userManagedMerchant bool
}

func (s *fakeContactStore) GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (model.ResourceContactUnlockInfo, error) {
	if s.contactErr != nil {
		return model.ResourceContactUnlockInfo{}, s.contactErr
	}
	s.contact.ResourceID = resourceID
	return s.contact, nil
}

func (s *fakeContactStore) UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error) {
	if s.manageErr != nil {
		return false, s.manageErr
	}
	return s.userManagedMerchant, nil
}

func (s *fakeContactStore) HasActiveContactUnlock(ctx context.Context, resourceID string, userID string) (model.ContactUnlockState, error) {
	if s.unlockErr != nil {
		return model.ContactUnlockState{}, s.unlockErr
	}
	return s.unlockState, nil
}

func (s *fakeContactStore) FindActiveVIPManagedMerchant(ctx context.Context, userID string) (model.VIPManagedMerchant, error) {
	if s.vipErr != nil {
		return model.VIPManagedMerchant{}, s.vipErr
	}
	return s.vipManagedMerchant, nil
}

func (s *fakeContactStore) UpsertContactUnlock(ctx context.Context, input model.ContactUnlockInput) (model.ContactUnlockResult, error) {
	if s.upsertUnlockErr != nil {
		return model.ContactUnlockResult{}, s.upsertUnlockErr
	}
	s.unlockInput = input
	return model.ContactUnlockResult{ID: "unlock-1", ResourceID: input.ResourceID, UserID: input.UserID, SourceType: input.SourceType, ExpiresAt: input.ExpiresAt}, nil
}

func (s *fakeContactStore) RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error) {
	if s.eventErr != nil {
		return model.ResourceContactEventResult{}, s.eventErr
	}
	s.eventInput = input
	return s.eventResult, nil
}

func (s *fakeContactStore) UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error {
	if s.metricErr != nil {
		return s.metricErr
	}
	s.metricDelta = delta
	return nil
}

func (s *fakeContactStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInput = input
	return nil, nil
}
