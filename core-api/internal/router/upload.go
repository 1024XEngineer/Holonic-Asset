package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type UploadRouter interface {
	CreateUploadTarget(
		c *echox.Context,
		request dto.CreateUploadTargetRequest,
	) (*dto.UploadTarget, error)
}

type createUploadTargetInput struct {
	Body dto.CreateUploadTargetRequest
}

type createUploadTargetOutput struct {
	Body dto.UploadTarget
}

// RegisterUploadRoutes registers the upload HTTP routes.
func RegisterUploadRoutes(api huma.API, r UploadRouter) {
	huma.Register(api, huma.Operation{
		OperationID:      "createUploadTarget",
		Method:           http.MethodPost,
		Path:             "/uploads",
		Summary:          "Create an upload target",
		Tags:             []string{"Uploads"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *createUploadTargetInput) (*createUploadTargetOutput, error) {
		target, err := r.CreateUploadTarget(echox.FromContext(ctx), input.Body)
		if err != nil {
			return nil, openAPIError(err)
		}
		if target == nil {
			return &createUploadTargetOutput{}, nil
		}
		return &createUploadTargetOutput{Body: *target}, nil
	})
}
