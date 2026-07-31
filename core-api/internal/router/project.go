package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
)

type ProjectRouter interface {
	Create(
		c *echox.Context,
		request dto.CreateProjectRequest,
	) (dto.SuccessResponse[dto.CreateProjectResponse], error)
	ListByUID(
		c *echox.Context,
		request dto.ListProjectsRequest,
	) (dto.SuccessResponse[dto.ListProjectsResponse], error)
	GetDetail(
		c *echox.Context,
		request dto.ProjectDetailRequest,
	) (dto.SuccessResponse[dto.ProjectDetailResponse], error)
	Update(
		c *echox.Context,
		request dto.UpdateProjectRequest,
	) (dto.SuccessResponse[dto.UpdateProjectResponse], error)
	Delete(
		c *echox.Context,
		request dto.DeleteProjectRequest,
	) (dto.SuccessResponse[dto.DeleteProjectResponse], error)
}

type createProjectInput struct {
	Body dto.CreateProjectRequest
}

type createProjectOutput struct {
	Body dto.SuccessResponse[dto.CreateProjectResponse]
}

type listProjectsInput dto.ListProjectsRequest

type listProjectsOutput struct {
	Body dto.SuccessResponse[dto.ListProjectsResponse]
}

type projectDetailInput dto.ProjectDetailRequest

type projectDetailOutput struct {
	Body dto.SuccessResponse[dto.ProjectDetailResponse]
}

type updateProjectInput struct {
	Body dto.UpdateProjectRequest
}

type updateProjectOutput struct {
	Body dto.SuccessResponse[dto.UpdateProjectResponse]
}

type deleteProjectInput struct {
	Body dto.DeleteProjectRequest
}

type deleteProjectOutput struct {
	Body dto.SuccessResponse[dto.DeleteProjectResponse]
}

// RegisterProjectRoutes registers the project HTTP contract.
func RegisterProjectRoutes(api huma.API, r ProjectRouter) {
	huma.Register(api, huma.Operation{
		OperationID:      "createProject",
		Method:           http.MethodPost,
		Path:             "/project/create",
		Summary:          "Create a project",
		Tags:             []string{"Projects"},
		Errors:           []int{http.StatusBadRequest},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *createProjectInput) (*createProjectOutput, error) {
		response, err := r.Create(echox.FromContext(ctx), input.Body)
		return &createProjectOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:        "listProjects",
		Method:             http.MethodGet,
		Path:               "/project/list",
		Summary:            "List projects",
		Tags:               []string{"Projects"},
		Errors:             []int{http.StatusBadRequest},
		SkipValidateParams: true,
	}, func(ctx context.Context, input *listProjectsInput) (*listProjectsOutput, error) {
		response, err := r.ListByUID(echox.FromContext(ctx), dto.ListProjectsRequest(*input))
		return &listProjectsOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:        "getProject",
		Method:             http.MethodGet,
		Path:               "/project/detail",
		Summary:            "Get a project",
		Tags:               []string{"Projects"},
		Errors:             []int{http.StatusBadRequest, http.StatusNotFound},
		SkipValidateParams: true,
	}, func(ctx context.Context, input *projectDetailInput) (*projectDetailOutput, error) {
		response, err := r.GetDetail(echox.FromContext(ctx), dto.ProjectDetailRequest(*input))
		return &projectDetailOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "updateProject",
		Method:           http.MethodPost,
		Path:             "/project/update",
		Summary:          "Update a project",
		Tags:             []string{"Projects"},
		Errors:           []int{http.StatusBadRequest, http.StatusNotFound},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *updateProjectInput) (*updateProjectOutput, error) {
		response, err := r.Update(echox.FromContext(ctx), input.Body)
		return &updateProjectOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID:      "deleteProject",
		Method:           http.MethodPost,
		Path:             "/project/delete",
		Summary:          "Delete a project",
		Tags:             []string{"Projects"},
		Errors:           []int{http.StatusBadRequest, http.StatusNotFound},
		SkipValidateBody: true,
	}, func(ctx context.Context, input *deleteProjectInput) (*deleteProjectOutput, error) {
		response, err := r.Delete(echox.FromContext(ctx), input.Body)
		return &deleteProjectOutput{Body: response}, openAPIError(err)
	})
}
