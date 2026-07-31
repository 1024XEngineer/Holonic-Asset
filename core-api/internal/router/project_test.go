package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type unreachableProjectDao struct {
	dao.ProjectDao
}

type projectRouterStub struct {
	router.ProjectRouter
	createRequest dto.CreateProjectRequest
}

func (s *projectRouterStub) Create(
	_ context.Context,
	request dto.CreateProjectRequest,
) (dto.SuccessResponse[dto.CreateProjectResponse], error) {
	s.createRequest = request
	return dto.NewTypedSuccessResponse(dto.CreateProjectResponse{ID: 42}), nil
}

func TestProjectCreateUsesOpenAPIContract(t *testing.T) {
	stub := &projectRouterStub{}
	e := router.Register(nil, stub, nil, nil)
	recorder := serveProjectRequest(
		t,
		e,
		http.MethodPost,
		"/api/v1/project/create",
		`{"userID":7,"name":"Prototype","gameType":"RPG","viewType":"TopDown","targetPlatform":"PC"}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if stub.createRequest.UserID != 7 || stub.createRequest.Name != "Prototype" {
		t.Fatalf("unexpected create request: %+v", stub.createRequest)
	}
	if recorder.Body.String() != "{\"code\":200,\"message\":\"success\",\"data\":{\"id\":42}}\n" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestProjectRoutesRejectInvalidRequests(t *testing.T) {
	e := newProjectTestServer(&unreachableProjectDao{})
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "create without name",
			method:         http.MethodPost,
			path:           "/api/v1/project/create",
			body:           `{"userID":7,"gameType":"RPG","viewType":"TopDown","targetPlatform":"PC"}`,
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "list without user ID",
			method:         http.MethodGet,
			path:           "/api/v1/project/list",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "update without fields",
			method:         http.MethodPost,
			path:           "/api/v1/project/update",
			body:           `{"projectID":42}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "delete without project ID",
			method:         http.MethodPost,
			path:           "/api/v1/project/delete",
			body:           `{}`,
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := serveProjectRequest(t, e, test.method, test.path, test.body)
			if recorder.Code != test.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", test.expectedStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func newProjectTestServer(projectDao dao.ProjectDao) *echo.Echo {
	projectRepository := repository.NewProjectRepository(projectDao)
	projectManager := project.NewManager(projectRepository)
	projectHandler := handler.NewProjectHandler(projectManager)
	return router.Register(nil, projectHandler, nil, nil)
}

func serveProjectRequest(t *testing.T, e *echo.Echo, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	return recorder
}
