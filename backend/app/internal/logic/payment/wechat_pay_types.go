package payment

import "context"

type WechatPayGateway interface {
	CreatePrepay(ctx context.Context, input WechatPrepayInput) (WechatPayParams, error)
	DecodeNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotification, error)
}

type WechatPrepayInput struct {
	OutTradeNo  string
	Description string
	OpenID      string
	AmountTotal int64
	Currency    string
	Attach      string
}

type WechatPayParams struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type WechatPayNotifyReq struct {
	Headers map[string]string
	Body    []byte
}

type WechatPayNotification struct {
	OutTradeNo    string
	TransactionID string
	Attach        string
	AmountTotal   int64
	SuccessTime   string
	RawPayload    map[string]interface{}
}

type WechatPayNotifyResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
