package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type assetRouterStub struct {
	router.AssetRouter
	projectID string
	assetID   string
}

func (s *assetRouterStub) GetAssets(context *echox.Context) (dto.Response, error) {
	s.projectID = context.Param("project_id")
	return dto.NewSuccessResponse(dto.GetAssetsResponse{Assets: []dto.AssetListItemResponse{{AssetID: 7, ProjectID: 42}}}), nil
}

func (s *assetRouterStub) Detail(context *echox.Context) (dto.Response, error) {
	s.assetID = context.Param("asset_id")
	return dto.NewSuccessResponse(dto.AssetDetailResponse{AssetID: 7}), nil
}

func TestAssetRoutesBindPathParameters(t *testing.T) {
	assetStub := &assetRouterStub{}
	e := router.Register(assetStub, nil, nil, nil, nil)

	t.Run("get assets", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/42/assets", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.projectID != "42" {
			t.Fatalf("expected project ID 42, got %q", assetStub.projectID)
		}
		if recorder.Body.String() != `{"code":200,"message":"success","data":{"assets":[{"assetId":7,"name":"","projectId":42,"type":"","description":"","tags":null,"version":0}]}}`+"\n" {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	})

	t.Run("asset detail", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/asset/7", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.assetID != "7" {
			t.Fatalf("expected asset ID 7, got %q", assetStub.assetID)
		}
		if recorder.Body.String() != `{"code":200,"message":"success","data":{"assetId":7,"name":"","projectId":0,"type":"","description":"","tags":null,"attributes":null,"version":0}}`+"\n" {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	})
}
