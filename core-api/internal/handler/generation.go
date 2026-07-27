package handler

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type GenerationHandler struct {
	service service.RequestService
}

func NewGenerationHandler(generationService service.RequestService) *GenerationHandler {
	return &GenerationHandler{service: generationService}
}

func (h *GenerationHandler) Create(
	ctx *echox.Context,
	request dto.CreateGenerationRequest,
) (dto.CreateGenerationResponse, error) {
	runID, err := h.service.Create(ctx, &domain.GenerationRequest{
		ProjectID:              request.ProjectID,
		AssetID:                request.AssetID,
		Kind:                   request.Kind,
		Prompt:                 request.Prompt,
		ReferenceMediaIDs:      request.ReferenceMediaIDs,
		TargetAssetResourceIDs: request.TargetAssetResourceIDs,
		Parameters:             request.Parameters,
	})
	return dto.CreateGenerationResponse{GenerationRunID: runID}, err
}

func (h *GenerationHandler) Get(
	ctx *echox.Context,
	request dto.GetGenerationRequest,
) (dto.GetGenerationResponse, error) {
	detail, err := h.service.Get(ctx, request.GenerationRunID)
	if err != nil {
		return dto.GetGenerationResponse{}, err
	}

	steps := make([]dto.StepResponse, len(detail.Steps))
	for i := range detail.Steps {
		stepDetail := detail.Steps[i]
		step := stepDetail.Step
		steps[i] = dto.StepResponse{
			ID:           step.ID,
			Type:         step.Type,
			Executor:     step.Executor,
			Dependencies: step.Dependencies,
			TaskStatus:   stepDetail.TaskStatus,
		}
	}

	candidates := make([]dto.CandidateResponse, len(detail.Candidates))
	for i := range detail.Candidates {
		candidate := detail.Candidates[i]
		candidates[i] = dto.CandidateResponse{
			ID:       candidate.ID,
			MediaIDs: candidate.MediaIDs,
		}
	}

	return dto.GetGenerationResponse{
		ID:                   detail.Run.ID,
		ProjectID:            detail.Run.ProjectID,
		Kind:                 detail.Run.Request.Kind,
		Lifecycle:            detail.Run.Lifecycle,
		ConfirmedCandidateID: detail.Run.ConfirmedCandidateID,
		Steps:                steps,
		Candidates:           candidates,
	}, nil
}

func (h *GenerationHandler) Cancel(
	ctx *echox.Context,
	request dto.CancelGenerationRequest,
) (dto.CancelGenerationResponse, error) {
	err := h.service.Cancel(ctx, request.GenerationRunID)
	return dto.CancelGenerationResponse{Cancelled: err == nil}, err
}

func (h *GenerationHandler) ConfirmCandidate(
	ctx *echox.Context,
	request dto.ConfirmCandidateRequest,
) (dto.ConfirmCandidateResponse, error) {
	err := h.service.ConfirmCandidate(ctx, request.GenerationRunID, request.CandidateID)
	return dto.ConfirmCandidateResponse{Confirmed: err == nil}, err
}
