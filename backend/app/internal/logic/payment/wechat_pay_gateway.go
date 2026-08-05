package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx"
)

const wechatPayAPIBaseURL = "https://api.mch.weixin.qq.com"

type HTTPWechatPayGateway struct {
	cfg        config.WechatPayConfig
	httpClient *http.Client
	apiBaseURL string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	observer   externalcall.Observer
}

type WechatPayOrderGateway interface {
	QueryOrder(ctx context.Context, outTradeNo string) (WechatPayOrder, error)
	CloseOrder(ctx context.Context, outTradeNo string) error
}

type WechatPayOrder struct {
	OutTradeNo    string
	TransactionID string
	Attach        string
	TradeState    string
	SuccessTime   string
	AmountTotal   int64
	RawPayload    map[string]interface{}
}

type WechatPayAPIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *WechatPayAPIError) Error() string {
	return fmt.Sprintf("微信支付 API 请求失败: status=%d code=%s message=%s", e.HTTPStatus, e.Code, e.Message)
}

func NewHTTPWechatPayGateway(cfg config.WechatPayConfig, observers ...externalcall.Observer) (*HTTPWechatPayGateway, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	privateKey, err := loadRSAPrivateKey(cfg.MerchantPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载微信支付商户私钥失败: %w", err)
	}
	publicKey, err := loadRSAPublicKey(cfg.PlatformPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载微信支付平台公钥失败: %w", err)
	}
	// variadic 仅用于保持旧调用兼容：这是可选单 Observer，传入多个时只使用第一个。
	var observer externalcall.Observer
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &HTTPWechatPayGateway{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
		apiBaseURL: wechatPayAPIBaseURL,
		privateKey: privateKey,
		publicKey:  publicKey,
		observer:   observer,
	}, nil
}

func (g *HTTPWechatPayGateway) CreatePrepay(ctx context.Context, input WechatPrepayInput) (WechatPayParams, error) {
	if g == nil {
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付暂未配置，请联系平台运营")
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	orderExpire := g.cfg.OrderExpire
	if orderExpire <= 0 {
		orderExpire = 30 * time.Minute
	}
	body := map[string]interface{}{
		"appid":        g.cfg.AppID,
		"mchid":        g.cfg.MchID,
		"description":  input.Description,
		"out_trade_no": input.OutTradeNo,
		"time_expire":  time.Now().Add(orderExpire).Format(time.RFC3339),
		"notify_url":   g.cfg.NotifyURL,
		"amount": map[string]interface{}{
			"total":    input.AmountTotal,
			"currency": currency,
		},
		"payer": map[string]interface{}{
			"openid": input.OpenID,
		},
	}
	if strings.TrimSpace(input.Attach) != "" {
		body["attach"] = strings.TrimSpace(input.Attach)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return WechatPayParams{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL()+"/v3/pay/transactions/jsapi", bytes.NewReader(bodyBytes))
	if err != nil {
		// 请求创建错误可能包含配置 URL；向上只返回稳定、可展示的中文错误。
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodPost, "/v3/pay/transactions/jsapi", string(bodyBytes))
	if err != nil {
		return WechatPayParams{}, err
	}
	req.Header.Set("Authorization", authHeader)

	// 本地参数编码、请求构造与签名完成后才算真正开始外呼。
	call := externalcall.Start(g.observer, externalcall.ProviderWechat, externalcall.OperationPayCreate)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		// http.Client 的错误可能携带完整 URL、请求参数和底层原文，只记录安全枚举。
		logx.Errorf("调用微信支付下单接口失败: outTradeNo=%s outcome=%s", input.OutTradeNo, outcome)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, resp.StatusCode)
		logx.Errorf("读取微信支付下单响应失败: outTradeNo=%s outcome=%s status=%d", input.OutTradeNo, externalcall.OutcomeTransportError, resp.StatusCode)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		call.Finish(ctx, externalcall.OutcomeHTTPError, resp.StatusCode)
		// 不记录响应体或供应商 message，避免上游错误内容把支付数据带入日志。
		logx.Errorf("微信支付下单接口返回失败: outTradeNo=%s outcome=%s status=%d", input.OutTradeNo, externalcall.OutcomeHTTPError, resp.StatusCode)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		logx.Errorf("微信支付下单响应验签失败: outTradeNo=%s outcome=%s status=%d", input.OutTradeNo, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayParams{}, err
	}
	var decoded struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付响应解析失败，请稍后重试")
	}
	if strings.TrimSpace(decoded.PrepayID) == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付预支付单无效，请稍后重试")
	}
	// 微信响应已经完整通过验签与字段校验；后续小程序支付签名属于本地计算。
	call.Finish(ctx, externalcall.OutcomeSuccess, resp.StatusCode)
	pkg := "prepay_id=" + decoded.PrepayID
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := randomNonce()
	paySign, err := g.sign(g.cfg.AppID + "\n" + timestamp + "\n" + nonce + "\n" + pkg + "\n")
	if err != nil {
		return WechatPayParams{}, err
	}
	return WechatPayParams{
		TimeStamp: timestamp,
		NonceStr:  nonce,
		Package:   pkg,
		SignType:  "RSA",
		PaySign:   paySign,
	}, nil
}

func (g *HTTPWechatPayGateway) QueryOrder(ctx context.Context, outTradeNo string) (WechatPayOrder, error) {
	if g == nil {
		return WechatPayOrder{}, errx.New(errx.CodeInternalError, "微信支付暂未配置")
	}
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return WechatPayOrder{}, errx.New(errx.CodeValidationFailed, "商户支付单号不能为空")
	}
	canonicalURL := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(outTradeNo) + "?mchid=" + url.QueryEscape(g.cfg.MchID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL()+canonicalURL, nil)
	if err != nil {
		return WechatPayOrder{}, errx.New(errx.CodeInternalError, "查询微信支付订单失败，请稍后重试")
	}
	req.Header.Set("Accept", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodGet, canonicalURL, "")
	if err != nil {
		return WechatPayOrder{}, err
	}
	req.Header.Set("Authorization", authHeader)

	call := externalcall.Start(g.observer, externalcall.ProviderWechat, externalcall.OperationPayQuery)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		logx.Errorf("查询微信支付订单失败: outTradeNo=%s outcome=%s", outTradeNo, outcome)
		return WechatPayOrder{}, newWechatPaySafeCallError("查询微信支付订单失败", outcome, ctx, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, resp.StatusCode)
		logx.Errorf("读取微信支付查单响应失败: outTradeNo=%s outcome=%s status=%d", outTradeNo, externalcall.OutcomeTransportError, resp.StatusCode)
		return WechatPayOrder{}, newWechatPaySafeCallError("读取微信支付查单响应失败", externalcall.OutcomeTransportError, ctx, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		call.Finish(ctx, externalcall.OutcomeHTTPError, resp.StatusCode)
		apiErr := decodeWechatPayAPIError(resp.StatusCode, respBody)
		logx.Errorf("微信支付查单 HTTP 状态异常: outTradeNo=%s outcome=%s status=%d", outTradeNo, externalcall.OutcomeHTTPError, resp.StatusCode)
		return WechatPayOrder{}, apiErr
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayOrder{}, err
	}

	var order struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		Attach        string `json:"attach"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(respBody, &order); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayOrder{}, errx.New(errx.CodeInternalError, "微信支付查单响应解析失败，请稍后重试")
	}
	if strings.TrimSpace(order.OutTradeNo) != outTradeNo || strings.TrimSpace(order.TradeState) == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return WechatPayOrder{}, errx.New(errx.CodeInternalError, "微信支付查单响应数据不完整，请稍后重试")
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(respBody, &raw)
	call.Finish(ctx, externalcall.OutcomeSuccess, resp.StatusCode)
	return WechatPayOrder{
		OutTradeNo:    order.OutTradeNo,
		TransactionID: order.TransactionID,
		Attach:        order.Attach,
		TradeState:    order.TradeState,
		SuccessTime:   order.SuccessTime,
		AmountTotal:   order.Amount.Total,
		RawPayload:    raw,
	}, nil
}

func (g *HTTPWechatPayGateway) CloseOrder(ctx context.Context, outTradeNo string) error {
	if g == nil {
		return errx.New(errx.CodeInternalError, "微信支付暂未配置")
	}
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return errx.New(errx.CodeValidationFailed, "商户支付单号不能为空")
	}
	bodyBytes, err := json.Marshal(map[string]string{"mchid": g.cfg.MchID})
	if err != nil {
		return err
	}
	canonicalURL := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(outTradeNo) + "/close"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL()+canonicalURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return errx.New(errx.CodeInternalError, "关闭微信支付订单失败，请稍后重试")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodPost, canonicalURL, string(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authHeader)

	call := externalcall.Start(g.observer, externalcall.ProviderWechat, externalcall.OperationPayClose)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		logx.Errorf("关闭微信支付订单失败: outTradeNo=%s outcome=%s", outTradeNo, outcome)
		return newWechatPaySafeCallError("关闭微信支付订单失败", outcome, ctx, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, resp.StatusCode)
		logx.Errorf("读取微信支付关单响应失败: outTradeNo=%s outcome=%s status=%d", outTradeNo, externalcall.OutcomeTransportError, resp.StatusCode)
		return newWechatPaySafeCallError("读取微信支付关单响应失败", externalcall.OutcomeTransportError, ctx, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		call.Finish(ctx, externalcall.OutcomeHTTPError, resp.StatusCode)
		apiErr := decodeWechatPayAPIError(resp.StatusCode, respBody)
		logx.Errorf("微信支付关单 HTTP 状态异常: outTradeNo=%s outcome=%s status=%d", outTradeNo, externalcall.OutcomeHTTPError, resp.StatusCode)
		return apiErr
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, resp.StatusCode)
		return err
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, resp.StatusCode)
	return nil
}

// wechatPaySafeCallError 是支付传输错误的向上传播边界。它只暴露稳定终态，
// 并最多保留 context 的安全 sentinel，避免上层日志通过 err.Error() 泄漏 URL/query 或底层错误原文。
type wechatPaySafeCallError struct {
	message  string
	outcome  externalcall.Outcome
	sentinel error
}

func newWechatPaySafeCallError(message string, outcome externalcall.Outcome, ctx context.Context, err error) error {
	var sentinel error
	switch {
	case errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)):
		sentinel = context.Canceled
	case errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)):
		sentinel = context.DeadlineExceeded
	}
	return &wechatPaySafeCallError{message: message, outcome: outcome, sentinel: sentinel}
}

func (e *wechatPaySafeCallError) Error() string {
	if e == nil {
		return "微信支付调用失败"
	}
	if e.outcome == "" {
		return e.message
	}
	return fmt.Sprintf("%s: outcome=%s", e.message, e.outcome)
}

func (e *wechatPaySafeCallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.sentinel
}

func (g *HTTPWechatPayGateway) DecodeNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotification, error) {
	if g == nil {
		return WechatPayNotification{}, errx.New(errx.CodeInternalError, "微信支付暂未配置")
	}
	if err := g.verifyNotifySignature(req.Headers, req.Body); err != nil {
		return WechatPayNotification{}, err
	}
	var body struct {
		Resource struct {
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
			Ciphertext     string `json:"ciphertext"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return WechatPayNotification{}, errx.New(errx.CodeValidationFailed, "支付通知格式不正确")
	}
	plaintext, err := decryptWechatPayResource(g.cfg.APIv3Key, body.Resource.AssociatedData, body.Resource.Nonce, body.Resource.Ciphertext)
	if err != nil {
		return WechatPayNotification{}, errx.New(errx.CodeValidationFailed, "支付通知解密失败")
	}
	var transaction struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		Attach        string `json:"attach"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plaintext, &transaction); err != nil {
		return WechatPayNotification{}, errx.New(errx.CodeValidationFailed, "支付通知内容不正确")
	}
	if transaction.TradeState != "SUCCESS" {
		return WechatPayNotification{}, errx.New(errx.CodeStateConflict, "支付状态未成功")
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(plaintext, &raw)
	return WechatPayNotification{
		OutTradeNo:    transaction.OutTradeNo,
		TransactionID: transaction.TransactionID,
		Attach:        transaction.Attach,
		AmountTotal:   transaction.Amount.Total,
		SuccessTime:   transaction.SuccessTime,
		RawPayload:    raw,
	}, nil
}

func (g *HTTPWechatPayGateway) authorizationHeader(method string, canonicalURL string, body string) (string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := randomNonce()
	message := strings.ToUpper(method) + "\n" + canonicalURL + "\n" + timestamp + "\n" + nonce + "\n" + body + "\n"
	signature, err := g.sign(message)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		g.cfg.MchID,
		nonce,
		signature,
		timestamp,
		g.cfg.MerchantSerialNo,
	), nil
}

func (g *HTTPWechatPayGateway) sign(message string) (string, error) {
	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, g.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", errx.New(errx.CodeInternalError, "微信支付签名失败，请稍后重试")
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (g *HTTPWechatPayGateway) verifyNotifySignature(headers map[string]string, body []byte) error {
	timestamp := headerValue(headers, "Wechatpay-Timestamp")
	nonce := headerValue(headers, "Wechatpay-Nonce")
	signatureText := headerValue(headers, "Wechatpay-Signature")
	if timestamp == "" || nonce == "" || signatureText == "" {
		return errx.New(errx.CodeValidationFailed, "支付通知签名头缺失")
	}
	signature, err := base64.StdEncoding.DecodeString(signatureText)
	if err != nil {
		return errx.New(errx.CodeValidationFailed, "支付通知签名不正确")
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	hashed := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(g.publicKey, crypto.SHA256, hashed[:], signature); err != nil {
		return errx.New(errx.CodeValidationFailed, "支付通知验签失败")
	}
	return nil
}

func (g *HTTPWechatPayGateway) verifyResponseSignature(headers http.Header, body []byte) error {
	timestamp := strings.TrimSpace(headers.Get("Wechatpay-Timestamp"))
	nonce := strings.TrimSpace(headers.Get("Wechatpay-Nonce"))
	signatureText := strings.TrimSpace(headers.Get("Wechatpay-Signature"))
	if timestamp == "" || nonce == "" || signatureText == "" {
		return errx.New(errx.CodeInternalError, "微信支付响应验签信息缺失，请稍后重试")
	}
	signature, err := base64.StdEncoding.DecodeString(signatureText)
	if err != nil {
		return errx.New(errx.CodeInternalError, "微信支付响应签名不正确，请稍后重试")
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	hashed := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(g.publicKey, crypto.SHA256, hashed[:], signature); err != nil {
		return errx.New(errx.CodeInternalError, "微信支付响应验签失败，请稍后重试")
	}
	return nil
}

func (g *HTTPWechatPayGateway) baseURL() string {
	if strings.TrimSpace(g.apiBaseURL) != "" {
		return strings.TrimRight(g.apiBaseURL, "/")
	}
	return wechatPayAPIBaseURL
}

func decodeWechatPayAPIError(status int, body []byte) error {
	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &decoded)
	return &WechatPayAPIError{
		HTTPStatus: status,
		Code:       strings.TrimSpace(decoded.Code),
		Message:    strings.TrimSpace(decoded.Message),
	}
}

func decryptWechatPayResource(apiV3Key string, associatedData string, nonce string, ciphertext string) ([]byte, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, []byte(nonce), cipherBytes, []byte(associatedData))
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("PEM 内容为空")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("不是 RSA 私钥")
	}
	return key, nil
}

func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("PEM 内容为空")
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		key, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("证书不是 RSA 公钥")
		}
		return key, nil
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是 RSA 公钥")
	}
	return key, nil
}

func randomNonce() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func headerValue(headers map[string]string, key string) string {
	for headerKey, value := range headers {
		if strings.EqualFold(headerKey, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
