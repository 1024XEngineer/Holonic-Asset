package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectDaoStub struct {
	dao.ProjectDao
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
