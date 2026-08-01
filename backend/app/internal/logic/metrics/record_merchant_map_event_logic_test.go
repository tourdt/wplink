package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestRecordMerchantMapEventAcceptsSupportedEvents(t *testing.T) {
	tests := []struct {
		name             string
		eventType        string
		targetMerchantID string
	}{
		{name: "位置入口点击", eventType: "location_entry_click"},
		{name: "位置页浏览", eventType: "location_view"},
		{name: "导航点击", eventType: "navigation_click"},
		{name: "周边抽屉打开", eventType: "nearby_drawer_open"},
		{name: "周边标记点击", eventType: "nearby_marker_click", targetMerchantID: "target-1"},
		{name: "周边商家点击", eventType: "nearby_merchant_click", targetMerchantID: "target-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMerchantMapEventStore{}
			logic := NewRecordMerchantMapEventLogic(store)

			resp, err := logic.RecordMerchantMapEvent(context.Background(), RecordMerchantMapEventReq{
				UserID:           " user-1 ",
				MerchantID:       " merchant-1 ",
				TargetMerchantID: " " + tt.targetMerchantID + " ",
				VisitorKey:       " visitor-1 ",
				SessionID:        " session-1 ",
				EventType:        tt.eventType,
				Source:           "merchant_location",
			})
			if err != nil {
				t.Fatalf("RecordMerchantMapEvent() error = %v", err)
			}
			if !resp.Recorded {
				t.Fatal("Recorded = false, want true")
			}
			want := model.MerchantMapEventInput{
				UserID:           "user-1",
				MerchantID:       "merchant-1",
				TargetMerchantID: tt.targetMerchantID,
				VisitorKey:       "visitor-1",
				SessionID:        "session-1",
				EventType:        tt.eventType,
				Source:           "merchant_location",
			}
			if store.input != want {
				t.Fatalf("store input = %#v, want %#v", store.input, want)
			}
		})
	}
}

func TestRecordMerchantMapEventRejectsUnknownEventType(t *testing.T) {
	store := &fakeMerchantMapEventStore{}
	logic := NewRecordMerchantMapEventLogic(store)

	_, err := logic.RecordMerchantMapEvent(context.Background(), validMerchantMapEventReq("merchant_open"))

	assertMerchantMapEventError(t, err, errx.CodeValidationFailed, "地图行为类型无效")
	if store.called {
		t.Fatal("store called for an unsupported event type")
	}
}

func TestRecordMerchantMapEventRejectsUnknownSource(t *testing.T) {
	store := &fakeMerchantMapEventStore{}
	logic := NewRecordMerchantMapEventLogic(store)
	req := validMerchantMapEventReq("location_view")
	req.Source = "home"

	_, err := logic.RecordMerchantMapEvent(context.Background(), req)

	assertMerchantMapEventError(t, err, errx.CodeValidationFailed, "地图行为来源无效")
	if store.called {
		t.Fatal("store called for an unsupported source")
	}
}

func TestRecordMerchantMapEventRejectsInvalidVisitorOrSession(t *testing.T) {
	tests := []struct {
		name       string
		visitorKey string
		sessionID  string
	}{
		{name: "访客为空", visitorKey: "", sessionID: "session-1"},
		{name: "会话为空", visitorKey: "visitor-1", sessionID: ""},
		{name: "访客超长", visitorKey: strings.Repeat("v", 97), sessionID: "session-1"},
		{name: "会话超长", visitorKey: "visitor-1", sessionID: strings.Repeat("s", 97)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMerchantMapEventStore{}
			logic := NewRecordMerchantMapEventLogic(store)
			req := validMerchantMapEventReq("location_view")
			req.VisitorKey = tt.visitorKey
			req.SessionID = tt.sessionID

			_, err := logic.RecordMerchantMapEvent(context.Background(), req)

			assertMerchantMapEventError(t, err, errx.CodeValidationFailed, "地图行为会话参数无效")
			if store.called {
				t.Fatal("store called for invalid visitor/session parameters")
			}
		})
	}
}

func TestRecordMerchantMapEventRequiresTargetForNearbyClicks(t *testing.T) {
	for _, eventType := range []string{"nearby_marker_click", "nearby_merchant_click"} {
		t.Run(eventType, func(t *testing.T) {
			store := &fakeMerchantMapEventStore{}
			logic := NewRecordMerchantMapEventLogic(store)

			_, err := logic.RecordMerchantMapEvent(context.Background(), validMerchantMapEventReq(eventType))

			assertMerchantMapEventError(t, err, errx.CodeValidationFailed, "地图行为会话参数无效")
			if store.called {
				t.Fatal("store called for a nearby click without targetMerchantId")
			}
		})
	}
}

func TestRecordMerchantMapEventHidesDatabaseError(t *testing.T) {
	store := &fakeMerchantMapEventStore{err: errors.New("pq: relation merchant_map_events does not exist")}
	logic := NewRecordMerchantMapEventLogic(store)

	_, err := logic.RecordMerchantMapEvent(context.Background(), validMerchantMapEventReq("location_view"))

	assertMerchantMapEventError(t, err, errx.CodeInternalError, "地图行为记录失败，请稍后重试")
	if strings.Contains(errx.PublicMessage(err), "merchant_map_events") || strings.Contains(errx.PublicMessage(err), "pq:") {
		t.Fatalf("public error leaked database detail: %q", errx.PublicMessage(err))
	}
}

func validMerchantMapEventReq(eventType string) RecordMerchantMapEventReq {
	return RecordMerchantMapEventReq{
		MerchantID: "merchant-1",
		VisitorKey: "visitor-1",
		SessionID:  "session-1",
		EventType:  eventType,
		Source:     "merchant_location",
	}
}

func assertMerchantMapEventError(t *testing.T, err error, code string, message string) {
	t.Helper()
	if errx.CodeOf(err) != code || errx.PublicMessage(err) != message {
		t.Fatalf("error code=%q message=%q, want code=%q message=%q", errx.CodeOf(err), errx.PublicMessage(err), code, message)
	}
}

type fakeMerchantMapEventStore struct {
	called bool
	input  model.MerchantMapEventInput
	err    error
}

func (s *fakeMerchantMapEventStore) RecordMerchantMapEvent(ctx context.Context, input model.MerchantMapEventInput) error {
	s.called = true
	s.input = input
	return s.err
}
