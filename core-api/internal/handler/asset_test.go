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
	getAssetsErr error
	getDetailErr error
	projectID    uint
	assetID      uint
}

func (s *assetServiceStub) GetAssets(_ context.Context, projectID uint) ([]domain.Asset, error) {
	s.projectID = projectID
	return s.assets, s.getAssetsErr
}

func (s *assetServiceStub) GetDetail(_ context.Context, assetID uint) (domain.Asset, error) {
	s.assetID = assetID
	return s.asset, s.getDetailErr
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
	h := handler.NewHandler(serviceStub, nil, nil)

	response, err := h.GetAssets(newAssetHandlerContext("project_id", "42"))
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
	h := handler.NewHandler(serviceStub, nil, nil)

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
	h := handler.NewHandler(&assetServiceStub{}, nil, nil)
	if _, err := h.GetAssets(newAssetHandlerContext("project_id", "0")); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero project ID, got %v", err)
	}
	if _, err := h.Detail(newAssetHandlerContext("asset_id", "invalid")); !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request for zero asset ID, got %v", err)
	}
}

func TestAssetHandlerPropagatesServiceErrors(t *testing.T) {
	wantErr := errors.New("asset service failed")
	h := handler.NewHandler(&assetServiceStub{getDetailErr: wantErr}, nil, nil)

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
