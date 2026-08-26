package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/datatypes"
)

func TestAssetRecordDaoCreate(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := (&AssetRecordDaoImpl{}).WithDB(db)

	// nil check
	_, err := dao.CreateAssetRecord(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on nil record")
	}

	// success
	record := &AssetRecord{
		AssetID:     1,
		Version:     1,
		ContentID:   2,
		Name:        "Rec1",
		Description: "Desc1",
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(`{"w":16,"h":16}`),
		CreatedAt:   time.Now(),
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_records"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(50))
	mock.ExpectCommit()

	id, err := dao.CreateAssetRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("create asset record: %v", err)
	}
	if id != 50 {
		t.Fatalf("expected id 50, got %d", id)
	}

	// error
	wantErr := errors.New("insert error")
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_records"`).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	_, err = dao.CreateAssetRecord(context.Background(), record)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetRecordDaoCreateRecords(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetRecordDaoImpl{DB: db}

	// empty check
	if err := dao.CreateAssetRecords(context.Background(), nil); err != nil {
		t.Fatalf("expected nil for empty records, got %v", err)
	}

	records := []AssetRecord{
		{AssetID: 1, Version: 1, ContentID: 1},
		{AssetID: 1, Version: 2, ContentID: 2},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_records"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	mock.ExpectCommit()

	if err := dao.CreateAssetRecords(context.Background(), records); err != nil {
		t.Fatalf("create asset records: %v", err)
	}

	// error
	wantErr := errors.New("batch insert error")
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "asset_records"`).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.CreateAssetRecords(context.Background(), records); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetRecordDaoGetOperations(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetRecordDaoImpl{DB: db}

	// GetAssetRecord success
	mock.ExpectQuery(`SELECT .* FROM "asset_records" WHERE asset_id = \$1 AND version = \$2 ORDER BY "asset_records"\."id" LIMIT \$3`).
		WithArgs(10, 2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "asset_id", "version", "name"}).
			AddRow(5, 10, 2, "Sprite"))

	rec, err := dao.GetAssetRecord(context.Background(), 10, 2)
	if err != nil {
		t.Fatalf("get asset record: %v", err)
	}
	if rec.ID != 5 || rec.AssetID != 10 || rec.Version != 2 || rec.Name != "Sprite" {
		t.Fatalf("unexpected record: %+v", rec)
	}

	// GetAssetRecord error
	wantErr := errors.New("not found")
	mock.ExpectQuery(`SELECT .* FROM "asset_records" WHERE asset_id = \$1 AND version = \$2 ORDER BY "asset_records"\."id" LIMIT \$3`).
		WithArgs(10, 2, 1).
		WillReturnError(wantErr)

	_, err = dao.GetAssetRecord(context.Background(), 10, 2)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	// GetAssetRecordsByAssetID success
	mock.ExpectQuery(`SELECT .* FROM "asset_records" WHERE asset_id = \$1 ORDER BY version ASC`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "asset_id", "version"}).
			AddRow(1, 10, 1).
			AddRow(2, 10, 2))

	list, err := dao.GetAssetRecordsByAssetID(context.Background(), 10)
	if err != nil {
		t.Fatalf("get asset records by asset id: %v", err)
	}
	if len(list) != 2 || list[0].Version != 1 || list[1].Version != 2 {
		t.Fatalf("unexpected list: %+v", list)
	}

	// GetAssetRecordsByAssetID error
	mock.ExpectQuery(`SELECT .* FROM "asset_records" WHERE asset_id = \$1 ORDER BY version ASC`).
		WithArgs(10).
		WillReturnError(wantErr)

	_, err = dao.GetAssetRecordsByAssetID(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetRecordDaoDeleteOperations(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := &AssetRecordDaoImpl{DB: db}
	wantErr := errors.New("delete error")

	// DeleteAssetRecord
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1 AND version = $2`)).
		WithArgs(10, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.DeleteAssetRecord(context.Background(), 10, 3); err != nil {
		t.Fatalf("delete asset record: %v", err)
	}

	// DeleteAssetRecord error
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1 AND version = $2`)).
		WithArgs(10, 3).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.DeleteAssetRecord(context.Background(), 10, 3); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	// DeleteAssetRecordsAfterVersion
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1 AND version > $2`)).
		WithArgs(10, 3).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	if err := dao.DeleteAssetRecordsAfterVersion(context.Background(), 10, 3); err != nil {
		t.Fatalf("delete asset records after version: %v", err)
	}

	// DeleteAssetRecordsAfterVersion error
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1 AND version > $2`)).
		WithArgs(10, 3).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.DeleteAssetRecordsAfterVersion(context.Background(), 10, 3); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	// DeleteAssetRecordsByAssetID
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1`)).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectCommit()

	if err := dao.DeleteAssetRecordsByAssetID(context.Background(), 10); err != nil {
		t.Fatalf("delete asset records by asset id: %v", err)
	}

	// DeleteAssetRecordsByAssetID error
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "asset_records" WHERE asset_id = $1`)).
		WithArgs(10).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.DeleteAssetRecordsByAssetID(context.Background(), 10); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
