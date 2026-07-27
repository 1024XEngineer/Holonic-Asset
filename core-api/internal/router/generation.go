package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type GenerationRouter interface {
	Create(*echox.Context, dto.CreateGenerationRequest) (dto.CreateGenerationResponse, error)
	Get(*echox.Context, dto.GetGenerationRequest) (dto.GetGenerationResponse, error)
	Cancel(*echox.Context, dto.CancelGenerationRequest) (dto.CancelGenerationResponse, error)
	ConfirmCandidate(*echox.Context, dto.ConfirmCandidateRequest) (dto.ConfirmCandidateResponse, error)
}

// RegisterRoutes exposes generation lifecycle use cases. Step execution and AI
// provider operations remain internal application interfaces.
func RegisterGenerationRoutes(e *echo.Group, r GenerationRouter) {
	projects := e.Group("/projects")
	projects.POST("/:project_id/generation-runs", echox.WrapReq(r.Create))

	runs := e.Group("/generation-runs")
	runs.GET("/:run_id", echox.WrapReq(r.Get))
	runs.POST("/:run_id/cancel", echox.WrapReq(r.Cancel))
	runs.POST("/:run_id/candidates/:candidate_id/confirm", echox.WrapReq(r.ConfirmCandidate))
}
