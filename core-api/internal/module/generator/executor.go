package generator

import (
	"context"
	"encoding/json"
)

// Executor owns generation and any resulting asset creation.
type Executor interface {
	Generate(ctx context.Context, taskType TaskType, payload json.RawMessage) (json.RawMessage, error)
}
