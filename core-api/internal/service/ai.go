// Package service defines the internal application capabilities of the AI module.
// These interfaces are called directly by generation and job handlers; the AI
// module intentionally has no HTTP handler or router.
package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/ai"
)

// PlanProposer turns a high-level instruction into a constrained plan proposal.
// The generation module remains responsible for the authoritative Plan lifecycle.
type PlanProposer interface {
	Propose(ctx context.Context, request *domain.PlanRequest) (*domain.PlanProposal, error)
}

type PromptBuilder interface {
	Build(ctx context.Context, request *domain.PromptRequest) (*domain.Prompt, error)
}

type PlanValidator interface {
	Validate(ctx context.Context, proposal *domain.PlanProposal, constraints domain.PlanConstraints) error
}

type ImageService interface {
	Generate(ctx context.Context, request *domain.GenerateImageRequest) (*domain.ImageResult, error)
	Edit(ctx context.Context, request *domain.EditImageRequest) (*domain.ImageResult, error)
	RemoveBackground(ctx context.Context, request *domain.EditImageRequest) (*domain.ImageResult, error)
	Segment(ctx context.Context, request *domain.EditImageRequest) (*domain.ImageResult, error)
	GenerateMask(ctx context.Context, request *domain.EditImageRequest) (*domain.ImageResult, error)
}

type AudioService interface {
	Generate(ctx context.Context, request *domain.GenerateAudioRequest) (*domain.AudioResult, error)
	Edit(ctx context.Context, request *domain.EditAudioRequest) (*domain.AudioResult, error)
}

type ModelRouter interface {
	Select(ctx context.Context, capability domain.ModelCapability) (*domain.ModelSelection, error)
}

type MeteringService interface {
	Collect(ctx context.Context, usage domain.Usage) error
}
