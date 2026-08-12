package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type UserRepository struct {
	dao dao.UserDao
}

func NewUserRepository(userDao dao.UserDao) *UserRepository {
	return &UserRepository{dao: userDao}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	user, err := r.dao.FindByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &auth.User{ID: user.UserID, Username: user.Username, PasswordHash: user.Password, Email: user.Email}, nil
}

var _ auth.Store = (*UserRepository)(nil)
