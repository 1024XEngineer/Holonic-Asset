package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
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
	case CreateTileSetPayload, EditTilesetItemPayload, EditTilesPayload:
		return value, nil
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
	case GenerateTileSet:
		parameters := struct {
			AssetName  string                        `json:"asset_name"`
			Dimensions assetdomain.TileSetDimensions `json:"dimensions"`
			Items      []TileSetItemDefinition       `json:"items"`
		}{}
		if request.AssetID != nil || len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("generate_tileset does not accept assetId or targetAssetPaths")
		}
		if err := decodeTileSetParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := CreateTileSetPayload{
			AssetName:     parameters.AssetName,
			ProjectID:     request.ProjectID,
			CreativeBrief: request.CreativeBrief,
			Dimensions:    parameters.Dimensions,
			Items:         parameters.Items,
		}
		if err := validateCreateTileSetPayload(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	case EditTilesetItem:
		parameters := struct{}{}
		if err := decodeTileSetParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesetItemPayload{
			ProjectID:        request.ProjectID,
			CreativeBrief:    request.CreativeBrief,
			TargetAssetPaths: append([]string(nil), request.TargetAssetPaths...),
		}
		if request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		if err := validateEditTilesetItemPayload(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	case EditTiles:
		parameters := struct{}{}
		if err := decodeTileSetParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesPayload{
			ProjectID:        request.ProjectID,
			CreativeBrief:    request.CreativeBrief,
			TargetAssetPaths: append([]string(nil), request.TargetAssetPaths...),
		}
		if request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		if err := validateEditTilesPayload(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	case EditCharacterFrames,
		EditObjectFrames,
		EditAnimation:
		return struct{}{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, request.Kind)
	}
}

func decodeTileSetParameters(request *Request, payload any) error {
	if len(bytes.TrimSpace(request.Parameters)) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(request.Parameters))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		return invalidTaskPayload("decode %s parameters: %v", request.Kind, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return invalidTaskPayload("decode %s parameters: trailing JSON data", request.Kind)
	}
	return nil
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
