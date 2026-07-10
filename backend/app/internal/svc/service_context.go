package svc

import (
	"context"
	"database/sql"
	"fmt"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	citylogic "wplink/backend/app/internal/logic/city"
	paymentlogic "wplink/backend/app/internal/logic/payment"
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
	*model.ResourceMetricDailyModel
	*model.OperationLogModel
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
}

func NewServiceContext(c config.Config, db *sql.DB) (*ServiceContext, error) {
	adminTokenService := session.NewHMACAdminTokenIssuer(c.AdminAuth.TokenSecret, c.AdminAuth.TokenTTL)
	adminTokenIssuer := adminauth.NewSessionTokenIssuer(adminTokenService)
	apiStore := newAPIStore(db)
	wechatPayGateway, err := paymentlogic.NewHTTPWechatPayGateway(c.WechatPay)
	if err != nil {
		return nil, fmt.Errorf("初始化微信支付网关失败: %w", err)
	}
	return &ServiceContext{
		Config:              c,
		DB:                  db,
		APIStore:            apiStore,
		CityStore:           apiStore,
		AdminLoginService:   adminauth.NewLoginService(adminauth.NewSQLAdminStore(db), adminauth.BcryptPasswordHasher{}, adminTokenIssuer),
		AdminTokenService:   adminTokenService,
		UploadTokenService:  uploadlogic.NewUploadTokenLogic(c.Storage),
		UserTokenService:    session.NewHMACUserTokenService(c.AdminAuth.TokenSecret, c.AdminAuth.TokenTTL),
		WechatSessionClient: authlogic.NewWechatSessionClient(c.Wechat, "", nil),
		SMSVerifier:         authlogic.NewConfiguredSMSVerifier(c.SMS),
		WechatPayGateway:    wechatPayGateway,
	}, nil
}

func newAPIStore(db *sql.DB) *APIStore {
	return &APIStore{
		CityStationModel:          model.NewCityStationModel(db),
		ResourceTypeConfigModel:   model.NewResourceTypeConfigModel(db),
		AdminDashboardModel:       model.NewAdminDashboardModel(db),
		UserModel:                 model.NewUserModel(db),
		MerchantModel:             model.NewMerchantModel(db),
		ResourceModel:             model.NewResourceModel(db),
		BannerTopicModel:          model.NewBannerTopicModel(db),
		HotSearchKeywordModel:     model.NewHotSearchKeywordModel(db),
		VerificationModel:         model.NewVerificationModel(db),
		MerchantEntitlementModel:  model.NewMerchantEntitlementModel(db),
		VIPModel:                  model.NewVIPModel(db),
		MessageModel:              model.NewMessageModel(db),
		SearchLogModel:            model.NewSearchLogModel(db),
		ResourceContactEventModel: model.NewResourceContactEventModel(db),
		ResourceMetricDailyModel:  model.NewResourceMetricDailyModel(db),
		OperationLogModel:         model.NewOperationLogModel(db),
		GrowthCampaignModel:       model.NewGrowthCampaignModel(db),
		FavoriteModel:             model.NewFavoriteModel(db),
		MapModel:                  model.NewMapModel(db),
	}
}
