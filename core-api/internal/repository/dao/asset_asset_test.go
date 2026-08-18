package dao

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func TestAssetDaoUpdateAssetEncodesTagsAsJSON(t *testing.T) {
	tests := []struct {
		name       string
		tags       []string
		storedTags string
	}{
		{
			name:       "one tag",
			tags:       []string{"pixel-art"},
			storedTags: `["pixel-art"]`,
		},
		{
			name:       "multiple tags",
			tags:       []string{"object", "pixel-art"},
			storedTags: `["object","pixel-art"]`,
		},
		{
			name:       "legacy scalar is readable",
			tags:       []string{"pixel-art"},
			storedTags: `pixel-art`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockAssetDatabase(t)
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "assets" SET "tags"=$1 WHERE id = $2`)).
				WithArgs(encodeTestTags(t, tt.tags), uint(32)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			mock.ExpectQuery(
				regexp.QuoteMeta(`SELECT `)+`.+`+
					regexp.QuoteMeta(` FROM "assets" WHERE "assets"."id" = $1 ORDER BY "assets"."id" LIMIT $2`),
			).
				WithArgs(uint(32), 1).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "name", "project_id", "type", "description", "tags",
					"perspective", "dimensions", "content_id", "version",
				}).AddRow(
					32, "槟榔树苗", 1, "object", "槟榔，棕榈科的常绿乔木", tt.storedTags,
					"Top-Down", `{"width":48,"height":48}`, nil, 1,
				))

			got, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tt.tags})
			if err != nil {
				t.Fatalf("update asset: %v", err)
			}
			if len(got.Tags) != len(tt.tags) {
				t.Fatalf("unexpected tags: %#v, want %#v", got.Tags, tt.tags)
			}
			for index := range tt.tags {
				if got.Tags[index] != tt.tags[index] {
					t.Fatalf("unexpected tags: %#v, want %#v", got.Tags, tt.tags)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestDecodeAssetTagsAcceptsJSONArraysAndLegacyScalars(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
	}{
		{name: "array", input: `["object","pixel-art"]`, want: []string{"object", "pixel-art"}},
		{name: "json scalar", input: `"pixel-art"`, want: []string{"pixel-art"}},
		{name: "plain text", input: "pixel-art", want: []string{"pixel-art"}},
		{name: "null", input: "null", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAssetTags(tt.input)
			if err != nil {
				t.Fatalf("decode tags: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("unexpected tags: %#v, want %#v", got, tt.want)
			}
			for index := range got {
				if got[index] != tt.want[index] {
					t.Fatalf("unexpected tags: %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func encodeTestTags(t *testing.T, tags []string) string {
	t.Helper()
	encoded, err := json.Marshal(tags)
	if err != nil {
		t.Fatalf("encode tags: %v", err)
	}
	return string(encoded)
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
