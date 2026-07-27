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

	got, ok := registry.Get("example.v1")
	if !ok {
		t.Fatal("expected registered handler")
	}
	if got == nil {
		t.Fatal("expected non-nil handler")
	}

	message := &task.Task{Type: "generation.character.v1"}
	if err := got.Handle(context.Background(), message); err != nil {
		t.Fatalf("handle generic task: %v", err)
	}
}
