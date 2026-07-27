package service

import (
	"context"

	"github.com/1024XEngineer/Holonic-Asset/internal/generation/domain"
	"github.com/1024XEngineer/Holonic-Asset/internal/generation/port"
	"github.com/1024XEngineer/Holonic-Asset/internal/generation/repository"
	taskservice "github.com/1024XEngineer/Holonic-Asset/internal/task/service"
)

// RequestService exposes the generation lifecycle use cases used by transports.
type RequestService interface {
	Create(ctx context.Context, request *domain.GenerationRequest) (domain.RunID, error)
	Get(ctx context.Context, runID domain.RunID) (*domain.GenerationDetail, error)
	Cancel(ctx context.Context, runID domain.RunID) error
	ConfirmCandidate(ctx context.Context, runID domain.RunID, candidateID domain.CandidateID) error
}

// PlanningService is called by planning job handlers.
type PlanningService interface {
	Plan(ctx context.Context, runID domain.RunID) (*domain.Plan, error)
}

// StepResultService records generation-owned outputs. Task handlers maintain
// execution status, attempts, retry, and cancellation.
type StepResultService interface {
	RecordResult(ctx context.Context, stepID domain.StepID, result *domain.StepResult) error
}

type GenerationService interface {
	RequestService
	PlanningService
	StepResultService
}

// generationService is the application-service skeleton. Persistence, planning,
// task coordination, and business lifecycle transitions are intentionally deferred.
type generationService struct {
	reader     repository.Reader
	unitOfWork port.UnitOfWork
	tasks      taskservice.TaskService
}

func NewGenerationService(
	reader repository.Reader,
	unitOfWork port.UnitOfWork,
	tasks taskservice.TaskService,
) GenerationService {
	return &generationService{
		reader:     reader,
		unitOfWork: unitOfWork,
		tasks:      tasks,
	}
}

func (*generationService) Create(context.Context, *domain.GenerationRequest) (domain.RunID, error) {
	return 0, nil
}

func (*generationService) Get(context.Context, domain.RunID) (*domain.GenerationDetail, error) {
	return &domain.GenerationDetail{}, nil
}

func (*generationService) Cancel(context.Context, domain.RunID) error {
	return nil
}

func (*generationService) ConfirmCandidate(context.Context, domain.RunID, domain.CandidateID) error {
	return nil
}

func (*generationService) Plan(context.Context, domain.RunID) (*domain.Plan, error) {
	return &domain.Plan{}, nil
}

func (*generationService) RecordResult(context.Context, domain.StepID, *domain.StepResult) error {
	return nil
}

var _ GenerationService = (*generationService)(nil)
