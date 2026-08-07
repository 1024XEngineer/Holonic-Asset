package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type assetRouterStub struct {
	router.AssetRouter
	projectID uint
	assetID   uint
	request   dto.GetAssetsRequest
	update    dto.UpdateAssetRequest
	record    dto.RecordAssetRequest
	records   bool
	rollback  dto.RollBackAssetRequest
	deleted   dto.DeleteAssetRequest
}

func (s *assetRouterStub) GetAssets(
	_ context.Context,
	request dto.GetAssetsRequest,
) (dto.SuccessResponse[dto.GetAssetsResponse], error) {
	s.projectID = request.ProjectID
	s.request = request
	return dto.NewTypedSuccessResponse(dto.GetAssetsResponse{Assets: []dto.AssetListItemResponse{{AssetID: 7, ProjectID: 42}}}), nil
}

func (s *assetRouterStub) Detail(
	_ context.Context,
	request dto.AssetDetailRequest,
) (dto.SuccessResponse[dto.AssetDetailResponse], error) {
	s.assetID = request.AssetID
	return dto.NewTypedSuccessResponse(dto.AssetDetailResponse{AssetID: 7}), nil
}

func (s *assetRouterStub) UpdateAsset(
	_ context.Context,
	request dto.UpdateAssetRequest,
) (dto.SuccessResponse[dto.UpdateAssetResponse], error) {
	s.update = request
	return dto.NewTypedSuccessResponse(dto.UpdateAssetResponse{AssetID: request.AssetID}), nil
}

func (s *assetRouterStub) Record(
	_ context.Context,
	request dto.RecordAssetRequest,
) (dto.SuccessResponse[dto.RecordAssetResponse], error) {
	s.record = request
	return dto.NewTypedSuccessResponse(dto.RecordAssetResponse{AssetID: request.AssetID, Version: 2}), nil
}

func (s *assetRouterStub) Records(
	_ context.Context,
	request dto.GetAssetRecordsRequest,
) (dto.SuccessResponse[dto.GetAssetRecordsResponse], error) {
	s.assetID = request.AssetID
	s.records = true
	return dto.NewTypedSuccessResponse(dto.GetAssetRecordsResponse{Records: []dto.AssetRecordResponse{}}), nil
}

func (s *assetRouterStub) RollBackAsset(
	_ context.Context,
	request dto.RollBackAssetRequest,
) (dto.SuccessResponse[dto.RollBackAssetResponse], error) {
	s.rollback = request
	return dto.NewTypedSuccessResponse(dto.RollBackAssetResponse{AssetID: request.AssetID, Version: request.Version}), nil
}

func (s *assetRouterStub) Delete(
	_ context.Context,
	request dto.DeleteAssetRequest,
) (dto.SuccessResponse[dto.DeleteAssetResponse], error) {
	s.deleted = request
	return dto.NewTypedSuccessResponse(dto.DeleteAssetResponse{Success: true}), nil
}

func TestAssetRoutesBindPathParameters(t *testing.T) {
	assetStub := &assetRouterStub{}
	e := router.Register(assetStub, nil, nil, nil)

	t.Run("get assets", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/42/assets", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.projectID != 42 {
			t.Fatalf("expected project ID 42, got %d", assetStub.projectID)
		}
		if recorder.Body.String() != `{"code":200,"message":"success","data":{"assets":[{"assetId":7,"name":"","projectId":42,"type":"","description":"","perspective":"","scale":null,"tags":null,"version":0}]}}`+"\n" {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	})

	t.Run("binds asset filters", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/42/assets?query=hero&tags=hero&tags=player&types=character", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if len(assetStub.request.Tags) != 2 || assetStub.request.Tags[0] != "hero" || assetStub.request.Tags[1] != "player" {
			t.Fatalf("unexpected tag filters: %+v", assetStub.request.Tags)
		}
		if len(assetStub.request.Types) != 1 || assetStub.request.Types[0] != "character" {
			t.Fatalf("unexpected type filters: %+v", assetStub.request.Types)
		}
		if assetStub.request.Query != "hero" {
			t.Fatalf("unexpected query filter: %q", assetStub.request.Query)
		}
	})

	t.Run("binds asset update", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/asset/update", strings.NewReader(`{
			"assetId": 7,
			"name": "hero",
			"description": "main character",
			"tags": ["player"],
			"perspective": "Top-Down",
			"scale": {"width": 64, "height": 64}
		}`))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.update.AssetID != 7 || assetStub.update.Name == nil || *assetStub.update.Name != "hero" || assetStub.update.Perspective == nil || *assetStub.update.Perspective != domain.PerspectiveTopDown {
			t.Fatalf("unexpected asset update: %+v", assetStub.update)
		}
	})

	t.Run("binds asset record", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/asset/save", strings.NewReader(`{"assetId":7}`))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.record.AssetID != 7 {
			t.Fatalf("unexpected record request: %+v", assetStub.record)
		}
	})

	t.Run("binds asset rollback", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/asset/rollback", strings.NewReader(`{"assetId":7,"version":2}`))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.rollback.AssetID != 7 || assetStub.rollback.Version != 2 {
			t.Fatalf("unexpected rollback request: %+v", assetStub.rollback)
		}
	})

	t.Run("binds asset records", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/asset/7/records", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if !assetStub.records {
			t.Fatal("expected asset records route to be called")
		}
	})

	t.Run("binds asset delete", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodDelete, "/api/v1/asset/delete", strings.NewReader(`{"assetId":7}`))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.deleted.AssetID != 7 {
			t.Fatalf("unexpected delete request: %+v", assetStub.deleted)
		}
	})

	t.Run("asset detail", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/asset/7", nil)
		recorder := httptest.NewRecorder()

		e.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
		}
		if assetStub.assetID != 7 {
			t.Fatalf("expected asset ID 7, got %d", assetStub.assetID)
		}
		if recorder.Body.String() != `{"code":200,"message":"success","data":{"assetId":7,"name":"","projectId":0,"type":"","description":"","perspective":"","scale":null,"tags":null,"version":0}}`+"\n" {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
	})
}

func TestAssetRoutesRejectZeroPathIDs(t *testing.T) {
	e := router.Register(&assetRouterStub{}, nil, nil, nil)

	for _, path := range []string{
		"/api/v1/projects/0/assets",
		"/api/v1/asset/0",
		"/api/v1/asset/0/records",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAssetUpdateRejectsImmutableAndNonAttributeFields(t *testing.T) {
	for _, field := range []string{
		`"projectId":42`,
		`"type":"object"`,
		`"attributes":{}`,
		`"content":{}`,
	} {
		t.Run(field, func(t *testing.T) {
			stub := &assetRouterStub{}
			e := router.Register(stub, nil, nil, nil)
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/asset/update",
				strings.NewReader(`{"assetId":7,`+field+`}`),
			)
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
			}
			if stub.update.AssetID != 0 {
				t.Fatalf("expected invalid body not to reach the handler, got %+v", stub.update)
			}
		})
	}
}
