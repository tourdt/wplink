package contentaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"wplink/backend/app/internal/config"
	resourcelogic "wplink/backend/app/internal/logic/resource"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultWechatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
	defaultWechatMsgSecCheckURL = "https://api.weixin.qq.com/wxa/msg_sec_check"
	defaultWechatMediaCheckURL  = "https://api.weixin.qq.com/wxa/media_check_async"
)

type WechatAuditor struct {
	wechat         config.WechatConfig
	cfg            config.ContentAuditConfig
	mediaBaseURL   string
	client         *http.Client
	accessTokenURL string
	msgSecCheckURL string
	mediaCheckURL  string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewWechatAuditor(wechat config.WechatConfig, cfg config.ContentAuditConfig, mediaBaseURL string, client *http.Client) *WechatAuditor {
	if !cfg.Enabled {
		return nil
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if cfg.TextScene == 0 {
		cfg.TextScene = 3
	}
	if cfg.MediaScene == 0 {
		cfg.MediaScene = cfg.TextScene
	}
	if cfg.MaxTextChars == 0 {
		cfg.MaxTextChars = 2500
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &WechatAuditor{
		wechat:         wechat,
		cfg:            cfg,
		mediaBaseURL:   strings.TrimRight(strings.TrimSpace(mediaBaseURL), "/"),
		client:         client,
		accessTokenURL: defaultWechatAccessTokenURL,
		msgSecCheckURL: defaultWechatMsgSecCheckURL,
		mediaCheckURL:  defaultWechatMediaCheckURL,
	}
}

func (a *WechatAuditor) WithURLs(accessTokenURL string, msgSecCheckURL string, mediaCheckURL string) *WechatAuditor {
	if strings.TrimSpace(accessTokenURL) != "" {
		a.accessTokenURL = accessTokenURL
	}
	if strings.TrimSpace(msgSecCheckURL) != "" {
		a.msgSecCheckURL = msgSecCheckURL
	}
	if strings.TrimSpace(mediaCheckURL) != "" {
		a.mediaCheckURL = mediaCheckURL
	}
	return a
}

func (a *WechatAuditor) AuditResource(ctx context.Context, input resourcelogic.ContentAuditInput) (resourcelogic.ContentAuditResult, error) {
	if a == nil || !a.cfg.Enabled {
		return resourcelogic.ContentAuditResult{Decision: resourcelogic.ContentAuditDecisionPass}, nil
	}
	if resourcelogic.IsLegacyWechatOpenID(input.OpenID) {
		// 历史开发身份不属于真实小程序，若继续调用微信会反复触发 invalid openid 并污染重试队列。
		logx.Errorf("微信内容安全拒绝历史开发身份: merchantId=%s resourceId=%s typeCode=%s", input.MerchantID, input.ResourceID, input.TypeCode)
		return resourcelogic.ContentAuditResult{}, resourcelogic.ErrLegacyWechatAuditIdentity
	}
	if strings.TrimSpace(input.OpenID) == "" {
		return resourcelogic.ContentAuditResult{}, fmt.Errorf("微信内容安全需要用户 openid")
	}
	if strings.TrimSpace(a.wechat.AppID) == "" || strings.TrimSpace(a.wechat.AppSecret) == "" {
		logx.Errorf("微信内容安全未配置 appid/appsecret: merchantId=%s resourceId=%s typeCode=%s", input.MerchantID, input.ResourceID, input.TypeCode)
		return resourcelogic.ContentAuditResult{}, fmt.Errorf("微信内容安全未配置 appid/appsecret")
	}

	token, err := a.getAccessToken(ctx)
	if err != nil {
		logx.Errorf("获取微信内容安全 access_token 失败: merchantId=%s resourceId=%s typeCode=%s err=%+v", input.MerchantID, input.ResourceID, input.TypeCode, err)
		return resourcelogic.ContentAuditResult{}, err
	}

	result := resourcelogic.ContentAuditResult{Decision: resourcelogic.ContentAuditDecisionPass}
	text := resourcelogic.BuildResourceAuditText(input, a.cfg.MaxTextChars)
	if strings.TrimSpace(text) != "" {
		textResult, err := a.checkText(ctx, token, input, text)
		if err != nil {
			logx.Errorf("微信文本内容安全检测失败: merchantId=%s resourceId=%s typeCode=%s err=%+v", input.MerchantID, input.ResourceID, input.TypeCode, err)
			return resourcelogic.ContentAuditResult{}, err
		}
		result.Decision = stricterDecision(result.Decision, textResult.Decision)
		result.Reason = textResult.Reason
		result.Labels = append(result.Labels, textResult.Labels...)
		result.TraceIDs = append(result.TraceIDs, textResult.TraceIDs...)
		if normalizeDecision(textResult.Decision) == resourcelogic.ContentAuditDecisionRisky {
			return result, nil
		}
	}

	if a.cfg.MediaEnabled {
		mediaTasks, err := a.submitImages(ctx, token, input)
		if err != nil {
			return resourcelogic.ContentAuditResult{}, err
		}
		for _, task := range mediaTasks {
			result.TraceIDs = append(result.TraceIDs, task.TraceID)
		}
		result.MediaTasks = append(result.MediaTasks, mediaTasks...)
	}
	return result, nil
}

func (a *WechatAuditor) checkText(ctx context.Context, token string, input resourcelogic.ContentAuditInput, content string) (resourcelogic.ContentAuditResult, error) {
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(map[string]any{
		"openid":  input.OpenID,
		"scene":   a.cfg.TextScene,
		"version": 2,
		"title":   input.Title,
		"content": content,
	}); err != nil {
		return resourcelogic.ContentAuditResult{}, fmt.Errorf("构造文本审核请求失败: %w", err)
	}
	endpoint, err := withAccessToken(a.msgSecCheckURL, token)
	if err != nil {
		return resourcelogic.ContentAuditResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &payload)
	if err != nil {
		return resourcelogic.ContentAuditResult{}, fmt.Errorf("创建文本审核请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var data msgSecCheckResp
	if err := a.doJSON(req, &data); err != nil {
		return resourcelogic.ContentAuditResult{}, err
	}
	if data.ErrCode != 0 {
		return resourcelogic.ContentAuditResult{}, fmt.Errorf("微信文本审核接口返回错误: errcode=%d errmsg=%s", data.ErrCode, data.ErrMsg)
	}
	decision := data.Result.Suggest
	if decision == "" {
		decision = resourcelogic.ContentAuditDecisionReview
	}
	labels := contentAuditLabels(data.Result.Label, data.Detail)
	return resourcelogic.ContentAuditResult{
		Decision: decision,
		Reason:   wechatAuditReason(decision, labels, "内容疑似存在安全风险，请修改后重新提交"),
		Labels:   labels,
		TraceIDs: trimNonEmpty([]string{data.TraceID}),
	}, nil
}

func (a *WechatAuditor) submitImages(ctx context.Context, token string, input resourcelogic.ContentAuditInput) ([]resourcelogic.ContentAuditMediaTask, error) {
	tasks := make([]resourcelogic.ContentAuditMediaTask, 0, len(input.Images))
	for _, imageURL := range input.Images {
		normalizedURL := a.normalizeMediaURL(imageURL)
		if normalizedURL == "" {
			return nil, fmt.Errorf("图片地址无法用于微信内容安全检测")
		}
		traceID, err := a.submitMedia(ctx, token, input, normalizedURL)
		if err != nil {
			logx.Errorf("微信图片内容安全任务提交失败: merchantId=%s resourceId=%s typeCode=%s mediaURLPresent=%t err=%+v", input.MerchantID, input.ResourceID, input.TypeCode, normalizedURL != "", err)
			return nil, err
		}
		if strings.TrimSpace(traceID) == "" {
			return nil, fmt.Errorf("微信图片审核接口未返回 trace_id")
		}
		tasks = append(tasks, resourcelogic.ContentAuditMediaTask{TraceID: traceID, MediaURL: normalizedURL})
	}
	return tasks, nil
}

func (a *WechatAuditor) submitMedia(ctx context.Context, token string, input resourcelogic.ContentAuditInput, mediaURL string) (string, error) {
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(map[string]any{
		"openid":     input.OpenID,
		"scene":      a.cfg.MediaScene,
		"version":    2,
		"media_type": 2,
		"media_url":  mediaURL,
	}); err != nil {
		return "", fmt.Errorf("构造图片审核请求失败: %w", err)
	}
	endpoint, err := withAccessToken(a.mediaCheckURL, token)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &payload)
	if err != nil {
		return "", fmt.Errorf("创建图片审核请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var data mediaCheckResp
	if err := a.doJSON(req, &data); err != nil {
		return "", err
	}
	if data.ErrCode != 0 {
		return "", fmt.Errorf("微信图片审核接口返回错误: errcode=%d errmsg=%s", data.ErrCode, data.ErrMsg)
	}
	logx.Infof("微信图片内容安全任务已提交: merchantId=%s resourceId=%s typeCode=%s traceId=%s", input.MerchantID, input.ResourceID, input.TypeCode, data.TraceID)
	return strings.TrimSpace(data.TraceID), nil
}

func (a *WechatAuditor) getAccessToken(ctx context.Context) (string, error) {
	now := time.Now()
	a.mu.Lock()
	if a.accessToken != "" && now.Before(a.tokenExpiry) {
		token := a.accessToken
		a.mu.Unlock()
		return token, nil
	}
	a.mu.Unlock()

	endpoint, err := url.Parse(a.accessTokenURL)
	if err != nil {
		return "", fmt.Errorf("微信 access_token 地址配置错误: %w", err)
	}
	query := endpoint.Query()
	query.Set("appid", a.wechat.AppID)
	query.Set("secret", a.wechat.AppSecret)
	query.Set("grant_type", "client_credential")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("创建微信 access_token 请求失败: %w", err)
	}
	var data accessTokenResp
	if err := a.doJSON(req, &data); err != nil {
		return "", err
	}
	if data.ErrCode != 0 {
		return "", fmt.Errorf("微信 access_token 接口返回错误: errcode=%d errmsg=%s", data.ErrCode, data.ErrMsg)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return "", fmt.Errorf("微信 access_token 接口未返回 token")
	}
	expiresIn := data.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	refreshAfter := expiresIn - 300
	if refreshAfter <= 0 {
		refreshAfter = 60
	}
	expiry := now.Add(time.Duration(refreshAfter) * time.Second)
	a.mu.Lock()
	a.accessToken = strings.TrimSpace(data.AccessToken)
	a.tokenExpiry = expiry
	a.mu.Unlock()
	return strings.TrimSpace(data.AccessToken), nil
}

func (a *WechatAuditor) doJSON(req *http.Request, out any) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return fmt.Errorf("读取微信内容安全响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("微信内容安全 HTTP 状态异常: status=%d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析微信内容安全响应失败: %w", err)
	}
	return nil
}

func (a *WechatAuditor) normalizeMediaURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if a.mediaBaseURL == "" {
		return ""
	}
	return a.mediaBaseURL + "/" + strings.TrimLeft(value, "/")
}

func withAccessToken(rawURL string, token string) (string, error) {
	endpoint, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("微信内容安全地址配置错误: %w", err)
	}
	query := endpoint.Query()
	query.Set("access_token", token)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func stricterDecision(left string, right string) string {
	left = normalizeDecision(left)
	right = normalizeDecision(right)
	if left == resourcelogic.ContentAuditDecisionRisky || right == resourcelogic.ContentAuditDecisionRisky {
		return resourcelogic.ContentAuditDecisionRisky
	}
	if left == resourcelogic.ContentAuditDecisionReview || right == resourcelogic.ContentAuditDecisionReview {
		return resourcelogic.ContentAuditDecisionReview
	}
	return resourcelogic.ContentAuditDecisionPass
}

func normalizeDecision(decision string) string {
	switch strings.TrimSpace(strings.ToLower(decision)) {
	case resourcelogic.ContentAuditDecisionRisky:
		return resourcelogic.ContentAuditDecisionRisky
	case resourcelogic.ContentAuditDecisionReview:
		return resourcelogic.ContentAuditDecisionReview
	default:
		return resourcelogic.ContentAuditDecisionPass
	}
}

func contentAuditLabels(resultLabel int64, detail []auditDetail) []string {
	labels := make([]string, 0, len(detail)+1)
	if resultLabel != 0 {
		labels = append(labels, fmt.Sprintf("%d", resultLabel))
	}
	for _, item := range detail {
		if item.Label != 0 {
			labels = append(labels, fmt.Sprintf("%d", item.Label))
		}
	}
	return trimNonEmpty(labels)
}

func wechatAuditReason(decision string, labels []string, fallback string) string {
	if normalizeDecision(decision) != resourcelogic.ContentAuditDecisionRisky {
		return ""
	}
	return resourcelogic.ResourceAuditRejectReason(resourcelogic.ContentAuditResult{Labels: labels}, fallback)
}

func trimNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

type accessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int64  `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type msgSecCheckResp struct {
	ErrCode int64         `json:"errcode"`
	ErrMsg  string        `json:"errmsg"`
	TraceID string        `json:"trace_id"`
	Result  auditResult   `json:"result"`
	Detail  []auditDetail `json:"detail"`
}

type mediaCheckResp struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	TraceID string `json:"trace_id"`
}

type auditResult struct {
	Suggest string `json:"suggest"`
	Label   int64  `json:"label"`
}

type auditDetail struct {
	Suggest string `json:"suggest"`
	Label   int64  `json:"label"`
}
