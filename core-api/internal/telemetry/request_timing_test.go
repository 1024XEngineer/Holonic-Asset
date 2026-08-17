package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/telemetry"
)

func TestRequestTimingRecordsPhasesOnInstrumentedContext(t *testing.T) {
	ctx, timing := telemetry.WithRequestTiming(context.Background())
	telemetry.RecordRequestTiming(ctx, "store", 12*time.Millisecond)

	metrics := timing.Metrics()
	if len(metrics) != 1 || metrics[0].Name != "store" || metrics[0].Duration != 12*time.Millisecond {
		t.Fatalf("unexpected timing metrics: %+v", metrics)
	}

	telemetry.RecordRequestTiming(context.Background(), "ignored", time.Second)
	if len(timing.Metrics()) != 1 {
		t.Fatalf("un-instrumented context changed metrics: %+v", timing.Metrics())
	}
}
