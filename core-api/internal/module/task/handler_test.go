package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestManagerExecutionPersistsSuccessfulHandlerResult(t *testing.T) {
	store := &taskStoreStub{}
	queue := &queue{
		registry: newRegistry(),
		repo:     store,
	}
	queue.Register("example.v1", HandlerFunc(func(_ context.Context, message *Task) (any, error) {
		if store.status != StatusProcessing || message.Status != StatusProcessing {
			t.Fatalf("handler received task before processing transition: persisted=%s message=%s",
				store.status, message.Status)
		}
		return map[string]any{"ok": true}, nil
	}))

	message := &Task{ID: 7, Type: "example.v1"}
	if err := queue.dispatch(context.Background(), message); err != nil {
		t.Fatalf("dispatch task: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(store.result, &result); err != nil {
		t.Fatalf("decode persisted result: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected persisted result: %v", result)
	}
	if len(store.statusUpdates) != 1 || store.statusUpdates[0] != StatusProcessing {
		t.Fatalf("unexpected status updates: %v", store.statusUpdates)
	}
	if store.resultCalls != 1 {
		t.Fatalf("expected one completed result update, got %d", store.resultCalls)
	}
	if message.Status != StatusCompleted || string(message.Result) != string(store.result) {
		t.Fatalf("unexpected task state: status=%s result=%s", message.Status, message.Result)
	}
}

func TestManagerExecutionDoesNotInvokeHandlerWhenProcessingTransitionFails(t *testing.T) {
	statusErr := errors.New("database unavailable")
	store := &taskStoreStub{statusErr: statusErr}
	queue := &queue{
		registry: newRegistry(),
		repo:     store,
	}
	handlerCalled := false
	queue.Register("example.v1", HandlerFunc(func(context.Context, *Task) (any, error) {
		handlerCalled = true
		return struct{}{}, nil
	}))

	message := &Task{ID: 7, Type: "example.v1", Status: StatusPending}
	err := queue.dispatch(context.Background(), message)
	if !errors.Is(err, statusErr) {
		t.Fatalf("expected processing transition error, got %v", err)
	}
	if handlerCalled {
		t.Fatal("handler must not run before the processing transition is persisted")
	}
	if message.Status != StatusPending {
		t.Fatalf("unexpected in-memory task status: %s", message.Status)
	}
	if store.resultCalls != 0 {
		t.Fatalf("result must not be persisted, got %d calls", store.resultCalls)
	}
}
