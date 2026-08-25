package dao

import (
	"context"
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
