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

func (e *Engine) handleTileSet(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateTileSetPayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
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
	manager.Register(string(EditObjectProtoType), taskdomain.HandlerFunc(e.handleEditObjectPrototype))
	manager.Register(string(GenerateObjectProtoType), taskdomain.HandlerFunc(e.handleObjectPrototype))
	manager.Register(string(GenerateAnimation), taskdomain.HandlerFunc(e.handleAnimation))
	manager.Register(string(GenerateTileSet), taskdomain.HandlerFunc(e.handleTileSet))

	emptyHandler := taskdomain.HandlerFunc(e.handleEmptyTask)
	for _, taskType := range []TaskType{
		EditCharacterProtoType,
		EditCharacterFrames,
		EditObjectFrames,
		EditAnimation,
		EditTilesetItem,
		EditTiles,
	} {
		manager.Register(string(taskType), emptyHandler)
	}
}
