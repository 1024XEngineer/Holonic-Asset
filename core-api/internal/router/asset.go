package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type AssetRouter interface {
	GetAssets(
		*echox.Context,
		dto.GetAssetsRequest,
	) (dto.SuccessResponse[dto.GetAssetsResponse], error)
	Detail(
		*echox.Context,
		dto.AssetDetailRequest,
	) (dto.SuccessResponse[dto.AssetDetailResponse], error)
	Record(
		*echox.Context,
		dto.RecordAssetRequest,
	) (dto.SuccessResponse[dto.RecordAssetResponse], error)
	Records(
		*echox.Context,
		dto.GetAssetRecordsRequest,
	) (dto.SuccessResponse[dto.GetAssetRecordsResponse], error)
	CopyAsset(
		*echox.Context,
		dto.CopyAssetRequest,
	) (dto.SuccessResponse[dto.CopyAssetResponse], error)
	RollBackAsset(
		*echox.Context,
		dto.RollBackAssetRequest,
	) (dto.SuccessResponse[dto.RollBackAssetResponse], error)
	UpdateAsset(
		*echox.Context,
		dto.UpdateAssetRequest,
	) (dto.SuccessResponse[dto.UpdateAssetResponse], error)
}

type listAssetsInput dto.GetAssetsRequest

type listAssetsOutput struct {
	Body dto.SuccessResponse[dto.GetAssetsResponse]
}

type getAssetInput dto.AssetDetailRequest

type getAssetOutput struct {
	Body dto.SuccessResponse[dto.AssetDetailResponse]
}

type listAssetRecordsInput dto.GetAssetRecordsRequest

type listAssetRecordsOutput struct {
	Body dto.SuccessResponse[dto.GetAssetRecordsResponse]
}

type recordAssetInput struct {
	Body dto.RecordAssetRequest
}

type recordAssetOutput struct {
	Body dto.SuccessResponse[dto.RecordAssetResponse]
}

type copyAssetInput struct {
	Body dto.CopyAssetRequest
}

type copyAssetOutput struct {
	Body dto.SuccessResponse[dto.CopyAssetResponse]
}

type rollbackAssetInput struct {
	Body dto.RollBackAssetRequest
}

type rollbackAssetOutput struct {
	Body dto.SuccessResponse[dto.RollBackAssetResponse]
}

type updateAssetInput struct {
	Body dto.UpdateAssetRequest
}

type updateAssetOutput struct {
	Body dto.SuccessResponse[dto.UpdateAssetResponse]
}

// RegisterAssetRoutes registers the asset lifecycle contract.
func RegisterAssetRoutes(api huma.API, r AssetRouter) {
	huma.Register(api, huma.Operation{
		OperationID: "listAssets",
		Method:      http.MethodGet,
		Path:        "/projects/{project_id}/assets",
		Summary:     "List project assets",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *listAssetsInput) (*listAssetsOutput, error) {
		response, err := r.GetAssets(echox.FromContext(ctx), dto.GetAssetsRequest(*input))
		return &listAssetsOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listAssetRecords",
		Method:      http.MethodGet,
		Path:        "/asset/{asset_id}/records",
		Summary:     "List asset records",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *listAssetRecordsInput) (*listAssetRecordsOutput, error) {
		response, err := r.Records(echox.FromContext(ctx), dto.GetAssetRecordsRequest(*input))
		return &listAssetRecordsOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getAsset",
		Method:      http.MethodGet,
		Path:        "/asset/{asset_id}",
		Summary:     "Get an asset",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *getAssetInput) (*getAssetOutput, error) {
		response, err := r.Detail(echox.FromContext(ctx), dto.AssetDetailRequest(*input))
		return &getAssetOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "recordAsset",
		Method:           http.MethodPost,
		Path:             "/asset/save",
		Summary:          "Create an asset record",
		Tags:             []string{"Assets"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *recordAssetInput) (*recordAssetOutput, error) {
		response, err := r.Record(echox.FromContext(ctx), input.Body)
		return &recordAssetOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "copyAsset",
		Method:           http.MethodPost,
		Path:             "/asset/copy",
		Summary:          "Copy an asset",
		Tags:             []string{"Assets"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *copyAssetInput) (*copyAssetOutput, error) {
		response, err := r.CopyAsset(echox.FromContext(ctx), input.Body)
		return &copyAssetOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "rollbackAsset",
		Method:           http.MethodPost,
		Path:             "/asset/rollback",
		Summary:          "Roll back an asset",
		Tags:             []string{"Assets"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *rollbackAssetInput) (*rollbackAssetOutput, error) {
		response, err := r.RollBackAsset(echox.FromContext(ctx), input.Body)
		return &rollbackAssetOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "updateAsset",
		Method:           http.MethodPost,
		Path:             "/asset/update",
		Summary:          "Update an asset",
		Tags:             []string{"Assets"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *updateAssetInput) (*updateAssetOutput, error) {
		response, err := r.UpdateAsset(echox.FromContext(ctx), input.Body)
		return &updateAssetOutput{Body: response}, openAPIError(err)
	})
}
