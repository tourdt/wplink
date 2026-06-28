package svc

import (
	"context"
	"database/sql"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/logic/adminauth"
	citylogic "wplink/backend/app/internal/logic/city"
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
	*model.DemandModel
	*model.BannerTopicModel
	*model.VerificationModel
	*model.MerchantEntitlementModel
	*model.MessageModel
	*model.MatchCaseModel
	*model.SearchLogModel
	*model.ResourceContactEventModel
	*model.ResourceMetricDailyModel
	*model.OperationLogModel
}

type ServiceContext struct {
	Config            config.Config
	DB                *sql.DB
	APIStore          *APIStore
	CityStore         CityStore
	AdminLoginService AdminLoginService
	UserTokenService  *session.HMACUserTokenService
}

func NewServiceContext(c config.Config, db *sql.DB) *ServiceContext {
	adminTokenIssuer := adminauth.NewSessionTokenIssuer(session.NewHMACAdminTokenIssuer(c.AdminAuth.TokenSecret, c.AdminAuth.TokenTTL))
	apiStore := newAPIStore(db)
	return &ServiceContext{
		Config:            c,
		DB:                db,
		APIStore:          apiStore,
		CityStore:         apiStore,
		AdminLoginService: adminauth.NewLoginService(adminauth.NewSQLAdminStore(db), adminauth.BcryptPasswordHasher{}, adminTokenIssuer),
		UserTokenService:  session.NewHMACUserTokenService(c.AdminAuth.TokenSecret, c.AdminAuth.TokenTTL),
	}
}

func newAPIStore(db *sql.DB) *APIStore {
	return &APIStore{
		CityStationModel:          model.NewCityStationModel(db),
		ResourceTypeConfigModel:   model.NewResourceTypeConfigModel(db),
		AdminDashboardModel:       model.NewAdminDashboardModel(db),
		UserModel:                 model.NewUserModel(db),
		MerchantModel:             model.NewMerchantModel(db),
		ResourceModel:             model.NewResourceModel(db),
		DemandModel:               model.NewDemandModel(db),
		BannerTopicModel:          model.NewBannerTopicModel(db),
		VerificationModel:         model.NewVerificationModel(db),
		MerchantEntitlementModel:  model.NewMerchantEntitlementModel(db),
		MessageModel:              model.NewMessageModel(db),
		MatchCaseModel:            model.NewMatchCaseModel(db),
		SearchLogModel:            model.NewSearchLogModel(db),
		ResourceContactEventModel: model.NewResourceContactEventModel(db),
		ResourceMetricDailyModel:  model.NewResourceMetricDailyModel(db),
		OperationLogModel:         model.NewOperationLogModel(db),
	}
}
