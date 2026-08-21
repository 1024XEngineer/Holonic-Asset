package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type taskDaoStub struct {
	dao.TaskDao
	taskID        uint
	currentStatus uint
	status        uint
	result        datatypes.JSON
	updated       bool
	err           error
}

func (s *taskDaoStub) UpdateStatusFrom(
	_ context.Context,
	taskID uint,
	currentStatus uint,
	status uint,
) (bool, error) {
	s.taskID = taskID
	s.currentStatus = currentStatus
	s.status = status
	return s.updated, s.err
}

func (s *taskDaoStub) UpdateResult(
	_ context.Context,
	taskID uint,
	status uint,
	result datatypes.JSON,
) error {
	s.taskID = taskID
	s.status = status
	s.result = append(datatypes.JSON(nil), result...)
	return s.err
}

func TestTaskRepositoryCompleteTaskTransitionsAwaitingApplication(t *testing.T) {
	stub := &taskDaoStub{updated: true}
	repo := &repository.TaskRepositoryImpl{TaskDao: stub}

	if err := repo.CompleteTask(context.Background(), 17); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if stub.taskID != 17 || stub.currentStatus != uint(taskdomain.StatusAwaitingApplication) ||
		stub.status != uint(taskdomain.StatusCompleted) {
		t.Fatalf("unexpected transition: %+v", stub)
	}
}

func TestTaskRepositoryCompleteTaskReturnsTransitionErrors(t *testing.T) {
	t.Run("dao error", func(t *testing.T) {
		wantErr := errors.New("update failed")
		repo := &repository.TaskRepositoryImpl{TaskDao: &taskDaoStub{err: wantErr}}
		if err := repo.CompleteTask(context.Background(), 17); !errors.Is(err, wantErr) {
			t.Fatalf("expected DAO error, got %v", err)
		}
	})

	t.Run("stale status", func(t *testing.T) {
		repo := &repository.TaskRepositoryImpl{TaskDao: &taskDaoStub{updated: false}}
		err := repo.CompleteTask(context.Background(), 17)
		if err == nil || !strings.Contains(err.Error(), "not awaiting application") {
			t.Fatalf("expected transition error, got %v", err)
		}
	})
}

func TestTaskRepositoryUpdateTaskResultForwardsCompletionStatus(t *testing.T) {
	stub := &taskDaoStub{}
	repo := &repository.TaskRepositoryImpl{TaskDao: stub}
	result := json.RawMessage(`{"asset_id":9}`)

	if err := repo.UpdateTaskResult(
		context.Background(),
		17,
		taskdomain.StatusAwaitingApplication,
		result,
	); err != nil {
		t.Fatalf("update task result: %v", err)
	}
	if stub.taskID != 17 || stub.status != uint(taskdomain.StatusAwaitingApplication) || string(stub.result) != string(result) {
		t.Fatalf("unexpected result update: %+v", stub)
	}
}

func TestTaskRepositoryRetryFailedTaskResetsAndRequeuesAtomically(t *testing.T) {
	db, mock := newMockTaskRepositoryDatabase(t)
	const taskID = uint(17)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "tasks".*FOR UPDATE`).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows(taskID, taskdomain.StatusFailed))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "error"=$1,"result"=$2,"status"=$3,"updated_at"=$4 WHERE id = $5 AND status = $6`)).
		WithArgs("", nil, uint(taskdomain.StatusPending), sqlmock.AnyArg(), taskID, uint(taskdomain.StatusFailed)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO "outboxes" .*RETURNING "id"`).
		WithArgs(taskID, 0, string(generator.GenerateAnimation), sqlmock.AnyArg(), int64(0), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
	mock.ExpectCommit()

	repo := repository.NewTaskRepository(db)
	if err := repo.RetryFailedTask(
		context.Background(),
		taskID,
		taskdomain.StatusAwaitingApplication,
	); err != nil {
		t.Fatalf("retry failed task: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestTaskRepositoryRetryFailedTaskRejectsStaleStatus(t *testing.T) {
	db, mock := newMockTaskRepositoryDatabase(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "tasks".*FOR UPDATE`).
		WithArgs(uint(17), 1).
		WillReturnRows(taskRows(17, taskdomain.StatusProcessing))
	mock.ExpectRollback()

	err := repository.NewTaskRepository(db).RetryFailedTask(context.Background(), 17, taskdomain.StatusCompleted)
	if !errors.Is(err, taskdomain.ErrTaskNotFailed) {
		t.Fatalf("expected failed task status error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestTaskRepositoryDeleteFailedTaskRemovesTaskAndOutboxAtomically(t *testing.T) {
	db, mock := newMockTaskRepositoryDatabase(t)
	const taskID = uint(17)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "tasks".*FOR UPDATE`).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows(taskID, taskdomain.StatusFailed))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "outboxes" WHERE task_id = $1`)).
		WithArgs(taskID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tasks" WHERE id = $1 AND status = $2`)).
		WithArgs(taskID, uint(taskdomain.StatusFailed)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repository.NewTaskRepository(db).DeleteFailedTask(context.Background(), taskID); err != nil {
		t.Fatalf("delete failed task: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestTaskRepositoryRetryFailedTaskRollsBackFailures(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	for _, test := range []struct {
		name    string
		arrange func(sqlmock.Sqlmock)
		wantErr error
		want    string
	}{
		{
			name: "task lock",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLockError(mock, 17, databaseErr)
			},
			wantErr: databaseErr,
		},
		{
			name: "task reset",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectFailedTaskReset(mock, 17, 0, databaseErr)
			},
			wantErr: databaseErr,
		},
		{
			name: "stale reset",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectFailedTaskReset(mock, 17, 0, nil)
			},
			wantErr: taskdomain.ErrTaskNotFailed,
		},
		{
			name: "invalid persisted payload",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, []byte(`{`))
				expectFailedTaskReset(mock, 17, 1, nil)
			},
			want: "marshal retried task",
		},
		{
			name: "retry outbox",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectFailedTaskReset(mock, 17, 1, nil)
				expectRetryOutboxError(mock, 17, databaseErr)
			},
			wantErr: databaseErr,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock := newMockTaskRepositoryDatabase(t)
			mock.ExpectBegin()
			test.arrange(mock)
			mock.ExpectRollback()

			err := repository.NewTaskRepository(db).RetryFailedTask(
				context.Background(),
				17,
				taskdomain.StatusAwaitingApplication,
			)
			assertTaskRepositoryError(t, err, test.wantErr, test.want)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestTaskRepositoryDeleteFailedTaskRollsBackFailures(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	for _, test := range []struct {
		name    string
		arrange func(sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "task lock",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLockError(mock, 17, databaseErr)
			},
			wantErr: databaseErr,
		},
		{
			name: "non-failed task",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusProcessing, validTaskPayload)
			},
			wantErr: taskdomain.ErrTaskNotFailed,
		},
		{
			name: "outbox delete",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectOutboxDelete(mock, 17, databaseErr)
			},
			wantErr: databaseErr,
		},
		{
			name: "task delete",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectOutboxDelete(mock, 17, nil)
				expectFailedTaskDelete(mock, 17, 0, databaseErr)
			},
			wantErr: databaseErr,
		},
		{
			name: "stale delete",
			arrange: func(mock sqlmock.Sqlmock) {
				expectTaskLock(mock, 17, taskdomain.StatusFailed, validTaskPayload)
				expectOutboxDelete(mock, 17, nil)
				expectFailedTaskDelete(mock, 17, 0, nil)
			},
			wantErr: taskdomain.ErrTaskNotFailed,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock := newMockTaskRepositoryDatabase(t)
			mock.ExpectBegin()
			test.arrange(mock)
			mock.ExpectRollback()

			err := repository.NewTaskRepository(db).DeleteFailedTask(context.Background(), 17)
			assertTaskRepositoryError(t, err, test.wantErr, "")
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet database expectations: %v", err)
			}
		})
	}
}

var validTaskPayload = []byte(`{"project_id":42,"asset_id":9}`)

func expectTaskLock(
	mock sqlmock.Sqlmock,
	taskID uint,
	status taskdomain.Status,
	payload []byte,
) {
	mock.ExpectQuery(`SELECT \* FROM "tasks".*FOR UPDATE`).
		WithArgs(taskID, 1).
		WillReturnRows(taskRowsWithPayload(taskID, status, payload))
}

func expectTaskLockError(mock sqlmock.Sqlmock, taskID uint, queryErr error) {
	mock.ExpectQuery(`SELECT \* FROM "tasks".*FOR UPDATE`).
		WithArgs(taskID, 1).
		WillReturnError(queryErr)
}

func expectFailedTaskReset(
	mock sqlmock.Sqlmock,
	taskID uint,
	rowsAffected int64,
	queryErr error,
) {
	expectation := mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "error"=$1,"result"=$2,"status"=$3,"updated_at"=$4 WHERE id = $5 AND status = $6`)).
		WithArgs("", nil, uint(taskdomain.StatusPending), sqlmock.AnyArg(), taskID, uint(taskdomain.StatusFailed))
	if queryErr != nil {
		expectation.WillReturnError(queryErr)
		return
	}
	expectation.WillReturnResult(sqlmock.NewResult(0, rowsAffected))
}

func expectRetryOutboxError(mock sqlmock.Sqlmock, taskID uint, queryErr error) {
	mock.ExpectQuery(`INSERT INTO "outboxes" .*RETURNING "id"`).
		WithArgs(taskID, 0, string(generator.GenerateAnimation), sqlmock.AnyArg(), int64(0), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(queryErr)
}

func expectOutboxDelete(mock sqlmock.Sqlmock, taskID uint, queryErr error) {
	expectation := mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "outboxes" WHERE task_id = $1`)).
		WithArgs(taskID)
	if queryErr != nil {
		expectation.WillReturnError(queryErr)
		return
	}
	expectation.WillReturnResult(sqlmock.NewResult(0, 2))
}

func expectFailedTaskDelete(
	mock sqlmock.Sqlmock,
	taskID uint,
	rowsAffected int64,
	queryErr error,
) {
	expectation := mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tasks" WHERE id = $1 AND status = $2`)).
		WithArgs(taskID, uint(taskdomain.StatusFailed))
	if queryErr != nil {
		expectation.WillReturnError(queryErr)
		return
	}
	expectation.WillReturnResult(sqlmock.NewResult(0, rowsAffected))
}

func assertTaskRepositoryError(t *testing.T, err, wantErr error, want string) {
	t.Helper()
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func newMockTaskRepositoryDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	connection, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: connection}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open gorm database: %v", err)
	}
	return db, mock
}

func taskRows(taskID uint, status taskdomain.Status) *sqlmock.Rows {
	return taskRowsWithPayload(taskID, status, validTaskPayload)
}

func taskRowsWithPayload(
	taskID uint,
	status taskdomain.Status,
	payload []byte,
) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "type", "status", "payload", "result", "error", "created_at", "updated_at",
	}).AddRow(
		taskID,
		string(generator.GenerateAnimation),
		uint(status),
		payload,
		nil,
		"provider failed",
		sql.NullTime{},
		sql.NullTime{},
	)
}
