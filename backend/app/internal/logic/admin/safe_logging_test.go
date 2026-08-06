package admin

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestSafeAdminErrorCategoryUsesStableCausesWithoutReadingErrorText(t *testing.T) {
	const sensitiveDetail = "password=db-secret Authorization=Bearer-secret requestBody={full-config}"
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{
		{name: "timeout", ctx: context.Background(), err: fmt.Errorf("%s: %w", sensitiveDetail, context.DeadlineExceeded), want: "timeout"},
		{name: "canceled", ctx: context.Background(), err: fmt.Errorf("%s: %w", sensitiveDetail, context.Canceled), want: "canceled"},
		{name: "unavailable", ctx: context.Background(), err: fmt.Errorf("%s: %w", sensitiveDetail, driver.ErrBadConn), want: "unavailable"},
		{name: "conflict", ctx: context.Background(), err: fmt.Errorf("%s: %w", sensitiveDetail, model.ErrAdminOperatorLoginNameExists), want: "conflict"},
		{name: "curated conflict", ctx: context.Background(), err: errx.New(errx.CodeStateConflict, sensitiveDetail), want: "conflict"},
		{name: "unknown", ctx: context.Background(), err: errors.New(sensitiveDetail), want: "unknown"},
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	tests = append(tests, struct {
		name string
		ctx  context.Context
		err  error
		want string
	}{name: "context cancellation", ctx: canceledCtx, err: errors.New(sensitiveDetail), want: "canceled"})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SafeAdminErrorCategory(tc.ctx, tc.err); got != tc.want {
				t.Fatalf("SafeAdminErrorCategory()=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestAdminLogicDependencyFailuresDoNotLogRawSensitiveErrors(t *testing.T) {
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

	const sensitiveError = "database password=admin-secret Authorization=Bearer secret-token requestBody={vipRules:full}"
	store := &failingAdminSafeLogStore{err: errors.New(sensitiveError)}
	actor := AdminPermissionActor{OperatorID: "admin-safe-log", Roles: []string{permission.RoleSuperAdmin}}
	_, _ = NewAdminPermissionLogic(store, fakeAdminPasswordHasher{}).ListOperators(context.Background(), ListAdminOperatorsReq{}, actor)
	_, _ = NewReviewResourceLogic(store).ReviewResource(context.Background(), "resource-1", ReviewResourceReq{
		Action: "reject", Reason: "资料不完整", ReviewerID: actor.OperatorID,
	})
	_, _ = NewResourceReportLogic(store).ListResourceReports(context.Background(), ListResourceReportsReq{})
	_, _ = NewVIPConfigAdminLogic(store).ListVIPPlans(context.Background())
	_, _ = NewResourceTypeConfigLogic(store).CreateResourceTypeConfig(context.Background(), CreateResourceTypeConfigReq{
		CityCode: "zhili", TypeCode: "safe_type", TypeName: "安全类型", Direction: "supply",
		GroupCode: "safe_group", GroupName: "安全分组", DefaultValidDays: 7,
	})

	logText := logBuffer.String()
	for _, secret := range []string{sensitiveError, "admin-secret", "secret-token", "vipRules:full", "Authorization=Bearer"} {
		if strings.Contains(logText, secret) {
			t.Fatalf("log=%q, must not contain sensitive dependency text %q", logText, secret)
		}
	}
	for _, operation := range []string{
		"list_admin_operators", "review_resource", "list_resource_reports", "list_vip_plans", "create_resource_type_config",
	} {
		if !strings.Contains(logText, `"operation":"`+operation+`"`) {
			t.Fatalf("log=%q, want operation %q", logText, operation)
		}
	}
	for _, field := range []string{
		`"operatorId":"admin-safe-log"`, `"resourceId":"resource-1"`, `"errorCategory":"unknown"`, `"errorType":"*errors.errorString"`,
	} {
		if !strings.Contains(logText, field) {
			t.Fatalf("log=%q, want safe diagnostic field %q", logText, field)
		}
	}
}

type failingAdminSafeLogStore struct {
	err error
}

func (s *failingAdminSafeLogStore) ListAdminOperators(context.Context, model.AdminOperatorFilter) (model.ListAdminOperatorsResult, error) {
	return model.ListAdminOperatorsResult{}, s.err
}

func (s *failingAdminSafeLogStore) CreateAdminOperator(context.Context, model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	return model.AdminOperatorItem{}, s.err
}

func (s *failingAdminSafeLogStore) UpdateAdminOperator(context.Context, model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	return model.AdminOperatorItem{}, s.err
}

func (s *failingAdminSafeLogStore) UpdateAdminOperatorStatus(context.Context, model.AdminOperatorStatusInput) (model.AdminOperatorItem, error) {
	return model.AdminOperatorItem{}, s.err
}

func (s *failingAdminSafeLogStore) GetAdminRoleModulePermissions(context.Context, string) (model.AdminRoleModulePermission, error) {
	return model.AdminRoleModulePermission{}, s.err
}

func (s *failingAdminSafeLogStore) UpdateAdminRoleModulePermissions(context.Context, model.AdminRoleModulePermissionInput) (model.AdminRoleModulePermission, error) {
	return model.AdminRoleModulePermission{}, s.err
}

func (s *failingAdminSafeLogStore) ReviewResource(context.Context, string, model.ReviewResourceInput) (model.ReviewResourceResult, error) {
	return model.ReviewResourceResult{}, s.err
}

func (s *failingAdminSafeLogStore) ListAdminResourceReports(context.Context, model.AdminResourceReportFilter) (model.ListAdminResourceReportsResult, error) {
	return model.ListAdminResourceReportsResult{}, s.err
}

func (s *failingAdminSafeLogStore) ReviewResourceReport(context.Context, model.ReviewResourceReportInput) (model.ReviewResourceReportResult, error) {
	return model.ReviewResourceReportResult{}, s.err
}

func (s *failingAdminSafeLogStore) ListAdminVIPPlans(context.Context) ([]model.AdminVIPPlanConfig, error) {
	return nil, s.err
}

func (s *failingAdminSafeLogStore) SaveAdminVIPPlan(context.Context, model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error) {
	return model.AdminVIPConfigSaveResult{}, s.err
}

func (s *failingAdminSafeLogStore) ListAdminQuotaPacks(context.Context) ([]model.AdminQuotaPackConfig, error) {
	return nil, s.err
}

func (s *failingAdminSafeLogStore) SaveAdminQuotaPack(context.Context, model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error) {
	return model.AdminVIPConfigSaveResult{}, s.err
}

func (s *failingAdminSafeLogStore) ListAdminVIPPromotions(context.Context) ([]model.AdminVIPPromotionConfig, error) {
	return nil, s.err
}

func (s *failingAdminSafeLogStore) SaveAdminVIPPromotion(context.Context, model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error) {
	return model.AdminVIPConfigSaveResult{}, s.err
}

func (s *failingAdminSafeLogStore) ListResourceTypeConfigs(context.Context, string, string) ([]model.AdminResourceTypeConfig, error) {
	return nil, s.err
}

func (s *failingAdminSafeLogStore) CreateResourceTypeConfig(context.Context, model.CreateResourceTypeConfigInput) (model.CreateResourceTypeConfigResult, error) {
	return model.CreateResourceTypeConfigResult{}, s.err
}

func (s *failingAdminSafeLogStore) UpdateResourceTypeConfig(context.Context, string, model.ResourceTypeConfigPatch) (model.UpdateResourceTypeConfigResult, error) {
	return model.UpdateResourceTypeConfigResult{}, s.err
}
