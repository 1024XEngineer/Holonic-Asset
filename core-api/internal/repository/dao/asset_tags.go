package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// ProjectTag is the reusable tag selected by assets in one project.
type ProjectTag struct {
	ID          uint   `gorm:"primaryKey"`
	ProjectID   uint   `gorm:"not null;index"`
	Name        string `gorm:"not null"`
	Description string
	Color       string  `gorm:"not null"`
	Project     Project `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// AssetTag associates one asset with one project-scoped tag.
type AssetTag struct {
	AssetID uint       `gorm:"primaryKey"`
	TagID   uint       `gorm:"primaryKey;index"`
	Asset   Asset      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Tag     ProjectTag `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type assetTagRow struct {
	AssetID     uint
	Name        string
	Description string
	Color       string
}

func normalizeAssetTags(tags []assetdomain.Tag) []assetdomain.Tag {
	result := make([]assetdomain.Tag, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		normalizedName := strings.ToLower(name)
		if normalizedName == "" {
			continue
		}
		if _, exists := seen[normalizedName]; exists {
			continue
		}
		seen[normalizedName] = struct{}{}
		color := strings.TrimSpace(tag.Color)
		if color == "" {
			color = assetdomain.DefaultTagColor
		}
		result = append(result, assetdomain.Tag{
			Name:        name,
			Description: strings.TrimSpace(tag.Description),
			Color:       color,
		})
	}
	return result
}

func (a *AssetDaoImpl) resolveProjectTag(
	ctx context.Context,
	projectID uint,
	value assetdomain.Tag,
) (ProjectTag, error) {
	var existing ProjectTag
	lookup := a.DB.WithContext(ctx).
		Where("project_id = ? AND lower(trim(name)) = lower(trim(?))", projectID, value.Name).
		First(&existing)
	if lookup.Error == nil {
		return existing, nil
	}
	if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return ProjectTag{}, fmt.Errorf("dao: get reusable project tag %q: %w", value.Name, lookup.Error)
	}

	tag := ProjectTag{
		ProjectID:   projectID,
		Name:        value.Name,
		Description: value.Description,
		Color:       value.Color,
	}
	result := a.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&tag)
	if result.Error != nil {
		return ProjectTag{}, fmt.Errorf("dao: create project tag %q: %w", value.Name, result.Error)
	}
	if result.RowsAffected > 0 {
		return tag, nil
	}
	if err := a.DB.WithContext(ctx).
		Where("project_id = ? AND lower(trim(name)) = lower(trim(?))", projectID, value.Name).
		First(&existing).Error; err != nil {
		return ProjectTag{}, fmt.Errorf("dao: get concurrently created project tag %q: %w", value.Name, err)
	}
	return existing, nil
}

func (a *AssetDaoImpl) replaceAssetTags(
	ctx context.Context,
	assetID uint,
	projectID uint,
	tags []assetdomain.Tag,
) ([]assetdomain.Tag, error) {
	if err := a.DB.WithContext(ctx).Where("asset_id = ?", assetID).Delete(&AssetTag{}).Error; err != nil {
		return nil, fmt.Errorf("dao: clear tags for asset %d: %w", assetID, err)
	}

	normalized := normalizeAssetTags(tags)
	result := make([]assetdomain.Tag, 0, len(normalized))
	for _, value := range normalized {
		tag, err := a.resolveProjectTag(ctx, projectID, value)
		if err != nil {
			return nil, err
		}
		if err := a.DB.WithContext(ctx).Create(&AssetTag{
			AssetID: assetID,
			TagID:   tag.ID,
		}).Error; err != nil {
			return nil, fmt.Errorf("dao: associate tag %d with asset %d: %w", tag.ID, assetID, err)
		}
		result = append(result, assetdomain.Tag{
			Name:        tag.Name,
			Description: tag.Description,
			Color:       tag.Color,
		})
	}
	return result, nil
}

func (a *AssetDaoImpl) loadAssetTags(ctx context.Context, assets []Asset) error {
	if len(assets) == 0 {
		return nil
	}

	indices := make(map[uint]int, len(assets))
	assetIDs := make([]uint, len(assets))
	for index := range assets {
		indices[assets[index].ID] = index
		assetIDs[index] = assets[index].ID
		assets[index].Tags = make([]assetdomain.Tag, 0)
	}

	var rows []assetTagRow
	if err := a.DB.WithContext(ctx).
		Table("asset_tags").
		Select("asset_tags.asset_id, project_tags.name, project_tags.description, project_tags.color").
		Joins("JOIN project_tags ON project_tags.id = asset_tags.tag_id").
		Where("asset_tags.asset_id IN ?", assetIDs).
		Order("asset_tags.asset_id ASC, project_tags.name ASC, project_tags.id ASC").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("dao: load asset tags: %w", err)
	}
	for _, row := range rows {
		index, exists := indices[row.AssetID]
		if !exists {
			continue
		}
		assets[index].Tags = append(assets[index].Tags, assetdomain.Tag{
			Name:        row.Name,
			Description: row.Description,
			Color:       row.Color,
		})
	}
	return nil
}

func decodeAssetTags(dbValue any) ([]assetdomain.Tag, error) {
	if dbValue == nil {
		return nil, nil
	}

	var raw []byte
	switch value := dbValue.(type) {
	case []byte:
		raw = append([]byte(nil), value...)
	case string:
		raw = []byte(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("dao: decode asset tags: %w", err)
		}
		raw = encoded
	}

	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return nil, nil
	}

	var tags []assetdomain.Tag
	if err := json.Unmarshal(raw, &tags); err == nil {
		return tags, nil
	} else if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		return nil, fmt.Errorf("dao: decode asset tags: %w", err)
	}

	var tag string
	if err := json.Unmarshal(raw, &tag); err == nil {
		return []assetdomain.Tag{{Name: tag, Color: assetdomain.DefaultTagColor}}, nil
	}
	return []assetdomain.Tag{{Name: value, Color: assetdomain.DefaultTagColor}}, nil
}
