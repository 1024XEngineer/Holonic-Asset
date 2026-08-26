package generator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type mockTaskManager struct {
	tasks       []*taskdomain.Task
	listErr     error
	listFunc    func(ctx context.Context, filter *taskdomain.ListFilter) ([]*taskdomain.Task, error)
	detailTask  *taskdomain.Task
	detailErr   error
	retryErr    error
	deleteErr   error
	cancelErr   error
	completeErr error
	publishID   uint
	publishErr  error
	handlers    map[string]taskdomain.Handler
}

func newMockTaskManager() *mockTaskManager {
	return &mockTaskManager{handlers: make(map[string]taskdomain.Handler)}
}

func (m *mockTaskManager) Start(_ context.Context) error {
	return nil
}

func (m *mockTaskManager) Stop() error {
	return nil
}

func (m *mockTaskManager) Publish(_ context.Context, _ *taskdomain.Task) (uint, error) {
	return m.publishID, m.publishErr
}

func (m *mockTaskManager) Register(taskType string, handler taskdomain.Handler) {
	if m.handlers == nil {
		m.handlers = make(map[string]taskdomain.Handler)
	}
	m.handlers[taskType] = handler
}

func (m *mockTaskManager) Dispatch(ctx context.Context, task *taskdomain.Task) (any, error) {
	if task == nil {
		return nil, errors.New("task required")
	}
	h, ok := m.handlers[task.Type]
	if !ok {
		return nil, errors.New("handler not found")
	}
	return h.Handle(ctx, task)
}

func (m *mockTaskManager) List(ctx context.Context, filter *taskdomain.ListFilter) ([]*taskdomain.Task, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return m.tasks, m.listErr
}

func (m *mockTaskManager) GetDetail(_ context.Context, _ uint) (*taskdomain.Task, error) {
	return m.detailTask, m.detailErr
}

func (m *mockTaskManager) RetryFailed(_ context.Context, _ uint, _ taskdomain.Status) error {
	return m.retryErr
}

func (m *mockTaskManager) DeleteFailed(_ context.Context, _ uint) error {
	return m.deleteErr
}

func (m *mockTaskManager) Cancel(_ context.Context, _ uint) error {
	return m.cancelErr
}

func (m *mockTaskManager) Complete(_ context.Context, _ uint) error {
	return m.completeErr
}

var _ taskdomain.Manager = (*mockTaskManager)(nil)

func TestRunReaderListRuns(t *testing.T) {
	t.Run("nil filter", func(t *testing.T) {
		reader := NewRunReader(newMockTaskManager())
		_, err := reader.ListRuns(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil filter")
		}
	})

	t.Run("nil reader or tasks returns empty page", func(t *testing.T) {
		var nilReader *RunReader
		page, err := nilReader.ListRuns(context.Background(), &RunListFilter{})
		if err != nil || len(page.Runs) != 0 {
			t.Fatalf("unexpected result: page=%+v err=%v", page, err)
		}

		readerWithNilTasks := NewRunReader(nil)
		page, err = readerWithNilTasks.ListRuns(context.Background(), &RunListFilter{})
		if err != nil || len(page.Runs) != 0 {
			t.Fatalf("unexpected result: page=%+v err=%v", page, err)
		}
	})

	t.Run("invalid cursor returns error", func(t *testing.T) {
		reader := NewRunReader(newMockTaskManager())
		_, err := reader.ListRuns(context.Background(), &RunListFilter{
			Cursor: "invalid-cursor",
		})
		if err == nil {
			t.Fatal("expected error for invalid cursor")
		}
	})

	t.Run("empty statuses or types returns empty page", func(t *testing.T) {
		reader := NewRunReader(newMockTaskManager())
		page, err := reader.ListRuns(context.Background(), &RunListFilter{
			Statuses:         nil,
			IncludeTaskTypes: []TaskType{GenerateCharacterProtoType},
		})
		if err != nil || len(page.Runs) != 0 {
			t.Fatalf("unexpected page: %+v", page)
		}
	})

	t.Run("tasks list error", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.listErr = errors.New("db error")
		reader := NewRunReader(tasks)
		_, err := reader.ListRuns(context.Background(), &RunListFilter{
			ProjectID:        1,
			Statuses:         []taskdomain.Status{taskdomain.StatusPending},
			IncludeTaskTypes: []TaskType{GenerateCharacterProtoType},
		})
		if err == nil {
			t.Fatal("expected error from tasks.List")
		}
	})

	t.Run("task with invalid payload error", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.tasks = []*taskdomain.Task{
			{
				ID:      10,
				Type:    string(GenerateCharacterProtoType),
				Payload: []byte(`{invalid`),
			},
		}
		reader := NewRunReader(tasks)
		_, err := reader.ListRuns(context.Background(), &RunListFilter{
			ProjectID:        1,
			Statuses:         []taskdomain.Status{taskdomain.StatusPending},
			IncludeTaskTypes: []TaskType{GenerateCharacterProtoType},
		})
		if err == nil {
			t.Fatal("expected error decoding task payload")
		}
	})

	t.Run("pagination and filtering with scope and next cursor", func(t *testing.T) {
		assetID1 := uint(100)
		assetID2 := uint(200)
		task1Payload, _ := json.Marshal(map[string]any{"project_id": 1, "asset_id": assetID1})
		task2Payload, _ := json.Marshal(map[string]any{"project_id": 1, "parent_id": assetID1})
		task3Payload, _ := json.Marshal(map[string]any{"project_id": 1, "asset_id": assetID2})
		task4Payload, _ := json.Marshal(map[string]any{"project_id": 2, "asset_id": assetID1})

		tasks := newMockTaskManager()
		tasks.tasks = []*taskdomain.Task{
			{ID: 104, Type: string(GenerateCharacterProtoType), Status: taskdomain.StatusCompleted, Payload: task1Payload},
			{ID: 103, Type: string(GenerateCharacterProtoType), Status: taskdomain.StatusCompleted, Payload: task2Payload},
			{ID: 102, Type: string(GenerateCharacterProtoType), Status: taskdomain.StatusCompleted, Payload: task3Payload},
			{ID: 101, Type: string(GenerateCharacterProtoType), Status: taskdomain.StatusCompleted, Payload: task4Payload},
		}
		reader := NewRunReader(tasks)

		page, err := reader.ListRuns(context.Background(), &RunListFilter{
			ProjectID:        1,
			AssetID:          &assetID1,
			Statuses:         []taskdomain.Status{taskdomain.StatusCompleted},
			IncludeTaskTypes: []TaskType{GenerateCharacterProtoType},
			Limit:            1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Runs) != 1 {
			t.Fatalf("expected 1 run, got %d", len(page.Runs))
		}
		if page.Runs[0].ID != 104 {
			t.Fatalf("expected run ID 104, got %d", page.Runs[0].ID)
		}
		if page.NextCursor != "104" {
			t.Fatalf("expected next cursor '104', got %q", page.NextCursor)
		}
	})

	t.Run("multi-batch iteration when tasks batch is full", func(t *testing.T) {
		batch1 := make([]*taskdomain.Task, 100)
		for i := range 100 {
			p, _ := json.Marshal(map[string]any{"project_id": 999}) // does not match target project 1
			batch1[i] = &taskdomain.Task{
				ID:      uint(1000 - i),
				Type:    string(GenerateCharacterProtoType),
				Status:  taskdomain.StatusCompleted,
				Payload: p,
			}
		}

		targetPayload, _ := json.Marshal(map[string]any{"project_id": 1})
		batch2 := []*taskdomain.Task{
			{
				ID:      800,
				Type:    string(GenerateCharacterProtoType),
				Status:  taskdomain.StatusCompleted,
				Payload: targetPayload,
			},
		}

		callCount := 0
		tasksMock := newMockTaskManager()
		tasksMock.listFunc = func(_ context.Context, filter *taskdomain.ListFilter) ([]*taskdomain.Task, error) {
			callCount++
			if filter.BeforeID == 0 {
				return batch1, nil
			}
			return batch2, nil
		}
		reader := NewRunReader(tasksMock)
		page, err := reader.ListRuns(context.Background(), &RunListFilter{
			ProjectID:        1,
			Statuses:         []taskdomain.Status{taskdomain.StatusCompleted},
			IncludeTaskTypes: []TaskType{GenerateCharacterProtoType},
			Limit:            10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Runs) != 1 || page.Runs[0].ID != 800 {
			t.Fatalf("expected run ID 800 from second batch: %+v", page)
		}
	})
}

func TestDecodeRunCursor(t *testing.T) {
	tests := []struct {
		cursor  string
		wantID  uint
		wantErr bool
	}{
		{cursor: "", wantID: 0, wantErr: false},
		{cursor: "123", wantID: 123, wantErr: false},
		{cursor: "0", wantID: 0, wantErr: true},
		{cursor: "abc", wantID: 0, wantErr: true},
		{cursor: "-5", wantID: 0, wantErr: true},
	}

	for _, tt := range tests {
		got, err := decodeRunCursor(tt.cursor)
		if (err != nil) != tt.wantErr {
			t.Errorf("decodeRunCursor(%q) error = %v, wantErr = %v", tt.cursor, err, tt.wantErr)
		}
		if got != tt.wantID {
			t.Errorf("decodeRunCursor(%q) got = %d, want = %d", tt.cursor, got, tt.wantID)
		}
	}
}

func TestFilteredTaskTypes(t *testing.T) {
	all := filteredTaskTypes(nil, nil)
	if len(all) != len(TaskTypes()) {
		t.Fatalf("expected all task types, got %d", len(all))
	}

	included := filteredTaskTypes([]TaskType{GenerateCharacterProtoType, GenerateAnimation}, nil)
	if len(included) != 2 {
		t.Fatalf("expected 2 included task types, got %d", len(included))
	}

	excluded := filteredTaskTypes(nil, []TaskType{GenerateCharacterProtoType})
	if len(excluded) != len(TaskTypes())-1 {
		t.Fatalf("expected %d task types, got %d", len(TaskTypes())-1, len(excluded))
	}
}
