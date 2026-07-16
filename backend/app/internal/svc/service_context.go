package svc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	citylogic "wplink/backend/app/internal/logic/city"
	"wplink/backend/app/internal/logic/contentaudit"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	resourcelogic "wplink/backend/app/internal/logic/resource"
	uploadlogic "wplink/backend/app/internal/logic/upload"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

type CityStore interface {
	citylogic.CityStationStore
	citylogic.ResourceTypeStore
}

type AdminLoginService interface {
	Login(ctx context.Context, req adminauth.LoginRequest) (adminauth.LoginResponse, error)
}

type APIStore struct {
	*model.CityStationModel
	*model.ResourceTypeConfigModel
	*model.AdminDashboardModel
	*model.UserModel
	*model.MerchantModel
	*model.ResourceModel
	*model.BannerTopicModel
	*model.HotSearchKeywordModel
	*model.VerificationModel
	*model.MerchantEntitlementModel
	*model.VIPModel
	*model.MessageModel
	*model.SearchLogModel
	*model.ResourceContactEventModel
	*model.ResourceContactUnlockModel
	*model.ResourceMetricDailyModel
	*model.OperationLogModel
	*model.AdminPermissionModel
	*model.GrowthCampaignModel
	*model.FavoriteModel
	*model.MapModel
}

type ServiceContext struct {
	Config              config.Config
	DB                  *sql.DB
	APIStore            *APIStore
	CityStore           CityStore
	AdminLoginService   AdminLoginService
	AdminTokenService   *session.HMACAdminTokenIssuer
	UploadTokenService  *uploadlogic.UploadTokenLogic
	UserTokenService    *session.HMACUserTokenService
	WechatSessionClient authlogic.WechatSessionClient
	SMSVerifier         authlogic.SMSVerifier
	WechatPayGateway    paymentlogic.WechatPayGateway
	ContentAuditor      resourcelogic.ContentAuditor
}

func NewServiceContext(c config.Config, db *sql.DB) (*ServiceContext, error) {
	adminTokenService := session.NewHMACAdminTokenIssuer(c.AdminAuth.TokenSecret, c.AdminAuth.TokenTTL)
	adminTokenIssuer := adminauth.NewSessionTokenIssuer(adminTokenService)
	adminLoginOptions := make([]adminauth.LoginServiceOption, 0, 1)
	if masterPassword := enabledAdminMasterPassword(c); masterPassword != "" {
		adminLoginOptions = append(adminLoginOptions, adminauth.WithMasterPassword(masterPassword))
	}
	apiStore := newAPIStore(db)
	wechatPayGateway, err := paymentlogic.NewHTTPWechatPayGateway(c.WechatPay)
	if err != nil {
		return nil, fmt.Errorf("初始化微信支付网关失败: %w", err)
	}
	var contentAuditor resourcelogic.ContentAuditor
	if auditor := contentaudit.NewWechatAuditor(c.Wechat, c.ContentAudit, c.Storage.PublicBaseURL, nil); auditor != nil {
		contentAuditor = auditor
	}
	return &ServiceContext{
		Config:              c,
		DB:                  db,
		APIStore:            apiStore,
		CityStore:           apiStore,
		AdminLoginService:   adminauth.NewLoginService(adminauth.NewSQLAdminStore(db), adminauth.BcryptPasswordHasher{}, adminTokenIssuer, adminLoginOptions...),
		AdminTokenService:   adminTokenService,
		UploadTokenService:  uploadlogic.NewUploadTokenLogic(c.Storage),
		UserTokenService:    session.NewHMACUserTokenService(c.UserAuth.TokenSecret, c.UserAuth.TokenTTL),
		WechatSessionClient: authlogic.NewWechatSessionClient(c.Wechat, "", nil),
		SMSVerifier:         authlogic.NewConfiguredSMSVerifier(c.SMS),
		WechatPayGateway:    wechatPayGateway,
		ContentAuditor:      contentAuditor,
	}, nil
}

func enabledAdminMasterPassword(c config.Config) string {
	if !config.IsDevelopmentMode(c.RuntimeMode) {
		return ""
	}
	return strings.TrimSpace(c.AdminAuth.MasterPassword)
}

func newAPIStore(db *sql.DB) *APIStore {
	return &APIStore{
		CityStationModel:           model.NewCityStationModel(db),
		ResourceTypeConfigModel:    model.NewResourceTypeConfigModel(db),
		AdminDashboardModel:        model.NewAdminDashboardModel(db),
		UserModel:                  model.NewUserModel(db),
		MerchantModel:              model.NewMerchantModel(db),
		ResourceModel:              model.NewResourceModel(db),
		BannerTopicModel:           model.NewBannerTopicModel(db),
		HotSearchKeywordModel:      model.NewHotSearchKeywordModel(db),
		VerificationModel:          model.NewVerificationModel(db),
		MerchantEntitlementModel:   model.NewMerchantEntitlementModel(db),
		VIPModel:                   model.NewVIPModel(db),
		MessageModel:               model.NewMessageModel(db),
		SearchLogModel:             model.NewSearchLogModel(db),
		ResourceContactEventModel:  model.NewResourceContactEventModel(db),
		ResourceContactUnlockModel: model.NewResourceContactUnlockModel(db),
		ResourceMetricDailyModel:   model.NewResourceMetricDailyModel(db),
		OperationLogModel:          model.NewOperationLogModel(db),
		AdminPermissionModel:       model.NewAdminPermissionModel(db),
		GrowthCampaignModel:        model.NewGrowthCampaignModel(db),
		FavoriteModel:              model.NewFavoriteModel(db),
		MapModel:                   model.NewMapModel(db),
	}
}
