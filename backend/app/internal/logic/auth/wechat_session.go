package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

const defaultWechatCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

type WechatSession struct {
	OpenID  string
	UnionID string
}

type WechatSessionClient interface {
	Code2Session(ctx context.Context, code string) (WechatSession, error)
}

type HTTPWechatSessionClient struct {
	cfg     config.WechatConfig
	baseURL string
	client  *http.Client
}

func NewWechatSessionClient(cfg config.WechatConfig, baseURL string, client *http.Client) *HTTPWechatSessionClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultWechatCode2SessionURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPWechatSessionClient{cfg: cfg, baseURL: baseURL, client: client}
}

func (c *HTTPWechatSessionClient) Code2Session(ctx context.Context, code string) (WechatSession, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return WechatSession{}, errx.New(errx.CodeValidationFailed, "请提供微信登录凭证")
	}
	if c.cfg.AllowDevCode && strings.HasPrefix(code, "local-dev-") {
		return WechatSession{OpenID: "dev:" + code}, nil
	}
	if strings.TrimSpace(c.cfg.AppID) == "" || strings.TrimSpace(c.cfg.AppSecret) == "" {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录服务未配置，请稍后重试")
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录服务配置错误，请稍后重试")
	}
	query := endpoint.Query()
	query.Set("appid", c.cfg.AppID)
	query.Set("secret", c.cfg.AppSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录请求创建失败，请稍后重试")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录服务暂不可用，请稍后重试")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录响应读取失败，请稍后重试")
	}

	var data struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int64  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录响应异常，请稍后重试")
	}
	if data.ErrCode != 0 {
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	if strings.TrimSpace(data.OpenID) == "" {
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	return WechatSession{OpenID: data.OpenID, UnionID: data.UnionID}, nil
}
