package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

type projectTagSQLStateError struct {
	state string
}

func (e projectTagSQLStateError) Error() string {
	return "database write failed"
}

func (e projectTagSQLStateError) SQLState() string {
	return e.state
}

func TestProjectTagDaoCreateScopesTagToExistingProject(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	expectProjectExists(mock, 42)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "project_tags" .*ON CONFLICT DO NOTHING RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectCommit()

	tag := &ProjectTag{ProjectID: 42, Name: "player", Color: "#123456"}
	if err := NewGormProjectTagDao(db).Create(context.Background(), tag); err != nil {
		t.Fatalf("create project tag: %v", err)
	}
	if tag.ID != 7 {
		t.Fatalf("expected generated tag ID 7, got %d", tag.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProjectTagDaoCreateRejectsMissingProject(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectQuery(`SELECT "id" FROM "projects" WHERE "projects"\."id" = \$1`).
		WithArgs(uint(404), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	err := NewGormProjectTagDao(db).Create(context.Background(), &ProjectTag{ProjectID: 404, Name: "player"})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected missing project error, got %v", err)
	}
}

func TestProjectTagDaoListsTagsInStableOrder(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	expectProjectExists(mock, 42)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, project_id, name, description, color FROM "project_tags" WHERE project_id = $1 ORDER BY lower(trim(name)) ASC, id ASC`)).
		WithArgs(uint(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "name", "description", "color"}).
			AddRow(8, 42, "environment", "world", "#16A34A").
			AddRow(7, 42, "player", "hero", "#123456"))

	tags, err := NewGormProjectTagDao(db).ListByProjectID(context.Background(), 42)
	if err != nil {
		t.Fatalf("list project tags: %v", err)
	}
	if len(tags) != 2 || tags[0].ID != 8 || tags[1].ID != 7 {
		t.Fatalf("unexpected tags: %+v", tags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProjectTagDaoGetsUpdatesAndDeletesWithinProjectScope(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectQuery(`SELECT id, project_id, name, description, color FROM "project_tags" WHERE project_id = \$1 AND id = \$2`).
		WithArgs(uint(42), uint(7), 1).
		WillReturnRows(projectTagRows().AddRow(7, 42, "player", "hero", "#123456"))

	tag, err := NewGormProjectTagDao(db).FindByID(context.Background(), 42, 7)
	if err != nil || tag.ID != 7 {
		t.Fatalf("get project tag: tag=%+v err=%v", tag, err)
	}

	color := "#654321"
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "project_tags" SET "color"=$1 WHERE project_id = $2 AND id = $3`)).
		WithArgs(color, uint(42), uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT id, project_id, name, description, color FROM "project_tags" WHERE project_id = \$1 AND id = \$2`).
		WithArgs(uint(42), uint(7), 1).
		WillReturnRows(projectTagRows().AddRow(7, 42, "player", "hero", color))

	tag, err = NewGormProjectTagDao(db).Update(context.Background(), 42, 7, &ProjectTagUpdate{Color: &color})
	if err != nil || tag.Color != color {
		t.Fatalf("update project tag: tag=%+v err=%v", tag, err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "project_tags" WHERE project_id = $1 AND id = $2`)).
		WithArgs(uint(42), uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := NewGormProjectTagDao(db).Delete(context.Background(), 42, 7); err != nil {
		t.Fatalf("delete project tag: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProjectTagDaoMapsMissingRowsAndDuplicateWrites(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectQuery(`SELECT id, project_id, name, description, color FROM "project_tags" WHERE project_id = \$1 AND id = \$2`).
		WithArgs(uint(42), uint(999), 1).
		WillReturnRows(projectTagRows())
	if _, err := NewGormProjectTagDao(db).FindByID(context.Background(), 42, 999); !errors.Is(err, ErrProjectTagNotFound) {
		t.Fatalf("expected missing tag error, got %v", err)
	}

	if got := normalizeProjectTagWriteError(gorm.ErrDuplicatedKey); !errors.Is(got, ErrProjectTagConflict) {
		t.Fatalf("expected duplicate tag conflict, got %v", got)
	}
}

func TestProjectTagDaoRejectsNilWrites(t *testing.T) {
	dao := NewGormProjectTagDao(nil)
	if err := dao.Create(context.Background(), nil); !errors.Is(err, ErrProjectTagNil) {
		t.Fatalf("expected nil tag error, got %v", err)
	}
	if _, err := dao.Update(context.Background(), 42, 7, nil); !errors.Is(err, ErrProjectTagUpdateNil) {
		t.Fatalf("expected nil update error, got %v", err)
	}
}

func TestProjectTagDaoMapsMissingMutations(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	name := "hero"
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "project_tags" SET "name"=$1 WHERE project_id = $2 AND id = $3`)).
		WithArgs(name, uint(42), uint(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if _, err := NewGormProjectTagDao(db).Update(
		context.Background(),
		42,
		999,
		&ProjectTagUpdate{Name: &name},
	); !errors.Is(err, ErrProjectTagNotFound) {
		t.Fatalf("expected missing update error, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "project_tags" WHERE project_id = $1 AND id = $2`)).
		WithArgs(uint(42), uint(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := NewGormProjectTagDao(db).Delete(context.Background(), 42, 999); !errors.Is(err, ErrProjectTagNotFound) {
		t.Fatalf("expected missing delete error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProjectTagDaoReturnsExistingTagForEmptyUpdate(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	mock.ExpectQuery(`SELECT id, project_id, name, description, color FROM "project_tags" WHERE project_id = \$1 AND id = \$2`).
		WithArgs(uint(42), uint(7), 1).
		WillReturnRows(projectTagRows().AddRow(7, 42, "player", "hero", "#123456"))

	tag, err := NewGormProjectTagDao(db).Update(context.Background(), 42, 7, &ProjectTagUpdate{})
	if err != nil || tag.ID != 7 {
		t.Fatalf("empty update lookup: tag=%+v err=%v", tag, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProjectTagDaoBuildsUpdateFieldsAndNormalizesWriteErrors(t *testing.T) {
	name := "hero"
	description := "main character"
	color := "#654321"
	fields := projectTagUpdateFields(&ProjectTagUpdate{Name: &name, Description: &description, Color: &color})
	if fields["name"] != name || fields["description"] != description || fields["color"] != color {
		t.Fatalf("unexpected update fields: %+v", fields)
	}
	if fields := projectTagUpdateFields(&ProjectTagUpdate{}); len(fields) != 0 {
		t.Fatalf("expected no update fields, got %+v", fields)
	}

	uniqueViolation := projectTagSQLStateError{state: "23505"}
	if got := normalizeProjectTagWriteError(uniqueViolation); !errors.Is(got, ErrProjectTagConflict) {
		t.Fatalf("expected SQL state conflict, got %v", got)
	}
	unknown := projectTagSQLStateError{state: "22000"}
	if got := normalizeProjectTagWriteError(unknown); !errors.Is(got, unknown) {
		t.Fatalf("expected unknown write error passthrough, got %v", got)
	}
}

func expectProjectExists(mock sqlmock.Sqlmock, projectID uint) {
	mock.ExpectQuery(`SELECT "id" FROM "projects" WHERE "projects"\."id" = \$1`).
		WithArgs(projectID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(projectID))
}

func projectTagRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "project_id", "name", "description", "color"})
}
