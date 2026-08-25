package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

type ExportRouter interface {
	Create(context.Context, dto.CreateAssetExportRequest) (dto.SuccessResponse[dto.CreateAssetExportResponse], error)
	Get(context.Context, uint) (dto.SuccessResponse[dto.AssetExportResponse], error)
}

type createExportInput struct{ Body dto.CreateAssetExportRequest }
type createExportOutput struct {
	Body dto.SuccessResponse[dto.CreateAssetExportResponse]
}
type getExportInput struct {
	ExportID uint `param:"export_id" path:"export_id" minimum:"1"`
}
type getExportOutput struct {
	Body dto.SuccessResponse[dto.AssetExportResponse]
}

func RegisterExportRoutes(api huma.API, r ExportRouter) {
	huma.Register(api, huma.Operation{OperationID: "createAssetExport", Method: http.MethodPost, Path: "/asset/export", Summary: "Export an asset", Tags: []string{"Exports"}, Errors: []int{http.StatusBadRequest, http.StatusNotFound}}, func(ctx context.Context, input *createExportInput) (*createExportOutput, error) {
		response, err := r.Create(ctx, input.Body)
		return &createExportOutput{Body: response}, openAPIError(err)
	})
	huma.Register(api, huma.Operation{OperationID: "getAssetExport", Method: http.MethodGet, Path: "/export/{export_id}", Summary: "Get an asset export", Tags: []string{"Exports"}, Errors: []int{http.StatusBadRequest, http.StatusNotFound}}, func(ctx context.Context, input *getExportInput) (*getExportOutput, error) {
		response, err := r.Get(ctx, input.ExportID)
		return &getExportOutput{Body: response}, openAPIError(err)
	})
}
