package repository

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type userDaoStub struct {
	user *dao.User
	err  error
}

func (s userDaoStub) FindByUsername(context.Context, string) (*dao.User, error) {
	return s.user, s.err
}

func TestUserRepositoryFindByUsernameMapsUser(t *testing.T) {
	repository := NewUserRepository(userDaoStub{user: &dao.User{
		UserID:   7,
		Username: "login-test-user",
		Password: "password-hash",
		Email:    "login-test-user@example.com",
	}})

	user, err := repository.FindByUsername(context.Background(), "login-test-user")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if user.ID != 7 || user.Username != "login-test-user" || user.PasswordHash != "password-hash" || user.Email != "login-test-user@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestUserRepositoryFindByUsernameMapsNotFound(t *testing.T) {
	repository := NewUserRepository(userDaoStub{err: gorm.ErrRecordNotFound})

	_, err := repository.FindByUsername(context.Background(), "missing")
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("expected user not found, got %v", err)
	}
}

func TestUserRepositoryFindByUsernamePreservesErrors(t *testing.T) {
	daoErr := errors.New("database unavailable")
	repository := NewUserRepository(userDaoStub{err: daoErr})

	_, err := repository.FindByUsername(context.Background(), "login-test-user")
	if !errors.Is(err, daoErr) {
		t.Fatalf("expected DAO error, got %v", err)
	}
}
