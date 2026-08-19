package task

import (
	"context"
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
