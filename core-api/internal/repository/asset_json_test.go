package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
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
	err     error
}

func (s *jsonAssetRecordDaoStub) CreateAssetRecord(_ context.Context, record *dao.AssetRecord) (uint, error) {
	if s.err != nil {
		return 0, s.err
	}
	if record.ID == 0 {
		s.nextID++
		record.ID = s.nextID
	}
	s.records[record.ID] = *record
	return record.ID, nil
}

func TestAssetRepositoryDoesNotAdvanceContentWhenRecordCreationFails(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID: 7, Type: string(domain.AssetTypeCharacter), Version: 2,
		ContentID: &contentID, Content: datatypes.JSON(payload),
	}}
	wantErr := errors.New("record write failed")
	repo := &repository.AssetRepositoryImpl{
		AssetDao: daoStub,
		ContentDao: &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
			contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
		}},
		RecordDao: &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, err: wantErr},
	}

	_, err = repo.CreateAnimation(context.Background(), 7, domain.Animation{Name: "idle"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected record creation error, got %v", err)
	}
	if daoStub.updatedAsset != 0 || daoStub.updatedVersion != 0 || daoStub.updatedContent != 0 {
		t.Fatalf("asset advanced after record failure: %+v", daoStub)
	}
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

func TestAssetRepositoryUpdatesAnimationFrames(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	generation := &domain.AnimationGenerationConfig{
		Direction:   "back_right",
		Style:       "painted pixel art",
		Action:      "raise the hat in greeting",
		FrameCount:  8,
		Columns:     4,
		FrameWidth:  128,
		FrameHeight: 128,
		FPS:         12,
		Resolution:  "1080p",
		Duration:    8,
		AspectRatio: "1:1",
	}
	content.Animations = []domain.Animation{{
		ID:         9,
		Name:       "walk",
		Frames:     []domain.Frame{},
		Generation: generation,
	}}
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID:          7,
		Name:        "hero",
		Description: "main character",
		Type:        string(domain.AssetTypeCharacter),
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(`{"width":64,"height":64}`),
		Version:     2,
		ContentID:   &contentID,
		Content:     datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	recordDao := &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}

	err = repo.UpdateAnimationFrames(
		context.Background(),
		7,
		9,
		[]domain.Frame{{ID: 2201, URL: new("https://cdn.example/walk-01.png")}},
	)
	if err != nil {
		t.Fatalf("update animation frames: %v", err)
	}
	if daoStub.updatedAsset != 7 || daoStub.updatedContent == contentID {
		t.Fatalf("unexpected content pointer update: %+v", daoStub)
	}
	updated, err := (&domain.Asset{Content: json.RawMessage(contentDao.contents[daoStub.updatedContent].Content)}).DecodeContent()
	if err != nil {
		t.Fatalf("decode updated content: %v", err)
	}
	animation := updated.Animations[0]
	if animation.Name != "walk" || !reflect.DeepEqual(animation.Generation, generation) ||
		len(animation.Frames) != 1 || animation.Frames[0].ID != 2201 || animation.Frames[0].URL == nil {
		t.Fatalf("unexpected animation content: %+v", animation)
	}
	if daoStub.updatedVersion != 3 {
		t.Fatalf("asset version = %d, want 3", daoStub.updatedVersion)
	}
	record, ok := recordDao.records[21]
	if !ok || record.Version != 3 || record.ContentID != daoStub.updatedContent {
		t.Fatalf("unexpected animation frame content record: %+v", recordDao.records)
	}
}

func TestAssetRepositoryCreatesAnimationInsideAssetContent(t *testing.T) {
	content := domain.NewAssetContent(domain.AssetTypeCharacter)
	content.DirectionCount = 4
	content.Metadata = map[string]any{"source": "prototype"}
	prototypeURL := "uploads/hero/prototype.png"
	prototype := domain.Prototype{{ID: 21, URL: &prototypeURL}}
	content.Prototype = &prototype
	payload, err := domain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode content: %v", err)
	}
	contentID := uint(11)
	daoStub := &jsonAssetDaoStub{asset: dao.Asset{
		ID:          7,
		Name:        "hero",
		Description: "main character",
		Type:        string(domain.AssetTypeCharacter),
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(`{"width":64,"height":64}`),
		Version:     2,
		ContentID:   &contentID,
		Content:     datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	recordDao := &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}

	frameURL := "uploads/hero/walk/001.png"
	generation := &domain.AnimationGenerationConfig{
		Direction:   "front",
		Style:       "painted pixel art",
		Action:      "walking cycle",
		FrameCount:  8,
		Columns:     4,
		FrameWidth:  128,
		FrameHeight: 128,
		FPS:         12,
		Resolution:  "1080p",
		Duration:    8,
		AspectRatio: "1:1",
	}
	animationID, err := repo.CreateAnimation(
		context.Background(),
		7,
		domain.Animation{
			Name:       "walk",
			Frames:     []domain.Frame{{ID: 1, URL: &frameURL, Duration: 100}},
			Generation: generation,
		},
	)
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
	if len(updated.Animations) != 1 || updated.Animations[0].Name != "walk" ||
		len(updated.Animations[0].Frames) != 1 || updated.Animations[0].Frames[0].URL == nil ||
		*updated.Animations[0].Frames[0].URL != frameURL ||
		!reflect.DeepEqual(updated.Animations[0].Generation, generation) {
		t.Fatalf("unexpected animation content: %+v", updated.Animations)
	}
	if updated.DirectionCount != content.DirectionCount || updated.Prototype == nil ||
		len(*updated.Prototype) != 1 || (*updated.Prototype)[0].URL == nil ||
		*(*updated.Prototype)[0].URL != prototypeURL || updated.Metadata["source"] != "prototype" {
		t.Fatalf("non-animation content changed: %+v", updated)
	}
	if len(contentDao.contents) != 2 {
		t.Fatalf("expected one content write, got %d content records", len(contentDao.contents))
	}
	if daoStub.updatedVersion != 3 {
		t.Fatalf("asset version = %d, want 3", daoStub.updatedVersion)
	}
	if daoStub.updatedThumbnail != prototypeURL {
		t.Fatalf("asset thumbnail = %q, want %q", daoStub.updatedThumbnail, prototypeURL)
	}
	record, ok := recordDao.records[21]
	if !ok {
		t.Fatalf("expected animation content record, got %+v", recordDao.records)
	}
	if record.AssetID != 7 || record.Version != 3 || record.ContentID != daoStub.updatedContent ||
		record.Name != "hero" || record.Description != "main character" ||
		record.Perspective != "Top-Down" || string(record.Dimensions) != `{"width":64,"height":64}` {
		t.Fatalf("unexpected animation content record: %+v", record)
	}
}

func TestAssetRepositoryUpdatesPrototypeImages(t *testing.T) {
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
		Version:   2,
		ContentID: &contentID,
		Content:   datatypes.JSON(payload),
	}}
	contentDao := &jsonAssetContentDaoStub{contents: map[uint]dao.AssetContent{
		contentID: {ID: contentID, AssetID: 7, Content: datatypes.JSON(payload)},
	}}
	recordDao := &jsonAssetRecordDaoStub{records: map[uint]dao.AssetRecord{}, nextID: 20}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   daoStub,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}

	err = repo.UpdatePrototypeImages(context.Background(), 7, []domain.ImageResource{
		{ID: 2101, URL: new("  ")},
		{ID: 2102, URL: new("https://cdn.example/prototype-02.png")},
		{ID: 2103, URL: new("https://cdn.example/prototype-03.png")},
		{ID: 2104, URL: new("https://cdn.example/prototype-04.png")},
	})
	if err != nil {
		t.Fatalf("update prototype images: %v", err)
	}
	updated, err := (&domain.Asset{Content: json.RawMessage(contentDao.contents[daoStub.updatedContent].Content)}).DecodeContent()
	if err != nil {
		t.Fatalf("decode updated content: %v", err)
	}
	prototype := updated.Prototype
	if prototype == nil || len(*prototype) != 4 {
		t.Fatalf("unexpected prototype content: %+v", prototype)
	}
	for index, image := range *prototype {
		if image.ID != uint(2101+index) || image.URL == nil {
			t.Fatalf("unexpected prototype image at index %d: %+v", index, image)
		}
	}
	if daoStub.updatedVersion != 3 {
		t.Fatalf("asset version = %d, want 3", daoStub.updatedVersion)
	}
	if daoStub.updatedThumbnail != "https://cdn.example/prototype-02.png" {
		t.Fatalf("asset thumbnail = %q, want first non-blank prototype URL", daoStub.updatedThumbnail)
	}
	record, ok := recordDao.records[21]
	if !ok || record.Version != 3 || record.ContentID != daoStub.updatedContent {
		t.Fatalf("unexpected prototype content record: %+v", recordDao.records)
	}
}
