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

func TestOutboxDaoInsert(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewOutboxDao(db)

	record := &Outbox{
		TaskID:    1,
		TaskType:  "video_gen",
		Payload:   datatypes.JSON(`{"key":"value"}`),
		Status:    0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "outboxes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))
	mock.ExpectCommit()

	tx := db.Begin()
	if err := dao.Insert(context.Background(), tx, record); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit: %v", err)
	}
	if record.ID != 123 {
		t.Fatalf("expected id 123, got %d", record.ID)
	}
}

func TestOutboxDaoFetchPending(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewOutboxDao(db)

	// success
	mock.ExpectQuery(`SELECT .* FROM "outboxes" WHERE status = 0 ORDER BY id ASC LIMIT \$1`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "status"}).
			AddRow(1, 10, 0).
			AddRow(2, 11, 0))

	list, err := dao.FetchPending(context.Background(), 5)
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if len(list) != 2 || list[0].ID != 1 || list[1].ID != 2 {
		t.Fatalf("unexpected pending outbox list: %+v", list)
	}

	// error
	wantErr := errors.New("db error")
	mock.ExpectQuery(`SELECT .* FROM "outboxes" WHERE status = 0 ORDER BY id ASC LIMIT \$1`).
		WithArgs(5).
		WillReturnError(wantErr)

	_, err = dao.FetchPending(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestOutboxDaoMarkPublished(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewOutboxDao(db)

	// success
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxes" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.MarkPublished(context.Background(), 10, 500); err != nil {
		t.Fatalf("mark published: %v", err)
	}

	// error
	wantErr := errors.New("update error")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxes" SET`)).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.MarkPublished(context.Background(), 10, 500); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
