package generator

import (
	"context"
	"encoding/json"
	"fmt"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

func (e *Engine) handleCharacterPrototype(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateCharacterPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, GenerateCharacterProtoType, message.Payload)
}

func (e *Engine) handleEditCharacterPrototype(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := EditCharacterPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, EditCharacterProtoType, message.Payload)
}

func (e *Engine) handleEditObjectPrototype(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := EditObjectPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, EditObjectProtoType, message.Payload)
}

func (e *Engine) handleAnimation(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateAnimationPayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, GenerateAnimation, message.Payload)
}

func (e *Engine) handleObjectPrototype(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateObjectPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, GenerateObjectProtoType, message.Payload)
}

func (e *Engine) handleScenery(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateSceneryPayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return e.execute(ctx, GenerateScenery, message.Payload)
}

func (e *Engine) handleTileSet(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateTileSetPayload{}
	if err := decodeTileSetTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	if err := validateCreateTileSetPayload(&payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // Execution starts in the Tileset generation phase.
}

func (e *Engine) handleEditTilesetItem(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := EditTilesetItemPayload{}
	if err := decodeTileSetTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	if err := validateEditTilesetItemPayload(&payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // Execution starts in the Item editing phase.
}

func (e *Engine) handleEditTiles(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := EditTilesPayload{}
	if err := decodeTileSetTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	if err := validateEditTilesPayload(&payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // Execution starts in the Tile editing phase.
}

func (e *Engine) handleEmptyTask(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := struct{}{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func decodeTaskPayload(message *taskdomain.Task, payload any) error {
	if message == nil {
		return ErrTaskRequired
	}
	if err := json.Unmarshal(message.Payload, payload); err != nil {
		return fmt.Errorf("generator: decode %s task %d payload: %w", message.Type, message.ID, err)
	}
	return nil
}

func decodeTileSetTaskPayload(message *taskdomain.Task, payload any) error {
	if message == nil {
		return ErrTaskRequired
	}
	request := &Request{Kind: TaskType(message.Type), Parameters: message.Payload}
	if err := decodeStrictParameters(request, payload); err != nil {
		return fmt.Errorf("generator: decode %s task %d payload: %w", message.Type, message.ID, err)
	}
	return nil
}

func (e *Engine) execute(
	ctx context.Context,
	taskType TaskType,
	payload json.RawMessage,
) (any, error) {
	if e.executor == nil {
		return nil, ErrExecutorRequired
	}
	return e.executor.Generate(ctx, taskType, append(json.RawMessage(nil), payload...))
}

func (e *Engine) registerTaskHandlers(manager taskdomain.Manager) {
	manager.Register(string(GenerateCharacterProtoType), taskdomain.HandlerFunc(e.handleCharacterPrototype))
	manager.Register(string(EditCharacterProtoType), taskdomain.HandlerFunc(e.handleEditCharacterPrototype))
	manager.Register(string(EditObjectProtoType), taskdomain.HandlerFunc(e.handleEditObjectPrototype))
	manager.Register(string(GenerateObjectProtoType), taskdomain.HandlerFunc(e.handleObjectPrototype))
	manager.Register(string(GenerateAnimation), taskdomain.HandlerFunc(e.handleAnimation))
	manager.Register(string(GenerateScenery), taskdomain.HandlerFunc(e.handleScenery))
	manager.Register(string(GenerateTileSet), taskdomain.HandlerFunc(e.handleTileSet))
	manager.Register(string(EditTilesetItem), taskdomain.HandlerFunc(e.handleEditTilesetItem))
	manager.Register(string(EditTiles), taskdomain.HandlerFunc(e.handleEditTiles))

	emptyHandler := taskdomain.HandlerFunc(e.handleEmptyTask)
	for _, taskType := range []TaskType{
		EditCharacterFrames,
		EditObjectFrames,
		EditAnimation,
	} {
		manager.Register(string(taskType), emptyHandler)
	}
}
