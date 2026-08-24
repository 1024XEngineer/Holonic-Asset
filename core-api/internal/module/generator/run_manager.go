package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
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
	ResolveApplication(ctx context.Context, runID RunID, applied bool) error
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

	task := &taskdomain.Task{
		Type:    string(request.Kind),
		Status:  taskdomain.StatusPending,
		Payload: payload,
	}
	if request.Kind.AwaitsApplication() {
		task.CompletionStatus = taskdomain.StatusAwaitingApplication
	}
	taskID, err := e.tasks.Publish(ctx, task)
	return RunID(taskID), err
}

func (e *Engine) prepareTaskPayload(ctx context.Context, projectID uint, payload any) (any, error) {
	persistReference := func(role, reference string) (string, error) {
		if strings.TrimSpace(reference) == "" {
			return "", nil
		}
		if e.references == nil {
			return "", ErrReferenceStoreRequired
		}
		persisted, err := e.references.PersistReference(ctx, reference)
		if err != nil {
			return "", referencePersistenceError(role, err)
		}
		return persisted, nil
	}
	preparePrototypeReferences := func(
		creatingReference string,
		tags []assetdomain.Tag,
	) (string, string, []string, error) {
		projectReference := ""
		if e.projects != nil && projectID != 0 {
			project, err := e.projects.GetDetail(ctx, projectID)
			if err != nil {
				return "", "", nil, fmt.Errorf("generator: load project %d reference: %w", projectID, err)
			}
			if project != nil {
				projectReference = project.Reference
			}
		}
		var err error
		projectReference, err = persistReference("project", projectReference)
		if err != nil {
			return "", "", nil, err
		}
		creatingReference, err = persistReference("creating", creatingReference)
		if err != nil {
			return "", "", nil, err
		}
		nexusLimit := 3
		if strings.TrimSpace(creatingReference) != "" {
			nexusLimit = 2
		}
		nexusReferences, err := e.selectNexusReferences(
			ctx,
			projectID,
			tags,
			nexusLimit,
			projectReference,
			creatingReference,
		)
		if err != nil {
			return "", "", nil, err
		}
		prepared := make([]string, 0, len(nexusReferences))
		seen := map[string]struct{}{}
		for _, reference := range []string{projectReference, creatingReference} {
			if reference != "" {
				seen[reference] = struct{}{}
			}
		}
		for _, reference := range nexusReferences {
			if len(prepared) == nexusLimit {
				break
			}
			persisted, persistErr := persistReference("nexus", reference)
			if persistErr != nil {
				return "", "", nil, persistErr
			}
			if persisted == "" {
				continue
			}
			if _, duplicate := seen[persisted]; duplicate {
				continue
			}
			seen[persisted] = struct{}{}
			prepared = append(prepared, persisted)
		}
		return projectReference, creatingReference, prepared, nil
	}
	switch value := payload.(type) {
	case CreateCharacterPrototypePayload:
		var err error
		value.ProjectReference, value.CreatingReference, value.NexusReferences, err =
			preparePrototypeReferences(value.CreatingReference, value.Tags)
		return value, err
	case CreateObjectPrototypePayload:
		var err error
		value.ProjectReference, value.CreatingReference, value.NexusReferences, err =
			preparePrototypeReferences(value.CreatingReference, value.Tags)
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
		value.ProjectReference, err = persistReference("project", project.Reference)
		if err != nil {
			return nil, err
		}
		value.CreatingReference, err = persistReference("creating", value.CreatingReference)
		if err != nil {
			return nil, err
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
		value.CreatingReference, err = persistReference("creating", value.CreatingReference)
		return value, err
	case EditTilesPayload:
		var err error
		value.CreatingReference, err = persistReference("creating", value.CreatingReference)
		return value, err
	default:
		return payload, nil
	}
}

type scoredNexusReference struct {
	reference string
	score     int
	version   uint
	assetID   uint
}

func (e *Engine) selectNexusReferences(
	ctx context.Context,
	projectID uint,
	tags []assetdomain.Tag,
	limit int,
	excludedReferences ...string,
) ([]string, error) {
	requested := normalizedTagNameSet(tags)
	if projectID == 0 || len(requested) == 0 || limit <= 0 {
		return nil, nil
	}
	if e.assets == nil {
		return nil, ErrAssetReaderRequired
	}

	assets, err := e.assets.GetAssets(ctx, projectID, assetdomain.AssetListFilter{})
	if err != nil {
		return nil, fmt.Errorf("generator: list project %d assets for Nexus References: %w", projectID, err)
	}
	candidates := make([]scoredNexusReference, 0, len(assets))
	for _, asset := range assets {
		if asset.ProjectID != 0 && asset.ProjectID != projectID {
			continue
		}
		reference := strings.TrimSpace(asset.ThumbnailURL)
		score := tagMatchScore(requested, asset.Tags)
		if reference == "" || score == 0 {
			continue
		}
		candidates = append(candidates, scoredNexusReference{
			reference: reference,
			score:     score,
			version:   asset.Version,
			assetID:   asset.ID,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].version != candidates[j].version {
			return candidates[i].version > candidates[j].version
		}
		return candidates[i].assetID > candidates[j].assetID
	})
	seen := make(map[string]struct{}, len(excludedReferences)+limit)
	for _, reference := range excludedReferences {
		if reference = strings.TrimSpace(reference); reference != "" {
			seen[reference] = struct{}{}
		}
	}
	references := make([]string, 0, min(limit, len(candidates)))
	for _, candidate := range candidates {
		if _, duplicate := seen[candidate.reference]; duplicate {
			continue
		}
		seen[candidate.reference] = struct{}{}
		references = append(references, candidate.reference)
		if len(references) == limit {
			break
		}
	}
	return references, nil
}

func normalizedTagNameSet(tags []assetdomain.Tag) map[string]struct{} {
	names := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if name := strings.ToLower(strings.TrimSpace(tag.Name)); name != "" {
			names[name] = struct{}{}
		}
	}
	return names
}

func tagMatchScore(requested map[string]struct{}, tags []assetdomain.Tag) int {
	matched := make(map[string]struct{})
	for _, tag := range tags {
		name := strings.ToLower(strings.TrimSpace(tag.Name))
		if _, ok := requested[name]; ok {
			matched[name] = struct{}{}
		}
	}
	return len(matched)
}

func referencePersistenceError(role string, err error) error {
	if errors.Is(err, upload.ErrUntrustedReference) ||
		errors.Is(err, upload.ErrInvalidObjectData) ||
		errors.Is(err, upload.ErrInvalidUploadRequest) {
		return invalidTaskPayload(
			"%s reference must be an object key, configured object-storage URL, or image data URL",
			role,
		)
	}
	return fmt.Errorf("generator: persist %s reference: %w", role, err)
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
		payload.ProjectReference = ""
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
		payload.ProjectReference = ""
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = request.CreativeBrief
		return payload, nil
	case GenerateScenery:
		parameters := struct {
			AssetName         string           `json:"asset_name"`
			Dimensions        assetdomain.Size `json:"dimensions"`
			CreatingReference string           `json:"creating_reference"`
		}{}
		if request.AssetID != nil || len(request.TargetAssetPaths) != 0 {
			return nil, fmt.Errorf("%w: generate_scenery does not accept assetId or targetAssetPaths", ErrInvalidSceneryPayload)
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidSceneryPayload, err)
		}
		payload := CreateSceneryPayload{
			AssetName: parameters.AssetName, CreativeBrief: request.CreativeBrief,
			Dimensions: parameters.Dimensions, CreatingReference: parameters.CreatingReference,
			ProjectID: request.ProjectID,
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
			Target            *TileSetEditTarget `json:"target"`
			CreatingReference string             `json:"creating_reference,omitempty"`
		}{}
		if len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("edit_tileset_item does not accept targetAssetPaths")
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesetItemPayload{
			ProjectID:         request.ProjectID,
			CreativeBrief:     request.CreativeBrief,
			Target:            parameters.Target,
			CreatingReference: parameters.CreatingReference,
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
			Targets           []TileSetEditTarget `json:"targets"`
			CreatingReference string              `json:"creating_reference,omitempty"`
		}{}
		if len(request.TargetAssetPaths) != 0 {
			return nil, invalidTaskPayload("edit_tiles does not accept targetAssetPaths")
		}
		if err := decodeStrictParameters(request, &parameters); err != nil {
			return nil, err
		}
		payload := EditTilesPayload{
			ProjectID:         request.ProjectID,
			CreativeBrief:     request.CreativeBrief,
			Targets:           append([]TileSetEditTarget(nil), parameters.Targets...),
			CreatingReference: parameters.CreatingReference,
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

func (e *Engine) ResolveApplication(ctx context.Context, runID RunID, applied bool) error {
	if e.tasks == nil {
		return ErrTaskManagerRequired
	}
	message, err := e.tasks.GetDetail(ctx, uint(runID))
	if err != nil {
		return err
	}
	if message.Status != taskdomain.StatusAwaitingApplication {
		return fmt.Errorf("generator: run %d is not awaiting application", runID)
	}
	if !applied && e.references != nil {
		result := ExecutionResult{}
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return fmt.Errorf("generator: decode run %d result for discard: %w", runID, err)
		}
		if len(result.GeneratedResources) > 0 {
			if err := e.references.DeleteObjects(ctx, result.GeneratedResources); err != nil {
				return fmt.Errorf("generator: discard run %d resources: %w", runID, err)
			}
		}
	}
	return e.tasks.Complete(ctx, uint(runID))
}
