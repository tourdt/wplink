package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"wplink/backend/app/internal/adminweb"
	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/server"
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
	cityStore := struct {
		*model.CityStationModel
		*model.ResourceTypeConfigModel
	}{
		CityStationModel:        model.NewCityStationModel(db),
		ResourceTypeConfigModel: model.NewResourceTypeConfigModel(db),
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("启动 %s: addr=%s pid=%d", cfg.Name, addr, os.Getpid())
	if err := http.ListenAndServe(addr, server.NewRouter(adminHandler, server.NewAPIRouter(cityStore))); err != nil {
		log.Fatalf("服务退出: err=%v", err)
	}
}
