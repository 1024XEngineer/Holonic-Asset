package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
	persistEditReference := func(reference string) (string, error) {
		if reference == "" || e.references == nil {
			return reference, nil
		}
		persisted, err := e.references.PersistReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("generator: persist edit reference: %w", err)
		}
		return persisted, nil
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
	case CreateSceneryPayload:
		if e.projects == nil {
			return nil, ErrProjectReaderRequired
		}
		project, err := e.projects.GetDetail(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("generator: load project %d context: %w", projectID, err)
		}
		if project == nil {
			return nil, fmt.Errorf("generator: load project %d context: empty result", projectID)
		}
		value.Perspective = string(project.Perspective)
		value.ProjectContext = SceneryProjectContext{
			Name: strings.TrimSpace(project.Name), GameType: strings.TrimSpace(string(project.GameType)),
			TargetPlatform: strings.TrimSpace(string(project.TargetPlatform)), Description: strings.TrimSpace(project.Description),
		}
		if strings.TrimSpace(value.Style) == "" {
			value.Style = strings.TrimSpace(project.Style)
		}
		if strings.TrimSpace(value.Reference) == "" {
			value.Reference = project.Reference
		}
		if e.references != nil && strings.TrimSpace(value.Reference) != "" {
			value.Reference, err = e.references.PersistReference(ctx, value.Reference)
			if err != nil {
				return nil, fmt.Errorf("generator: persist reference: %w", err)
			}
		}
		if err := validateSceneryPayload(value); err != nil {
			return nil, err
		}
		return value, nil
	case CreateAnimationPayload:
		if e.projects == nil || projectID == 0 {
			return value, nil
		}
		project, err := e.projects.GetDetail(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("generator: load project %d style: %w", projectID, err)
		}
		if project != nil {
			value.Style = project.Style
		}
		return value, nil
	case CreateTileSetPayload:
		return value, nil
	case EditTilesetItemPayload:
		var err error
		value.Reference, err = persistEditReference(value.Reference)
		return value, err
	case EditTilesPayload:
		var err error
		value.Reference, err = persistEditReference(value.Reference)
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
	case GenerateScenery:
		parameters := struct {
			AssetName  string           `json:"asset_name"`
			Style      string           `json:"style"`
			Dimensions assetdomain.Size `json:"dimensions"`
			Reference  string           `json:"reference"`
		}{}
		if request.AssetID != nil || len(request.TargetAssetPaths) != 0 {
			return nil, fmt.Errorf("%w: generate_scenery does not accept assetId or targetAssetPaths", ErrInvalidSceneryPayload)
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidSceneryPayload, err)
		}
		payload := CreateSceneryPayload{
			AssetName: parameters.AssetName, CreativeBrief: request.CreativeBrief,
			Style: parameters.Style, Dimensions: parameters.Dimensions,
			Reference: parameters.Reference, ProjectID: request.ProjectID,
		}
		if payload.ProjectID == 0 || strings.TrimSpace(payload.AssetName) == "" || strings.TrimSpace(payload.CreativeBrief) == "" {
			return nil, fmt.Errorf("%w: project ID, asset name, and creative brief are required", ErrInvalidSceneryPayload)
		}
		return payload, nil
	case GenerateAnimation:
		parameters := struct {
			AnimationName string `json:"animation_name"`
			Direction     string `json:"direction"`
			FrameCount    int    `json:"frame_count,omitempty"`
			FPS           int    `json:"fps,omitempty"`
			Resolution    string `json:"resolution,omitempty"`
			Duration      int    `json:"duration,omitempty"`
		}{}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := CreateAnimationPayload{
			AnimationName: parameters.AnimationName,
			ProjectID:     request.ProjectID,
			Direction:     parameters.Direction,
			CreativeBrief: request.CreativeBrief,
			FrameCount:    parameters.FrameCount,
			FPS:           parameters.FPS,
			Resolution:    parameters.Resolution,
			Duration:      parameters.Duration,
		}
		if request.AssetID != nil {
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
		parameters := struct {
			AssetName  string                        `json:"asset_name"`
			Dimensions assetdomain.TileSetDimensions `json:"dimensions"`
			Items      []TileSetItemDefinition       `json:"items"`
		}{}
		if request.AssetID != nil || len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("generate_tileset does not accept assetId or targetAssetPaths")
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
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
		parameters := struct {
			Target    *TileSetEditTarget `json:"target"`
			Reference string             `json:"reference,omitempty"`
		}{}
		if len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("edit_tileset_item does not accept targetAssetPaths")
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesetItemPayload{
			ProjectID:     request.ProjectID,
			CreativeBrief: request.CreativeBrief,
			Target:        parameters.Target,
			Reference:     parameters.Reference,
		}
		if request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		if err := validateEditTilesetItemPayload(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	case EditTiles:
		parameters := struct {
			Targets   []TileSetEditTarget `json:"targets"`
			Reference string              `json:"reference,omitempty"`
		}{}
		if len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("edit_tiles does not accept targetAssetPaths")
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesPayload{
			ProjectID:     request.ProjectID,
			CreativeBrief: request.CreativeBrief,
			Targets:       append([]TileSetEditTarget(nil), parameters.Targets...),
			Reference:     parameters.Reference,
		}
		if request.AssetID != nil {
			payload.AssetID = *request.AssetID
		}
		if err := validateEditTilesPayload(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	case EditFrames:
		if request.AssetID == nil || *request.AssetID == 0 {
			return nil, fmt.Errorf("generator: asset id is required for %s", request.Kind)
		}
		parameters := EditFramesParameters{}
		if err := decodeParameters(request, &parameters); err != nil {
			return nil, err
		}
		prompt := strings.TrimSpace(request.CreativeBrief)
		payload := EditFramesPayload{
			AssetID:     *request.AssetID,
			ProjectID:   request.ProjectID,
			AnimationID: parameters.AnimationID,
			FrameIDs:    append([]uint(nil), parameters.FrameIDs...),
			Prompt:      prompt,
		}
		if len(payload.FrameIDs) == 0 {
			return nil, fmt.Errorf("generator: frame ids are required for %s", request.Kind)
		}
		if payload.AnimationID == 0 {
			return nil, fmt.Errorf("generator: animation id is required for %s", request.Kind)
		}
		if strings.TrimSpace(payload.Prompt) == "" {
			return nil, fmt.Errorf("generator: creative brief is required for %s", request.Kind)
		}
		return payload, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, request.Kind)
	}
}

func decodeStrictParameters(request *Request, payload any) error {
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

func validateSceneryPayload(payload CreateSceneryPayload) error {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", ErrInvalidSceneryPayload, reason) }
	if payload.ProjectID == 0 || strings.TrimSpace(payload.AssetName) == "" || strings.TrimSpace(payload.CreativeBrief) == "" {
		return invalid("project ID, asset name, and creative brief are required")
	}
	dimensions, err := json.Marshal(payload.Dimensions)
	if err != nil {
		return invalid("dimensions are invalid")
	}
	if err := assetdomain.ValidateDimensions(assetdomain.AssetTypeScenery, dimensions); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSceneryPayload, err)
	}
	if !assetdomain.Perspective(payload.Perspective).Valid() {
		return invalid("project perspective is invalid")
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
