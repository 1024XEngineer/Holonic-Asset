package dao

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateProjectDoesNotSeedProjectTags(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "projects"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectCommit()

	project := &Project{Name: "new project"}
	id, err := NewGormProjectDao(db).CreateProject(context.Background(), project)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if id != 42 || project.ID != 42 {
		t.Fatalf("unexpected project ID: returned=%d model=%d", id, project.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateProjectRejectsNilAndPropagatesInsertError(t *testing.T) {
	dao := NewGormProjectDao(nil)
	if _, err := dao.CreateProject(context.Background(), nil); !errors.Is(err, ErrProjectNil) {
		t.Fatalf("expected nil project error, got %v", err)
	}

	db, mock := newMockTableDatabase(t)
	wantErr := errors.New("insert failed")
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "projects"`).WillReturnError(wantErr)
	mock.ExpectRollback()

	if _, err := NewGormProjectDao(db).CreateProject(context.Background(), &Project{Name: "new project"}); !errors.Is(err, wantErr) {
		t.Fatalf("expected insert error %v, got %v", wantErr, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
