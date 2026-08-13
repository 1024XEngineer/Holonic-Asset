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
	payloadValue, err = e.prepareTaskPayload(ctx, request.ProjectID, payloadValue)
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

func (e *Engine) prepareTaskPayload(ctx context.Context, projectID uint, payload any) (any, error) {
	prepare := func(reference string) (string, error) {
		if reference == "" && e.projects != nil && projectID != 0 {
			project, err := e.projects.GetDetail(ctx, projectID)
			if err != nil {
				return "", fmt.Errorf("generator: load project %d reference: %w", projectID, err)
			}
			if project != nil {
				reference = project.Reference
			}
		}
		if e.references == nil || reference == "" {
			return reference, nil
		}
		resolved, err := e.references.PersistReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("generator: persist reference: %w", err)
		}
		return resolved, nil
	}

	switch value := payload.(type) {
	case CreateCharacterPrototypePayload:
		var err error
		value.Reference, err = prepare(value.Reference)
		return value, err
	case CreateObjectPrototypePayload:
		var err error
		value.Reference, err = prepare(value.Reference)
		return value, err
	case CreateTileSetPayload:
		var err error
		value.Reference, err = prepare(value.Reference)
		return value, err
	default:
		return payload, nil
	}
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
		payload.CreativeBrief = request.CreativeBrief
		return payload, nil
	case EditCharacterProtoType:
		if request.AssetID == nil || *request.AssetID == 0 {
			return nil, fmt.Errorf("generator: asset id is required for %s", request.Kind)
		}
		return EditCharacterPrototypePayload{
			AssetID:          *request.AssetID,
			ProjectID:        request.ProjectID,
			EditInstructions: request.CreativeBrief,
		}, nil
	case EditObjectProtoType:
		if request.AssetID == nil || *request.AssetID == 0 {
			return nil, fmt.Errorf("generator: asset id is required for %s", request.Kind)
		}
		return EditObjectPrototypePayload{
			AssetID:          *request.AssetID,
			ProjectID:        request.ProjectID,
			EditInstructions: request.CreativeBrief,
		}, nil
	case GenerateObjectProtoType:
		payload := CreateObjectPrototypePayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = request.CreativeBrief
		return payload, nil
	case GenerateAnimation:
		payload := CreateAnimationPayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = request.CreativeBrief
		if payload.AssetID == 0 && request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		return payload, nil
	case EditAnimation:
		payload := EditAnimationPayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		if request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		if payload.AssetID == 0 {
			return nil, fmt.Errorf("generator: asset id is required for %s", request.Kind)
		}
		if payload.AnimationID == 0 {
			return nil, fmt.Errorf("generator: animation id is required for %s", request.Kind)
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = request.CreativeBrief
		return payload, nil
	case GenerateTileSet:
		payload := CreateTileSetPayload{}
		if err := decodeParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = request.CreativeBrief
		if payload.TileNum == 0 {
			payload.TileNum = uint(len(payload.TileDescriptions))
		}
		return payload, nil
	case EditCharacterFrames,
		EditObjectFrames,
		EditTilesetItem,
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

	run, err := taskToRun(message)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (e *Engine) Cancel(ctx context.Context, runID RunID) error {
	if e.tasks == nil {
		return ErrTaskManagerRequired
	}
	return e.tasks.Cancel(ctx, uint(runID))
}
