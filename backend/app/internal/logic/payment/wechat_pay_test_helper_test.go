package payment

import "context"

type fakeWechatPayGateway struct {
	prepayInput WechatPrepayInput
	prepay      WechatPayParams
	notify      WechatPayNotification
}

func (g *fakeWechatPayGateway) CreatePrepay(ctx context.Context, input WechatPrepayInput) (WechatPayParams, error) {
	g.prepayInput = input
	return g.prepay, nil
}

func (g *fakeWechatPayGateway) DecodeNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotification, error) {
	return g.notify, nil
}
