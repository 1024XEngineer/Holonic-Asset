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

	updateID     uint
	updateInput  *domain.AssetUpdate
	updateResult *domain.Asset
	updateErr    error

	characterAssetInput  *domain.Asset
	characterAssetResult *domain.Asset
	characterAssetErr    error

	objectAssetInput *domain.Asset
	objectAssetID    uint
	objectAssetErr   error

	tileSetAssetInput *domain.Asset
	tileSetAssetID    uint
	tileSetAssetErr   error

	uiSetAssetInput *domain.Asset
	uiSetAssetID    uint
	uiSetAssetErr   error

	sceneryAssetInput *domain.Asset
	sceneryAssetID    uint
	sceneryAssetErr   error

	recordInput           *domain.AssetRecord
	recordExpectedVersion uint
	recordResult          *domain.AssetRecord
	recordErr             error

	historyAssetID uint
	historyResult  []domain.AssetRecord
	historyErr     error

	rollbackAssetID uint
	rollbackVersion uint
	rollbackResult  *domain.AssetRecord
	rollbackErr     error

	copyAssetID uint
	copyVersion uint
	copyResult  uint
	copyErr     error
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

func (s *assetStoreStub) UpdateAsset(_ context.Context, id uint, update *domain.AssetUpdate) (*domain.Asset, error) {
	s.updateID = id
	s.updateInput = update
	return s.updateResult, s.updateErr
}

func (s *assetStoreStub) CreateCharacterAsset(_ context.Context, asset *domain.Asset) (*domain.Asset, error) {
	s.characterAssetInput = asset
	return s.characterAssetResult, s.characterAssetErr
}

func (s *assetStoreStub) CreateObjectAsset(_ context.Context, asset *domain.Asset) (uint, error) {
	s.objectAssetInput = asset
	return s.objectAssetID, s.objectAssetErr
}

func (s *assetStoreStub) CreateTileSetAsset(_ context.Context, asset *domain.Asset) (uint, error) {
	s.tileSetAssetInput = asset
	return s.tileSetAssetID, s.tileSetAssetErr
}

func (s *assetStoreStub) CreateUISetAsset(_ context.Context, asset *domain.Asset) (uint, error) {
	s.uiSetAssetInput = asset
	return s.uiSetAssetID, s.uiSetAssetErr
}

func (s *assetStoreStub) CreateSceneryAsset(_ context.Context, asset *domain.Asset) (uint, error) {
	s.sceneryAssetInput = asset
	return s.sceneryAssetID, s.sceneryAssetErr
}

func (s *assetStoreStub) CreateRecord(_ context.Context, record *domain.AssetRecord, expectedVersion uint) (*domain.AssetRecord, error) {
	s.recordInput = record
	s.recordExpectedVersion = expectedVersion
	return s.recordResult, s.recordErr
}

func (s *assetStoreStub) GetRecordHistory(_ context.Context, assetID uint) ([]domain.AssetRecord, error) {
	s.historyAssetID = assetID
	return s.historyResult, s.historyErr
}

func (s *assetStoreStub) RollBackRecord(_ context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	s.rollbackAssetID = assetID
	s.rollbackVersion = version
	return s.rollbackResult, s.rollbackErr
}

func (s *assetStoreStub) Copy(_ context.Context, assetID uint, version uint) (uint, error) {
	s.copyAssetID = assetID
	s.copyVersion = version
	return s.copyResult, s.copyErr
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

func TestAssetManagerGetDetailReturnsZeroAssetWhenNotFound(t *testing.T) {
	store := &assetStoreStub{asset: nil}
	manager := domain.NewManager(store)

	got, err := manager.GetDetail(context.Background(), 999)
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if !reflect.DeepEqual(got, domain.Asset{}) {
		t.Fatalf("expected empty asset, got %+v", got)
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

func TestAssetManagerUpdateAssetForwardsUpdate(t *testing.T) {
	name := "Updated Name"
	update := &domain.AssetUpdate{Name: &name}
	expected := &domain.Asset{ID: 10, Name: name}
	store := &assetStoreStub{updateResult: expected}
	manager := domain.NewManager(store)

	result, err := manager.UpdateAsset(context.Background(), 10, update)
	if err != nil {
		t.Fatalf("update asset: %v", err)
	}
	if store.updateID != 10 || store.updateInput != update || result != expected {
		t.Fatalf("unexpected update forwarding: id=%d input=%+v result=%+v", store.updateID, store.updateInput, result)
	}
}

func TestAssetManagerCreateDifferentAssetTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("create character asset", func(t *testing.T) {
		input := &domain.Asset{Name: "Knight"}
		expected := &domain.Asset{ID: 1, Name: "Knight"}
		store := &assetStoreStub{characterAssetResult: expected}
		manager := domain.NewManager(store)

		res, err := manager.CreateCharacterAsset(ctx, input)
		if err != nil {
			t.Fatalf("create character asset: %v", err)
		}
		if store.characterAssetInput != input || res != expected {
			t.Fatalf("unexpected character asset forwarding: input=%+v res=%+v", store.characterAssetInput, res)
		}
	})

	t.Run("create object asset", func(t *testing.T) {
		input := &domain.Asset{Name: "Chest"}
		store := &assetStoreStub{objectAssetID: 2}
		manager := domain.NewManager(store)

		id, err := manager.CreateObjectAsset(ctx, input)
		if err != nil {
			t.Fatalf("create object asset: %v", err)
		}
		if store.objectAssetInput != input || id != 2 {
			t.Fatalf("unexpected object asset forwarding: input=%+v id=%d", store.objectAssetInput, id)
		}
	})

	t.Run("create tileset asset", func(t *testing.T) {
		input := &domain.Asset{Name: "Dungeon Tiles"}
		store := &assetStoreStub{tileSetAssetID: 3}
		manager := domain.NewManager(store)

		id, err := manager.CreateTileSetAsset(ctx, input)
		if err != nil {
			t.Fatalf("create tileset asset: %v", err)
		}
		if store.tileSetAssetInput != input || id != 3 {
			t.Fatalf("unexpected tileset asset forwarding: input=%+v id=%d", store.tileSetAssetInput, id)
		}
	})

	t.Run("create uiset asset", func(t *testing.T) {
		input := &domain.Asset{Name: "HUD Pack"}
		store := &assetStoreStub{uiSetAssetID: 4}
		manager := domain.NewManager(store)

		id, err := manager.CreateUISetAsset(ctx, input)
		if err != nil {
			t.Fatalf("create uiset asset: %v", err)
		}
		if store.uiSetAssetInput != input || id != 4 {
			t.Fatalf("unexpected uiset asset forwarding: input=%+v id=%d", store.uiSetAssetInput, id)
		}
	})

	t.Run("create scenery asset", func(t *testing.T) {
		input := &domain.Asset{Name: "Forest Background"}
		store := &assetStoreStub{sceneryAssetID: 5}
		manager := domain.NewManager(store)

		id, err := manager.CreateSceneryAsset(ctx, input)
		if err != nil {
			t.Fatalf("create scenery asset: %v", err)
		}
		if store.sceneryAssetInput != input || id != 5 {
			t.Fatalf("unexpected scenery asset forwarding: input=%+v id=%d", store.sceneryAssetInput, id)
		}
	})
}

func TestAssetManagerRecordOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("create record", func(t *testing.T) {
		record := &domain.AssetRecord{AssetID: 10, Version: 2}
		expected := &domain.AssetRecord{ID: 100, AssetID: 10, Version: 2}
		store := &assetStoreStub{recordResult: expected}
		manager := domain.NewManager(store)

		res, err := manager.CreateRecord(ctx, record, 1)
		if err != nil {
			t.Fatalf("create record: %v", err)
		}
		if store.recordInput != record || store.recordExpectedVersion != 1 || res != expected {
			t.Fatalf("unexpected record creation forwarding: input=%+v expectedVer=%d res=%+v", store.recordInput, store.recordExpectedVersion, res)
		}
	})

	t.Run("get record history", func(t *testing.T) {
		history := []domain.AssetRecord{{ID: 1, Version: 1}, {ID: 2, Version: 2}}
		store := &assetStoreStub{historyResult: history}
		manager := domain.NewManager(store)

		res, err := manager.GetRecordHistory(ctx, 10)
		if err != nil {
			t.Fatalf("get record history: %v", err)
		}
		if store.historyAssetID != 10 || len(res) != 2 {
			t.Fatalf("unexpected history lookup: assetID=%d res=%+v", store.historyAssetID, res)
		}
	})

	t.Run("rollback record", func(t *testing.T) {
		rolledBack := &domain.AssetRecord{ID: 1, Version: 1}
		store := &assetStoreStub{rollbackResult: rolledBack}
		manager := domain.NewManager(store)

		res, err := manager.RollBackRecord(ctx, 10, 1)
		if err != nil {
			t.Fatalf("rollback record: %v", err)
		}
		if store.rollbackAssetID != 10 || store.rollbackVersion != 1 || res != rolledBack {
			t.Fatalf("unexpected rollback forwarding: assetID=%d version=%d res=%+v", store.rollbackAssetID, store.rollbackVersion, res)
		}
	})

	t.Run("copy record", func(t *testing.T) {
		store := &assetStoreStub{copyResult: 99}
		manager := domain.NewManager(store)

		newID, err := manager.Copy(ctx, 10, 2)
		if err != nil {
			t.Fatalf("copy record: %v", err)
		}
		if store.copyAssetID != 10 || store.copyVersion != 2 || newID != 99 {
			t.Fatalf("unexpected copy forwarding: assetID=%d version=%d newID=%d", store.copyAssetID, store.copyVersion, newID)
		}
	})
}
