package upload

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

type CreateUploadTokenReq struct {
	Purpose     string `json:"purpose"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	FileSize    int64  `json:"fileSize"`
}

type CreateUploadTokenResp struct {
	UploadToken   string `json:"uploadToken"`
	UploadURL     string `json:"uploadUrl"`
	PublicBaseURL string `json:"publicBaseUrl"`
	ObjectKey     string `json:"objectKey"`
	ExpiresAt     string `json:"expiresAt"`
}

type UploadTokenLogic struct {
	cfg config.StorageConfig
}

func NewUploadTokenLogic(cfg config.StorageConfig) *UploadTokenLogic {
	return &UploadTokenLogic{cfg: cfg}
}

func (l *UploadTokenLogic) CreateUploadToken(_ context.Context, req CreateUploadTokenReq) (CreateUploadTokenResp, error) {
	if err := l.validate(req); err != nil {
		return CreateUploadTokenResp{}, err
	}

	expire := l.cfg.UploadExpire
	if expire <= 0 {
		expire = 15 * time.Minute
	}
	expiresAt := time.Now().Add(expire).UTC()
	objectKey := buildObjectKey(req.Purpose, req.FileName)
	policy := map[string]interface{}{
		"scope":     l.cfg.Bucket + ":" + objectKey,
		"deadline":  expiresAt.Unix(),
		"mimeLimit": strings.TrimSpace(req.ContentType),
	}
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return CreateUploadTokenResp{}, errx.New(errx.CodeInternalError, "生成上传凭证失败，请稍后重试")
	}
	encodedPolicy := base64.URLEncoding.EncodeToString(policyBytes)
	mac := hmac.New(sha1.New, []byte(l.cfg.AccessKeySecret))
	_, _ = mac.Write([]byte(encodedPolicy))
	sign := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	return CreateUploadTokenResp{
		UploadToken:   l.cfg.AccessKeyID + ":" + sign + ":" + encodedPolicy,
		UploadURL:     strings.TrimRight(l.cfg.Endpoint, "/"),
		PublicBaseURL: strings.TrimRight(l.cfg.PublicBaseURL, "/"),
		ObjectKey:     objectKey,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	}, nil
}

func (l *UploadTokenLogic) validate(req CreateUploadTokenReq) error {
	if l.cfg.Provider != "qiniu-kodo" {
		return errx.New(errx.CodeInternalError, "上传服务未配置，请稍后重试")
	}
	if strings.TrimSpace(l.cfg.Endpoint) == "" || strings.TrimSpace(l.cfg.Bucket) == "" || strings.TrimSpace(l.cfg.AccessKeyID) == "" || strings.TrimSpace(l.cfg.AccessKeySecret) == "" {
		return errx.New(errx.CodeInternalError, "上传服务未配置，请稍后重试")
	}
	if !qiniuEndpointMatchesRegion(l.cfg.Endpoint, l.cfg.Region) {
		return errx.New(errx.CodeInternalError, "上传服务区域配置不一致，请联系管理员处理")
	}
	if strings.TrimSpace(req.Purpose) == "" {
		return errx.New(errx.CodeValidationFailed, "请提供上传用途")
	}
	if strings.TrimSpace(req.FileName) == "" {
		return errx.New(errx.CodeValidationFailed, "请提供文件名")
	}
	if strings.TrimSpace(req.ContentType) == "" {
		return errx.New(errx.CodeValidationFailed, "请提供文件类型")
	}
	if req.FileSize <= 0 {
		return errx.New(errx.CodeValidationFailed, "请提供文件大小")
	}
	if l.cfg.MaxFileSizeBytes > 0 && req.FileSize > l.cfg.MaxFileSizeBytes {
		return errx.New(errx.CodeValidationFailed, "文件过大，请压缩后再上传")
	}
	if len(l.cfg.AllowedContentTypes) > 0 && !containsContentType(l.cfg.AllowedContentTypes, req.ContentType) {
		return errx.New(errx.CodeValidationFailed, "暂不支持该文件类型")
	}
	return nil
}

func qiniuEndpointMatchesRegion(endpoint string, region string) bool {
	region = strings.TrimSpace(strings.ToLower(region))
	if region == "" {
		return true
	}
	endpointRegion := qiniuEndpointRegion(endpoint)
	return endpointRegion == "" || endpointRegion == region
}

func qiniuEndpointRegion(endpoint string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "up.qiniup.com", "upload.qiniup.com", "up-z0.qiniup.com", "upload-z0.qiniup.com":
		return "z0"
	case "up-z1.qiniup.com", "upload-z1.qiniup.com":
		return "z1"
	case "up-z2.qiniup.com", "upload-z2.qiniup.com":
		return "z2"
	case "up-na0.qiniup.com", "upload-na0.qiniup.com":
		return "na0"
	case "up-as0.qiniup.com", "upload-as0.qiniup.com":
		return "as0"
	default:
		return ""
	}
}

func containsContentType(allowed []string, contentType string) bool {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	for _, item := range allowed {
		if strings.TrimSpace(strings.ToLower(item)) == contentType {
			return true
		}
	}
	return false
}

func buildObjectKey(purpose string, fileName string) string {
	purpose = sanitizeKeyPart(purpose)
	if purpose == "" {
		purpose = "misc"
	}
	ext := strings.ToLower(filepath.Ext(path.Base(fileName)))
	return "wplink/uploads/" + purpose + "/" + time.Now().UTC().Format("20060102") + "/" + randomHex(12) + ext
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf)
}
