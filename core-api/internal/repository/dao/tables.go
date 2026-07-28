package dao

import (
	"errors"

	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	if db == nil {
		return errors.New("dao: database is nil")
	}

	return db.AutoMigrate(
		&Project{},
		&Asset{},
		&AssetResource{},
		&AssetVersion{},
		&Task{},
		&Outbox{},
	)
}
