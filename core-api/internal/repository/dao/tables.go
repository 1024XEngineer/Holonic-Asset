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
	bootstrapProjectTemplates := false
	if db.Name() == "postgres" {
		bootstrapProjectTemplates = !db.Migrator().HasColumn(&ProjectTag{}, "template_id")
		if err := migrateLegacyTagTableName(db); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(
		&User{},
		&Project{},
		&Asset{},
		&TagTemplate{},
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
	if err := seedTagTemplates(db); err != nil {
		return err
	}
	if bootstrapProjectTemplates {
		if err := bootstrapProjectTagsFromTemplates(db); err != nil {
			return err
		}
	}
	return migrateAssetAttributes(db)
}

// cleanupLegacyTagColumns removes columns that belonged to the former tags and
// ordered asset_tags schemas. AutoMigrate intentionally does not drop columns.
func cleanupLegacyTagColumns(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_project_normalized_name`,
		`DROP INDEX IF EXISTS idx_project_tags_template`,
		`ALTER TABLE project_tags DROP COLUMN IF EXISTS normalized_name`,
		`ALTER TABLE asset_tags DROP COLUMN IF EXISTS position`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("dao: cleanup legacy tag schema: %w", err)
		}
	}
	return nil
}

// bootstrapProjectTagsFromTemplates runs only when template_id is introduced.
// Later project-tag deletions remain deleted across application restarts.
func bootstrapProjectTagsFromTemplates(db *gorm.DB) error {
	if err := db.Exec(`UPDATE project_tags
		SET template_id = tag_templates.id
		FROM tag_templates
		WHERE project_tags.template_id IS NULL
		  AND lower(trim(project_tags.name)) = lower(trim(tag_templates.name))
		  AND NOT EXISTS (
			SELECT 1
			FROM project_tags AS linked
			WHERE linked.project_id = project_tags.project_id
			  AND linked.template_id = tag_templates.id
		  )`).Error; err != nil {
		return fmt.Errorf("dao: link existing project tags to templates: %w", err)
	}
	if err := db.Exec(`INSERT INTO project_tags (project_id, template_id, name, description, color)
		SELECT projects.id, tag_templates.id, tag_templates.name, tag_templates.description, tag_templates.color
		FROM projects
		CROSS JOIN tag_templates
		WHERE NOT EXISTS (
			SELECT 1
			FROM project_tags
			WHERE project_tags.project_id = projects.id
			  AND (
				project_tags.template_id = tag_templates.id
				OR lower(trim(project_tags.name)) = lower(trim(tag_templates.name))
			  )
		)
		ON CONFLICT DO NOTHING`).Error; err != nil {
		return fmt.Errorf("dao: bootstrap project tags from templates: %w", err)
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
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_project_tags_project_name_ci ON project_tags (project_id, lower(trim(name)))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_project_tags_project_template ON project_tags (project_id, template_id) WHERE template_id IS NOT NULL`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("dao: ensure project tag index: %w", err)
		}
	}
	return nil
}

var defaultTagTemplates = []TagTemplate{
	{Name: "warriors", Description: "Warrior characters and combat units", Color: "#4F46E5"},
	{Name: "weapon", Description: "Weapons and combat equipment", Color: "#DC2626"},
	{Name: "environment", Description: "Environment and scenery assets", Color: "#16A34A"},
	{Name: "background", Description: "Backgrounds and backdrop assets", Color: "#0891B2"},
}

func seedTagTemplates(db *gorm.DB) error {
	for index := range defaultTagTemplates {
		template := defaultTagTemplates[index]
		if err := db.Transaction(func(tx *gorm.DB) error {
			result := tx.Where("name = ?", template.Name).FirstOrCreate(&template)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			return copyTagTemplateToProjects(tx, template)
		}); err != nil {
			return fmt.Errorf("dao: seed tag template %q: %w", template.Name, err)
		}
	}
	return nil
}

func copyTagTemplateToProjects(db *gorm.DB, template TagTemplate) error {
	if err := db.Exec(`UPDATE project_tags
		SET template_id = ?
		WHERE project_tags.template_id IS NULL
		  AND lower(trim(project_tags.name)) = lower(trim(?))
		  AND NOT EXISTS (
			SELECT 1
			FROM project_tags AS linked
			WHERE linked.project_id = project_tags.project_id
			  AND linked.template_id = ?
		  )`, template.ID, template.Name, template.ID).Error; err != nil {
		return fmt.Errorf("dao: link tag template %q to projects: %w", template.Name, err)
	}
	if err := db.Exec(`INSERT INTO project_tags (project_id, template_id, name, description, color)
		SELECT projects.id, ?, ?, ?, ?
		FROM projects
		WHERE NOT EXISTS (
			SELECT 1
			FROM project_tags
			WHERE project_tags.project_id = projects.id
			  AND (
				project_tags.template_id = ?
				OR lower(trim(project_tags.name)) = lower(trim(?))
			  )
		)
		ON CONFLICT DO NOTHING`,
		template.ID, template.Name, template.Description, template.Color,
		template.ID, template.Name,
	).Error; err != nil {
		return fmt.Errorf("dao: copy tag template %q to projects: %w", template.Name, err)
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
