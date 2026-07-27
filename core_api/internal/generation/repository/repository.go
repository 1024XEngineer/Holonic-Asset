package repository

import (
	"context"

	"github.com/1024XEngineer/Holonic-Asset/internal/generation/domain"
)

// Reader defines generation queries that do not require a write transaction.
type Reader interface {
	GetRun(ctx context.Context, runID domain.RunID) (*domain.GenerationRun, error)
	GetStep(ctx context.Context, stepID domain.StepID) (*domain.Step, error)
	ListSteps(ctx context.Context, runID domain.RunID) ([]domain.Step, error)
	GetCandidate(ctx context.Context, candidateID domain.CandidateID) (*domain.Candidate, error)
	ListCandidates(ctx context.Context, runID domain.RunID) ([]domain.Candidate, error)
}

// Writer persists generation-owned data only. Task state and queue operations
// are handled through the task application service.
type Writer interface {
	CreateRun(ctx context.Context, run *domain.GenerationRun) error
	SetPlanningTask(ctx context.Context, runID domain.RunID, taskID uint) error
	TransitionRun(
		ctx context.Context,
		runID domain.RunID,
		from domain.RunLifecycle,
		to domain.RunLifecycle,
		failure *domain.Failure,
	) (bool, error)

	// CreatePlan persists the Plan and its initial Steps as one aggregate.
	CreatePlan(ctx context.Context, plan *domain.Plan) error
	SetStepTask(ctx context.Context, stepID domain.StepID, taskID uint) error
	SaveStepResult(ctx context.Context, stepID domain.StepID, result *domain.StepResult) error

	CreateCandidate(ctx context.Context, candidate *domain.Candidate) error
	ConfirmCandidate(ctx context.Context, runID domain.RunID, candidateID domain.CandidateID) (bool, error)
}

// Repository is the transaction-bound persistence contract.
type Repository interface {
	Reader
	Writer
}
