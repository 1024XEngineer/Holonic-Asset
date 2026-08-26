package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/datatypes"
)

func TestAssetContentDaoCreateAssetContent(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := (&AssetContentDaoImpl{}).WithDB(db)

	// nil check
	_, err := dao.CreateAssetContent(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on nil content")
	}

	// success
	content := &AssetContent{
		AssetID: 10,
		Content: datatypes.JSON(`{"key":"value"}`),
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_contents"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
	mock.ExpectCommit()

	created, err := dao.CreateAssetContent(context.Background(), content)
	if err != nil {
		t.Fatalf("create asset content: %v", err)
	}
	if created.ID != 100 || created.AssetID != 10 {
		t.Fatalf("unexpected created content: %+v", created)
	}

	// db error
	wantErr := errors.New("db error")
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_contents"`).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	_, err = dao.CreateAssetContent(context.Background(), content)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetContentDaoCreateAssetContents(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := (&AssetContentDaoImpl{DB: db})

	// empty slice returns nil
	if err := dao.CreateAssetContents(context.Background(), nil); err != nil {
		t.Fatalf("expected nil for empty contents, got %v", err)
	}

	// success
	contents := []AssetContent{
		{AssetID: 1, Content: datatypes.JSON(`{"item":1}`)},
		{AssetID: 1, Content: datatypes.JSON(`{"item":2}`)},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_contents"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	mock.ExpectCommit()

	if err := dao.CreateAssetContents(context.Background(), contents); err != nil {
		t.Fatalf("create asset contents: %v", err)
	}

	// db error
	wantErr := errors.New("batch insert error")
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_contents"`).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.CreateAssetContents(context.Background(), contents); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetContentDaoGetAssetContent(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetContentDaoImpl{DB: db}

	// success
	mock.ExpectQuery(`SELECT .* FROM "asset_contents" WHERE "asset_contents"\."id" = \$1 ORDER BY "asset_contents"\."id" LIMIT \$2`).
		WithArgs(5, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "asset_id", "content"}).
			AddRow(5, 20, datatypes.JSON(`{"test":true}`)))

	item, err := dao.GetAssetContent(context.Background(), 5)
	if err != nil {
		t.Fatalf("get asset content: %v", err)
	}
	if item.ID != 5 || item.AssetID != 20 {
		t.Fatalf("unexpected content: %+v", item)
	}

	// error
	wantErr := errors.New("record not found")
	mock.ExpectQuery(`SELECT .* FROM "asset_contents" WHERE "asset_contents"\."id" = \$1 ORDER BY "asset_contents"\."id" LIMIT \$2`).
		WithArgs(5, 1).
		WillReturnError(wantErr)

	_, err = dao.GetAssetContent(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetContentDaoGetAssetContentsByAssetID(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetContentDaoImpl{DB: db}

	// success
	mock.ExpectQuery(`SELECT .* FROM "asset_contents" WHERE asset_id = \$1 ORDER BY id ASC`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "asset_id", "content"}).
			AddRow(1, 10, datatypes.JSON(`{"frame":1}`)).
			AddRow(2, 10, datatypes.JSON(`{"frame":2}`)))

	items, err := dao.GetAssetContentsByAssetID(context.Background(), 10)
	if err != nil {
		t.Fatalf("get asset contents by asset id: %v", err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 {
		t.Fatalf("unexpected contents: %+v", items)
	}

	// error
	wantErr := errors.New("query error")
	mock.ExpectQuery(`SELECT .* FROM "asset_contents" WHERE asset_id = \$1 ORDER BY id ASC`).
		WithArgs(10).
		WillReturnError(wantErr)

	_, err = dao.GetAssetContentsByAssetID(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetContentDaoDeleteAssetContents(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetContentDaoImpl{DB: db}

	// DeleteAssetContents empty slice
	if err := dao.DeleteAssetContents(context.Background(), nil); err != nil {
		t.Fatalf("expected nil for empty ids, got %v", err)
	}

	// DeleteAssetContents with ids
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_contents" WHERE id IN ($1,$2)`)).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	if err := dao.DeleteAssetContents(context.Background(), []uint{1, 2}); err != nil {
		t.Fatalf("delete asset contents: %v", err)
	}

	// DeleteAssetContents error
	wantErr := errors.New("delete error")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_contents" WHERE id IN ($1,$2)`)).
		WithArgs(1, 2).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.DeleteAssetContents(context.Background(), []uint{1, 2}); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	// DeleteAssetContentsByAssetID
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_contents" WHERE asset_id = $1`)).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	if err := dao.DeleteAssetContentsByAssetID(context.Background(), 10); err != nil {
		t.Fatalf("delete asset contents by asset id: %v", err)
	}

	// DeleteAssetContentsByAssetID error
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_contents" WHERE asset_id = $1`)).
		WithArgs(10).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.DeleteAssetContentsByAssetID(context.Background(), 10); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
