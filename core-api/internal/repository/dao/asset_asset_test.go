package dao

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAssetListLoadsCurrentContentInOneQuery(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT a.id, a.name, a.project_id, a.type, a.description, a.tags, a.perspective, a.dimensions, c.content, a.version FROM assets AS a LEFT JOIN asset_contents AS c ON c.id = a.content_id WHERE a.project_id = $1 ORDER BY a.id ASC`,
	)).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "project_id", "type", "description", "tags", "perspective", "dimensions", "content", "version",
	}).AddRow(
		7, "hero", 42, "character", "main character", `["player"]`, "Top-Down",
		`{"width":64,"height":64}`,
		`{"prototype":[{"id":1,"url":"uploads/hero.png"}]}`,
		3,
	))

	assets, err := (&AssetDaoImpl{DB: db}).GetAssetsByProjectID(context.Background(), 42)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || string(assets[0].Content) != `{"prototype":[{"id":1,"url":"uploads/hero.png"}]}` {
		t.Fatalf("expected current content in list result, got %+v", assets)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
