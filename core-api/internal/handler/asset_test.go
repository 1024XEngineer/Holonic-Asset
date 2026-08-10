package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type assetManagerStub struct {
	domain.Manager
	assets          []domain.Asset
	asset           domain.Asset
	updatedAsset    *domain.Asset
	getAssetsErr    error
	getDetailErr    error
	updateErr       error
	projectID       uint
	assetID         uint
	filter          domain.AssetListFilter
	updateID        uint
	update          *domain.AssetUpdate
	record          *domain.AssetRecord
	recordRequest   *domain.AssetRecord
	records         []domain.AssetRecord
	rollbackAsset   uint
	rollbackVersion uint
	rollbackResult  *domain.AssetRecord
	deletedAssetID  uint
	deleteErr       error
}

func (s *assetManagerStub) CreateRecord(_ context.Context, record *domain.AssetRecord) (*domain.AssetRecord, error) {
	s.recordRequest = record
	return s.record, nil
}

func (s *assetManagerStub) GetRecordHistory(_ context.Context, _ uint) ([]domain.AssetRecord, error) {
	return s.records, nil
}

func (s *assetManagerStub) RollBackRecord(_ context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	s.rollbackAsset = assetID
	s.rollbackVersion = version
	return s.rollbackResult, nil
}

func (s *assetManagerStub) Delete(_ context.Context, assetID uint) error {
	s.deletedAssetID = assetID
	return s.deleteErr
}

func (s *assetManagerStub) GetAssets(_ context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	s.projectID = projectID
	s.filter = filter
	return s.assets, s.getAssetsErr
}

func (s *assetManagerStub) GetDetail(_ context.Context, assetID uint) (domain.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetManagerStub) UpdateAsset(_ context.Context, assetID uint, update *domain.AssetUpdate) (*domain.Asset, error) {
	s.updateID = assetID
	s.update = update
	if s.updatedAsset != nil {
		return s.updatedAsset, s.updateErr
	}
	return &domain.Asset{ID: assetID}, s.updateErr
}

func TestAssetHandlerGetAssetsMapsResponse(t *testing.T) {
	managerStub := &assetManagerStub{assets: []domain.Asset{{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        domain.AssetTypeCharacter,
		Description: "main character",
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Tags:        []string{"player"},
		Version:     3,
	}}}
	h := handler.NewHandler(managerStub)

	response, err := h.GetAssets(context.Background(), dto.GetAssetsRequest{ProjectID: 42})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if managerStub.projectID != 42 {
		t.Fatalf("expected project ID 42, got %d", managerStub.projectID)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response: %+v", response)
	}
	data := response.Data
	if len(data.Assets) != 1 || data.Assets[0].AssetID != 7 || data.Assets[0].ProjectID != 42 {
		t.Fatalf("unexpected response data: %+v", data)
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if string(payload) != `{"code":200,"message":"success","data":{"assets":[{"assetId":7,"name":"hero","projectId":42,"type":"character","description":"main character","perspective":"Top-Down","dimensions":{"width":64,"height":64},"tags":["player"],"version":3}]}}` {
		t.Fatalf("unexpected JSON response: %s", payload)
	}
}

func TestAssetHandlerPassesAssetQueryFilter(t *testing.T) {
	managerStub := &assetManagerStub{}
	h := handler.NewHandler(managerStub)

	_, err := h.GetAssets(context.Background(), dto.GetAssetsRequest{
		ProjectID: 42,
		Query:     "hero",
		Tags:      []string{"player"},
		Types:     []domain.AssetType{domain.AssetTypeCharacter},
	})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if managerStub.filter.Query != "hero" || len(managerStub.filter.Tags) != 1 || len(managerStub.filter.Types) != 1 {
		t.Fatalf("unexpected asset filter: %+v", managerStub.filter)
	}
}

func TestAssetHandlerUpdatesAssetBasicsWithoutContent(t *testing.T) {
	name := "updated hero"
	projectID := uint(99)
	typeValue := domain.AssetTypeObject
	description := "updated description"
	tags := []string{"prop"}
	perspective := domain.PerspectiveSideOn
	dimensions := json.RawMessage(`{"width":64,"height":64}`)
	version := uint(4)
	managerStub := &assetManagerStub{updatedAsset: &domain.Asset{
		ID:          7,
		Name:        name,
		ProjectID:   projectID,
		Type:        typeValue,
		Description: description,
		Tags:        tags,
		Perspective: perspective,
		Dimensions:  dimensions,
		Version:     version,
	}}
	h := handler.NewHandler(managerStub)

	response, err := h.UpdateAsset(context.Background(), dto.UpdateAssetRequest{
		AssetID:     7,
		Name:        &name,
		Description: &description,
		Tags:        &tags,
		Perspective: &perspective,
		Dimensions:  &dimensions,
	})
	if err != nil {
		t.Fatalf("update asset basics: %v", err)
	}
	if managerStub.updateID != 7 || managerStub.update == nil || managerStub.update.Name == nil || *managerStub.update.Name != name {
		t.Fatalf("unexpected update request: %+v", managerStub.update)
	}
	data := response.Data
	if data.AssetID != 7 || data.Name != name || data.Perspective != perspective || string(data.Dimensions) != string(dimensions) {
		t.Fatalf("unexpected update response: %+v", response.Data)
	}
}

func TestAssetHandlerRecordReturnsCreatedSnapshot(t *testing.T) {
	managerStub := &assetManagerStub{record: &domain.AssetRecord{
		ID:        15,
		AssetID:   7,
		Version:   3,
		ContentID: 21,
	}}
	h := handler.NewHandler(managerStub)

	response, err := h.Record(context.Background(), dto.RecordAssetRequest{AssetID: 7})
	if err != nil {
		t.Fatalf("record asset: %v", err)
	}
	if managerStub.recordRequest == nil || managerStub.recordRequest.AssetID != 7 {
		t.Fatalf("unexpected record request: %+v", managerStub.recordRequest)
	}
	data := response.Data
	if data.RecordID != 15 || data.AssetID != 7 || data.Version != 3 || data.ContentID != 21 {
		t.Fatalf("unexpected record response: %+v", response.Data)
	}
}

func TestAssetHandlerRollbackUsesRequestedVersion(t *testing.T) {
	managerStub := &assetManagerStub{rollbackResult: &domain.AssetRecord{
		AssetID:   7,
		Version:   2,
		ContentID: 9,
	}}
	h := handler.NewHandler(managerStub)

	response, err := h.RollBackAsset(context.Background(), dto.RollBackAssetRequest{AssetID: 7, Version: 2})
	if err != nil {
		t.Fatalf("rollback asset: %v", err)
	}
	if managerStub.rollbackAsset != 7 || managerStub.rollbackVersion != 2 {
		t.Fatalf("unexpected rollback request: asset=%d version=%d", managerStub.rollbackAsset, managerStub.rollbackVersion)
	}
	data := response.Data
	if data.AssetID != 7 || data.Version != 2 || data.ContentID != 9 {
		t.Fatalf("unexpected rollback response: %+v", response.Data)
	}
}

func TestAssetHandlerRecordsReturnsHistory(t *testing.T) {
	managerStub := &assetManagerStub{records: []domain.AssetRecord{
		{ID: 15, AssetID: 7, Version: 1, ContentID: 21},
		{ID: 16, AssetID: 7, Version: 2, ContentID: 22},
	}}
	h := handler.NewHandler(managerStub)

	response, err := h.Records(
		context.Background(),
		dto.GetAssetRecordsRequest{AssetID: 7},
	)
	if err != nil {
		t.Fatalf("get asset records: %v", err)
	}
	data := response.Data
	if len(data.Records) != 2 || data.Records[1].Version != 2 || data.Records[1].ContentID != 22 {
		t.Fatalf("unexpected asset record history: %+v", response.Data)
	}
}

func TestAssetHandlerDetailMapsResponse(t *testing.T) {
	managerStub := &assetManagerStub{asset: domain.Asset{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        domain.AssetTypeObject,
		Description: "prop",
		Tags:        []string{"scene"},
		Perspective: domain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Version:     2,
	}}
	h := handler.NewHandler(managerStub)

	response, err := h.Detail(
		context.Background(),
		dto.AssetDetailRequest{AssetID: 7},
	)
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if managerStub.assetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", managerStub.assetID)
	}
	data := response.Data
	if data.AssetID != 7 || data.Perspective != domain.PerspectiveTopDown || string(data.Dimensions) != `{"width":64,"height":64}` {
		t.Fatalf("unexpected response data: %+v", data)
	}
}

func TestAssetHandlerResolvesNestedObjectKeysOnlyForResponse(t *testing.T) {
	prototypeURL := "uploads/prototype.png"
	frameURL := "uploads/frame.png"
	tileURL := "uploads/tile.png"
	content, err := domain.EncodeContent(domain.AssetContent{
		Prototype:  &domain.Prototype{{ID: 1, URL: &prototypeURL}},
		Animations: []domain.Animation{{Frames: []domain.Frame{{ID: 2, URL: &frameURL}}}},
		Items:      []domain.TileSetItem{{Tiles: []domain.Tile{{URL: &tileURL}}}},
	})
	if err != nil {
		t.Fatalf("encode asset content: %v", err)
	}
	managerStub := &assetManagerStub{asset: domain.Asset{ID: 7, Type: domain.AssetTypeCharacter, Content: content}}
	resolver := &referenceResolverStub{}
	h := handler.NewHandler(managerStub, resolver)

	response, err := h.Detail(context.Background(), dto.AssetDetailRequest{AssetID: 7})
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	var decoded domain.AssetContent
	if err := json.Unmarshal(response.Data.Content, &decoded); err != nil {
		t.Fatalf("decode resolved content: %v", err)
	}
	if *(*decoded.Prototype)[0].URL != "signed:uploads/prototype.png" ||
		*decoded.Animations[0].Frames[0].URL != "signed:uploads/frame.png" ||
		*decoded.Items[0].Tiles[0].URL != "signed:uploads/tile.png" {
		t.Fatalf("unexpected resolved content: %+v", decoded)
	}
	var persisted domain.AssetContent
	if err := json.Unmarshal(managerStub.asset.Content, &persisted); err != nil {
		t.Fatalf("decode original content: %v", err)
	}
	if *(*persisted.Prototype)[0].URL != prototypeURL || *persisted.Animations[0].Frames[0].URL != frameURL || *persisted.Items[0].Tiles[0].URL != tileURL {
		t.Fatalf("handler mutated persisted content: %+v", persisted)
	}
}

func TestAssetHandlerPreservesUnmodeledContentFields(t *testing.T) {
	raw := json.RawMessage(`{
		"directionCount":9007199254740993,
		"prototype":[{"id":1,"url":"uploads/prototype.png","custom":{"keep":true},"futureValue":12345678901234567890}],
		"animations":null,
		"items":[{"name":"floor","tiles":null,"customItem":"kept"}],
		"customTopLevel":{"nested":"kept"}
	}`)
	managerStub := &assetManagerStub{asset: domain.Asset{ID: 7, Content: raw}}
	h := handler.NewHandler(managerStub, &referenceResolverStub{})

	response, err := h.Detail(context.Background(), dto.AssetDetailRequest{AssetID: 7})
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(response.Data.Content, &decoded); err != nil {
		t.Fatalf("decode response content: %v", err)
	}
	if string(decoded["directionCount"]) != "9007199254740993" ||
		string(decoded["customTopLevel"]) != `{"nested":"kept"}` {
		t.Fatalf("unmodeled top-level fields changed: %s", response.Data.Content)
	}
	var prototype []map[string]json.RawMessage
	if err := json.Unmarshal(decoded["prototype"], &prototype); err != nil {
		t.Fatalf("decode prototype: %v", err)
	}
	if string(prototype[0]["custom"]) != `{"keep":true}` ||
		string(prototype[0]["futureValue"]) != "12345678901234567890" ||
		string(prototype[0]["url"]) != `"signed:uploads/prototype.png"` {
		t.Fatalf("prototype fields changed: %s", decoded["prototype"])
	}
	if string(decoded["animations"]) != "null" || string(decoded["items"]) == "null" {
		t.Fatalf("null or item fields were lost: %s", response.Data.Content)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(decoded["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	if string(items[0]["tiles"]) != "null" || string(items[0]["customItem"]) != `"kept"` {
		t.Fatalf("item fields changed: %s", decoded["items"])
	}
	if string(managerStub.asset.Content) != string(raw) {
		t.Fatalf("handler mutated persisted content: %s", managerStub.asset.Content)
	}
}

func TestAssetHandlerRejectsZeroIDs(t *testing.T) {
	h := handler.NewHandler(&assetManagerStub{})
	if _, err := h.GetAssets(context.Background(), dto.GetAssetsRequest{}); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero project ID, got %v", err)
	}
	if _, err := h.Detail(
		context.Background(),
		dto.AssetDetailRequest{},
	); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero asset ID, got %v", err)
	}
	if _, err := h.Records(
		context.Background(),
		dto.GetAssetRecordsRequest{},
	); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero record asset ID, got %v", err)
	}
}

func TestAssetHandlerPropagatesManagerErrors(t *testing.T) {
	wantErr := errors.New("asset manager failed")
	h := handler.NewHandler(&assetManagerStub{getDetailErr: wantErr})

	_, err := h.Detail(
		context.Background(),
		dto.AssetDetailRequest{AssetID: 7},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestAssetHandlerDelete(t *testing.T) {
	managerStub := &assetManagerStub{}
	h := handler.NewHandler(managerStub)

	response, err := h.Delete(context.Background(), dto.DeleteAssetRequest{AssetID: 7})
	if err != nil {
		t.Fatalf("delete asset: %v", err)
	}
	if managerStub.deletedAssetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", managerStub.deletedAssetID)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage || !response.Data.Success {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestAssetHandlerDeleteRejectsZeroAssetID(t *testing.T) {
	h := handler.NewHandler(&assetManagerStub{})

	_, err := h.Delete(context.Background(), dto.DeleteAssetRequest{})
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}
