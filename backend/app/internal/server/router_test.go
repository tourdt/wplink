package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterReturnsHealthz(t *testing.T) {
	router := NewRouter(http.NotFoundHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}

func TestRouterMountsAdminHandler(t *testing.T) {
	router := NewRouter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want admin handler status", rec.Code)
	}
}

func TestRouterReportsApiHandlersNotConnected(t *testing.T) {
	router := NewRouter(http.NotFoundHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q, want json", rec.Header().Get("Content-Type"))
	}
}
