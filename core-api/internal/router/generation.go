package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type GenerationRouter interface {
	Create(*echox.Context, dto.CreateGenerationRequest) (dto.CreateGenerationResponse, error)
	List(*echox.Context, dto.ListGenerationRunsRequest) (dto.ListGenerationRunsResponse, error)
	Get(*echox.Context, dto.GetGenerationRequest) (dto.GetGenerationResponse, error)
	Cancel(*echox.Context, dto.CancelGenerationRequest) (dto.CancelGenerationResponse, error)
}

// RegisterGenerationRoutes exposes task-backed generation use cases. AI Service
// remains responsible for generation and any resulting asset creation.
func RegisterGenerationRoutes(e *echo.Group, r GenerationRouter) {
	projects := e.Group("/projects")
	projects.POST("/:project_id/generation-runs", echox.WrapReq(r.Create))
	projects.GET("/:project_id/generation-runs", echox.WrapReq(r.List))

	runs := e.Group("/generation-runs")
	runs.GET("/:run_id", echox.WrapReq(r.Get))
	runs.POST("/:run_id/cancel", echox.WrapReq(r.Cancel))
}
