package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/datatypes"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type jsonAssetDaoStub struct {
	dao.AssetDao
	asset          dao.Asset
	created        dao.Asset
	updatedAsset   uint
	updatedVersion uint
	updatedContent uint
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

func (s *jsonAssetDaoStub) GetAssetForUpdate(_ context.Context, _ uint) (dao.Asset, error) {
	return s.asset, nil
}

func (s *jsonAssetDaoStub) CreateAsset(_ context.Context, asset *dao.Asset) (dao.Asset, error) {
	s.created = *asset
	s.created.ID = 23
	return s.created, nil
}

func (s *jsonAssetDaoStub) UpdateAssetCurrentContent(_ context.Context, assetID uint, version uint, contentID uint) error {
	s.updatedAsset = assetID
	s.updatedVersion = version
	s.updatedContent = contentID
	return nil
}

func TestAssetRepositoryReadsAssetContent(t *testing.T) {
	upURL := "https://cdn.example/prototype-up.png"
	downURL := "https://cdn.example/prototype-down.png"
	leftURL := "https://cdn.example/prototype-left.png"
	rightURL := "https://cdn.example/prototype-right.png"
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 4
	content.Prototype.Directions = map[string]domain.PrototypeDirection{
		"up": {
			Image: &domain.ImageResource{URL: &upURL},
		},
		"down": {
			Image: &domain.ImageResource{URL: &downURL},
		},
		"left": {
			Image: &domain.ImageResource{URL: &leftURL},
		},
		"right": {
			Image: &domain.ImageResource{URL: &rightURL},
		},
	}
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
	if decoded.Prototype == nil || len(decoded.Prototype.Directions) != 4 || decoded.Prototype.Directions["up"].Image == nil || decoded.Prototype.Directions["up"].Image.URL == nil || *decoded.Prototype.Directions["up"].Image.URL != "https://cdn.example/prototype-up.png" {
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

	created, err := repo.CreateCharacterAsset(context.Background(), &domain.Asset{
		Name:      "hero",
		ProjectID: 42,
	})
	if err != nil {
		t.Fatalf("create character asset: %v", err)
	}
	if created == nil || created.ID != 23 {
		t.Fatalf("expected created asset ID 23, got %+v", created)
	}
	content, err := (&domain.Asset{Content: []byte(daoStub.created.Content), Type: domain.AssetTypeCharacter}).DecodeContent()
	if err != nil {
		t.Fatalf("decode created content: %v", err)
	}
	if content.Prototype == nil {
		t.Fatalf("expected prototype: %+v", content)
	}
}

func TestAssetRepositoryUpdatesDynamicAnimationDirection(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.Animations = []domain.Animation{{
		ID:         9,
		Name:       "walk",
		Directions: map[string]domain.AnimationDirection{"left": {}},
	}}
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID:        7,
		Type:      string(domain.AssetTypeCharacter),
		ContentID: &contentID,
		Content:   datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20},
	}

	err = repo.UpdateAnimationDirection(
		context.Background(),
		7,
		9,
		"right",
		[]domain.Frame{{URL: new("https://cdn.example/walk-right-1.png")}},
	)
	if err != nil {
		t.Fatalf("update animation direction: %v", err)
	}
	if daoStub.updatedAsset != 7 || daoStub.updatedContent == contentID {
		t.Fatalf("unexpected content pointer update: %+v", daoStub)
	}
	updated, err := (&domain.Asset{Content: json.RawMessage(contentDao.contents[daoStub.updatedContent].Content)}).DecodeContent()
	if err != nil {
		t.Fatalf("decode updated content: %v", err)
	}
	animation := updated.Animations[0]
	if len(animation.Directions["right"].Frames) != 1 || animation.Directions["right"].Frames[0].URL == nil {
		t.Fatalf("unexpected animation content: %+v", animation)
	}
}

func TestAssetRepositoryCreatesAnimationInsideAssetContent(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID:        7,
		Type:      string(domain.AssetTypeCharacter),
		ContentID: &contentID,
		Content:   datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20},
	}

	animationID, err := repo.CreateAnimation(context.Background(), 7, "walk")
	if err != nil {
		t.Fatalf("create animation: %v", err)
	}
	if animationID != 1 {
		t.Fatalf("expected animation ID 1, got %d", animationID)
	}
	updated, err := (&domain.Asset{Content: json.RawMessage(contentDao.contents[daoStub.updatedContent].Content)}).DecodeContent()
	if err != nil {
		t.Fatalf("decode updated content: %v", err)
	}
	if len(updated.Animations) != 1 || updated.Animations[0].Name != "walk" {
		t.Fatalf("unexpected animation content: %+v", updated.Animations)
	}
}

func TestAssetRepositoryUpdatesOnePrototypeImagePerDirection(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 4
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID:        7,
		Type:      string(domain.AssetTypeCharacter),
		ContentID: &contentID,
		Content:   datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20},
	}

	err = repo.UpdatePrototypeImages(context.Background(), 7, map[string]domain.ImageResource{
		"up":    {URL: new("https://cdn.example/prototype-up.png")},
		"down":  {URL: new("https://cdn.example/prototype-down.png")},
		"left":  {URL: new("https://cdn.example/prototype-left.png")},
		"right": {URL: new("https://cdn.example/prototype-right.png")},
	})
	if err != nil {
		t.Fatalf("update prototype images: %v", err)
	}
	updated, err := (&domain.Asset{Content: json.RawMessage(contentDao.contents[daoStub.updatedContent].Content)}).DecodeContent()
	if err != nil {
		t.Fatalf("decode updated content: %v", err)
	}
	prototype := updated.Prototype
	if prototype == nil || len(prototype.Directions) != 4 {
		t.Fatalf("unexpected prototype content: %+v", prototype)
	}
	for _, direction := range domain.DirectionsForCount(content.DirectionCount) {
		if prototype.Directions[direction].Image == nil {
			t.Fatalf("expected prototype image for direction %q", direction)
		}
	}
}
