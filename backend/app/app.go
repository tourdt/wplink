package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"wplink/backend/app/internal/adminweb"
	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/server"
	"wplink/backend/app/internal/session"
)

func main() {
	configPath := flag.String("f", "etc/app.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: path=%s err=%v", *configPath, err)
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 4000
	}

	adminHandler, err := adminweb.EmbeddedHandler("/admin/")
	if err != nil {
		log.Fatalf("加载管理后台静态资源失败: err=%v", err)
	}
	db, err := model.OpenPostgres(cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: err=%v", err)
	}
	defer db.Close()
	apiStore := struct {
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
	}{
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

	addr := fmt.Sprintf("%s:%d", host, port)
	adminTokenIssuer := adminauth.NewSessionTokenIssuer(session.NewHMACAdminTokenIssuer(cfg.AdminAuth.TokenSecret, cfg.AdminAuth.TokenTTL))
	adminLoginService := adminauth.NewLoginService(adminauth.NewSQLAdminStore(db), adminauth.BcryptPasswordHasher{}, adminTokenIssuer)
	userTokenService := session.NewHMACUserTokenService(cfg.AdminAuth.TokenSecret, cfg.AdminAuth.TokenTTL)
	log.Printf("启动 %s: addr=%s pid=%d", cfg.Name, addr, os.Getpid())
	if err := http.ListenAndServe(addr, server.NewRouter(adminHandler, server.NewAPIRouter(
		apiStore,
		server.WithAdminLoginService(adminLoginService),
		server.WithUserTokenService(userTokenService),
	))); err != nil {
		log.Fatalf("服务退出: err=%v", err)
	}
}
