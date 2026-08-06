package admin

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"

	"github.com/zeromicro/go-zero/core/logx"
)

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
	if strings.Count(logText, "errorType=") < 5 {
		t.Fatalf("log=%q, want a safe errorType for each dependency failure", logText)
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
