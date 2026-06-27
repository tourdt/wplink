package svc

import (
	"database/sql"

	"wplink/backend/app/internal/config"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
}

func NewServiceContext(c config.Config, db *sql.DB) *ServiceContext {
	return &ServiceContext{
		Config: c,
		DB:     db,
	}
}
