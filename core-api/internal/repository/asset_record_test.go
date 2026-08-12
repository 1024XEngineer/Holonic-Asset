package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gorm.io/datatypes"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

var testRecordCreatedAt = time.Unix(1700000000, 0).UTC()

type recordAssetDaoStub struct {
	dao.AssetDao
	asset          dao.Asset
	updatedAsset   uint
	updatedVersion uint
	updatedContent uint
}

func (s *recordAssetDaoStub) GetAssetDetail(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *recordAssetDaoStub) GetAsset(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *recordAssetDaoStub) GetAssetForUpdate(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *recordAssetDaoStub) UpdateAssetCurrentContent(_ context.Context, assetID uint, version uint, contentID uint) error {
	s.updatedAsset = assetID
	s.updatedVersion = version
	s.updatedContent = contentID
	return nil
}

func (s *recordAssetDaoStub) UpdateAsset(_ context.Context, _ uint, update *dao.AssetUpdate) (dao.Asset, error) {
	if update.Name != nil {
		s.asset.Name = *update.Name
	}
	if update.Description != nil {
		s.asset.Description = *update.Description
	}
	if update.Perspective != nil {
		s.asset.Perspective = *update.Perspective
	}
	if update.Dimensions != nil {
		s.asset.Dimensions = append(datatypes.JSON(nil), (*update.Dimensions)...)
	}
	return s.asset, nil
}

type recordDaoStub struct {
	dao.AssetRecordDao
	records map[uint]dao.AssetRecord
	nextID  uint
}

func (s *recordDaoStub) CreateAssetRecord(_ context.Context, record *dao.AssetRecord) (uint, error) {
	if record.ID == 0 {
		s.nextID++
		record.ID = s.nextID
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = testRecordCreatedAt
	}
	s.records[record.ID] = *record
	return record.ID, nil
}

func (s *recordDaoStub) CreateAssetRecords(ctx context.Context, records []dao.AssetRecord) error {
	for index := range records {
		if _, err := s.CreateAssetRecord(ctx, &records[index]); err != nil {
			return err
		}
	}
	return nil
}

func (s *recordDaoStub) GetAssetRecord(_ context.Context, assetID uint, version uint) (dao.AssetRecord, error) {
	for _, candidate := range s.records {
		if candidate.AssetID == assetID && candidate.Version == version {
			return candidate, nil
		}
	}
	return dao.AssetRecord{}, testingError("asset record not found")
}

func (s *recordDaoStub) GetAssetRecordsByAssetID(_ context.Context, assetID uint) ([]dao.AssetRecord, error) {
	result := make([]dao.AssetRecord, 0)
	for _, candidate := range s.records {
		if candidate.AssetID == assetID {
			result = append(result, candidate)
		}
	}
	return result, nil
}

func (s *recordDaoStub) DeleteAssetRecordsAfterVersion(_ context.Context, assetID uint, version uint) error {
	for id, record := range s.records {
		if record.AssetID == assetID && record.Version > version {
			delete(s.records, id)
		}
	}
	return nil
}

type recordContentDaoStub struct {
	dao.AssetContentDao
	contents map[uint]dao.AssetContent
	nextID   uint
	getErr   error
}

func (s *recordContentDaoStub) CreateAssetContent(_ context.Context, content *dao.AssetContent) (dao.AssetContent, error) {
	if content.ID == 0 {
		s.nextID++
		content.ID = s.nextID
	}
	s.contents[content.ID] = *content
	return *content, nil
}

func (s *recordContentDaoStub) CreateAssetContents(_ context.Context, contents []dao.AssetContent) error {
	for index := range contents {
		if contents[index].ID == 0 {
			s.nextID++
			contents[index].ID = s.nextID
		}
		s.contents[contents[index].ID] = contents[index]
	}
	return nil
}

func (s *recordContentDaoStub) GetAssetContent(_ context.Context, id uint) (dao.AssetContent, error) {
	if s.getErr != nil {
		return dao.AssetContent{}, s.getErr
	}
	return s.contents[id], nil
}

func (s *recordContentDaoStub) GetAssetContentsByAssetID(_ context.Context, assetID uint) ([]dao.AssetContent, error) {
	result := make([]dao.AssetContent, 0)
	for _, content := range s.contents {
		if content.AssetID == assetID {
			result = append(result, content)
		}
	}
	return result, nil
}

func (s *recordContentDaoStub) DeleteAssetContents(_ context.Context, ids []uint) error {
	for _, id := range ids {
		delete(s.contents, id)
	}
	return nil
}

func TestAssetRepositoryCreatesContentSnapshotAndMovesCurrentPointer(t *testing.T) {
	currentContentID := uint(4)
	currentContent := datatypes.JSON(`{"prototype":[{"id":1,"url":"https://cdn.example/up.png"}]}`)
	assetDao := &recordAssetDaoStub{asset: dao.Asset{
		ID:          7,
		Name:        "hero",
		Description: "main character",
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(`{"width":64,"height":64}`),
		Version:     2,
		ContentID:   &currentContentID,
	}}
	recordDao := &recordDaoStub{
		records: map[uint]dao.AssetRecord{
			1: {ID: 1, AssetID: 7, Version: 2, ContentID: currentContentID},
		},
		nextID: 1,
	}
	contentDao := &recordContentDaoStub{
		contents: map[uint]dao.AssetContent{
			currentContentID: {ID: currentContentID, AssetID: 7, Content: currentContent},
		},
		nextID: currentContentID,
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}

	record, err := repo.CreateRecord(context.Background(), &domain.AssetRecord{AssetID: 7})
	if err != nil {
		t.Fatalf("create record: %v", err)
	}
	if record.ID != 2 || record.AssetID != 7 || record.Version != 3 || record.ContentID != 5 {
		t.Fatalf("unexpected record: %+v", record)
	}
	if string(record.Content) != string(currentContent) {
		t.Fatalf("record content was not copied: %s", record.Content)
	}
	if record.Name != "hero" || record.Description != "main character" || record.Perspective != domain.PerspectiveTopDown || string(record.Dimensions) != `{"width":64,"height":64}` {
		t.Fatalf("record attributes were not copied: %+v", record)
	}
	if assetDao.updatedAsset != 7 || assetDao.updatedVersion != 3 || assetDao.updatedContent != 5 {
		t.Fatalf("asset current pointer was not updated: %+v", assetDao)
	}
}

func TestAssetRepositoryCreateRecordReturnsCurrentContentLookupError(t *testing.T) {
	currentContentID := uint(4)
	wantErr := testingError("content unavailable")
	repo := &repository.AssetRepositoryImpl{
		AssetDao: &recordAssetDaoStub{asset: dao.Asset{
			ID:        7,
			Version:   2,
			ContentID: &currentContentID,
		}},
		ContentDao: &recordContentDaoStub{getErr: wantErr},
		RecordDao:  &recordDaoStub{records: map[uint]dao.AssetRecord{}},
	}

	_, err := repo.CreateRecord(context.Background(), &domain.AssetRecord{AssetID: 7})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected content lookup error %v, got %v", wantErr, err)
	}
}

func TestAssetRepositoryRejectsReplacementContentFromStaleVersion(t *testing.T) {
	currentContentID := uint(5)
	assetDao := &recordAssetDaoStub{asset: dao.Asset{
		ID:        7,
		Version:   3,
		ContentID: &currentContentID,
	}}
	recordDao := &recordDaoStub{
		records: map[uint]dao.AssetRecord{
			1: {ID: 1, AssetID: 7, Version: 3, ContentID: currentContentID},
		},
		nextID: 1,
	}
	contentDao := &recordContentDaoStub{
		contents: map[uint]dao.AssetContent{
			currentContentID: {ID: currentContentID, AssetID: 7, Content: datatypes.JSON(`{"version":3}`)},
		},
		nextID: currentContentID,
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}

	_, err := repo.CreateRecord(context.Background(), &domain.AssetRecord{
		AssetID:         7,
		ExpectedVersion: 2,
		Content:         json.RawMessage(`{"version":2,"prototype":[{"id":1,"url":"stale.png"}]}`),
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	if len(contentDao.contents) != 1 || contentDao.nextID != currentContentID {
		t.Fatalf("stale content was persisted: contents=%+v next_id=%d", contentDao.contents, contentDao.nextID)
	}
	if len(recordDao.records) != 1 || recordDao.nextID != 1 {
		t.Fatalf("stale record was persisted: records=%+v next_id=%d", recordDao.records, recordDao.nextID)
	}
	if assetDao.updatedAsset != 0 || assetDao.updatedVersion != 0 || assetDao.updatedContent != 0 {
		t.Fatalf("stale write moved the asset pointer: %+v", assetDao)
	}
}

func TestAssetRepositoryRejectsReplacementWhenCurrentContentChangedAtSameVersion(t *testing.T) {
	currentContentID := uint(5)
	assetDao := &recordAssetDaoStub{asset: dao.Asset{
		ID:        7,
		Version:   2,
		ContentID: &currentContentID,
	}}
	recordDao := &recordDaoStub{
		records: map[uint]dao.AssetRecord{
			1: {ID: 1, AssetID: 7, Version: 2, ContentID: currentContentID},
		},
		nextID: 1,
	}
	contentDao := &recordContentDaoStub{
		contents: map[uint]dao.AssetContent{
			currentContentID: {
				ID:      currentContentID,
				AssetID: 7,
				Content: datatypes.JSON(`{"version":2,"animations":[{"id":1,"name":"walk"}]}`),
			},
		},
		nextID: currentContentID,
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}

	_, err := repo.CreateRecord(context.Background(), &domain.AssetRecord{
		AssetID:         7,
		ExpectedVersion: 2,
		ExpectedContent: json.RawMessage(`{"version":2,"animations":[]}`),
		Content:         json.RawMessage(`{"version":3,"prototype":[{"id":1,"url":"generated.png"}],"animations":[]}`),
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected content version conflict, got %v", err)
	}
	if len(contentDao.contents) != 1 || contentDao.nextID != currentContentID {
		t.Fatalf("stale generated content was persisted: contents=%+v next_id=%d", contentDao.contents, contentDao.nextID)
	}
	if len(recordDao.records) != 1 || recordDao.nextID != 1 {
		t.Fatalf("stale generated record was persisted: records=%+v next_id=%d", recordDao.records, recordDao.nextID)
	}
	if assetDao.updatedAsset != 0 || assetDao.updatedVersion != 0 || assetDao.updatedContent != 0 {
		t.Fatalf("stale generated write moved the asset pointer: %+v", assetDao)
	}
}

func TestAssetRepositoryCreatesRecordFromReplacementContent(t *testing.T) {
	currentContentID := uint(4)
	assetDao := &recordAssetDaoStub{asset: dao.Asset{
		ID:          7,
		Name:        "hero",
		Description: "main character",
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(`{"width":64,"height":64}`),
		Version:     2,
		ContentID:   &currentContentID,
	}}
	recordDao := &recordDaoStub{
		records: map[uint]dao.AssetRecord{
			1: {ID: 1, AssetID: 7, Version: 2, ContentID: currentContentID},
		},
		nextID: 1,
	}
	contentDao := &recordContentDaoStub{
		contents: map[uint]dao.AssetContent{
			currentContentID: {ID: currentContentID, AssetID: 7, Content: datatypes.JSON(`{"version":2}`)},
		},
		nextID: currentContentID,
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}
	replacement := json.RawMessage(`{"version":3,"prototype":[{"id":1,"url":"new.png"}]}`)

	record, err := repo.CreateRecord(context.Background(), &domain.AssetRecord{
		AssetID:         7,
		ExpectedVersion: 2,
		ExpectedContent: json.RawMessage(`{"version":2}`),
		Content:         replacement,
	})
	if err != nil {
		t.Fatalf("create replacement record: %v", err)
	}
	if record.Version != 3 || record.ContentID != 5 || string(record.Content) != string(replacement) {
		t.Fatalf("unexpected replacement record: %+v", record)
	}
	if got := string(contentDao.contents[record.ContentID].Content); got != string(replacement) {
		t.Fatalf("replacement content was not persisted: %s", got)
	}
	if assetDao.updatedVersion != 3 || assetDao.updatedContent != record.ContentID {
		t.Fatalf("asset current pointer was not moved to replacement: %+v", assetDao)
	}
}

func TestAssetRepositoryRollsBackToContentSnapshot(t *testing.T) {
	targetContentID := uint(3)
	currentContentID := uint(6)
	assetDao := &recordAssetDaoStub{asset: dao.Asset{
		ID:          7,
		Name:        "current hero",
		Description: "current description",
		Perspective: "Isometric",
		Dimensions:  datatypes.JSON(`{"width":128,"height":128}`),
		Version:     4,
		ContentID:   &currentContentID,
	}}
	recordDao := &recordDaoStub{records: map[uint]dao.AssetRecord{
		2: {
			ID:          2,
			AssetID:     7,
			Version:     2,
			ContentID:   targetContentID,
			Name:        "saved hero",
			Description: "saved description",
			Perspective: "Top-Down",
			Dimensions:  datatypes.JSON(`{"width":64,"height":64}`),
		},
		3: {
			ID:        3,
			AssetID:   7,
			Version:   3,
			ContentID: 5,
		},
	}}
	contentDao := &recordContentDaoStub{contents: map[uint]dao.AssetContent{
		targetContentID: {
			ID:      targetContentID,
			AssetID: 7,
			Content: datatypes.JSON(`{"prototype":[{"id":1,"url":"https://cdn.example/up.png"}]}`),
		},
		5: {ID: 5, AssetID: 7, Content: datatypes.JSON(`{"prototype":[{"id":2,"url":"https://cdn.example/new.png"}]}`)},
		6: {ID: 6, AssetID: 7, Content: datatypes.JSON(`{"prototype":[{"id":3,"url":"https://cdn.example/current.png"}]}`)},
	}}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}

	record, err := repo.RollBackRecord(context.Background(), 7, 2)
	if err != nil {
		t.Fatalf("rollback record: %v", err)
	}
	if record == nil || record.Version != 2 || record.ContentID != targetContentID {
		t.Fatalf("unexpected rollback record: %+v", record)
	}
	if assetDao.updatedAsset != 7 || assetDao.updatedVersion != 2 || assetDao.updatedContent != targetContentID {
		t.Fatalf("asset current pointer was not rolled back: %+v", assetDao)
	}
	if assetDao.asset.Name != "saved hero" || assetDao.asset.Description != "saved description" || assetDao.asset.Perspective != "Top-Down" || string(assetDao.asset.Dimensions) != `{"width":64,"height":64}` {
		t.Fatalf("asset attributes were not rolled back: %+v", assetDao.asset)
	}
	if _, ok := recordDao.records[3]; ok {
		t.Fatal("records after rollback target should be deleted")
	}
	if _, ok := contentDao.contents[5]; ok {
		t.Fatal("content for deleted records should be deleted")
	}
	if _, ok := contentDao.contents[6]; ok {
		t.Fatal("unrecorded current content should be deleted")
	}
}

func TestAssetRepositoryGetsRecordHistoryWithContent(t *testing.T) {
	createdAt := testRecordCreatedAt
	recordDao := &recordDaoStub{records: map[uint]dao.AssetRecord{
		1: {ID: 1, AssetID: 7, Version: 1, ContentID: 11, CreatedAt: createdAt},
		2: {ID: 2, AssetID: 7, Version: 2, ContentID: 12, CreatedAt: createdAt.Add(time.Hour)},
	}}
	contentDao := &recordContentDaoStub{contents: map[uint]dao.AssetContent{
		11: {ID: 11, AssetID: 7, Content: datatypes.JSON(`{"version":1}`)},
		12: {ID: 12, AssetID: 7, Content: datatypes.JSON(`{"version":2}`)},
	}}
	repo := &repository.AssetRepositoryImpl{RecordDao: recordDao, ContentDao: contentDao}

	records, err := repo.GetRecordHistory(context.Background(), 7)
	if err != nil {
		t.Fatalf("get record history: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two records, got %d", len(records))
	}
	byVersion := make(map[uint]domain.AssetRecord, len(records))
	for _, record := range records {
		byVersion[record.Version] = record
	}
	if byVersion[1].CreatedAt != createdAt || string(byVersion[2].Content) != `{"version":2}` {
		t.Fatalf("record history did not map content and createdAt: %+v", records)
	}
}

type copyAssetDaoStub struct {
	dao.AssetDao
	source         dao.Asset
	created        *dao.Asset
	updatedAsset   uint
	updatedVersion uint
	updatedContent uint
}

func (s *copyAssetDaoStub) GetAssetForUpdate(_ context.Context, assetID uint) (dao.Asset, error) {
	if assetID != s.source.ID {
		return dao.Asset{}, testingError("source asset not found")
	}
	return s.source, nil
}

func (s *copyAssetDaoStub) CreateAsset(_ context.Context, asset *dao.Asset) (dao.Asset, error) {
	copy := *asset
	copy.ID = 99
	if copy.Version == 0 {
		copy.Version = 1
	}
	s.created = &copy
	return copy, nil
}

func (s *copyAssetDaoStub) UpdateAssetCurrentContent(_ context.Context, assetID uint, version uint, contentID uint) error {
	s.updatedAsset = assetID
	s.updatedVersion = version
	s.updatedContent = contentID
	return nil
}

func TestAssetRepositoryCopiesAssetWithAllRecordsAndContents(t *testing.T) {
	firstCreatedAt := testRecordCreatedAt
	secondCreatedAt := firstCreatedAt.Add(time.Hour)
	currentContentID := uint(12)
	assetDao := &copyAssetDaoStub{source: dao.Asset{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        "character",
		Description: "main character",
		Tags:        []string{"hero", "player"},
		Perspective: "Side-On",
		Dimensions:  []byte(`{"width":64,"height":64}`),
		ContentID:   &currentContentID,
		Version:     2,
	}}
	recordDao := &recordDaoStub{
		records: map[uint]dao.AssetRecord{
			1: {ID: 1, AssetID: 7, Version: 1, ContentID: 10, CreatedAt: firstCreatedAt},
			2: {ID: 2, AssetID: 7, Version: 2, ContentID: 12, CreatedAt: secondCreatedAt},
		},
		nextID: 2,
	}
	contentDao := &recordContentDaoStub{
		contents: map[uint]dao.AssetContent{
			10: {ID: 10, AssetID: 7, Content: datatypes.JSON(`{"version":1}`)},
			11: {ID: 11, AssetID: 7, Content: datatypes.JSON(`{"temporary":true}`)},
			12: {ID: 12, AssetID: 7, Content: datatypes.JSON(`{"version":2}`)},
		},
		nextID: 12,
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: assetDao, ContentDao: contentDao, RecordDao: recordDao}

	newAssetID, err := repo.Copy(context.Background(), 7, 0)
	if err != nil {
		t.Fatalf("copy asset: %v", err)
	}
	if newAssetID == 7 || newAssetID != 99 {
		t.Fatalf("expected a new asset ID, got %d", newAssetID)
	}
	if assetDao.created == nil {
		t.Fatal("expected copied asset to be created")
	}
	if assetDao.created.Name != assetDao.source.Name ||
		assetDao.created.ProjectID != assetDao.source.ProjectID ||
		assetDao.created.Type != assetDao.source.Type ||
		assetDao.created.Description != assetDao.source.Description ||
		assetDao.created.Version != assetDao.source.Version {
		t.Fatalf("copied asset metadata was not preserved: %+v", assetDao.created)
	}

	copiedContents := make(map[string]dao.AssetContent)
	for _, content := range contentDao.contents {
		if content.AssetID == newAssetID {
			copiedContents[string(content.Content)] = content
		}
	}
	if len(copiedContents) != 3 {
		t.Fatalf("expected all three contents to be copied, got %d", len(copiedContents))
	}

	copiedRecords := make([]dao.AssetRecord, 0)
	for _, record := range recordDao.records {
		if record.AssetID == newAssetID {
			copiedRecords = append(copiedRecords, record)
		}
	}
	if len(copiedRecords) != 2 {
		t.Fatalf("expected both records to be copied, got %d", len(copiedRecords))
	}
	for _, record := range copiedRecords {
		if record.ID == 1 || record.ID == 2 || record.AssetID != newAssetID {
			t.Fatalf("record ID or asset ID was reused: %+v", record)
		}
		content, ok := contentDao.contents[record.ContentID]
		if !ok || content.AssetID != newAssetID {
			t.Fatalf("record does not point to copied content: %+v", record)
		}
		if record.Version == 1 && !record.CreatedAt.Equal(firstCreatedAt) {
			t.Fatalf("version 1 timestamp was not preserved: %+v", record)
		}
		if record.Version == 2 && !record.CreatedAt.Equal(secondCreatedAt) {
			t.Fatalf("version 2 timestamp was not preserved: %+v", record)
		}
	}
	if assetDao.updatedAsset != newAssetID || assetDao.updatedVersion != 2 {
		t.Fatalf("copied asset current pointer was not updated: %+v", assetDao)
	}
	currentContent, ok := contentDao.contents[assetDao.updatedContent]
	if !ok || string(currentContent.Content) != `{"version":2}` || currentContent.AssetID != newAssetID {
		t.Fatalf("current content pointer does not map to source current content: %+v", assetDao)
	}

	if contentDao.contents[10].AssetID != 7 || contentDao.contents[12].AssetID != 7 {
		t.Fatal("source contents were modified")
	}
}

type testingError string

func (e testingError) Error() string {
	return string(e)
}
