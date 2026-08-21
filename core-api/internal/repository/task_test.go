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
	return sqlmock.NewRows([]string{
		"id", "type", "status", "payload", "result", "error", "created_at", "updated_at",
	}).AddRow(
		taskID,
		string(generator.GenerateAnimation),
		uint(status),
		[]byte(`{"project_id":42,"asset_id":9}`),
		nil,
		"provider failed",
		sql.NullTime{},
		sql.NullTime{},
	)
}
