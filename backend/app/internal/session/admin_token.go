package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type AdminTokenSubject struct {
	UserID string
	Roles  []string
}

type HMACAdminTokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewHMACAdminTokenIssuer(secret string, ttl time.Duration) *HMACAdminTokenIssuer {
	return &HMACAdminTokenIssuer{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (i *HMACAdminTokenIssuer) IssueAdminToken(_ context.Context, subject AdminTokenSubject) (string, error) {
	if len(i.secret) == 0 {
		return "", errors.New("后台 token 密钥未配置")
	}
	if strings.TrimSpace(subject.UserID) == "" {
		return "", errors.New("后台 token 用户不能为空")
	}

	now := time.Now()
	payload := map[string]interface{}{
		"sub":   subject.UserID,
		"roles": subject.Roles,
		"iat":   now.Unix(),
		"exp":   now.Add(i.ttl).Unix(),
	}
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerPart, err := encodeTokenPart(header)
	if err != nil {
		return "", err
	}
	payloadPart, err := encodeTokenPart(payload)
	if err != nil {
		return "", err
	}

	signingInput := headerPart + "." + payloadPart
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func encodeTokenPart(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
