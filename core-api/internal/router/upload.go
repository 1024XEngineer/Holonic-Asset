package router

import (
	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type UploadRouter interface {
	CreateUploadTarget(
		c *echox.Context,
		request dto.CreateUploadTargetRequest,
	) (*dto.UploadTarget, error)
}

// RegisterUploadRoutes registers the upload HTTP routes.
func RegisterUploadRoutes(e *echo.Group, r UploadRouter) {
	e.POST("/uploads", echox.WrapReq(r.CreateUploadTarget))
}
