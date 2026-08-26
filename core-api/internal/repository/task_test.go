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
	errorMessage  string
	detail        *dao.Task
	listResult    []*dao.Task
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

func (s *taskDaoStub) UpdateStatus(_ context.Context, taskID uint, status uint) error {
	s.taskID = taskID
	s.status = status
	return s.err
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

func (s *taskDaoStub) UpdateFailure(_ context.Context, taskID uint, status uint, errorMessage string) error {
	s.taskID = taskID
	s.status = status
	s.errorMessage = errorMessage
	return s.err
}

func (s *taskDaoStub) GetDetail(_ context.Context, taskID uint) (*dao.Task, error) {
	s.taskID = taskID
	return s.detail, s.err
}

func (s *taskDaoStub) List(_ context.Context, _ dao.TaskListFilter) ([]*dao.Task, error) {
	return s.listResult, s.err
}

type outboxDaoStub struct {
	dao.OutboxDao
	inserted       *dao.Outbox
	insertErr      error
	pendingRecords []*dao.Outbox
	fetchErr       error
	markedID       uint
	markedQueueID  int64
	markErr        error
}

func (s *outboxDaoStub) Insert(_ context.Context, _ *gorm.DB, record *dao.Outbox) error {
	s.inserted = record
	return s.insertErr
}

func (s *outboxDaoStub) FetchPending(_ context.Context, _ int) ([]*dao.Outbox, error) {
	return s.pendingRecords, s.fetchErr
}

func (s *outboxDaoStub) MarkPublished(_ context.Context, id uint, queueID int64) error {
	s.markedID = id
	s.markedQueueID = queueID
	return s.markErr
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

func TestTaskRepositoryCreateWithOutbox(t *testing.T) {
	db, mock := newMockTaskRepositoryDatabase(t)
	outboxMock := &outboxDaoStub{}
	repo := &repository.TaskRepositoryImpl{
		DB:        db,
		OutboxDao: outboxMock,
	}

	task := &taskdomain.Task{
		Type:    string(generator.GenerateAnimation),
		Payload: json.RawMessage(`{"project_id":42}`),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "tasks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(88))
	mock.ExpectCommit()

	id, err := repo.CreateWithOutbox(context.Background(), task)
	if err != nil {
		t.Fatalf("create with outbox: %v", err)
	}
	if id != 88 || task.ID != 88 {
		t.Fatalf("expected task id 88, got %d", id)
	}
	if outboxMock.inserted == nil || outboxMock.inserted.TaskID != 88 {
		t.Fatalf("unexpected outbox inserted: %+v", outboxMock.inserted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestTaskRepositoryUpdateTaskStatusAndFailure(t *testing.T) {
	stub := &taskDaoStub{}
	repo := &repository.TaskRepositoryImpl{TaskDao: stub}

	if err := repo.UpdateTaskStatus(context.Background(), 12, taskdomain.StatusProcessing); err != nil {
		t.Fatalf("update task status: %v", err)
	}
	if stub.taskID != 12 || stub.status != uint(taskdomain.StatusProcessing) {
		t.Fatalf("unexpected status update: %+v", stub)
	}

	if err := repo.UpdateTaskFailure(context.Background(), 12, "timeout"); err != nil {
		t.Fatalf("update task failure: %v", err)
	}
	if stub.taskID != 12 || stub.status != uint(taskdomain.StatusFailed) || stub.errorMessage != "timeout" {
		t.Fatalf("unexpected failure update: %+v", stub)
	}
}

func TestTaskRepositoryGetTaskByID(t *testing.T) {
	stub := &taskDaoStub{
		detail: &dao.Task{
			ID:      55,
			Type:    string(generator.GenerateAnimation),
			Status:  uint(taskdomain.StatusCompleted),
			Payload: datatypes.JSON(`{"key":"val"}`),
			Result:  datatypes.JSON(`{"res":1}`),
		},
	}
	repo := &repository.TaskRepositoryImpl{TaskDao: stub}

	task, err := repo.GetTaskByID(context.Background(), 55)
	if err != nil {
		t.Fatalf("get task by id: %v", err)
	}
	if task.ID != 55 || task.Type != string(generator.GenerateAnimation) || task.Status != taskdomain.StatusCompleted {
		t.Fatalf("unexpected task: %+v", task)
	}

	// Error case
	wantErr := errors.New("not found")
	stubErr := &taskDaoStub{err: wantErr}
	_, err = (&repository.TaskRepositoryImpl{TaskDao: stubErr}).GetTaskByID(context.Background(), 55)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error containing not found, got %v", err)
	}
}

func TestTaskRepositoryListTasks(t *testing.T) {
	stub := &taskDaoStub{
		listResult: []*dao.Task{
			{ID: 10, Type: "gen", Status: 1},
			{ID: 9, Type: "gen", Status: 2},
		},
	}
	repo := &repository.TaskRepositoryImpl{TaskDao: stub}

	// Nil filter error
	if _, err := repo.ListTasks(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil filter")
	}

	filter := &taskdomain.ListFilter{
		Statuses: []taskdomain.Status{taskdomain.StatusPending},
		Types:    []string{"gen"},
		Limit:    10,
	}
	tasks, err := repo.ListTasks(context.Background(), filter)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 || tasks[0].ID != 10 || tasks[1].ID != 9 {
		t.Fatalf("unexpected list output: %+v", tasks)
	}

	// Error case
	wantErr := errors.New("list failed")
	stubErr := &taskDaoStub{err: wantErr}
	_, err = (&repository.TaskRepositoryImpl{TaskDao: stubErr}).ListTasks(context.Background(), filter)
	if err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("expected list failed error, got %v", err)
	}
}

func TestTaskRepositoryOutboxOperations(t *testing.T) {
	outboxStub := &outboxDaoStub{
		pendingRecords: []*dao.Outbox{
			{ID: 1, Payload: datatypes.JSON(`{"id":1}`)},
			{ID: 2, Payload: datatypes.JSON(`{"id":2}`)},
		},
	}
	repo := &repository.TaskRepositoryImpl{OutboxDao: outboxStub}

	records, err := repo.FetchPendingOutbox(context.Background(), 10)
	if err != nil {
		t.Fatalf("fetch pending outbox: %v", err)
	}
	if len(records) != 2 || records[0].ID != 1 || string(records[0].Payload) != `{"id":1}` {
		t.Fatalf("unexpected records: %+v", records)
	}

	if err := repo.MarkOutboxPublished(context.Background(), 1, 999); err != nil {
		t.Fatalf("mark outbox published: %v", err)
	}
	if outboxStub.markedID != 1 || outboxStub.markedQueueID != 999 {
		t.Fatalf("unexpected mark published params: id=%d queueID=%d", outboxStub.markedID, outboxStub.markedQueueID)
	}

	// Fetch error case
	wantErr := errors.New("fetch error")
	stubErr := &outboxDaoStub{fetchErr: wantErr}
	_, err = (&repository.TaskRepositoryImpl{OutboxDao: stubErr}).FetchPendingOutbox(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
