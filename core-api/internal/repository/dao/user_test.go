package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserTableName(t *testing.T) {
	if table := (User{}).TableName(); table != "users" {
		t.Fatalf("expected users table, got %q", table)
	}
}

func TestGormUserDaoFindByUsername(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "userid","username","password","email" FROM "users" WHERE username = $1 ORDER BY "users"."userid" LIMIT $2`)).
		WithArgs("login-test-user", 1).
		WillReturnRows(sqlmock.NewRows([]string{"userid", "username", "password", "email"}).
			AddRow(7, "login-test-user", "password-hash", "login-test-user@example.com"))

	user, err := NewGormUserDao(db).FindByUsername(context.Background(), "login-test-user")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if user.UserID != 7 || user.Username != "login-test-user" || user.Password != "password-hash" || user.Email != "login-test-user@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestGormUserDaoFindByUsernameReturnsQueryError(t *testing.T) {
	db, mock := newMockUserDatabase(t)
	queryErr := errors.New("query failed")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "userid","username","password","email" FROM "users" WHERE username = $1 ORDER BY "users"."userid" LIMIT $2`)).
		WithArgs("login-test-user", 1).
		WillReturnError(queryErr)

	_, err := NewGormUserDao(db).FindByUsername(context.Background(), "login-test-user")
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func newMockUserDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open GORM database: %v", err)
	}
	return db, mock
}
