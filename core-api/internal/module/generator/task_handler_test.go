package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type mockExecutor struct {
	result   json.RawMessage
	err      error
	taskType TaskType
	payload  json.RawMessage
}

func (m *mockExecutor) Generate(_ context.Context, taskType TaskType, payload json.RawMessage) (json.RawMessage, error) {
	m.taskType = taskType
	m.payload = payload
	return m.result, m.err
}

func TestEngineTaskHandlers(t *testing.T) {
	expectedRes := json.RawMessage(`{"status":"ok"}`)
	exec := &mockExecutor{result: expectedRes}
	engine := &Engine{executor: exec}

	t.Run("nil message returns error", func(t *testing.T) {
		if _, err := engine.handleCharacterPrototype(context.Background(), nil); !errors.Is(err, ErrTaskRequired) {
			t.Fatalf("expected ErrTaskRequired, got %v", err)
		}
		if _, err := engine.handleTileSet(context.Background(), nil); !errors.Is(err, ErrTaskRequired) {
			t.Fatalf("expected ErrTaskRequired, got %v", err)
		}
	})

	t.Run("nil executor returns error", func(t *testing.T) {
		nilEngine := &Engine{executor: nil}
		validPayload, _ := json.Marshal(CreateCharacterPrototypePayload{
			AssetName:     "Hero",
			CreativeBrief: "brief",
		})
		_, err := nilEngine.handleCharacterPrototype(context.Background(), &taskdomain.Task{
			ID:      1,
			Type:    string(GenerateCharacterProtoType),
			Payload: validPayload,
		})
		if !errors.Is(err, ErrExecutorRequired) {
			t.Fatalf("expected ErrExecutorRequired, got %v", err)
		}
	})

	t.Run("handleCharacterPrototype success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(CreateCharacterPrototypePayload{AssetName: "Hero"})
		res, err := engine.handleCharacterPrototype(context.Background(), &taskdomain.Task{ID: 1, Type: string(GenerateCharacterProtoType), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleCharacterPrototype(context.Background(), &taskdomain.Task{ID: 1, Type: string(GenerateCharacterProtoType), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditCharacterPrototype success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(EditCharacterPrototypePayload{AssetID: 1})
		res, err := engine.handleEditCharacterPrototype(context.Background(), &taskdomain.Task{ID: 2, Type: string(EditCharacterProtoType), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleEditCharacterPrototype(context.Background(), &taskdomain.Task{ID: 2, Type: string(EditCharacterProtoType), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditObjectPrototype success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(EditObjectPrototypePayload{AssetID: 1})
		res, err := engine.handleEditObjectPrototype(context.Background(), &taskdomain.Task{ID: 3, Type: string(EditObjectProtoType), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleEditObjectPrototype(context.Background(), &taskdomain.Task{ID: 3, Type: string(EditObjectProtoType), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleAnimation success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(CreateAnimationPayload{AnimationName: "Walk"})
		res, err := engine.handleAnimation(context.Background(), &taskdomain.Task{ID: 4, Type: string(GenerateAnimation), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleAnimation(context.Background(), &taskdomain.Task{ID: 4, Type: string(GenerateAnimation), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditAnimation success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(EditAnimationPayload{AnimationID: 1})
		res, err := engine.handleEditAnimation(context.Background(), &taskdomain.Task{ID: 5, Type: string(EditAnimation), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleEditAnimation(context.Background(), &taskdomain.Task{ID: 5, Type: string(EditAnimation), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleObjectPrototype success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(CreateObjectPrototypePayload{AssetName: "Chest"})
		res, err := engine.handleObjectPrototype(context.Background(), &taskdomain.Task{ID: 6, Type: string(GenerateObjectProtoType), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleObjectPrototype(context.Background(), &taskdomain.Task{ID: 6, Type: string(GenerateObjectProtoType), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleScenery success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(CreateSceneryPayload{AssetName: "Scene"})
		res, err := engine.handleScenery(context.Background(), &taskdomain.Task{ID: 7, Type: string(GenerateScenery), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleScenery(context.Background(), &taskdomain.Task{ID: 7, Type: string(GenerateScenery), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleTileSet success, invalid json and validation failure", func(t *testing.T) {
		valid, _ := json.Marshal(CreateTileSetPayload{
			ProjectID:     1,
			AssetName:     "Tileset",
			CreativeBrief: "brief",
			Dimensions: assetdomain.TileSetDimensions{
				TileSize:   assetdomain.Size{Width: 32, Height: 32},
				TileAmount: assetdomain.TileAmount{Columns: 2, Rows: 2},
			},
			Items: []TileSetItemDefinition{
				{Name: "Item", Description: "Desc", Shape: []TileSetCoordinate{{0, 0}}},
			},
		})
		res, err := engine.handleTileSet(context.Background(), &taskdomain.Task{ID: 8, Type: string(GenerateTileSet), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		invalidVal, _ := json.Marshal(CreateTileSetPayload{ProjectID: 0})
		_, err = engine.handleTileSet(context.Background(), &taskdomain.Task{ID: 8, Type: string(GenerateTileSet), Payload: invalidVal})
		if err == nil {
			t.Fatal("expected validation error")
		}

		_, err = engine.handleTileSet(context.Background(), &taskdomain.Task{ID: 8, Type: string(GenerateTileSet), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleAddTilesetItem success, validation failure and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(AddTilesetItemPayload{
			ProjectID:     1,
			AssetID:       10,
			CreativeBrief: "brief",
			Item: &AddTileSetItemDefinition{
				Name:        "Item",
				Description: "Desc",
				Shape:       []TileSetCoordinate{{0, 0}},
			},
		})
		res, err := engine.handleAddTilesetItem(context.Background(), &taskdomain.Task{ID: 9, Type: string(AddTilesetItem), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		invalidVal, _ := json.Marshal(AddTilesetItemPayload{ProjectID: 0})
		_, err = engine.handleAddTilesetItem(context.Background(), &taskdomain.Task{ID: 9, Type: string(AddTilesetItem), Payload: invalidVal})
		if err == nil {
			t.Fatal("expected validation error")
		}

		_, err = engine.handleAddTilesetItem(context.Background(), &taskdomain.Task{ID: 9, Type: string(AddTilesetItem), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditTilesetItem success, validation failure and invalid json", func(t *testing.T) {
		x, y := 0, 0
		valid, _ := json.Marshal(EditTilesetItemPayload{
			ProjectID:     1,
			AssetID:       10,
			CreativeBrief: "brief",
			Target:        &TileSetEditTarget{Position: &TileSetEditPosition{X: &x, Y: &y}},
		})
		res, err := engine.handleEditTilesetItem(context.Background(), &taskdomain.Task{ID: 10, Type: string(EditTilesetItem), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		invalidVal, _ := json.Marshal(EditTilesetItemPayload{ProjectID: 0})
		_, err = engine.handleEditTilesetItem(context.Background(), &taskdomain.Task{ID: 10, Type: string(EditTilesetItem), Payload: invalidVal})
		if err == nil {
			t.Fatal("expected validation error")
		}

		_, err = engine.handleEditTilesetItem(context.Background(), &taskdomain.Task{ID: 10, Type: string(EditTilesetItem), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditTiles success, validation failure and invalid json", func(t *testing.T) {
		x, y := 0, 0
		valid, _ := json.Marshal(EditTilesPayload{
			ProjectID:     1,
			AssetID:       10,
			CreativeBrief: "brief",
			Targets:       []TileSetEditTarget{{Position: &TileSetEditPosition{X: &x, Y: &y}}},
		})
		res, err := engine.handleEditTiles(context.Background(), &taskdomain.Task{ID: 11, Type: string(EditTiles), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		invalidVal, _ := json.Marshal(EditTilesPayload{ProjectID: 0})
		_, err = engine.handleEditTiles(context.Background(), &taskdomain.Task{ID: 11, Type: string(EditTiles), Payload: invalidVal})
		if err == nil {
			t.Fatal("expected validation error")
		}

		_, err = engine.handleEditTiles(context.Background(), &taskdomain.Task{ID: 11, Type: string(EditTiles), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("handleEditFrames success and invalid json", func(t *testing.T) {
		valid, _ := json.Marshal(EditFramesPayload{
			AssetID:     1,
			ProjectID:   2,
			AnimationID: 3,
			FrameIDs:    []uint{1, 2},
			Prompt:      "prompt",
		})
		res, err := engine.handleEditFrames(context.Background(), &taskdomain.Task{ID: 12, Type: string(EditFrames), Payload: valid})
		if err != nil || !bytes.Equal(res.(json.RawMessage), expectedRes) {
			t.Fatalf("unexpected result: res=%v err=%v", res, err)
		}

		_, err = engine.handleEditFrames(context.Background(), &taskdomain.Task{ID: 12, Type: string(EditFrames), Payload: []byte(`{invalid`)})
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("registerTaskHandlers registers all task types", func(t *testing.T) {
		mgr := newMockTaskManager()
		engine.registerTaskHandlers(mgr)
		expectedTypes := TaskTypes()
		for _, tt := range expectedTypes {
			if _, ok := mgr.handlers[string(tt)]; !ok {
				t.Fatalf("expected handler registered for %s", tt)
			}
		}
	})
}
