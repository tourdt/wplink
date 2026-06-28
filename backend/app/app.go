package main

import (
	"flag"
	"log"
	"os"

	"wplink/backend/app/internal/adminweb"
	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/server"
	"wplink/backend/app/internal/svc"
)

func main() {
	configPath := flag.String("f", "etc/app.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: path=%s err=%v", *configPath, err)
	}
	if err := config.ValidateForProduction(cfg); err != nil {
		log.Fatalf("生产配置校验失败: err=%v", err)
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
	db, err := model.OpenPostgres(cfg.Postgres.DSN, model.PostgresOptions{
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: err=%v", err)
	}
	defer db.Close()
	svcCtx := svc.NewServiceContext(cfg, db)

	apiHandler := server.NewAPIRouter(
		svcCtx.APIStore,
		server.WithAdminLoginService(svcCtx.AdminLoginService),
		server.WithAdminTokenService(svcCtx.AdminTokenService),
		server.WithUploadTokenService(svcCtx.UploadTokenService),
		server.WithUserTokenService(svcCtx.UserTokenService),
		server.WithWechatSessionClient(svcCtx.WechatSessionClient),
		server.WithSMSVerifier(svcCtx.SMSVerifier),
	)
	goZeroServer, err := server.NewGoZeroServer(cfg, svcCtx, adminHandler, apiHandler)
	if err != nil {
		log.Fatalf("初始化 go-zero HTTP 服务失败: err=%v", err)
	}
	defer goZeroServer.Stop()

	log.Printf("启动 %s: addr=%s:%d pid=%d", cfg.Name, host, port, os.Getpid())
	goZeroServer.Start()
}
