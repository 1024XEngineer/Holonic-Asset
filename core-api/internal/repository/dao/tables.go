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
	if err := migrateAssetContentPerspective(db); err != nil {
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
				WHEN regexp_replace(lower(perspective), '[-_ ]', '', 'g') = 'topdown' THEN 'Top-Down'
				WHEN regexp_replace(lower(perspective), '[-_ ]', '', 'g') IN ('sideon', 'sideview') THEN 'Side-On'
				WHEN lower(perspective) = 'isometric' THEN 'Isometric'
				ELSE 'Top-Down'
			END
			WHERE perspective IS NULL OR perspective NOT IN ('Top-Down', 'Side-On', 'Isometric')`).Error; err != nil {
			return fmt.Errorf("dao: normalize project perspective values: %w", err)
		}
	}

	return nil
}

func migrateAssetContentPerspective(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&AssetContent{}) || !migrator.HasColumn(&AssetContent{}, "content") {
		return nil
	}

	if err := db.Exec(`UPDATE asset_contents
		SET content = (content - 'viewMode') || jsonb_build_object(
			'perspective',
			CASE
				WHEN regexp_replace(lower(COALESCE(NULLIF(content->>'perspective', ''), content->>'viewMode')), '[-_ ]', '', 'g') = 'topdown' THEN 'Top-Down'
				WHEN regexp_replace(lower(COALESCE(NULLIF(content->>'perspective', ''), content->>'viewMode')), '[-_ ]', '', 'g') IN ('sideon', 'sideview') THEN 'Side-On'
				WHEN lower(COALESCE(NULLIF(content->>'perspective', ''), content->>'viewMode')) = 'isometric' THEN 'Isometric'
				ELSE 'Top-Down'
			END
		)
		WHERE content ? 'viewMode'
		   OR content ? 'perspective'`).Error; err != nil {
		return fmt.Errorf("dao: normalize asset content perspective values: %w", err)
	}

	return nil
}
