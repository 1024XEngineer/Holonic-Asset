package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type assetDaoStub struct {
	dao.AssetDao
	assets       []dao.Asset
	asset        dao.Asset
	getAssetsErr error
	getDetailErr error
	projectID    uint
	assetID      uint
}

func (s *assetDaoStub) GetAssetsByProjectID(_ context.Context, projectID uint) ([]dao.Asset, error) {
	s.projectID = projectID
	return s.assets, s.getAssetsErr
}

func (s *assetDaoStub) GetAssetDetail(_ context.Context, assetID uint) (dao.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func TestAssetRepositoryGetAssetsMapsDAOResults(t *testing.T) {
	attributes := json.RawMessage(`{"width":128}`)
	daoStub := &assetDaoStub{assets: []dao.Asset{{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        "character",
		Description: "main character",
		Tags:        []string{"player", "hero"},
		Attributes:  attributes,
		Version:     3,
	}}}
	repo := &repository.AssetRepositoryImpl{AssetDao: daoStub}

	got, err := repo.GetAssetsByProjectID(context.Background(), 42)
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
	if string(got[0].Attributes) != string(attributes) || len(got[0].Tags) != 2 {
		t.Fatalf("asset data was not mapped: %+v", got[0])
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
