package handlerx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotMigratedReturnsSafeInternalError(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/private?token=secret", nil)

	NotMigrated("PrivateHandler")(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var body struct {
		Code      int    `json:"code"`
		ErrorCode string `json:"errorCode"`
		Message   string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != http.StatusInternalServerError {
		t.Fatalf("body code = %d, want %d", body.Code, http.StatusInternalServerError)
	}
	if body.ErrorCode != "INTERNAL_ERROR" {
		t.Fatalf("error code = %q, want INTERNAL_ERROR", body.ErrorCode)
	}
	if body.Message != "接口暂不可用，请稍后重试" {
		t.Fatalf("message = %q, want safe unavailable message", body.Message)
	}
}
