package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	riverqueue "github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
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

func TestManagerExecutionCanWaitForApplication(t *testing.T) {
	store := &taskStoreStub{}
	queue := &queue{registry: newRegistry(), repo: store}
	queue.Register("candidate.v1", HandlerFunc(func(context.Context, *Task) (any, error) {
		return map[string]any{"candidate": true}, nil
	}))
	message := &Task{
		ID:               8,
		Type:             "candidate.v1",
		CompletionStatus: StatusAwaitingApplication,
	}

	if err := queue.dispatch(context.Background(), message); err != nil {
		t.Fatalf("dispatch candidate task: %v", err)
	}
	if store.resultStatus != StatusAwaitingApplication || message.Status != StatusAwaitingApplication {
		t.Fatalf("candidate did not wait for application: persisted=%s message=%s", store.resultStatus, message.Status)
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

func TestQueueErrorHandlerMarksTaskFailedImmediately(t *testing.T) {
	store := &taskStoreStub{}
	handler := &queueErrorHandler{repo: store}
	job := &rivertype.JobRow{
		Kind:        queueTaskKind,
		Attempt:     1,
		MaxAttempts: 3,
		EncodedArgs: []byte(`{"task":{"id":7}}`),
	}

	result := handler.HandleError(context.Background(), job, errors.New("handler failure"))
	if result == nil || !result.SetCancelled {
		t.Fatalf("handler failure must stop River retries, got %+v", result)
	}
	if len(store.statusUpdates) != 1 || store.statusUpdates[0] != StatusFailed {
		t.Fatalf("unexpected failure status updates: %v", store.statusUpdates)
	}
	if store.failure != "handler failure" {
		t.Fatalf("persisted failure = %q, want handler failure", store.failure)
	}
}

func TestQueueErrorHandlerMarksTimedOutTaskFailedImmediately(t *testing.T) {
	store := &taskStoreStub{}
	handler := &queueErrorHandler{repo: store}
	job := &rivertype.JobRow{
		Kind:        queueTaskKind,
		Attempt:     1,
		MaxAttempts: 3,
		EncodedArgs: []byte(`{"task":{"id":19}}`),
	}
	timeoutErr := fmt.Errorf("generate animation: %w", context.DeadlineExceeded)

	result := handler.HandleError(context.Background(), job, timeoutErr)
	if result == nil || !result.SetCancelled {
		t.Fatalf("timeout must stop River retries, got %+v", result)
	}
	if len(store.statusUpdates) != 1 || store.statusUpdates[0] != StatusFailed {
		t.Fatalf("unexpected timeout status updates: %v", store.statusUpdates)
	}
	if !strings.Contains(store.failure, context.DeadlineExceeded.Error()) {
		t.Fatalf("persisted failure = %q", store.failure)
	}
}

func TestQueueDispatchRejectsNilAndUnknownTasks(t *testing.T) {
	store := &executionStoreStub{}
	queue := &queue{registry: newRegistry(), repo: store}

	if err := queue.dispatch(context.Background(), nil); err == nil {
		t.Fatal("expected nil task dispatch to fail")
	}
	if err := queue.dispatch(context.Background(), &Task{ID: 4, Type: "missing"}); err == nil || !strings.Contains(err.Error(), "no handler") {
		t.Fatalf("unknown task error = %v", err)
	}
	if store.status != StatusProcessing {
		t.Fatalf("unknown task should still transition to processing, got %s", store.status)
	}
}

func TestQueueDispatchPropagatesHandlerAndPersistenceErrors(t *testing.T) {
	t.Run("handler error", func(t *testing.T) {
		store := &executionStoreStub{}
		queue := &queue{registry: newRegistry(), repo: store}
		wantErr := errors.New("handler failed")
		queue.Register("broken", HandlerFunc(func(context.Context, *Task) (any, error) { return nil, wantErr }))

		err := queue.dispatch(context.Background(), &Task{ID: 5, Type: "broken"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("dispatch error = %v, want %v", err, wantErr)
		}
		if store.resultCall != 0 {
			t.Fatalf("handler failure must not persist a result")
		}
	})

	t.Run("result encoding", func(t *testing.T) {
		store := &executionStoreStub{}
		queue := &queue{registry: newRegistry(), repo: store}
		queue.Register("unencodable", HandlerFunc(func(context.Context, *Task) (any, error) {
			return make(chan int), nil
		}))

		err := queue.dispatch(context.Background(), &Task{ID: 6, Type: "unencodable"})
		if err == nil || !strings.Contains(err.Error(), "encode result") {
			t.Fatalf("encoding error = %v", err)
		}
		if store.resultCall != 0 {
			t.Fatalf("encoding failure must not persist a result")
		}
	})

	t.Run("result persistence", func(t *testing.T) {
		wantErr := errors.New("result store failed")
		store := &executionStoreStub{resultErr: wantErr}
		queue := &queue{registry: newRegistry(), repo: store}
		queue.Register("result-error", HandlerFunc(func(context.Context, *Task) (any, error) {
			return map[string]bool{"ok": true}, nil
		}))

		err := queue.dispatch(context.Background(), &Task{ID: 7, Type: "result-error"})
		if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "persist result") {
			t.Fatalf("persistence error = %v", err)
		}
	})
}

func TestQueuePublishRejectsNilTask(t *testing.T) {
	queue := &queue{}
	if err := queue.publish(context.Background(), nil); err == nil {
		t.Fatal("expected nil queue publish to fail")
	}
}

type queueClientStub struct {
	insertResult *rivertype.JobInsertResult
	insertErr    error
	startErr     error
	stopErr      error
	started      bool
	stopped      bool
}

func (s *queueClientStub) Insert(context.Context, riverqueue.JobArgs, *riverqueue.InsertOpts) (*rivertype.JobInsertResult, error) {
	if s.insertErr != nil {
		return nil, s.insertErr
	}
	if s.insertResult != nil {
		return s.insertResult, nil
	}
	return &rivertype.JobInsertResult{}, nil
}

func (s *queueClientStub) Start(context.Context) error {
	s.started = true
	return s.startErr
}

func (s *queueClientStub) Stop(context.Context) error {
	s.stopped = true
	return s.stopErr
}

func TestQueuePublishForwardsTasksAndHandlesRiverResults(t *testing.T) {
	t.Run("insert error", func(t *testing.T) {
		wantErr := errors.New("queue unavailable")
		queue := &queue{client: &queueClientStub{insertErr: wantErr}}
		err := queue.publish(context.Background(), &Task{Type: "example.v1"})
		if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), `publish task "example.v1"`) {
			t.Fatalf("publish error = %v", err)
		}
	})

	for _, duplicate := range []bool{false, true} {
		t.Run(fmt.Sprintf("duplicate_%t", duplicate), func(t *testing.T) {
			client := &queueClientStub{insertResult: &rivertype.JobInsertResult{UniqueSkippedAsDuplicate: duplicate}}
			queue := &queue{client: client}
			if err := queue.publish(context.Background(), &Task{ID: 9, Type: "example.v1"}); err != nil {
				t.Fatalf("publish task: %v", err)
			}
		})
	}
}

func TestQueueStartPropagatesClientErrors(t *testing.T) {
	wantErr := errors.New("database unavailable")
	queue := &queue{client: &queueClientStub{startErr: wantErr}}
	if err := queue.start(context.Background()); !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "start queue") {
		t.Fatalf("start error = %v", err)
	}

	client := &queueClientStub{}
	queue.client = client
	if err := queue.start(context.Background()); err != nil || !client.started {
		t.Fatalf("start success: err=%v started=%v", err, client.started)
	}
}

func TestQueueTaskArgsExposeRiverKind(t *testing.T) {
	if got := (queueTaskArgs{}).Kind(); got != queueTaskKind {
		t.Fatalf("queue task kind = %q, want %q", got, queueTaskKind)
	}
}

func TestQueueWorkerDelegatesToDispatch(t *testing.T) {
	store := &executionStoreStub{}
	queue := &queue{registry: newRegistry(), repo: store}
	queue.Register("worker-task", HandlerFunc(func(context.Context, *Task) (any, error) {
		return "done", nil
	}))
	worker := &queueWorker{queue: queue}

	job := &riverqueue.Job[queueTaskArgs]{Args: queueTaskArgs{Task: Task{ID: 12, Type: "worker-task"}}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("worker dispatch: %v", err)
	}
	if store.status != StatusProcessing || store.resultCall != 1 {
		t.Fatalf("worker did not dispatch task: status=%s resultCalls=%d", store.status, store.resultCall)
	}
}

func TestQueueErrorHandlerHandlesPanicAndSkipsInvalidJobs(t *testing.T) {
	store := &executionStoreStub{}
	handler := &queueErrorHandler{repo: store}
	job := &rivertype.JobRow{Kind: queueTaskKind, ID: 20, EncodedArgs: []byte(`{"task":{"id":21}}`)}

	result := handler.HandlePanic(context.Background(), job, "boom", "stack")
	if result == nil || !result.SetCancelled || store.failureID != 21 || store.failure != "task panicked: boom" {
		t.Fatalf("panic handling state: result=%+v store=%+v", result, store)
	}

	store.failureID = 0
	for _, invalid := range []*rivertype.JobRow{
		nil,
		{Kind: "other", EncodedArgs: []byte(`{"task":{"id":22}}`)},
		{Kind: queueTaskKind, Attempt: 1, MaxAttempts: 3, EncodedArgs: []byte(`{"task":{"id":23}}`)},
		{Kind: queueTaskKind, EncodedArgs: []byte(`not-json`)},
		{Kind: queueTaskKind, EncodedArgs: []byte(`{"task":{}}`)},
	} {
		handler.markFailed(context.Background(), invalid, nil, false)
	}
	if store.failureID != 0 {
		t.Fatalf("invalid jobs should not update failure state, got ID %d", store.failureID)
	}
}

func TestQueueErrorHandlerUsesDefaultFailureAndIgnoresStoreError(t *testing.T) {
	store := &executionStoreStub{failureErr: errors.New("database unavailable")}
	handler := &queueErrorHandler{repo: store}
	job := &rivertype.JobRow{Kind: queueTaskKind, EncodedArgs: []byte(`{"task":{"id":24}}`)}

	handler.markFailed(context.Background(), job, nil, true)
	if store.failureID != 0 {
		t.Fatal("store error should not be observable as a successful failure write")
	}
}

func TestQueueErrorHandlerUsesDefaultFailureMessage(t *testing.T) {
	store := &executionStoreStub{}
	handler := &queueErrorHandler{repo: store}
	job := &rivertype.JobRow{Kind: queueTaskKind, EncodedArgs: []byte(`{"task":{"id":25}}`)}

	handler.markFailed(context.Background(), job, nil, true)
	if store.failureID != 25 || store.failure != "task execution failed" {
		t.Fatalf("default failure state: id=%d message=%q", store.failureID, store.failure)
	}
}
