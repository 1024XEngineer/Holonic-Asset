package dao

import (
	"errors"

	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	if db == nil {
		return errors.New("dao: database is nil")
	}
	if db.Name() == "postgres" {
		if err := migrateAssetTagsToJSONB(db); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(
		&User{},
		&Project{},
		&Asset{},
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
	if err := migrateAssetTagsToJSONB(db); err != nil {
		return err
	}
	return migrateAssetAttributes(db)
}

func migrateAssetTagsToJSONB(db *gorm.DB) error {
	if !db.Migrator().HasTable(&Asset{}) || !db.Migrator().HasColumn(&Asset{}, "tags") {
		return nil
	}

	var dataType string
	if err := db.Raw(`SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'assets'
		  AND column_name = 'tags'`).Scan(&dataType).Error; err != nil {
		return err
	}
	if dataType != "jsonb" {
		if err := db.Exec(`ALTER TABLE assets
			ALTER COLUMN tags TYPE jsonb
			USING CASE
				WHEN tags IS NULL OR btrim(tags::text) IN ('', 'null') THEN '[]'::jsonb
				WHEN left(btrim(tags::text), 1) = '[' THEN tags::jsonb
				WHEN left(btrim(tags::text), 1) IN ('{', '"') THEN jsonb_build_array(tags::jsonb)
				ELSE jsonb_build_array(btrim(tags::text))
			END`).Error; err != nil {
			return err
		}
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_assets_tags_gin ON assets USING GIN (tags)`).Error
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
