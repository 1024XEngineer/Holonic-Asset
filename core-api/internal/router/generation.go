package router

import (
	"context"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
)

type GenerationRouter interface {
	Create(context.Context, dto.CreateGenerationRequest) (dto.SuccessResponse[dto.CreateGenerationResponse], error)
	List(context.Context, dto.ListGenerationRunsRequest) (dto.SuccessResponse[dto.ListGenerationRunsResponse], error)
	Get(context.Context, dto.GetGenerationRequest) (dto.SuccessResponse[dto.GetGenerationResponse], error)
	Cancel(context.Context, dto.CancelGenerationRequest) (dto.SuccessResponse[dto.CancelGenerationResponse], error)
}

type createGenerationInput struct {
	ProjectID uint `path:"project_id" minimum:"1"`
	Body      dto.CreateGenerationRequest
}

type createGenerationOutput struct {
	Body dto.SuccessResponse[dto.CreateGenerationResponse]
}

// optionalUint tracks whether a query parameter was present. Huma otherwise
// binds both an absent unsigned value and an explicit zero to 0.
type optionalUint struct {
	Value uint
	IsSet bool
}

func (value optionalUint) Schema(registry huma.Registry) *huma.Schema {
	return huma.SchemaFromType(registry, reflect.TypeOf(value.Value))
}

func (value *optionalUint) Receiver() reflect.Value {
	return reflect.ValueOf(value).Elem().FieldByName("Value")
}

func (value *optionalUint) OnParamSet(isSet bool, _ any) {
	value.IsSet = isSet
}

type listGenerationRunsInput struct {
	ProjectID uint         `path:"project_id" minimum:"1"`
	AssetID   optionalUint `query:"assetId" minimum:"1"`
	Status    string       `query:"status" enum:"active"`
	Limit     int          `query:"limit" minimum:"1" maximum:"100"`
	Cursor    string       `query:"cursor"`
}

type listGenerationRunsOutput struct {
	Body dto.SuccessResponse[dto.ListGenerationRunsResponse]
}

type getGenerationInput dto.GetGenerationRequest

type getGenerationOutput struct {
	Body dto.SuccessResponse[dto.GetGenerationResponse]
}

type cancelGenerationInput dto.CancelGenerationRequest

type cancelGenerationOutput struct {
	Body dto.SuccessResponse[dto.CancelGenerationResponse]
}

// RegisterGenerationRoutes exposes task-backed generation use cases. AI Service
// remains responsible for generation and any resulting asset creation.
func RegisterGenerationRoutes(api huma.API, r GenerationRouter) {
	huma.Register(api, huma.Operation{
		OperationID: "createGenerationRun",
		Method:      http.MethodPost,
		Path:        "/projects/{project_id}/generation-runs",
		Summary:     "Create a generation run",
		Tags:        []string{"Generation"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *createGenerationInput) (*createGenerationOutput, error) {
		request := input.Body
		request.ProjectID = input.ProjectID
		response, err := r.Create(ctx, request)
		return &createGenerationOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listGenerationRuns",
		Method:      http.MethodGet,
		Path:        "/projects/{project_id}/generation-runs",
		Summary:     "List generation runs",
		Tags:        []string{"Generation"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *listGenerationRunsInput) (*listGenerationRunsOutput, error) {
		request := dto.ListGenerationRunsRequest{
			ProjectID: input.ProjectID,
			Status:    generator.RunListStatus(input.Status),
			Limit:     input.Limit,
			Cursor:    input.Cursor,
		}
		if input.AssetID.IsSet {
			request.AssetID = &input.AssetID.Value
		}
		response, err := r.List(ctx, request)
		return &listGenerationRunsOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getGenerationRun",
		Method:      http.MethodGet,
		Path:        "/generation-runs/{run_id}",
		Summary:     "Get a generation run",
		Tags:        []string{"Generation"},
	}, func(ctx context.Context, input *getGenerationInput) (*getGenerationOutput, error) {
		response, err := r.Get(ctx, dto.GetGenerationRequest(*input))
		return &getGenerationOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "cancelGenerationRun",
		Method:      http.MethodPost,
		Path:        "/generation-runs/{run_id}/cancel",
		Summary:     "Cancel a generation run",
		Tags:        []string{"Generation"},
	}, func(ctx context.Context, input *cancelGenerationInput) (*cancelGenerationOutput, error) {
		response, err := r.Cancel(ctx, dto.CancelGenerationRequest(*input))
		return &cancelGenerationOutput{Body: response}, openAPIError(err)
	})
}
