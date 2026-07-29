package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

type assetRepositoryStub struct {
	repository.AssetRepository
	assets       []domain.Asset
	asset        *domain.Asset
	getAssetsErr error
	getDetailErr error
	projectID    uint
	assetID      uint
	filter       domain.AssetListFilter
}

func (s *assetRepositoryStub) GetAssetsByProjectID(_ context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	s.projectID = projectID
	s.filter = filter
	return s.assets, s.getAssetsErr
}

func (s *assetRepositoryStub) GetAssetDetail(_ context.Context, assetID uint) (*domain.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func TestAssetServiceGetAssetsForwardsProjectIDAndResult(t *testing.T) {
	want := []domain.Asset{{ID: 7, ProjectID: 42, Name: "hero"}}
	repositoryStub := &assetRepositoryStub{assets: want}
	assetService := service.NewAssetService(repositoryStub)

	got, err := assetService.GetAssets(context.Background(), 42, domain.AssetListFilter{})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if repositoryStub.projectID != 42 {
		t.Fatalf("expected project ID 42, got %d", repositoryStub.projectID)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want[0]) {
		t.Fatalf("unexpected assets: %+v", got)
	}
}

func TestAssetServiceGetDetailReturnsRepositoryAsset(t *testing.T) {
	want := &domain.Asset{ID: 7, ProjectID: 42, Name: "hero"}
	repositoryStub := &assetRepositoryStub{asset: want}
	assetService := service.NewAssetService(repositoryStub)

	got, err := assetService.GetDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if repositoryStub.assetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", repositoryStub.assetID)
	}
	if !reflect.DeepEqual(got, *want) {
		t.Fatalf("unexpected asset: %+v", got)
	}
}

func TestAssetServicePropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("asset lookup failed")
	assetService := service.NewAssetService(&assetRepositoryStub{getDetailErr: wantErr})

	_, err := assetService.GetDetail(context.Background(), 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
