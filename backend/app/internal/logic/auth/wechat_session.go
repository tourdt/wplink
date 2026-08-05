package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultWechatCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"
const defaultWechatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
const defaultWechatPhoneNumberURL = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"

type wechatFailureCause string

const (
	wechatFailureCauseCanceled          wechatFailureCause = "canceled"
	wechatFailureCauseTimeout           wechatFailureCause = "timeout"
	wechatFailureCauseDNS               wechatFailureCause = "dns"
	wechatFailureCauseTLS               wechatFailureCause = "tls"
	wechatFailureCauseConnectionRefused wechatFailureCause = "connection_refused"
	wechatFailureCauseConnectionReset   wechatFailureCause = "connection_reset"
	wechatFailureCauseUnexpectedEOF     wechatFailureCause = "unexpected_eof"
	wechatFailureCauseNetwork           wechatFailureCause = "network"
	wechatFailureCauseOther             wechatFailureCause = "other"
)

// isLegacyWechatDevCode 识别已废弃的本地开发凭证，防止伪造身份进入真实微信审核链路。
func isLegacyWechatDevCode(code string) bool {
	return strings.HasPrefix(strings.TrimSpace(code), "local-dev-")
}

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
	observer       externalcall.Observer
}

func NewWechatSessionClient(cfg config.WechatConfig, baseURL string, client *http.Client, observers ...externalcall.Observer) *HTTPWechatSessionClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultWechatCode2SessionURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	// variadic 仅用于保持旧调用兼容：这是可选单 Observer，传入多个时只使用第一个。
	var observer externalcall.Observer
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &HTTPWechatSessionClient{
		cfg:            cfg,
		baseURL:        baseURL,
		accessTokenURL: defaultWechatAccessTokenURL,
		phoneNumberURL: defaultWechatPhoneNumberURL,
		client:         client,
		observer:       observer,
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
	if isLegacyWechatDevCode(code) {
		logx.Errorf("微信登录拒绝历史开发凭证: isLegacyDevCode=true")
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "请在微信内重新登录")
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
		// 请求错误可能引用带凭据的完整 URL，因此日志只保留固定分类。
		logx.Errorf("创建微信登录请求失败: cause=request_create")
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录请求创建失败，请稍后重试")
	}
	// 请求创建成功才开始观测，避免把本地校验或配置错误误计为微信故障。
	call := externalcall.Start(c.observer, externalcall.ProviderWechat, externalcall.OperationCodeToSession)
	resp, err := c.client.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		logWechatTransportFailure("微信登录", outcome, ctx, err, 0)
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录服务暂不可用，请稍后重试")
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, statusCode)
		logWechatTransportFailure("微信登录响应读取", externalcall.OutcomeTransportError, ctx, err, statusCode)
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录响应读取失败，请稍后重试")
	}
	// HTTP 状态是传输协议层的首要终态；后续仍沿用原响应解析与公开错误映射。
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		call.Finish(ctx, externalcall.OutcomeHTTPError, statusCode)
		logx.Errorf("微信登录 HTTP 状态异常: status=%d", statusCode)
	}

	var data struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int64  `json:"errcode"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("解析微信登录响应失败: cause=json_invalid status=%d", statusCode)
		return WechatSession{}, errx.New(errx.CodeInternalError, "微信登录响应异常，请稍后重试")
	}
	if data.ErrCode != 0 {
		call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
		// errmsg 由外部供应商控制，可能回显请求内容；仅记录稳定错误码。
		logx.Errorf("微信登录接口返回错误: errCode=%d", data.ErrCode)
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	if strings.TrimSpace(data.OpenID) == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("微信登录接口未返回 openid")
		return WechatSession{}, errx.New(errx.CodeUnauthorized, "微信登录凭证无效，请重新登录")
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
	return WechatSession{OpenID: data.OpenID, UnionID: data.UnionID}, nil
}

func (c *HTTPWechatSessionClient) GetPhoneNumber(ctx context.Context, code string) (WechatPhoneNumber, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return WechatPhoneNumber{}, errx.New(errx.CodeValidationFailed, "请授权微信手机号")
	}
	if isLegacyWechatDevCode(code) {
		logx.Errorf("微信手机号授权拒绝历史开发凭证: isLegacyDevCode=true")
		return WechatPhoneNumber{}, errx.New(errx.CodeUnauthorized, "请在微信内重新授权手机号")
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
		logx.Errorf("创建微信手机号请求失败: cause=request_create")
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号请求创建失败，请手动填写")
	}
	req.Header.Set("Content-Type", "application/json")
	// token 获取与手机号获取是两次独立外呼，各自提交且仅提交一个终态。
	call := externalcall.Start(c.observer, externalcall.ProviderWechat, externalcall.OperationPhoneNumber)
	resp, err := c.client.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		logWechatTransportFailure("微信手机号", outcome, ctx, err, 0)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号服务暂不可用，请手动填写")
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, statusCode)
		logWechatTransportFailure("微信手机号响应读取", externalcall.OutcomeTransportError, ctx, err, statusCode)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号响应读取失败，请手动填写")
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		call.Finish(ctx, externalcall.OutcomeHTTPError, statusCode)
		logx.Errorf("微信手机号 HTTP 状态异常: status=%d", statusCode)
	}

	var data struct {
		ErrCode   int64 `json:"errcode"`
		PhoneInfo struct {
			PhoneNumber     string `json:"phoneNumber"`
			PurePhoneNumber string `json:"purePhoneNumber"`
			CountryCode     string `json:"countryCode"`
		} `json:"phone_info"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("解析微信手机号响应失败: cause=json_invalid status=%d", statusCode)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "微信手机号响应异常，请手动填写")
	}
	if data.ErrCode != 0 {
		call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
		logx.Errorf("微信手机号接口返回错误: errCode=%d", data.ErrCode)
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	if strings.TrimSpace(data.PhoneInfo.PurePhoneNumber) == "" && strings.TrimSpace(data.PhoneInfo.PhoneNumber) == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("微信手机号接口未返回号码")
		return WechatPhoneNumber{}, errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
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
		logx.Errorf("创建微信 access_token 请求失败: cause=request_create")
		return "", errx.New(errx.CodeInternalError, "微信手机号请求创建失败，请手动填写")
	}
	call := externalcall.Start(c.observer, externalcall.ProviderWechat, externalcall.OperationAccessToken)
	resp, err := c.client.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		logWechatTransportFailure("微信 access_token", outcome, ctx, err, 0)
		return "", errx.New(errx.CodeInternalError, "微信手机号服务暂不可用，请手动填写")
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, statusCode)
		logWechatTransportFailure("微信 access_token 响应读取", externalcall.OutcomeTransportError, ctx, err, statusCode)
		return "", errx.New(errx.CodeInternalError, "微信手机号响应读取失败，请手动填写")
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		call.Finish(ctx, externalcall.OutcomeHTTPError, statusCode)
		logx.Errorf("微信 access_token HTTP 状态异常: status=%d", statusCode)
	}

	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int64  `json:"errcode"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("解析微信 access_token 响应失败: cause=json_invalid status=%d", statusCode)
		return "", errx.New(errx.CodeInternalError, "微信手机号响应异常，请手动填写")
	}
	if data.ErrCode != 0 {
		call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
		logx.Errorf("微信 access_token 接口返回错误: errCode=%d", data.ErrCode)
		return "", errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
		logx.Errorf("微信 access_token 接口未返回 token: expiresIn=%d", data.ExpiresIn)
		return "", errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
	return data.AccessToken, nil
}

func logWechatTransportFailure(operation string, outcome externalcall.Outcome, ctx context.Context, err error, statusCode int) {
	cause := classifyWechatFailureCause(ctx, err)
	if statusCode > 0 {
		logx.Errorf("%s失败: outcome=%s cause=%s status=%d", operation, outcome, cause, statusCode)
		return
	}
	logx.Errorf("%s失败: outcome=%s cause=%s", operation, outcome, cause)
}

func classifyWechatFailureCause(ctx context.Context, err error) wechatFailureCause {
	// http.Client 的 *url.Error 会携带完整 URL/query；先只取底层错误，再做白名单枚举分类。
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		err = urlErr.Err
	}
	switch {
	case errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)):
		return wechatFailureCauseCanceled
	case errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)):
		return wechatFailureCauseTimeout
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return wechatFailureCauseDNS
	}
	if isWechatTLSFailure(err) {
		return wechatFailureCauseTLS
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return wechatFailureCauseConnectionRefused
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return wechatFailureCauseConnectionReset
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return wechatFailureCauseUnexpectedEOF
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return wechatFailureCauseTimeout
		}
		return wechatFailureCauseNetwork
	}
	return wechatFailureCauseOther
}

func isWechatTLSFailure(err error) bool {
	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		return true
	}
	var certificateInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certificateInvalidErr) {
		return true
	}
	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return true
	}
	var recordHeaderErr tls.RecordHeaderError
	return errors.As(err, &recordHeaderErr)
}
