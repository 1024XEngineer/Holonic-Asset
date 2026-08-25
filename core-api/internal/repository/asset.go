package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type AssetRepositoryImpl struct {
	DB         *gorm.DB
	AssetDao   dao.AssetDao
	ContentDao dao.AssetContentDao
	RecordDao  dao.AssetRecordDao
}

type assetDaoWithDB interface {
	WithDB(db *gorm.DB) dao.AssetDao
}

type assetContentDaoWithDB interface {
	WithDB(db *gorm.DB) dao.AssetContentDao
}

type assetRecordDaoWithDB interface {
	WithDB(db *gorm.DB) dao.AssetRecordDao
}

type assetDBProvider interface {
	DBHandle() *gorm.DB
}

func NewAssetRepository(
	assetDao dao.AssetDao,
	contentDao dao.AssetContentDao,
	recordDao dao.AssetRecordDao,
) domain.Store {
	return &AssetRepositoryImpl{
		AssetDao:   assetDao,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}
}

func NewAssetRepositoryWithDB(
	db *gorm.DB,
	assetDao dao.AssetDao,
	contentDao dao.AssetContentDao,
	recordDao dao.AssetRecordDao,
) domain.Store {
	return &AssetRepositoryImpl{
		DB:         db,
		AssetDao:   assetDao,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}
}

var _ domain.Store = (*AssetRepositoryImpl)(nil)

func (r *AssetRepositoryImpl) transactionDB() *gorm.DB {
	if r.DB != nil {
		return r.DB
	}
	if provider, ok := r.AssetDao.(assetDBProvider); ok {
		return provider.DBHandle()
	}
	return nil
}

func (r *AssetRepositoryImpl) withDB(db *gorm.DB) (*AssetRepositoryImpl, error) {
	assetDao, ok := r.AssetDao.(assetDaoWithDB)
	if !ok {
		return nil, fmt.Errorf("repository: asset DAO does not support transactions")
	}
	contentDao, ok := r.ContentDao.(assetContentDaoWithDB)
	if !ok {
		return nil, fmt.Errorf("repository: asset content DAO does not support transactions")
	}
	recordDao, ok := r.RecordDao.(assetRecordDaoWithDB)
	if !ok {
		return nil, fmt.Errorf("repository: asset record DAO does not support transactions")
	}
	return &AssetRepositoryImpl{
		DB:         db,
		AssetDao:   assetDao.WithDB(db),
		ContentDao: contentDao.WithDB(db),
		RecordDao:  recordDao.WithDB(db),
	}, nil
}

func (r *AssetRepositoryImpl) inTransaction(ctx context.Context, fn func(*AssetRepositoryImpl) error) error {
	db := r.transactionDB()
	if db == nil {
		return fn(r)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transactionRepository, err := r.withDB(tx)
		if err != nil {
			return err
		}
		return fn(transactionRepository)
	})
}

func (r *AssetRepositoryImpl) GetAssetsByProjectID(ctx context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	assets, err := r.AssetDao.GetAssetsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Asset, 0, len(assets))
	for index := range assets {
		asset := convertAssetToDomain(assets[index])
		if !matchesAssetFilter(asset, filter) {
			continue
		}
		result = append(result, asset)
	}
	return result, nil
}

func matchesAssetFilter(asset domain.Asset, filter domain.AssetListFilter) bool {
	query := strings.TrimSpace(filter.Query)
	if query != "" &&
		!containsFold(asset.Name, query) &&
		!containsFold(asset.Description, query) &&
		!matchesTagQuery(asset.Tags, query) {
		return false
	}

	if len(filter.Types) > 0 && !containsAssetType(filter.Types, asset.Type) {
		return false
	}

	for _, tag := range filter.Tags {
		if !containsTag(asset.Tags, tag) {
			return false
		}
	}

	return true
}

func containsAssetType(types []domain.AssetType, target domain.AssetType) bool {
	return slices.Contains(types, target)
}

func containsTag(tags []domain.Tag, target string) bool {
	target = strings.TrimSpace(target)
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag.Name), target) {
			return true
		}
	}
	return false
}

func matchesTagQuery(tags []domain.Tag, query string) bool {
	for _, tag := range tags {
		if containsFold(tag.Name, query) ||
			containsFold(tag.Description, query) ||
			containsFold(tag.Color, query) {
			return true
		}
	}
	return false
}

func containsFold(value, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

func (r *AssetRepositoryImpl) GetAssetDetail(ctx context.Context, id uint) (*domain.Asset, error) {
	asset, err := r.AssetDao.GetAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	content, err := r.resolveAssetContent(ctx, asset)
	if err != nil {
		return nil, err
	}
	asset.Content = content
	result := convertAssetToDomain(asset)
	return &result, nil
}

func (r *AssetRepositoryImpl) Delete(ctx context.Context, assetID uint) error {
	if r.ContentDao == nil || r.RecordDao == nil {
		return fmt.Errorf("repository: content and record storage are required")
	}
	return r.inTransaction(ctx, func(transactionRepository *AssetRepositoryImpl) error {
		return transactionRepository.deleteAsset(ctx, assetID)
	})
}

func (r *AssetRepositoryImpl) deleteAsset(ctx context.Context, assetID uint) error {
	if _, err := r.AssetDao.GetAssetForUpdate(ctx, assetID); err != nil {
		return err
	}
	if err := r.RecordDao.DeleteAssetRecordsByAssetID(ctx, assetID); err != nil {
		return err
	}
	if err := r.ContentDao.DeleteAssetContentsByAssetID(ctx, assetID); err != nil {
		return err
	}
	return r.AssetDao.DeleteAsset(ctx, assetID)
}

func (r *AssetRepositoryImpl) UpdateAsset(
	ctx context.Context,
	id uint,
	update *domain.AssetUpdate,
) (*domain.Asset, error) {
	if update == nil {
		return nil, fmt.Errorf("repository: asset update is nil")
	}

	value := &dao.AssetUpdate{
		Name:        update.Name,
		Description: update.Description,
	}
	if update.Tags != nil {
		value.Tags = update.Tags
	}
	var current *dao.Asset
	loadCurrent := func() error {
		if current != nil {
			return nil
		}
		asset, err := r.AssetDao.GetAsset(ctx, id)
		if err != nil {
			return err
		}
		current = &asset
		return nil
	}
	if update.Perspective != nil {
		if err := loadCurrent(); err != nil {
			return nil, err
		}
		if current.Type != string(domain.AssetTypeAudio) {
			if !update.Perspective.Valid() {
				return nil, fmt.Errorf("repository: invalid perspective %q", *update.Perspective)
			}
			perspective := string(*update.Perspective)
			value.Perspective = &perspective
		}
	}
	if update.Dimensions != nil {
		if err := loadCurrent(); err != nil {
			return nil, err
		}
		if current.Type != string(domain.AssetTypeAudio) {
			if err := domain.ValidateDimensions(domain.AssetType(current.Type), *update.Dimensions); err != nil {
				return nil, err
			}
			dimensions := datatypes.JSON(append([]byte(nil), (*update.Dimensions)...))
			value.Dimensions = &dimensions
		}
	}

	asset, err := r.AssetDao.UpdateAsset(ctx, id, value)
	if err != nil {
		return nil, err
	}
	result := convertAssetToDomain(asset)
	return &result, nil
}

func convertAssetToDomain(asset dao.Asset) domain.Asset {
	return domain.Asset{
		ID:           asset.ID,
		Name:         asset.Name,
		ProjectID:    asset.ProjectID,
		Type:         domain.AssetType(asset.Type),
		Description:  asset.Description,
		Tags:         append([]domain.Tag(nil), asset.Tags...),
		Perspective:  domain.Perspective(asset.Perspective),
		Dimensions:   append([]byte(nil), asset.Dimensions...),
		ThumbnailURL: asset.ThumbnailURL,
		Content:      append([]byte(nil), asset.Content...),
		Version:      asset.Version,
	}
}

func (r *AssetRepositoryImpl) resolveAssetContent(ctx context.Context, asset dao.Asset) ([]byte, error) {
	if r.ContentDao != nil && asset.ContentID != nil {
		content, err := r.ContentDao.GetAssetContent(ctx, *asset.ContentID)
		if err != nil {
			return nil, err
		}
		return append([]byte(nil), content.Content...), nil
	}
	return append([]byte(nil), asset.Content...), nil
}

func convertAssetToDAO(asset *domain.Asset, assetType domain.AssetType) (*dao.Asset, error) {
	if asset == nil {
		asset = &domain.Asset{}
	}
	perspective := asset.Perspective
	dimensions := asset.Dimensions
	if assetType == domain.AssetTypeAudio {
		// Audio assets are not visual and therefore do not carry visual metadata.
		perspective = ""
		dimensions = nil
	}
	if assetType != domain.AssetTypeAudio && !perspective.Valid() {
		return nil, fmt.Errorf("repository: invalid perspective %q", asset.Perspective)
	}
	if err := domain.ValidateDimensions(assetType, dimensions); err != nil {
		return nil, err
	}
	content, err := asset.DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("repository: decode asset content: %w", err)
	}
	if (assetType == domain.AssetTypeCharacter || assetType == domain.AssetTypeObject) && content.Prototype == nil {
		content.Prototype = &domain.Prototype{}
	}
	encoded, err := domain.EncodeContent(content)
	if err != nil {
		return nil, err
	}

	return &dao.Asset{
		ID:           asset.ID,
		Name:         asset.Name,
		ProjectID:    asset.ProjectID,
		Type:         string(assetType),
		Description:  asset.Description,
		Tags:         append([]domain.Tag(nil), asset.Tags...),
		Perspective:  string(perspective),
		Dimensions:   datatypes.JSON(append([]byte(nil), dimensions...)),
		ThumbnailURL: thumbnailURLFromContent(content),
		Content:      datatypes.JSON(encoded),
		Version:      asset.Version,
	}, nil
}

func thumbnailURLFromContent(content domain.AssetContent) string {
	if content.Prototype == nil {
		return ""
	}
	for _, resource := range *content.Prototype {
		if resource.URL == nil {
			continue
		}
		if thumbnailURL := strings.TrimSpace(*resource.URL); thumbnailURL != "" {
			return thumbnailURL
		}
	}
	return ""
}

func (r *AssetRepositoryImpl) createAsset(ctx context.Context, asset *domain.Asset, assetType domain.AssetType) (uint, error) {
	created, err := r.createAssetResult(ctx, asset, assetType)
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (r *AssetRepositoryImpl) createAssetResult(ctx context.Context, asset *domain.Asset, assetType domain.AssetType) (*domain.Asset, error) {
	value, err := convertAssetToDAO(asset, assetType)
	if err != nil {
		return nil, err
	}
	if r.ContentDao == nil || r.RecordDao == nil {
		return nil, fmt.Errorf("repository: content storage is not configured")
	}
	var result *domain.Asset
	if err := r.inTransaction(ctx, func(transactionRepository *AssetRepositoryImpl) error {
		var err error
		result, err = transactionRepository.createAssetResultInTransaction(ctx, value)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AssetRepositoryImpl) createAssetResultInTransaction(ctx context.Context, value *dao.Asset) (*domain.Asset, error) {
	created, err := r.AssetDao.CreateAsset(ctx, value)
	if err != nil {
		return nil, err
	}
	currentVersion := created.Version
	if currentVersion == 0 {
		currentVersion = 1
		created.Version = currentVersion
	}
	contentRecord, err := r.ContentDao.CreateAssetContent(ctx, &dao.AssetContent{
		AssetID: created.ID,
		Content: datatypes.JSON(value.Content),
	})
	if err != nil {
		return nil, err
	}
	if _, err := r.RecordDao.CreateAssetRecord(ctx, &dao.AssetRecord{
		AssetID:     created.ID,
		Version:     currentVersion,
		ContentID:   contentRecord.ID,
		Name:        created.Name,
		Description: created.Description,
		Perspective: created.Perspective,
		Dimensions:  append(datatypes.JSON(nil), created.Dimensions...),
	}); err != nil {
		return nil, err
	}
	if err := r.AssetDao.UpdateAssetCurrentContent(
		ctx,
		created.ID,
		currentVersion,
		contentRecord.ID,
		created.ThumbnailURL,
	); err != nil {
		return nil, err
	}
	created.ContentID = &contentRecord.ID
	created.Content = append([]byte(nil), value.Content...)
	result := convertAssetToDomain(created)
	return &result, nil
}

func (r *AssetRepositoryImpl) CreateCharacterAsset(ctx context.Context, asset *domain.Asset) (*domain.Asset, error) {
	return r.createAssetResult(ctx, asset, domain.AssetTypeCharacter)
}

func (r *AssetRepositoryImpl) CreateObjectAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return r.createAsset(ctx, asset, domain.AssetTypeObject)
}

func (r *AssetRepositoryImpl) CreateTileSetAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return r.createAsset(ctx, asset, domain.AssetTypeTileSet)
}

func (r *AssetRepositoryImpl) CreateUISetAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return r.createAsset(ctx, asset, domain.AssetTypeUISet)
}

func (r *AssetRepositoryImpl) CreateSceneryAsset(ctx context.Context, asset *domain.Asset) (uint, error) {
	return r.createAsset(ctx, asset, domain.AssetTypeScenery)
}

func (r *AssetRepositoryImpl) CreateRecord(ctx context.Context, record *domain.AssetRecord, expectedVersion uint) (*domain.AssetRecord, error) {
	if record == nil {
		return nil, fmt.Errorf("repository: asset record is nil")
	}
	recordCopy := *record
	var snapshot *domain.AssetRecord
	if err := r.inTransaction(ctx, func(transactionRepository *AssetRepositoryImpl) error {
		var err error
		snapshot, err = transactionRepository.createRecord(ctx, &recordCopy, expectedVersion)
		return err
	}); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (r *AssetRepositoryImpl) createRecord(ctx context.Context, record *domain.AssetRecord, expectedVersion uint) (*domain.AssetRecord, error) {
	asset, err := r.AssetDao.GetAssetForUpdate(ctx, record.AssetID)
	if err != nil {
		return nil, err
	}
	if expectedVersion != 0 && asset.Version != expectedVersion {
		return nil, fmt.Errorf(
			"%w: asset %d expected version %d, current version %d",
			domain.ErrVersionConflict,
			asset.ID,
			expectedVersion,
			asset.Version,
		)
	}
	currentContent, err := r.resolveAssetContent(ctx, asset)
	if err != nil {
		return nil, err
	}
	content := append([]byte(nil), record.Content...)
	if len(content) == 0 {
		content = currentContent
	}
	if r.ContentDao == nil || r.RecordDao == nil {
		return nil, fmt.Errorf("repository: content storage is not configured")
	}
	decodedContent, err := (domain.Asset{
		Type:    domain.AssetType(asset.Type),
		Content: content,
	}).DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("repository: decode asset %d content: %w", asset.ID, err)
	}
	history, err := r.RecordDao.GetAssetRecordsByAssetID(ctx, asset.ID)
	if err != nil {
		return nil, err
	}
	version := record.Version
	if version == 0 {
		version = asset.Version + 1
		for _, candidate := range history {
			if candidate.Version >= version {
				version = candidate.Version + 1
			}
		}
	}
	snapshot := &domain.AssetRecord{
		AssetID:     record.AssetID,
		Version:     version,
		Name:        asset.Name,
		Description: asset.Description,
		Perspective: domain.Perspective(asset.Perspective),
		Dimensions:  append([]byte(nil), asset.Dimensions...),
		Content:     append([]byte(nil), content...),
	}
	if record.Description != "" {
		snapshot.Description = record.Description
	}
	contentRecord, err := r.ContentDao.CreateAssetContent(ctx, &dao.AssetContent{
		AssetID: asset.ID,
		Content: datatypes.JSON(snapshot.Content),
	})
	if err != nil {
		return nil, err
	}
	daoRecord := &dao.AssetRecord{
		AssetID:     snapshot.AssetID,
		Version:     snapshot.Version,
		ContentID:   contentRecord.ID,
		Name:        snapshot.Name,
		Description: snapshot.Description,
		Perspective: string(snapshot.Perspective),
		Dimensions:  datatypes.JSON(append([]byte(nil), snapshot.Dimensions...)),
	}
	recordID, err := r.RecordDao.CreateAssetRecord(ctx, daoRecord)
	if err != nil {
		return nil, err
	}
	snapshot.ID = recordID
	snapshot.ContentID = contentRecord.ID
	snapshot.CreatedAt = daoRecord.CreatedAt
	if err := r.AssetDao.UpdateAssetCurrentContent(
		ctx,
		asset.ID,
		snapshot.Version,
		contentRecord.ID,
		thumbnailURLFromContent(decodedContent),
	); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (r *AssetRepositoryImpl) GetRecordHistory(ctx context.Context, assetID uint) ([]domain.AssetRecord, error) {
	if r.RecordDao == nil {
		return nil, fmt.Errorf("repository: record storage is not configured")
	}
	records, err := r.RecordDao.GetAssetRecordsByAssetID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if r.ContentDao == nil {
		return nil, fmt.Errorf("repository: content storage is not configured")
	}
	result := make([]domain.AssetRecord, 0, len(records))
	for _, record := range records {
		content, err := r.ContentDao.GetAssetContent(ctx, record.ContentID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.AssetRecord{
			ID:          record.ID,
			AssetID:     record.AssetID,
			Version:     record.Version,
			ContentID:   record.ContentID,
			CreatedAt:   record.CreatedAt,
			Name:        record.Name,
			Description: record.Description,
			Perspective: domain.Perspective(record.Perspective),
			Dimensions:  append([]byte(nil), record.Dimensions...),
			Content:     append([]byte(nil), content.Content...),
		})
	}
	return result, nil
}

func (r *AssetRepositoryImpl) RollBackRecord(ctx context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	if r.RecordDao == nil {
		return nil, fmt.Errorf("repository: record storage is not configured")
	}
	if r.ContentDao == nil {
		return nil, fmt.Errorf("repository: content storage is not configured")
	}
	var result *domain.AssetRecord
	if err := r.inTransaction(ctx, func(transactionRepository *AssetRepositoryImpl) error {
		var err error
		result, err = transactionRepository.rollbackRecord(ctx, assetID, version)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AssetRepositoryImpl) rollbackRecord(ctx context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	asset, err := r.AssetDao.GetAssetForUpdate(ctx, assetID)
	if err != nil {
		return nil, err
	}
	candidate, err := r.RecordDao.GetAssetRecord(ctx, assetID, version)
	if err != nil {
		return nil, err
	}
	content, err := r.ContentDao.GetAssetContent(ctx, candidate.ContentID)
	if err != nil {
		return nil, err
	}
	decodedContent, err := (domain.Asset{
		Type:    domain.AssetType(asset.Type),
		Content: json.RawMessage(content.Content),
	}).DecodeContent()
	if err != nil {
		return nil, fmt.Errorf("repository: decode asset %d content: %w", asset.ID, err)
	}
	history, err := r.RecordDao.GetAssetRecordsByAssetID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	retainedContentIDs := make(map[uint]struct{}, len(history))
	contentIDsToDelete := make([]uint, 0)
	for _, record := range history {
		if record.Version <= version {
			retainedContentIDs[record.ContentID] = struct{}{}
			continue
		}
		contentIDsToDelete = appendUniqueUint(contentIDsToDelete, record.ContentID)
	}
	if asset.ContentID != nil {
		if _, retained := retainedContentIDs[*asset.ContentID]; !retained && *asset.ContentID != candidate.ContentID {
			contentIDsToDelete = appendUniqueUint(contentIDsToDelete, *asset.ContentID)
		}
	}
	name := candidate.Name
	description := candidate.Description
	perspective := candidate.Perspective
	dimensions := json.RawMessage(candidate.Dimensions)
	if name == "" {
		name = asset.Name
	}
	if perspective == "" {
		perspective = asset.Perspective
	}
	if len(dimensions) == 0 {
		dimensions = append([]byte(nil), asset.Dimensions...)
	}
	if _, err := r.AssetDao.UpdateAsset(ctx, assetID, &dao.AssetUpdate{
		Name:        &name,
		Description: &description,
		Perspective: &perspective,
		Dimensions:  dimensionsValue(dimensions),
	}); err != nil {
		return nil, err
	}
	if err := r.AssetDao.UpdateAssetCurrentContent(
		ctx,
		assetID,
		version,
		candidate.ContentID,
		thumbnailURLFromContent(decodedContent),
	); err != nil {
		return nil, err
	}
	if err := r.RecordDao.DeleteAssetRecordsAfterVersion(ctx, assetID, version); err != nil {
		return nil, err
	}
	if err := r.ContentDao.DeleteAssetContents(ctx, contentIDsToDelete); err != nil {
		return nil, err
	}
	return &domain.AssetRecord{
		ID:          candidate.ID,
		AssetID:     candidate.AssetID,
		Version:     candidate.Version,
		ContentID:   candidate.ContentID,
		CreatedAt:   candidate.CreatedAt,
		Name:        name,
		Description: description,
		Perspective: domain.Perspective(perspective),
		Dimensions:  append([]byte(nil), dimensions...),
		Content:     append([]byte(nil), content.Content...),
	}, nil
}

func (r *AssetRepositoryImpl) Copy(ctx context.Context, assetID uint, _ uint) (uint, error) {
	if r.ContentDao == nil || r.RecordDao == nil {
		return 0, fmt.Errorf("repository: content storage is not configured")
	}
	var newAssetID uint
	if err := r.inTransaction(ctx, func(transactionRepository *AssetRepositoryImpl) error {
		var err error
		newAssetID, err = transactionRepository.copyAssetInTransaction(ctx, assetID)
		return err
	}); err != nil {
		return 0, err
	}
	return newAssetID, nil
}

func (r *AssetRepositoryImpl) copyAssetInTransaction(ctx context.Context, assetID uint) (uint, error) {
	asset, err := r.AssetDao.GetAssetForUpdate(ctx, assetID)
	if err != nil {
		return 0, err
	}
	records, err := r.RecordDao.GetAssetRecordsByAssetID(ctx, assetID)
	if err != nil {
		return 0, err
	}
	contents, err := r.ContentDao.GetAssetContentsByAssetID(ctx, assetID)
	if err != nil {
		return 0, err
	}

	copyAsset := &dao.Asset{
		Name:         asset.Name,
		ProjectID:    asset.ProjectID,
		Type:         asset.Type,
		Description:  asset.Description,
		Tags:         append([]domain.Tag(nil), asset.Tags...),
		Perspective:  asset.Perspective,
		Dimensions:   append(datatypes.JSON(nil), asset.Dimensions...),
		ThumbnailURL: asset.ThumbnailURL,
		Version:      asset.Version,
	}
	created, err := r.AssetDao.CreateAsset(ctx, copyAsset)
	if err != nil {
		return 0, err
	}

	contentCopies := make([]dao.AssetContent, len(contents))
	for index, content := range contents {
		contentCopies[index] = dao.AssetContent{
			AssetID: created.ID,
			Content: append(datatypes.JSON(nil), content.Content...),
		}
	}
	if err := r.ContentDao.CreateAssetContents(ctx, contentCopies); err != nil {
		return 0, err
	}

	contentIDs := make(map[uint]uint, len(contents))
	for index, content := range contents {
		contentIDs[content.ID] = contentCopies[index].ID
	}

	recordCopies := make([]dao.AssetRecord, 0, len(records))
	for _, record := range records {
		contentID, ok := contentIDs[record.ContentID]
		if !ok {
			return 0, fmt.Errorf("repository: content %d for asset record %d/%d not found", record.ContentID, record.AssetID, record.Version)
		}
		recordCopies = append(recordCopies, dao.AssetRecord{
			AssetID:     created.ID,
			Version:     record.Version,
			ContentID:   contentID,
			Name:        record.Name,
			Description: record.Description,
			Perspective: record.Perspective,
			Dimensions:  append(datatypes.JSON(nil), record.Dimensions...),
			CreatedAt:   record.CreatedAt,
		})
	}
	if err := r.RecordDao.CreateAssetRecords(ctx, recordCopies); err != nil {
		return 0, err
	}

	if asset.ContentID != nil {
		contentID, ok := contentIDs[*asset.ContentID]
		if !ok {
			return 0, fmt.Errorf("repository: current content %d for asset %d not found", *asset.ContentID, assetID)
		}
		if err := r.AssetDao.UpdateAssetCurrentContent(
			ctx,
			created.ID,
			asset.Version,
			contentID,
			asset.ThumbnailURL,
		); err != nil {
			return 0, err
		}
	}
	return created.ID, nil
}

func appendUniqueUint(values []uint, value uint) []uint {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func dimensionsValue(raw json.RawMessage) *datatypes.JSON {
	value := datatypes.JSON(append([]byte(nil), raw...))
	return &value
}
