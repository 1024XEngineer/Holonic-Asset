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

func TestBootstrapProjectTagsFromTemplatesIsConflictSafe(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectExec(`UPDATE project_tags .*FROM tag_templates.*linked\.template_id = tag_templates\.id`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO project_tags .*CROSS JOIN tag_templates.*ON CONFLICT DO NOTHING`).
		WillReturnResult(sqlmock.NewResult(0, 6))

	if err := bootstrapProjectTagsFromTemplates(db); err != nil {
		t.Fatalf("bootstrap project tags: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
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

func expectLegacyTagColumn(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
		WithArgs("assets", "BASE TABLE").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
		WithArgs("assets", "tags").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
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
