package generator

import (
	"context"
	"encoding/json"
)

// Executor executes queued generation workflows.
type Executor interface {
	Generate(context.Context, TaskType, json.RawMessage) (json.RawMessage, error)
}

// ReferenceStore is the object-storage boundary shared by run preparation and
// generation execution.
type ReferenceStore interface {
	ResolveReference(context.Context, string) (string, error)
	PersistReference(context.Context, string) (string, error)
	NewObjectKey(string) (string, error)
	PersistReferenceAt(context.Context, string, string) error
}

type ExecutionResult struct {
	AssetID     uint `json:"asset_id"`
	AnimationID uint `json:"animation_id,omitempty"`
	Version     uint `json:"version,omitempty"`
}
