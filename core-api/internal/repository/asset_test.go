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

func TestAssetRepositoryGetAssetsMapsDAOResults(t *testing.T) {
	dimensions := json.RawMessage(`{"width":128,"height":128}`)
	daoStub := &assetDaoStub{assets: []dao.Asset{{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        "character",
		Description: "main character",
		Tags:        []string{"player", "hero"},
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
		{ID: 1, ProjectID: 42, Name: "hero", Type: "character", Tags: []string{"hero", "player"}},
		{ID: 2, ProjectID: 42, Type: "object", Tags: []string{"hero", "prop"}},
		{ID: 3, ProjectID: 42, Type: "character", Tags: []string{"npc"}},
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

func TestAssetRepositoryUpdatesAssetBasics(t *testing.T) {
	name := "updated hero"
	projectID := uint(42)
	typeValue := domain.AssetTypeCharacter
	description := "updated description"
	tags := []string{"prop"}
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
		Tags:      []string{"prop"},
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
