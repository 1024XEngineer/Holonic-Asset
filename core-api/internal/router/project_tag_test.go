package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type projectTagRouterStub struct {
	created dto.CreateProjectTagRequest
	updated dto.UpdateProjectTagRequest
	deleted dto.DeleteProjectTagRequest
}

func (s *projectTagRouterStub) CreateProjectTag(
	_ context.Context,
	request dto.CreateProjectTagRequest,
) (dto.SuccessResponse[dto.CreateProjectTagResponse], error) {
	s.created = request
	return dto.NewTypedSuccessResponse(dto.CreateProjectTagResponse{Tag: dto.ProjectTagResponse{
		TagID: 7, ProjectID: request.ProjectID, Name: request.Name, Color: request.Color,
	}}), nil
}

func (s *projectTagRouterStub) ListProjectTags(
	_ context.Context,
	request dto.ListProjectTagsRequest,
) (dto.SuccessResponse[dto.ListProjectTagsResponse], error) {
	return dto.NewTypedSuccessResponse(dto.ListProjectTagsResponse{Tags: []dto.ProjectTagResponse{{
		TagID: 7, ProjectID: request.ProjectID, Name: "player", Color: "#123456",
	}}}), nil
}

func (s *projectTagRouterStub) GetProjectTag(
	_ context.Context,
	request dto.ProjectTagDetailRequest,
) (dto.SuccessResponse[dto.ProjectTagDetailResponse], error) {
	return dto.NewTypedSuccessResponse(dto.ProjectTagDetailResponse{Tag: dto.ProjectTagResponse{
		TagID: request.TagID, ProjectID: request.ProjectID, Name: "player", Color: "#123456",
	}}), nil
}

func (s *projectTagRouterStub) UpdateProjectTag(
	_ context.Context,
	request dto.UpdateProjectTagRequest,
) (dto.SuccessResponse[dto.UpdateProjectTagResponse], error) {
	s.updated = request
	return dto.NewTypedSuccessResponse(dto.UpdateProjectTagResponse{Tag: dto.ProjectTagResponse{
		TagID: request.TagID, ProjectID: request.ProjectID, Name: *request.Name, Color: "#123456",
	}}), nil
}

func (s *projectTagRouterStub) DeleteProjectTag(
	_ context.Context,
	request dto.DeleteProjectTagRequest,
) (dto.SuccessResponse[dto.DeleteProjectTagResponse], error) {
	s.deleted = request
	return dto.NewTypedSuccessResponse(dto.DeleteProjectTagResponse{Success: true}), nil
}

func TestProjectTagRoutesExposeScopedCRUD(t *testing.T) {
	stub := &projectTagRouterStub{}
	server := router.Register(nil, nil, nil, nil, stub)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/projects/42/tags", `{"name":"player","color":"#123456"}`},
		{http.MethodGet, "/api/v1/projects/42/tags", ""},
		{http.MethodGet, "/api/v1/projects/42/tags/7", ""},
		{http.MethodPut, "/api/v1/projects/42/tags/7", `{"name":"hero"}`},
		{http.MethodDelete, "/api/v1/projects/42/tags/7", ""},
	}
	for _, request := range requests {
		req := httptest.NewRequest(request.method, request.path, strings.NewReader(request.body))
		if request.body != "" {
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d: %s", request.method, request.path, response.Code, response.Body.String())
		}
	}

	if stub.created.ProjectID != 42 || stub.created.Name != "player" {
		t.Fatalf("unexpected create request: %+v", stub.created)
	}
	if stub.updated.ProjectID != 42 || stub.updated.TagID != 7 || stub.updated.Name == nil || *stub.updated.Name != "hero" {
		t.Fatalf("unexpected update request: %+v", stub.updated)
	}
	if stub.deleted.ProjectID != 42 || stub.deleted.TagID != 7 {
		t.Fatalf("unexpected delete request: %+v", stub.deleted)
	}
}
