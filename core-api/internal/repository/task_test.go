package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/datatypes"

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
