package dao

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestInitTablesRejectsNilDatabase(t *testing.T) {
	if err := InitTables(nil); err == nil {
		t.Fatal("expected nil database to be rejected")
	}
}

func TestMigrateAssetTagsToTablesSkipsWhenLegacyColumnIsMissing(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
		WithArgs("assets", "BASE TABLE").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
		WithArgs("assets", "tags").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	if err := migrateAssetTagsToTables(db); err != nil {
		t.Fatalf("expected migration to skip missing column, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCleanupLegacyTagColumnsRemovesFormerSchemaFields(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_template`,
		`DROP INDEX IF EXISTS idx_project_tags_project_template`,
		`ALTER TABLE project_tags DROP COLUMN IF EXISTS normalized_name`,
		`ALTER TABLE project_tags DROP COLUMN IF EXISTS template_id`,
		`ALTER TABLE asset_tags DROP COLUMN IF EXISTS position`,
		`DROP TABLE IF EXISTS tag_templates CASCADE`,
	} {
		mock.ExpectExec(regexp.QuoteMeta(statement)).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	if err := cleanupLegacyTagColumns(db); err != nil {
		t.Fatalf("cleanup legacy tag columns: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateLegacyTagTableName(t *testing.T) {
	t.Run("rename legacy table", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		expectTableExists(mock, "project_tags", 0)
		expectTableExists(mock, "tags", 1)
		mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE tags RENAME TO project_tags`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := migrateLegacyTagTableName(db); err != nil {
			t.Fatalf("rename legacy tag table: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("keep current table", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		expectTableExists(mock, "project_tags", 1)

		if err := migrateLegacyTagTableName(db); err != nil {
			t.Fatalf("keep current project tag table: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestEnsureProjectTagIndexesCreatesCurrentIndexes(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	statement := `CREATE UNIQUE INDEX IF NOT EXISTS idx_project_tags_project_name_ci ON project_tags (project_id, lower(trim(name)))`
	mock.ExpectExec(regexp.QuoteMeta(statement)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureProjectTagIndexes(db); err != nil {
		t.Fatalf("ensure project tag indexes: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTagSchemaInitializationPropagatesErrors(t *testing.T) {
	t.Run("cleanup legacy columns", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		wantErr := errors.New("cleanup failed")
		mock.ExpectExec(regexp.QuoteMeta(`DROP INDEX IF EXISTS idx_tags_project_normalized_name`)).
			WillReturnError(wantErr)

		if err := cleanupLegacyTagColumns(db); !errors.Is(err, wantErr) {
			t.Fatalf("expected cleanup error %v, got %v", wantErr, err)
		}
	})

	t.Run("create project tag indexes", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		wantErr := errors.New("index failed")
		mock.ExpectExec(`CREATE UNIQUE INDEX`).WillReturnError(wantErr)

		if err := ensureProjectTagIndexes(db); !errors.Is(err, wantErr) {
			t.Fatalf("expected index error %v, got %v", wantErr, err)
		}
	})
}

func TestMigrateAssetTagsToTablesBackfillsReusableTagsAndDropsLegacyColumn(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	expectLegacyTagColumn(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id AS asset_id, project_id, tags::text AS tags
			FROM assets
			WHERE tags IS NOT NULL
			ORDER BY id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}).
			AddRow(7, 42, `["hero","pixel-art"]`))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_tags" WHERE asset_id = $1`)).
		WithArgs(uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	expectTagLookup(mock, 42, "hero", 101)
	expectAssetTagInsert(mock, 7, 101, 0)
	expectTagLookup(mock, 42, "pixel-art", 102)
	expectAssetTagInsert(mock, 7, 102, 1)
	mock.ExpectExec(regexp.QuoteMeta(`DROP INDEX IF EXISTS idx_assets_tags_gin`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE assets DROP COLUMN tags`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := migrateAssetTagsToTables(db); err != nil {
		t.Fatalf("migrate legacy tags: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateAssetTagsToTablesRollsBackMalformedLegacyValue(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	expectLegacyTagColumn(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id AS asset_id, project_id, tags::text AS tags
			FROM assets
			WHERE tags IS NOT NULL
			ORDER BY id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}).
			AddRow(7, 42, `[{"name":`))
	mock.ExpectRollback()

	err := migrateAssetTagsToTables(db)
	if err == nil {
		t.Fatal("expected malformed legacy tags to fail migration")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateAssetTagsToTablesPropagatesLegacyReadError(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	expectLegacyTagColumn(mock)
	mock.ExpectBegin()
	wantErr := errors.New("legacy read failed")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id AS asset_id, project_id, tags::text AS tags
			FROM assets
			WHERE tags IS NOT NULL
			ORDER BY id ASC`)).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := migrateAssetTagsToTables(db); !errors.Is(err, wantErr) {
		t.Fatalf("expected legacy read error %v, got %v", wantErr, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateAssetTagsToTablesRollsBackProcessingFailures(t *testing.T) {
	tests := []struct {
		name   string
		rows   *sqlmock.Rows
		finish func(sqlmock.Sqlmock, error)
	}{
		{
			name: "scan legacy row",
			rows: sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}).
				AddRow("invalid-id", 42, `[]`),
		},
		{
			name: "iterate legacy rows",
			rows: sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}).
				AddRow(7, 42, `[]`).
				RowError(0, errors.New("row failed")),
		},
		{
			name: "replace asset tags",
			rows: sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}).
				AddRow(7, 42, `[{"name":"chair","color":"#123456"}]`),
			finish: func(mock sqlmock.Sqlmock, wantErr error) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_tags" WHERE asset_id = $1`)).
					WithArgs(uint(7)).
					WillReturnError(wantErr)
			},
		},
		{
			name: "drop legacy index",
			rows: sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}),
			finish: func(mock sqlmock.Sqlmock, wantErr error) {
				mock.ExpectExec(regexp.QuoteMeta(`DROP INDEX IF EXISTS idx_assets_tags_gin`)).
					WillReturnError(wantErr)
			},
		},
		{
			name: "drop legacy column",
			rows: sqlmock.NewRows([]string{"asset_id", "project_id", "tags"}),
			finish: func(mock sqlmock.Sqlmock, wantErr error) {
				mock.ExpectExec(regexp.QuoteMeta(`DROP INDEX IF EXISTS idx_assets_tags_gin`)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE assets DROP COLUMN tags`)).
					WillReturnError(wantErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockTableDatabase(t)
			expectLegacyTagColumn(mock)
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT id AS asset_id, project_id, tags::text AS tags
				FROM assets
				WHERE tags IS NOT NULL
				ORDER BY id ASC`)).
				WillReturnRows(tt.rows)
			wantErr := errors.New(tt.name + " failed")
			if tt.finish != nil {
				tt.finish(mock, wantErr)
			}
			mock.ExpectRollback()

			if err := migrateAssetTagsToTables(db); err == nil {
				t.Fatalf("expected %s to fail", tt.name)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func expectLegacyTagColumn(mock sqlmock.Sqlmock) {
	expectTableExists(mock, "assets", 1)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
		WithArgs("assets", "tags").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
}

func expectTableExists(mock sqlmock.Sqlmock, table string, count int) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
		WithArgs(table, "BASE TABLE").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

func expectTagLookup(mock sqlmock.Sqlmock, projectID uint, name string, id uint) {
	mock.ExpectQuery(`SELECT \* FROM "project_tags" WHERE project_id = \$1 AND lower\(trim\(name\)\) = lower\(trim\(\$2\)\) ORDER BY "project_tags"\."id" LIMIT \$3`).
		WithArgs(projectID, name, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "name", "description", "color",
		}).AddRow(id, projectID, name, "", "#4F46E5"))
}

func expectAssetTagInsert(mock sqlmock.Sqlmock, assetID, tagID, _ uint) {
	mock.ExpectExec(`INSERT INTO "asset_tags"`).
		WithArgs(assetID, tagID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func newMockTableDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm database: %v", err)
	}
	return gormDB, mock
}
