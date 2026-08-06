package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectDaoStub struct {
	dao.ProjectDao
}

type assetStoreStub struct {
	assetdomain.Store
}

func TestNewAppBuildsApplication(t *testing.T) {
	app := NewApp(&projectDaoStub{})
	if app == nil {
		t.Fatal("expected server application")
	}
	if app.engine == nil {
		t.Fatal("expected server engine")
	}
}

func TestResolveConfigPathUsesConfigYamlByDefault(t *testing.T) {
	t.Setenv(configPathEnv, "")
	if path := resolveConfigPath(); path != "config.yaml" {
		t.Fatalf("unexpected default config path: %q", path)
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
	t.Setenv(configPathEnv, path)

	app, err := InitServer()
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
