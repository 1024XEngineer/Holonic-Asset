package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type assetServiceStub struct {
	service.AssetService
	assets       []domain.Asset
	asset        domain.Asset
	updatedAsset *domain.Asset
	getAssetsErr error
	getDetailErr error
	updateErr    error
	projectID    uint
	assetID      uint
	filter       domain.AssetListFilter
	updateID     uint
	update       *domain.AssetUpdate
}

type assetRecordServiceStub struct {
	service.AssetRecordService
	record          *domain.AssetRecord
	recordRequest   *domain.AssetRecord
	records         []domain.AssetRecord
	rollbackAsset   uint
	rollbackVersion uint
	rollbackResult  *domain.AssetRecord
}

func (s *assetRecordServiceStub) CreateRecord(_ context.Context, record *domain.AssetRecord) (*domain.AssetRecord, error) {
	s.recordRequest = record
	return s.record, nil
}

func (s *assetRecordServiceStub) GetRecordHistory(_ context.Context, _ uint) ([]domain.AssetRecord, error) {
	return s.records, nil
}

func (s *assetRecordServiceStub) RollBackRecord(_ context.Context, assetID uint, version uint) (*domain.AssetRecord, error) {
	s.rollbackAsset = assetID
	s.rollbackVersion = version
	return s.rollbackResult, nil
}

func (s *assetServiceStub) GetAssets(_ context.Context, projectID uint, filter domain.AssetListFilter) ([]domain.Asset, error) {
	s.projectID = projectID
	s.filter = filter
	return s.assets, s.getAssetsErr
}

func (s *assetServiceStub) GetDetail(_ context.Context, assetID uint) (domain.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
}

func (s *assetServiceStub) UpdateAsset(_ context.Context, assetID uint, update *domain.AssetUpdate) (*domain.Asset, error) {
	s.updateID = assetID
	s.update = update
	if s.updatedAsset != nil {
		return s.updatedAsset, s.updateErr
	}
	return &domain.Asset{ID: assetID}, s.updateErr
}

func TestAssetHandlerGetAssetsMapsResponse(t *testing.T) {
	serviceStub := &assetServiceStub{assets: []domain.Asset{{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        domain.AssetTypeCharacter,
		Description: "main character",
		Tags:        []string{"player"},
		Version:     3,
	}}}
	h := handler.NewHandler(serviceStub, nil)

	response, err := h.GetAssets(newAssetHandlerContext("project_id", "42"), dto.GetAssetsRequest{})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if serviceStub.projectID != 42 {
		t.Fatalf("expected project ID 42, got %d", serviceStub.projectID)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response: %+v", response)
	}
	data, ok := response.Data.(dto.GetAssetsResponse)
	if !ok {
		t.Fatalf("expected GetAssetsResponse data, got %T", response.Data)
	}
	if len(data.Assets) != 1 || data.Assets[0].AssetID != 7 || data.Assets[0].ProjectID != 42 {
		t.Fatalf("unexpected response data: %+v", data)
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if string(payload) != `{"code":200,"message":"success","data":{"assets":[{"assetId":7,"name":"hero","projectId":42,"type":"character","description":"main character","tags":["player"],"version":3}]}}` {
		t.Fatalf("unexpected JSON response: %s", payload)
	}
}

func TestAssetHandlerPassesAssetQueryFilter(t *testing.T) {
	serviceStub := &assetServiceStub{}
	h := handler.NewHandler(serviceStub, nil)

	_, err := h.GetAssets(newAssetHandlerContext("project_id", "42"), dto.GetAssetsRequest{
		Query: "hero",
		Tags:  []string{"player"},
		Types: []domain.AssetType{domain.AssetTypeCharacter},
	})
	if err != nil {
		t.Fatalf("get assets: %v", err)
	}
	if serviceStub.filter.Query != "hero" || len(serviceStub.filter.Tags) != 1 || len(serviceStub.filter.Types) != 1 {
		t.Fatalf("unexpected asset filter: %+v", serviceStub.filter)
	}
}

func TestAssetHandlerUpdatesAssetBasicsWithoutContent(t *testing.T) {
	name := "updated hero"
	projectID := uint(99)
	typeValue := domain.AssetTypeObject
	description := "updated description"
	tags := []string{"prop"}
	attributes := json.RawMessage(`{"scale":2}`)
	version := uint(4)
	serviceStub := &assetServiceStub{updatedAsset: &domain.Asset{
		ID:          7,
		Name:        name,
		ProjectID:   projectID,
		Type:        typeValue,
		Description: description,
		Tags:        tags,
		Attributes:  attributes,
		Version:     version,
	}}
	h := handler.NewHandler(serviceStub, nil)

	response, err := h.UpdateAsset(newAssetHandlerContext("asset_id", "7"), dto.UpdateAssetRequest{
		AssetID:     7,
		Name:        &name,
		ProjectID:   &projectID,
		Type:        &typeValue,
		Description: &description,
		Tags:        &tags,
		Attributes:  &attributes,
	})
	if err != nil {
		t.Fatalf("update asset basics: %v", err)
	}
	if serviceStub.updateID != 7 || serviceStub.update == nil || serviceStub.update.Name == nil || *serviceStub.update.Name != name {
		t.Fatalf("unexpected update request: %+v", serviceStub.update)
	}
	data, ok := response.Data.(dto.UpdateAssetResponse)
	if !ok || data.AssetID != 7 || data.Name != name || string(data.Attributes) != string(attributes) {
		t.Fatalf("unexpected update response: %+v", response.Data)
	}
}

func TestAssetHandlerRecordReturnsCreatedSnapshot(t *testing.T) {
	recordService := &assetRecordServiceStub{record: &domain.AssetRecord{
		ID:        15,
		AssetID:   7,
		Version:   3,
		ContentID: 21,
	}}
	h := handler.NewHandler(&assetServiceStub{}, recordService)

	response, err := h.Record(newAssetHandlerContext("asset_id", "7"), dto.RecordAssetRequest{AssetID: 7})
	if err != nil {
		t.Fatalf("record asset: %v", err)
	}
	if recordService.recordRequest == nil || recordService.recordRequest.AssetID != 7 {
		t.Fatalf("unexpected record request: %+v", recordService.recordRequest)
	}
	data, ok := response.Data.(dto.RecordAssetResponse)
	if !ok || data.RecordID != 15 || data.AssetID != 7 || data.Version != 3 || data.ContentID != 21 {
		t.Fatalf("unexpected record response: %+v", response.Data)
	}
}

func TestAssetHandlerRollbackUsesRequestedVersion(t *testing.T) {
	recordService := &assetRecordServiceStub{rollbackResult: &domain.AssetRecord{
		AssetID:   7,
		Version:   2,
		ContentID: 9,
	}}
	h := handler.NewHandler(&assetServiceStub{}, recordService)

	response, err := h.RollBackAsset(newAssetHandlerContext("asset_id", "7"), dto.RollBackAssetRequest{AssetID: 7, Version: 2})
	if err != nil {
		t.Fatalf("rollback asset: %v", err)
	}
	if recordService.rollbackAsset != 7 || recordService.rollbackVersion != 2 {
		t.Fatalf("unexpected rollback request: asset=%d version=%d", recordService.rollbackAsset, recordService.rollbackVersion)
	}
	data, ok := response.Data.(dto.RollBackAssetResponse)
	if !ok || data.AssetID != 7 || data.Version != 2 || data.ContentID != 9 {
		t.Fatalf("unexpected rollback response: %+v", response.Data)
	}
}

func TestAssetHandlerRecordsReturnsHistory(t *testing.T) {
	recordService := &assetRecordServiceStub{records: []domain.AssetRecord{
		{ID: 15, AssetID: 7, Version: 1, ContentID: 21},
		{ID: 16, AssetID: 7, Version: 2, ContentID: 22},
	}}
	h := handler.NewHandler(&assetServiceStub{}, recordService)

	response, err := h.Records(newAssetHandlerContext("asset_id", "7"))
	if err != nil {
		t.Fatalf("get asset records: %v", err)
	}
	data, ok := response.Data.(dto.GetAssetRecordsResponse)
	if !ok || len(data.Records) != 2 || data.Records[1].Version != 2 || data.Records[1].ContentID != 22 {
		t.Fatalf("unexpected asset record history: %+v", response.Data)
	}
}

func TestAssetHandlerDetailMapsResponse(t *testing.T) {
	serviceStub := &assetServiceStub{asset: domain.Asset{
		ID:          7,
		Name:        "hero",
		ProjectID:   42,
		Type:        domain.AssetTypeObject,
		Description: "prop",
		Tags:        []string{"scene"},
		Attributes:  json.RawMessage(`{"mesh":"hero.glb"}`),
		Version:     2,
	}}
	h := handler.NewHandler(serviceStub, nil)

	response, err := h.Detail(newAssetHandlerContext("asset_id", "7"))
	if err != nil {
		t.Fatalf("get asset detail: %v", err)
	}
	if serviceStub.assetID != 7 {
		t.Fatalf("expected asset ID 7, got %d", serviceStub.assetID)
	}
	data, ok := response.Data.(dto.AssetDetailResponse)
	if !ok {
		t.Fatalf("expected AssetDetailResponse data, got %T", response.Data)
	}
	if data.AssetID != 7 || data.Attributes == nil || string(data.Attributes) != `{"mesh":"hero.glb"}` {
		t.Fatalf("unexpected response data: %+v", data)
	}
}

func TestAssetHandlerRejectsZeroIDs(t *testing.T) {
	h := handler.NewHandler(&assetServiceStub{}, nil)
	if _, err := h.GetAssets(newAssetHandlerContext("project_id", "0"), dto.GetAssetsRequest{}); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero project ID, got %v", err)
	}
	if _, err := h.Detail(newAssetHandlerContext("asset_id", "invalid")); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero asset ID, got %v", err)
	}
}

func TestAssetHandlerPropagatesServiceErrors(t *testing.T) {
	wantErr := errors.New("asset service failed")
	h := handler.NewHandler(&assetServiceStub{getDetailErr: wantErr}, nil)

	_, err := h.Detail(newAssetHandlerContext("asset_id", "7"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func newAssetHandlerContext(paramName string, paramValue string) *echox.Context {
	e := echo.New()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	context := e.NewContext(request, httptest.NewRecorder())
	context.SetParamNames(paramName)
	context.SetParamValues(paramValue)
	return &echox.Context{Context: context}
}
