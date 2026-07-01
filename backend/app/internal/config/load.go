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

	cfg := Config{Log: defaultLogConfig()}
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
	case "WechatPay":
		return applyWechatPayValue(&cfg.WechatPay, key, value)
	case "Log":
		return applyLogValue(&cfg.Log, key, value)
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

func defaultLogConfig() LogConfig {
	return LogConfig{
		Mode:     "file",
		Encoding: "json",
		Path:     "logs",
		Level:    "info",
		Rotation: "daily",
		KeepDays: 7,
		Stat:     true,
	}
}

func applyLogValue(cfg *LogConfig, key string, value string) error {
	switch key {
	case "ServiceName":
		cfg.ServiceName = value
	case "Mode":
		cfg.Mode = value
	case "Encoding":
		cfg.Encoding = value
	case "TimeFormat":
		cfg.TimeFormat = value
	case "Path":
		cfg.Path = value
	case "Level":
		cfg.Level = value
	case "MaxContentLength":
		length, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("Log.MaxContentLength 配置必须是数字: %w", err)
		}
		cfg.MaxContentLength = uint32(length)
	case "Compress":
		compress, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("Log.Compress 配置必须是布尔值: %w", err)
		}
		cfg.Compress = compress
	case "Stat":
		stat, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("Log.Stat 配置必须是布尔值: %w", err)
		}
		cfg.Stat = stat
	case "KeepDays":
		days, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Log.KeepDays 配置必须是数字: %w", err)
		}
		cfg.KeepDays = days
	case "StackCooldownMillis":
		millis, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Log.StackCooldownMillis 配置必须是数字: %w", err)
		}
		cfg.StackCooldownMillis = millis
	case "MaxBackups":
		count, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Log.MaxBackups 配置必须是数字: %w", err)
		}
		cfg.MaxBackups = count
	case "MaxSize":
		size, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Log.MaxSize 配置必须是数字: %w", err)
		}
		cfg.MaxSize = size
	case "Rotation":
		cfg.Rotation = value
	case "FileTimeFormat":
		cfg.FileTimeFormat = value
	}
	return nil
}

func applyWechatPayValue(cfg *WechatPayConfig, key string, value string) error {
	switch key {
	case "Enabled":
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("WechatPay.Enabled 配置必须是布尔值: %w", err)
		}
		cfg.Enabled = enabled
	case "DevMockEnabled":
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("WechatPay.DevMockEnabled 配置必须是布尔值: %w", err)
		}
		cfg.DevMockEnabled = enabled
	case "MchID":
		cfg.MchID = value
	case "AppID":
		cfg.AppID = value
	case "APIv3Key":
		cfg.APIv3Key = value
	case "MerchantSerialNo":
		cfg.MerchantSerialNo = value
	case "MerchantPrivateKeyPath":
		cfg.MerchantPrivateKeyPath = value
	case "PlatformPublicKeyPath":
		cfg.PlatformPublicKeyPath = value
	case "NotifyURL":
		cfg.NotifyURL = value
	case "RequestTimeout":
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("WechatPay.RequestTimeout 配置不正确: %w", err)
		}
		cfg.RequestTimeout = timeout
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
