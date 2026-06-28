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

type UserTokenSubject struct {
	UserID string
	Roles  []string
}

type HMACUserTokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewHMACUserTokenService(secret string, ttl time.Duration) *HMACUserTokenService {
	return &HMACUserTokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (s *HMACUserTokenService) IssueUserToken(_ context.Context, subject UserTokenSubject) (string, error) {
	if len(s.secret) == 0 {
		return "", errors.New("用户 token 密钥未配置")
	}
	if strings.TrimSpace(subject.UserID) == "" {
		return "", errors.New("用户 token 用户不能为空")
	}

	now := time.Now()
	payload := map[string]interface{}{
		"sub":   subject.UserID,
		"roles": subject.Roles,
		"typ":   "user",
		"iat":   now.Unix(),
		"exp":   now.Add(s.ttl).Unix(),
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
	signature := signUserToken(signingInput, s.secret)
	return signingInput + "." + signature, nil
}

func (s *HMACUserTokenService) ParseUserToken(_ context.Context, token string) (UserTokenSubject, error) {
	if len(s.secret) == 0 {
		return UserTokenSubject{}, errors.New("用户 token 密钥未配置")
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := signUserToken(signingInput, s.secret)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}
	var payload struct {
		Sub   string   `json:"sub"`
		Roles []string `json:"roles"`
		Typ   string   `json:"typ"`
		Exp   int64    `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}
	if payload.Typ != "user" || strings.TrimSpace(payload.Sub) == "" {
		return UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}
	if payload.Exp > 0 && time.Now().Unix() >= payload.Exp {
		return UserTokenSubject{}, errors.New("登录已过期，请重新登录")
	}

	return UserTokenSubject{UserID: payload.Sub, Roles: payload.Roles}, nil
}

func signUserToken(signingInput string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
