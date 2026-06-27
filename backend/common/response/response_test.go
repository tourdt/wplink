package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/common/errx"
)

func TestJSONWritesSuccessEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, map[string]string{"id": "resource-1"}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["code"] != float64(200) {
		t.Fatalf("code = %#v, want 200", body["code"])
	}
	if body["msg"] != "ok" {
		t.Fatalf("msg = %#v, want ok", body["msg"])
	}
}

func TestJSONMapsBusinessErrorToFriendlyMessage(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, nil, errx.New(errx.CodeForbidden, "您没有权限进行此操作"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["msg"] != "您没有权限进行此操作" {
		t.Fatalf("msg = %#v, want friendly forbidden message", body["msg"])
	}
}
