package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type User struct {
	UserID    uint   `gorm:"column:userid;primaryKey"`
	Username  string `gorm:"size:64;not null;uniqueIndex"`
	Password  string `gorm:"column:password;size:255;not null"`
	Email     string `gorm:"size:254"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserDao interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
}

type GormUserDao struct {
	db *gorm.DB
}

func NewGormUserDao(db *gorm.DB) *GormUserDao {
	return &GormUserDao{db: db}
}

func (User) TableName() string {
	return "users"
}

func (d *GormUserDao) FindByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	if err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

var _ UserDao = (*GormUserDao)(nil)
