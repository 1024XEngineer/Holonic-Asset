package generator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type mockProjectReader struct {
	project *projectdomain.Project
	err     error
}

func (m *mockProjectReader) GetDetail(_ context.Context, _ uint) (*projectdomain.Project, error) {
	return m.project, m.err
}

type mockAssetReader struct {
	asset  assetdomain.Asset
	assets []assetdomain.Asset
	err    error
}

func (m *mockAssetReader) GetDetail(_ context.Context, _ uint) (assetdomain.Asset, error) {
	return m.asset, m.err
}

func (m *mockAssetReader) GetAssets(_ context.Context, _ uint, _ assetdomain.AssetListFilter) ([]assetdomain.Asset, error) {
	return m.assets, m.err
}

func TestRunManagerCreate(t *testing.T) {
	t.Run("nil tasks returns error", func(t *testing.T) {
		engine := &Engine{}
		_, err := engine.Create(context.Background(), &Request{Kind: GenerateCharacterProtoType})
		if !errors.Is(err, ErrTaskManagerRequired) {
			t.Fatalf("expected ErrTaskManagerRequired, got %v", err)
		}
	})

	t.Run("nil request returns error", func(t *testing.T) {
		tasks := newMockTaskManager()
		engine := &Engine{tasks: tasks}
		_, err := engine.Create(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("create character prototype with project and nexus references", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.publishID = 101
		refs := &mockTileSetReferenceStore{}
		projects := &mockProjectReader{
			project: &projectdomain.Project{
				Reference: "proj-ref",
			},
		}
		assets := &mockAssetReader{
			assets: []assetdomain.Asset{
				{
					ID:           1,
					ProjectID:    10,
					Type:         assetdomain.AssetTypeCharacter,
					ThumbnailURL: "thumb-1",
					Tags:         []assetdomain.Tag{{Name: "warrior"}, {Name: "human"}},
					Version:      2,
				},
				{
					ID:           2,
					ProjectID:    10,
					Type:         assetdomain.AssetTypeObject,
					ThumbnailURL: "thumb-2",
					Tags:         []assetdomain.Tag{{Name: "warrior"}},
					Version:      1,
				},
			},
		}

		engine := NewEngine(tasks, nil, EngineDependencies{
			Projects:   projects,
			Assets:     assets,
			References: refs,
		})

		req := &Request{
			Kind:          GenerateCharacterProtoType,
			ProjectID:     10,
			CreativeBrief: "A brave warrior",
			Parameters: json.RawMessage(`{
				"asset_name": "Warrior",
				"tags": [{"name": "warrior"}, {"name": "human"}]
			}`),
		}

		runID, err := engine.Create(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runID != 101 {
			t.Fatalf("got runID %d, want 101", runID)
		}
	})

	t.Run("create scenery fails if project reader missing", func(t *testing.T) {
		tasks := newMockTaskManager()
		engine := NewEngine(tasks, nil, EngineDependencies{})
		req := &Request{
			Kind:          GenerateScenery,
			ProjectID:     1,
			CreativeBrief: "A sunny beach",
			Parameters: json.RawMessage(`{
				"asset_name": "Beach",
				"dimensions": {"width": 1024, "height": 1024}
			}`),
		}
		_, err := engine.Create(context.Background(), req)
		if !errors.Is(err, ErrProjectReaderRequired) {
			t.Fatalf("expected ErrProjectReaderRequired, got %v", err)
		}
	})

	t.Run("referencePersistenceError converts untrusted reference", func(t *testing.T) {
		err := referencePersistenceError("creating", upload.ErrUntrustedReference)
		if !errors.Is(err, ErrInvalidTaskPayload) {
			t.Fatalf("expected ErrInvalidTaskPayload, got %v", err)
		}
	})
}

func TestRunManagerSelectNexusReferences(t *testing.T) {
	engine := &Engine{}

	t.Run("empty cases return nil", func(t *testing.T) {
		refs, err := engine.selectNexusReferences(context.Background(), 0, nil, "", 2)
		if err != nil || len(refs) != 0 {
			t.Fatalf("expected nil refs, got %v err=%v", refs, err)
		}

		refs, err = engine.selectNexusReferences(context.Background(), 1, []assetdomain.Tag{{Name: "tag"}}, "", 0)
		if err != nil || len(refs) != 0 {
			t.Fatalf("expected nil refs, got %v err=%v", refs, err)
		}
	})

	t.Run("missing asset reader returns error", func(t *testing.T) {
		_, err := engine.selectNexusReferences(context.Background(), 1, []assetdomain.Tag{{Name: "tag"}}, "", 2)
		if !errors.Is(err, ErrAssetReaderRequired) {
			t.Fatalf("expected ErrAssetReaderRequired, got %v", err)
		}
	})
}

func TestRunManagerList(t *testing.T) {
	tasks := newMockTaskManager()
	engine := NewEngine(tasks, nil, EngineDependencies{})

	t.Run("invalid status", func(t *testing.T) {
		_, err := engine.List(context.Background(), &RunListQuery{Status: "unknown"})
		if !errors.Is(err, ErrInvalidRunListStatus) {
			t.Fatalf("expected ErrInvalidRunListStatus, got %v", err)
		}
	})

	t.Run("default limit and project level query", func(t *testing.T) {
		page, err := engine.List(context.Background(), &RunListQuery{ProjectID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page == nil {
			t.Fatal("expected non-nil page")
		}
	})

	t.Run("asset level query with limit cap", func(t *testing.T) {
		assetID := uint(5)
		page, err := engine.List(context.Background(), &RunListQuery{
			ProjectID: 1,
			AssetID:   &assetID,
			Limit:     200, // Should be capped to 100
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page == nil {
			t.Fatal("expected non-nil page")
		}
	})
}

func TestRunManagerGet(t *testing.T) {
	t.Run("nil tasks returns error", func(t *testing.T) {
		engine := &Engine{}
		_, err := engine.Get(context.Background(), 1)
		if !errors.Is(err, ErrTaskManagerRequired) {
			t.Fatalf("expected ErrTaskManagerRequired, got %v", err)
		}
	})

	t.Run("get detail error", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailErr = errors.New("not found")
		engine := NewEngine(tasks, nil, EngineDependencies{})
		_, err := engine.Get(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("successful get", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Status:  taskdomain.StatusCompleted,
			Payload: []byte(`{"project_id": 10}`),
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		run, err := engine.Get(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.ID != 1 || run.ProjectID != 10 {
			t.Fatalf("unexpected run: %+v", run)
		}
	})
}

func TestRunManagerRetryAndDelete(t *testing.T) {
	t.Run("retry nil tasks returns error", func(t *testing.T) {
		engine := &Engine{}
		_, err := engine.Retry(context.Background(), 1)
		if !errors.Is(err, ErrTaskManagerRequired) {
			t.Fatalf("expected ErrTaskManagerRequired, got %v", err)
		}
	})

	t.Run("retry run not failed returns error", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Status:  taskdomain.StatusProcessing,
			Payload: []byte(`{"project_id": 1}`),
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		_, err := engine.Retry(context.Background(), 1)
		if !errors.Is(err, ErrRunNotFailed) {
			t.Fatalf("expected ErrRunNotFailed, got %v", err)
		}
	})

	t.Run("retry success", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Status:  taskdomain.StatusFailed,
			Payload: []byte(`{"project_id": 1}`),
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		runID, err := engine.Retry(context.Background(), 1)
		if err != nil || runID != 1 {
			t.Fatalf("unexpected retry result: runID=%d err=%v", runID, err)
		}
	})

	t.Run("delete run not failed returns error", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Status:  taskdomain.StatusCompleted,
			Payload: []byte(`{"project_id": 1}`),
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		err := engine.Delete(context.Background(), 1)
		if !errors.Is(err, ErrRunNotFailed) {
			t.Fatalf("expected ErrRunNotFailed, got %v", err)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Status:  taskdomain.StatusFailed,
			Payload: []byte(`{"project_id": 1}`),
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		err := engine.Delete(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRunManagerCancelAndResolveApplication(t *testing.T) {
	t.Run("cancel nil tasks returns error", func(t *testing.T) {
		engine := &Engine{}
		if err := engine.Cancel(context.Background(), 1); !errors.Is(err, ErrTaskManagerRequired) {
			t.Fatalf("expected ErrTaskManagerRequired, got %v", err)
		}
	})

	t.Run("cancel success", func(t *testing.T) {
		tasks := newMockTaskManager()
		engine := NewEngine(tasks, nil, EngineDependencies{})
		if err := engine.Cancel(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resolve application not awaiting status", func(t *testing.T) {
		tasks := newMockTaskManager()
		tasks.detailTask = &taskdomain.Task{
			ID:     1,
			Status: taskdomain.StatusCompleted,
		}
		engine := NewEngine(tasks, nil, EngineDependencies{})
		err := engine.ResolveApplication(context.Background(), 1, true)
		if err == nil {
			t.Fatal("expected error for non-awaiting status")
		}
	})

	t.Run("resolve application discard with resources", func(t *testing.T) {
		tasks := newMockTaskManager()
		resJSON, _ := json.Marshal(ExecutionResult{
			GeneratedResources: []string{"res-1", "res-2"},
		})
		tasks.detailTask = &taskdomain.Task{
			ID:      1,
			Status:  taskdomain.StatusAwaitingApplication,
			Result:  resJSON,
			Payload: []byte(`{"project_id": 1}`),
		}
		refs := &mockTileSetReferenceStore{}
		engine := NewEngine(tasks, nil, EngineDependencies{References: refs})

		err := engine.ResolveApplication(context.Background(), 1, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(refs.deletedKeys) != 2 {
			t.Fatalf("expected 2 deleted resources, got %d (%v)", len(refs.deletedKeys), refs.deletedKeys)
		}
	})
}
