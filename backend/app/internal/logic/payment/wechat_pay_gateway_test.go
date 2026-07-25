package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"wplink/backend/app/internal/config"
)

func TestHTTPWechatPayGatewayQueriesAndClosesOrder(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	var queried bool
	var closed bool
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v3/pay/transactions/out-trade-no/"):
			queried = true
			if r.URL.Query().Get("mchid") != "merchant-1" {
				t.Errorf("mchid = %q, want merchant-1", r.URL.Query().Get("mchid"))
			}
			return signedWechatHTTPResponse(t, privateKey, http.StatusOK, `{"out_trade_no":"VIP202607250001","transaction_id":"wx-1","attach":"vip:order-1","trade_state":"SUCCESS","success_time":"2026-07-25T12:00:00Z","amount":{"total":1990}}`), nil
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/close"):
			closed = true
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				return nil, readErr
			}
			var decoded map[string]string
			if err := json.Unmarshal(body, &decoded); err != nil || decoded["mchid"] != "merchant-1" {
				t.Errorf("close body = %s err=%v, want merchant id", body, err)
			}
			return signedWechatHTTPResponse(t, privateKey, http.StatusNoContent, ""), nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"code":"NOT_FOUND"}`)),
			}, nil
		}
	})}

	gateway := &HTTPWechatPayGateway{
		cfg:        config.WechatPayConfig{MchID: "merchant-1", MerchantSerialNo: "serial-1"},
		httpClient: client,
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}

	order, err := gateway.QueryOrder(context.Background(), "VIP202607250001")
	if err != nil {
		t.Fatalf("QueryOrder() error = %v", err)
	}
	if order.TradeState != "SUCCESS" || order.Attach != "vip:order-1" || order.AmountTotal != 1990 {
		t.Fatalf("order = %#v, want paid order", order)
	}
	if err := gateway.CloseOrder(context.Background(), "VIP202607250001"); err != nil {
		t.Fatalf("CloseOrder() error = %v", err)
	}
	if !queried || !closed {
		t.Fatalf("queried=%t closed=%t, want both requests", queried, closed)
	}
}

func TestHTTPWechatPayGatewayRejectsUnsignedQueryResponse(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"out_trade_no":"VIP202607250001","trade_state":"NOTPAY","amount":{"total":1990}}`)),
		}, nil
	})}

	gateway := &HTTPWechatPayGateway{
		cfg:        config.WechatPayConfig{MchID: "merchant-1", MerchantSerialNo: "serial-1"},
		httpClient: client,
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}
	if _, err := gateway.QueryOrder(context.Background(), "VIP202607250001"); err == nil {
		t.Fatal("QueryOrder() error = nil, want response signature validation failure")
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func signedWechatHTTPResponse(t *testing.T, privateKey *rsa.PrivateKey, status int, body string) *http.Response {
	t.Helper()
	timestamp := "1784971200"
	nonce := "response-nonce"
	message := timestamp + "\n" + nonce + "\n" + body + "\n"
	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}
	headers := make(http.Header)
	headers.Set("Wechatpay-Timestamp", timestamp)
	headers.Set("Wechatpay-Nonce", nonce)
	headers.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(signature))
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
