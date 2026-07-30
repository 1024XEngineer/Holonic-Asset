package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

const (
	defaultRunListLimit = 20
	maxRunListLimit     = 100
)

var (
	ErrInvalidRunListStatus   = errors.New("generation: invalid run list status")
	ErrTaskServiceRequired    = errors.New("generation: task service is required")
	ErrAIServiceRequired      = errors.New("generation: AI service is required")
	ErrGenerationTaskRequired = errors.New("generation: task is required")
	ErrUnsupportedTaskType    = errors.New("generation: unsupported task type")
)

// GenerationAIService owns image generation and the resulting asset creation.
// Payload is the same generation-owned JSON stored on the task record.
type GenerationAIService interface {
	Generate(
		ctx context.Context,
		taskType domain.TaskType,
		payload json.RawMessage,
	) (json.RawMessage, error)
}

// RequestService exposes task-backed generation use cases used by transports.
type RequestService interface {
	Create(ctx context.Context, request *domain.GenerationRequest) (domain.RunID, error)
	List(ctx context.Context, query *domain.RunListQuery) (*domain.RunListPage, error)
	Get(ctx context.Context, runID domain.RunID) (*domain.GenerationRun, error)
	Cancel(ctx context.Context, runID domain.RunID) error
}

// GenerationTaskHandler defines the concrete generation task handlers. Each
// method owns the workflow for one payload type.
type GenerationTaskHandler interface {
	HandleCharacterPrototype(ctx context.Context, message *taskdomain.Task) (any, error)
	HandleCharacterAnimation(ctx context.Context, message *taskdomain.Task) (any, error)
	HandleObjectPrototype(ctx context.Context, message *taskdomain.Task) (any, error)
	HandleObjectAnimation(ctx context.Context, message *taskdomain.Task) (any, error)
	HandleTileSet(ctx context.Context, message *taskdomain.Task) (any, error)
	HandleEmptyGenerationTask(ctx context.Context, message *taskdomain.Task) (any, error)
}

type GenerationService interface {
	RequestService
	GenerationTaskHandler
}

// generationService is the application boundary over the generic task module.
// AI Service performs generation and owns any asset creation required by it.
type generationService struct {
	reader repository.GenerationTaskReader
	tasks  TaskService
	ai     GenerationAIService
}

func NewGenerationService(
	reader repository.GenerationTaskReader,
	tasks TaskService,
	ai GenerationAIService,
) GenerationService {
	return &generationService{
		reader: reader,
		tasks:  tasks,
		ai:     ai,
	}
}

func (s *generationService) Create(
	ctx context.Context,
	request *domain.GenerationRequest,
) (domain.RunID, error) {
	if s.tasks == nil {
		return 0, ErrTaskServiceRequired
	}

	payloadValue, err := buildGenerationTaskPayload(request)
	if err != nil {
		return 0, err
	}

	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return 0, err
	}

	taskID, err := s.tasks.Create(ctx, &taskdomain.Task{
		Type:    string(request.Kind),
		Status:  taskdomain.StatusPending,
		Payload: payload,
	})
	return domain.RunID(taskID), err
}

func buildGenerationTaskPayload(request *domain.GenerationRequest) (any, error) {
	if request == nil {
		return nil, fmt.Errorf("generation: request is required")
	}

	switch request.Kind {
	case domain.GenerateCharacterProtoType:
		payload := domain.CreateCharacterPrototypePayload{}
		if err := decodeGenerationParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		return payload, nil
	case domain.GenerateCharacterAnimation:
		payload := domain.CreateCharacterAnimationPayload{}
		if err := decodeGenerationParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		if payload.ParentID == 0 && request.AssetID != nil {
			payload.ParentID = *request.AssetID
		}
		return payload, nil
	case domain.GenerateObjectProtoType:
		payload := domain.CreateObjectPrototypePayload{}
		if err := decodeGenerationParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		return payload, nil
	case domain.GenerateObjectAnimation:
		payload := domain.CreateObjectAnimationPayload{}
		if err := decodeGenerationParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		if payload.ParentID == 0 && request.AssetID != nil {
			payload.ParentID = *request.AssetID
		}
		return payload, nil
	case domain.GenerateTileSet:
		payload := domain.CreateTileSetPayload{}
		if err := decodeGenerationParameters(request, &payload); err != nil {
			return nil, err
		}
		payload.ProjectID = request.ProjectID
		payload.CreativeBrief = valueOrFallback(payload.CreativeBrief, request.Prompt)
		payload.Reference = valueOrFallback(payload.Reference, firstReference(request.ReferenceMediaIDs))
		if payload.TileNum == 0 {
			payload.TileNum = uint(len(payload.TileDescriptions))
		}
		return payload, nil
	case domain.RegenerateCharacterProtoType,
		domain.RegenerateCharacterAnimation,
		domain.RegenerateCharacterFrames,
		domain.RegenerateObjectProtoType,
		domain.RegenerateObjectAnimation,
		domain.RegenerateObjectFrames,
		domain.RegenerateItem,
		domain.RegenerateTiles:
		return struct{}{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, request.Kind)
	}
}

func decodeGenerationParameters(request *domain.GenerationRequest, payload any) error {
	if len(request.Parameters) == 0 {
		return nil
	}
	if err := json.Unmarshal(request.Parameters, payload); err != nil {
		return fmt.Errorf("generation: decode %s parameters: %w", request.Kind, err)
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

func (s *generationService) List(
	ctx context.Context,
	query *domain.RunListQuery,
) (*domain.RunListPage, error) {
	status := query.Status
	if status == "" {
		status = domain.RunListStatusActive
	}
	if status != domain.RunListStatusActive {
		return nil, ErrInvalidRunListStatus
	}

	limit := query.Limit
	if limit <= 0 {
		limit = defaultRunListLimit
	} else if limit > maxRunListLimit {
		limit = maxRunListLimit
	}

	filter := &domain.RunListFilter{
		ProjectID: query.ProjectID,
		AssetID:   query.AssetID,
		Statuses:  domain.ActiveTaskStatuses(),
		Limit:     limit,
		Cursor:    query.Cursor,
	}
	if query.AssetID == nil {
		filter.IncludeTaskTypes = domain.ProjectLevelTaskTypes()
	} else {
		filter.ExcludeTaskTypes = domain.ProjectLevelTaskTypes()
	}

	// Generation persistence is intentionally deferred while the module remains
	// an architecture skeleton.
	if s.reader == nil {
		return &domain.RunListPage{Runs: []domain.GenerationRun{}}, nil
	}
	return s.reader.ListRuns(ctx, filter)
}

func (s *generationService) Get(ctx context.Context, runID domain.RunID) (*domain.GenerationRun, error) {
	if s.tasks == nil {
		return nil, ErrTaskServiceRequired
	}

	message, err := s.tasks.GetDetail(ctx, uint(runID))
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

	return &domain.GenerationRun{
		ID:        domain.RunID(message.ID),
		ProjectID: scope.ProjectID,
		AssetID:   assetID,
		Kind:      domain.TaskType(message.Type),
		Status:    message.Status,
		Result:    message.Result,
		Error:     message.Error,
	}, nil
}

func (s *generationService) Cancel(ctx context.Context, runID domain.RunID) error {
	if s.tasks == nil {
		return ErrTaskServiceRequired
	}
	return s.tasks.UpdateStatus(ctx, uint(runID), taskdomain.StatusCancelled)
}

func (s *generationService) HandleCharacterPrototype(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := domain.CreateCharacterPrototypePayload{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (s *generationService) HandleCharacterAnimation(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := domain.CreateCharacterAnimationPayload{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (s *generationService) HandleObjectPrototype(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := domain.CreateObjectPrototypePayload{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (s *generationService) HandleObjectAnimation(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := domain.CreateObjectAnimationPayload{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (s *generationService) HandleTileSet(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := domain.CreateTileSetPayload{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func (s *generationService) HandleEmptyGenerationTask(
	_ context.Context,
	message *taskdomain.Task,
) (any, error) {
	payload := struct{}{}
	if err := decodeGenerationTaskPayload(message, &payload); err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // The handler has no business result until its workflow is implemented.
}

func decodeGenerationTaskPayload(message *taskdomain.Task, payload any) error {
	if message == nil {
		return ErrGenerationTaskRequired
	}
	if err := json.Unmarshal(message.Payload, payload); err != nil {
		return fmt.Errorf("generation: decode %s task %d payload: %w", message.Type, message.ID, err)
	}
	return nil
}

type generationTaskHandleFunc func(context.Context, *taskdomain.Task) (any, error)

func adaptGenerationTaskHandler(handle generationTaskHandleFunc) taskdomain.Handler {
	return taskdomain.HandlerFunc(func(ctx context.Context, message *taskdomain.Task) error {
		_, err := handle(ctx, message)
		return err
	})
}

func RegisterGenerationTaskHandlers(
	registry *taskdomain.Registry,
	handlers GenerationTaskHandler,
) {
	registry.Register(string(domain.GenerateCharacterProtoType),
		adaptGenerationTaskHandler(handlers.HandleCharacterPrototype))
	registry.Register(string(domain.GenerateCharacterAnimation),
		adaptGenerationTaskHandler(handlers.HandleCharacterAnimation))
	registry.Register(string(domain.GenerateObjectProtoType),
		adaptGenerationTaskHandler(handlers.HandleObjectPrototype))
	registry.Register(string(domain.GenerateObjectAnimation),
		adaptGenerationTaskHandler(handlers.HandleObjectAnimation))
	registry.Register(string(domain.GenerateTileSet),
		adaptGenerationTaskHandler(handlers.HandleTileSet))

	emptyHandler := adaptGenerationTaskHandler(handlers.HandleEmptyGenerationTask)
	for _, taskType := range []domain.TaskType{
		domain.RegenerateCharacterProtoType,
		domain.RegenerateCharacterAnimation,
		domain.RegenerateCharacterFrames,
		domain.RegenerateObjectProtoType,
		domain.RegenerateObjectAnimation,
		domain.RegenerateObjectFrames,
		domain.RegenerateItem,
		domain.RegenerateTiles,
	} {
		registry.Register(string(taskType), emptyHandler)
	}
}

var _ GenerationService = (*generationService)(nil)
