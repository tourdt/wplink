package task

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type PaymentReconciliationStore interface {
	paymentlogic.UnifiedWechatPayNotifyStore
	ListPendingPaymentOrders(ctx context.Context, createdBefore time.Time, limit int64) ([]model.PendingPaymentOrder, error)
	MarkPaymentOrderClosed(ctx context.Context, businessType string, businessOrderID string, outTradeNo string) error
}

type PaymentReconciliationResult struct {
	ScannedCount int64
	PaidCount    int64
	ClosedCount  int64
	PendingCount int64
	FailedCount  int64
}

type PaymentReconciliationTask struct {
	store          PaymentReconciliationStore
	gateway        paymentlogic.WechatPayOrderGateway
	queryDelay     time.Duration
	pendingTimeout time.Duration
	batchSize      int64
	now            func() time.Time
}

func NewPaymentReconciliationTask(
	store PaymentReconciliationStore,
	gateway paymentlogic.WechatPayOrderGateway,
	queryDelay time.Duration,
	pendingTimeout time.Duration,
	batchSize int64,
) *PaymentReconciliationTask {
	if queryDelay <= 0 {
		queryDelay = 2 * time.Minute
	}
	if pendingTimeout <= 0 {
		pendingTimeout = 30 * time.Minute
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &PaymentReconciliationTask{
		store:          store,
		gateway:        gateway,
		queryDelay:     queryDelay,
		pendingTimeout: pendingTimeout,
		batchSize:      batchSize,
		now:            time.Now,
	}
}

func (t *PaymentReconciliationTask) Run(ctx context.Context) (PaymentReconciliationResult, error) {
	if t == nil || t.store == nil || t.gateway == nil {
		return PaymentReconciliationResult{}, nil
	}
	now := t.now()
	orders, err := t.store.ListPendingPaymentOrders(ctx, now.Add(-t.queryDelay), t.batchSize)
	if err != nil {
		return PaymentReconciliationResult{}, err
	}

	result := PaymentReconciliationResult{ScannedCount: int64(len(orders))}
	notifyLogic := paymentlogic.NewUnifiedWechatPayNotifyLogic(t.store, nil)
	for _, order := range orders {
		remoteOrder, queryErr := t.gateway.QueryOrder(ctx, order.OutTradeNo)
		if queryErr != nil {
			if t.shouldClose(order, now) && isWechatOrderNotExist(queryErr) {
				if closeErr := t.markLocalOrderClosed(ctx, order); closeErr == nil {
					result.ClosedCount++
					continue
				}
			}
			result.FailedCount++
			logx.Errorf("微信支付补偿查单失败: businessType=%s businessOrderId=%s outTradeNo=%s err=%+v", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, queryErr)
			continue
		}

		switch strings.ToUpper(strings.TrimSpace(remoteOrder.TradeState)) {
		case "SUCCESS":
			_, handleErr := notifyLogic.HandleNotification(ctx, paymentlogic.WechatPayNotification{
				OutTradeNo:    remoteOrder.OutTradeNo,
				TransactionID: remoteOrder.TransactionID,
				Attach:        remoteOrder.Attach,
				AmountTotal:   remoteOrder.AmountTotal,
				SuccessTime:   remoteOrder.SuccessTime,
				RawPayload:    remoteOrder.RawPayload,
			})
			if handleErr != nil {
				result.FailedCount++
				logx.Errorf("微信支付补偿入账失败: businessType=%s businessOrderId=%s outTradeNo=%s err=%+v", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, handleErr)
				continue
			}
			result.PaidCount++
		case "CLOSED", "REVOKED", "PAYERROR":
			if closeErr := t.markLocalOrderClosed(ctx, order); closeErr != nil {
				result.FailedCount++
				logx.Errorf("同步微信已关闭支付单失败: businessType=%s businessOrderId=%s outTradeNo=%s tradeState=%s err=%+v", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, remoteOrder.TradeState, closeErr)
				continue
			}
			result.ClosedCount++
		case "NOTPAY", "USERPAYING":
			if !t.shouldClose(order, now) {
				result.PendingCount++
				continue
			}
			// 微信官方要求超时未支付订单先主动关单，成功后才能关闭本地订单，避免出现本地关闭但用户仍可支付。
			if closeErr := t.gateway.CloseOrder(ctx, order.OutTradeNo); closeErr != nil {
				result.FailedCount++
				logx.Errorf("关闭超时微信支付单失败: businessType=%s businessOrderId=%s outTradeNo=%s err=%+v", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, closeErr)
				continue
			}
			if closeErr := t.markLocalOrderClosed(ctx, order); closeErr != nil {
				result.FailedCount++
				logx.Errorf("关闭超时本地支付单失败: businessType=%s businessOrderId=%s outTradeNo=%s err=%+v", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, closeErr)
				continue
			}
			result.ClosedCount++
		default:
			result.FailedCount++
			logx.Errorf("微信支付补偿遇到未支持的交易状态: businessType=%s businessOrderId=%s outTradeNo=%s tradeState=%s", order.BusinessType, order.BusinessOrderID, order.OutTradeNo, remoteOrder.TradeState)
		}
	}
	return result, nil
}

func (t *PaymentReconciliationTask) shouldClose(order model.PendingPaymentOrder, now time.Time) bool {
	if order.ExpiresAt.Valid && !order.ExpiresAt.Time.After(now) {
		return true
	}
	return !order.CreatedAt.Add(t.pendingTimeout).After(now)
}

func (t *PaymentReconciliationTask) markLocalOrderClosed(ctx context.Context, order model.PendingPaymentOrder) error {
	err := t.store.MarkPaymentOrderClosed(ctx, order.BusinessType, order.BusinessOrderID, order.OutTradeNo)
	if errors.Is(err, sql.ErrNoRows) {
		// 多实例任务可能同时处理同一订单；目标已被其他实例推进时视为幂等成功。
		return nil
	}
	return err
}

func isWechatOrderNotExist(err error) bool {
	var apiErr *paymentlogic.WechatPayAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.ToUpper(strings.TrimSpace(apiErr.Code))
	return code == "ORDER_NOT_EXIST" || code == "ORDERNOTEXIST"
}
