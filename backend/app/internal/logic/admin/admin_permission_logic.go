package admin

import (
	"context"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminPermissionStore interface {
	ListAdminOperators(ctx context.Context, filter model.AdminOperatorFilter) (model.ListAdminOperatorsResult, error)
	CreateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error)
	UpdateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error)
	UpdateAdminOperatorStatus(ctx context.Context, input model.AdminOperatorStatusInput) (model.AdminOperatorItem, error)
	GetAdminRoleModulePermissions(ctx context.Context, roleCode string) (model.AdminRoleModulePermission, error)
	UpdateAdminRoleModulePermissions(ctx context.Context, input model.AdminRoleModulePermissionInput) (model.AdminRoleModulePermission, error)
}

type AdminOperatorPasswordHasher interface {
	Hash(password string) (string, error)
}

type AdminPermissionActor struct {
	OperatorID string
	Roles      []string
}

type ListAdminOperatorsReq struct {
	Keyword  string
	Role     string
	Status   string
	Page     int64
	PageSize int64
}

type SaveAdminOperatorReq struct {
	LoginName string
	RealName  string
	Password  string
	Roles     []string
	Status    string
}

type UpdateAdminOperatorStatusReq struct {
	Status string
}

type UpdateAdminRoleModulePermissionsReq struct {
	Modules []string
}

type AdminOperatorItem struct {
	OperatorID  string   `json:"operatorId"`
	LoginName   string   `json:"loginName"`
	RealName    string   `json:"realName"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	CreatedAt   string   `json:"createdAt"`
	LastLoginAt string   `json:"lastLoginAt,omitempty"`
}

type AdminModuleItem struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type AdminRoleModulePermissionItem struct {
	RoleCode string   `json:"roleCode"`
	RoleName string   `json:"roleName"`
	Modules  []string `json:"modules"`
}

type ListAdminOperatorsResp struct {
	Items    []AdminOperatorItem `json:"items"`
	Page     int64               `json:"page"`
	PageSize int64               `json:"pageSize"`
	Total    int64               `json:"total"`
}

type AdminModulePermissionsResp struct {
	Modules []AdminModuleItem               `json:"modules"`
	Roles   []AdminRoleModulePermissionItem `json:"roles"`
}

type SaveAdminOperatorResp struct {
	OperatorID string `json:"operatorId"`
	Message    string `json:"message"`
}

type SaveAdminRoleModulePermissionsResp struct {
	RoleCode string   `json:"roleCode"`
	Modules  []string `json:"modules"`
	Message  string   `json:"message"`
}

type AdminPermissionLogic struct {
	store  AdminPermissionStore
	hasher AdminOperatorPasswordHasher
}

func NewAdminPermissionLogic(store AdminPermissionStore, hasher AdminOperatorPasswordHasher) *AdminPermissionLogic {
	return &AdminPermissionLogic{store: store, hasher: hasher}
}

func (l *AdminPermissionLogic) ListOperators(ctx context.Context, req ListAdminOperatorsReq, actor AdminPermissionActor) (ListAdminOperatorsResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return ListAdminOperatorsResp{}, err
	}
	role := strings.TrimSpace(req.Role)
	if role != "" && role != permission.RolePlatformOperator && role != permission.RoleSuperAdmin {
		return ListAdminOperatorsResp{}, errx.New(errx.CodeValidationFailed, "管理员角色筛选条件不正确")
	}
	status := normalizeAdminOperatorStatus(req.Status)
	if strings.TrimSpace(req.Status) != "" && status == "" {
		return ListAdminOperatorsResp{}, errx.New(errx.CodeValidationFailed, "账号状态筛选条件不正确")
	}

	result, err := l.store.ListAdminOperators(ctx, model.AdminOperatorFilter{
		Keyword:  strings.TrimSpace(req.Keyword),
		Role:     role,
		Status:   status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		logx.Errorf("查询后台管理员账号失败: operatorId=%s role=%s status=%s err=%+v", actor.OperatorID, role, status, err)
		return ListAdminOperatorsResp{}, errx.New(errx.CodeInternalError, "管理员账号加载失败，请稍后重试")
	}
	return mapAdminOperatorsResult(result), nil
}

func (l *AdminPermissionLogic) CreateOperator(ctx context.Context, req SaveAdminOperatorReq, actor AdminPermissionActor) (SaveAdminOperatorResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return SaveAdminOperatorResp{}, err
	}
	input, err := l.buildOperatorInput("", req, actor, true)
	if err != nil {
		return SaveAdminOperatorResp{}, err
	}
	item, err := l.store.CreateAdminOperator(ctx, input)
	if err != nil {
		return SaveAdminOperatorResp{}, l.mapSaveError("创建后台管理员账号", actor.OperatorID, input.LoginName, err)
	}
	logx.Infof("创建后台管理员账号成功: operatorId=%s targetOperatorId=%s loginName=%s roles=%v", actor.OperatorID, item.OperatorID, item.LoginName, item.Roles)
	return SaveAdminOperatorResp{OperatorID: item.OperatorID, Message: "管理员账号已创建"}, nil
}

func (l *AdminPermissionLogic) UpdateOperator(ctx context.Context, operatorID string, req SaveAdminOperatorReq, actor AdminPermissionActor) (SaveAdminOperatorResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return SaveAdminOperatorResp{}, err
	}
	input, err := l.buildOperatorInput(operatorID, req, actor, false)
	if err != nil {
		return SaveAdminOperatorResp{}, err
	}
	// 防止超级管理员误操作移除自己的最高权限，导致后台无人可继续管理管理员账号。
	if strings.TrimSpace(input.OperatorID) == strings.TrimSpace(actor.OperatorID) && !containsRole(input.Roles, permission.RoleSuperAdmin) {
		return SaveAdminOperatorResp{}, errx.New(errx.CodeForbidden, "不能移除当前登录账号的超级管理员权限")
	}
	item, err := l.store.UpdateAdminOperator(ctx, input)
	if err != nil {
		return SaveAdminOperatorResp{}, l.mapSaveError("更新后台管理员账号", actor.OperatorID, input.LoginName, err)
	}
	logx.Infof("更新后台管理员账号成功: operatorId=%s targetOperatorId=%s loginName=%s roles=%v status=%s", actor.OperatorID, item.OperatorID, item.LoginName, item.Roles, item.Status)
	return SaveAdminOperatorResp{OperatorID: item.OperatorID, Message: "管理员账号已更新"}, nil
}

func (l *AdminPermissionLogic) UpdateOperatorStatus(ctx context.Context, operatorID string, req UpdateAdminOperatorStatusReq, actor AdminPermissionActor) (SaveAdminOperatorResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return SaveAdminOperatorResp{}, err
	}
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return SaveAdminOperatorResp{}, errx.New(errx.CodeValidationFailed, "管理员账号不存在")
	}
	status := normalizeAdminOperatorStatus(req.Status)
	if status == "" {
		return SaveAdminOperatorResp{}, errx.New(errx.CodeValidationFailed, "账号状态不正确")
	}
	// 当前登录账号不能停用自己，否则 token 过期后会造成超级管理员无法恢复账号。
	if operatorID == strings.TrimSpace(actor.OperatorID) && status == model.AdminCredentialStatusDisabled {
		return SaveAdminOperatorResp{}, errx.New(errx.CodeForbidden, "不能停用当前登录账号")
	}
	item, err := l.store.UpdateAdminOperatorStatus(ctx, model.AdminOperatorStatusInput{
		OperatorID: operatorID,
		Status:     status,
		ActorID:    strings.TrimSpace(actor.OperatorID),
	})
	if err != nil {
		return SaveAdminOperatorResp{}, l.mapSaveError("更新后台管理员账号状态", actor.OperatorID, operatorID, err)
	}
	logx.Infof("更新后台管理员账号状态成功: operatorId=%s targetOperatorId=%s status=%s", actor.OperatorID, item.OperatorID, item.Status)
	return SaveAdminOperatorResp{OperatorID: item.OperatorID, Message: "管理员账号状态已更新"}, nil
}

func (l *AdminPermissionLogic) ListModulePermissions(ctx context.Context, actor AdminPermissionActor) (AdminModulePermissionsResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return AdminModulePermissionsResp{}, err
	}
	rolePermission, err := l.store.GetAdminRoleModulePermissions(ctx, permission.RolePlatformOperator)
	if err != nil {
		logx.Errorf("查询后台角色模块权限失败: operatorId=%s roleCode=%s err=%+v", actor.OperatorID, permission.RolePlatformOperator, err)
		return AdminModulePermissionsResp{}, errx.New(errx.CodeInternalError, "模块权限配置加载失败，请稍后重试")
	}
	modules := permission.NormalizeConfigurableAdminModules(rolePermission.Modules)
	if len(modules) == 0 {
		modules = permission.DefaultPlatformOperatorAdminModules()
	}
	return AdminModulePermissionsResp{
		Modules: adminModuleItems(),
		Roles: []AdminRoleModulePermissionItem{{
			RoleCode: permission.RolePlatformOperator,
			RoleName: "平台运营",
			Modules:  modules,
		}},
	}, nil
}

func (l *AdminPermissionLogic) UpdateRoleModulePermissions(ctx context.Context, roleCode string, req UpdateAdminRoleModulePermissionsReq, actor AdminPermissionActor) (SaveAdminRoleModulePermissionsResp, error) {
	if err := requireSuperAdminActor(actor); err != nil {
		return SaveAdminRoleModulePermissionsResp{}, err
	}
	roleCode = strings.TrimSpace(roleCode)
	if roleCode != permission.RolePlatformOperator {
		return SaveAdminRoleModulePermissionsResp{}, errx.New(errx.CodeForbidden, "当前只支持配置平台运营的模块权限")
	}
	modules := permission.NormalizeConfigurableAdminModules(req.Modules)
	if len(modules) == 0 {
		return SaveAdminRoleModulePermissionsResp{}, errx.New(errx.CodeValidationFailed, "请至少选择一个后台功能模块")
	}
	rolePermission, err := l.store.UpdateAdminRoleModulePermissions(ctx, model.AdminRoleModulePermissionInput{
		RoleCode:   roleCode,
		Modules:    modules,
		OperatorID: strings.TrimSpace(actor.OperatorID),
	})
	if err != nil {
		if errors.Is(err, model.ErrAdminRoleNotFound) {
			return SaveAdminRoleModulePermissionsResp{}, errx.New(errx.CodeResourceNotFound, "管理员角色不存在")
		}
		logx.Errorf("保存后台角色模块权限失败: operatorId=%s roleCode=%s modules=%v err=%+v", actor.OperatorID, roleCode, modules, err)
		return SaveAdminRoleModulePermissionsResp{}, errx.New(errx.CodeInternalError, "模块权限配置保存失败，请稍后重试")
	}
	logx.Infof("保存后台角色模块权限成功: operatorId=%s roleCode=%s modules=%v", actor.OperatorID, roleCode, rolePermission.Modules)
	return SaveAdminRoleModulePermissionsResp{
		RoleCode: rolePermission.RoleCode,
		Modules:  permission.NormalizeConfigurableAdminModules(rolePermission.Modules),
		Message:  "模块权限配置已保存，平台运营重新登录后生效",
	}, nil
}

func (l *AdminPermissionLogic) buildOperatorInput(operatorID string, req SaveAdminOperatorReq, actor AdminPermissionActor, passwordRequired bool) (model.AdminOperatorInput, error) {
	loginName := strings.TrimSpace(req.LoginName)
	realName := strings.TrimSpace(req.RealName)
	if loginName == "" {
		return model.AdminOperatorInput{}, errx.New(errx.CodeValidationFailed, "请填写登录账号")
	}
	if realName == "" {
		return model.AdminOperatorInput{}, errx.New(errx.CodeValidationFailed, "请填写管理员姓名")
	}
	status := normalizeAdminOperatorStatus(req.Status)
	if status == "" {
		status = model.AdminCredentialStatusEnabled
	}
	roles, err := normalizeAdminOperatorRoles(req.Roles)
	if err != nil {
		return model.AdminOperatorInput{}, err
	}

	password := strings.TrimSpace(req.Password)
	var passwordHash string
	if passwordRequired || password != "" {
		if password == "" {
			return model.AdminOperatorInput{}, errx.New(errx.CodeValidationFailed, "请填写登录密码")
		}
		if len(password) < 6 {
			return model.AdminOperatorInput{}, errx.New(errx.CodeValidationFailed, "登录密码至少需要 6 位")
		}
		hash, err := l.hasher.Hash(password)
		if err != nil {
			logx.Errorf("生成后台管理员密码哈希失败: operatorId=%s loginName=%s err=%+v", actor.OperatorID, loginName, err)
			return model.AdminOperatorInput{}, errx.New(errx.CodeInternalError, "管理员账号保存失败，请稍后重试")
		}
		passwordHash = hash
	}

	return model.AdminOperatorInput{
		OperatorID:   strings.TrimSpace(operatorID),
		LoginName:    loginName,
		RealName:     realName,
		PasswordHash: passwordHash,
		Status:       status,
		Roles:        roles,
		ActorID:      strings.TrimSpace(actor.OperatorID),
	}, nil
}

func (l *AdminPermissionLogic) mapSaveError(action string, operatorID string, target string, err error) error {
	switch {
	case errors.Is(err, model.ErrAdminOperatorLoginNameExists):
		return errx.New(errx.CodeStateConflict, "登录账号已存在，请更换后重试")
	case errors.Is(err, model.ErrAdminOperatorNotFound):
		return errx.New(errx.CodeResourceNotFound, "管理员账号不存在或已被删除")
	default:
		logx.Errorf("%s失败: operatorId=%s target=%s err=%+v", action, strings.TrimSpace(operatorID), strings.TrimSpace(target), err)
		return errx.New(errx.CodeInternalError, "管理员账号保存失败，请稍后重试")
	}
}

func requireSuperAdminActor(actor AdminPermissionActor) error {
	if strings.TrimSpace(actor.OperatorID) == "" {
		return errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	if !permission.CanManageAdminPermissions(actor.Roles) {
		return errx.New(errx.CodeForbidden, "只有超级管理员可以管理管理员权限")
	}
	return nil
}

func normalizeAdminOperatorStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "":
		return ""
	case model.AdminCredentialStatusEnabled:
		return model.AdminCredentialStatusEnabled
	case model.AdminCredentialStatusDisabled:
		return model.AdminCredentialStatusDisabled
	default:
		return ""
	}
}

func normalizeAdminOperatorRoles(roles []string) ([]string, error) {
	if len(roles) == 0 {
		return []string{permission.RolePlatformOperator}, nil
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if role != permission.RolePlatformOperator && role != permission.RoleSuperAdmin {
			return nil, errx.New(errx.CodeValidationFailed, "管理员角色不正确")
		}
		if !seen[role] {
			normalized = append(normalized, role)
			seen[role] = true
		}
	}
	if len(normalized) == 0 {
		return nil, errx.New(errx.CodeValidationFailed, "请至少选择一个管理员角色")
	}
	return normalized, nil
}

func containsRole(roles []string, target string) bool {
	for _, role := range roles {
		if strings.TrimSpace(role) == target {
			return true
		}
	}
	return false
}

func mapAdminOperatorsResult(result model.ListAdminOperatorsResult) ListAdminOperatorsResp {
	items := make([]AdminOperatorItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, AdminOperatorItem{
			OperatorID:  item.OperatorID,
			LoginName:   item.LoginName,
			RealName:    item.RealName,
			Status:      item.Status,
			Roles:       append([]string(nil), item.Roles...),
			CreatedAt:   item.CreatedAt,
			LastLoginAt: item.LastLoginAt,
		})
	}
	return ListAdminOperatorsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}
}

func adminModuleItems() []AdminModuleItem {
	labels := map[string]AdminModuleItem{
		permission.AdminModuleDashboard:           {Code: permission.AdminModuleDashboard, Label: "数据概览", Group: "概览"},
		permission.AdminModuleResourceReview:      {Code: permission.AdminModuleResourceReview, Label: "供需信息审核", Group: "审核"},
		permission.AdminModuleResourceReports:     {Code: permission.AdminModuleResourceReports, Label: "举报审核", Group: "审核"},
		permission.AdminModuleVerificationReview:  {Code: permission.AdminModuleVerificationReview, Label: "认证审核", Group: "审核"},
		permission.AdminModuleMerchants:           {Code: permission.AdminModuleMerchants, Label: "商家管理", Group: "运营"},
		permission.AdminModuleEntitlements:        {Code: permission.AdminModuleEntitlements, Label: "权益发放", Group: "运营"},
		permission.AdminModuleBannerTopics:        {Code: permission.AdminModuleBannerTopics, Label: "首页运营位", Group: "运营配置"},
		permission.AdminModuleHotSearchKeywords:   {Code: permission.AdminModuleHotSearchKeywords, Label: "热门搜索词", Group: "运营配置"},
		permission.AdminModuleVIPConfigs:          {Code: permission.AdminModuleVIPConfigs, Label: "VIP 配置", Group: "运营配置"},
		permission.AdminModuleGrowthCampaigns:     {Code: permission.AdminModuleGrowthCampaigns, Label: "增长活动", Group: "运营配置"},
		permission.AdminModuleSourcingMap:         {Code: permission.AdminModuleSourcingMap, Label: "拿货地图", Group: "运营配置"},
		permission.AdminModuleResourceTypeConfigs: {Code: permission.AdminModuleResourceTypeConfigs, Label: "供需类型配置", Group: "运营配置"},
		permission.AdminModuleOperationLogs:       {Code: permission.AdminModuleOperationLogs, Label: "操作日志", Group: "日志"},
		permission.AdminModuleSearchLogs:          {Code: permission.AdminModuleSearchLogs, Label: "搜索日志", Group: "日志"},
	}
	modules := permission.ConfigurableAdminModules()
	items := make([]AdminModuleItem, 0, len(modules))
	for _, module := range modules {
		if item, ok := labels[module]; ok {
			items = append(items, item)
		}
	}
	return items
}
