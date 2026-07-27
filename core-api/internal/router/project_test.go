package router_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func TestProjectUpdateRoutePreservesOmittedFields(t *testing.T) {
	projectDao := dao.NewMemoryProjectDao()
	projectRepository := repository.NewProjectRepository(projectDao)
	project := &domain.Project{
		UserID:      7,
		Name:        "Prototype",
		Description: "original description",
		Reference:   "old-reference",
		Style:       "pixel",
	}
	if err := projectRepository.Insert(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectService := service.NewProjectService(projectRepository)
	projectHandler := handler.NewProjectHandler(projectService)
	e := router.Register(nil, projectHandler, nil, nil, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/project/update",
		strings.NewReader(fmt.Sprintf(`{"projectID":%d,"reference":"new-reference"}`, project.ID)),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"success\":true}\n" {
		t.Fatalf("unexpected update response: %s", recorder.Body.String())
	}

	stored, err := projectDao.FindByID(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("find project: %v", err)
	}
	if stored.Reference != "new-reference" {
		t.Fatalf("expected updated reference, got %q", stored.Reference)
	}
	if stored.Name != "Prototype" || stored.Description != "original description" || stored.Style != "pixel" {
		t.Fatalf("omitted fields changed: %+v", stored)
	}
}
