package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

type ProjectTagRouter interface {
	CreateProjectTag(
		context.Context,
		dto.CreateProjectTagRequest,
	) (dto.SuccessResponse[dto.CreateProjectTagResponse], error)
	ListProjectTags(
		context.Context,
		dto.ListProjectTagsRequest,
	) (dto.SuccessResponse[dto.ListProjectTagsResponse], error)
	GetProjectTag(
		context.Context,
		dto.ProjectTagDetailRequest,
	) (dto.SuccessResponse[dto.ProjectTagDetailResponse], error)
	UpdateProjectTag(
		context.Context,
		dto.UpdateProjectTagRequest,
	) (dto.SuccessResponse[dto.UpdateProjectTagResponse], error)
	DeleteProjectTag(
		context.Context,
		dto.DeleteProjectTagRequest,
	) (dto.SuccessResponse[dto.DeleteProjectTagResponse], error)
}

type createProjectTagInput struct {
	ProjectID uint `path:"project_id" minimum:"1"`
	Body      dto.CreateProjectTagRequest
}

type createProjectTagOutput struct {
	Body dto.SuccessResponse[dto.CreateProjectTagResponse]
}

type listProjectTagsInput dto.ListProjectTagsRequest

type listProjectTagsOutput struct {
	Body dto.SuccessResponse[dto.ListProjectTagsResponse]
}

type getProjectTagInput dto.ProjectTagDetailRequest

type getProjectTagOutput struct {
	Body dto.SuccessResponse[dto.ProjectTagDetailResponse]
}

type updateProjectTagInput struct {
	ProjectID uint `path:"project_id" minimum:"1"`
	TagID     uint `path:"tag_id" minimum:"1"`
	Body      dto.UpdateProjectTagRequest
}

type updateProjectTagOutput struct {
	Body dto.SuccessResponse[dto.UpdateProjectTagResponse]
}

type deleteProjectTagInput dto.DeleteProjectTagRequest

type deleteProjectTagOutput struct {
	Body dto.SuccessResponse[dto.DeleteProjectTagResponse]
}

func RegisterProjectTagRoutes(api huma.API, r ProjectTagRouter) {
	huma.Register(api, huma.Operation{
		OperationID: "createProjectTag",
		Method:      http.MethodPost,
		Path:        "/projects/{project_id}/tags",
		Summary:     "Create a project tag",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *createProjectTagInput) (*createProjectTagOutput, error) {
		input.Body.ProjectID = input.ProjectID
		response, err := r.CreateProjectTag(ctx, input.Body)
		return &createProjectTagOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listProjectTags",
		Method:      http.MethodGet,
		Path:        "/projects/{project_id}/tags",
		Summary:     "List project tags",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *listProjectTagsInput) (*listProjectTagsOutput, error) {
		response, err := r.ListProjectTags(ctx, dto.ListProjectTagsRequest(*input))
		return &listProjectTagsOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getProjectTag",
		Method:      http.MethodGet,
		Path:        "/projects/{project_id}/tags/{tag_id}",
		Summary:     "Get a project tag",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *getProjectTagInput) (*getProjectTagOutput, error) {
		response, err := r.GetProjectTag(ctx, dto.ProjectTagDetailRequest(*input))
		return &getProjectTagOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateProjectTag",
		Method:      http.MethodPut,
		Path:        "/projects/{project_id}/tags/{tag_id}",
		Summary:     "Update a project tag",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *updateProjectTagInput) (*updateProjectTagOutput, error) {
		input.Body.ProjectID = input.ProjectID
		input.Body.TagID = input.TagID
		response, err := r.UpdateProjectTag(ctx, input.Body)
		return &updateProjectTagOutput{Body: response}, openAPIError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteProjectTag",
		Method:      http.MethodDelete,
		Path:        "/projects/{project_id}/tags/{tag_id}",
		Summary:     "Delete a project tag",
		Tags:        []string{"Assets"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *deleteProjectTagInput) (*deleteProjectTagOutput, error) {
		response, err := r.DeleteProjectTag(ctx, dto.DeleteProjectTagRequest(*input))
		return &deleteProjectTagOutput{Body: response}, openAPIError(err)
	})
}
