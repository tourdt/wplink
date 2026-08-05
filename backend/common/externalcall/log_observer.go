package externalcall

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogObserver struct{}

func NewLogObserver() Observer {
	return LogObserver{}
}

func (LogObserver) Observe(_ context.Context, event Event) {
	fields := []logx.LogField{
		logx.Field("event", EventName),
		logx.Field("provider", event.Provider),
		logx.Field("operation", event.Operation),
		logx.Field("outcome", string(event.Outcome)),
		logx.Field("duration_ms", event.Duration.Milliseconds()),
	}
	if event.StatusCode > 0 {
		fields = append(fields, logx.Field("status_code", event.StatusCode))
	}

	switch event.Outcome {
	case OutcomeSuccess, OutcomeCanceled, OutcomeProviderRejected:
		logx.Infow("第三方调用完成", fields...)
	default:
		logx.Errorw("第三方调用失败", fields...)
	}
}
