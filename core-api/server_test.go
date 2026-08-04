package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
