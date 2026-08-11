package dao

import (
	"errors"

	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	if db == nil {
		return errors.New("dao: database is nil")
	}

	if err := db.AutoMigrate(
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
	return migrateAssetAttributes(db)
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
