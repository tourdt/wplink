package admin

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"
)

func TestAdminPermissionCreateOperatorRequiresSuperAdmin(t *testing.T) {
	logic := NewAdminPermissionLogic(&fakeAdminPermissionStore{}, fakeAdminPasswordHasher{})

	_, err := logic.CreateOperator(context.Background(), SaveAdminOperatorReq{
		LoginName: "operator",
		RealName:  "运营",
		Password:  "secret123",
		Roles:     []string{permission.RolePlatformOperator},
	}, AdminPermissionActor{OperatorID: "admin-1", Roles: []string{permission.RolePlatformOperator}})
	if errx.CodeOf(err) != errx.CodeForbidden {
		t.Fatalf("CreateOperator() error = %v, want forbidden", err)
	}
}

func TestAdminPermissionCreateOperatorPassesSanitizedInput(t *testing.T) {
	store := &fakeAdminPermissionStore{}
	logic := NewAdminPermissionLogic(store, fakeAdminPasswordHasher{})

	resp, err := logic.CreateOperator(context.Background(), SaveAdminOperatorReq{
		LoginName: " 18800000001 ",
		RealName:  " 张运营 ",
		Password:  " secret123 ",
		Roles:     []string{permission.RolePlatformOperator, permission.RolePlatformOperator},
	}, AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if err != nil {
		t.Fatalf("CreateOperator() error = %v", err)
	}
	if resp.UserID != "user-created" {
		t.Fatalf("UserID = %q, want user-created", resp.UserID)
	}
	if store.createInput.LoginName != "18800000001" || store.createInput.RealName != "张运营" {
		t.Fatalf("create input = %#v, want trimmed login and real name", store.createInput)
	}
	if store.createInput.PasswordHash != "hashed:secret123" {
		t.Fatalf("PasswordHash = %q, want hashed password", store.createInput.PasswordHash)
	}
	if len(store.createInput.Roles) != 1 || store.createInput.Roles[0] != permission.RolePlatformOperator {
		t.Fatalf("Roles = %#v, want deduplicated platform operator", store.createInput.Roles)
	}
}

func TestAdminPermissionUpdateRejectsRemovingOwnSuperAdminRole(t *testing.T) {
	logic := NewAdminPermissionLogic(&fakeAdminPermissionStore{}, fakeAdminPasswordHasher{})

	_, err := logic.UpdateOperator(context.Background(), "super-1", SaveAdminOperatorReq{
		LoginName: "super",
		RealName:  "超级管理员",
		Roles:     []string{permission.RolePlatformOperator},
		Status:    model.AdminCredentialStatusEnabled,
	}, AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if errx.CodeOf(err) != errx.CodeForbidden {
		t.Fatalf("UpdateOperator() error = %v, want forbidden", err)
	}
}

func TestAdminPermissionUpdateStatusRejectsDisablingSelf(t *testing.T) {
	logic := NewAdminPermissionLogic(&fakeAdminPermissionStore{}, fakeAdminPasswordHasher{})

	_, err := logic.UpdateOperatorStatus(context.Background(), "super-1", UpdateAdminOperatorStatusReq{
		Status: model.AdminCredentialStatusDisabled,
	}, AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if errx.CodeOf(err) != errx.CodeForbidden {
		t.Fatalf("UpdateOperatorStatus() error = %v, want forbidden", err)
	}
}

func TestAdminPermissionMapsDuplicateLoginName(t *testing.T) {
	logic := NewAdminPermissionLogic(&fakeAdminPermissionStore{createErr: model.ErrAdminOperatorLoginNameExists}, fakeAdminPasswordHasher{})

	_, err := logic.CreateOperator(context.Background(), SaveAdminOperatorReq{
		LoginName: "operator",
		RealName:  "运营",
		Password:  "secret123",
	}, AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("CreateOperator() error = %v, want state conflict", err)
	}
}

func TestAdminPermissionListModulePermissionsDefaultsPlatformOperator(t *testing.T) {
	logic := NewAdminPermissionLogic(&fakeAdminPermissionStore{}, fakeAdminPasswordHasher{})

	resp, err := logic.ListModulePermissions(context.Background(), AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if err != nil {
		t.Fatalf("ListModulePermissions() error = %v", err)
	}
	if len(resp.Modules) == 0 {
		t.Fatal("Modules is empty, want configurable modules")
	}
	if len(resp.Roles) != 1 || resp.Roles[0].RoleCode != permission.RolePlatformOperator {
		t.Fatalf("Roles = %#v, want platform operator role", resp.Roles)
	}
	want := permission.DefaultPlatformOperatorAdminModules()
	if len(resp.Roles[0].Modules) != len(want) || resp.Roles[0].Modules[0] != want[0] {
		t.Fatalf("platform modules = %#v, want default audit modules %#v", resp.Roles[0].Modules, want)
	}
}

func TestAdminPermissionUpdateRoleModulePermissionsNormalizesModules(t *testing.T) {
	store := &fakeAdminPermissionStore{}
	logic := NewAdminPermissionLogic(store, fakeAdminPasswordHasher{})

	resp, err := logic.UpdateRoleModulePermissions(context.Background(), permission.RolePlatformOperator, UpdateAdminRoleModulePermissionsReq{
		Modules: []string{
			permission.AdminModuleMerchants,
			permission.AdminModuleAdminPermissions,
			permission.AdminModuleMerchants,
			"unknown",
			permission.AdminModuleResourceReview,
		},
	}, AdminPermissionActor{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}})
	if err != nil {
		t.Fatalf("UpdateRoleModulePermissions() error = %v", err)
	}
	want := []string{permission.AdminModuleMerchants, permission.AdminModuleResourceReview}
	if len(store.moduleInput.Modules) != len(want) || store.moduleInput.Modules[0] != want[0] || store.moduleInput.Modules[1] != want[1] {
		t.Fatalf("module input = %#v, want %#v", store.moduleInput.Modules, want)
	}
	if resp.RoleCode != permission.RolePlatformOperator || resp.Message == "" {
		t.Fatalf("resp = %#v, want saved platform operator response", resp)
	}
}

type fakeAdminPermissionStore struct {
	createInput model.AdminOperatorInput
	updateInput model.AdminOperatorInput
	statusInput model.AdminOperatorStatusInput
	createErr   error
	updateErr   error
	statusErr   error
	listErr     error
	roleModules []string
	moduleInput model.AdminRoleModulePermissionInput
	moduleErr   error
}

func (s *fakeAdminPermissionStore) ListAdminOperators(ctx context.Context, filter model.AdminOperatorFilter) (model.ListAdminOperatorsResult, error) {
	if s.listErr != nil {
		return model.ListAdminOperatorsResult{}, s.listErr
	}
	return model.ListAdminOperatorsResult{
		Items: []model.AdminOperatorItem{{UserID: "user-1", LoginName: "operator", RealName: "运营", Status: model.AdminCredentialStatusEnabled, Roles: []string{permission.RolePlatformOperator}}},
		Page:  filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeAdminPermissionStore) CreateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	s.createInput = input
	if s.createErr != nil {
		return model.AdminOperatorItem{}, s.createErr
	}
	return model.AdminOperatorItem{UserID: "user-created", LoginName: input.LoginName, RealName: input.RealName, Status: input.Status, Roles: input.Roles}, nil
}

func (s *fakeAdminPermissionStore) UpdateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	s.updateInput = input
	if s.updateErr != nil {
		return model.AdminOperatorItem{}, s.updateErr
	}
	return model.AdminOperatorItem{UserID: input.UserID, LoginName: input.LoginName, RealName: input.RealName, Status: input.Status, Roles: input.Roles}, nil
}

func (s *fakeAdminPermissionStore) UpdateAdminOperatorStatus(ctx context.Context, input model.AdminOperatorStatusInput) (model.AdminOperatorItem, error) {
	s.statusInput = input
	if s.statusErr != nil {
		return model.AdminOperatorItem{}, s.statusErr
	}
	return model.AdminOperatorItem{UserID: input.UserID, Status: input.Status}, nil
}

func (s *fakeAdminPermissionStore) GetAdminRoleModulePermissions(ctx context.Context, roleCode string) (model.AdminRoleModulePermission, error) {
	if s.moduleErr != nil {
		return model.AdminRoleModulePermission{}, s.moduleErr
	}
	return model.AdminRoleModulePermission{RoleCode: roleCode, Modules: append([]string(nil), s.roleModules...)}, nil
}

func (s *fakeAdminPermissionStore) UpdateAdminRoleModulePermissions(ctx context.Context, input model.AdminRoleModulePermissionInput) (model.AdminRoleModulePermission, error) {
	s.moduleInput = input
	if s.moduleErr != nil {
		return model.AdminRoleModulePermission{}, s.moduleErr
	}
	return model.AdminRoleModulePermission{RoleCode: input.RoleCode, Modules: append([]string(nil), input.Modules...)}, nil
}

type fakeAdminPasswordHasher struct{}

func (fakeAdminPasswordHasher) Hash(password string) (string, error) {
	if password == "bad-hash" {
		return "", errors.New("hash failed")
	}
	return "hashed:" + password, nil
}
