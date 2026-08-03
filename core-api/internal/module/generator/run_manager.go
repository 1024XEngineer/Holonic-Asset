package generator

import (
	"context"
	"encoding/json"
	"fmt"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

const (
	defaultRunListLimit = 20
	maxRunListLimit     = 100
)

// RunManager exposes task-backed Generator run operations to transports.
type RunManager interface {
	Create(ctx context.Context, request *Request) (RunID, error)
	List(ctx context.Context, query *RunListQuery) (*RunListPage, error)
	Get(ctx context.Context, runID RunID) (*Run, error)
	Cancel(ctx context.Context, runID RunID) error
}

func (e *Engine) Create(ctx context.Context, request *Request) (RunID, error) {
	if e.tasks == nil {
		return 0, ErrTaskManagerRequired
	}

	payloadValue, err := buildTaskPayload(request)
	if err != nil {
		return 0, err
	}

	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return 0, err
	}

	taskID, err := e.tasks.Publish(ctx, &taskdomain.Task{
		Type:    string(request.Kind),
		Status:  taskdomain.StatusPending,
		Payload: payload,
	})
	return RunID(taskID), err
}

func buildTaskPayload(request *Request) (any, error) {
	if request == nil {
		return nil, fmt.Errorf("generator: request is required")
	}

	switch request.Kind {
	case GenerateCharacterProtoType:
		payload := CreateCharacterPrototypePayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		return payload, nil
	case GenerateObjectProtoType:
		payload := CreateObjectPrototypePayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		return payload, nil
	case GenerateAnimation:
		payload := CreateAnimationPayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		if payload.ParentID == 0 && request.AssetID != nil {
			payload.ParentID = *request.AssetID
		}
		return payload, nil
	case GenerateTileSet:
		payload := CreateTileSetPayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		if payload.TileNum == 0 {
			payload.TileNum = uint(len(payload.TileDescriptions))
		}
		return payload, nil
	case EditCharacterProtoType,
		EditCharacterFrames,
		EditObjectProtoType,
		EditObjectFrames,
		EditAnimation,
		EditItem,
		EditTiles:
		return struct{}{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, request.Kind)
	}
}

func decodeParameters(request *Request, payload any) error {
	if len(request.Parameters) == 0 {
		return nil
	}
	if err := json.Unmarshal(request.Parameters, payload); err != nil {
		return fmt.Errorf("generator: decode %s parameters: %w", request.Kind, err)
	}
	return nil
}

func valueOrFallback(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func firstReference(references []string) string {
	if len(references) == 0 {
		return ""
	}
	return references[0]
}

func (e *Engine) List(ctx context.Context, query *RunListQuery) (*RunListPage, error) {
	status := query.Status
	if status == "" {
		status = RunListStatusActive
	}
	if status != RunListStatusActive {
		return nil, ErrInvalidRunListStatus
	}

	limit := query.Limit
	if limit <= 0 {
		limit = defaultRunListLimit
	} else if limit > maxRunListLimit {
		limit = maxRunListLimit
	}

	filter := &RunListFilter{
		ProjectID: query.ProjectID,
		AssetID:   query.AssetID,
		Statuses:  ActiveTaskStatuses(),
		Limit:     limit,
		Cursor:    query.Cursor,
	}
	if query.AssetID == nil {
		filter.IncludeTaskTypes = ProjectLevelTaskTypes()
	} else {
		filter.ExcludeTaskTypes = ProjectLevelTaskTypes()
	}

	if e.reader == nil {
		return &RunListPage{Runs: []Run{}}, nil
	}
	return e.reader.ListRuns(ctx, filter)
}

func (e *Engine) Get(ctx context.Context, runID RunID) (*Run, error) {
	if e.tasks == nil {
		return nil, ErrTaskManagerRequired
	}

	message, err := e.tasks.GetDetail(ctx, uint(runID))
	if err != nil {
		return nil, err
	}

	var scope struct {
		ProjectID uint  `json:"project_id"`
		AssetID   *uint `json:"asset_id"`
		ParentID  *uint `json:"parent_id"`
	}
	if err := json.Unmarshal(message.Payload, &scope); err != nil {
		return nil, err
	}
	assetID := scope.ParentID
	if assetID == nil {
		assetID = scope.AssetID
	}

	return &Run{
		ID:        RunID(message.ID),
		ProjectID: scope.ProjectID,
		AssetID:   assetID,
		Kind:      TaskType(message.Type),
		Status:    message.Status,
		Result:    message.Result,
		Error:     message.Error,
	}, nil
}

func (e *Engine) Cancel(ctx context.Context, runID RunID) error {
	if e.tasks == nil {
		return ErrTaskManagerRequired
	}
	return e.tasks.Cancel(ctx, uint(runID))
}
