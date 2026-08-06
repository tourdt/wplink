package growth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestPublicGrowthHandlerLogsFailureWithoutSensitiveDetails(t *testing.T) {
	const sensitive = "Authorization=Bearer admin-secret token=user-secret"
	var logBuffer bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logBuffer))
	t.Cleanup(func() {
		if currentWriter := logx.Reset(); currentWriter != nil {
			_ = currentWriter.Close()
		}
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/growth-campaigns/active", nil)
	listActiveGrowthCampaignsHTTPHandler(failingPublicGrowthStore{err: errors.New(sensitive)})(rec, req)

	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), errx.CodeInternalError) || strings.Contains(rec.Body.String(), sensitive) {
		t.Fatalf("status=%d body=%q, want safe internal error", rec.Code, rec.Body.String())
	}
	logText := logBuffer.String()
	if !strings.Contains(logText, "查询公开增长活动失败") || !strings.Contains(logText, `"errorType":"*errors.errorString"`) {
		t.Fatalf("log=%q, want operation and safe error type", logText)
	}
	for _, forbidden := range []string{sensitive, "Authorization", "admin-secret", "user-secret"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("log contains sensitive detail %q: %q", forbidden, logText)
		}
	}
}

type failingPublicGrowthStore struct {
	err error
}

func (s failingPublicGrowthStore) ListActiveGrowthCampaigns(context.Context) ([]model.PublicGrowthCampaign, error) {
	return nil, s.err
}
