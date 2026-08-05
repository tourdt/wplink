package externalcall

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type recordingObserver struct {
	events []Event
}

func (o *recordingObserver) Observe(_ context.Context, event Event) {
	o.events = append(o.events, event)
}

func TestCallFinishReportsOnceAndClampsNegativeDuration(t *testing.T) {
	base := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	times := []time.Time{base, base.Add(-time.Second)}
	index := 0
	now := func() time.Time {
		value := times[index]
		index++
		return value
	}
	recorder := &recordingObserver{}
	call := startWithClock(recorder, ProviderWechat, OperationCodeToSession, now)

	call.Finish(context.Background(), OutcomeSuccess, 200)
	call.Finish(context.Background(), OutcomeHTTPError, 500)

	if len(recorder.events) != 1 {
		t.Fatalf("events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.Duration != 0 || event.Outcome != OutcomeSuccess || event.StatusCode != 200 {
		t.Fatalf("event = %+v, want clamped success event", event)
	}
}

type channelObserver struct {
	events chan Event
}

func (o channelObserver) Observe(_ context.Context, event Event) {
	o.events <- event
}

func TestCallFinishReportsOnceWhenCalledConcurrently(t *testing.T) {
	observer := channelObserver{events: make(chan Event, 32)}
	call := startWithClock(observer, ProviderWechat, OperationCodeToSession, time.Now)

	var waitGroup sync.WaitGroup
	for i := 0; i < 32; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			call.Finish(context.Background(), OutcomeSuccess, 200)
		}()
	}
	waitGroup.Wait()

	select {
	case <-observer.events:
	default:
		t.Fatal("Finish() did not report an event")
	}
	select {
	case event := <-observer.events:
		t.Fatalf("Finish() reported more than once: %+v", event)
	default:
	}
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

var _ net.Error = timeoutNetError{}

func TestClassifyTransport(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want Outcome
	}{
		{name: "canceled", ctx: canceledContext(), err: context.Canceled, want: OutcomeCanceled},
		{name: "deadline", ctx: context.Background(), err: context.DeadlineExceeded, want: OutcomeTimeout},
		{name: "net timeout", ctx: context.Background(), err: timeoutNetError{}, want: OutcomeTimeout},
		{name: "network", ctx: context.Background(), err: errors.New("connection reset"), want: OutcomeTransportError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyTransport(tt.ctx, tt.err); got != tt.want {
				t.Fatalf("ClassifyTransport() = %q, want %q", got, tt.want)
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
