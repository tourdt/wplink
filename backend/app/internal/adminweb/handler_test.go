package adminweb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandlerServesStaticAssetUnderAdminBase(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html":         {Data: []byte("<html>admin</html>")},
		"assets/app-test.js": {Data: []byte("console.log('admin')")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/app-test.js", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('admin')" {
		t.Fatalf("body = %q, want asset content", rec.Body.String())
	}
}

func TestHandlerFallsBackToIndexForAdminHistoryRoute(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html": {Data: []byte("<html>admin shell</html>")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/resources/pending", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "<html>admin shell</html>" {
		t.Fatalf("body = %q, want index fallback", rec.Body.String())
	}
}

func TestHandlerReturnsNotFoundForMissingAsset(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html": {Data: []byte("<html>admin shell</html>")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/missing.js", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
