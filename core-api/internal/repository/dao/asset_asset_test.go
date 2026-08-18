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

func TestAssetDaoUpdatesCurrentContentAndThumbnail(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "assets" SET "content_id"=$1,"thumbnail_url"=$2,"version"=$3 WHERE id = $4`,
	)).WithArgs(11, "uploads/hero.png", 4, 7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := (&AssetDaoImpl{DB: db}).UpdateAssetCurrentContent(
		context.Background(),
		7,
		4,
		11,
		"uploads/hero.png",
	)
	if err != nil {
		t.Fatalf("update current content: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAssetDaoUpdateCurrentContentReportsMissingAsset(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "assets" SET "content_id"=$1,"thumbnail_url"=$2,"version"=$3 WHERE id = $4`,
	)).WithArgs(11, "", 4, 7).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := (&AssetDaoImpl{DB: db}).UpdateAssetCurrentContent(
		context.Background(),
		7,
		4,
		11,
		"",
	)
	if err == nil || !regexp.MustCompile(`asset 7 not found`).MatchString(err.Error()) {
		t.Fatalf("expected missing asset error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
