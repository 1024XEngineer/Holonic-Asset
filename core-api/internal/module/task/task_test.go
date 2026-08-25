package task

import (
	"context"
	"errors"
	"testing"
)

type handler struct{}

func (handler) Handle(context.Context, *Task) (any, error) { return struct{}{}, nil }

func TestRegistryIsBusinessAgnostic(t *testing.T) {
	registry := newRegistry()
	registry.register("example.v1", handler{})

	if _, err := registry.dispatch(context.Background(), &Task{Type: "example.v1"}); err != nil {
		t.Fatalf("dispatch generic task: %v", err)
	}
}

func TestRegistryDispatchReturnsErrorForUnknownType(t *testing.T) {
	registry := newRegistry()

	if _, err := registry.dispatch(context.Background(), &Task{Type: "unknown.v1"}); err == nil {
		t.Fatal("expected unknown task type error")
	}
}

func TestAwaitingApplicationStatusString(t *testing.T) {
	if got := StatusAwaitingApplication.String(); got != "awaiting_application" {
		t.Fatalf("unexpected awaiting application status: %q", got)
	}
}

func TestStatusStringCoversKnownAndUnknownValues(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPending, "pending"},
		{StatusProcessing, "processing"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
		{StatusAwaitingApplication, "awaiting_application"},
		{Status(99), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("status %d = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestRegistryDispatchRejectsNilTaskAndPropagatesHandlerError(t *testing.T) {
	registry := newRegistry()
	if _, err := registry.dispatch(context.Background(), nil); err == nil {
		t.Fatal("expected nil registry dispatch to fail")
	}
	wantErr := errors.New("business failure")
	registry.register("error", HandlerFunc(func(context.Context, *Task) (any, error) { return nil, wantErr }))
	if _, err := registry.dispatch(context.Background(), &Task{Type: "error"}); !errors.Is(err, wantErr) {
		t.Fatalf("registry error = %v, want %v", err, wantErr)
	}
}
