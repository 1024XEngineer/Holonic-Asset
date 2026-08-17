package telemetry

import (
	"context"
	"time"
)

type requestTimingKey struct{}

type RequestTiming struct {
	metrics []Metric
}

type Metric struct {
	Name     string
	Duration time.Duration
}

func WithRequestTiming(ctx context.Context) (context.Context, *RequestTiming) {
	timing := &RequestTiming{}
	return context.WithValue(ctx, requestTimingKey{}, timing), timing
}

func RecordRequestTiming(ctx context.Context, name string, duration time.Duration) {
	timing, _ := ctx.Value(requestTimingKey{}).(*RequestTiming)
	if timing == nil {
		return
	}
	timing.metrics = append(timing.metrics, Metric{Name: name, Duration: duration})
}

func (t *RequestTiming) Metrics() []Metric {
	if t == nil {
		return nil
	}
	return append([]Metric(nil), t.metrics...)
}
