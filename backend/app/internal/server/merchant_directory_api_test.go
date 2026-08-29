package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/app/internal/svc"
)

func TestMerchantDirectoryRoutesUseGeneratedHandlers(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{})
	for _, target := range []string{
		"/api/v1/merchant-directory?cityCode=zhili",
		"/api/v1/merchants/801/location-context",
	} {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
			body := decodeEnvelope(t, recorder, http.StatusInternalServerError)
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s still uses generated stub: %#v", target, body)
			}
		})
	}
}
