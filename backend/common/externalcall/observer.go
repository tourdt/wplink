package externalcall

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

const EventName = "external_call"

const (
	ProviderWechat     = "wechat"
	ProviderSMS        = "sms"
	ProviderTencentMap = "tencent_map"
)

const (
	OperationCodeToSession      = "code_to_session"
	OperationPhoneNumber        = "phone_number"
	OperationAccessToken        = "access_token"
	OperationContentTextCheck   = "content_text_check"
	OperationContentMediaSubmit = "content_media_submit"
	OperationPayCreate          = "pay_create"
	OperationPayQuery           = "pay_query"
	OperationPayClose           = "pay_close"
	OperationSMSCodeSend        = "code_send"
	OperationSMSCodeVerify      = "code_verify"
	OperationReverseGeocode     = "reverse_geocode"
)

type Outcome string

const (
	OutcomeSuccess          Outcome = "success"
	OutcomeCanceled         Outcome = "canceled"
	OutcomeTimeout          Outcome = "timeout"
	OutcomeTransportError   Outcome = "transport_error"
	OutcomeHTTPError        Outcome = "http_error"
	OutcomeProviderRejected Outcome = "provider_rejected"
	OutcomeDecodeError      Outcome = "decode_error"
)

type Event struct {
	Provider   string
	Operation  string
	Outcome    Outcome
	Duration   time.Duration
	StatusCode int
}

type Observer interface {
	Observe(ctx context.Context, event Event)
}

type Call struct {
	observer  Observer
	provider  string
	operation string
	startedAt time.Time
	now       func() time.Time
	once      sync.Once
}

func Start(observer Observer, provider string, operation string) *Call {
	return startWithClock(observer, provider, operation, time.Now)
}

func startWithClock(observer Observer, provider string, operation string, now func() time.Time) *Call {
	if observer == nil {
		observer = NewLogObserver()
	}
	if now == nil {
		now = time.Now
	}

	return &Call{
		observer:  observer,
		provider:  provider,
		operation: operation,
		startedAt: now(),
		now:       now,
	}
}

func (c *Call) Finish(ctx context.Context, outcome Outcome, statusCode int) {
	if c == nil {
		return
	}

	c.once.Do(func() {
		duration := c.now().Sub(c.startedAt)
		if duration < 0 {
			duration = 0
		}
		if statusCode < 0 {
			statusCode = 0
		}
		c.observer.Observe(ctx, Event{
			Provider:   c.provider,
			Operation:  c.operation,
			Outcome:    outcome,
			Duration:   duration,
			StatusCode: statusCode,
		})
	})
}

func ClassifyTransport(ctx context.Context, err error) Outcome {
	if errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)) {
		return OutcomeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
		return OutcomeTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return OutcomeTimeout
	}

	return OutcomeTransportError
}
