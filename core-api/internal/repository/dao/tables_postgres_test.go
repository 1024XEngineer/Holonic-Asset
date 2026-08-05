//go:build integration

package dao

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultMigrationTestDatabaseURL = "postgres://holonic:holonic_dev_password@localhost:5432/holonic_asset?sslmode=disable"

func TestMigrateProjectPerspectiveWithPostgreSQL(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("PROJECT_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = defaultMigrationTestDatabaseURL
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open migration test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get migration test database connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close migration test database: %v", err)
		}
	})

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin migration test transaction: %v", tx.Error)
	}
	t.Cleanup(func() {
		_ = tx.Rollback().Error
	})

	schema := fmt.Sprintf("perspective_migration_%d", time.Now().UnixNano())
	if err := tx.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create migration test schema: %v", err)
	}
	if err := tx.Exec(`SET LOCAL search_path TO "` + schema + `"`).Error; err != nil {
		t.Fatalf("select migration test schema: %v", err)
	}
	if err := tx.Exec(`CREATE TABLE projects (
		id BIGINT PRIMARY KEY,
		view_type TEXT
	)`).Error; err != nil {
		t.Fatalf("create legacy projects table: %v", err)
	}
	if err := tx.Exec(`INSERT INTO projects (id, view_type) VALUES
		(1, 'SideView'),
		(2, 'TopDown'),
		(3, NULL)`).Error; err != nil {
		t.Fatalf("insert legacy project perspectives: %v", err)
	}

	if err := migrateProjectPerspective(tx); err != nil {
		t.Fatalf("migrate legacy project perspective: %v", err)
	}
	assertProjectPerspectiveColumns(t, tx)
	assertProjectPerspectiveValues(t, tx, map[int64]string{
		1: "SideOn",
		2: "TopDown",
		3: "TopDown",
	})

	if err := tx.Exec(`ALTER TABLE projects ADD COLUMN view_type TEXT`).Error; err != nil {
		t.Fatalf("add partial-migration legacy column: %v", err)
	}
	if err := tx.Exec(`UPDATE projects SET perspective = '', view_type = 'Isometric' WHERE id = 2`).Error; err != nil {
		t.Fatalf("prepare partial migration: %v", err)
	}
	if err := migrateProjectPerspective(tx); err != nil {
		t.Fatalf("merge partial project perspective migration: %v", err)
	}
	assertProjectPerspectiveColumns(t, tx)
	assertProjectPerspectiveValues(t, tx, map[int64]string{2: "Isometric"})

	if err := migrateProjectPerspective(tx); err != nil {
		t.Fatalf("repeat project perspective migration: %v", err)
	}
}

func assertProjectPerspectiveColumns(t *testing.T, db *gorm.DB) {
	t.Helper()
	if !db.Migrator().HasColumn(&Project{}, "perspective") {
		t.Fatal("expected perspective column after migration")
	}
	if db.Migrator().HasColumn(&Project{}, "view_type") {
		t.Fatal("expected legacy view_type column to be removed")
	}
}

func assertProjectPerspectiveValues(t *testing.T, db *gorm.DB, expected map[int64]string) {
	t.Helper()
	for id, want := range expected {
		var got string
		if err := db.Table("projects").Select("perspective").Where("id = ?", id).Scan(&got).Error; err != nil {
			t.Fatalf("read perspective for project %d: %v", id, err)
		}
		if got != want {
			t.Errorf("project %d: expected perspective %q, got %q", id, want, got)
		}
	}
}
