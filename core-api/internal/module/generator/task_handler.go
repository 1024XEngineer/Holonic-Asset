package generator

import (
	"context"
	"encoding/json"
	"fmt"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

func (e *Engine) handleCharacterPrototype(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateCharacterPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (e *Engine) handleCharacterAnimation(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateCharacterAnimationPayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (e *Engine) handleObjectPrototype(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateObjectPrototypePayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (e *Engine) handleObjectAnimation(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := CreateObjectAnimationPayload{}
	if err := decodeTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
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

func (e *Engine) registerTaskHandlers(queue taskdomain.Queue) {
	queue.Register(string(GenerateCharacterProtoType), taskdomain.HandlerFunc(e.handleCharacterPrototype))
	queue.Register(string(GenerateCharacterAnimation), taskdomain.HandlerFunc(e.handleCharacterAnimation))
	queue.Register(string(GenerateObjectProtoType), taskdomain.HandlerFunc(e.handleObjectPrototype))
	queue.Register(string(GenerateObjectAnimation), taskdomain.HandlerFunc(e.handleObjectAnimation))
	queue.Register(string(GenerateTileSet), taskdomain.HandlerFunc(e.handleTileSet))

	emptyHandler := taskdomain.HandlerFunc(e.handleEmptyTask)
	for _, taskType := range []TaskType{
		RegenerateCharacterProtoType,
		RegenerateCharacterAnimation,
		RegenerateCharacterFrames,
		RegenerateObjectProtoType,
		RegenerateObjectAnimation,
		RegenerateObjectFrames,
		RegenerateItem,
		RegenerateTiles,
	} {
		queue.Register(string(taskType), emptyHandler)
	}
}
