package task

import (
	"context"
	"encoding/json"
	"testing"
)

func TestManagerExecutionPersistsSuccessfulHandlerResult(t *testing.T) {
	store := &taskStoreStub{}
	queue := &queue{
		registry: newRegistry(),
		repo:     store,
	}
	queue.Register("example.v1", HandlerFunc(func(context.Context, *Task) (any, error) {
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
	if message.Status != StatusCompleted || string(message.Result) != string(store.result) {
		t.Fatalf("unexpected task state: status=%s result=%s", message.Status, message.Result)
	}
}
