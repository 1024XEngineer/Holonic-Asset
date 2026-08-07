package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectDaoStub struct {
	dao.ProjectDao
}

type assetStoreStub struct {
	assetdomain.Store
}

type taskManagerLifecycleStub struct {
	task.Manager
	starts int
	stops  int
}

func (s *taskManagerLifecycleStub) Start(context.Context) error {
	s.starts++
	return nil
}

func (s *taskManagerLifecycleStub) Stop() error {
	s.stops++
	return nil
}

type loggerLifecycleStub struct {
	logger.Logger
	infos int
	syncs int
}

func (s *loggerLifecycleStub) Info(string, ...logger.Field) {
	s.infos++
}

func (s *loggerLifecycleStub) Error(string, ...logger.Field) {}

func (s *loggerLifecycleStub) Sync() error {
	s.syncs++
	return nil
}

func TestNewAppBuildsApplication(t *testing.T) {
	engine := echo.New()
	app := NewApp(engine, nil, nil, logger.NewDefaultLogger())
	if app == nil {
		t.Fatal("expected server application")
	}
	if app.engine != engine {
		t.Fatal("expected server engine")
	}
}

func TestResolveConfigPathUsesInternalConfigByDefault(t *testing.T) {
	t.Setenv(configPathEnv, "")
	if path := resolveConfigPath(); path != "./internal/config/config.yaml" {
		t.Fatalf("unexpected default config path: %q", path)
	}
}

func TestResolveConfigPathUsesEnvironmentOverride(t *testing.T) {
	t.Setenv(configPathEnv, "/tmp/holonic/config.yaml")
	if path := resolveConfigPath(); path != "/tmp/holonic/config.yaml" {
		t.Fatalf("unexpected overridden config path: %q", path)
	}
}

func TestInitRouterRegistersApplicationRoutes(t *testing.T) {
	projectStore := repository.NewProjectRepository(&projectDaoStub{})
	workspaceModule := workspace.New(projectStore, &assetStoreStub{}, nil)
	handlers := InitHandlers(
		workspaceModule,
		generator.NewEngine(nil, nil),
		upload.NewManager(nil),
	)
	engine := InitRouter(handlers)

	for _, expectedPath := range []string{
		"/api/v1/projects/:project_id/assets",
		"/api/v1/asset/:asset_id",
		"/api/v1/projects/:project_id/generation-runs",
		"/api/v1/uploads",
	} {
		found := false
		for _, route := range engine.Routes() {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected production app to register route %q", expectedPath)
		}
	}
}

func TestNewAppWithAssetStoreRegistersAssetRoutes(t *testing.T) {
	app := newApp(&projectDaoStub{}, &assetStoreStub{}, nil)

	for _, expectedPath := range []string{
		"/api/v1/projects/:project_id/assets",
		"/api/v1/asset/:asset_id",
	} {
		found := false
		for _, route := range app.engine.Routes() {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected production app to register asset route %q", expectedPath)
		}
	}
}

func TestNewAppWithUploadStoreReturnsSignedUploadTarget(t *testing.T) {
	uploadStore, err := upload.NewQiniuStorage(config.QiniuConfig{
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Bucket:    "asset-bucket",
		Domain:    "cdn.example.com",
	})
	if err != nil {
		t.Fatalf("create upload store: %v", err)
	}
	app := newAppWithServices(&projectDaoStub{}, nil, nil, uploadStore)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/uploads",
		strings.NewReader(`{"contentType":"image/png","contentLength":8}`),
	)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	app.engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			ObjectKey   string `json:"objectKey"`
			ObjectURL   string `json:"objectURL"`
			UploadURL   string `json:"uploadURL"`
			UploadToken string `json:"uploadToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.HasPrefix(response.Data.ObjectKey, "uploads/") || response.Data.UploadToken == "" {
		t.Fatalf("expected signed upload target, got %+v", response.Data)
	}
	if !strings.HasPrefix(response.Data.ObjectURL, "https://cdn.example.com/"+response.Data.ObjectKey+"?") ||
		!strings.Contains(response.Data.ObjectURL, "token=") {
		t.Fatalf("unexpected object URL: %q", response.Data.ObjectURL)
	}
	if response.Data.UploadURL != "https://upload.qiniup.com" {
		t.Fatalf("unexpected upload URL: %q", response.Data.UploadURL)
	}
}

func TestInitServerRejectsInvalidDatabaseConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
db:
  dsn: ""
queue:
  databaseURL: postgres://localhost/holonic
log:
  path: ./logs/app.log
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	app, err := InitServer(path)
	if err == nil {
		t.Fatal("expected invalid database config to be rejected")
	}
	if app != nil {
		t.Fatalf("expected no app on startup failure, got %+v", app)
	}
	if !strings.Contains(err.Error(), "database DSN is required") {
		t.Fatalf("expected database DSN error, got %v", err)
	}
}

func TestAppShutdownStopsLifecycleComponentsOnce(t *testing.T) {
	tasks := &taskManagerLifecycleStub{}
	appLogger := &loggerLifecycleStub{}
	app := NewApp(echo.New(), tasks, nil, appLogger)

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown app: %v", err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown app twice: %v", err)
	}
	if tasks.stops != 1 {
		t.Fatalf("expected task manager to stop once, got %d", tasks.stops)
	}
	if appLogger.syncs != 1 {
		t.Fatalf("expected logger to sync once, got %d", appLogger.syncs)
	}
}

func TestAppStartValidatesInputs(t *testing.T) {
	app := NewApp(echo.New(), nil, nil, logger.NewDefaultLogger())
	if err := app.Start(context.TODO(), ""); err == nil {
		t.Fatal("expected empty HTTP address to be rejected")
	}
}
