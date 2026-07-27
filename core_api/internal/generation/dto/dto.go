package dto

import (
	"encoding/json"

	"github.com/1024XEngineer/Holonic-Asset/internal/generation/domain"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/task/domain"
)

type CreateGenerationRequest struct {
	ProjectID              uint               `param:"project_id" json:"-"`
	AssetID                uint               `json:"assetId,omitempty"`
	Kind                   domain.RequestKind `json:"kind"`
	Prompt                 string             `json:"prompt"`
	ReferenceMediaIDs      []string           `json:"referenceMediaIds,omitempty"`
	TargetAssetResourceIDs []uint             `json:"targetAssetResourceIds,omitempty"`
	Parameters             json.RawMessage    `json:"parameters,omitempty"`
}

type CreateGenerationResponse struct {
	GenerationRunID domain.RunID `json:"generationRunId"`
}

type GetGenerationRequest struct {
	GenerationRunID domain.RunID `param:"run_id" json:"-"`
}

type StepResponse struct {
	ID           domain.StepID       `json:"id"`
	Type         string              `json:"type"`
	Executor     domain.StepExecutor `json:"executor"`
	Dependencies []domain.StepID     `json:"dependencies"`
	TaskStatus   *taskdomain.Status  `json:"taskStatus,omitempty"`
}

type CandidateResponse struct {
	ID       domain.CandidateID `json:"id"`
	MediaIDs []string           `json:"mediaIds"`
}

type GetGenerationResponse struct {
	ID                   domain.RunID        `json:"id"`
	ProjectID            uint                `json:"projectId"`
	Kind                 domain.RequestKind  `json:"kind"`
	Lifecycle            domain.RunLifecycle `json:"lifecycle"`
	ConfirmedCandidateID *domain.CandidateID `json:"confirmedCandidateId,omitempty"`
	Steps                []StepResponse      `json:"steps"`
	Candidates           []CandidateResponse `json:"candidates"`
}

type CancelGenerationRequest struct {
	GenerationRunID domain.RunID `param:"run_id" json:"-"`
}

type CancelGenerationResponse struct {
	Cancelled bool `json:"cancelled"`
}

type ConfirmCandidateRequest struct {
	GenerationRunID domain.RunID       `param:"run_id" json:"-"`
	CandidateID     domain.CandidateID `param:"candidate_id" json:"-"`
}

type ConfirmCandidateResponse struct {
	Confirmed bool `json:"confirmed"`
}
