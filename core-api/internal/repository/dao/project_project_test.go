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

func TestGormProjectDaoFindByID(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	dao := NewGormProjectDao(db)

	// success
	mock.ExpectQuery(`SELECT \* FROM "projects" WHERE "projects"\."id" = \$1 ORDER BY "projects"\."id" LIMIT \$2`).
		WithArgs(10, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name"}).AddRow(10, 1, "Game 10"))

	project, err := dao.FindByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("find project: %v", err)
	}
	if project.ID != 10 || project.Name != "Game 10" {
		t.Fatalf("unexpected project: %+v", project)
	}

	// not found
	mock.ExpectQuery(`SELECT \* FROM "projects" WHERE "projects"\."id" = \$1 ORDER BY "projects"\."id" LIMIT \$2`).
		WithArgs(99, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name"}))

	_, err = dao.FindByID(context.Background(), 99)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// other error
	wantErr := errors.New("query failed")
	mock.ExpectQuery(`SELECT \* FROM "projects" WHERE "projects"\."id" = \$1 ORDER BY "projects"\."id" LIMIT \$2`).
		WithArgs(100, 1).
		WillReturnError(wantErr)

	_, err = dao.FindByID(context.Background(), 100)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestGormProjectDaoFindByUserID(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	dao := NewGormProjectDao(db)

	mock.ExpectQuery(`SELECT \* FROM "projects" WHERE user_id = \$1 ORDER BY id ASC`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name"}).
			AddRow(1, 7, "P1").
			AddRow(2, 7, "P2"))

	list, err := dao.FindByUserID(context.Background(), 7)
	if err != nil {
		t.Fatalf("find by user id: %v", err)
	}
	if len(list) != 2 || list[0].Name != "P1" || list[1].Name != "P2" {
		t.Fatalf("unexpected projects: %+v", list)
	}
}

func TestGormProjectDaoUpdate(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	dao := NewGormProjectDao(db)

	// nil update
	if err := dao.Update(context.Background(), nil); !errors.Is(err, ErrProjectUpdateNil) {
		t.Fatalf("expected ErrProjectUpdateNil, got %v", err)
	}

	// empty fields update returns nil without executing DB query
	if err := dao.Update(context.Background(), &ProjectUpdate{ID: 10}); err != nil {
		t.Fatalf("expected nil on empty update fields, got %v", err)
	}

	name := "New Name"
	gameType := "Action"
	persp := "Side-On"
	platform := "Web"
	desc := "New Desc"
	ref := "New Ref"
	style := "Pixel"

	fullUpdate := &ProjectUpdate{
		ID:             10,
		Name:           &name,
		GameType:       &gameType,
		Perspective:    &persp,
		TargetPlatform: &platform,
		Description:    &desc,
		Reference:      &ref,
		Style:          &style,
	}

	// update success
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "projects" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.Update(context.Background(), fullUpdate); err != nil {
		t.Fatalf("update project: %v", err)
	}

	// update not found (0 rows affected)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "projects" SET`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := dao.Update(context.Background(), fullUpdate); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// update db error
	wantErr := errors.New("update fail")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "projects" SET`).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.Update(context.Background(), fullUpdate); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestGormProjectDaoDelete(t *testing.T) {
	db, mock := newMockTableDatabase(t)
	dao := NewGormProjectDao(db)

	// success
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "projects" WHERE "projects"\."id" = \$1`).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.Delete(context.Background(), 10); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	// not found
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "projects" WHERE "projects"\."id" = \$1`).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := dao.Delete(context.Background(), 10); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// db error
	wantErr := errors.New("delete fail")
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "projects" WHERE "projects"\."id" = \$1`).
		WithArgs(10).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	if err := dao.Delete(context.Background(), 10); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
