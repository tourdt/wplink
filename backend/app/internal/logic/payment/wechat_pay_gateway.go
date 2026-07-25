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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const wechatPayAPIBaseURL = "https://api.mch.weixin.qq.com"

type HTTPWechatPayGateway struct {
	cfg        config.WechatPayConfig
	httpClient *http.Client
	apiBaseURL string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
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

func NewHTTPWechatPayGateway(cfg config.WechatPayConfig) (*HTTPWechatPayGateway, error) {
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
	return &HTTPWechatPayGateway{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
		apiBaseURL: wechatPayAPIBaseURL,
		privateKey: privateKey,
		publicKey:  publicKey,
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
		return WechatPayParams{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodPost, "/v3/pay/transactions/jsapi", string(bodyBytes))
	if err != nil {
		return WechatPayParams{}, err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		logx.Errorf("调用微信支付下单接口失败: outTradeNo=%s err=%+v", input.OutTradeNo, err)
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logx.Errorf("微信支付下单接口返回失败: outTradeNo=%s status=%d err=%+v", input.OutTradeNo, resp.StatusCode, decodeWechatPayAPIError(resp.StatusCode, respBody))
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
		logx.Errorf("微信支付下单响应验签失败: outTradeNo=%s err=%+v", input.OutTradeNo, err)
		return WechatPayParams{}, err
	}
	var decoded struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付响应解析失败，请稍后重试")
	}
	if strings.TrimSpace(decoded.PrepayID) == "" {
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付预支付单无效，请稍后重试")
	}
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
		return WechatPayOrder{}, err
	}
	req.Header.Set("Accept", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodGet, canonicalURL, "")
	if err != nil {
		return WechatPayOrder{}, err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return WechatPayOrder{}, fmt.Errorf("查询微信支付订单失败: outTradeNo=%s: %w", outTradeNo, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WechatPayOrder{}, decodeWechatPayAPIError(resp.StatusCode, respBody)
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
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
		return WechatPayOrder{}, fmt.Errorf("解析微信支付查单响应失败: outTradeNo=%s: %w", outTradeNo, err)
	}
	if strings.TrimSpace(order.OutTradeNo) != outTradeNo || strings.TrimSpace(order.TradeState) == "" {
		return WechatPayOrder{}, fmt.Errorf("微信支付查单响应数据不完整: outTradeNo=%s", outTradeNo)
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(respBody, &raw)
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
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	authHeader, err := g.authorizationHeader(http.MethodPost, canonicalURL, string(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("关闭微信支付订单失败: outTradeNo=%s: %w", outTradeNo, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeWechatPayAPIError(resp.StatusCode, respBody)
	}
	if err := g.verifyResponseSignature(resp.Header, respBody); err != nil {
		return err
	}
	return nil
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
