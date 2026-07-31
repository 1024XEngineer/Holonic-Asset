package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type ProjectRouter interface {
	Create(c *echox.Context, request dto.CreateProjectRequest) (dto.Response, error)
	ListByUID(c *echox.Context, request dto.ListProjectsRequest) (dto.Response, error)
	GetDetail(c *echox.Context, request dto.ProjectDetailRequest) (dto.Response, error)
	Update(c *echox.Context, request dto.UpdateProjectRequest) (dto.Response, error)
	Delete(c *echox.Context, request dto.DeleteProjectRequest) (dto.Response, error)
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
