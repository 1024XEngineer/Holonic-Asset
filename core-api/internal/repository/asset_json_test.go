package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/datatypes"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type jsonAssetDaoStub struct {
	dao.AssetDao
	asset            dao.Asset
	created          dao.Asset
	updatedAsset     uint
	updatedVersion   uint
	updatedContent   uint
	updatedThumbnail string
}

type jsonAssetRecordDaoStub struct {
	dao.AssetRecordDao
	records map[uint]dao.AssetRecord
	nextID  uint
}

func (s *jsonAssetRecordDaoStub) CreateAssetRecord(_ context.Context, record *dao.AssetRecord) (uint, error) {
	if record.ID == 0 {
		s.nextID++
		record.ID = s.nextID
	}
	s.records[record.ID] = *record
	return record.ID, nil
}

type jsonAssetContentDaoStub struct {
	dao.AssetContentDao
	contents map[uint]dao.AssetContent
	nextID   uint
}

func (s *jsonAssetContentDaoStub) CreateAssetContent(_ context.Context, content *dao.AssetContent) (dao.AssetContent, error) {
	if content.ID == 0 {
		s.nextID++
		content.ID = s.nextID
	}
	s.contents[content.ID] = *content
	return *content, nil
}

func (s *jsonAssetContentDaoStub) GetAssetContent(_ context.Context, id uint) (dao.AssetContent, error) {
	return s.contents[id], nil
}

func (s *jsonAssetContentDaoStub) DeleteAssetContents(_ context.Context, ids []uint) error {
	for _, id := range ids {
		delete(s.contents, id)
	}
	return nil
}

func (s *jsonAssetDaoStub) GetAssetDetail(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *jsonAssetDaoStub) GetAsset(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *jsonAssetDaoStub) CreateAsset(_ context.Context, asset *dao.Asset) (dao.Asset, error) {
	s.created = *asset
	s.created.ID = 23
	return s.created, nil
}

func (s *jsonAssetDaoStub) UpdateAssetCurrentContent(_ context.Context, assetID uint, version uint, contentID uint, thumbnailURL string) error {
	s.updatedAsset = assetID
	s.updatedVersion = version
	s.updatedContent = contentID
	s.updatedThumbnail = thumbnailURL
	return nil
}

func TestAssetRepositoryReadsAssetContent(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 4
	prototype := domain.Prototype{
		{ID: 2101, URL: new("https://cdn.example/prototype-01.png")},
		{ID: 2102, URL: new("https://cdn.example/prototype-02.png")},
		{ID: 2103, URL: new("https://cdn.example/prototype-03.png")},
		{ID: 2104, URL: new("https://cdn.example/prototype-04.png")},
	}
	content.Prototype = &prototype
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}

	contentID := uint(11)
	repo := &repository.AssetRepositoryImpl{
		AssetDao: &jsonAssetDaoStub{asset: dao.Asset{
			ID:        7,
			Version:   2,
			Type:      string(domain.AssetTypeCharacter),
			ContentID: &contentID,
		}},
		ContentDao: &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
			contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
		}},
	}
	asset, err := repo.GetAssetDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("get asset: %v", err)
	}
	decoded, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode asset: %v", err)
	}
	if decoded.Prototype == nil || len(*decoded.Prototype) != 4 || (*decoded.Prototype)[0].ID != 2101 || (*decoded.Prototype)[0].URL == nil || *(*decoded.Prototype)[0].URL != "https://cdn.example/prototype-01.png" {
		t.Fatalf("unexpected asset content: %+v", decoded)
	}
}

func TestAssetRepositoryReturnsAssetContentWithoutGenerationState(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}

	repo := &repository.AssetRepositoryImpl{AssetDao: &jsonAssetDaoStub{asset: dao.Asset{
		ID:      7,
		Version: 1,
		Type:    string(domain.AssetTypeCharacter),
		Content: datatypes.JSON(payload),
	}}}
	asset, err := repo.GetAssetDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("get asset: %v", err)
	}
	decoded, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode asset: %v", err)
	}
	if decoded.Prototype == nil {
		t.Fatalf("unexpected asset content: %+v", decoded)
	}
}

func TestAssetRepositoryCreatesCharacterWithPrototype(t *testing.T) {
	daoStub := &jsonAssetDaoStub{}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{}, nextID: 10}
	recordDao := &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub, ContentDao: contentDao, RecordDao: recordDao}
	prototypeURL := "uploads/hero/prototype.png"
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	prototype := domain.Prototype{{ID: 1, URL: &prototypeURL}}
	content.Prototype = &prototype
	encoded, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode character content: %v", err)
	}

	created, err := repo.CreateCharacterAsset(context.Background(), &domain.Asset{
		Name:        "hero",
		ProjectID:   42,
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Content:     encoded,
	})
	if err != nil {
		t.Fatalf("create character asset: %v", err)
	}
	if created == nil || created.ID != 23 {
		t.Fatalf("expected created asset ID 23, got %+v", created)
	}
	if created.ThumbnailURL != prototypeURL || daoStub.created.ThumbnailURL != prototypeURL || daoStub.updatedThumbnail != prototypeURL {
		t.Fatalf("prototype thumbnail was not stored: created=%+v dao=%+v", created, daoStub)
	}
	content, err = (&domain.Asset{Content: []byte(daoStub.created.Content), Type: domain.AssetTypeCharacter}).DecodeContent()
	if err != nil {
		t.Fatalf("decode created content: %v", err)
	}
	if content.Prototype == nil {
		t.Fatalf("expected prototype: %+v", content)
	}
}
