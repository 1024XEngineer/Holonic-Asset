package dao

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
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
		tags       []assetdomain.Tag
		storedTags string
	}{
		{
			name:       "one tag",
			tags:       []assetdomain.Tag{{Name: "pixel-art", Color: "#123456"}},
			storedTags: `[{"name":"pixel-art","color":"#123456"}]`,
		},
		{
			name: "multiple tags",
			tags: []assetdomain.Tag{
				{Name: "object", Description: "game object"},
				{Name: "pixel-art", Color: "#123456"},
			},
			storedTags: `[{"name":"object","description":"game object"},{"name":"pixel-art","color":"#123456"}]`,
		},
		{
			name:       "legacy scalar is readable",
			tags:       []assetdomain.Tag{{Name: "pixel-art", Color: assetdomain.DefaultTagColor}},
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
				if !reflect.DeepEqual(got.Tags[index], tt.tags[index]) {
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
		want  []assetdomain.Tag
	}{
		{name: "structured array", input: `[{"name":"object","description":"prop","color":"#123456"}]`, want: []assetdomain.Tag{{Name: "object", Description: "prop", Color: "#123456"}}},
		{name: "legacy array", input: `["object","pixel-art"]`, want: []assetdomain.Tag{{Name: "object", Color: assetdomain.DefaultTagColor}, {Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "json scalar", input: `"pixel-art"`, want: []assetdomain.Tag{{Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "plain text", input: "pixel-art", want: []assetdomain.Tag{{Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected tags: %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAssetTagsSerializerValue(t *testing.T) {
	serializer := assetTagsSerializer{}

	got, err := serializer.Value(context.Background(), nil, reflect.Value{}, []assetdomain.Tag{{Name: "object"}, {Name: "pixel-art", Color: "#123456"}})
	if err != nil {
		t.Fatalf("serialize tags: %v", err)
	}
	if got != `[{"name":"object"},{"name":"pixel-art","color":"#123456"}]` {
		t.Fatalf("unexpected serialized tags: %#v", got)
	}

	_, err = serializer.Value(context.Background(), nil, reflect.Value{}, func() {})
	if err == nil {
		t.Fatal("expected unsupported value to fail JSON serialization")
	}
}

func TestDecodeAssetTagsInputTypes(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []assetdomain.Tag
		wantErr bool
	}{
		{name: "nil", input: nil, want: nil},
		{name: "empty string", input: "", want: nil},
		{name: "whitespace", input: "  ", want: nil},
		{name: "byte array", input: []byte(`["object","pixel-art"]`), want: []assetdomain.Tag{{Name: "object", Color: assetdomain.DefaultTagColor}, {Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "string slice", input: []string{"object", "pixel-art"}, want: []assetdomain.Tag{{Name: "object", Color: assetdomain.DefaultTagColor}, {Name: "pixel-art", Color: assetdomain.DefaultTagColor}}},
		{name: "unsupported database type", input: func() {}, wantErr: true},
		{name: "malformed json array", input: `[{"name":`, wantErr: true},
		{name: "malformed json object", input: `{"name":`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAssetTags(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected decoding to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("decode tags: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected tags: %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAssetTagsSerializerScanReturnsDecodeError(t *testing.T) {
	db, _ := newMockAssetDatabase(t)
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&Asset{}); err != nil {
		t.Fatalf("parse asset schema: %v", err)
	}
	field := statement.Schema.LookUpField("Tags")
	if field == nil {
		t.Fatal("tags field not found")
	}

	asset := Asset{}
	err := (assetTagsSerializer{}).Scan(
		context.Background(),
		field,
		reflect.ValueOf(&asset),
		func() {},
	)
	if err == nil {
		t.Fatal("expected scan to return a decode error")
	}
}

func TestAssetDaoUpdateAssetWithoutFieldsReloadsExistingAsset(t *testing.T) {
	db, mock := newMockAssetDatabase(t)
	mock.ExpectQuery(
		regexp.QuoteMeta(`SELECT `)+`.+`+
			regexp.QuoteMeta(` FROM "assets" WHERE "assets"."id" = $1 ORDER BY "assets"."id" LIMIT $2`),
	).
		WithArgs(uint(32), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "project_id", "type", "description", "tags",
			"perspective", "dimensions", "content_id", "version",
		}).AddRow(
			32, "槟榔树苗", 1, "object", "槟榔，棕榈科的常绿乔木", `["pixel-art"]`,
			"Top-Down", `{"width":48,"height":48}`, nil, 1,
		))

	got, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{})
	if err != nil {
		t.Fatalf("reload asset: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0].Name != "pixel-art" || got.Tags[0].Color != assetdomain.DefaultTagColor {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAssetDaoUpdateAssetErrors(t *testing.T) {
	t.Run("nil update", func(t *testing.T) {
		_, err := (&AssetDaoImpl{}).UpdateAsset(context.Background(), 32, nil)
		if err == nil {
			t.Fatal("expected nil update to fail")
		}
	})

	t.Run("update query error", func(t *testing.T) {
		db, mock := newMockAssetDatabase(t)
		tags := []assetdomain.Tag{{Name: "pixel-art"}}
		queryErr := errors.New("update failed")
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "assets" SET "tags"=$1 WHERE id = $2`)).
			WithArgs(`[{"name":"pixel-art"}]`, uint(32)).
			WillReturnError(queryErr)
		mock.ExpectRollback()

		_, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tags})
		if !errors.Is(err, queryErr) {
			t.Fatalf("expected update error %v, got %v", queryErr, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet database expectations: %v", err)
		}
	})

	t.Run("asset not found", func(t *testing.T) {
		db, mock := newMockAssetDatabase(t)
		tags := []assetdomain.Tag{{Name: "pixel-art"}}
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "assets" SET "tags"=$1 WHERE id = $2`)).
			WithArgs(`[{"name":"pixel-art"}]`, uint(32)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		_, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tags})
		if err == nil {
			t.Fatal("expected missing asset to fail")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet database expectations: %v", err)
		}
	})

	t.Run("reload error", func(t *testing.T) {
		db, mock := newMockAssetDatabase(t)
		tags := []assetdomain.Tag{{Name: "pixel-art"}}
		queryErr := errors.New("reload failed")
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "assets" SET "tags"=$1 WHERE id = $2`)).
			WithArgs(`[{"name":"pixel-art"}]`, uint(32)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(
			regexp.QuoteMeta(`SELECT `)+`.+`+
				regexp.QuoteMeta(` FROM "assets" WHERE "assets"."id" = $1 ORDER BY "assets"."id" LIMIT $2`),
		).
			WithArgs(uint(32), 1).
			WillReturnError(queryErr)

		_, err := (&AssetDaoImpl{DB: db}).UpdateAsset(context.Background(), 32, &AssetUpdate{Tags: &tags})
		if !errors.Is(err, queryErr) {
			t.Fatalf("expected reload error %v, got %v", queryErr, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet database expectations: %v", err)
		}
	})
}

func encodeTestTags(t *testing.T, tags []assetdomain.Tag) string {
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
