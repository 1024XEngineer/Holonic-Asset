package dao

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAssetListLoadsStoredThumbnailWithoutJoiningContent(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, project_id, type, description, tags, perspective, dimensions, thumbnail_url, version FROM "assets" WHERE project_id = $1 ORDER BY id ASC`,
	)).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "project_id", "type", "description", "tags", "perspective", "dimensions", "thumbnail_url", "version",
	}).AddRow(
		7, "hero", 42, "character", "main character", `["player"]`, "Top-Down",
		`{"width":64,"height":64}`,
		"uploads/hero.png",
		3,
	))

	assets, err := (&AssetDaoImpl{DB: db}).GetAssetsByProjectID(context.Background(), 42)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].ThumbnailURL != "uploads/hero.png" || len(assets[0].Content) != 0 {
		t.Fatalf("expected stored thumbnail without content in list result, got %+v", assets)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
