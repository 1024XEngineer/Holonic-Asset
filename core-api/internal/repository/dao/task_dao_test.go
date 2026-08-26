package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTaskDaoUpdateStatusFrom(t *testing.T) {
	const updateQuery = `UPDATE "tasks" SET "status"=$1,"updated_at"=$2 WHERE id = $3 AND status = $4`

	tests := []struct {
		name         string
		rowsAffected int64
		queryErr     error
		wantUpdated  bool
		wantErr      bool
	}{
		{name: "transitions matching status", rowsAffected: 1, wantUpdated: true},
		{name: "leaves stale status unchanged"},
		{name: "returns update error", queryErr: errors.New("update failed"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockUserDatabase(t)
			mock.ExpectBegin()
			expectation := mock.ExpectExec(regexp.QuoteMeta(updateQuery)).
				WithArgs(2, sqlmock.AnyArg(), 17, 5)
			if tt.queryErr != nil {
				expectation.WillReturnError(tt.queryErr)
				mock.ExpectRollback()
			} else {
				expectation.WillReturnResult(sqlmock.NewResult(0, tt.rowsAffected))
				mock.ExpectCommit()
			}

			updated, err := NewTaskDao(db).UpdateStatusFrom(context.Background(), 17, 5, 2)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected update error: %v", err)
			}
			if updated != tt.wantUpdated {
				t.Fatalf("updated = %t, want %t", updated, tt.wantUpdated)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestTaskDaoCreate(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	task := &Task{
		Type:   "image_gen",
		Status: 1,
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "tasks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectCommit()

	if err := dao.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID != 10 {
		t.Fatalf("expected task id 10, got %d", task.ID)
	}
}

func TestTaskDaoUpdateStatus(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	// success
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "status"=$1,"updated_at"=$2 WHERE id = $3`)).
		WithArgs(2, sqlmock.AnyArg(), 5).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.UpdateStatus(context.Background(), 5, 2); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// error
	wantErr := errors.New("db error")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "status"=$1,"updated_at"=$2 WHERE id = $3`)).
		WithArgs(2, sqlmock.AnyArg(), 5).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.UpdateStatus(context.Background(), 5, 2); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestTaskDaoUpdateResult(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	// success
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.UpdateResult(context.Background(), 5, 3, []byte(`{"url":"xyz"}`)); err != nil {
		t.Fatalf("update result: %v", err)
	}

	// error
	wantErr := errors.New("db error")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET`)).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.UpdateResult(context.Background(), 5, 3, []byte(`{"url":"xyz"}`)); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTaskDaoUpdateFailure(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	// success
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.UpdateFailure(context.Background(), 5, 4, "timeout occurred"); err != nil {
		t.Fatalf("update failure: %v", err)
	}

	// error
	wantErr := errors.New("db error")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET`)).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.UpdateFailure(context.Background(), 5, 4, "timeout occurred"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTaskDaoGetDetail(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	// success
	mock.ExpectQuery(`SELECT .* FROM "tasks" WHERE "tasks"\."id" = \$1 ORDER BY "tasks"\."id" LIMIT \$2`).
		WithArgs(5, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "status"}).AddRow(5, "anim", 2))

	task, err := dao.GetDetail(context.Background(), 5)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if task.ID != 5 || task.Type != "anim" {
		t.Fatalf("unexpected task: %+v", task)
	}

	// error
	wantErr := errors.New("query error")
	mock.ExpectQuery(`SELECT .* FROM "tasks" WHERE "tasks"\."id" = \$1 ORDER BY "tasks"\."id" LIMIT \$2`).
		WithArgs(5, 1).
		WillReturnError(wantErr)

	_, err = dao.GetDetail(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestTaskDaoList(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	dao := NewTaskDao(db)

	filter := TaskListFilter{
		Statuses: []uint{1, 2},
		Types:    []string{"anim"},
		BeforeID: 100,
		Limit:    10,
	}

	// success
	mock.ExpectQuery(`SELECT .* FROM "tasks" WHERE status IN \(\$1,\$2\) AND type IN \(\$3\) AND id < \$4 ORDER BY id DESC LIMIT \$5`).
		WithArgs(1, 2, "anim", 100, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "status"}).
			AddRow(90, "anim", 1).
			AddRow(80, "anim", 2))

	list, err := dao.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(list) != 2 || list[0].ID != 90 {
		t.Fatalf("unexpected list: %+v", list)
	}

	// error
	wantErr := errors.New("list error")
	mock.ExpectQuery(`SELECT .* FROM "tasks" WHERE status IN \(\$1,\$2\) AND type IN \(\$3\) AND id < \$4 ORDER BY id DESC LIMIT \$5`).
		WithArgs(1, 2, "anim", 100, 10).
		WillReturnError(wantErr)

	_, err = dao.List(context.Background(), filter)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
