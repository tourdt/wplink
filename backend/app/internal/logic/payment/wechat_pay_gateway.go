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
	"os"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

const wechatPayAPIBaseURL = "https://api.mch.weixin.qq.com"

type HTTPWechatPayGateway struct {
	cfg        config.WechatPayConfig
	httpClient *http.Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
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
	body := map[string]interface{}{
		"appid":        g.cfg.AppID,
		"mchid":        g.cfg.MchID,
		"description":  input.Description,
		"out_trade_no": input.OutTradeNo,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wechatPayAPIBaseURL+"/v3/pay/transactions/jsapi", bytes.NewReader(bodyBytes))
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
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WechatPayParams{}, errx.New(errx.CodeInternalError, "微信支付下单失败，请稍后重试")
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
