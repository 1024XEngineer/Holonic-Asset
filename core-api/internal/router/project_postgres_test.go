//go:build integration

package router_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

const defaultProjectTestDatabaseURL = "postgres://holonic:holonic_dev_password@localhost:5432/holonic_asset?sslmode=disable"

func TestProjectRoutesSupportCRUDLifecycleWithPostgreSQL(t *testing.T) {
	projectDao := newPostgresProjectDao(t)
	e := newProjectTestServer(projectDao)
	const userID uint = 4_000_000_001

	createRecorder := serveProjectRequest(t, e, http.MethodPost, "/api/v1/project/create", fmt.Sprintf(`{
		"userID":%d,
		"name":"Prototype",
		"gameType":"RPG",
		"viewType":"TopDown",
		"targetPlatform":"PC",
		"description":"original description",
		"reference":"old-reference",
		"style":"pixel"
	}`, userID))
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("create project: expected status %d, got %d: %s", http.StatusOK, createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse dto.CreateProjectResponse
	decodeProjectResponse(t, createRecorder, &createResponse)
	if createResponse.ID == 0 {
		t.Fatal("expected generated project ID")
	}

	listRecorder := serveProjectRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v1/project/list?userID=%d", userID), "")
	var listResponse dto.ListProjectsResponse
	decodeProjectResponse(t, listRecorder, &listResponse)
	if len(listResponse.Projects) != 1 || listResponse.Projects[0].ID != createResponse.ID {
		t.Fatalf("unexpected project list: %+v", listResponse.Projects)
	}

	detailPath := fmt.Sprintf("/api/v1/project/detail?projectID=%d", createResponse.ID)
	detailRecorder := serveProjectRequest(t, e, http.MethodGet, detailPath, "")
	var detailResponse dto.ProjectDetailResponse
	decodeProjectResponse(t, detailRecorder, &detailResponse)
	if detailResponse.Project == nil || detailResponse.Project.Name != "Prototype" || detailResponse.Project.Reference != "old-reference" {
		t.Fatalf("unexpected project detail: %+v", detailResponse.Project)
	}

	updateBody := fmt.Sprintf(`{"projectID":%d,"description":"","reference":"new-reference"}`, createResponse.ID)
	updateRecorder := serveProjectRequest(t, e, http.MethodPost, "/api/v1/project/update", updateBody)
	var updateResponse dto.UpdateProjectResponse
	decodeProjectResponse(t, updateRecorder, &updateResponse)
	if !updateResponse.Success {
		t.Fatal("expected successful project update")
	}

	detailRecorder = serveProjectRequest(t, e, http.MethodGet, detailPath, "")
	decodeProjectResponse(t, detailRecorder, &detailResponse)
	if detailResponse.Project.Description != "" || detailResponse.Project.Reference != "new-reference" {
		t.Fatalf("unexpected updated project detail: %+v", detailResponse.Project)
	}
	if detailResponse.Project.Name != "Prototype" || detailResponse.Project.Style != "pixel" {
		t.Fatalf("partial update changed omitted fields: %+v", detailResponse.Project)
	}

	deleteBody := fmt.Sprintf(`{"projectID":%d}`, createResponse.ID)
	deleteRecorder := serveProjectRequest(t, e, http.MethodPost, "/api/v1/project/delete", deleteBody)
	var deleteResponse dto.DeleteProjectResponse
	decodeProjectResponse(t, deleteRecorder, &deleteResponse)
	if !deleteResponse.Success {
		t.Fatal("expected successful project deletion")
	}

	missingRecorder := serveProjectRequest(t, e, http.MethodGet, detailPath, "")
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after deletion, got %d: %s", http.StatusNotFound, missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestProjectUpdateRoutePreservesOmittedFieldsInPostgreSQL(t *testing.T) {
	projectDao := newPostgresProjectDao(t)
	projectRepository := repository.NewProjectRepository(projectDao)
	project := &domain.Project{
		UserID:      4_000_000_002,
		Name:        "Prototype",
		Description: "original description",
		Reference:   "old-reference",
		Style:       "pixel",
	}
	if err := projectRepository.Insert(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	e := newProjectTestServer(projectDao)

	recorder := serveProjectRequest(
		t,
		e,
		http.MethodPost,
		"/api/v1/project/update",
		fmt.Sprintf(`{"projectID":%d,"reference":"new-reference"}`, project.ID),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "{\"code\":200,\"message\":\"success\",\"data\":{\"success\":true}}\n" {
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

func newPostgresProjectDao(t *testing.T) *dao.GormProjectDao {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("PROJECT_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = defaultProjectTestDatabaseURL
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open Project test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get Project test connection pool: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("ping Project test database: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		_ = sqlDB.Close()
		t.Fatalf("begin Project test transaction: %v", tx.Error)
	}
	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("roll back Project test transaction: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close Project test database: %v", err)
		}
	})
	if err := tx.AutoMigrate(&dao.Project{}); err != nil {
		t.Fatalf("migrate Project test table: %v", err)
	}
	return dao.NewGormProjectDao(tx)
}

func decodeProjectResponse(t *testing.T, recorder *httptest.ResponseRecorder, response any) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != dto.SuccessCode || envelope.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response envelope: %+v", envelope)
	}
	if err := json.Unmarshal(envelope.Data, response); err != nil {
		t.Fatalf("decode response data: %v", err)
	}
}
