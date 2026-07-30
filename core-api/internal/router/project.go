package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type ProjectRouter interface {
	Create(c *echox.Context, request dto.CreateProjectRequest) (dto.CreateProjectResponse, error)
	ListByUID(c *echox.Context, request dto.ListProjectsRequest) (dto.ListProjectsResponse, error)
	GetDetail(c *echox.Context, request dto.ProjectDetailRequest) (dto.ProjectDetailResponse, error)
	Update(c *echox.Context, request dto.UpdateProjectRequest) (dto.UpdateProjectResponse, error)
	Delete(c *echox.Context, request dto.DeleteProjectRequest) (dto.DeleteProjectResponse, error)
}

// RegisterRoutes registers all project HTTP routes.
func RegisterProjectRoutes(e *echo.Group, r ProjectRouter) {
	project := e.Group("/project")
	project.POST("/create", echox.WrapReq(r.Create))
	project.GET("/list", echox.WrapReq(r.ListByUID))
	project.GET("/detail", echox.WrapReq(r.GetDetail))
	project.POST("/update", echox.WrapReq(r.Update))
	project.POST("/delete", echox.WrapReq(r.Delete))
}
