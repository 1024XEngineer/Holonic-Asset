package task_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type handler struct{}

func (handler) Handle(context.Context, *task.Task) error { return nil }

func TestRegistryIsBusinessAgnostic(t *testing.T) {
	registry := task.NewRegistry()
	registry.Register("example.v1", handler{})

	if err := registry.Dispatch(context.Background(), &task.Task{Type: "example.v1"}); err != nil {
		t.Fatalf("dispatch generic task: %v", err)
	}
}

func TestRegistryDispatchReturnsErrorForUnknownType(t *testing.T) {
	registry := task.NewRegistry()

	if err := registry.Dispatch(context.Background(), &task.Task{Type: "unknown.v1"}); err == nil {
		t.Fatal("expected unknown task type error")
	}
}
