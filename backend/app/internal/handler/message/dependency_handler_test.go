package message

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessageHandlersRejectNilServiceContextWithoutPanic(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		target  string
		body    string
	}{
		{name: "list", handler: ListMessagesHandler(nil), method: http.MethodGet, target: "/api/v1/messages"},
		{name: "read", handler: ReadMessageHandler(nil), method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer user-token")
			tc.handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s, want 500", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "消息服务暂不可用，请稍后重试") {
				t.Fatalf("body=%s, want safe dependency error", rec.Body.String())
			}
		})
	}
}
