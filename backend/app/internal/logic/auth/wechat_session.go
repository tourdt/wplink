package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultWechatCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"
const defaultWechatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
const defaultWechatPhoneNumberURL = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"

type WechatSession struct {
	OpenID  string
	UnionID string
}

type WechatPhoneNumber struct {
	PhoneNumber     string
	PurePhoneNumber string
	CountryCode     string
}

type WechatSessionClient interface {
	Code2Session(ctx context.Context, code string) (WechatSession, error)
	GetPhoneNumber(ctx context.Context, code string) (WechatPhoneNumber, error)
}

type HTTPWechatSessionClient struct {
	cfg            config.WechatConfig
	baseURL        string
	accessTokenURL string
	phoneNumberURL string
	client         *http.Client
}

func NewWechatSessionClient(cfg config.WechatConfig, baseURL string, client *http.Client) *HTTPWechatSessionClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultWechatCode2SessionURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPWechatSessionClient{
		cfg:            cfg,
		baseURL:        baseURL,
		accessTokenURL: defaultWechatAccessTokenURL,
		phoneNumberURL: defaultWechatPhoneNumberURL,
		client:         client,
	}
}

func (c *HTTPWechatSessionClient) WithPhoneNumberURLs(accessTokenURL string, phoneNumberURL string) *HTTPWechatSessionClient {
	if strings.TrimSpace(accessTokenURL) != "" {
		c.accessTokenURL = accessTokenURL
	}
	if strings.TrimSpace(phoneNumberURL) != "" {
		c.phoneNumberURL = phoneNumberURL
	}
	return c
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
		logx.Errorf("微信登录接口返回错误: errCode=%d errMsg=%s isDevCode=%t allowDevCode=%t", data.ErrCode, data.ErrMsg, strings.HasPrefix(code, "local-dev-"), c.cfg.AllowDevCode)
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	if strings.TrimSpace(data.OpenID) == "" {
		logx.Errorf("微信登录接口未返回 openid: isDevCode=%t allowDevCode=%t", strings.HasPrefix(code, "local-dev-"), c.cfg.AllowDevCode)
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	return WechatSession{OpenID: data.OpenID, UnionID: data.UnionID}, nil
}

func (c *HTTPWechatSessionClient) GetPhoneNumber(ctx context.Context, code string) (WechatPhoneNumber, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return WechatPhoneNumber{}, errx.New(errx.CodeValidationFailed, "请授权微信手机号")
	}
	if c.cfg.AllowDevCode && strings.HasPrefix(code, "local-dev-phone-") {
		phone := sanitizeWechatPhoneNumber(strings.TrimPrefix(code, "local-dev-phone-"))
		if phone == "" {
			phone = "18800000000"
		}
		return WechatPhoneNumber{PhoneNumber: phone, PurePhoneNumber: phone, CountryCode: "86"}, nil
	}
	if strings.TrimSpace(c.cfg.AppID) == "" || strings.TrimSpace(c.cfg.AppSecret) == "" {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号服务未配置，请手动填写")
	}

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return WechatPhoneNumber{}, err
	}
	endpoint, err := url.Parse(c.phoneNumberURL)
	if err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号服务配置错误，请手动填写")
	}
	query := endpoint.Query()
	query.Set("access_token", accessToken)
	endpoint.RawQuery = query.Encode()

	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(map[string]string{"code": code}); err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号请求创建失败，请手动填写")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), &payload)
	if err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号请求创建失败，请手动填写")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号服务暂不可用，请手动填写")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号响应读取失败，请手动填写")
	}

	var data struct {
		ErrCode   int64  `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
		PhoneInfo struct {
			PhoneNumber     string `json:"phoneNumber"`
			PurePhoneNumber string `json:"purePhoneNumber"`
			CountryCode     string `json:"countryCode"`
		} `json:"phone_info"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号响应异常，请手动填写")
	}
	if data.ErrCode != 0 {
		logx.Errorf("微信手机号接口返回错误: errCode=%d errMsg=%s allowDevCode=%t", data.ErrCode, data.ErrMsg, c.cfg.AllowDevCode)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	if strings.TrimSpace(data.PhoneInfo.PurePhoneNumber) == "" && strings.TrimSpace(data.PhoneInfo.PhoneNumber) == "" {
		logx.Errorf("微信手机号接口未返回号码: allowDevCode=%t", c.cfg.AllowDevCode)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	return WechatPhoneNumber{
		PhoneNumber:     data.PhoneInfo.PhoneNumber,
		PurePhoneNumber: data.PhoneInfo.PurePhoneNumber,
		CountryCode:     data.PhoneInfo.CountryCode,
	}, nil
}

func (c *HTTPWechatSessionClient) getAccessToken(ctx context.Context) (string, error) {
	endpoint, err := url.Parse(c.accessTokenURL)
	if err != nil {
		return "", errx.New(errx.CodeInternalError, "微信手机号服务配置错误，请手动填写")
	}
	query := endpoint.Query()
	query.Set("appid", c.cfg.AppID)
	query.Set("secret", c.cfg.AppSecret)
	query.Set("grant_type", "client_credential")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", errx.New(errx.CodeInternalError, "微信手机号请求创建失败，请手动填写")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", errx.New(errx.CodeInternalError, "微信手机号服务暂不可用，请手动填写")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", errx.New(errx.CodeInternalError, "微信手机号响应读取失败，请手动填写")
	}

	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int64  `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", errx.New(errx.CodeInternalError, "微信手机号响应异常，请手动填写")
	}
	if data.ErrCode != 0 {
		logx.Errorf("微信 access_token 接口返回错误: errCode=%d errMsg=%s", data.ErrCode, data.ErrMsg)
		return "", errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		logx.Errorf("微信 access_token 接口未返回 token: expiresIn=%d", data.ExpiresIn)
		return "", errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	return data.AccessToken, nil
}

func sanitizeWechatPhoneNumber(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
