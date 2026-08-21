package dao

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestAssetListLoadsStoredThumbnailAndReusableTags(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, project_id, type, description, perspective, dimensions, thumbnail_url, version FROM "assets" WHERE project_id = $1 ORDER BY id ASC`,
	)).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "project_id", "type", "description", "perspective", "dimensions", "thumbnail_url", "version",
	}).AddRow(
		7, "hero", 42, "character", "main character", "Top-Down",
		`{"width":64,"height":64}`,
		"uploads/hero.png",
		3,
	))
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT asset_tags.asset_id, project_tags.name, project_tags.description, project_tags.color FROM "asset_tags" JOIN project_tags ON project_tags.id = asset_tags.tag_id WHERE asset_tags.asset_id IN ($1) ORDER BY asset_tags.asset_id ASC, project_tags.name ASC, project_tags.id ASC`,
	)).WithArgs(uint(7)).WillReturnRows(sqlmock.NewRows([]string{
		"asset_id", "name", "description", "color",
	}).AddRow(7, "hero", "main role", "#123456").AddRow(
		7, "player", "", assetdomain.DefaultTagColor,
	))

	assets, err := (&AssetDaoImpl{DB: db}).GetAssetsByProjectID(context.Background(), 42)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].ThumbnailURL != "uploads/hero.png" || len(assets[0].Content) != 0 {
		t.Fatalf("expected stored thumbnail without content in list result, got %+v", assets)
	}
	wantTags := []assetdomain.Tag{
		{Name: "hero", Description: "main role", Color: "#123456"},
		{Name: "player", Color: assetdomain.DefaultTagColor},
	}
	if !reflect.DeepEqual(assets[0].Tags, wantTags) {
		t.Fatalf("unexpected reusable tags: got %#v want %#v", assets[0].Tags, wantTags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAssetListPropagatesTagLoadFailure(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	mock.ExpectQuery(`SELECT id, name, project_id, type, description, perspective, dimensions, thumbnail_url, version FROM "assets"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id"}).AddRow(7, 42))
	wantErr := errors.New("tag query failed")
	mock.ExpectQuery(`SELECT asset_tags\.asset_id, project_tags\.name, project_tags\.description, project_tags\.color FROM "asset_tags"`).
		WillReturnError(wantErr)

	_, err := (&AssetDaoImpl{DB: db}).GetAssetsByProjectID(context.Background(), 42)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected tag query error %v, got %v", wantErr, err)
	}
}

func TestAssetDaoUpdatesCurrentContentAndThumbnail(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
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
	db, mock := newMockAssetDatabase(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "assets" SET "content_id"=$1,"thumbnail_url"=$2,"version"=$3 WHERE id = $4`,
	)).WithArgs(11, "", 4, 7).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := (&AssetDaoImpl{DB: db}).UpdateAssetCurrentContent(context.Background(), 7, 4, 11, "")
	if err == nil || !regexp.MustCompile(`asset 7 not found`).MatchString(err.Error()) {
		t.Fatalf("expected missing asset error, got %v", err)
	}
}

func TestAssetDaoUpdateAssetReusesProjectTagAndDeduplicatesNames(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	tags := []assetdomain.Tag{
		{Name: " Knight ", Description: "request metadata", Color: "#ABCDEF"},
		{Name: "KNIGHT", Description: "duplicate", Color: "#000000"},
		{Name: "", Description: "ignored"},
	}

	mock.ExpectBegin()
	expectAssetRow(mock, 32, 1)
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_tags" WHERE asset_id = $1`)).
		WithArgs(uint(32)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT \* FROM "project_tags" WHERE project_id = \$1 AND lower\(trim\(name\)\) = lower\(trim\(\$2\)\) ORDER BY "project_tags"\."id" LIMIT \$3`).
		WithArgs(uint(1), "Knight", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "name", "description", "color",
		}).AddRow(9, 1, "knight", "canonical metadata", "#123456"))
	mock.ExpectExec(`INSERT INTO "asset_tags"`).
		WithArgs(uint(32), uint(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tags})
	if err != nil {
		t.Fatalf("update asset tags: %v", err)
	}
	want := []assetdomain.Tag{{Name: "knight", Description: "canonical metadata", Color: "#123456"}}
	if !reflect.DeepEqual(got.Tags, want) {
		t.Fatalf("unexpected tags: got %#v want %#v", got.Tags, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestResolveProjectTagCreatesMissingTagWithoutRaceFailure(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	mock.ExpectQuery(`SELECT \* FROM "project_tags" WHERE project_id = \$1 AND lower\(trim\(name\)\) = lower\(trim\(\$2\)\) ORDER BY "project_tags"\."id" LIMIT \$3`).
		WithArgs(uint(42), "knight", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "project_tags" .*ON CONFLICT DO NOTHING RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectCommit()

	tag, err := (&AssetDaoImpl{DB: db}).resolveProjectTag(context.Background(), 42, assetdomain.Tag{
		Name: "knight", Description: "armored", Color: "#123456",
	})
	if err != nil {
		t.Fatalf("resolve project tag: %v", err)
	}
	if tag.ID != 9 || tag.ProjectID != 42 || tag.Name != "knight" {
		t.Fatalf("unexpected project tag: %+v", tag)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAssetDaoUpdateAssetWithoutFieldsReloadsTags(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	mock.ExpectBegin()
	expectAssetRow(mock, 32, 1)
	mock.ExpectQuery(`SELECT asset_tags\.asset_id, project_tags\.name, project_tags\.description, project_tags\.color FROM "asset_tags"`).
		WithArgs(uint(32)).
		WillReturnRows(sqlmock.NewRows([]string{"asset_id", "name", "description", "color"}).
			AddRow(32, "pixel-art", "", assetdomain.DefaultTagColor))
	mock.ExpectCommit()

	got, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{})
	if err != nil {
		t.Fatalf("reload asset: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0].Name != "pixel-art" || got.Tags[0].Color != assetdomain.DefaultTagColor {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}
}

func TestAssetDaoUpdateAssetErrors(t *testing.T) {
	t.Run("nil update", func(t *testing.T) {
		_, err := (&AssetDaoImpl{}).UpdateAsset(context.Background(), 32, nil)
		if err == nil {
			t.Fatal("expected nil update to fail")
		}
	})

	t.Run("asset not found", func(t *testing.T) {
		db, mock := newMockAssetDatabase(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "assets" WHERE "assets"\."id" = \$1`).
			WithArgs(uint(32), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		mock.ExpectRollback()

		_, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{})
		if err == nil {
			t.Fatal("expected missing asset to fail")
		}
	})

	t.Run("association clear failure rolls back", func(t *testing.T) {
		db, mock := newMockAssetDatabase(t)
		tags := []assetdomain.Tag{{Name: "pixel-art"}}
		wantErr := errors.New("clear failed")
		mock.ExpectBegin()
		expectAssetRow(mock, 32, 1)
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_tags" WHERE asset_id = $1`)).
			WithArgs(uint(32)).
			WillReturnError(wantErr)
		mock.ExpectRollback()

		_, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tags})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected clear error %v, got %v", wantErr, err)
		}
	})
}

func TestNormalizeAssetTagsDefaultsAndDeduplicates(t *testing.T) {
	got := normalizeAssetTags([]assetdomain.Tag{
		{Name: " Knight ", Description: " role "},
		{Name: "KNIGHT", Color: "#000000"},
		{Name: " prop ", Color: " #123456 "},
		{Name: "  "},
	})
	want := []assetdomain.Tag{
		{Name: "Knight", Description: "role", Color: assetdomain.DefaultTagColor},
		{Name: "prop", Color: "#123456"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized tags: got %#v want %#v", got, want)
	}
}

func TestDecodeAssetTagsAcceptsJSONArraysAndLegacyScalars(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []assetdomain.Tag
	}{
		{name: "structured array", input: `[{"name":"object","description":"prop","color":"#123456"}]`, want: []assetdomain.Tag{{Name: "object", Description: "prop", Color: "#123456"}}},
		{name: "legacy array", input: `["object","pixel-art"]`, want: []assetdomain.Tag{{Name: "object", Color: assetdomain.DefaultTagColor}, {Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "json scalar", input: `"pixel-art"`, want: []assetdomain.Tag{{Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "plain text", input: "pixel-art", want: []assetdomain.Tag{{Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "null", input: "null", want: nil},
		{name: "string slice", input: []string{"object"}, want: []assetdomain.Tag{{Name: "object", Color: assetdomain.DefaultTagColor}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAssetTags(tt.input)
			if err != nil {
				t.Fatalf("decode tags: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected tags: got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestDecodeAssetTagsRejectsMalformedAndUnsupportedValues(t *testing.T) {
	for _, input := range []any{func() {}, `[{"name":`, `{"name":`} {
		if _, err := decodeAssetTags(input); err == nil {
			t.Fatalf("expected decoding %#v to fail", input)
		}
	}
}

func expectAssetRow(mock sqlmock.Sqlmock, assetID, projectID uint) {
	mock.ExpectQuery(`SELECT \* FROM "assets" WHERE "assets"\."id" = \$1 ORDER BY "assets"\."id" LIMIT \$2`).
		WithArgs(assetID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "project_id", "type", "description", "perspective", "dimensions", "content_id", "version",
		}).AddRow(
			assetID, "seedling", projectID, "object", "tree", "Top-Down", `{"width":48,"height":48}`, nil, 1,
		))
}

func newMockAssetDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open GORM database: %v", err)
	}
	return db, mock
}
