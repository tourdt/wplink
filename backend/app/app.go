package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"wplink/backend/app/internal/adminweb"
	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/server"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/task"

	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	configPath := flag.String("f", "etc/app.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: path=%s err=%v", *configPath, err)
	}
	if err := setupLogging(cfg); err != nil {
		log.Fatalf("初始化日志失败: path=%s err=%v", cfg.Log.Path, err)
	}
	if err := config.ValidateForProduction(cfg); err != nil {
		fatalf("生产配置校验失败: err=%v", err)
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
		fatalf("加载管理后台静态资源失败: err=%v", err)
	}
	db, err := model.OpenPostgres(cfg.Postgres.DSN, model.PostgresOptions{
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		fatalf("连接 PostgreSQL 失败: err=%v", err)
	}
	defer db.Close()
	svcCtx, err := svc.NewServiceContext(cfg, db)
	if err != nil {
		fatalf("初始化服务上下文失败: err=%v", err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lifecycleScheduler := task.NewResourceLifecycleScheduler(
		task.NewResourceLifecycleTask(svcCtx.APIStore),
		cfg.Tasks.ResourceLifecycleInterval,
		log.Default(),
	)
	if lifecycleScheduler.Enabled() {
		logx.Infof("资源生命周期自动任务已启用: interval=%s", cfg.Tasks.ResourceLifecycleInterval)
		lifecycleScheduler.Start(appCtx)
	}
	mapEventCleanupScheduler := task.NewMerchantMapEventCleanupScheduler(
		task.NewMerchantMapEventCleanupTask(
			svcCtx.APIStore,
			cfg.Tasks.MerchantMapEventRetentionDays,
			time.Now,
		),
		cfg.Tasks.MerchantMapEventCleanupInterval,
		log.Default(),
	)
	if mapEventCleanupScheduler.Enabled() {
		logx.Infof(
			"商家地图行为清理任务已启用: interval=%s retentionDays=%d",
			cfg.Tasks.MerchantMapEventCleanupInterval,
			cfg.Tasks.MerchantMapEventRetentionDays,
		)
		mapEventCleanupScheduler.Start(appCtx)
	}
	contentAuditRetryScheduler := task.NewContentAuditRetryScheduler(
		task.NewContentAuditRetryTask(
			svcCtx.APIStore,
			svcCtx.ContentAuditor,
			cfg.Tasks.ContentAuditRetryBatchSize,
		),
		cfg.Tasks.ContentAuditRetryInterval,
		log.Default(),
	)
	if svcCtx.ContentAuditor != nil && contentAuditRetryScheduler.Enabled() {
		logx.Infof(
			"内容审核自动重试任务已启用: interval=%s batchSize=%d",
			cfg.Tasks.ContentAuditRetryInterval,
			cfg.Tasks.ContentAuditRetryBatchSize,
		)
		contentAuditRetryScheduler.Start(appCtx)
	}
	paymentScheduler := task.NewPaymentReconciliationScheduler(
		task.NewPaymentReconciliationTask(
			svcCtx.APIStore,
			svcCtx.WechatPayOrderGateway,
			cfg.Tasks.PaymentQueryDelay,
			cfg.WechatPay.OrderExpire,
			cfg.Tasks.PaymentBatchSize,
		),
		cfg.Tasks.PaymentReconcileInterval,
		log.Default(),
	)
	if svcCtx.WechatPayOrderGateway != nil && paymentScheduler.Enabled() {
		logx.Infof(
			"微信支付补偿任务已启用: interval=%s queryDelay=%s pendingTimeout=%s batchSize=%d",
			cfg.Tasks.PaymentReconcileInterval,
			cfg.Tasks.PaymentQueryDelay,
			cfg.WechatPay.OrderExpire,
			cfg.Tasks.PaymentBatchSize,
		)
		paymentScheduler.Start(appCtx)
	}

	apiHandler, err := server.NewProductionAPIRouter(
		svcCtx.APIStore,
		server.WithAdminLoginService(svcCtx.AdminLoginService),
		server.WithAdminTokenService(svcCtx.AdminTokenService),
		server.WithUploadTokenService(svcCtx.UploadTokenService),
		server.WithUserTokenService(svcCtx.UserTokenService),
		server.WithWechatSessionClient(svcCtx.WechatSessionClient),
		server.WithSMSVerifier(svcCtx.SMSVerifier),
		server.WithWechatPayGateway(svcCtx.WechatPayGateway),
		server.WithWechatPayDevMock(cfg.WechatPay.DevMockEnabled && !config.IsProductionMode(cfg.RuntimeMode)),
		server.WithContentAuditor(svcCtx.ContentAuditor),
		server.WithContentAuditCallbackVerifier(svcCtx.ContentAuditCallbackVerifier, cfg.Wechat.AppID),
		server.WithLocationGeocoder(svcCtx.LocationGeocoder),
	)
	if err != nil {
		fatalf("初始化 API 路由失败: err=%v", err)
	}
	goZeroServer, err := server.NewGoZeroServer(cfg, svcCtx, adminHandler, apiHandler)
	if err != nil {
		fatalf("初始化 go-zero HTTP 服务失败: err=%v", err)
	}
	defer goZeroServer.Stop()

	logx.Infof("启动 %s: addr=%s:%d pid=%d", cfg.Name, host, port, os.Getpid())
	goZeroServer.Start()
}

func setupLogging(cfg config.Config) error {
	if err := logx.SetUp(cfg.Log); err != nil {
		return err
	}
	log.SetFlags(0)
	log.SetOutput(logxStdWriter{})
	return nil
}

func fatalf(format string, v ...any) {
	logx.Errorf(format, v...)
	logx.Close()
	os.Exit(1)
}

type logxStdWriter struct{}

func (logxStdWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		logx.Infof("%s", msg)
	}
	return len(p), nil
}
