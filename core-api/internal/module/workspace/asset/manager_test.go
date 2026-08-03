package asset_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type assetStoreStub struct {
	domain.Store
	assets       []domain.Asset
	asset        *domain.Asset
	getAssetsErr error
	getDetailErr error
	projectID    uint
	assetID      uint
	filter       domain.AssetListFilter
	deletedID    uint
	deleteErr    error
}

func (s *assetStoreStub) GetAssetsByProjectID(_ context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	s.projectID = projectID
	s.filter = filter
	return s.assets, s.getAssetsErr
}

func (s *assetStoreStub) GetAssetDetail(_ context.Context, assetID uint) (*domain.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetStoreStub) Delete(_ context.Context, assetID uint) error {
	s.deletedID = assetID
	return s.deleteErr
}

func TestAssetManagerGetAssetsForwardsProjectIDAndResult(t *testing.T) {
	want := []domain.Asset{{ID: 7, ProjectID: 42, Name: "hero"}}
	store := &assetStoreStub{assets: want}
	manager := domain.NewManager(store)

	got, err := manager.GetAssets(context.Background(), 42, domain.AssetListFilter{})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if store.projectID != 42 {
		t.Fatalf("expected project ID 42, got %d", store.projectID)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want[0]) {
		t.Fatalf("unexpected assets: %+v", got)
	}
}

func TestAssetManagerGetDetailReturnsStoreAsset(t *testing.T) {
	want := &domain.Asset{ID: 7, ProjectID: 42, Name: "hero"}
	store := &assetStoreStub{asset: want}
	manager := domain.NewManager(store)

	got, err := manager.GetDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if store.assetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", store.assetID)
	}
	if !reflect.DeepEqual(got, *want) {
		t.Fatalf("unexpected asset: %+v", got)
	}
}

func TestAssetManagerPropagatesStoreErrors(t *testing.T) {
	wantErr := errors.New("asset lookup failed")
	manager := domain.NewManager(&assetStoreStub{getDetailErr: wantErr})

	_, err := manager.GetDetail(context.Background(), 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetManagerDeleteForwardsAssetID(t *testing.T) {
	store := &assetStoreStub{}
	manager := domain.NewManager(store)

	if err := manager.Delete(context.Background(), 7); err != nil {
		t.Fatalf("delete asset: %v", err)
	}
	if store.deletedID != 7 {
		t.Fatalf("expected asset ID 7, got %d", store.deletedID)
	}
}
