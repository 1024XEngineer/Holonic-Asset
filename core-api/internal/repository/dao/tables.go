package dao

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	if db == nil {
		return errors.New("dao: database is nil")
	}
	if err := migrateProjectPerspective(db); err != nil {
		return err
	}

	return db.AutoMigrate(
		&Project{},
		&Asset{},
		&AssetContent{},
		&AssetRecord{},
		&Task{},
		&Outbox{},
	)
}

func migrateProjectPerspective(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&Project{}) {
		return nil
	}

	hasLegacyColumn := migrator.HasColumn(&Project{}, "view_type")
	hasPerspective := migrator.HasColumn(&Project{}, "perspective")

	if hasLegacyColumn && !hasPerspective {
		if err := migrator.RenameColumn(&Project{}, "view_type", "perspective"); err != nil {
			return fmt.Errorf("dao: rename project view_type column: %w", err)
		}
		hasLegacyColumn = false
		hasPerspective = true
	}

	if hasLegacyColumn && hasPerspective {
		if err := db.Exec(`UPDATE projects
			SET perspective = view_type
			WHERE (perspective IS NULL OR perspective = '')
			  AND view_type IS NOT NULL`).Error; err != nil {
			return fmt.Errorf("dao: merge legacy project perspective values: %w", err)
		}
		if err := migrator.DropColumn(&Project{}, "view_type"); err != nil {
			return fmt.Errorf("dao: drop legacy project view_type column: %w", err)
		}
	}

	if hasPerspective {
		if err := db.Exec(`UPDATE projects
			SET perspective = CASE
				WHEN perspective IS NULL OR perspective = '' THEN 'TopDown'
				WHEN perspective = 'SideView' THEN 'SideOn'
				ELSE 'TopDown'
			END
			WHERE perspective IS NULL OR perspective NOT IN ('TopDown', 'SideOn', 'Isometric')`).Error; err != nil {
			return fmt.Errorf("dao: normalize project perspective values: %w", err)
		}
	}

	return nil
}
