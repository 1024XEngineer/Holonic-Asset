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

func newMockTableDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open gorm database: %v", err)
	}
	return gormDB, mock
}

func TestMigrateAssetTagsToJSONB(t *testing.T) {
	t.Run("skips when table does not exist", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		if err := migrateAssetTagsToJSONB(db); err != nil {
			t.Fatalf("expected nil when table does not exist, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("skips when column does not exist", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		if err := migrateAssetTagsToJSONB(db); err != nil {
			t.Fatalf("expected nil when column does not exist, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("skips alter when column is already jsonb and creates index", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'assets' AND column_name = 'tags'`)).
			WillReturnRows(sqlmock.NewRows([]string{"data_type"}).AddRow("jsonb"))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS idx_assets_tags_gin ON assets USING GIN (tags)`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := migrateAssetTagsToJSONB(db); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("runs alter table when column is text and creates index", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'assets' AND column_name = 'tags'`)).
			WillReturnRows(sqlmock.NewRows([]string{"data_type"}).AddRow("text"))
		mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE assets`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS idx_assets_tags_gin ON assets USING GIN (tags)`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := migrateAssetTagsToJSONB(db); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("information schema error propagates", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'assets' AND column_name = 'tags'`)).
			WillReturnError(errors.New("query failed"))

		err := migrateAssetTagsToJSONB(db)
		if err == nil || err.Error() != "query failed" {
			t.Fatalf("expected query failed error, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("alter table error propagates", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'assets' AND column_name = 'tags'`)).
			WillReturnRows(sqlmock.NewRows([]string{"data_type"}).AddRow("text"))
		mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE assets`)).
			WillReturnError(errors.New("alter failed"))

		err := migrateAssetTagsToJSONB(db)
		if err == nil || err.Error() != "alter failed" {
			t.Fatalf("expected alter failed error, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("index creation error propagates", func(t *testing.T) {
		db, mock := newMockTableDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).
			WithArgs("assets", "BASE TABLE").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND column_name = $2`)).
			WithArgs("assets", "tags").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'assets' AND column_name = 'tags'`)).
			WillReturnRows(sqlmock.NewRows([]string{"data_type"}).AddRow("jsonb"))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS idx_assets_tags_gin ON assets USING GIN (tags)`)).
			WillReturnError(errors.New("index failed"))

		err := migrateAssetTagsToJSONB(db)
		if err == nil || err.Error() != "index failed" {
			t.Fatalf("expected index failed error, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
