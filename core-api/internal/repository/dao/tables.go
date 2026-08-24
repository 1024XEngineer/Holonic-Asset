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
	if db.Name() == "postgres" {
		if err := migrateLegacyTagTableName(db); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(
		&User{},
		&Project{},
		&Asset{},
		&ProjectTag{},
		&AssetTag{},
		&AssetContent{},
		&AssetRecord{},
		&Task{},
		&Outbox{},
	); err != nil {
		return err
	}
	if db.Name() != "postgres" {
		return nil
	}
	if err := ensureProjectTagIndexes(db); err != nil {
		return err
	}
	if err := cleanupLegacyTagColumns(db); err != nil {
		return err
	}
	if err := migrateAssetTagsToTables(db); err != nil {
		return err
	}
	return migrateAssetAttributes(db)
}

// cleanupLegacyTagColumns removes columns that belonged to the former tags,
// tag_templates, and ordered asset_tags schemas. AutoMigrate intentionally does not drop columns.
func cleanupLegacyTagColumns(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_template`,
		`DROP INDEX IF EXISTS idx_project_tags_project_template`,
		`ALTER TABLE project_tags DROP COLUMN IF EXISTS normalized_name`,
		`ALTER TABLE project_tags DROP COLUMN IF EXISTS template_id`,
		`ALTER TABLE asset_tags DROP COLUMN IF EXISTS position`,
		`DROP TABLE IF EXISTS tag_templates CASCADE`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("dao: cleanup legacy tag schema: %w", err)
		}
	}
	return nil
}

func migrateLegacyTagTableName(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ProjectTag{}) && db.Migrator().HasTable("tags") {
		return db.Exec(`ALTER TABLE tags RENAME TO project_tags`).Error
	}
	return nil
}

func ensureProjectTagIndexes(db *gorm.DB) error {
	statement := `CREATE UNIQUE INDEX IF NOT EXISTS idx_project_tags_project_name_ci ON project_tags (project_id, lower(trim(name)))`
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("dao: ensure project tag index: %w", err)
	}
	return nil
}

type legacyAssetTagsRow struct {
	AssetID   uint
	ProjectID uint
	Tags      string
}

func migrateAssetTagsToTables(db *gorm.DB) error {
	if !db.Migrator().HasTable(&Asset{}) || !db.Migrator().HasColumn(&Asset{}, "tags") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		rows, err := tx.Raw(`SELECT id AS asset_id, project_id, tags::text AS tags
			FROM assets
			WHERE tags IS NOT NULL
			ORDER BY id ASC`).Rows()
		if err != nil {
			return fmt.Errorf("dao: read legacy asset tags: %w", err)
		}
		legacyRows := make([]legacyAssetTagsRow, 0)
		for rows.Next() {
			var row legacyAssetTagsRow
			if err := rows.Scan(&row.AssetID, &row.ProjectID, &row.Tags); err != nil {
				_ = rows.Close()
				return fmt.Errorf("dao: scan legacy asset tags: %w", err)
			}
			legacyRows = append(legacyRows, row)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("dao: iterate legacy asset tags: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("dao: close legacy asset tags: %w", err)
		}

		tagDAO := &AssetDaoImpl{DB: tx}
		for _, row := range legacyRows {
			tags, err := decodeAssetTags(row.Tags)
			if err != nil {
				return fmt.Errorf("dao: migrate tags for asset %d: %w", row.AssetID, err)
			}
			if _, err := tagDAO.replaceAssetTags(tx.Statement.Context, row.AssetID, row.ProjectID, tags); err != nil {
				return fmt.Errorf("dao: migrate tags for asset %d: %w", row.AssetID, err)
			}
		}
		if err := tx.Exec(`DROP INDEX IF EXISTS idx_assets_tags_gin`).Error; err != nil {
			return fmt.Errorf("dao: drop legacy asset tag index: %w", err)
		}
		if err := tx.Exec(`ALTER TABLE assets DROP COLUMN tags`).Error; err != nil {
			return fmt.Errorf("dao: drop legacy asset tag column: %w", err)
		}
		return nil
	})
}

func migrateAssetAttributes(db *gorm.DB) error {
	statements := []string{
		`UPDATE assets AS a
		 SET perspective = c.content->>'perspective'
		 FROM asset_contents AS c
		 WHERE a.content_id = c.id
		   AND COALESCE(a.perspective, '') = ''
		   AND COALESCE(c.content->>'perspective', '') <> ''`,
		`UPDATE assets AS a
		 SET perspective = p.perspective
		 FROM projects AS p
		 WHERE a.project_id = p.id
		   AND COALESCE(a.perspective, '') = ''`,
		`UPDATE asset_records AS r
		 SET name = a.name,
		     description = a.description,
		     perspective = COALESCE(
		       NULLIF((SELECT c.content->>'perspective' FROM asset_contents AS c WHERE c.id = r.content_id), ''),
		       a.perspective
		     ),
		     dimensions = a.dimensions
		 FROM assets AS a
		 WHERE r.asset_id = a.id`,
		`UPDATE asset_contents
		 SET content = content - 'perspective' - 'tileSize'
		 WHERE content ? 'perspective' OR content ? 'tileSize'`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasColumn(&Asset{}, "attributes") {
		return db.Migrator().DropColumn(&Asset{}, "attributes")
	}
	return nil
}
