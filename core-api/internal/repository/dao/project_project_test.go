package dao

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateProjectSeedsDefaultProjectTags(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "projects"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "project_tags"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2).AddRow(3).AddRow(4))
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
