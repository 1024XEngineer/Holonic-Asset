package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

type UploadRouter interface {
	CreateUploadTarget(
		c context.Context,
		request dto.CreateUploadTargetRequest,
	) (dto.SuccessResponse[dto.UploadTarget], error)
}

type createUploadTargetInput struct {
	Body dto.CreateUploadTargetRequest
}

type createUploadTargetOutput struct {
	Body dto.SuccessResponse[dto.UploadTarget]
}

// RegisterUploadRoutes registers the upload HTTP routes.
func RegisterUploadRoutes(api huma.API, r UploadRouter) {
	huma.Register(api, huma.Operation{
		OperationID: "createUploadTarget",
		Method:      http.MethodPost,
		Path:        "/uploads",
		Summary:     "Create an upload target",
		Tags:        []string{"Uploads"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *createUploadTargetInput) (*createUploadTargetOutput, error) {
		target, err := r.CreateUploadTarget(ctx, input.Body)
		if err != nil {
			return nil, openAPIError(err)
		}
		return &createUploadTargetOutput{Body: target}, nil
	})
}
