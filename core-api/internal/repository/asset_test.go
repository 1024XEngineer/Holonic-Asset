package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/datatypes"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type assetDaoStub struct {
	dao.AssetDao
	assets       []dao.Asset
	asset        dao.Asset
	getAssetsErr error
	getDetailErr error
	updatedAsset dao.Asset
	updateErr    error
	projectID    uint
	assetID      uint
	updateID     uint
	update       *dao.AssetUpdate
}

func (s *assetDaoStub) GetAssetsByProjectID(_ context.Context, projectID uint) ([]dao.Asset, error) {
	s.projectID = projectID
	return s.assets, s.getAssetsErr
}

func (s *assetDaoStub) GetAssetDetail(_ context.Context, assetID uint) (dao.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetDaoStub) GetAsset(_ context.Context, assetID uint) (dao.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetDaoStub) GetAssetForUpdate(_ context.Context, assetID uint) (dao.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetDaoStub) UpdateAsset(_ context.Context, assetID uint, update *dao.AssetUpdate) (dao.Asset, error) {
	s.updateID = assetID
	s.update = update
	return s.updatedAsset, s.updateErr
}

func (s *assetDaoStub) CreateAsset(_ context.Context, asset *dao.Asset) (dao.Asset, error) {
	if asset.ID == 0 {
		asset.ID = 100
	}
	s.asset = *asset
	return *asset, nil
}

func (s *assetDaoStub) DeleteAsset(_ context.Context, assetID uint) error {
	s.assetID = assetID
	return nil
}

func (s *assetDaoStub) UpdateAssetCurrentContent(_ context.Context, _ uint, _ uint, _ uint, _ string) error {
	return nil
}

type assetContentDaoStub struct {
	dao.AssetContentDao
	deletedAssetID uint
	deleteErr      error
}

func (s *assetContentDaoStub) DeleteAssetContentsByAssetID(_ context.Context, assetID uint) error {
	s.deletedAssetID = assetID
	return s.deleteErr
}

func (s *assetContentDaoStub) CreateAssetContent(_ context.Context, c *dao.AssetContent) (dao.AssetContent, error) {
	c.ID = 1
	return *c, nil
}

type assetRecordDaoStub struct {
	dao.AssetRecordDao
	deletedAssetID uint
	deleteErr      error
}

func (s *assetRecordDaoStub) DeleteAssetRecordsByAssetID(_ context.Context, assetID uint) error {
	s.deletedAssetID = assetID
	return s.deleteErr
}

func (s *assetRecordDaoStub) CreateAssetRecord(_ context.Context, r *dao.AssetRecord) (uint, error) {
	r.ID = 1
	return 1, nil
}

func TestAssetRepositoryGetAssetsMapsDAOResults(t *testing.T) {
	dimensions := json.RawMessage(`{"width":128,"height":128}`)
	daoStub := &assetDaoStub{assets: []dao.Asset{{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        "character",
		Description: "main character",
		Tags:        []domain.Tag{{Name: "player"}, {Name: "hero", Description: "main role"}},
		Perspective: "Top-Down",
		Dimensions:  datatypes.JSON(dimensions),
		Version:     3,
	}}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.GetAssetsByProjectID(context.Background(), 42, domain.AssetListFilter{})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if daoStub.projectID != 42 {
		t.Fatalf("expected project ID 42, got %d", daoStub.projectID)
	}
	if len(got) != 1 {
		t.Fatalf("expected one asset, got %d", len(got))
	}
	if got[0].ID != 7 || got[0].Type != domain.AssetTypeCharacter || got[0].Version != 3 {
		t.Fatalf("unexpected mapped asset: %+v", got[0])
	}
	if string(got[0].Dimensions) != string(dimensions) || got[0].Perspective != domain.PerspectiveTopDown || len(got[0].Tags) != 2 {
		t.Fatalf("asset data was not mapped: %+v", got[0])
	}
}

func TestAssetRepositoryFiltersAssetsByAllTagsAndTypes(t *testing.T) {
	daoStub := &assetDaoStub{assets: []dao.Asset{
		{ID: 1, ProjectID: 42, Name: "hero", Type: "character", Tags: []domain.Tag{{Name: "hero"}, {Name: "player"}}},
		{ID: 2, ProjectID: 42, Type: "object", Tags: []domain.Tag{{Name: "hero"}, {Name: "prop"}}},
		{ID: 3, ProjectID: 42, Type: "character", Tags: []domain.Tag{{Name: "npc"}}},
	}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.GetAssetsByProjectID(context.Background(), 42, domain.AssetListFilter{
		Query: "hero",
		Tags:  []string{"hero", "player"},
		Types: []domain.AssetType{domain.AssetTypeCharacter},
	})
	if err != nil {
		t.Fatalf("filter assets: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unexpected filtered assets: %+v", got)
	}
}

func TestAssetRepositoryMatchesAssetQueryByNameOrDescription(t *testing.T) {
	daoStub := &assetDaoStub{assets: []dao.Asset{
		{ID: 1, ProjectID: 42, Name: "Hero Knight"},
		{ID: 2, ProjectID: 42, Description: "A forest prop"},
		{ID: 3, ProjectID: 42, Name: "Enemy"},
	}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.GetAssetsByProjectID(context.Background(), 42, domain.AssetListFilter{Query: "forest"})
	if err != nil {
		t.Fatalf("filter assets by query: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("unexpected query results: %+v", got)
	}
}

func TestAssetRepositoryMatchesAssetQueryByTagAttributes(t *testing.T) {
	daoStub := &assetDaoStub{assets: []dao.Asset{
		{ID: 1, ProjectID: 42, Tags: []domain.Tag{{Name: "knight", Description: "heavy armored guardian", Color: "#123456"}}},
		{ID: 2, ProjectID: 42, Tags: []domain.Tag{{Name: "villager"}}},
	}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	for _, query := range []string{"KNIGHT", "armored", "#123456"} {
		got, err := repo.GetAssetsByProjectID(context.Background(), 42, domain.AssetListFilter{Query: query})
		if err != nil {
			t.Fatalf("filter assets by tag query %q: %v", query, err)
		}
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("unexpected query results for %q: %+v", query, got)
		}
	}
}

func TestAssetRepositoryUpdatesAssetBasics(t *testing.T) {
	name := "updated hero"
	projectID := uint(42)
	typeValue := domain.AssetTypeCharacter
	description := "updated description"
	tags := []domain.Tag{{Name: "prop", Description: "scene object", Color: "#123456"}}
	perspective := domain.PerspectiveSideOn
	dimensions := json.RawMessage(`{"width":64,"height":64}`)
	version := uint(4)
	daoStub := &assetDaoStub{updatedAsset: dao.Asset{
		ID:          7,
		Name:        name,
		ProjectID:   projectID,
		Type:        string(typeValue),
		Description: description,
		Tags:        tags,
		Perspective: string(perspective),
		Dimensions:  datatypes.JSON(dimensions),
		Version:     version,
	}}
	daoStub.asset = dao.Asset{ID: 7, Type: string(typeValue)}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.UpdateAsset(context.Background(), 7, &domain.AssetUpdate{
		Name:        &name,
		Description: &description,
		Tags:        &tags,
		Perspective: &perspective,
		Dimensions:  &dimensions,
	})
	if err != nil {
		t.Fatalf("update asset basics: %v", err)
	}
	if daoStub.updateID != 7 || daoStub.update == nil || daoStub.update.Perspective == nil || *daoStub.update.Perspective != string(perspective) {
		t.Fatalf("unexpected DAO update: %+v", daoStub.update)
	}
	if got == nil || got.Name != name || got.ProjectID != projectID || got.Type != typeValue || string(got.Dimensions) != string(dimensions) {
		t.Fatalf("unexpected updated asset: %+v", got)
	}
}

func TestAssetRepositoryRejectsInvalidDimensionsBeforeUpdate(t *testing.T) {
	dimensions := json.RawMessage(`{"width":64,"height":64,"unexpected":true}`)
	daoStub := &assetDaoStub{asset: dao.Asset{ID: 7, Type: "character"}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	if _, err := repo.UpdateAsset(context.Background(), 7, &domain.AssetUpdate{Dimensions: &dimensions}); err == nil {
		t.Fatal("expected invalid dimensions to be rejected")
	}
	if daoStub.update != nil {
		t.Fatalf("invalid dimensions reached DAO update: %+v", daoStub.update)
	}
}

func TestAssetRepositoryIgnoresAudioVisualMetadataOnUpdate(t *testing.T) {
	perspective := domain.PerspectiveTopDown
	dimensions := json.RawMessage(`{"width":64,"height":64}`)
	daoStub := &assetDaoStub{
		asset:        dao.Asset{ID: 7, Type: string(domain.AssetTypeAudio)},
		updatedAsset: dao.Asset{ID: 7, Type: string(domain.AssetTypeAudio)},
	}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	if _, err := repo.UpdateAsset(context.Background(), 7, &domain.AssetUpdate{
		Perspective: &perspective,
		Dimensions:  &dimensions,
	}); err != nil {
		t.Fatalf("update audio asset: %v", err)
	}
	if daoStub.update == nil {
		t.Fatal("expected DAO update")
	}
	if daoStub.update.Perspective != nil || daoStub.update.Dimensions != nil {
		t.Fatalf("audio visual metadata reached DAO update: %+v", daoStub.update)
	}
}

func TestAssetRepositoryGetDetailMapsDAOResult(t *testing.T) {
	daoStub := &assetDaoStub{asset: dao.Asset{
		ID:        7,
		ProjectID: 42,
		Type:      "object",
		Tags:      []domain.Tag{{Name: "prop"}},
	}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.GetAssetDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if daoStub.assetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", daoStub.assetID)
	}
	if got == nil || got.ID != 7 || got.ProjectID != 42 || got.Type != domain.AssetTypeObject {
		t.Fatalf("unexpected mapped detail: %+v", got)
	}
}

func TestAssetRepositoryPropagatesDAOErrors(t *testing.T) {
	wantErr := errors.New("asset lookup failed")
	repo := &repository.AssetRepositoryImpl{AssetDao: &assetDaoStub{getDetailErr: wantErr}}

	_, err := repo.GetAssetDetail(context.Background(), 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetRepositoryDelete(t *testing.T) {
	// Missing ContentDao or RecordDao returns error
	repoMissing := &repository.AssetRepositoryImpl{
		AssetDao: &assetDaoStub{},
	}
	if err := repoMissing.Delete(context.Background(), 10); err == nil {
		t.Fatal("expected error on missing daos")
	}

	// Success
	assetDao := &assetDaoStub{}
	contentDao := &assetContentDaoStub{}
	recordDao := &assetRecordDaoStub{}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   assetDao,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}

	if err := repo.Delete(context.Background(), 42); err != nil {
		t.Fatalf("delete asset: %v", err)
	}
	if assetDao.assetID != 42 || contentDao.deletedAssetID != 42 || recordDao.deletedAssetID != 42 {
		t.Fatalf("expected assetID 42 to be deleted across all DAOs")
	}

	// GetAssetForUpdate error
	wantErr := errors.New("lock failed")
	repoLockErr := &repository.AssetRepositoryImpl{
		AssetDao:   &assetDaoStub{getDetailErr: wantErr},
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}
	if err := repoLockErr.Delete(context.Background(), 42); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}

	// RecordDao delete error
	repoRecordErr := &repository.AssetRepositoryImpl{
		AssetDao:   assetDao,
		ContentDao: contentDao,
		RecordDao:  &assetRecordDaoStub{deleteErr: wantErr},
	}
	if err := repoRecordErr.Delete(context.Background(), 42); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}

	// ContentDao delete error
	repoContentErr := &repository.AssetRepositoryImpl{
		AssetDao:   assetDao,
		ContentDao: &assetContentDaoStub{deleteErr: wantErr},
		RecordDao:  recordDao,
	}
	if err := repoContentErr.Delete(context.Background(), 42); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestAssetRepositoryCreateSpecificAssetTypes(t *testing.T) {
	assetDao := &assetDaoStub{}
	contentDao := &assetContentDaoStub{}
	recordDao := &assetRecordDaoStub{}
	repo := &repository.AssetRepositoryImpl{
		AssetDao:   assetDao,
		ContentDao: contentDao,
		RecordDao:  recordDao,
	}

	ctx := context.Background()

	// Nil asset error
	if _, err := repo.CreateObjectAsset(ctx, nil); err == nil {
		t.Fatal("expected error on nil asset")
	}
	if _, err := repo.CreateTileSetAsset(ctx, nil); err == nil {
		t.Fatal("expected error on nil asset")
	}
	if _, err := repo.CreateUISetAsset(ctx, nil); err == nil {
		t.Fatal("expected error on nil asset")
	}
	if _, err := repo.CreateSceneryAsset(ctx, nil); err == nil {
		t.Fatal("expected error on nil asset")
	}

	dim := json.RawMessage(`{"width":64,"height":64}`)
	tileDim := json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":10,"rows":8}}`)

	// CreateObjectAsset
	id, err := repo.CreateObjectAsset(ctx, &domain.Asset{
		Name:        "Chest",
		ProjectID:   1,
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  dim,
	})
	if err != nil {
		t.Fatalf("create object asset: %v", err)
	}
	if id != 100 || assetDao.asset.Type != string(domain.AssetTypeObject) {
		t.Fatalf("unexpected object asset: id=%d, type=%s", id, assetDao.asset.Type)
	}

	// CreateTileSetAsset
	id, err = repo.CreateTileSetAsset(ctx, &domain.Asset{
		Name:        "GrassTiles",
		ProjectID:   1,
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  tileDim,
	})
	if err != nil {
		t.Fatalf("create tileSet asset: %v", err)
	}
	if id != 100 || assetDao.asset.Type != string(domain.AssetTypeTileSet) {
		t.Fatalf("unexpected tileset asset: id=%d, type=%s", id, assetDao.asset.Type)
	}

	// CreateUISetAsset
	id, err = repo.CreateUISetAsset(ctx, &domain.Asset{
		Name:        "HealthBar",
		ProjectID:   1,
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  dim,
	})
	if err != nil {
		t.Fatalf("create uiSet asset: %v", err)
	}
	if id != 100 || assetDao.asset.Type != string(domain.AssetTypeUISet) {
		t.Fatalf("unexpected uiset asset: id=%d, type=%s", id, assetDao.asset.Type)
	}

	// CreateSceneryAsset
	id, err = repo.CreateSceneryAsset(ctx, &domain.Asset{
		Name:        "BackgroundMountain",
		ProjectID:   1,
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  dim,
	})
	if err != nil {
		t.Fatalf("create scenery asset: %v", err)
	}
	if id != 100 || assetDao.asset.Type != string(domain.AssetTypeScenery) {
		t.Fatalf("unexpected scenery asset: id=%d, type=%s", id, assetDao.asset.Type)
	}
}

func TestNewAssetRepository(t *testing.T) {
	r1 := repository.NewAssetRepository(nil, nil, nil)
	if r1 == nil {
		t.Fatal("expected non-nil AssetRepository")
	}
	r2 := repository.NewAssetRepositoryWithDB(nil, nil, nil, nil)
	if r2 == nil {
		t.Fatal("expected non-nil AssetRepository")
	}
}
