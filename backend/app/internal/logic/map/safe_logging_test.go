package maplogic

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestGeneratedMapLogicDependencyFailuresDoNotLogRawSensitiveErrors(t *testing.T) {
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

	const sensitiveError = "postgres password=map-secret Authorization=Bearer-map-token coordinates=30.1,120.2"
	dependencyErr := errors.New(sensitiveError)
	_, _ = NewAdminLogic(&failingAdminMapLogStore{err: dependencyErr}).ListScenes(context.Background(), ListAdminScenesReq{})
	_, _ = NewPublicLogic(&failingPublicMapLogStore{err: dependencyErr}).ListScenes(context.Background(), ListScenesReq{})
	_, _ = NewBindingLogic(&failingBindingMapLogStore{err: dependencyErr}).GetStatus(context.Background(), "merchant-safe-1")
	_, _ = NewReportLogic(&fakeMapReportStore{err: dependencyErr}).Submit(context.Background(), "object-safe-1", model.MapObjectReportKindRiskReport, SubmitMapObjectReportReq{
		ReporterUserID: "user-safe-1",
		ReasonCode:     "false_information",
	})

	logs := logBuffer.String()
	for _, forbidden := range []string{sensitiveError, "map-secret", "Bearer-map-token", "30.1,120.2"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("log = %q, must not contain raw map dependency detail %q", logs, forbidden)
		}
	}
	for _, operation := range []string{"list_admin_scenes", "list_public_scenes", "get_binding_status", "create_map_object_report"} {
		if !strings.Contains(logs, `"operation":"`+operation+`"`) {
			t.Fatalf("log = %q, want operation %q", logs, operation)
		}
	}
	for _, field := range []string{
		`"errorCategory":"unknown"`, `"merchantId":"merchant-safe-1"`,
		`"objectId":"object-safe-1"`, `"reporterUserId":"user-safe-1"`,
	} {
		if !strings.Contains(logs, field) {
			t.Fatalf("log = %q, want safe diagnostic field %s", logs, field)
		}
	}
}

type failingAdminMapLogStore struct {
	fakeAdminMapStore
	err error
}

func (s *failingAdminMapLogStore) ListAdminScenes(context.Context, model.ListMapScenesFilter) ([]model.MapScene, error) {
	return nil, s.err
}

type failingPublicMapLogStore struct {
	fakePublicMapStore
	err error
}

func (s *failingPublicMapLogStore) ListPublishedScenes(context.Context, model.ListMapScenesFilter) ([]model.MapScene, error) {
	return nil, s.err
}

type failingBindingMapLogStore struct {
	fakeBindingStore
	err error
}

func (s *failingBindingMapLogStore) GetMapBindingStatus(context.Context, string) (model.MapBindingStatus, error) {
	return model.MapBindingStatus{}, s.err
}
