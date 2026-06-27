package config

import "time"

type Config struct {
	Name      string
	Host      string
	Port      int
	Postgres  PostgresConfig
	AdminAuth AdminAuthConfig
	Storage   StorageConfig
}

type PostgresConfig struct {
	DSN string
}

type AdminAuthConfig struct {
	TokenSecret string
	TokenTTL    time.Duration
}

type StorageConfig struct {
	Provider            string
	Endpoint            string
	Bucket              string
	Region              string
	AccessKeyID         string
	AccessKeySecret     string
	PublicBaseURL       string
	UploadExpire        time.Duration
	MaxFileSizeBytes    int64
	AllowedContentTypes []string
}
