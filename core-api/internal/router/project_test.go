package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type unreachableProjectDao struct {
	dao.ProjectDao
}

func TestProjectRoutesRejectInvalidRequests(t *testing.T) {
	e := newProjectTestServer(&unreachableProjectDao{})
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create without name",
			method: http.MethodPost,
			path:   "/api/v1/project/create",
			body:   `{"userID":7,"gameType":"RPG","viewType":"TopDown","targetPlatform":"PC"}`,
		},
		{
			name:   "list without user ID",
			method: http.MethodGet,
			path:   "/api/v1/project/list",
		},
		{
			name:   "update without fields",
			method: http.MethodPost,
			path:   "/api/v1/project/update",
			body:   `{"projectID":42}`,
		},
		{
			name:   "delete without project ID",
			method: http.MethodPost,
			path:   "/api/v1/project/delete",
			body:   `{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := serveProjectRequest(t, e, test.method, test.path, test.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
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
