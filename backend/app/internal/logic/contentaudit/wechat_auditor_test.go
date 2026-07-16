package contentaudit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	resourcelogic "wplink/backend/app/internal/logic/resource"
)

func TestWechatAuditorReturnsRiskyTextDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg":
			if r.URL.Query().Get("access_token") != "token-1" {
				t.Fatalf("access_token = %q, want token-1", r.URL.Query().Get("access_token"))
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode msg body: %v", err)
			}
			if body["openid"] != "openid-1" || !strings.Contains(body["content"].(string), "女童春款卫衣库存整包清") {
				t.Fatalf("body = %#v, want openid and content", body)
			}
			writeJSON(t, w, map[string]any{
				"errcode":  0,
				"trace_id": "trace-text",
				"result":   map[string]any{"suggest": "risky", "label": 20006},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, RequestTimeout: time.Second, MaxTextChars: 2500},
		"",
		server.Client(),
	).WithURLs(server.URL+"/token", server.URL+"/msg", "")

	result, err := auditor.AuditResource(context.Background(), resourcelogic.ContentAuditInput{
		OpenID:      "openid-1",
		MerchantID:  "merchant-1",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Description: "整包优先，可现场看货。",
	})
	if err != nil {
		t.Fatalf("AuditResource() error = %v", err)
	}
	if result.Decision != resourcelogic.ContentAuditDecisionRisky {
		t.Fatalf("decision = %q, want risky", result.Decision)
	}
	if len(result.Labels) == 0 || result.Labels[0] != "20006" {
		t.Fatalf("labels = %#v, want label 20006", result.Labels)
	}
}

func TestWechatAuditorSubmitsImageCheckWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg":
			writeJSON(t, w, map[string]any{
				"errcode":  0,
				"trace_id": "trace-text",
				"result":   map[string]any{"suggest": "pass", "label": 100},
			})
		case "/media":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode media body: %v", err)
			}
			if body["media_type"] != float64(2) || body["media_url"] != "https://cdn.example.com/a.png" {
				t.Fatalf("media body = %#v, want image media payload", body)
			}
			writeJSON(t, w, map[string]any{"errcode": 0, "trace_id": "trace-media"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, MediaEnabled: true, MediaScene: 3, RequestTimeout: time.Second, MaxTextChars: 2500},
		"https://cdn.example.com",
		server.Client(),
	).WithURLs(server.URL+"/token", server.URL+"/msg", server.URL+"/media")

	result, err := auditor.AuditResource(context.Background(), resourcelogic.ContentAuditInput{
		OpenID:      "openid-1",
		MerchantID:  "merchant-1",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Description: "整包优先，可现场看货。",
		Images:      []string{"/a.png"},
	})
	if err != nil {
		t.Fatalf("AuditResource() error = %v", err)
	}
	if result.Decision != resourcelogic.ContentAuditDecisionPass {
		t.Fatalf("decision = %q, want pass", result.Decision)
	}
	if len(result.TraceIDs) != 2 || result.TraceIDs[1] != "trace-media" {
		t.Fatalf("traceIDs = %#v, want text and media traces", result.TraceIDs)
	}
	if len(result.MediaTasks) != 1 || result.MediaTasks[0].TraceID != "trace-media" || result.MediaTasks[0].MediaURL != "https://cdn.example.com/a.png" {
		t.Fatalf("mediaTasks = %#v, want persisted image audit task", result.MediaTasks)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
