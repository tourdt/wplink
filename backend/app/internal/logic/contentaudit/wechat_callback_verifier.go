package contentaudit

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"wplink/backend/common/errx"
)

var ErrWechatCallbackReplay = errors.New("微信回调重复请求")

type WechatCallbackVerifier interface {
	Verify(signature string, timestamp string, nonce string, remember bool) error
}

type SHA1WechatCallbackVerifier struct {
	token   string
	maxSkew time.Duration
	now     func() time.Time

	mu     sync.Mutex
	recent map[string]time.Time
}

func NewSHA1WechatCallbackVerifier(token string, maxSkew time.Duration) *SHA1WechatCallbackVerifier {
	if maxSkew <= 0 {
		maxSkew = 5 * time.Minute
	}
	return &SHA1WechatCallbackVerifier{
		token:   strings.TrimSpace(token),
		maxSkew: maxSkew,
		now:     time.Now,
		recent:  make(map[string]time.Time),
	}
}

func (v *SHA1WechatCallbackVerifier) Verify(signature string, timestamp string, nonce string, remember bool) error {
	if v == nil || v.token == "" {
		return errx.New(errx.CodeInternalError, "微信内容审核回调验签未配置")
	}
	signature = strings.ToLower(strings.TrimSpace(signature))
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	if signature == "" || timestamp == "" || nonce == "" {
		return errx.New(errx.CodeUnauthorized, "微信内容审核回调签名参数缺失")
	}
	unixSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errx.New(errx.CodeUnauthorized, "微信内容审核回调时间戳不正确")
	}
	callbackTime := time.Unix(unixSeconds, 0)
	now := v.now()
	if callbackTime.Before(now.Add(-v.maxSkew)) || callbackTime.After(now.Add(v.maxSkew)) {
		return errx.New(errx.CodeUnauthorized, "微信内容审核回调已过期")
	}

	values := []string{v.token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	expected := hex.EncodeToString(sum[:])
	if len(signature) != len(expected) ||
		subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return errx.New(errx.CodeUnauthorized, "微信内容审核回调验签失败")
	}
	fingerprint := signature + ":" + timestamp + ":" + nonce
	v.mu.Lock()
	defer v.mu.Unlock()
	for key, expiresAt := range v.recent {
		if !expiresAt.After(now) {
			delete(v.recent, key)
		}
	}
	if expiresAt, exists := v.recent[fingerprint]; exists && expiresAt.After(now) {
		return ErrWechatCallbackReplay
	}
	if remember {
		v.recent[fingerprint] = now.Add(v.maxSkew)
	}
	return nil
}
