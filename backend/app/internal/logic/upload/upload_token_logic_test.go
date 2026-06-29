package upload

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
)

func TestCreateUploadTokenSignsQiniuPolicy(t *testing.T) {
	logic := NewUploadTokenLogic(config.StorageConfig{
		Provider:            "qiniu-kodo",
		Endpoint:            "https://upload-z2.qiniup.com",
		Bucket:              "wplink-test",
		AccessKeyID:         "ak-test",
		AccessKeySecret:     "sk-test",
		PublicBaseURL:       "https://cdn.example.com",
		UploadExpire:        time.Minute,
		MaxFileSizeBytes:    1024,
		AllowedContentTypes: []string{"image/png"},
	})

	resp, err := logic.CreateUploadToken(context.Background(), CreateUploadTokenReq{
		Purpose:     "resource",
		FileName:    "sample.png",
		ContentType: "image/png",
		FileSize:    128,
	})
	if err != nil {
		t.Fatalf("CreateUploadToken() error = %v", err)
	}

	parts := strings.Split(resp.UploadToken, ":")
	if len(parts) != 3 || parts[0] != "ak-test" {
		t.Fatalf("upload token = %q, want qiniu accessKey:sign:policy", resp.UploadToken)
	}
	policyBytes, err := base64.URLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	expectedEncodedPolicy := base64.URLEncoding.EncodeToString(policyBytes)
	if parts[2] != expectedEncodedPolicy {
		t.Fatalf("encoded policy = %q, want qiniu padded url base64 %q", parts[2], expectedEncodedPolicy)
	}
	mac := hmac.New(sha1.New, []byte("sk-test"))
	_, _ = mac.Write([]byte(expectedEncodedPolicy))
	expectedSign := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	if parts[1] != expectedSign {
		t.Fatalf("sign = %q, want qiniu padded url base64 sign %q", parts[1], expectedSign)
	}
	var policy map[string]interface{}
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		t.Fatalf("unmarshal policy: %v", err)
	}
	if policy["scope"] != "wplink-test:"+resp.ObjectKey {
		t.Fatalf("policy = %#v, want scoped object key", policy)
	}
	if resp.UploadURL != "https://upload-z2.qiniup.com" || resp.PublicBaseURL != "https://cdn.example.com" {
		t.Fatalf("resp = %#v, want configured endpoints", resp)
	}
	if !strings.HasPrefix(resp.ObjectKey, "wplink/uploads/resource/") || !strings.HasSuffix(resp.ObjectKey, ".png") {
		t.Fatalf("object key = %q, want wplink resource png key", resp.ObjectKey)
	}
}

func TestCreateUploadTokenRejectsUnsupportedContentType(t *testing.T) {
	logic := NewUploadTokenLogic(config.StorageConfig{
		Provider:            "qiniu-kodo",
		Endpoint:            "https://upload-z2.qiniup.com",
		Bucket:              "wplink-test",
		AccessKeyID:         "ak-test",
		AccessKeySecret:     "sk-test",
		AllowedContentTypes: []string{"image/png"},
	})

	_, err := logic.CreateUploadToken(context.Background(), CreateUploadTokenReq{
		Purpose:     "resource",
		FileName:    "sample.gif",
		ContentType: "image/gif",
		FileSize:    128,
	})
	if err == nil {
		t.Fatal("CreateUploadToken() error = nil, want unsupported content type error")
	}
}

func TestCreateUploadTokenAcceptsMatchingZ0EndpointRegion(t *testing.T) {
	logic := NewUploadTokenLogic(config.StorageConfig{
		Provider:        "qiniu-kodo",
		Endpoint:        "https://up-z0.qiniup.com",
		Bucket:          "wplink-test",
		Region:          "z0",
		AccessKeyID:     "ak-test",
		AccessKeySecret: "sk-test",
		PublicBaseURL:   "https://cdn.example.com",
	})

	resp, err := logic.CreateUploadToken(context.Background(), CreateUploadTokenReq{
		Purpose:     "banner",
		FileName:    "sample.png",
		ContentType: "image/png",
		FileSize:    128,
	})
	if err != nil {
		t.Fatalf("CreateUploadToken() error = %v, want nil", err)
	}
	if resp.UploadURL != "https://up-z0.qiniup.com" {
		t.Fatalf("upload url = %q, want z0 endpoint", resp.UploadURL)
	}
}

func TestCreateUploadTokenRejectsRegionEndpointMismatch(t *testing.T) {
	logic := NewUploadTokenLogic(config.StorageConfig{
		Provider:        "qiniu-kodo",
		Endpoint:        "https://up-z0.qiniup.com",
		Bucket:          "wplink-test",
		Region:          "z2",
		AccessKeyID:     "ak-test",
		AccessKeySecret: "sk-test",
		PublicBaseURL:   "https://cdn.example.com",
	})

	_, err := logic.CreateUploadToken(context.Background(), CreateUploadTokenReq{
		Purpose:     "banner",
		FileName:    "sample.png",
		ContentType: "image/png",
		FileSize:    128,
	})
	if err == nil {
		t.Fatal("CreateUploadToken() error = nil, want region endpoint mismatch error")
	}
}
