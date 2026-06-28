package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	var section string
	var listKey string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			if section == "Storage" && listKey == "AllowedContentTypes" {
				cfg.Storage.AllowedContentTypes = append(cfg.Storage.AllowedContentTypes, cleanValue(strings.TrimPrefix(line, "- ")))
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Config{}, fmt.Errorf("配置行格式不正确: %s", raw)
		}
		key = strings.TrimSpace(key)
		rawValue := strings.TrimSpace(value)
		value = cleanValue(value)
		if rawValue == "" {
			if isTopLevelLine(raw) {
				section = key
				listKey = ""
			} else {
				listKey = key
			}
			continue
		}
		if err := applyConfigValue(&cfg, section, key, value); err != nil {
			return Config{}, err
		}
		listKey = key
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyConfigValue(cfg *Config, section string, key string, value string) error {
	switch section {
	case "":
		switch key {
		case "Name":
			cfg.Name = value
		case "RuntimeMode":
			cfg.RuntimeMode = value
		case "Host":
			cfg.Host = value
		case "Port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("Port 配置必须是数字: %w", err)
			}
			cfg.Port = port
		}
	case "Postgres":
		switch key {
		case "DSN":
			cfg.Postgres.DSN = value
		case "MaxOpenConns":
			size, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("MaxOpenConns 配置必须是数字: %w", err)
			}
			cfg.Postgres.MaxOpenConns = size
		case "MaxIdleConns":
			size, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("MaxIdleConns 配置必须是数字: %w", err)
			}
			cfg.Postgres.MaxIdleConns = size
		case "ConnMaxLifetime":
			lifetime, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("ConnMaxLifetime 配置不正确: %w", err)
			}
			cfg.Postgres.ConnMaxLifetime = lifetime
		case "ConnMaxIdleTime":
			idleTime, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("ConnMaxIdleTime 配置不正确: %w", err)
			}
			cfg.Postgres.ConnMaxIdleTime = idleTime
		}
	case "AdminAuth":
		switch key {
		case "TokenSecret":
			cfg.AdminAuth.TokenSecret = value
		case "TokenTTL":
			ttl, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("TokenTTL 配置不正确: %w", err)
			}
			cfg.AdminAuth.TokenTTL = ttl
		}
	case "Wechat":
		switch key {
		case "AppID":
			cfg.Wechat.AppID = value
		case "AppSecret":
			cfg.Wechat.AppSecret = value
		case "AllowDevCode":
			allow, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("AllowDevCode 配置必须是布尔值: %w", err)
			}
			cfg.Wechat.AllowDevCode = allow
		}
	case "SMS":
		switch key {
		case "Provider":
			cfg.SMS.Provider = value
		case "SendURL":
			cfg.SMS.SendURL = value
		case "VerifyURL":
			cfg.SMS.VerifyURL = value
		case "SendMinInterval":
			interval, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("SendMinInterval 配置不正确: %w", err)
			}
			cfg.SMS.SendMinInterval = interval
		case "DailySendLimit":
			limit, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("DailySendLimit 配置必须是数字: %w", err)
			}
			cfg.SMS.DailySendLimit = limit
		case "AccessKeyID":
			cfg.SMS.AccessKeyID = value
		case "AccessKeySecret":
			cfg.SMS.AccessKeySecret = value
		case "SignName":
			cfg.SMS.SignName = value
		case "TemplateCode":
			cfg.SMS.TemplateCode = value
		case "DevCode":
			cfg.SMS.DevCode = value
		}
	case "Tasks":
		switch key {
		case "ResourceLifecycleInterval":
			interval, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("ResourceLifecycleInterval 配置不正确: %w", err)
			}
			cfg.Tasks.ResourceLifecycleInterval = interval
		}
	case "Storage":
		return applyStorageValue(&cfg.Storage, key, value)
	}
	return nil
}

func applyStorageValue(cfg *StorageConfig, key string, value string) error {
	switch key {
	case "Provider":
		cfg.Provider = value
	case "Endpoint":
		cfg.Endpoint = value
	case "Bucket":
		cfg.Bucket = value
	case "Region":
		cfg.Region = value
	case "AccessKeyID":
		cfg.AccessKeyID = value
	case "AccessKeySecret":
		cfg.AccessKeySecret = value
	case "PublicBaseURL":
		cfg.PublicBaseURL = value
	case "UploadExpire":
		expire, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("UploadExpire 配置不正确: %w", err)
		}
		cfg.UploadExpire = expire
	case "MaxFileSizeBytes":
		size, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("MaxFileSizeBytes 配置必须是数字: %w", err)
		}
		cfg.MaxFileSizeBytes = size
	case "AllowedContentTypes":
		cfg.AllowedContentTypes = nil
	}
	return nil
}

func isTopLevelLine(line string) bool {
	return len(line) == len(strings.TrimLeft(line, " \t"))
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return os.ExpandEnv(value)
}
